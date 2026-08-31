-- name: UpsertExtension :one
INSERT INTO extensions (
    repository_id, package_name, name, version, content_type, lang,
    icon_url, apk_url, jar_url, is_nsfw
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(package_name) DO UPDATE SET
    name = excluded.name,
    version = excluded.version,
    content_type = excluded.content_type,
    lang = excluded.lang,
    icon_url = excluded.icon_url,
    apk_url = excluded.apk_url,
    jar_url = excluded.jar_url,
    is_nsfw = excluded.is_nsfw
RETURNING *;

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

-- name: GetExtensionsByIDs :many

SELECT * FROM extensions WHERE id IN (sqlc.slice('ids'));
