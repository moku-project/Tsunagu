-- name: CreateTag :one
INSERT INTO tags (name) VALUES (?)
ON CONFLICT(name) DO UPDATE SET name = excluded.name
RETURNING *;

-- name: ListTags :many
SELECT * FROM tags ORDER BY name;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = ?;

-- name: AddTagToEntry :exec
INSERT INTO library_entry_tags (library_entry_id, tag_id) VALUES (?, ?)
ON CONFLICT DO NOTHING;

-- name: RemoveTagFromEntry :exec
DELETE FROM library_entry_tags WHERE library_entry_id = ? AND tag_id = ?;

-- name: ListTagsForEntry :many
SELECT t.* FROM tags t
JOIN library_entry_tags let ON let.tag_id = t.id
WHERE let.library_entry_id = ?
ORDER BY t.name;

-- name: ListEntriesForTag :many
SELECT le.* FROM library_entries le
JOIN library_entry_tags let ON let.library_entry_id = le.id
WHERE let.tag_id = ?
ORDER BY le.added_at DESC;
