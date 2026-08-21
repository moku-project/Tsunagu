CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS library_entry_tags (
    library_entry_id INTEGER NOT NULL REFERENCES library_entries(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (library_entry_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_library_entry_tags_tag_id ON library_entry_tags(tag_id);

CREATE TABLE IF NOT EXISTS folders (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'custom' CHECK (kind IN ('reading_status', 'custom')),
    system_key TEXT UNIQUE,
    parent_folder_id INTEGER REFERENCES folders(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_folders_parent_folder_id ON folders(parent_folder_id);
CREATE INDEX IF NOT EXISTS idx_folders_kind ON folders(kind);

CREATE TABLE IF NOT EXISTS library_entry_folders (
    library_entry_id INTEGER NOT NULL REFERENCES library_entries(id) ON DELETE CASCADE,
    folder_id INTEGER NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
    added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (library_entry_id, folder_id)
);

CREATE INDEX IF NOT EXISTS idx_library_entry_folders_folder_id ON library_entry_folders(folder_id);

INSERT INTO folders (name, kind, system_key, sort_order) VALUES
    ('Reading', 'reading_status', 'reading', 0),
    ('Plan to Read', 'reading_status', 'plan_to_read', 1),
    ('Completed', 'reading_status', 'completed', 2),
    ('On Hold', 'reading_status', 'on_hold', 3),
    ('Dropped', 'reading_status', 'dropped', 4)
ON CONFLICT(system_key) DO NOTHING;
