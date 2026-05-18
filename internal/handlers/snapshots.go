package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suenot/w-popularity-backend/internal/middleware"
)

// SnapshotsAPI exposes channel & post time-series endpoints.
type SnapshotsAPI struct {
	DB *pgxpool.Pool
}

// List handles GET /api/v1/channels/:id/snapshots?from=&to=.
// from/to are RFC3339; both optional. Defaults: from=now-30d, to=now.
func (h *SnapshotsAPI) List(c *gin.Context) {
	userID := middleware.UserID(c)
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "bad id"})
		return
	}

	now := time.Now().UTC()
	from := now.Add(-30 * 24 * time.Hour)
	to := now
	if s := c.Query("from"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			from = t
		}
	}
	if s := c.Query("to"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			to = t
		}
	}

	// Ownership check.
	var owned bool
	if err := h.DB.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM channels WHERE id=$1 AND user_id=$2)`,
		channelID, userID,
	).Scan(&owned); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !owned {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	rows, err := h.DB.Query(c.Request.Context(), `
		SELECT ts, followers, posts_count, total_likes, total_views, total_comments
		FROM channel_snapshots
		WHERE channel_id=$1 AND ts >= $2 AND ts <= $3
		ORDER BY ts ASC`,
		channelID, from, to)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type point struct {
		TS            time.Time `json:"ts"`
		Followers     int64     `json:"followers"`
		PostsCount    int64     `json:"posts_count"`
		TotalLikes    int64     `json:"total_likes"`
		TotalViews    int64     `json:"total_views"`
		TotalComments int64     `json:"total_comments"`
	}
	out := []point{}
	for rows.Next() {
		var p point
		if err := rows.Scan(&p.TS, &p.Followers, &p.PostsCount, &p.TotalLikes, &p.TotalViews, &p.TotalComments); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out = append(out, p)
	}
	c.JSON(http.StatusOK, gin.H{"snapshots": out, "from": from, "to": to})
}

// Posts handles GET /api/v1/channels/:id/posts (latest snapshot per post).
func (h *SnapshotsAPI) Posts(c *gin.Context) {
	userID := middleware.UserID(c)
	channelID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "bad id"})
		return
	}

	var owned bool
	if err := h.DB.QueryRow(c.Request.Context(),
		`SELECT EXISTS(SELECT 1 FROM channels WHERE id=$1 AND user_id=$2)`,
		channelID, userID,
	).Scan(&owned); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !owned {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	rows, err := h.DB.Query(c.Request.Context(), `
		SELECT p.id, p.platform_post_id, p.url, p.kind, p.published_at,
		       ls.ts, ls.likes, ls.views, ls.comments, ls.shares
		FROM posts p
		LEFT JOIN LATERAL (
			SELECT ts, likes, views, comments, shares
			FROM post_snapshots
			WHERE post_id = p.id
			ORDER BY ts DESC
			LIMIT 1
		) ls ON true
		WHERE p.channel_id=$1
		ORDER BY p.published_at DESC NULLS LAST
		LIMIT 200`, channelID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type post struct {
		ID             int64      `json:"id"`
		PlatformPostID string     `json:"platform_post_id"`
		URL            string     `json:"url"`
		Kind           *string    `json:"kind"`
		PublishedAt    *time.Time `json:"published_at"`
		LatestTS       *time.Time `json:"latest_ts"`
		Likes          *int64     `json:"likes"`
		Views          *int64     `json:"views"`
		Comments       *int64     `json:"comments"`
		Shares         *int64     `json:"shares"`
	}
	out := []post{}
	for rows.Next() {
		var p post
		if err := rows.Scan(&p.ID, &p.PlatformPostID, &p.URL, &p.Kind, &p.PublishedAt,
			&p.LatestTS, &p.Likes, &p.Views, &p.Comments, &p.Shares); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out = append(out, p)
	}
	c.JSON(http.StatusOK, gin.H{"posts": out})
}
