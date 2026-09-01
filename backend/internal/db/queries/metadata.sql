-- name: GetMetadataLink :one
SELECT * FROM metadata_links WHERE media_id = ? AND provider = ?;

-- name: ListMetadataLinksByMedia :many
SELECT * FROM metadata_links WHERE media_id = ?;

-- name: UpsertMetadataLink :one
INSERT INTO metadata_links (media_id, provider, provider_id, provider_url, cover_url, confidence, locked)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(media_id, provider) DO UPDATE SET
    provider_id = excluded.provider_id,
    provider_url = excluded.provider_url,
    cover_url = excluded.cover_url,
    confidence = excluded.confidence,
    locked = excluded.locked,
    matched_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: BackfillMissingCoversFromMetadata :execrows
UPDATE media SET cover_path = (
    SELECT ml.cover_url FROM metadata_links ml
    WHERE ml.media_id = media.id AND ml.cover_url IS NOT NULL AND ml.cover_url != ''
    ORDER BY ml.confidence DESC LIMIT 1
)
WHERE (cover_path IS NULL OR cover_path = '')
  AND (cover_local_path IS NULL OR cover_local_path = '')
  AND (cover_override IS NULL OR cover_override = '')
  AND EXISTS (
    SELECT 1 FROM metadata_links ml
    WHERE ml.media_id = media.id AND ml.cover_url IS NOT NULL AND ml.cover_url != ''
  );

-- name: GetMediaMetadataCover :one
SELECT cover_url FROM metadata_links
WHERE media_id = ? AND cover_url IS NOT NULL AND cover_url != ''
ORDER BY confidence DESC LIMIT 1;

-- name: DeleteMetadataLink :exec
DELETE FROM metadata_links WHERE media_id = ? AND provider = ?;

-- name: GapFillMediaMetadata :one

UPDATE media SET
    description = CASE WHEN description IS NULL OR description = ''
        THEN NULLIF(CAST(sqlc.arg(description) AS TEXT), '') ELSE description END,
    status = CASE WHEN status IS NULL OR status = ''
        THEN NULLIF(CAST(sqlc.arg(status) AS TEXT), '') ELSE status END,
    author = CASE WHEN author IS NULL OR author = ''
        THEN NULLIF(CAST(sqlc.arg(author) AS TEXT), '') ELSE author END,
    cover_path = CASE
        WHEN (cover_path IS NULL OR cover_path = '')
         AND (cover_local_path IS NULL OR cover_local_path = '')
         AND (cover_override IS NULL OR cover_override = '')
        THEN NULLIF(CAST(sqlc.arg(cover_path) AS TEXT), '') ELSE cover_path END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: ListMediaIDsWithoutMetadataLink :many

SELECT m.id
FROM media m
WHERE m.added_at IS NOT NULL
  AND m.extension_id IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM metadata_links ml WHERE ml.media_id = m.id);

-- name: ListMetadataLinksByMediaIDs :many
SELECT * FROM metadata_links WHERE media_id IN (sqlc.slice('media_ids'));
