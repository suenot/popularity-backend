package handlers_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/suenot/popularity-backend/internal/handlers"
)

// mockRow implements pgx.Row over a fixed value set.
type mockRow struct {
	vals []any
	err  error
}

func (m *mockRow) Scan(dest ...any) error {
	if m.err != nil {
		return m.err
	}
	if len(dest) != len(m.vals) {
		return errors.New("mockRow: arg count mismatch")
	}
	for i, d := range dest {
		switch p := d.(type) {
		case *int64:
			*p = m.vals[i].(int64)
		case *string:
			*p = m.vals[i].(string)
		case **int64:
			v := m.vals[i].(int64)
			*p = &v
		case **float64:
			v := m.vals[i].(float64)
			*p = &v
		case *[]byte:
			*p = m.vals[i].([]byte)
		default:
			return errors.New("mockRow: unsupported dest type")
		}
	}
	return nil
}

type mockDB struct {
	row *mockRow
}

func (m *mockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return m.row
}

func TestFetchStats_NotFound(t *testing.T) {
	db := &mockDB{row: &mockRow{err: pgx.ErrNoRows}}
	_, err := handlers.FetchStats(context.Background(), db, 1, "u")
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected ErrNoRows, got %v", err)
	}
}

func TestFetchStats_Happy(t *testing.T) {
	db := &mockDB{row: &mockRow{vals: []any{
		int64(42), "youtube", "google",
		int64(1000), int64(50000), int64(120),
		// d1, d7, d30, d90, d365, cagr_1y, velocity_7d, velocity_28d
		float64(0.5), float64(1.5), float64(5.0), float64(20.0), float64(80.0), float64(80.0),
		float64(10.0), float64(5.0),
		[]byte(`{"channel_id":"UC123","reputation":42}`),
	}}}
	row, err := handlers.FetchStats(context.Background(), db, 42, "u")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if row.ChannelID != 42 || row.Platform != "youtube" || row.Handle != "google" {
		t.Fatalf("bad row: %+v", row)
	}
	if row.Followers == nil || *row.Followers != 1000 {
		t.Fatalf("bad followers: %+v", row.Followers)
	}
	if row.D1Pct == nil || *row.D1Pct != 0.5 {
		t.Fatalf("bad d1_pct: %+v", row.D1Pct)
	}
	if row.D7Pct == nil || *row.D7Pct != 1.5 {
		t.Fatalf("bad d7_pct: %+v", row.D7Pct)
	}
}
