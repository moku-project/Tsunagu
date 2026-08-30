-- name: EnqueueDownload :one
INSERT INTO downloads (chapter_id, status, position)
VALUES (?, 'queued', COALESCE((SELECT MAX(position) FROM downloads WHERE status = 'queued'), 0) + 1)
RETURNING *;

-- name: ListQueuedDownloads :many
SELECT * FROM downloads WHERE status = 'queued' ORDER BY position;

-- name: CountDownloadsByStatus :many

SELECT status, COUNT(*) AS count FROM downloads GROUP BY status;

-- name: UpdateDownloadProgress :exec
UPDATE downloads SET status = ?, progress = ? WHERE id = ?;

-- name: CompleteDownload :exec
UPDATE downloads SET status = 'done', progress = 1, completed_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: FailDownload :exec
UPDATE downloads SET status = 'failed', error = ? WHERE id = ?;

-- name: DeleteDownload :exec
DELETE FROM downloads WHERE id = ?;

-- name: RequeueOrphanedDownloads :exec
UPDATE downloads SET status = 'queued', progress = 0 WHERE status = 'downloading';

-- name: UpdateDownloadStats :exec
UPDATE downloads SET status = ?, progress = ?, downloaded_bytes = ?, bytes_per_sec = ? WHERE id = ?;

-- name: GetLatestDownloadForChapter :one
SELECT * FROM downloads WHERE chapter_id = ? ORDER BY created_at DESC LIMIT 1;

-- name: GetLatestDownloadsByChapterIDs :many

SELECT d.* FROM downloads d
WHERE d.chapter_id IN (sqlc.slice('chapter_ids'))
  AND d.id = (
    SELECT d2.id FROM downloads d2
    WHERE d2.chapter_id = d.chapter_id
    ORDER BY d2.created_at DESC LIMIT 1
  );

-- name: RetryDownload :one
UPDATE downloads SET status = 'queued', progress = 0, error = NULL, downloaded_bytes = NULL, bytes_per_sec = NULL
WHERE id = ? AND status = 'failed'
RETURNING *;

-- name: DeleteDownloadByChapter :exec
DELETE FROM downloads WHERE chapter_id = ?;

-- name: GetQueuedDownloadByChapter :one
SELECT * FROM downloads WHERE chapter_id = ? AND status = 'queued';

-- name: SetDownloadPosition :exec
UPDATE downloads SET position = ? WHERE id = ?;

-- name: ClearDownloads :exec
DELETE FROM downloads WHERE status IN ('queued', 'failed');

-- name: ListAllDownloads :many
SELECT d.*, c.media_id AS media_id FROM downloads d JOIN chapters c ON c.id = d.chapter_id ORDER BY
  CASE d.status WHEN 'downloading' THEN 0 WHEN 'queued' THEN 1 ELSE 2 END,
  d.position,
  d.created_at DESC;

-- name: DeleteDownloadsByChapters :exec
DELETE FROM downloads WHERE chapter_id IN (sqlc.slice('chapter_ids'));

-- name: CountDownloadedChaptersByMediaIDs :many

SELECT c.media_id AS media_id, COUNT(*) AS download_count
FROM downloads d
JOIN chapters c ON c.id = d.chapter_id
WHERE c.media_id IN (sqlc.slice('media_ids'))
  AND d.status = 'done'
GROUP BY c.media_id;

-- name: ClearDownloadsByStatus :exec
DELETE FROM downloads WHERE status IN (sqlc.slice('statuses'));
