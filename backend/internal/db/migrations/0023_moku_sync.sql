-- Moku-Sync: CRDT state, the operation log and peers. Items are keyed by extension package and source IDs
-- joined with char(31), never by local row IDs, which differ on every device.

CREATE TABLE sync_meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Ops this device holds: its own, and others' that are still needed to bring another peer up to date.
CREATE TABLE sync_ops (
    origin TEXT NOT NULL,
    seq INTEGER NOT NULL,
    clock TEXT NOT NULL,
    kind TEXT NOT NULL,
    key TEXT NOT NULL,
    field TEXT NOT NULL,
    type TEXT NOT NULL,
    value TEXT,
    tags TEXT,
    PRIMARY KEY (origin, seq)
);

-- Highest seq seen per origin: this device's version vector.
CREATE TABLE sync_vv (
    origin TEXT PRIMARY KEY,
    seq INTEGER NOT NULL
);

-- LWW registers: the winning value and the op that wrote it.
CREATE TABLE sync_registers (
    kind TEXT NOT NULL,
    key TEXT NOT NULL,
    field TEXT NOT NULL,
    value TEXT NOT NULL,
    clock TEXT NOT NULL,
    origin TEXT NOT NULL,
    seq INTEGER NOT NULL,
    PRIMARY KEY (kind, key, field)
);

-- Observed-remove sets: an element is present while it has an add tag that no remove has observed.
CREATE TABLE sync_set_adds (
    kind TEXT NOT NULL,
    key TEXT NOT NULL,
    element TEXT NOT NULL,
    tag TEXT NOT NULL PRIMARY KEY,
    clock TEXT NOT NULL
);
CREATE INDEX idx_sync_set_adds_element ON sync_set_adds(kind, key, element);

CREATE TABLE sync_set_removed (
    tag TEXT PRIMARY KEY
);

CREATE TABLE sync_peers (
    node TEXT PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    token_hash TEXT NOT NULL DEFAULT '',
    dial_token TEXT NOT NULL DEFAULT '',
    vv TEXT NOT NULL DEFAULT '{}',
    last_seen_at INTEGER
);

CREATE TABLE sync_pairing_codes (
    code_hash TEXT PRIMARY KEY,
    expires_at INTEGER NOT NULL
);

-- Local writes waiting to become ops. Triggers fill it, so every write path is captured.
CREATE TABLE sync_dirty (
    id INTEGER PRIMARY KEY,
    kind TEXT NOT NULL,
    key TEXT NOT NULL,
    field TEXT NOT NULL,
    op TEXT NOT NULL,
    recorded_at INTEGER NOT NULL
);

ALTER TABLE folders ADD COLUMN sync_id TEXT;
UPDATE folders SET sync_id = CASE WHEN system_key IS NOT NULL THEN 'sys:' || system_key ELSE lower(hex(randomblob(16))) END;
CREATE UNIQUE INDEX idx_folders_sync_id ON folders(sync_id);

-- One trigger, so the ID exists before the change is recorded; trigger order isn't guaranteed.
CREATE TRIGGER sync_folder_insert AFTER INSERT ON folders
BEGIN
    UPDATE folders SET sync_id = CASE WHEN NEW.system_key IS NOT NULL THEN 'sys:' || NEW.system_key ELSE lower(hex(randomblob(16))) END
    WHERE id = NEW.id AND sync_id IS NULL;
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    SELECT 'folder', sync_id, '*', 'set', CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)
    FROM folders WHERE id = NEW.id;
END;

CREATE VIEW sync_media_keys AS
SELECT m.id AS media_id, e.package_name || char(31) || m.external_id AS key
FROM media m JOIN extensions e ON e.id = m.extension_id;

CREATE VIEW sync_chapter_keys AS
SELECT c.id AS chapter_id, c.media_id, mk.key || char(31) || c.external_id AS key
FROM chapters c JOIN sync_media_keys mk ON mk.media_id = c.media_id;

CREATE TRIGGER sync_progress_insert AFTER INSERT ON reading_progress
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    SELECT 'chapter', key, f.field, 'set', CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)
    FROM sync_chapter_keys, (SELECT 'completed' AS field UNION ALL SELECT 'position') f
    WHERE chapter_id = NEW.chapter_id;
END;

CREATE TRIGGER sync_progress_completed AFTER UPDATE OF completed ON reading_progress
WHEN OLD.completed IS NOT NEW.completed
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    SELECT 'chapter', key, 'completed', 'set', CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)
    FROM sync_chapter_keys WHERE chapter_id = NEW.chapter_id;
END;

CREATE TRIGGER sync_progress_position AFTER UPDATE OF progress, position_seconds, duration_seconds ON reading_progress
WHEN OLD.progress IS NOT NEW.progress OR OLD.position_seconds IS NOT NEW.position_seconds
    OR OLD.duration_seconds IS NOT NEW.duration_seconds
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    SELECT 'chapter', key, 'position', 'set', CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)
    FROM sync_chapter_keys WHERE chapter_id = NEW.chapter_id;
END;

CREATE TRIGGER sync_chapter_bookmark AFTER UPDATE OF bookmarked ON chapters
WHEN OLD.bookmarked IS NOT NEW.bookmarked
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    SELECT 'chapter', key, 'bookmarked', 'set', CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)
    FROM sync_chapter_keys WHERE chapter_id = NEW.id;
END;

CREATE TRIGGER sync_library_insert AFTER INSERT ON media WHEN NEW.added_at IS NOT NULL
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    SELECT 'library', '', key, 'add', CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)
    FROM sync_media_keys WHERE media_id = NEW.id;
END;

CREATE TRIGGER sync_library_update AFTER UPDATE OF added_at ON media
WHEN (OLD.added_at IS NULL) IS NOT (NEW.added_at IS NULL)
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    SELECT 'library', '', key, CASE WHEN NEW.added_at IS NULL THEN 'remove' ELSE 'add' END,
        CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)
    FROM sync_media_keys WHERE media_id = NEW.id;
END;

CREATE TRIGGER sync_library_delete AFTER DELETE ON media WHEN OLD.added_at IS NOT NULL
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    SELECT 'library', '', package_name || char(31) || OLD.external_id, 'remove',
        CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)
    FROM extensions WHERE id = OLD.extension_id;
END;

CREATE TRIGGER sync_folder_member_insert AFTER INSERT ON media_folders
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    SELECT 'folder_member', f.sync_id, mk.key, 'add', CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)
    FROM folders f, sync_media_keys mk WHERE f.id = NEW.folder_id AND mk.media_id = NEW.media_id;
END;

-- A media row deleted first leaves no key to read, but library removal already covers that title.
CREATE TRIGGER sync_folder_member_delete AFTER DELETE ON media_folders
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    SELECT 'folder_member', f.sync_id, mk.key, 'remove', CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)
    FROM folders f, sync_media_keys mk WHERE f.id = OLD.folder_id AND mk.media_id = OLD.media_id;
END;

CREATE TRIGGER sync_folder_update AFTER UPDATE OF name, sort_order, include_in_update, include_in_download, content_types ON folders
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    VALUES ('folder', NEW.sync_id, '*', 'set', CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER));
END;

CREATE TRIGGER sync_folder_delete AFTER DELETE ON folders WHEN OLD.sync_id IS NOT NULL
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    VALUES ('folder', OLD.sync_id, 'deleted', 'set', CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER));
END;
