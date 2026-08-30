CREATE TABLE repositories (
    id INTEGER PRIMARY KEY,
    index_url TEXT NOT NULL UNIQUE,
    name TEXT,
    content_type TEXT NOT NULL DEFAULT 'manga' CHECK (content_type IN ('manga', 'novel', 'anime')),
    added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_synced_at TIMESTAMP
);

CREATE TABLE extensions (
    id INTEGER PRIMARY KEY,
    repository_id INTEGER NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    package_name TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    version TEXT NOT NULL,
    content_type TEXT NOT NULL CHECK (content_type IN ('manga', 'novel', 'anime')),
    lang TEXT NOT NULL,
    icon_url TEXT,
    icon_local_path TEXT,
    apk_url TEXT NOT NULL,
    jar_url TEXT,
    jar_path TEXT,
    installed BOOLEAN NOT NULL DEFAULT FALSE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    discovered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    installed_at TIMESTAMP,
    installed_version TEXT,
    needs_update BOOLEAN GENERATED ALWAYS AS (
        installed AND installed_version IS NOT NULL AND installed_version != version
    ) STORED,
    is_nsfw BOOLEAN NOT NULL DEFAULT FALSE,
    supports_latest BOOLEAN NOT NULL DEFAULT TRUE
);
CREATE INDEX idx_extensions_repository_id ON extensions(repository_id);
CREATE INDEX idx_extensions_installed ON extensions(installed) WHERE installed = TRUE;

CREATE TABLE media (
    id INTEGER PRIMARY KEY,
    extension_id INTEGER REFERENCES extensions(id) ON DELETE SET NULL,
    extension_name TEXT NOT NULL DEFAULT '',
    external_id TEXT NOT NULL,
    content_type TEXT NOT NULL CHECK (content_type IN ('manga', 'novel', 'anime')),
    title TEXT NOT NULL,
    cover_path TEXT,
    cover_local_path TEXT,
    description TEXT,
    status TEXT,
    author TEXT,
    artist TEXT,
    extension_removed_at TIMESTAMP,
    added_at TIMESTAMP,
    last_viewed_at TIMESTAMP,

    details_fetched_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(extension_id, external_id)
);
CREATE INDEX idx_media_extension_id ON media(extension_id);
CREATE INDEX idx_media_added_at ON media(added_at) WHERE added_at IS NOT NULL;
CREATE INDEX idx_media_content_type ON media(content_type);

CREATE TABLE tags (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE media_tags (
    media_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (media_id, tag_id)
);
CREATE INDEX idx_media_tags_tag_id ON media_tags(tag_id);

CREATE TABLE genres (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE media_genres (
    media_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    genre_id INTEGER NOT NULL REFERENCES genres(id) ON DELETE CASCADE,
    PRIMARY KEY (media_id, genre_id)
);
CREATE INDEX idx_media_genres_genre_id ON media_genres(genre_id);

CREATE TABLE folders (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'custom' CHECK (kind IN ('reading_status', 'custom')),
    system_key TEXT UNIQUE,
    parent_folder_id INTEGER REFERENCES folders(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    include_in_update INTEGER NOT NULL DEFAULT 1,
    include_in_download INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_folders_parent_folder_id ON folders(parent_folder_id);
CREATE INDEX idx_folders_kind ON folders(kind);

CREATE TABLE media_folders (
    media_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    folder_id INTEGER NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
    added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (media_id, folder_id)
);
CREATE INDEX idx_media_folders_folder_id ON media_folders(folder_id);

CREATE TABLE chapters (
    id INTEGER PRIMARY KEY,
    media_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    external_id TEXT NOT NULL,
    title TEXT,
    number REAL,
    uploaded_at INTEGER,
    source_order INTEGER,
    UNIQUE(media_id, external_id)
);
CREATE INDEX idx_chapters_media_id ON chapters(media_id);

CREATE TABLE manga_pages (
    chapter_id INTEGER NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    page_number INTEGER NOT NULL,
    local_path TEXT,
    PRIMARY KEY (chapter_id, page_number)
);

CREATE TABLE novel_chapter_content (
    chapter_id INTEGER PRIMARY KEY REFERENCES chapters(id) ON DELETE CASCADE,
    local_path TEXT
);

CREATE TABLE anime_episode_streams (
    chapter_id INTEGER PRIMARY KEY REFERENCES chapters(id) ON DELETE CASCADE,
    stream_url TEXT,
    local_path TEXT
);

CREATE TABLE downloads (
    id INTEGER PRIMARY KEY,
    chapter_id INTEGER NOT NULL REFERENCES chapters(id),
    status TEXT NOT NULL CHECK (status IN ('queued', 'downloading', 'done', 'failed')),
    progress REAL NOT NULL DEFAULT 0,
    downloaded_bytes INTEGER,
    bytes_per_sec REAL,
    position INTEGER,
    error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);
CREATE INDEX idx_downloads_status ON downloads(status);
CREATE INDEX idx_downloads_chapter_id ON downloads(chapter_id);

CREATE TABLE reading_progress (
    id INTEGER PRIMARY KEY,
    media_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    chapter_id INTEGER NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    progress REAL NOT NULL DEFAULT 0,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    position_seconds REAL,
    duration_seconds REAL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(media_id, chapter_id)
);

CREATE TABLE tracker_accounts (
    id INTEGER PRIMARY KEY,
    tracker_type TEXT NOT NULL UNIQUE,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    expires_at TIMESTAMP
);

CREATE TABLE tracker_links (
    id INTEGER PRIMARY KEY,
    media_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    tracker_account_id INTEGER NOT NULL REFERENCES tracker_accounts(id) ON DELETE CASCADE,
    external_tracker_id TEXT NOT NULL,
    sync_progress BOOLEAN NOT NULL DEFAULT TRUE,
    last_synced_at TIMESTAMP,
    UNIQUE(media_id, tracker_account_id)
);
CREATE INDEX idx_tracker_links_tracker_account_id ON tracker_links(tracker_account_id);
