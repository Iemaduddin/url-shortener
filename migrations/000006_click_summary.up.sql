BEGIN;

LOCK TABLE click_events IN SHARE ROW EXCLUSIVE MODE;
LOCK TABLE short_urls IN SHARE ROW EXCLUSIVE MODE;

ALTER TABLE short_urls
    ADD COLUMN IF NOT EXISTS total_clicks BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_clicked_at TIMESTAMPTZ;

UPDATE short_urls s
SET total_clicks = summary.total_clicks,
    last_clicked_at = summary.last_clicked_at
FROM (
    SELECT short_url_id, COUNT(*) AS total_clicks, MAX(clicked_at) AS last_clicked_at
    FROM click_events
    GROUP BY short_url_id
) summary
WHERE s.id = summary.short_url_id;

CREATE OR REPLACE FUNCTION update_short_url_click_summary()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE short_urls
    SET total_clicks = total_clicks + 1,
        last_clicked_at = GREATEST(COALESCE(last_clicked_at, NEW.clicked_at), NEW.clicked_at)
    WHERE id = NEW.short_url_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS click_events_update_short_url_summary ON click_events;
CREATE TRIGGER click_events_update_short_url_summary
AFTER INSERT ON click_events
FOR EACH ROW EXECUTE FUNCTION update_short_url_click_summary();

COMMIT;
