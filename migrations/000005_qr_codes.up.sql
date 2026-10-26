CREATE TABLE IF NOT EXISTS qr_codes (
    short_url_id UUID PRIMARY KEY REFERENCES short_urls(id) ON DELETE CASCADE,
    png BYTEA NOT NULL,
    foreground TEXT NOT NULL DEFAULT '#000000',
    background TEXT NOT NULL DEFAULT '#FFFFFF',
    frame_text TEXT NOT NULL DEFAULT 'Scan untuk membuka URL',
    logo BYTEA NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
