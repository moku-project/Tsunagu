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
