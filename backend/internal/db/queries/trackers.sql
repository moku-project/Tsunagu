-- name: UpsertTrackerAccount :one
INSERT INTO tracker_accounts (tracker_type, access_token, refresh_token, expires_at, username, score_format)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(tracker_type) DO UPDATE SET
    access_token = excluded.access_token,
    refresh_token = excluded.refresh_token,
    expires_at = excluded.expires_at,
    username = excluded.username,
    score_format = excluded.score_format
RETURNING *;

-- name: GetTrackerAccount :one
SELECT * FROM tracker_accounts WHERE tracker_type = ?;

-- name: ListTrackerAccounts :many
SELECT * FROM tracker_accounts;

-- name: DeleteTrackerAccountByType :exec
DELETE FROM tracker_accounts WHERE tracker_type = ?;

-- name: UpsertTrackerLink :one
INSERT INTO tracker_links (
    media_id, tracker_account_id, external_tracker_id, library_id, tracker_title,
    remote_url, status, last_chapter_read, total_chapters, score,
    started_at, finished_at, private, sync_progress
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(media_id, tracker_account_id) DO UPDATE SET
    external_tracker_id = excluded.external_tracker_id,
    library_id = excluded.library_id,
    tracker_title = excluded.tracker_title,
    remote_url = excluded.remote_url,
    status = excluded.status,
    last_chapter_read = excluded.last_chapter_read,
    total_chapters = excluded.total_chapters,
    score = excluded.score,
    started_at = excluded.started_at,
    finished_at = excluded.finished_at,
    private = excluded.private,
    last_synced_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetTrackerLink :one
SELECT * FROM tracker_links WHERE id = ?;

-- name: ListTrackerLinksByMedia :many
SELECT * FROM tracker_links WHERE media_id = ?;

-- name: DeleteTrackerLink :exec
DELETE FROM tracker_links WHERE id = ?;

-- name: ListMediaIDsWithTrackerLinks :many
SELECT DISTINCT media_id FROM tracker_links WHERE sync_progress = 1;

-- name: ListTrackerLinksByMediaIDs :many
SELECT * FROM tracker_links WHERE media_id IN (sqlc.slice('media_ids'));
