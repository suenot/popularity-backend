// Command scheduler runs only the cron-driven daily fetch + worker pool,
// without the HTTP API. Use this when you want to scale the scheduler
// independently of the API tier.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/suenot/w-popularity-backend/internal/config"
	"github.com/suenot/w-popularity-backend/internal/server"
)

func main() {
	cfg := config.FromEnv()
	cfg.Mode = config.ModeScheduler

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := server.MustPool(ctx, cfg)
	if err != nil {
		log.Fatalf("db init: %v", err)
	}
	defer pool.Close()

	if err := server.RunScheduler(ctx, cfg, pool); err != nil {
		log.Fatalf("scheduler: %v", err)
	}
}
