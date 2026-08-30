-- name: CreateGenre :one
INSERT INTO genres (name) VALUES (?)
ON CONFLICT(name) DO UPDATE SET name = excluded.name
RETURNING *;

-- name: ListGenres :many
SELECT * FROM genres ORDER BY name;

-- name: DeleteGenre :exec
DELETE FROM genres WHERE id = ?;

-- name: AddGenreToMedia :exec
INSERT INTO media_genres (media_id, genre_id) VALUES (?, ?)
ON CONFLICT DO NOTHING;

-- name: RemoveGenreFromMedia :exec
DELETE FROM media_genres WHERE media_id = ? AND genre_id = ?;

-- name: ClearGenresForMedia :exec
DELETE FROM media_genres WHERE media_id = ?;

-- name: ListGenresForMedia :many
SELECT g.* FROM genres g
JOIN media_genres mg ON mg.genre_id = g.id
WHERE mg.media_id = ?
ORDER BY g.name;

-- name: ListGenresByMediaIDs :many

SELECT mg.media_id, g.name
FROM genres g
JOIN media_genres mg ON mg.genre_id = g.id
WHERE mg.media_id IN (sqlc.slice('media_ids'))
ORDER BY mg.media_id, g.name;

-- name: ListMediaForGenre :many
SELECT m.* FROM media m
JOIN media_genres mg ON mg.media_id = m.id
WHERE mg.genre_id = ?
ORDER BY m.added_at DESC;
