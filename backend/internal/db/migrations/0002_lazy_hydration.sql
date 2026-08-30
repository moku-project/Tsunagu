ALTER TABLE media ADD COLUMN chapters_synced_at TIMESTAMP;

UPDATE media SET chapters_synced_at = COALESCE(details_fetched_at, CURRENT_TIMESTAMP)
WHERE id IN (SELECT DISTINCT media_id FROM chapters);
