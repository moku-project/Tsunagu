ALTER TABLE media ADD COLUMN content_block_rank INTEGER;
CREATE INDEX idx_media_content_block_rank ON media(content_block_rank);

ALTER TABLE media_tags ADD COLUMN weight INTEGER NOT NULL DEFAULT 0;

CREATE TABLE content_filter_rules (
    id INTEGER PRIMARY KEY,
    category TEXT NOT NULL,
    field TEXT NOT NULL CHECK (field IN ('genre', 'tag', 'title', 'description')),
    keyword TEXT NOT NULL,
    min_weight INTEGER NOT NULL DEFAULT 0,
    block_level INTEGER NOT NULL CHECK (block_level IN (1, 2)),
    is_default INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (field, keyword, category)
);

INSERT INTO content_filter_rules (category, field, keyword, block_level, is_default) VALUES
    ('sexual', 'genre', 'hentai', 1, 1),
    ('sexual', 'genre', 'ecchi', 1, 1),
    ('sexual', 'genre', 'adult', 1, 1),
    ('sexual', 'genre', 'mature', 1, 1),
    ('sexual', 'genre', 'smut', 1, 1),
    ('sexual', 'genre', 'erotica', 1, 1),
    ('sexual', 'genre', 'pornographic', 1, 1),
    ('sexual', 'genre', 'nsfw', 1, 1),
    ('sexual', 'genre', 'lolicon', 1, 1),
    ('sexual', 'genre', 'shotacon', 1, 1),
    ('sexual', 'tag', 'nudity', 1, 1),
    ('sexual', 'tag', 'sexual content', 1, 1),
    ('sexual', 'tag', 'sexual abuse', 1, 1),
    ('sexual', 'tag', 'sexual violence', 1, 1),
    ('sexual', 'tag', 'lolicon', 1, 1),
    ('sexual', 'tag', 'shotacon', 1, 1),
    ('gore', 'genre', 'gore', 2, 1),
    ('gore', 'tag', 'gore', 2, 1),
    ('gore', 'tag', 'guro', 2, 1),
    ('gore', 'tag', 'body horror', 2, 1),
    ('gore', 'tag', 'torture', 2, 1),
    ('gore', 'tag', 'cannibalism', 2, 1),
    ('gore', 'tag', 'graphic violence', 2, 1);
