DROP TRIGGER IF EXISTS click_events_update_short_url_summary ON click_events;
DROP FUNCTION IF EXISTS update_short_url_click_summary();
ALTER TABLE short_urls
    DROP COLUMN IF EXISTS last_clicked_at,
    DROP COLUMN IF EXISTS total_clicks;
