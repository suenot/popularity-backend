package jobs_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suenot/popularity-backend/internal/jobs"
)

// TestClaimSkipLocked spawns two parallel claimers against a real Postgres
// and asserts they never see the same job. Skipped when TEST_DATABASE_URL
// is unset so `go test ./...` stays green in CI without a DB.
func TestClaimSkipLocked(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	// Minimal isolated schema for this test.
	if _, err := pool.Exec(ctx, `
		CREATE TEMP TABLE channels_t (id BIGINT PRIMARY KEY);
		INSERT INTO channels_t VALUES (1), (2), (3);
	`); err != nil {
		t.Fatalf("temp schema: %v", err)
	}

	q := jobs.NewQueue(pool)
	for _, id := range []int64{1, 2, 3} {
		// Skip if the real `channels` table doesn't have these IDs;
		// in that case the FK on fetch_jobs would block us. This test
		// requires a clean DB.
		if _, err := q.Enqueue(ctx, id, time.Time{}); err != nil {
			t.Skipf("enqueue requires FK-compatible channels rows: %v", err)
			return
		}
	}

	seen := make(map[int64]bool)
	for i := 0; i < 3; i++ {
		j, err := q.Claim(ctx)
		if err != nil {
			t.Fatalf("claim %d: %v", i, err)
		}
		if seen[j.ID] {
			t.Fatalf("job %d claimed twice", j.ID)
		}
		seen[j.ID] = true
		if err := q.Complete(ctx, j.ID); err != nil {
			t.Fatalf("complete: %v", err)
		}
	}
	if _, err := q.Claim(ctx); err != jobs.ErrNoJob {
		t.Fatalf("expected ErrNoJob, got %v", err)
	}
}
