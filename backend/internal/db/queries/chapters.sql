-- name: CreateChapter :one
INSERT INTO chapters (library_entry_id, external_id, title, number, uploaded_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(library_entry_id, external_id) DO UPDATE SET
    title = excluded.title,
    number = excluded.number,
    uploaded_at = excluded.uploaded_at
RETURNING *;

-- name: GetChapter :one
SELECT * FROM chapters WHERE id = ?;

-- name: ListChaptersByLibraryEntry :many
SELECT * FROM chapters WHERE library_entry_id = ? ORDER BY number;

-- name: UpsertMangaPage :exec
INSERT INTO manga_pages (chapter_id, page_number, local_path)
VALUES (?, ?, ?)
ON CONFLICT(chapter_id, page_number) DO UPDATE SET local_path = excluded.local_path;

-- name: ListMangaPages :many
SELECT * FROM manga_pages WHERE chapter_id = ? ORDER BY page_number;

-- name: UpsertNovelChapterContent :exec
INSERT INTO novel_chapter_content (chapter_id, local_path)
VALUES (?, ?)
ON CONFLICT(chapter_id) DO UPDATE SET local_path = excluded.local_path;

-- name: GetNovelChapterContent :one
SELECT * FROM novel_chapter_content WHERE chapter_id = ?;

-- name: UpsertAnimeEpisodeStream :exec
INSERT INTO anime_episode_streams (chapter_id, stream_url, local_path)
VALUES (?, ?, ?)
ON CONFLICT(chapter_id) DO UPDATE SET
    stream_url = excluded.stream_url,
    local_path = excluded.local_path;

-- name: GetAnimeEpisodeStream :one
SELECT * FROM anime_episode_streams WHERE chapter_id = ?;

-- name: GetChapterDownloadContext :one
SELECT
    c.id AS chapter_id,
    c.external_id AS source_chapter_id,
    le.external_id AS source_entry_id,
    le.content_type AS content_type,
    e.package_name AS extension_package_name
FROM chapters c
JOIN library_entries le ON le.id = c.library_entry_id
JOIN extensions e ON e.id = le.extension_id
WHERE c.id = ?;

-- name: GetChapterByLibraryEntryAndExternalID :one
SELECT * FROM chapters WHERE library_entry_id = ? AND external_id = ?;
