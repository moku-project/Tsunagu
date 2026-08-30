package sqlcgen

import (
	"context"
	"strings"
)

const addTagToMedia = `-- name: AddTagToMedia :exec
INSERT INTO media_tags (media_id, tag_id) VALUES (?, ?)
ON CONFLICT DO NOTHING
`

type AddTagToMediaParams struct {
	MediaID int64 `json:"media_id"`
	TagID   int64 `json:"tag_id"`
}

func (q *Queries) AddTagToMedia(ctx context.Context, arg AddTagToMediaParams) error {
	_, err := q.db.ExecContext(ctx, addTagToMedia, arg.MediaID, arg.TagID)
	return err
}

const createTag = `-- name: CreateTag :one
INSERT INTO tags (name) VALUES (?)
ON CONFLICT(name) DO UPDATE SET name = excluded.name
RETURNING id, name, created_at
`

func (q *Queries) CreateTag(ctx context.Context, name string) (Tag, error) {
	row := q.db.QueryRowContext(ctx, createTag, name)
	var i Tag
	err := row.Scan(&i.ID, &i.Name, &i.CreatedAt)
	return i, err
}

const deleteTag = `-- name: DeleteTag :exec
DELETE FROM tags WHERE id = ?
`

func (q *Queries) DeleteTag(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, deleteTag, id)
	return err
}

const listMediaForTag = `-- name: ListMediaForTag :many
SELECT m.id, m.extension_id, m.extension_name, m.external_id, m.content_type, m.title, m.cover_path, m.cover_local_path, m.description, m.status, m.author, m.artist, m.extension_removed_at, m.added_at, m.last_viewed_at, m.details_fetched_at, m.updated_at, m.chapters_synced_at, m.cover_override FROM media m
JOIN media_tags mt ON mt.media_id = m.id
WHERE mt.tag_id = ?
ORDER BY m.added_at DESC
`

func (q *Queries) ListMediaForTag(ctx context.Context, tagID int64) ([]Medium, error) {
	rows, err := q.db.QueryContext(ctx, listMediaForTag, tagID)
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

const listTags = `-- name: ListTags :many
SELECT id, name, created_at FROM tags ORDER BY name
`

func (q *Queries) ListTags(ctx context.Context) ([]Tag, error) {
	rows, err := q.db.QueryContext(ctx, listTags)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Tag{}
	for rows.Next() {
		var i Tag
		if err := rows.Scan(&i.ID, &i.Name, &i.CreatedAt); err != nil {
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

const listTagsByMediaIDs = `-- name: ListTagsByMediaIDs :many

SELECT mt.media_id, t.name
FROM tags t
JOIN media_tags mt ON mt.tag_id = t.id
WHERE mt.media_id IN (/*SLICE:media_ids*/?)
ORDER BY mt.media_id, t.name
`

type ListTagsByMediaIDsRow struct {
	MediaID int64  `json:"media_id"`
	Name    string `json:"name"`
}

func (q *Queries) ListTagsByMediaIDs(ctx context.Context, mediaIds []int64) ([]ListTagsByMediaIDsRow, error) {
	query := listTagsByMediaIDs
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
	items := []ListTagsByMediaIDsRow{}
	for rows.Next() {
		var i ListTagsByMediaIDsRow
		if err := rows.Scan(&i.MediaID, &i.Name); err != nil {
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

const listTagsForMedia = `-- name: ListTagsForMedia :many
SELECT t.id, t.name, t.created_at FROM tags t
JOIN media_tags mt ON mt.tag_id = t.id
WHERE mt.media_id = ?
ORDER BY t.name
`

func (q *Queries) ListTagsForMedia(ctx context.Context, mediaID int64) ([]Tag, error) {
	rows, err := q.db.QueryContext(ctx, listTagsForMedia, mediaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Tag{}
	for rows.Next() {
		var i Tag
		if err := rows.Scan(&i.ID, &i.Name, &i.CreatedAt); err != nil {
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

const removeTagFromMedia = `-- name: RemoveTagFromMedia :exec
DELETE FROM media_tags WHERE media_id = ? AND tag_id = ?
`

type RemoveTagFromMediaParams struct {
	MediaID int64 `json:"media_id"`
	TagID   int64 `json:"tag_id"`
}

func (q *Queries) RemoveTagFromMedia(ctx context.Context, arg RemoveTagFromMediaParams) error {
	_, err := q.db.ExecContext(ctx, removeTagFromMedia, arg.MediaID, arg.TagID)
	return err
}
