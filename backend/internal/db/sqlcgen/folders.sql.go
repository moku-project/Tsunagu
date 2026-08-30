package sqlcgen

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

const addMediaToFolder = `-- name: AddMediaToFolder :exec
INSERT INTO media_folders (media_id, folder_id) VALUES (?, ?)
ON CONFLICT DO NOTHING
`

type AddMediaToFolderParams struct {
	MediaID  int64 `json:"media_id"`
	FolderID int64 `json:"folder_id"`
}

func (q *Queries) AddMediaToFolder(ctx context.Context, arg AddMediaToFolderParams) error {
	_, err := q.db.ExecContext(ctx, addMediaToFolder, arg.MediaID, arg.FolderID)
	return err
}

const createFolder = `-- name: CreateFolder :one
INSERT INTO folders (name, kind, parent_folder_id) VALUES (?, 'custom', ?)
RETURNING id, name, kind, system_key, parent_folder_id, sort_order, created_at, include_in_update, include_in_download
`

type CreateFolderParams struct {
	Name           string        `json:"name"`
	ParentFolderID sql.NullInt64 `json:"parent_folder_id"`
}

func (q *Queries) CreateFolder(ctx context.Context, arg CreateFolderParams) (Folder, error) {
	row := q.db.QueryRowContext(ctx, createFolder, arg.Name, arg.ParentFolderID)
	var i Folder
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Kind,
		&i.SystemKey,
		&i.ParentFolderID,
		&i.SortOrder,
		&i.CreatedAt,
		&i.IncludeInUpdate,
		&i.IncludeInDownload,
	)
	return i, err
}

const deleteFolder = `-- name: DeleteFolder :exec
DELETE FROM folders WHERE id = ? AND kind = 'custom'
`

func (q *Queries) DeleteFolder(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, deleteFolder, id)
	return err
}

const getFolder = `-- name: GetFolder :one
SELECT id, name, kind, system_key, parent_folder_id, sort_order, created_at, include_in_update, include_in_download FROM folders WHERE id = ?
`

func (q *Queries) GetFolder(ctx context.Context, id int64) (Folder, error) {
	row := q.db.QueryRowContext(ctx, getFolder, id)
	var i Folder
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Kind,
		&i.SystemKey,
		&i.ParentFolderID,
		&i.SortOrder,
		&i.CreatedAt,
		&i.IncludeInUpdate,
		&i.IncludeInDownload,
	)
	return i, err
}

const getFolderBySystemKey = `-- name: GetFolderBySystemKey :one
SELECT id, name, kind, system_key, parent_folder_id, sort_order, created_at, include_in_update, include_in_download FROM folders WHERE system_key = ?
`

func (q *Queries) GetFolderBySystemKey(ctx context.Context, systemKey sql.NullString) (Folder, error) {
	row := q.db.QueryRowContext(ctx, getFolderBySystemKey, systemKey)
	var i Folder
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Kind,
		&i.SystemKey,
		&i.ParentFolderID,
		&i.SortOrder,
		&i.CreatedAt,
		&i.IncludeInUpdate,
		&i.IncludeInDownload,
	)
	return i, err
}

const listFolders = `-- name: ListFolders :many
SELECT id, name, kind, system_key, parent_folder_id, sort_order, created_at, include_in_update, include_in_download FROM folders ORDER BY kind, sort_order, name
`

func (q *Queries) ListFolders(ctx context.Context) ([]Folder, error) {
	rows, err := q.db.QueryContext(ctx, listFolders)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Folder{}
	for rows.Next() {
		var i Folder
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Kind,
			&i.SystemKey,
			&i.ParentFolderID,
			&i.SortOrder,
			&i.CreatedAt,
			&i.IncludeInUpdate,
			&i.IncludeInDownload,
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

const listFoldersByMediaIDs = `-- name: ListFoldersByMediaIDs :many

SELECT mf.media_id, f.id, f.name, f.kind, f.system_key, f.parent_folder_id, f.sort_order, f.created_at, f.include_in_update, f.include_in_download FROM folders f
JOIN media_folders mf ON mf.folder_id = f.id
WHERE mf.media_id IN (/*SLICE:media_ids*/?)
ORDER BY mf.media_id, f.kind, f.sort_order, f.name
`

type ListFoldersByMediaIDsRow struct {
	MediaID           int64          `json:"media_id"`
	ID                int64          `json:"id"`
	Name              string         `json:"name"`
	Kind              string         `json:"kind"`
	SystemKey         sql.NullString `json:"system_key"`
	ParentFolderID    sql.NullInt64  `json:"parent_folder_id"`
	SortOrder         int64          `json:"sort_order"`
	CreatedAt         time.Time      `json:"created_at"`
	IncludeInUpdate   int64          `json:"include_in_update"`
	IncludeInDownload int64          `json:"include_in_download"`
}

func (q *Queries) ListFoldersByMediaIDs(ctx context.Context, mediaIds []int64) ([]ListFoldersByMediaIDsRow, error) {
	query := listFoldersByMediaIDs
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
	items := []ListFoldersByMediaIDsRow{}
	for rows.Next() {
		var i ListFoldersByMediaIDsRow
		if err := rows.Scan(
			&i.MediaID,
			&i.ID,
			&i.Name,
			&i.Kind,
			&i.SystemKey,
			&i.ParentFolderID,
			&i.SortOrder,
			&i.CreatedAt,
			&i.IncludeInUpdate,
			&i.IncludeInDownload,
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

const listMediaInFolder = `-- name: ListMediaInFolder :many
SELECT m.id, m.extension_id, m.extension_name, m.external_id, m.content_type, m.title, m.cover_path, m.cover_local_path, m.description, m.status, m.author, m.artist, m.extension_removed_at, m.added_at, m.last_viewed_at, m.details_fetched_at, m.updated_at, m.chapters_synced_at, m.cover_override FROM media m
JOIN media_folders mf ON mf.media_id = m.id
WHERE mf.folder_id = ?
ORDER BY m.added_at DESC
`

func (q *Queries) ListMediaInFolder(ctx context.Context, folderID int64) ([]Medium, error) {
	rows, err := q.db.QueryContext(ctx, listMediaInFolder, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Medium{}
	for rows.Next() {
		var i Medium
		if err := rows.Scan(
			&i.ID,
			&i.ExtensionID,
			&i.ExtensionName,
			&i.ExternalID,
			&i.ContentType,
			&i.Title,
			&i.CoverPath,
			&i.CoverLocalPath,
			&i.Description,
			&i.Status,
			&i.Author,
			&i.Artist,
			&i.ExtensionRemovedAt,
			&i.AddedAt,
			&i.LastViewedAt,
			&i.DetailsFetchedAt,
			&i.UpdatedAt,
			&i.ChaptersSyncedAt,
			&i.CoverOverride,
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

const removeMediaFromFolder = `-- name: RemoveMediaFromFolder :exec
DELETE FROM media_folders WHERE media_id = ? AND folder_id = ?
`

type RemoveMediaFromFolderParams struct {
	MediaID  int64 `json:"media_id"`
	FolderID int64 `json:"folder_id"`
}

func (q *Queries) RemoveMediaFromFolder(ctx context.Context, arg RemoveMediaFromFolderParams) error {
	_, err := q.db.ExecContext(ctx, removeMediaFromFolder, arg.MediaID, arg.FolderID)
	return err
}

const removeMediaFromFoldersByKind = `-- name: RemoveMediaFromFoldersByKind :exec
DELETE FROM media_folders
WHERE media_id = ?
AND folder_id IN (SELECT id FROM folders WHERE kind = ?)
`

type RemoveMediaFromFoldersByKindParams struct {
	MediaID int64  `json:"media_id"`
	Kind    string `json:"kind"`
}

func (q *Queries) RemoveMediaFromFoldersByKind(ctx context.Context, arg RemoveMediaFromFoldersByKindParams) error {
	_, err := q.db.ExecContext(ctx, removeMediaFromFoldersByKind, arg.MediaID, arg.Kind)
	return err
}

const renameFolder = `-- name: RenameFolder :one
UPDATE folders SET name = ? WHERE id = ? AND kind = 'custom'
RETURNING id, name, kind, system_key, parent_folder_id, sort_order, created_at, include_in_update, include_in_download
`

type RenameFolderParams struct {
	Name string `json:"name"`
	ID   int64  `json:"id"`
}

func (q *Queries) RenameFolder(ctx context.Context, arg RenameFolderParams) (Folder, error) {
	row := q.db.QueryRowContext(ctx, renameFolder, arg.Name, arg.ID)
	var i Folder
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Kind,
		&i.SystemKey,
		&i.ParentFolderID,
		&i.SortOrder,
		&i.CreatedAt,
		&i.IncludeInUpdate,
		&i.IncludeInDownload,
	)
	return i, err
}

const setFolderSortOrder = `-- name: SetFolderSortOrder :one
UPDATE folders SET sort_order = ? WHERE id = ?
RETURNING id, name, kind, system_key, parent_folder_id, sort_order, created_at, include_in_update, include_in_download
`

type SetFolderSortOrderParams struct {
	SortOrder int64 `json:"sort_order"`
	ID        int64 `json:"id"`
}

func (q *Queries) SetFolderSortOrder(ctx context.Context, arg SetFolderSortOrderParams) (Folder, error) {
	row := q.db.QueryRowContext(ctx, setFolderSortOrder, arg.SortOrder, arg.ID)
	var i Folder
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Kind,
		&i.SystemKey,
		&i.ParentFolderID,
		&i.SortOrder,
		&i.CreatedAt,
		&i.IncludeInUpdate,
		&i.IncludeInDownload,
	)
	return i, err
}

const updateFolderFlags = `-- name: UpdateFolderFlags :one
UPDATE folders SET
    include_in_update = ?,
    include_in_download = ?
WHERE id = ?
RETURNING id, name, kind, system_key, parent_folder_id, sort_order, created_at, include_in_update, include_in_download
`

type UpdateFolderFlagsParams struct {
	IncludeInUpdate   int64 `json:"include_in_update"`
	IncludeInDownload int64 `json:"include_in_download"`
	ID                int64 `json:"id"`
}

func (q *Queries) UpdateFolderFlags(ctx context.Context, arg UpdateFolderFlagsParams) (Folder, error) {
	row := q.db.QueryRowContext(ctx, updateFolderFlags, arg.IncludeInUpdate, arg.IncludeInDownload, arg.ID)
	var i Folder
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Kind,
		&i.SystemKey,
		&i.ParentFolderID,
		&i.SortOrder,
		&i.CreatedAt,
		&i.IncludeInUpdate,
		&i.IncludeInDownload,
	)
	return i, err
}
