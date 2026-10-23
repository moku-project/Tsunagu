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

-- name: UpdateChapterExternalID :exec
UPDATE chapters SET external_id = ? WHERE id = ?;

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

SELECT mp.chapter_id, COUNT(*) AS page_count
FROM manga_pages mp
WHERE mp.chapter_id IN (sqlc.slice('chapter_ids'))
  AND mp.local_path IS NOT NULL AND mp.local_path != ''
  -- Loose downloads write pages one by one; until the job finishes the chapter is
  -- not downloaded, or the reader would get a cut-off page list.
  AND NOT EXISTS (
    SELECT 1 FROM downloads d
    WHERE d.chapter_id = mp.chapter_id AND d.status IN ('queued', 'downloading')
  )
GROUP BY mp.chapter_id;

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

-- name: ListAnyDownloadedMangaPathsForMediaIDs :many
SELECT media_id, local_path FROM (
  SELECT c.media_id AS media_id, mp.local_path AS local_path,
         ROW_NUMBER() OVER (PARTITION BY c.media_id ORDER BY mp.chapter_id, mp.page_number) AS rn
  FROM manga_pages mp
  JOIN chapters c ON c.id = mp.chapter_id
  WHERE c.media_id IN (sqlc.slice('media_ids')) AND mp.local_path IS NOT NULL AND mp.local_path != ''
) WHERE rn = 1;

-- name: ListAnyDownloadedNovelPathsForMediaIDs :many
SELECT media_id, local_path FROM (
  SELECT c.media_id AS media_id, nc.local_path AS local_path,
         ROW_NUMBER() OVER (PARTITION BY c.media_id ORDER BY nc.chapter_id) AS rn
  FROM novel_chapter_content nc
  JOIN chapters c ON c.id = nc.chapter_id
  WHERE c.media_id IN (sqlc.slice('media_ids')) AND nc.local_path IS NOT NULL AND nc.local_path != ''
) WHERE rn = 1;

-- name: ListAnyDownloadedAnimePathsForMediaIDs :many
SELECT media_id, local_path FROM (
  SELECT c.media_id AS media_id, a.local_path AS local_path,
         ROW_NUMBER() OVER (PARTITION BY c.media_id ORDER BY a.chapter_id) AS rn
  FROM anime_episode_streams a
  JOIN chapters c ON c.id = a.chapter_id
  WHERE c.media_id IN (sqlc.slice('media_ids')) AND a.local_path IS NOT NULL AND a.local_path != ''
) WHERE rn = 1;

-- name: DeleteMangaPage :exec
DELETE FROM manga_pages WHERE chapter_id = ? AND page_number = ?;

-- name: SetMangaPagePath :exec
UPDATE manga_pages SET local_path = ? WHERE chapter_id = ? AND page_number = ?;

-- name: SetNovelChapterContentPath :exec
UPDATE novel_chapter_content SET local_path = ? WHERE chapter_id = ?;

-- name: SetAnimeEpisodeStreamPath :exec
UPDATE anime_episode_streams SET local_path = ? WHERE chapter_id = ?;

-- name: GetChapterDownloadContext :one
SELECT
    c.id AS chapter_id,
    c.external_id AS source_chapter_id,
    c.title AS chapter_title,
    c.number AS chapter_number,
    m.external_id AS source_entry_id,
    m.title AS library_title,
    m.content_type AS content_type,
    e.package_name AS extension_package_name,
    -- Another chapter would get the same folder (same number and title, e.g. a second
    -- scanlation group), so this one needs a distinguishing suffix.
    CAST(EXISTS (
        SELECT 1 FROM chapters o
        WHERE o.media_id = c.media_id AND o.id != c.id AND o.number IS c.number AND o.title IS c.title
    ) AS BOOLEAN) AS name_clash
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

-- name: GetChapterContentType :one
-- Unlike GetChapterDownloadContext this works after the series' extension was
-- removed, so orphaned downloads can still be deleted.
SELECT m.content_type FROM chapters c JOIN media m ON m.id = c.media_id WHERE c.id = ?;

-- name: ClearChapterFirstSeen :exec
-- A series' first sync is its backlog, not new releases; keep it off the Updates feed.
UPDATE chapters SET first_seen_at = NULL WHERE media_id = ?;

-- name: ChapterHasDownload :one
SELECT EXISTS (SELECT 1 FROM downloads WHERE chapter_id = ?) AS has_download;

-- name: DeleteChapter :exec
DELETE FROM chapters WHERE id = ?;

-- name: MoveChapterProgress :exec
UPDATE OR IGNORE reading_progress SET chapter_id = sqlc.arg(to_id) WHERE chapter_id = sqlc.arg(from_id);

-- name: MoveChapterDownloads :exec
UPDATE downloads SET chapter_id = sqlc.arg(to_id) WHERE chapter_id = sqlc.arg(from_id);

-- name: MoveChapterPages :exec
UPDATE OR IGNORE manga_pages SET chapter_id = sqlc.arg(to_id) WHERE chapter_id = sqlc.arg(from_id);

-- name: MoveChapterNovelContent :exec
UPDATE OR IGNORE novel_chapter_content SET chapter_id = sqlc.arg(to_id) WHERE chapter_id = sqlc.arg(from_id);

-- name: MoveChapterEpisodeStream :exec
UPDATE OR IGNORE anime_episode_streams SET chapter_id = sqlc.arg(to_id) WHERE chapter_id = sqlc.arg(from_id);

-- name: ListChaptersByIDs :many
SELECT * FROM chapters WHERE id IN (sqlc.slice('ids'));

-- name: ListFirstMangaPagePathsByChapterIDs :many
SELECT mp.chapter_id, mp.local_path FROM manga_pages mp
WHERE mp.chapter_id IN (sqlc.slice('chapter_ids'))
  AND mp.page_number = (SELECT MIN(p2.page_number) FROM manga_pages p2 WHERE p2.chapter_id = mp.chapter_id);
