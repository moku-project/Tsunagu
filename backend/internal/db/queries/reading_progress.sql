-- name: ListReadingProgressByMedia :many
SELECT * FROM reading_progress WHERE media_id = ? ORDER BY updated_at DESC;

-- name: UpsertReadingProgress :one
INSERT INTO reading_progress (media_id, chapter_id, progress, completed, position_seconds, duration_seconds)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(media_id, chapter_id) DO UPDATE SET
    progress = excluded.progress,
    completed = excluded.completed,
    position_seconds = excluded.position_seconds,
    duration_seconds = excluded.duration_seconds,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: ListReadingProgressByChapterIDs :many

SELECT * FROM reading_progress
WHERE chapter_id IN (sqlc.slice('chapter_ids'));

-- name: ListReadingProgressByMediaIDs :many

SELECT * FROM reading_progress
WHERE media_id IN (sqlc.slice('media_ids'))
ORDER BY media_id, updated_at DESC;

-- name: MaxReadChapterNumber :one

SELECT CAST(COALESCE(MAX(c.number), 0) AS REAL) AS max_number
FROM reading_progress rp
JOIN chapters c ON c.id = rp.chapter_id
WHERE rp.media_id = ? AND rp.completed = 1;

-- name: ListCompletedChapterTitles :many

SELECT c.number, c.title
FROM reading_progress rp
JOIN chapters c ON c.id = rp.chapter_id
WHERE rp.media_id = ? AND rp.completed = 1;

-- name: MarkChaptersReadUpToNumber :exec

INSERT INTO reading_progress (media_id, chapter_id, progress, completed)
SELECT sqlc.arg(media_id), c.id, 1.0, TRUE
FROM chapters c
WHERE c.media_id = sqlc.arg(media_id)
  AND c.number > 0 AND c.number <= sqlc.arg(up_to)
ON CONFLICT(media_id, chapter_id) DO UPDATE SET
    completed = TRUE, progress = 1.0, updated_at = CURRENT_TIMESTAMP
WHERE reading_progress.completed = 0;
