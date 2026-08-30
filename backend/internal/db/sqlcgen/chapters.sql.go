package sqlcgen

import (
	"context"
	"database/sql"
	"strings"
)

const countChaptersByMediaIDs = `-- name: CountChaptersByMediaIDs :many
SELECT media_id, COUNT(*) AS chapter_count
FROM chapters
WHERE media_id IN (/*SLICE:media_ids*/?)
GROUP BY media_id
`

type CountChaptersByMediaIDsRow struct {
	MediaID      int64 `json:"media_id"`
	ChapterCount int64 `json:"chapter_count"`
}

func (q *Queries) CountChaptersByMediaIDs(ctx context.Context, mediaIds []int64) ([]CountChaptersByMediaIDsRow, error) {
	query := countChaptersByMediaIDs
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
	items := []CountChaptersByMediaIDsRow{}
	for rows.Next() {
		var i CountChaptersByMediaIDsRow
		if err := rows.Scan(&i.MediaID, &i.ChapterCount); err != nil {
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

const countDownloadedPagesByChapterIDs = `-- name: CountDownloadedPagesByChapterIDs :many

SELECT chapter_id, COUNT(*) AS page_count
FROM manga_pages
WHERE chapter_id IN (/*SLICE:chapter_ids*/?)
  AND local_path IS NOT NULL AND local_path != ''
GROUP BY chapter_id
`

type CountDownloadedPagesByChapterIDsRow struct {
	ChapterID int64 `json:"chapter_id"`
	PageCount int64 `json:"page_count"`
}

func (q *Queries) CountDownloadedPagesByChapterIDs(ctx context.Context, chapterIds []int64) ([]CountDownloadedPagesByChapterIDsRow, error) {
	query := countDownloadedPagesByChapterIDs
	var queryParams []interface{}
	if len(chapterIds) > 0 {
		for _, v := range chapterIds {
			queryParams = append(queryParams, v)
		}
		query = strings.Replace(query, "/*SLICE:chapter_ids*/?", strings.Repeat(",?", len(chapterIds))[1:], 1)
	} else {
		query = strings.Replace(query, "/*SLICE:chapter_ids*/?", "NULL", 1)
	}
	rows, err := q.db.QueryContext(ctx, query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []CountDownloadedPagesByChapterIDsRow{}
	for rows.Next() {
		var i CountDownloadedPagesByChapterIDsRow
		if err := rows.Scan(&i.ChapterID, &i.PageCount); err != nil {
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

const countUnreadChaptersByMediaIDs = `-- name: CountUnreadChaptersByMediaIDs :many

SELECT c.media_id AS media_id, COUNT(*) AS unread_count
FROM chapters c
LEFT JOIN reading_progress rp
    ON rp.chapter_id = c.id AND rp.media_id = c.media_id
WHERE c.media_id IN (/*SLICE:media_ids*/?)
  AND (rp.completed IS NULL OR rp.completed = FALSE)
GROUP BY c.media_id
`

type CountUnreadChaptersByMediaIDsRow struct {
	MediaID     int64 `json:"media_id"`
	UnreadCount int64 `json:"unread_count"`
}

func (q *Queries) CountUnreadChaptersByMediaIDs(ctx context.Context, mediaIds []int64) ([]CountUnreadChaptersByMediaIDsRow, error) {
	query := countUnreadChaptersByMediaIDs
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
	items := []CountUnreadChaptersByMediaIDsRow{}
	for rows.Next() {
		var i CountUnreadChaptersByMediaIDsRow
		if err := rows.Scan(&i.MediaID, &i.UnreadCount); err != nil {
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

const createChapter = `-- name: CreateChapter :one

INSERT INTO chapters (media_id, external_id, title, number, uploaded_at, source_order, scanlator, first_seen_at)
VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(media_id, external_id) DO UPDATE SET
    title = excluded.title,
    number = excluded.number,
    uploaded_at = excluded.uploaded_at,
    source_order = excluded.source_order,
    scanlator = excluded.scanlator
RETURNING id, media_id, external_id, title, number, uploaded_at, source_order, first_seen_at, scanlator
`

type CreateChapterParams struct {
	MediaID     int64           `json:"media_id"`
	ExternalID  string          `json:"external_id"`
	Title       sql.NullString  `json:"title"`
	Number      sql.NullFloat64 `json:"number"`
	UploadedAt  sql.NullInt64   `json:"uploaded_at"`
	SourceOrder sql.NullInt64   `json:"source_order"`
	Scanlator   string          `json:"scanlator"`
}

func (q *Queries) CreateChapter(ctx context.Context, arg CreateChapterParams) (Chapter, error) {
	row := q.db.QueryRowContext(ctx, createChapter,
		arg.MediaID,
		arg.ExternalID,
		arg.Title,
		arg.Number,
		arg.UploadedAt,
		arg.SourceOrder,
		arg.Scanlator,
	)
	var i Chapter
	err := row.Scan(
		&i.ID,
		&i.MediaID,
		&i.ExternalID,
		&i.Title,
		&i.Number,
		&i.UploadedAt,
		&i.SourceOrder,
		&i.FirstSeenAt,
		&i.Scanlator,
	)
	return i, err
}

const deleteAnimeEpisodeStream = `-- name: DeleteAnimeEpisodeStream :exec
DELETE FROM anime_episode_streams WHERE chapter_id = ?
`

func (q *Queries) DeleteAnimeEpisodeStream(ctx context.Context, chapterID int64) error {
	_, err := q.db.ExecContext(ctx, deleteAnimeEpisodeStream, chapterID)
	return err
}

const deleteMangaPage = `-- name: DeleteMangaPage :exec
DELETE FROM manga_pages WHERE chapter_id = ? AND page_number = ?
`

type DeleteMangaPageParams struct {
	ChapterID  int64 `json:"chapter_id"`
	PageNumber int64 `json:"page_number"`
}

func (q *Queries) DeleteMangaPage(ctx context.Context, arg DeleteMangaPageParams) error {
	_, err := q.db.ExecContext(ctx, deleteMangaPage, arg.ChapterID, arg.PageNumber)
	return err
}

const deleteMangaPages = `-- name: DeleteMangaPages :exec
DELETE FROM manga_pages WHERE chapter_id = ?
`

func (q *Queries) DeleteMangaPages(ctx context.Context, chapterID int64) error {
	_, err := q.db.ExecContext(ctx, deleteMangaPages, chapterID)
	return err
}

const deleteNovelChapterContent = `-- name: DeleteNovelChapterContent :exec
DELETE FROM novel_chapter_content WHERE chapter_id = ?
`

func (q *Queries) DeleteNovelChapterContent(ctx context.Context, chapterID int64) error {
	_, err := q.db.ExecContext(ctx, deleteNovelChapterContent, chapterID)
	return err
}

const getAnimeEpisodeStream = `-- name: GetAnimeEpisodeStream :one
SELECT chapter_id, stream_url, local_path FROM anime_episode_streams WHERE chapter_id = ?
`

func (q *Queries) GetAnimeEpisodeStream(ctx context.Context, chapterID int64) (AnimeEpisodeStream, error) {
	row := q.db.QueryRowContext(ctx, getAnimeEpisodeStream, chapterID)
	var i AnimeEpisodeStream
	err := row.Scan(&i.ChapterID, &i.StreamUrl, &i.LocalPath)
	return i, err
}

const getChapter = `-- name: GetChapter :one
SELECT id, media_id, external_id, title, number, uploaded_at, source_order, first_seen_at, scanlator FROM chapters WHERE id = ?
`

func (q *Queries) GetChapter(ctx context.Context, id int64) (Chapter, error) {
	row := q.db.QueryRowContext(ctx, getChapter, id)
	var i Chapter
	err := row.Scan(
		&i.ID,
		&i.MediaID,
		&i.ExternalID,
		&i.Title,
		&i.Number,
		&i.UploadedAt,
		&i.SourceOrder,
		&i.FirstSeenAt,
		&i.Scanlator,
	)
	return i, err
}

const getChapterByMediaAndExternalID = `-- name: GetChapterByMediaAndExternalID :one
SELECT id, media_id, external_id, title, number, uploaded_at, source_order, first_seen_at, scanlator FROM chapters WHERE media_id = ? AND external_id = ?
`

type GetChapterByMediaAndExternalIDParams struct {
	MediaID    int64  `json:"media_id"`
	ExternalID string `json:"external_id"`
}

func (q *Queries) GetChapterByMediaAndExternalID(ctx context.Context, arg GetChapterByMediaAndExternalIDParams) (Chapter, error) {
	row := q.db.QueryRowContext(ctx, getChapterByMediaAndExternalID, arg.MediaID, arg.ExternalID)
	var i Chapter
	err := row.Scan(
		&i.ID,
		&i.MediaID,
		&i.ExternalID,
		&i.Title,
		&i.Number,
		&i.UploadedAt,
		&i.SourceOrder,
		&i.FirstSeenAt,
		&i.Scanlator,
	)
	return i, err
}

const getChapterDownloadContext = `-- name: GetChapterDownloadContext :one
SELECT
    c.id AS chapter_id,
    c.external_id AS source_chapter_id,
    c.title AS chapter_title,
    c.number AS chapter_number,
    m.external_id AS source_entry_id,
    m.title AS library_title,
    m.content_type AS content_type,
    e.package_name AS extension_package_name
FROM chapters c
JOIN media m ON m.id = c.media_id
JOIN extensions e ON e.id = m.extension_id
WHERE c.id = ?
`

type GetChapterDownloadContextRow struct {
	ChapterID            int64           `json:"chapter_id"`
	SourceChapterID      string          `json:"source_chapter_id"`
	ChapterTitle         sql.NullString  `json:"chapter_title"`
	ChapterNumber        sql.NullFloat64 `json:"chapter_number"`
	SourceEntryID        string          `json:"source_entry_id"`
	LibraryTitle         string          `json:"library_title"`
	ContentType          string          `json:"content_type"`
	ExtensionPackageName string          `json:"extension_package_name"`
}

func (q *Queries) GetChapterDownloadContext(ctx context.Context, id int64) (GetChapterDownloadContextRow, error) {
	row := q.db.QueryRowContext(ctx, getChapterDownloadContext, id)
	var i GetChapterDownloadContextRow
	err := row.Scan(
		&i.ChapterID,
		&i.SourceChapterID,
		&i.ChapterTitle,
		&i.ChapterNumber,
		&i.SourceEntryID,
		&i.LibraryTitle,
		&i.ContentType,
		&i.ExtensionPackageName,
	)
	return i, err
}

const getNovelChapterContent = `-- name: GetNovelChapterContent :one
SELECT chapter_id, local_path FROM novel_chapter_content WHERE chapter_id = ?
`

func (q *Queries) GetNovelChapterContent(ctx context.Context, chapterID int64) (NovelChapterContent, error) {
	row := q.db.QueryRowContext(ctx, getNovelChapterContent, chapterID)
	var i NovelChapterContent
	err := row.Scan(&i.ChapterID, &i.LocalPath)
	return i, err
}

const latestChapterByMediaIDs = `-- name: LatestChapterByMediaIDs :many

SELECT c.id, c.media_id, c.external_id, c.title, c.number, c.uploaded_at, c.source_order, c.first_seen_at, c.scanlator
FROM chapters c
WHERE c.media_id IN (/*SLICE:media_ids*/?)
  AND c.id = (
    SELECT c2.id FROM chapters c2
    WHERE c2.media_id = c.media_id
    ORDER BY c2.number DESC, c2.source_order DESC
    LIMIT 1
  )
`

func (q *Queries) LatestChapterByMediaIDs(ctx context.Context, mediaIds []int64) ([]Chapter, error) {
	query := latestChapterByMediaIDs
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
	items := []Chapter{}
	for rows.Next() {
		var i Chapter
		if err := rows.Scan(
			&i.ID,
			&i.MediaID,
			&i.ExternalID,
			&i.Title,
			&i.Number,
			&i.UploadedAt,
			&i.SourceOrder,
			&i.FirstSeenAt,
			&i.Scanlator,
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

const listAllEpisodeStreamPaths = `-- name: ListAllEpisodeStreamPaths :many
SELECT chapter_id, local_path FROM anime_episode_streams
WHERE local_path IS NOT NULL AND local_path != ''
`

type ListAllEpisodeStreamPathsRow struct {
	ChapterID int64          `json:"chapter_id"`
	LocalPath sql.NullString `json:"local_path"`
}

func (q *Queries) ListAllEpisodeStreamPaths(ctx context.Context) ([]ListAllEpisodeStreamPathsRow, error) {
	rows, err := q.db.QueryContext(ctx, listAllEpisodeStreamPaths)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListAllEpisodeStreamPathsRow{}
	for rows.Next() {
		var i ListAllEpisodeStreamPathsRow
		if err := rows.Scan(&i.ChapterID, &i.LocalPath); err != nil {
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

const listAllMangaPagePaths = `-- name: ListAllMangaPagePaths :many
SELECT chapter_id, page_number, local_path FROM manga_pages
WHERE local_path IS NOT NULL AND local_path != ''
`

func (q *Queries) ListAllMangaPagePaths(ctx context.Context) ([]MangaPage, error) {
	rows, err := q.db.QueryContext(ctx, listAllMangaPagePaths)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []MangaPage{}
	for rows.Next() {
		var i MangaPage
		if err := rows.Scan(&i.ChapterID, &i.PageNumber, &i.LocalPath); err != nil {
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

const listAllNovelContentPaths = `-- name: ListAllNovelContentPaths :many
SELECT chapter_id, local_path FROM novel_chapter_content
WHERE local_path IS NOT NULL AND local_path != ''
`

func (q *Queries) ListAllNovelContentPaths(ctx context.Context) ([]NovelChapterContent, error) {
	rows, err := q.db.QueryContext(ctx, listAllNovelContentPaths)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []NovelChapterContent{}
	for rows.Next() {
		var i NovelChapterContent
		if err := rows.Scan(&i.ChapterID, &i.LocalPath); err != nil {
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

const listChapterExternalIDsByMedia = `-- name: ListChapterExternalIDsByMedia :many
SELECT external_id FROM chapters WHERE media_id = ?
`

func (q *Queries) ListChapterExternalIDsByMedia(ctx context.Context, mediaID int64) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, listChapterExternalIDsByMedia, mediaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []string{}
	for rows.Next() {
		var external_id string
		if err := rows.Scan(&external_id); err != nil {
			return nil, err
		}
		items = append(items, external_id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listChapterUpdates = `-- name: ListChapterUpdates :many

SELECT c.id, c.media_id, c.external_id, c.title, c.number, c.source_order, c.uploaded_at, c.first_seen_at
FROM chapters c
JOIN media m ON m.id = c.media_id
WHERE m.added_at IS NOT NULL
  AND c.first_seen_at IS NOT NULL
  AND (?1 IS NULL OR c.first_seen_at >= ?1)
ORDER BY c.first_seen_at DESC, c.id DESC
LIMIT ?2
`

type ListChapterUpdatesParams struct {
	Column1 interface{} `json:"column_1"`
	Limit   int64       `json:"limit"`
}

type ListChapterUpdatesRow struct {
	ID          int64           `json:"id"`
	MediaID     int64           `json:"media_id"`
	ExternalID  string          `json:"external_id"`
	Title       sql.NullString  `json:"title"`
	Number      sql.NullFloat64 `json:"number"`
	SourceOrder sql.NullInt64   `json:"source_order"`
	UploadedAt  sql.NullInt64   `json:"uploaded_at"`
	FirstSeenAt sql.NullTime    `json:"first_seen_at"`
}

func (q *Queries) ListChapterUpdates(ctx context.Context, arg ListChapterUpdatesParams) ([]ListChapterUpdatesRow, error) {
	rows, err := q.db.QueryContext(ctx, listChapterUpdates, arg.Column1, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListChapterUpdatesRow{}
	for rows.Next() {
		var i ListChapterUpdatesRow
		if err := rows.Scan(
			&i.ID,
			&i.MediaID,
			&i.ExternalID,
			&i.Title,
			&i.Number,
			&i.SourceOrder,
			&i.UploadedAt,
			&i.FirstSeenAt,
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

const listChaptersByMedia = `-- name: ListChaptersByMedia :many
SELECT id, media_id, external_id, title, number, uploaded_at, source_order, first_seen_at, scanlator FROM chapters WHERE media_id = ? ORDER BY source_order, number
`

func (q *Queries) ListChaptersByMedia(ctx context.Context, mediaID int64) ([]Chapter, error) {
	rows, err := q.db.QueryContext(ctx, listChaptersByMedia, mediaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Chapter{}
	for rows.Next() {
		var i Chapter
		if err := rows.Scan(
			&i.ID,
			&i.MediaID,
			&i.ExternalID,
			&i.Title,
			&i.Number,
			&i.UploadedAt,
			&i.SourceOrder,
			&i.FirstSeenAt,
			&i.Scanlator,
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

const listChaptersMissingNumber = `-- name: ListChaptersMissingNumber :many

SELECT id, title FROM chapters WHERE number IS NULL OR number <= 0
`

type ListChaptersMissingNumberRow struct {
	ID    int64          `json:"id"`
	Title sql.NullString `json:"title"`
}

func (q *Queries) ListChaptersMissingNumber(ctx context.Context) ([]ListChaptersMissingNumberRow, error) {
	rows, err := q.db.QueryContext(ctx, listChaptersMissingNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListChaptersMissingNumberRow{}
	for rows.Next() {
		var i ListChaptersMissingNumberRow
		if err := rows.Scan(&i.ID, &i.Title); err != nil {
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

const listDownloadedEpisodeStreamsByChapterIDs = `-- name: ListDownloadedEpisodeStreamsByChapterIDs :many
SELECT chapter_id, local_path
FROM anime_episode_streams
WHERE chapter_id IN (/*SLICE:chapter_ids*/?)
  AND local_path IS NOT NULL AND local_path != ''
`

type ListDownloadedEpisodeStreamsByChapterIDsRow struct {
	ChapterID int64          `json:"chapter_id"`
	LocalPath sql.NullString `json:"local_path"`
}

func (q *Queries) ListDownloadedEpisodeStreamsByChapterIDs(ctx context.Context, chapterIds []int64) ([]ListDownloadedEpisodeStreamsByChapterIDsRow, error) {
	query := listDownloadedEpisodeStreamsByChapterIDs
	var queryParams []interface{}
	if len(chapterIds) > 0 {
		for _, v := range chapterIds {
			queryParams = append(queryParams, v)
		}
		query = strings.Replace(query, "/*SLICE:chapter_ids*/?", strings.Repeat(",?", len(chapterIds))[1:], 1)
	} else {
		query = strings.Replace(query, "/*SLICE:chapter_ids*/?", "NULL", 1)
	}
	rows, err := q.db.QueryContext(ctx, query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListDownloadedEpisodeStreamsByChapterIDsRow{}
	for rows.Next() {
		var i ListDownloadedEpisodeStreamsByChapterIDsRow
		if err := rows.Scan(&i.ChapterID, &i.LocalPath); err != nil {
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

const listDownloadedNovelContentByChapterIDs = `-- name: ListDownloadedNovelContentByChapterIDs :many
SELECT chapter_id, local_path
FROM novel_chapter_content
WHERE chapter_id IN (/*SLICE:chapter_ids*/?)
  AND local_path IS NOT NULL AND local_path != ''
`

func (q *Queries) ListDownloadedNovelContentByChapterIDs(ctx context.Context, chapterIds []int64) ([]NovelChapterContent, error) {
	query := listDownloadedNovelContentByChapterIDs
	var queryParams []interface{}
	if len(chapterIds) > 0 {
		for _, v := range chapterIds {
			queryParams = append(queryParams, v)
		}
		query = strings.Replace(query, "/*SLICE:chapter_ids*/?", strings.Repeat(",?", len(chapterIds))[1:], 1)
	} else {
		query = strings.Replace(query, "/*SLICE:chapter_ids*/?", "NULL", 1)
	}
	rows, err := q.db.QueryContext(ctx, query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []NovelChapterContent{}
	for rows.Next() {
		var i NovelChapterContent
		if err := rows.Scan(&i.ChapterID, &i.LocalPath); err != nil {
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

const listMangaPages = `-- name: ListMangaPages :many
SELECT chapter_id, page_number, local_path FROM manga_pages WHERE chapter_id = ? ORDER BY page_number
`

func (q *Queries) ListMangaPages(ctx context.Context, chapterID int64) ([]MangaPage, error) {
	rows, err := q.db.QueryContext(ctx, listMangaPages, chapterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []MangaPage{}
	for rows.Next() {
		var i MangaPage
		if err := rows.Scan(&i.ChapterID, &i.PageNumber, &i.LocalPath); err != nil {
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

const listRecentChapters = `-- name: ListRecentChapters :many
SELECT
    c.id, c.media_id, c.external_id, c.title, c.number, c.source_order, c.uploaded_at
FROM chapters c
JOIN media m ON m.id = c.media_id
WHERE c.uploaded_at IS NOT NULL
  AND m.added_at IS NOT NULL
  AND (?1 IS NULL OR c.uploaded_at >= ?1)
ORDER BY c.uploaded_at DESC
LIMIT ?2
`

type ListRecentChaptersParams struct {
	Column1 interface{} `json:"column_1"`
	Limit   int64       `json:"limit"`
}

type ListRecentChaptersRow struct {
	ID          int64           `json:"id"`
	MediaID     int64           `json:"media_id"`
	ExternalID  string          `json:"external_id"`
	Title       sql.NullString  `json:"title"`
	Number      sql.NullFloat64 `json:"number"`
	SourceOrder sql.NullInt64   `json:"source_order"`
	UploadedAt  sql.NullInt64   `json:"uploaded_at"`
}

func (q *Queries) ListRecentChapters(ctx context.Context, arg ListRecentChaptersParams) ([]ListRecentChaptersRow, error) {
	rows, err := q.db.QueryContext(ctx, listRecentChapters, arg.Column1, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ListRecentChaptersRow{}
	for rows.Next() {
		var i ListRecentChaptersRow
		if err := rows.Scan(
			&i.ID,
			&i.MediaID,
			&i.ExternalID,
			&i.Title,
			&i.Number,
			&i.SourceOrder,
			&i.UploadedAt,
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

const nextUnreadChapterByMediaIDs = `-- name: NextUnreadChapterByMediaIDs :many

SELECT c.id, c.media_id, c.external_id, c.title, c.number, c.uploaded_at, c.source_order, c.first_seen_at, c.scanlator
FROM chapters c
WHERE c.media_id IN (/*SLICE:media_ids*/?)
  AND NOT EXISTS (
    SELECT 1 FROM reading_progress rp
    WHERE rp.chapter_id = c.id AND rp.completed = TRUE
  )
  AND c.id = (
    SELECT c2.id FROM chapters c2
    WHERE c2.media_id = c.media_id
      AND NOT EXISTS (
        SELECT 1 FROM reading_progress rp2
        WHERE rp2.chapter_id = c2.id AND rp2.completed = TRUE
      )
    ORDER BY c2.source_order, c2.number
    LIMIT 1
  )
`

func (q *Queries) NextUnreadChapterByMediaIDs(ctx context.Context, mediaIds []int64) ([]Chapter, error) {
	query := nextUnreadChapterByMediaIDs
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
	items := []Chapter{}
	for rows.Next() {
		var i Chapter
		if err := rows.Scan(
			&i.ID,
			&i.MediaID,
			&i.ExternalID,
			&i.Title,
			&i.Number,
			&i.UploadedAt,
			&i.SourceOrder,
			&i.FirstSeenAt,
			&i.Scanlator,
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

const setChapterNumber = `-- name: SetChapterNumber :exec
UPDATE chapters SET number = ? WHERE id = ?
`

type SetChapterNumberParams struct {
	Number sql.NullFloat64 `json:"number"`
	ID     int64           `json:"id"`
}

func (q *Queries) SetChapterNumber(ctx context.Context, arg SetChapterNumberParams) error {
	_, err := q.db.ExecContext(ctx, setChapterNumber, arg.Number, arg.ID)
	return err
}

const upsertAnimeEpisodeStream = `-- name: UpsertAnimeEpisodeStream :exec
INSERT INTO anime_episode_streams (chapter_id, stream_url, local_path)
VALUES (?, ?, ?)
ON CONFLICT(chapter_id) DO UPDATE SET
    stream_url = excluded.stream_url,
    local_path = excluded.local_path
`

type UpsertAnimeEpisodeStreamParams struct {
	ChapterID int64          `json:"chapter_id"`
	StreamUrl sql.NullString `json:"stream_url"`
	LocalPath sql.NullString `json:"local_path"`
}

func (q *Queries) UpsertAnimeEpisodeStream(ctx context.Context, arg UpsertAnimeEpisodeStreamParams) error {
	_, err := q.db.ExecContext(ctx, upsertAnimeEpisodeStream, arg.ChapterID, arg.StreamUrl, arg.LocalPath)
	return err
}

const upsertMangaPage = `-- name: UpsertMangaPage :exec
INSERT INTO manga_pages (chapter_id, page_number, local_path)
VALUES (?, ?, ?)
ON CONFLICT(chapter_id, page_number) DO UPDATE SET local_path = excluded.local_path
`

type UpsertMangaPageParams struct {
	ChapterID  int64          `json:"chapter_id"`
	PageNumber int64          `json:"page_number"`
	LocalPath  sql.NullString `json:"local_path"`
}

func (q *Queries) UpsertMangaPage(ctx context.Context, arg UpsertMangaPageParams) error {
	_, err := q.db.ExecContext(ctx, upsertMangaPage, arg.ChapterID, arg.PageNumber, arg.LocalPath)
	return err
}

const upsertNovelChapterContent = `-- name: UpsertNovelChapterContent :exec
INSERT INTO novel_chapter_content (chapter_id, local_path)
VALUES (?, ?)
ON CONFLICT(chapter_id) DO UPDATE SET local_path = excluded.local_path
`

type UpsertNovelChapterContentParams struct {
	ChapterID int64          `json:"chapter_id"`
	LocalPath sql.NullString `json:"local_path"`
}

func (q *Queries) UpsertNovelChapterContent(ctx context.Context, arg UpsertNovelChapterContentParams) error {
	_, err := q.db.ExecContext(ctx, upsertNovelChapterContent, arg.ChapterID, arg.LocalPath)
	return err
}
