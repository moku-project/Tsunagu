-- name: UpsertReadingProgress :one
INSERT INTO reading_progress (library_entry_id, chapter_id, progress, completed)
VALUES (?, ?, ?, ?)
ON CONFLICT(library_entry_id, chapter_id) DO UPDATE SET
    progress = excluded.progress,
    completed = excluded.completed,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetReadingProgress :one
SELECT * FROM reading_progress WHERE library_entry_id = ? AND chapter_id = ?;

-- name: ListReadingProgressByLibraryEntry :many
SELECT * FROM reading_progress WHERE library_entry_id = ? ORDER BY updated_at DESC;
