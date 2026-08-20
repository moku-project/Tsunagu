-- name: EnqueueDownload :one
INSERT INTO downloads (chapter_id, status)
VALUES (?, 'queued')
RETURNING *;

-- name: GetDownload :one
SELECT * FROM downloads WHERE id = ?;

-- name: ListQueuedDownloads :many
SELECT * FROM downloads WHERE status = 'queued' ORDER BY created_at;

-- name: ListDownloadsByStatus :many
SELECT * FROM downloads WHERE status = ? ORDER BY created_at;

-- name: UpdateDownloadProgress :exec
UPDATE downloads SET status = ?, progress = ? WHERE id = ?;

-- name: CompleteDownload :exec
UPDATE downloads SET status = 'done', progress = 1, completed_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: FailDownload :exec
UPDATE downloads SET status = 'failed', error = ? WHERE id = ?;

-- name: DeleteDownload :exec
DELETE FROM downloads WHERE id = ?;
