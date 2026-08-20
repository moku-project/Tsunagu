ALTER TABLE extensions ADD COLUMN installed_version TEXT;
ALTER TABLE extensions ADD COLUMN needs_update BOOLEAN GENERATED ALWAYS AS (
    installed AND installed_version IS NOT NULL AND installed_version != version
) STORED;
