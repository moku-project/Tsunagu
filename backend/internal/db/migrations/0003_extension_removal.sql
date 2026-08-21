CREATE TABLE library_entries_new (
    id INTEGER PRIMARY KEY,
    extension_id INTEGER REFERENCES extensions(id) ON DELETE SET NULL,
    extension_name TEXT NOT NULL DEFAULT '',
    external_id TEXT NOT NULL,
    content_type TEXT NOT NULL CHECK (content_type IN ('manga', 'novel', 'anime')),
    title TEXT NOT NULL,
    cover_path TEXT,
    description TEXT,
    status TEXT,
    extension_removed_at TIMESTAMP,
    added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(extension_id, external_id)
);

INSERT INTO library_entries_new (
    id, extension_id, extension_name, external_id, content_type, title,
    cover_path, description, status, added_at
)
SELECT
    le.id, le.extension_id, COALESCE(e.name, ''), le.external_id, le.content_type,
    le.title, le.cover_path, le.description, le.status, le.added_at
FROM library_entries le
LEFT JOIN extensions e ON e.id = le.extension_id;

DROP TABLE library_entries;
ALTER TABLE library_entries_new RENAME TO library_entries;

CREATE INDEX IF NOT EXISTS idx_library_entries_extension_id ON library_entries(extension_id);
