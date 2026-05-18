package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/suenot/w-popularity-backend/internal/middleware"
)

// StatsRow is one row from v_channel_stats — the public KPI shape.
type StatsRow struct {
	ChannelID  int64                  `json:"channel_id"`
	Platform   string                 `json:"platform"`
	Handle     string                 `json:"handle"`
	Followers  *int64                 `json:"followers"`
	TotalViews *int64                 `json:"total_views"`
	PostsCount *int64                 `json:"posts_count"`
	D7Pct      *float64               `json:"d7_pct"`
	D30Pct     *float64               `json:"d30_pct"`
	D90Pct     *float64               `json:"d90_pct"`
	D365Pct    *float64               `json:"d365_pct"`
	CAGR1YPct  *float64               `json:"cagr_1y_pct"`
	Velocity7  *float64               `json:"velocity_7d"`
	Velocity28 *float64               `json:"velocity_28d"`
	// Raw carries platform-specific fields from the latest snapshot
	// (e.g. Stack Overflow badge counts, Habr karma, T-Bank strategies).
	Raw map[string]interface{} `json:"raw,omitempty"`
}

// StatsQuerier is the minimal DB interface used by Stats (allows mocking).
type StatsQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// StatsAPI exposes per-channel KPI rows.
type StatsAPI struct {
	DB StatsQuerier
}

// Get handles GET /api/v1/channels/:id/stats.
func (h *StatsAPI) Get(c *gin.Context) {
	userID := middleware.UserID(c)
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "bad id"})
		return
	}
	row, err := FetchStats(c.Request.Context(), h.DB, channelID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, row)
}

// FetchStats reads one v_channel_stats row, scoped to userID.
// Exported so unit tests can drive it directly with a mock pool.
//
// Joins the latest channel_snapshots row to pull the platform-specific
// `raw` JSON (badge counts, karma, post-engagement details that don't
// fit the cross-platform KPI shape).
func FetchStats(ctx context.Context, db StatsQuerier, channelID int64, userID string) (StatsRow, error) {
	var r StatsRow
	var raw []byte
	err := db.QueryRow(ctx, `
		SELECT s.channel_id, s.platform, s.handle,
		       s.followers, s.total_views, s.posts_count,
		       s.d7_pct, s.d30_pct, s.d90_pct, s.d365_pct, s.cagr_1y_pct,
		       s.velocity_7d, s.velocity_28d,
		       COALESCE((
		           SELECT cs.raw FROM channel_snapshots cs
		           WHERE cs.channel_id = s.channel_id
		           ORDER BY cs.ts DESC LIMIT 1
		       ), '{}'::jsonb)
		FROM v_channel_stats s
		JOIN channels c ON c.id = s.channel_id
		WHERE s.channel_id=$1 AND c.user_id=$2`,
		channelID, userID,
	).Scan(&r.ChannelID, &r.Platform, &r.Handle,
		&r.Followers, &r.TotalViews, &r.PostsCount,
		&r.D7Pct, &r.D30Pct, &r.D90Pct, &r.D365Pct, &r.CAGR1YPct,
		&r.Velocity7, &r.Velocity28,
		&raw)
	if err == nil && len(raw) > 0 && string(raw) != "{}" {
		_ = json.Unmarshal(raw, &r.Raw)
	}
	return r, err
}
