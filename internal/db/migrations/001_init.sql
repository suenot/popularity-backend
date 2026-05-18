CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS channels (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform TEXT NOT NULL,
    handle TEXT NOT NULL,
    url TEXT NOT NULL,
    added_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(user_id, platform, handle)
);
CREATE INDEX IF NOT EXISTS idx_channels_user ON channels(user_id);

CREATE TABLE IF NOT EXISTS channel_snapshots (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    ts TIMESTAMPTZ NOT NULL,
    followers BIGINT NOT NULL DEFAULT 0,
    posts_count BIGINT NOT NULL DEFAULT 0,
    total_likes BIGINT NOT NULL DEFAULT 0,
    total_views BIGINT NOT NULL DEFAULT 0,
    total_comments BIGINT NOT NULL DEFAULT 0,
    raw JSONB
);
CREATE INDEX IF NOT EXISTS idx_chan_snap_brin ON channel_snapshots USING BRIN(ts);
CREATE INDEX IF NOT EXISTS idx_chan_snap_chan_ts ON channel_snapshots(channel_id, ts DESC);

CREATE TABLE IF NOT EXISTS posts (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    platform_post_id TEXT NOT NULL,
    url TEXT NOT NULL,
    kind TEXT,
    published_at TIMESTAMPTZ,
    UNIQUE(channel_id, platform_post_id)
);

CREATE TABLE IF NOT EXISTS post_snapshots (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    ts TIMESTAMPTZ NOT NULL,
    likes BIGINT, views BIGINT, comments BIGINT, shares BIGINT,
    raw JSONB
);
CREATE INDEX IF NOT EXISTS idx_post_snap_brin ON post_snapshots USING BRIN(ts);
CREATE INDEX IF NOT EXISTS idx_post_snap_post_ts ON post_snapshots(post_id, ts DESC);

CREATE TABLE IF NOT EXISTS fetch_jobs (
    id BIGSERIAL PRIMARY KEY,
    channel_id BIGINT NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_fetch_jobs_claimable ON fetch_jobs(status, scheduled_at) WHERE status='pending';
