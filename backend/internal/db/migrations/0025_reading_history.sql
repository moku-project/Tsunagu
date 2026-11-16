-- Reading history lives here, not in each app, so every device and every app shows the same history and stats.
CREATE TABLE reading_sessions (
    id TEXT PRIMARY KEY,
    media_id INTEGER NOT NULL REFERENCES media(id) ON DELETE CASCADE,
    start_chapter_id INTEGER REFERENCES chapters(id) ON DELETE SET NULL,
    end_chapter_id INTEGER REFERENCES chapters(id) ON DELETE SET NULL,
    start_page INTEGER NOT NULL DEFAULT 0,
    end_page INTEGER NOT NULL DEFAULT 0,
    started_at INTEGER NOT NULL,
    ended_at INTEGER NOT NULL,
    duration_ms INTEGER NOT NULL,
    chapters INTEGER NOT NULL DEFAULT 1,
    -- Cleared from the history list; still counted in stats.
    hidden INTEGER NOT NULL DEFAULT 0,
    -- Imported sessions are already counted in imported totals.
    counted INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX idx_reading_sessions_ended ON reading_sessions(ended_at);
CREATE INDEX idx_reading_sessions_media ON reading_sessions(media_id);

-- Per-day totals an app kept before history moved here, by the device that imported them and the reader's local date.
CREATE TABLE reading_days_imported (
    source TEXT NOT NULL,
    day TEXT NOT NULL,
    seconds INTEGER NOT NULL,
    chapters INTEGER NOT NULL,
    PRIMARY KEY (source, day)
);

CREATE TRIGGER sync_session_insert AFTER INSERT ON reading_sessions
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    VALUES ('session', NEW.id, 'data', 'set', CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER));
END;

CREATE TRIGGER sync_session_update AFTER UPDATE ON reading_sessions
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    VALUES ('session', NEW.id, 'data', 'set', CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER));
END;

-- A title deleted on this device takes its sessions with it, but that isn't a reason to delete them elsewhere.
CREATE TRIGGER sync_session_delete AFTER DELETE ON reading_sessions
WHEN EXISTS (SELECT 1 FROM media WHERE id = OLD.media_id)
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    VALUES ('session', OLD.id, 'data', 'set', CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER));
END;

CREATE TRIGGER sync_reading_day_insert AFTER INSERT ON reading_days_imported
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    VALUES ('reading_day', NEW.source || char(31) || NEW.day, 'totals', 'set', CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER));
END;

CREATE TRIGGER sync_reading_day_update AFTER UPDATE ON reading_days_imported
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    VALUES ('reading_day', NEW.source || char(31) || NEW.day, 'totals', 'set', CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER));
END;

CREATE TRIGGER sync_reading_day_delete AFTER DELETE ON reading_days_imported
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    VALUES ('reading_day', OLD.source || char(31) || OLD.day, 'totals', 'set', CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER));
END;

-- The page a reader stopped on in each chapter, so resuming lands on it on any device; progress alone can't name a page.
CREATE TABLE reading_pages (
    chapter_id INTEGER PRIMARY KEY REFERENCES chapters(id) ON DELETE CASCADE,
    page INTEGER NOT NULL
);

CREATE TRIGGER sync_reading_page_insert AFTER INSERT ON reading_pages
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    SELECT 'chapter', key, 'position', 'set', CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)
    FROM sync_chapter_keys WHERE chapter_id = NEW.chapter_id;
END;

CREATE TRIGGER sync_reading_page_update AFTER UPDATE OF page ON reading_pages WHEN OLD.page IS NOT NEW.page
BEGIN
    INSERT INTO sync_dirty (kind, key, field, op, recorded_at)
    SELECT 'chapter', key, 'position', 'set', CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)
    FROM sync_chapter_keys WHERE chapter_id = NEW.chapter_id;
END;
