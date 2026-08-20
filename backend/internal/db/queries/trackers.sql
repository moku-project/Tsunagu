-- name: UpsertTrackerAccount :one
INSERT INTO tracker_accounts (tracker_type, access_token, refresh_token, expires_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(tracker_type) DO UPDATE SET
    access_token = excluded.access_token,
    refresh_token = excluded.refresh_token,
    expires_at = excluded.expires_at
RETURNING *;

-- name: GetTrackerAccount :one
SELECT * FROM tracker_accounts WHERE tracker_type = ?;

-- name: ListTrackerAccounts :many
SELECT * FROM tracker_accounts;

-- name: DeleteTrackerAccount :exec
DELETE FROM tracker_accounts WHERE id = ?;

-- name: CreateTrackerLink :one
INSERT INTO tracker_links (library_entry_id, tracker_account_id, external_tracker_id, sync_progress)
VALUES (?, ?, ?, ?)
ON CONFLICT(library_entry_id, tracker_account_id) DO UPDATE SET
    external_tracker_id = excluded.external_tracker_id,
    sync_progress = excluded.sync_progress
RETURNING *;

-- name: GetTrackerLink :one
SELECT * FROM tracker_links WHERE library_entry_id = ? AND tracker_account_id = ?;

-- name: ListTrackerLinksByLibraryEntry :many
SELECT * FROM tracker_links WHERE library_entry_id = ?;

-- name: TouchTrackerLinkSync :exec
UPDATE tracker_links SET last_synced_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: DeleteTrackerLink :exec
DELETE FROM tracker_links WHERE id = ?;
