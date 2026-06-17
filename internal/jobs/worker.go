package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	shared "github.com/suenot/socials-auto"

	"github.com/suenot/popularity-backend/internal/parsers"
)

// Pool runs N goroutines that drain the fetch-job queue.
type Pool struct {
	N        int
	Queue    *Queue
	DB       *pgxpool.Pool
	Registry parsers.Registry
	Interval time.Duration
}

// Run blocks until ctx is cancelled. Each worker polls every Interval (or 2s).
func (p *Pool) Run(ctx context.Context) {
	if p.N <= 0 {
		p.N = 4
	}
	if p.Interval == 0 {
		p.Interval = 2 * time.Second
	}
	for i := 0; i < p.N; i++ {
		go p.loop(ctx, i)
	}
	<-ctx.Done()
}

func (p *Pool) loop(ctx context.Context, id int) {
	t := time.NewTicker(p.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		// Drain as many jobs as we can before sleeping again.
		for {
			if err := p.tickOne(ctx); err != nil {
				if errors.Is(err, ErrNoJob) {
					break
				}
				log.Printf("worker %d: %v", id, err)
				break
			}
		}
	}
}

func (p *Pool) tickOne(ctx context.Context) error {
	job, err := p.Queue.Claim(ctx)
	if err != nil {
		return err
	}

	// Resolve channel.
	var (
		platform string
		handle   string
	)
	if err := p.DB.QueryRow(ctx,
		`SELECT platform, handle FROM channels WHERE id=$1`,
		job.ChannelID,
	).Scan(&platform, &handle); err != nil {
		return p.Queue.Fail(ctx, job.ID, job.Attempts, "channel lookup: "+err.Error())
	}

	parser := p.Registry.Get(shared.Platform(platform))
	if parser == nil {
		return p.Queue.Fail(ctx, job.ID, MaxAttempts, "no parser for platform "+platform)
	}

	fetchCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	snap, err := parser.FetchChannel(fetchCtx, handle)
	if err != nil {
		// Map ErrNotImplemented to a permanent failure (skip retries).
		if errors.Is(err, shared.ErrNotImplemented) {
			return p.Queue.Fail(ctx, job.ID, MaxAttempts, err.Error())
		}
		return p.Queue.Fail(ctx, job.ID, job.Attempts, err.Error())
	}

	if err := p.persistChannelSnapshot(ctx, job.ChannelID, snap); err != nil {
		return p.Queue.Fail(ctx, job.ID, job.Attempts, "persist: "+err.Error())
	}

	// Best-effort post fetch (last 24h). Failures here don't fail the job.
	posts, err := parser.FetchRecentPosts(fetchCtx, handle, time.Now().Add(-24*time.Hour))
	if err == nil {
		_ = p.persistPostSnapshots(ctx, job.ChannelID, posts)
	}

	return p.Queue.Complete(ctx, job.ID)
}

func (p *Pool) persistChannelSnapshot(ctx context.Context, channelID int64, s shared.ChannelSnapshot) error {
	raw, _ := json.Marshal(s.Raw)
	_, err := p.DB.Exec(ctx, `
		INSERT INTO channel_snapshots
		    (channel_id, ts, followers, posts_count, total_likes, total_views, total_comments, raw)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		channelID, s.FetchedAt, s.Followers, s.PostsCount,
		s.TotalLikes, s.TotalViews, s.TotalComments, raw)
	return err
}

func (p *Pool) persistPostSnapshots(ctx context.Context, channelID int64, posts []shared.PostSnapshot) error {
	for _, ps := range posts {
		var postID int64
		err := p.DB.QueryRow(ctx, `
			INSERT INTO posts(channel_id, platform_post_id, url, kind, published_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (channel_id, platform_post_id)
			DO UPDATE SET url=EXCLUDED.url
			RETURNING id`,
			channelID, ps.PostID, ps.URL, string(ps.Kind), ps.PublishedAt,
		).Scan(&postID)
		if err != nil {
			return err
		}
		raw, _ := json.Marshal(ps.Raw)
		if _, err := p.DB.Exec(ctx, `
			INSERT INTO post_snapshots(post_id, ts, likes, views, comments, shares, raw)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			postID, ps.FetchedAt, ps.Likes, ps.Views, ps.Comments, ps.Shares, raw,
		); err != nil {
			return err
		}
	}
	return nil
}
