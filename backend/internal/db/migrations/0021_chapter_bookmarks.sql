-- Mihon-style chapter bookmarks. The flag lives on the chapter row, which
-- source refreshes update in place, so it survives chapter syncs.
ALTER TABLE chapters ADD COLUMN bookmarked BOOLEAN NOT NULL DEFAULT FALSE;
