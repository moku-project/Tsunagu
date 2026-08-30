ALTER TABLE chapters ADD COLUMN first_seen_at TIMESTAMP;
UPDATE chapters SET first_seen_at = CURRENT_TIMESTAMP WHERE first_seen_at IS NULL;
CREATE INDEX idx_chapters_first_seen_at ON chapters(first_seen_at);

ALTER TABLE media ADD COLUMN cover_override TEXT;
