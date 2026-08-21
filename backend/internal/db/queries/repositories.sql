-- name: CreateRepository :one
INSERT INTO repositories (index_url, name, content_type)
VALUES (?, ?, ?)
RETURNING *;

-- name: GetRepository :one
SELECT * FROM repositories WHERE id = ?;

-- name: GetRepositoryByURL :one
SELECT * FROM repositories WHERE index_url = ?;

-- name: ListRepositories :many
SELECT * FROM repositories ORDER BY added_at;

-- name: TouchRepositorySync :exec
UPDATE repositories SET last_synced_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: DeleteRepository :exec
DELETE FROM repositories WHERE id = ?;

-- name: UpdateRepositoryName :one
UPDATE repositories SET name = ? WHERE id = ?
RETURNING *;
