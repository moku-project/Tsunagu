-- name: CreateLibraryEntry :one
INSERT INTO library_entries (
    extension_id, extension_name, external_id, content_type, title, cover_path, description, status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(extension_id, external_id) DO UPDATE SET
    title = excluded.title,
    cover_path = excluded.cover_path,
    description = excluded.description,
    status = excluded.status
RETURNING *;

-- name: GetLibraryEntry :one
SELECT * FROM library_entries WHERE id = ?;

-- name: ListLibraryEntries :many
SELECT * FROM library_entries ORDER BY added_at DESC;

-- name: ListLibraryEntriesByContentType :many
SELECT * FROM library_entries WHERE content_type = ? ORDER BY added_at DESC;

-- name: MarkLibraryEntriesExtensionRemoved :exec
UPDATE library_entries
SET extension_removed_at = CURRENT_TIMESTAMP
WHERE extension_id = ? AND extension_removed_at IS NULL;

-- name: DeleteLibraryEntry :exec
DELETE FROM library_entries WHERE id = ?;
