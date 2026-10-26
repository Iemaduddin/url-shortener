CREATE TABLE IF NOT EXISTS click_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    short_url_id UUID NOT NULL REFERENCES short_urls(id) ON DELETE CASCADE,
    clicked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_agent TEXT,
    referer TEXT
);

CREATE INDEX IF NOT EXISTS click_events_short_url_clicked_at_idx
    ON click_events (short_url_id, clicked_at DESC);
