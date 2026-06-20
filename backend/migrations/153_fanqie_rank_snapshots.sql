-- Persist Fanqie rank snapshots so the writer workbench can reuse the latest
-- known hotlist without crawling the upstream site on every cache miss.
CREATE TABLE IF NOT EXISTS fanqie_rank_snapshots (
    id BIGSERIAL PRIMARY KEY,
    channel VARCHAR(32) NOT NULL UNIQUE,
    books JSONB NOT NULL DEFAULT '[]'::jsonb,
    source VARCHAR(64) NOT NULL DEFAULT '',
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fanqie_rank_snapshots_fetched_at
    ON fanqie_rank_snapshots (fetched_at DESC);
