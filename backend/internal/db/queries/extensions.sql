-- name: UpsertExtension :one
-- Another repo's sync doesn't take over a row; a sideloaded row is handed back.
INSERT INTO extensions (
    repository_id, package_name, name, version, content_type, lang,
    icon_url, apk_url, jar_url, is_nsfw
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(package_name) DO UPDATE SET
    repository_id = excluded.repository_id,
    name = excluded.name,
    version = excluded.version,
    content_type = excluded.content_type,
    lang = excluded.lang,
    icon_url = excluded.icon_url,
    apk_url = excluded.apk_url,
    jar_url = excluded.jar_url,
    is_nsfw = excluded.is_nsfw
WHERE extensions.repository_id = excluded.repository_id
   OR extensions.repository_id IN (SELECT id FROM repositories WHERE index_url = 'local:sideload')
RETURNING *;

-- name: UpsertSideloadedExtension :one
-- A sideload replaces whatever version a repo provided.
INSERT INTO extensions (
    repository_id, package_name, name, version, content_type, lang, apk_url
) VALUES (?, ?, ?, ?, ?, ?, '')
ON CONFLICT(package_name) DO UPDATE SET
    repository_id = excluded.repository_id,
    name = excluded.name,
    version = excluded.version,
    content_type = excluded.content_type,
    lang = excluded.lang,
    apk_url = '',
    jar_url = NULL
RETURNING *;

-- name: MoveInstalledExtensions :exec
-- Installed extensions outlive the repo they came from.
UPDATE extensions SET repository_id = sqlc.arg(to_repo) WHERE repository_id = sqlc.arg(from_repo) AND installed;

-- name: GetExtension :one
SELECT * FROM extensions WHERE id = ?;

-- name: GetExtensionByPackageName :one
SELECT * FROM extensions WHERE package_name = ?;

-- name: ListExtensionsByRepository :many
SELECT * FROM extensions WHERE repository_id = ? ORDER BY name;

-- name: ListInstalledExtensions :many
SELECT * FROM extensions WHERE installed = TRUE AND enabled = TRUE ORDER BY name;

-- name: QueryExtensions :many
SELECT * FROM extensions
WHERE (CAST(sqlc.narg('repository_id') AS INTEGER) IS NULL OR repository_id = sqlc.narg('repository_id'))
  AND (CAST(sqlc.narg('content_type') AS TEXT) IS NULL OR content_type = sqlc.narg('content_type'))
  AND (CAST(sqlc.narg('lang') AS TEXT) IS NULL OR lang = sqlc.narg('lang'))
  AND (CAST(sqlc.narg('installed') AS BOOLEAN) IS NULL OR installed = sqlc.narg('installed'))
  AND (CAST(sqlc.narg('search') AS TEXT) IS NULL
       OR name LIKE '%' || sqlc.narg('search') || '%'
       OR package_name LIKE '%' || sqlc.narg('search') || '%')
ORDER BY installed DESC, name
LIMIT sqlc.arg('lim') OFFSET sqlc.arg('off');

-- name: ListExtensionLanguages :many
SELECT DISTINCT lang FROM extensions
WHERE (CAST(sqlc.narg('repository_id') AS INTEGER) IS NULL OR repository_id = sqlc.narg('repository_id'))
  AND (CAST(sqlc.narg('content_type') AS TEXT) IS NULL OR content_type = sqlc.narg('content_type'))
  AND (CAST(sqlc.narg('installed') AS BOOLEAN) IS NULL OR installed = sqlc.narg('installed'))
  AND (CAST(sqlc.narg('search') AS TEXT) IS NULL
       OR name LIKE '%' || sqlc.narg('search') || '%'
       OR package_name LIKE '%' || sqlc.narg('search') || '%')
ORDER BY lang;

-- name: CountExtensions :one
SELECT COUNT(*) FROM extensions
WHERE (CAST(sqlc.narg('repository_id') AS INTEGER) IS NULL OR repository_id = sqlc.narg('repository_id'))
  AND (CAST(sqlc.narg('content_type') AS TEXT) IS NULL OR content_type = sqlc.narg('content_type'))
  AND (CAST(sqlc.narg('lang') AS TEXT) IS NULL OR lang = sqlc.narg('lang'))
  AND (CAST(sqlc.narg('installed') AS BOOLEAN) IS NULL OR installed = sqlc.narg('installed'))
  AND (CAST(sqlc.narg('search') AS TEXT) IS NULL
       OR name LIKE '%' || sqlc.narg('search') || '%'
       OR package_name LIKE '%' || sqlc.narg('search') || '%');

-- name: MarkExtensionInstalled :one
UPDATE extensions
SET installed = TRUE, jar_path = ?, installed_version = version, installed_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: MarkExtensionUpdated :one
UPDATE extensions
SET jar_path = ?, installed_version = version, installed_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING *;

-- name: MarkExtensionUninstalled :one
UPDATE extensions
SET installed = FALSE, jar_path = NULL, installed_version = NULL, installed_at = NULL
WHERE id = ?
RETURNING *;

-- name: SetExtensionEnabled :one
UPDATE extensions SET enabled = ? WHERE id = ?
RETURNING *;

-- name: UpdateExtensionIconLocalPath :exec
UPDATE extensions SET icon_local_path = ? WHERE id = ?;

-- name: ClearExtensionIconPaths :exec
UPDATE extensions SET icon_local_path = NULL;

-- name: UpdateExtensionSupportsLatest :one
UPDATE extensions SET supports_latest = ? WHERE id = ?
RETURNING *;

-- name: UpdateExtensionSourceID :one
UPDATE extensions SET source_id = ? WHERE id = ?
RETURNING *;

-- name: GetExtensionBySourceID :one
SELECT * FROM extensions WHERE source_id = ? AND source_id != 0 LIMIT 1;

-- name: ListInstalledExtensionsWithSourceID :many
SELECT * FROM extensions WHERE installed = TRUE AND source_id != 0 ORDER BY name;

-- name: GetExtensionsByIDs :many

SELECT * FROM extensions WHERE id IN (sqlc.slice('ids'));
