package server

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"

	"github.com/suenot/w-popularity-backend/internal/config"
	"github.com/suenot/w-popularity-backend/internal/jobs"
	"github.com/suenot/w-popularity-backend/internal/parsers"
)

// RunScheduler starts the cron + worker pool. Blocks until ctx is cancelled.
func RunScheduler(ctx context.Context, cfg config.Config, pool *pgxpool.Pool) error {
	queue := jobs.NewQueue(pool)
	registry := parsers.Build(cfg)

	c := cron.New()
	_, err := c.AddFunc(cfg.FetchCron, func() {
		n, err := queue.EnqueueAllChannels(ctx)
		if err != nil {
			log.Printf("cron enqueue: %v", err)
			return
		}
		log.Printf("cron enqueue: scheduled %d channels", n)
	})
	if err != nil {
		return fmt.Errorf("cron schedule: %w", err)
	}
	c.Start()
	defer c.Stop()

	log.Printf("scheduler: cron=%q workers=%d", cfg.FetchCron, cfg.FetchWorkers)
	wpool := &jobs.Pool{
		N:        cfg.FetchWorkers,
		Queue:    queue,
		DB:       pool,
		Registry: registry,
	}
	wpool.Run(ctx)
	return nil
}
