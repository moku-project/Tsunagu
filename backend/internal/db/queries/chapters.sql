-- name: CreateChapter :one

INSERT INTO chapters (media_id, external_id, title, number, uploaded_at, source_order, scanlator, first_seen_at)
VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(media_id, external_id) DO UPDATE SET
    title = excluded.title,
    number = excluded.number,
    uploaded_at = excluded.uploaded_at,
    source_order = excluded.source_order,
    scanlator = excluded.scanlator
RETURNING *;

-- name: ListChapterUpdates :many

SELECT c.id, c.media_id, c.external_id, c.title, c.number, c.source_order, c.uploaded_at, c.first_seen_at
FROM chapters c
JOIN media m ON m.id = c.media_id
WHERE m.added_at IS NOT NULL
  AND c.first_seen_at IS NOT NULL
  AND (?1 IS NULL OR c.first_seen_at >= ?1)
ORDER BY c.first_seen_at DESC, c.id DESC
LIMIT ?2;

-- name: ListChapterExternalIDsByMedia :many
SELECT external_id FROM chapters WHERE media_id = ?;

-- name: GetChapter :one
SELECT * FROM chapters WHERE id = ?;

-- name: ListChaptersByMedia :many
SELECT * FROM chapters WHERE media_id = ? ORDER BY source_order, number;

-- name: CountChaptersByMediaIDs :many
SELECT media_id, COUNT(*) AS chapter_count
FROM chapters
WHERE media_id IN (sqlc.slice('media_ids'))
GROUP BY media_id;

-- name: GetChapterByMediaAndExternalID :one
SELECT * FROM chapters WHERE media_id = ? AND external_id = ?;

-- name: NextUnreadChapterByMediaIDs :many

SELECT c.*
FROM chapters c
WHERE c.media_id IN (sqlc.slice('media_ids'))
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
  );

-- name: LatestChapterByMediaIDs :many

SELECT c.*
FROM chapters c
WHERE c.media_id IN (sqlc.slice('media_ids'))
  AND c.id = (
    SELECT c2.id FROM chapters c2
    WHERE c2.media_id = c.media_id
    ORDER BY c2.number DESC, c2.source_order DESC
    LIMIT 1
  );

-- name: UpsertMangaPage :exec
INSERT INTO manga_pages (chapter_id, page_number, local_path)
VALUES (?, ?, ?)
ON CONFLICT(chapter_id, page_number) DO UPDATE SET local_path = excluded.local_path;

-- name: ListMangaPages :many
SELECT * FROM manga_pages WHERE chapter_id = ? ORDER BY page_number;

-- name: CountDownloadedPagesByChapterIDs :many

SELECT chapter_id, COUNT(*) AS page_count
FROM manga_pages
WHERE chapter_id IN (sqlc.slice('chapter_ids'))
  AND local_path IS NOT NULL AND local_path != ''
GROUP BY chapter_id;

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

-- name: ListDownloadedEpisodeStreamsByChapterIDs :many
SELECT chapter_id, local_path
FROM anime_episode_streams
WHERE chapter_id IN (sqlc.slice('chapter_ids'))
  AND local_path IS NOT NULL AND local_path != '';

-- name: ListDownloadedNovelContentByChapterIDs :many
SELECT chapter_id, local_path
FROM novel_chapter_content
WHERE chapter_id IN (sqlc.slice('chapter_ids'))
  AND local_path IS NOT NULL AND local_path != '';

-- name: ListAllMangaPagePaths :many
SELECT chapter_id, page_number, local_path FROM manga_pages
WHERE local_path IS NOT NULL AND local_path != '';

-- name: ListAllNovelContentPaths :many
SELECT chapter_id, local_path FROM novel_chapter_content
WHERE local_path IS NOT NULL AND local_path != '';

-- name: ListAllEpisodeStreamPaths :many
SELECT chapter_id, local_path FROM anime_episode_streams
WHERE local_path IS NOT NULL AND local_path != '';

-- name: DeleteMangaPage :exec
DELETE FROM manga_pages WHERE chapter_id = ? AND page_number = ?;

-- name: GetChapterDownloadContext :one
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
WHERE c.id = ?;

-- name: DeleteMangaPages :exec
DELETE FROM manga_pages WHERE chapter_id = ?;

-- name: DeleteNovelChapterContent :exec
DELETE FROM novel_chapter_content WHERE chapter_id = ?;

-- name: DeleteAnimeEpisodeStream :exec
DELETE FROM anime_episode_streams WHERE chapter_id = ?;

-- name: ListRecentChapters :many
SELECT
    c.id, c.media_id, c.external_id, c.title, c.number, c.source_order, c.uploaded_at
FROM chapters c
JOIN media m ON m.id = c.media_id
WHERE c.uploaded_at IS NOT NULL
  AND m.added_at IS NOT NULL
  AND (?1 IS NULL OR c.uploaded_at >= ?1)
ORDER BY c.uploaded_at DESC
LIMIT ?2;

-- name: CountUnreadChaptersByMediaIDs :many

SELECT c.media_id AS media_id, COUNT(*) AS unread_count
FROM chapters c
LEFT JOIN reading_progress rp
    ON rp.chapter_id = c.id AND rp.media_id = c.media_id
WHERE c.media_id IN (sqlc.slice('media_ids'))
  AND (rp.completed IS NULL OR rp.completed = FALSE)
GROUP BY c.media_id;

-- name: ListChaptersMissingNumber :many

SELECT id, title FROM chapters WHERE number IS NULL OR number <= 0;

-- name: SetChapterNumber :exec
UPDATE chapters SET number = ? WHERE id = ?;
