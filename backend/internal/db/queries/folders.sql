-- name: CreateFolder :one
INSERT INTO folders (name, kind, parent_folder_id) VALUES (?, 'custom', ?)
RETURNING *;

-- name: GetFolder :one
SELECT * FROM folders WHERE id = ?;

-- name: GetFolderBySystemKey :one
SELECT * FROM folders WHERE system_key = ?;

-- name: ListFolders :many
SELECT * FROM folders ORDER BY kind, sort_order, name;

-- name: ListChildFolders :many
SELECT * FROM folders WHERE parent_folder_id = ? ORDER BY sort_order, name;

-- name: RenameFolder :one
UPDATE folders SET name = ? WHERE id = ? AND kind = 'custom'
RETURNING *;

-- name: DeleteFolder :exec
DELETE FROM folders WHERE id = ? AND kind = 'custom';

-- name: AddEntryToFolder :exec
INSERT INTO library_entry_folders (library_entry_id, folder_id) VALUES (?, ?)
ON CONFLICT DO NOTHING;

-- name: RemoveEntryFromFolder :exec
DELETE FROM library_entry_folders WHERE library_entry_id = ? AND folder_id = ?;

-- name: RemoveEntryFromFoldersByKind :exec
DELETE FROM library_entry_folders
WHERE library_entry_id = ?
AND folder_id IN (SELECT id FROM folders WHERE kind = ?);

-- name: ListFoldersForEntry :many
SELECT f.* FROM folders f
JOIN library_entry_folders lef ON lef.folder_id = f.id
WHERE lef.library_entry_id = ?
ORDER BY f.kind, f.sort_order, f.name;

-- name: ListEntriesInFolder :many
SELECT le.* FROM library_entries le
JOIN library_entry_folders lef ON lef.library_entry_id = le.id
WHERE lef.folder_id = ?
ORDER BY le.added_at DESC;
