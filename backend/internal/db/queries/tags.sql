-- name: CreateTag :one
INSERT INTO tags (name) VALUES (?)
ON CONFLICT(name) DO UPDATE SET name = excluded.name
RETURNING *;

-- name: ListTags :many
SELECT * FROM tags ORDER BY name;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = ?;

-- name: AddTagToMedia :exec
INSERT INTO media_tags (media_id, tag_id) VALUES (?, ?)
ON CONFLICT DO NOTHING;

-- name: RemoveTagFromMedia :exec
DELETE FROM media_tags WHERE media_id = ? AND tag_id = ?;

-- name: ListTagsForMedia :many
SELECT t.* FROM tags t
JOIN media_tags mt ON mt.tag_id = t.id
WHERE mt.media_id = ?
ORDER BY t.name;

-- name: ListTagsByMediaIDs :many

SELECT mt.media_id, t.name
FROM tags t
JOIN media_tags mt ON mt.tag_id = t.id
WHERE mt.media_id IN (sqlc.slice('media_ids'))
ORDER BY mt.media_id, t.name;

-- name: ListMediaForTag :many
SELECT m.* FROM media m
JOIN media_tags mt ON mt.media_id = m.id
WHERE mt.tag_id = ?
ORDER BY m.added_at DESC;
