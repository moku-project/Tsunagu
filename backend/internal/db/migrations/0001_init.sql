CREATE TABLE IF NOT EXISTS repositories (
    id INTEGER PRIMARY KEY,
    index_url TEXT NOT NULL UNIQUE,
    name TEXT,
    content_type TEXT NOT NULL DEFAULT 'manga' CHECK (content_type IN ('manga', 'novel', 'anime')),
    added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_synced_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS extensions (
    id INTEGER PRIMARY KEY,
    repository_id INTEGER NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    package_name TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    content_type TEXT NOT NULL CHECK (content_type IN ('manga', 'novel', 'anime')),
    lang TEXT NOT NULL,
    icon_url TEXT,
    apk_url TEXT NOT NULL,
    jar_url TEXT,
    jar_path TEXT,
    installed BOOLEAN NOT NULL DEFAULT FALSE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    discovered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    installed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_extensions_repository_id ON extensions(repository_id);
CREATE INDEX IF NOT EXISTS idx_extensions_installed ON extensions(installed) WHERE installed = TRUE;

CREATE TABLE IF NOT EXISTS library_entries (
    id INTEGER PRIMARY KEY,
    extension_id INTEGER NOT NULL REFERENCES extensions(id),
    external_id TEXT NOT NULL,
    content_type TEXT NOT NULL CHECK (content_type IN ('manga', 'novel', 'anime')),
    title TEXT NOT NULL,
    cover_path TEXT,
    description TEXT,
    status TEXT,
    added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(extension_id, external_id)
);

CREATE TABLE IF NOT EXISTS chapters (
    id INTEGER PRIMARY KEY,
    library_entry_id INTEGER NOT NULL REFERENCES library_entries(id) ON DELETE CASCADE,
    external_id TEXT NOT NULL,
    title TEXT,
    number REAL,
    uploaded_at TIMESTAMP,
    UNIQUE(library_entry_id, external_id)
);

CREATE INDEX IF NOT EXISTS idx_chapters_library_entry_id ON chapters(library_entry_id);

CREATE TABLE IF NOT EXISTS manga_pages (
    chapter_id INTEGER NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    page_number INTEGER NOT NULL,
    local_path TEXT,
    PRIMARY KEY (chapter_id, page_number)
);

CREATE TABLE IF NOT EXISTS novel_chapter_content (
    chapter_id INTEGER PRIMARY KEY REFERENCES chapters(id) ON DELETE CASCADE,
    local_path TEXT
);

CREATE TABLE IF NOT EXISTS anime_episode_streams (
    chapter_id INTEGER PRIMARY KEY REFERENCES chapters(id) ON DELETE CASCADE,
    stream_url TEXT,
    local_path TEXT
);

CREATE TABLE IF NOT EXISTS downloads (
    id INTEGER PRIMARY KEY,
    chapter_id INTEGER NOT NULL REFERENCES chapters(id),
    status TEXT NOT NULL CHECK (status IN ('queued', 'downloading', 'done', 'failed')),
    progress REAL NOT NULL DEFAULT 0,
    error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_downloads_status ON downloads(status);

CREATE TABLE IF NOT EXISTS reading_progress (
    id INTEGER PRIMARY KEY,
    library_entry_id INTEGER NOT NULL REFERENCES library_entries(id) ON DELETE CASCADE,
    chapter_id INTEGER NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    progress REAL NOT NULL DEFAULT 0,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(library_entry_id, chapter_id)
);

CREATE TABLE IF NOT EXISTS tracker_accounts (
    id INTEGER PRIMARY KEY,
    tracker_type TEXT NOT NULL UNIQUE,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    expires_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tracker_links (
    id INTEGER PRIMARY KEY,
    library_entry_id INTEGER NOT NULL REFERENCES library_entries(id) ON DELETE CASCADE,
    tracker_account_id INTEGER NOT NULL REFERENCES tracker_accounts(id) ON DELETE CASCADE,
    external_tracker_id TEXT NOT NULL,
    sync_progress BOOLEAN NOT NULL DEFAULT TRUE,
    last_synced_at TIMESTAMP,
    UNIQUE(library_entry_id, tracker_account_id)
);
