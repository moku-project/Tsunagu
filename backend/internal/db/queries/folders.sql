-- name: CreateFolder :one
INSERT INTO folders (name, kind, parent_folder_id) VALUES (?, 'custom', ?)
RETURNING *;

-- name: GetFolder :one
SELECT * FROM folders WHERE id = ?;

-- name: GetFolderBySystemKey :one
SELECT * FROM folders WHERE system_key = ?;

-- name: ListFolders :many
SELECT * FROM folders ORDER BY kind, sort_order, name;

-- name: RenameFolder :one
UPDATE folders SET name = ? WHERE id = ? AND kind = 'custom'
RETURNING *;

-- name: DeleteFolder :exec
DELETE FROM folders WHERE id = ? AND kind = 'custom';

-- name: AddMediaToFolder :exec
INSERT INTO media_folders (media_id, folder_id) VALUES (?, ?)
ON CONFLICT DO NOTHING;

-- name: RemoveMediaFromFolder :exec
DELETE FROM media_folders WHERE media_id = ? AND folder_id = ?;

-- name: RemoveMediaFromFoldersByKind :exec
DELETE FROM media_folders
WHERE media_id = ?
AND folder_id IN (SELECT id FROM folders WHERE kind = ?);

-- name: ListFoldersByMediaIDs :many

SELECT mf.media_id, f.* FROM folders f
JOIN media_folders mf ON mf.folder_id = f.id
WHERE mf.media_id IN (sqlc.slice('media_ids'))
ORDER BY mf.media_id, f.kind, f.sort_order, f.name;

-- name: ListMediaInFolder :many
SELECT m.* FROM media m
JOIN media_folders mf ON mf.media_id = m.id
WHERE mf.folder_id = ?
ORDER BY m.added_at DESC;

-- name: SetFolderSortOrder :one
UPDATE folders SET sort_order = ? WHERE id = ?
RETURNING *;

-- name: UpdateFolderFlags :one
UPDATE folders SET
    include_in_update = ?,
    include_in_download = ?
WHERE id = ?
RETURNING *;
