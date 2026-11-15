-- The site each loaded source reads from, reported by the extension host.
ALTER TABLE extensions ADD COLUMN base_url TEXT NOT NULL DEFAULT '';
