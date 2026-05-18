// Package jobs implements a Postgres-backed fetch-job queue using
// SELECT ... FOR UPDATE SKIP LOCKED for safe parallel consumption.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// MaxAttempts caps retries before a job is marked failed permanently.
const MaxAttempts = 3

// ErrNoJob is returned by Claim when no claimable job is available.
var ErrNoJob = errors.New("jobs: no job available")

// Job is one row from `fetch_jobs`.
type Job struct {
	ID          int64
	ChannelID   int64
	Status      string
	Attempts    int
	LastError   string
	ScheduledAt time.Time
}

// Queue wraps the pool for queue operations.
type Queue struct {
	pool *pgxpool.Pool
}

// NewQueue constructs a Queue.
func NewQueue(pool *pgxpool.Pool) *Queue { return &Queue{pool: pool} }

// Enqueue inserts a pending job scheduled for `at`. Pass time.Time{} for now.
func (q *Queue) Enqueue(ctx context.Context, channelID int64, at time.Time) (int64, error) {
	if at.IsZero() {
		at = time.Now()
	}
	var id int64
	err := q.pool.QueryRow(ctx, `
		INSERT INTO fetch_jobs(channel_id, status, scheduled_at)
		VALUES ($1, 'pending', $2)
		RETURNING id`, channelID, at).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("enqueue: %w", err)
	}
	return id, nil
}

// EnqueueAllChannels inserts a pending job for every channel. Used by the
// daily cron.
func (q *Queue) EnqueueAllChannels(ctx context.Context) (int64, error) {
	tag, err := q.pool.Exec(ctx, `
		INSERT INTO fetch_jobs(channel_id, status, scheduled_at)
		SELECT id, 'pending', now() FROM channels`)
	if err != nil {
		return 0, fmt.Errorf("enqueue all: %w", err)
	}
	return tag.RowsAffected(), nil
}

// Claim atomically picks one pending job, marks it running, and returns it.
// Returns ErrNoJob when the queue is empty (callers should sleep and retry).
func (q *Queue) Claim(ctx context.Context) (*Job, error) {
	tx, err := q.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("claim: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var j Job
	err = tx.QueryRow(ctx, `
		SELECT id, channel_id, status, attempts, COALESCE(last_error,''), scheduled_at
		FROM fetch_jobs
		WHERE status='pending' AND scheduled_at <= now()
		ORDER BY scheduled_at
		FOR UPDATE SKIP LOCKED
		LIMIT 1`).Scan(&j.ID, &j.ChannelID, &j.Status, &j.Attempts, &j.LastError, &j.ScheduledAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoJob
	}
	if err != nil {
		return nil, fmt.Errorf("claim: select: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE fetch_jobs
		SET status='running', attempts=attempts+1, started_at=now()
		WHERE id=$1`, j.ID); err != nil {
		return nil, fmt.Errorf("claim: update: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("claim: commit: %w", err)
	}
	j.Attempts++
	j.Status = "running"
	return &j, nil
}

// Complete marks job done.
func (q *Queue) Complete(ctx context.Context, id int64) error {
	_, err := q.pool.Exec(ctx, `
		UPDATE fetch_jobs SET status='done', finished_at=now(), last_error=NULL
		WHERE id=$1`, id)
	return err
}

// Fail marks job failed. If attempts < MaxAttempts the job is reset to
// pending with a 5-minute backoff; otherwise it stays failed.
func (q *Queue) Fail(ctx context.Context, id int64, attempts int, reason string) error {
	if attempts >= MaxAttempts {
		_, err := q.pool.Exec(ctx, `
			UPDATE fetch_jobs SET status='failed', finished_at=now(), last_error=$2
			WHERE id=$1`, id, reason)
		return err
	}
	_, err := q.pool.Exec(ctx, `
		UPDATE fetch_jobs SET status='pending', last_error=$2,
		    scheduled_at = now() + interval '5 minutes',
		    started_at=NULL
		WHERE id=$1`, id, reason)
	return err
}
