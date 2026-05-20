package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/suenot/w-popularity-backend/internal/jobs"
	"github.com/suenot/w-popularity-backend/internal/middleware"
)

// ChannelsAPI groups channel CRUD endpoints.
type ChannelsAPI struct {
	DB    *pgxpool.Pool
	Queue *jobs.Queue
}

type createChannelReq struct {
	Platform string `json:"platform" binding:"required"`
	Handle   string `json:"handle" binding:"required"`
	URL      string `json:"url" binding:"required"`
}

// Create handles POST /api/v1/channels. Upserts the auth user, inserts the
// channel, and enqueues an immediate fetch_job.
func (h *ChannelsAPI) Create(c *gin.Context) {
	userID := middleware.UserID(c)
	if userID == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no user"})
		return
	}
	email, _ := c.Get(middleware.CtxEmail)
	emailStr, _ := email.(string)

	var req createChannelReq
	if err := c.BindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := h.ensureUser(ctx, userID, emailStr); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var id int64
	err := h.DB.QueryRow(ctx, `
		INSERT INTO channels(user_id, platform, handle, url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, platform, handle)
		DO UPDATE SET url=EXCLUDED.url
		RETURNING id`,
		userID, req.Platform, req.Handle, req.URL,
	).Scan(&id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if h.Queue != nil {
		_, _ = h.Queue.Enqueue(ctx, id, time.Time{})
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "platform": req.Platform, "handle": req.Handle, "url": req.URL})
}

// ensureUser upserts a row in users keyed by the JWT sub.
func (h *ChannelsAPI) ensureUser(ctx context.Context, id, email string) error {
	if email == "" {
		email = id + "@unknown"
	}
	_, err := h.DB.Exec(ctx, `
		INSERT INTO users(id, email) VALUES ($1, $2)
		ON CONFLICT (id) DO NOTHING`, id, email)
	return err
}

// List handles GET /api/v1/channels.
func (h *ChannelsAPI) List(c *gin.Context) {
	userID := middleware.UserID(c)
	rows, err := h.DB.Query(c.Request.Context(), `
		SELECT c.id, c.platform, c.handle, c.url, c.added_at,
		       s.followers, s.total_views, s.posts_count,
		       s.d1_pct, s.d7_pct, s.d30_pct, s.d90_pct, s.d365_pct, s.cagr_1y_pct,
		       s.velocity_7d, s.velocity_28d, s.latest_ts
		FROM channels c
		LEFT JOIN v_channel_stats s ON s.channel_id = c.id
		WHERE c.user_id=$1
		ORDER BY c.id`, userID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type item struct {
		ID         int64    `json:"id"`
		Platform   string   `json:"platform"`
		Handle     string   `json:"handle"`
		URL        string   `json:"url"`
		AddedAt    string   `json:"added_at"`
		Followers  *int64   `json:"followers"`
		TotalViews *int64   `json:"total_views"`
		PostsCount *int64   `json:"posts_count"`
		D1Pct      *float64 `json:"d1_pct"`
		D7Pct      *float64 `json:"d7_pct"`
		D30Pct     *float64 `json:"d30_pct"`
		D90Pct     *float64 `json:"d90_pct"`
		D365Pct    *float64 `json:"d365_pct"`
		CAGR1YPct  *float64 `json:"cagr_1y_pct"`
		Velocity7  *float64 `json:"velocity_7d"`
		Velocity28 *float64 `json:"velocity_28d"`
		LatestTS   *string  `json:"latest_ts"`
	}
	out := []item{}
	for rows.Next() {
		var it item
		var addedAt time.Time
		var latestTS pgtype.Timestamptz
		if err := rows.Scan(&it.ID, &it.Platform, &it.Handle, &it.URL, &addedAt,
			&it.Followers, &it.TotalViews, &it.PostsCount,
			&it.D1Pct, &it.D7Pct, &it.D30Pct, &it.D90Pct, &it.D365Pct, &it.CAGR1YPct,
			&it.Velocity7, &it.Velocity28, &latestTS); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		it.AddedAt = addedAt.UTC().Format(time.RFC3339)
		if latestTS.Valid {
			ts := latestTS.Time.UTC().Format(time.RFC3339)
			it.LatestTS = &ts
		}
		out = append(out, it)
	}
	c.JSON(http.StatusOK, gin.H{"channels": out})
}

// Get handles GET /api/v1/channels/:id.
func (h *ChannelsAPI) Get(c *gin.Context) {
	userID := middleware.UserID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "bad id"})
		return
	}
	var (
		platform, handle, url string
	)
	err = h.DB.QueryRow(c.Request.Context(), `
		SELECT platform, handle, url FROM channels WHERE id=$1 AND user_id=$2`,
		id, userID,
	).Scan(&platform, &handle, &url)
	if errors.Is(err, pgx.ErrNoRows) {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id": id, "platform": platform, "handle": handle, "url": url,
	})
}

// Delete handles DELETE /api/v1/channels/:id.
func (h *ChannelsAPI) Delete(c *gin.Context) {
	userID := middleware.UserID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "bad id"})
		return
	}
	tag, err := h.DB.Exec(c.Request.Context(),
		`DELETE FROM channels WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if tag.RowsAffected() == 0 {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
