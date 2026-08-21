PRAGMA foreign_keys=OFF;

CREATE TABLE chapters_new (
    id INTEGER PRIMARY KEY,
    library_entry_id INTEGER NOT NULL REFERENCES library_entries(id) ON DELETE CASCADE,
    external_id TEXT NOT NULL,
    title TEXT,
    number REAL,
    uploaded_at INTEGER,
    UNIQUE(library_entry_id, external_id)
);

INSERT INTO chapters_new (id, library_entry_id, external_id, title, number, uploaded_at)
SELECT id, library_entry_id, external_id, title, number,
    CASE WHEN uploaded_at IS NULL THEN NULL ELSE strftime('%s', uploaded_at) END
FROM chapters;

DROP TABLE chapters;
ALTER TABLE chapters_new RENAME TO chapters;

CREATE INDEX IF NOT EXISTS idx_chapters_library_entry_id ON chapters(library_entry_id);

PRAGMA foreign_keys=ON;
