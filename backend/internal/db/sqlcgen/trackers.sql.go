package sqlcgen

import (
	"context"
	"database/sql"
	"strings"
)

const deleteTrackerAccountByType = `-- name: DeleteTrackerAccountByType :exec
DELETE FROM tracker_accounts WHERE tracker_type = ?
`

func (q *Queries) DeleteTrackerAccountByType(ctx context.Context, trackerType string) error {
	_, err := q.db.ExecContext(ctx, deleteTrackerAccountByType, trackerType)
	return err
}

const deleteTrackerLink = `-- name: DeleteTrackerLink :exec
DELETE FROM tracker_links WHERE id = ?
`

func (q *Queries) DeleteTrackerLink(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, deleteTrackerLink, id)
	return err
}

const getTrackerAccount = `-- name: GetTrackerAccount :one
SELECT id, tracker_type, access_token, refresh_token, expires_at, username, score_format FROM tracker_accounts WHERE tracker_type = ?
`

func (q *Queries) GetTrackerAccount(ctx context.Context, trackerType string) (TrackerAccount, error) {
	row := q.db.QueryRowContext(ctx, getTrackerAccount, trackerType)
	var i TrackerAccount
	err := row.Scan(
		&i.ID,
		&i.TrackerType,
		&i.AccessToken,
		&i.RefreshToken,
		&i.ExpiresAt,
		&i.Username,
		&i.ScoreFormat,
	)
	return i, err
}

const getTrackerLink = `-- name: GetTrackerLink :one
SELECT id, media_id, tracker_account_id, external_tracker_id, sync_progress, last_synced_at, library_id, tracker_title, remote_url, status, last_chapter_read, total_chapters, score, started_at, finished_at, private FROM tracker_links WHERE id = ?
`

func (q *Queries) GetTrackerLink(ctx context.Context, id int64) (TrackerLink, error) {
	row := q.db.QueryRowContext(ctx, getTrackerLink, id)
	var i TrackerLink
	err := row.Scan(
		&i.ID,
		&i.MediaID,
		&i.TrackerAccountID,
		&i.ExternalTrackerID,
		&i.SyncProgress,
		&i.LastSyncedAt,
		&i.LibraryID,
		&i.TrackerTitle,
		&i.RemoteUrl,
		&i.Status,
		&i.LastChapterRead,
		&i.TotalChapters,
		&i.Score,
		&i.StartedAt,
		&i.FinishedAt,
		&i.Private,
	)
	return i, err
}

const listMediaIDsWithTrackerLinks = `-- name: ListMediaIDsWithTrackerLinks :many
SELECT DISTINCT media_id FROM tracker_links WHERE sync_progress = 1
`

func (q *Queries) ListMediaIDsWithTrackerLinks(ctx context.Context) ([]int64, error) {
	rows, err := q.db.QueryContext(ctx, listMediaIDsWithTrackerLinks)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []int64{}
	for rows.Next() {
		var media_id int64
		if err := rows.Scan(&media_id); err != nil {
			return nil, err
		}
		items = append(items, media_id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listTrackerAccounts = `-- name: ListTrackerAccounts :many
SELECT id, tracker_type, access_token, refresh_token, expires_at, username, score_format FROM tracker_accounts
`

func (q *Queries) ListTrackerAccounts(ctx context.Context) ([]TrackerAccount, error) {
	rows, err := q.db.QueryContext(ctx, listTrackerAccounts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []TrackerAccount{}
	for rows.Next() {
		var i TrackerAccount
		if err := rows.Scan(
			&i.ID,
			&i.TrackerType,
			&i.AccessToken,
			&i.RefreshToken,
			&i.ExpiresAt,
			&i.Username,
			&i.ScoreFormat,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listTrackerLinksByMedia = `-- name: ListTrackerLinksByMedia :many
SELECT id, media_id, tracker_account_id, external_tracker_id, sync_progress, last_synced_at, library_id, tracker_title, remote_url, status, last_chapter_read, total_chapters, score, started_at, finished_at, private FROM tracker_links WHERE media_id = ?
`

func (q *Queries) ListTrackerLinksByMedia(ctx context.Context, mediaID int64) ([]TrackerLink, error) {
	rows, err := q.db.QueryContext(ctx, listTrackerLinksByMedia, mediaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []TrackerLink{}
	for rows.Next() {
		var i TrackerLink
		if err := rows.Scan(
			&i.ID,
			&i.MediaID,
			&i.TrackerAccountID,
			&i.ExternalTrackerID,
			&i.SyncProgress,
			&i.LastSyncedAt,
			&i.LibraryID,
			&i.TrackerTitle,
			&i.RemoteUrl,
			&i.Status,
			&i.LastChapterRead,
			&i.TotalChapters,
			&i.Score,
			&i.StartedAt,
			&i.FinishedAt,
			&i.Private,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listTrackerLinksByMediaIDs = `-- name: ListTrackerLinksByMediaIDs :many
SELECT id, media_id, tracker_account_id, external_tracker_id, sync_progress, last_synced_at, library_id, tracker_title, remote_url, status, last_chapter_read, total_chapters, score, started_at, finished_at, private FROM tracker_links WHERE media_id IN (/*SLICE:media_ids*/?)
`

func (q *Queries) ListTrackerLinksByMediaIDs(ctx context.Context, mediaIds []int64) ([]TrackerLink, error) {
	query := listTrackerLinksByMediaIDs
	var queryParams []interface{}
	if len(mediaIds) > 0 {
		for _, v := range mediaIds {
			queryParams = append(queryParams, v)
		}
		query = strings.Replace(query, "/*SLICE:media_ids*/?", strings.Repeat(",?", len(mediaIds))[1:], 1)
	} else {
		query = strings.Replace(query, "/*SLICE:media_ids*/?", "NULL", 1)
	}
	rows, err := q.db.QueryContext(ctx, query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []TrackerLink{}
	for rows.Next() {
		var i TrackerLink
		if err := rows.Scan(
			&i.ID,
			&i.MediaID,
			&i.TrackerAccountID,
			&i.ExternalTrackerID,
			&i.SyncProgress,
			&i.LastSyncedAt,
			&i.LibraryID,
			&i.TrackerTitle,
			&i.RemoteUrl,
			&i.Status,
			&i.LastChapterRead,
			&i.TotalChapters,
			&i.Score,
			&i.StartedAt,
			&i.FinishedAt,
			&i.Private,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const upsertTrackerAccount = `-- name: UpsertTrackerAccount :one
INSERT INTO tracker_accounts (tracker_type, access_token, refresh_token, expires_at, username, score_format)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(tracker_type) DO UPDATE SET
    access_token = excluded.access_token,
    refresh_token = excluded.refresh_token,
    expires_at = excluded.expires_at,
    username = excluded.username,
    score_format = excluded.score_format
RETURNING id, tracker_type, access_token, refresh_token, expires_at, username, score_format
`

type UpsertTrackerAccountParams struct {
	TrackerType  string         `json:"tracker_type"`
	AccessToken  string         `json:"access_token"`
	RefreshToken sql.NullString `json:"refresh_token"`
	ExpiresAt    sql.NullTime   `json:"expires_at"`
	Username     string         `json:"username"`
	ScoreFormat  string         `json:"score_format"`
}

func (q *Queries) UpsertTrackerAccount(ctx context.Context, arg UpsertTrackerAccountParams) (TrackerAccount, error) {
	row := q.db.QueryRowContext(ctx, upsertTrackerAccount,
		arg.TrackerType,
		arg.AccessToken,
		arg.RefreshToken,
		arg.ExpiresAt,
		arg.Username,
		arg.ScoreFormat,
	)
	var i TrackerAccount
	err := row.Scan(
		&i.ID,
		&i.TrackerType,
		&i.AccessToken,
		&i.RefreshToken,
		&i.ExpiresAt,
		&i.Username,
		&i.ScoreFormat,
	)
	return i, err
}

const upsertTrackerLink = `-- name: UpsertTrackerLink :one
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
RETURNING id, media_id, tracker_account_id, external_tracker_id, sync_progress, last_synced_at, library_id, tracker_title, remote_url, status, last_chapter_read, total_chapters, score, started_at, finished_at, private
`

type UpsertTrackerLinkParams struct {
	MediaID           int64          `json:"media_id"`
	TrackerAccountID  int64          `json:"tracker_account_id"`
	ExternalTrackerID string         `json:"external_tracker_id"`
	LibraryID         sql.NullString `json:"library_id"`
	TrackerTitle      string         `json:"tracker_title"`
	RemoteUrl         string         `json:"remote_url"`
	Status            int64          `json:"status"`
	LastChapterRead   float64        `json:"last_chapter_read"`
	TotalChapters     int64          `json:"total_chapters"`
	Score             float64        `json:"score"`
	StartedAt         sql.NullTime   `json:"started_at"`
	FinishedAt        sql.NullTime   `json:"finished_at"`
	Private           int64          `json:"private"`
	SyncProgress      bool           `json:"sync_progress"`
}

func (q *Queries) UpsertTrackerLink(ctx context.Context, arg UpsertTrackerLinkParams) (TrackerLink, error) {
	row := q.db.QueryRowContext(ctx, upsertTrackerLink,
		arg.MediaID,
		arg.TrackerAccountID,
		arg.ExternalTrackerID,
		arg.LibraryID,
		arg.TrackerTitle,
		arg.RemoteUrl,
		arg.Status,
		arg.LastChapterRead,
		arg.TotalChapters,
		arg.Score,
		arg.StartedAt,
		arg.FinishedAt,
		arg.Private,
		arg.SyncProgress,
	)
	var i TrackerLink
	err := row.Scan(
		&i.ID,
		&i.MediaID,
		&i.TrackerAccountID,
		&i.ExternalTrackerID,
		&i.SyncProgress,
		&i.LastSyncedAt,
		&i.LibraryID,
		&i.TrackerTitle,
		&i.RemoteUrl,
		&i.Status,
		&i.LastChapterRead,
		&i.TotalChapters,
		&i.Score,
		&i.StartedAt,
		&i.FinishedAt,
		&i.Private,
	)
	return i, err
}
