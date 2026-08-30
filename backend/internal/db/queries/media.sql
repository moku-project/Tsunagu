-- name: UpsertMediaBare :one

INSERT INTO media (
    extension_id, extension_name, external_id, content_type, title, cover_path
) VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(extension_id, external_id) DO UPDATE SET
    title = excluded.title,
    extension_name = excluded.extension_name,
    cover_path = COALESCE(excluded.cover_path, media.cover_path)
RETURNING *;

-- name: UpsertMediaDetails :one

INSERT INTO media (
    extension_id, extension_name, external_id, content_type, title, cover_path,
    description, status, author, artist, details_fetched_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)

ON CONFLICT(extension_id, external_id) DO UPDATE SET
    title       = COALESCE(NULLIF(excluded.title, ''),       media.title),
    cover_path  = COALESCE(NULLIF(excluded.cover_path, ''),  media.cover_path),
    description = COALESCE(NULLIF(excluded.description, ''),  media.description),
    status      = COALESCE(NULLIF(excluded.status, ''),      media.status),
    author      = COALESCE(NULLIF(excluded.author, ''),      media.author),
    artist      = COALESCE(NULLIF(excluded.artist, ''),      media.artist),
    details_fetched_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetMedia :one

SELECT * FROM media WHERE id = ?;

-- name: GetMediaByExtensionAndExternalID :one
SELECT * FROM media WHERE extension_id = ? AND external_id = ?;

-- name: ListMediaByIDs :many
SELECT * FROM media WHERE id IN (sqlc.slice('ids'));

-- name: MarkMediaExtensionRemoved :exec
UPDATE media
SET extension_removed_at = CURRENT_TIMESTAMP
WHERE extension_id = ? AND extension_removed_at IS NULL;

-- name: AddMediaToLibrary :one

UPDATE media SET added_at = COALESCE(added_at, CURRENT_TIMESTAMP) WHERE id = ?
RETURNING *;

-- name: RemoveMediaFromLibrary :one

UPDATE media SET added_at = NULL WHERE id = ?
RETURNING *;

-- name: TouchMediaViewed :exec

UPDATE media SET last_viewed_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: MarkChaptersSynced :exec

UPDATE media SET chapters_synced_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: UpdateMediaCoverLocalPath :exec
UPDATE media SET cover_local_path = ? WHERE id = ?;

-- name: ClearMediaCoverPaths :exec
UPDATE media SET cover_local_path = NULL;

-- name: SetMediaCoverOverride :one

UPDATE media SET cover_override = ?, cover_local_path = NULL WHERE id = ? RETURNING *;

-- name: ListUpdateTargetMediaIDs :many

SELECT m.id FROM media m
WHERE (
    EXISTS (
      SELECT 1 FROM media_folders mf
      JOIN folders f ON f.id = mf.folder_id
      WHERE mf.media_id = m.id AND f.include_in_update = 1
    )
    OR (
      m.added_at IS NOT NULL
      AND NOT EXISTS (SELECT 1 FROM media_folders mf2 WHERE mf2.media_id = m.id)
    )
  )
  AND NOT EXISTS (
    SELECT 1 FROM media_folders mfx
    JOIN folders fx ON fx.id = mfx.folder_id
    WHERE mfx.media_id = m.id AND fx.include_in_update = 0
  )
ORDER BY m.title;

-- name: GetLocalMediaByExternalID :one
SELECT * FROM media WHERE extension_id IS NULL AND external_id = ?;

-- name: ListLocalMedia :many
SELECT * FROM media WHERE extension_id IS NULL ORDER BY title;

-- name: CreateLocalMedia :one
INSERT INTO media (
    extension_id, extension_name, external_id, content_type, title,
    cover_local_path, details_fetched_at, chapters_synced_at, added_at
) VALUES (
    NULL, 'Local', ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
)
RETURNING *;

-- name: UpdateLocalMedia :one
UPDATE media SET title = ?, cover_local_path = COALESCE(?, cover_local_path) WHERE id = ? RETURNING *;
