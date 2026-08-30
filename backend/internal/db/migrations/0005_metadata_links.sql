CREATE TABLE metadata_links (
    id INTEGER PRIMARY KEY,
    media_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    provider_url TEXT NOT NULL DEFAULT '',
    confidence REAL NOT NULL DEFAULT 0,

    locked INTEGER NOT NULL DEFAULT 0,
    matched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(media_id, provider)
);
CREATE INDEX idx_metadata_links_media_id ON metadata_links(media_id);
