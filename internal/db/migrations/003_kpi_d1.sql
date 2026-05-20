-- Add a 1-day delta (d1_pct) to v_channel_stats.
--
-- CREATE OR REPLACE VIEW cannot insert a column in the middle of the column
-- list, so we DROP + recreate. The definition below is 002_kpi_views.sql's
-- view plus the s1 LATERAL join and the d1_pct column.

DROP VIEW IF EXISTS v_channel_stats;

CREATE VIEW v_channel_stats AS
SELECT
    c.id AS channel_id,
    c.platform,
    c.handle,
    l.followers,
    l.total_views,
    l.posts_count,
    l.ts AS latest_ts,
    s1.ts  AS ts_1d,
    s7.ts  AS ts_7d,
    s30.ts AS ts_30d,
    s90.ts AS ts_90d,
    s365.ts AS ts_365d,
    CASE WHEN s1.followers  IS NOT NULL AND s1.followers  > 0
         THEN (l.followers - s1.followers)::float  / s1.followers  * 100 END AS d1_pct,
    CASE WHEN s7.followers  IS NOT NULL AND s7.followers  > 0
         THEN (l.followers - s7.followers)::float  / s7.followers  * 100 END AS d7_pct,
    CASE WHEN s30.followers IS NOT NULL AND s30.followers > 0
         THEN (l.followers - s30.followers)::float / s30.followers * 100 END AS d30_pct,
    CASE WHEN s90.followers IS NOT NULL AND s90.followers > 0
         THEN (l.followers - s90.followers)::float / s90.followers * 100 END AS d90_pct,
    CASE WHEN s365.followers IS NOT NULL AND s365.followers > 0
         THEN (l.followers - s365.followers)::float / s365.followers * 100 END AS d365_pct,
    -- 1-year CAGR (simple ratio - 1 since window == 1 year).
    CASE WHEN s365.followers IS NOT NULL AND s365.followers > 0
         THEN (l.followers::float / s365.followers - 1) * 100 END AS cagr_1y_pct,
    -- Velocity: followers gained per day over the window.
    CASE WHEN s7.ts IS NOT NULL AND l.ts > s7.ts
         THEN (l.followers - s7.followers)::float
              / (EXTRACT(EPOCH FROM (l.ts - s7.ts)) / 86400.0) END AS velocity_7d,
    CASE WHEN s30.ts IS NOT NULL AND l.ts > s30.ts
         THEN (l.followers - s30.followers)::float
              / (EXTRACT(EPOCH FROM (l.ts - s30.ts)) / 86400.0) END AS velocity_28d
FROM channels c
JOIN v_channel_latest l ON l.channel_id = c.id
LEFT JOIN LATERAL snapshot_at(c.id, 1)   AS s1   ON true
LEFT JOIN LATERAL snapshot_at(c.id, 7)   AS s7   ON true
LEFT JOIN LATERAL snapshot_at(c.id, 30)  AS s30  ON true
LEFT JOIN LATERAL snapshot_at(c.id, 90)  AS s90  ON true
LEFT JOIN LATERAL snapshot_at(c.id, 365) AS s365 ON true;
