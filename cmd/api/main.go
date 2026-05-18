// Command api is the backend's main entrypoint. The MODE env var
// (api|scheduler|both) chooses which entrypoints run in this process;
// default is "both". A dedicated scheduler-only binary lives at
// cmd/scheduler for deployments that want to scale roles independently.
package main

import (
	"context"
	"log"
	"os/signal"
	"sync"
	"syscall"

	"github.com/suenot/w-popularity-backend/internal/config"
	"github.com/suenot/w-popularity-backend/internal/server"
)

func main() {
	cfg := config.FromEnv()
	if cfg.Mode == "" {
		cfg.Mode = config.ModeBoth
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := server.MustPool(ctx, cfg)
	if err != nil {
		log.Fatalf("db init: %v", err)
	}
	defer pool.Close()

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	if cfg.Mode == config.ModeAPI || cfg.Mode == config.ModeBoth {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := server.RunAPI(ctx, cfg, pool); err != nil {
				errCh <- err
			}
		}()
	}
	if cfg.Mode == config.ModeScheduler || cfg.Mode == config.ModeBoth {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := server.RunScheduler(ctx, cfg, pool); err != nil {
				errCh <- err
			}
		}()
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		log.Printf("error: %v", err)
	}
}
