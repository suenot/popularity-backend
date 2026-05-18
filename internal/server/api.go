// Package server holds the bootstrap functions shared by cmd/api and
// cmd/scheduler.
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suenot/w-popularity-backend/internal/auth"
	"github.com/suenot/w-popularity-backend/internal/config"
	"github.com/suenot/w-popularity-backend/internal/db"
	"github.com/suenot/w-popularity-backend/internal/handlers"
	"github.com/suenot/w-popularity-backend/internal/jobs"
	"github.com/suenot/w-popularity-backend/internal/middleware"
)

// RunAPI boots the HTTP server. Cancel ctx to shut down gracefully.
func RunAPI(ctx context.Context, cfg config.Config, pool *pgxpool.Pool) error {
	verifier := auth.NewVerifier(cfg.AuthJWKSURL, cfg.AuthIssuer)
	queue := jobs.NewQueue(pool)

	r := buildRouter(cfg, pool, verifier, queue)

	srv := &http.Server{
		Addr:              ":" + cfg.BackendPort,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Printf("api: listening on %s (frontend=%s, mode=%s)", srv.Addr, cfg.FrontendURL, cfg.Mode)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	}
}

func buildRouter(cfg config.Config, pool *pgxpool.Pool, v *auth.Verifier, q *jobs.Queue) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())

	corsCfg := cors.DefaultConfig()
	if cfg.FrontendURL == "*" {
		corsCfg.AllowAllOrigins = true
	} else {
		corsCfg.AllowOrigins = []string{cfg.FrontendURL}
	}
	corsCfg.AllowHeaders = append(corsCfg.AllowHeaders, "Authorization")
	r.Use(cors.New(corsCfg))

	r.GET("/healthz", handlers.Health(pool))

	api := r.Group("/api/v1")
	api.Use(middleware.JWT(v, cfg.AuthServiceName))
	{
		channels := &handlers.ChannelsAPI{DB: pool, Queue: q}
		api.POST("/channels", channels.Create)
		api.GET("/channels", channels.List)
		api.GET("/channels/:id", channels.Get)
		api.DELETE("/channels/:id", channels.Delete)

		snaps := &handlers.SnapshotsAPI{DB: pool}
		api.GET("/channels/:id/snapshots", snaps.List)
		api.GET("/channels/:id/posts", snaps.Posts)

		stats := &handlers.StatsAPI{DB: pool}
		api.GET("/channels/:id/stats", stats.Get)
	}
	return r
}

// MustPool dials Postgres and runs migrations, exiting on error.
func MustPool(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.Migrate(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return pool, nil
}
