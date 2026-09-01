-- name: ListContentFilterRules :many
SELECT * FROM content_filter_rules ORDER BY category, field, keyword;

-- name: CreateContentFilterRule :one
INSERT INTO content_filter_rules (category, field, keyword, min_weight, block_level, is_default)
VALUES (?, ?, ?, ?, ?, 0)
ON CONFLICT(field, keyword, category) DO UPDATE SET
    min_weight = excluded.min_weight,
    block_level = excluded.block_level
RETURNING *;

-- name: DeleteContentFilterRule :exec
DELETE FROM content_filter_rules WHERE id = ?;

-- name: DeleteUserContentFilterRules :exec
DELETE FROM content_filter_rules WHERE is_default = 0;

-- name: SetMediaContentBlockRank :exec
UPDATE media SET content_block_rank = ? WHERE id = ?;

-- name: ListRecomputableMediaIDs :many
SELECT id FROM media WHERE added_at IS NOT NULL ORDER BY id;

-- name: GetContentFilterInputs :one
SELECT title, description FROM media WHERE id = ?;

-- name: ListMediaIDsWithSparseTags :many
SELECT m.id
FROM media m
JOIN metadata_links ml ON ml.media_id = m.id AND ml.provider = 'anilist'
WHERE m.added_at IS NOT NULL
  AND (SELECT COUNT(*) FROM media_tags mt WHERE mt.media_id = m.id) < ?;

-- name: LibraryTagFacets :many
SELECT t.name AS name,
       CAST(COUNT(*) AS INTEGER) AS count,
       CAST(COALESCE(MAX(mt.weight), 0) AS INTEGER) AS max_weight
FROM media_tags mt
JOIN tags t ON t.id = mt.tag_id
JOIN media m ON m.id = mt.media_id
WHERE m.added_at IS NOT NULL
GROUP BY t.id
HAVING COUNT(*) >= sqlc.arg('min_count')
ORDER BY count DESC, name;

-- name: LibraryGenreFacets :many
SELECT g.name AS name,
       CAST(COUNT(*) AS INTEGER) AS count,
       CAST(0 AS INTEGER) AS max_weight
FROM media_genres mg
JOIN genres g ON g.id = mg.genre_id
JOIN media m ON m.id = mg.media_id
WHERE m.added_at IS NOT NULL
GROUP BY g.id
HAVING COUNT(*) >= sqlc.arg('min_count')
ORDER BY count DESC, name;
