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

-- name: UpdateDownloadBytes :exec
UPDATE downloads SET status = ?, progress = ?, downloaded_bytes = ? WHERE id = ?;

-- name: RequeueOrphanedDownloads :exec
UPDATE downloads SET status = 'queued', progress = 0 WHERE status = 'downloading';

-- name: UpdateDownloadStats :exec
UPDATE downloads SET status = ?, progress = ?, downloaded_bytes = ?, bytes_per_sec = ? WHERE id = ?;

-- name: GetLatestDownloadForChapter :one
SELECT * FROM downloads WHERE chapter_id = ? ORDER BY created_at DESC LIMIT 1;

-- name: RetryDownload :one
UPDATE downloads SET status = 'queued', progress = 0, error = NULL, downloaded_bytes = NULL, bytes_per_sec = NULL
WHERE id = ? AND status = 'failed'
RETURNING *;


-- name: DeleteDownloadByChapter :exec
DELETE FROM downloads WHERE chapter_id = ?;
