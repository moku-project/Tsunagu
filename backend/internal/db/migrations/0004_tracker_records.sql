ALTER TABLE tracker_accounts ADD COLUMN username TEXT NOT NULL DEFAULT '';
ALTER TABLE tracker_accounts ADD COLUMN score_format TEXT NOT NULL DEFAULT '';

ALTER TABLE tracker_links ADD COLUMN library_id TEXT;
ALTER TABLE tracker_links ADD COLUMN tracker_title TEXT NOT NULL DEFAULT '';
ALTER TABLE tracker_links ADD COLUMN remote_url TEXT NOT NULL DEFAULT '';
ALTER TABLE tracker_links ADD COLUMN status INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tracker_links ADD COLUMN last_chapter_read REAL NOT NULL DEFAULT 0;
ALTER TABLE tracker_links ADD COLUMN total_chapters INTEGER NOT NULL DEFAULT 0;
ALTER TABLE tracker_links ADD COLUMN score REAL NOT NULL DEFAULT 0;
ALTER TABLE tracker_links ADD COLUMN started_at TIMESTAMP;
ALTER TABLE tracker_links ADD COLUMN finished_at TIMESTAMP;
ALTER TABLE tracker_links ADD COLUMN private INTEGER NOT NULL DEFAULT 0;
