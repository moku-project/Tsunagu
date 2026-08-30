package sqlcgen

import (
	"context"
	"database/sql"
	"strings"
)

const clearExtensionIconPaths = `-- name: ClearExtensionIconPaths :exec
UPDATE extensions SET icon_local_path = NULL
`

func (q *Queries) ClearExtensionIconPaths(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, clearExtensionIconPaths)
	return err
}

const getExtension = `-- name: GetExtension :one
SELECT id, repository_id, package_name, name, version, content_type, lang, icon_url, icon_local_path, apk_url, jar_url, jar_path, installed, enabled, discovered_at, installed_at, installed_version, needs_update, is_nsfw, supports_latest FROM extensions WHERE id = ?
`

func (q *Queries) GetExtension(ctx context.Context, id int64) (Extension, error) {
	row := q.db.QueryRowContext(ctx, getExtension, id)
	var i Extension
	err := row.Scan(
		&i.ID,
		&i.RepositoryID,
		&i.PackageName,
		&i.Name,
		&i.Version,
		&i.ContentType,
		&i.Lang,
		&i.IconUrl,
		&i.IconLocalPath,
		&i.ApkUrl,
		&i.JarUrl,
		&i.JarPath,
		&i.Installed,
		&i.Enabled,
		&i.DiscoveredAt,
		&i.InstalledAt,
		&i.InstalledVersion,
		&i.NeedsUpdate,
		&i.IsNsfw,
		&i.SupportsLatest,
	)
	return i, err
}

const getExtensionByPackageName = `-- name: GetExtensionByPackageName :one
SELECT id, repository_id, package_name, name, version, content_type, lang, icon_url, icon_local_path, apk_url, jar_url, jar_path, installed, enabled, discovered_at, installed_at, installed_version, needs_update, is_nsfw, supports_latest FROM extensions WHERE package_name = ?
`

func (q *Queries) GetExtensionByPackageName(ctx context.Context, packageName string) (Extension, error) {
	row := q.db.QueryRowContext(ctx, getExtensionByPackageName, packageName)
	var i Extension
	err := row.Scan(
		&i.ID,
		&i.RepositoryID,
		&i.PackageName,
		&i.Name,
		&i.Version,
		&i.ContentType,
		&i.Lang,
		&i.IconUrl,
		&i.IconLocalPath,
		&i.ApkUrl,
		&i.JarUrl,
		&i.JarPath,
		&i.Installed,
		&i.Enabled,
		&i.DiscoveredAt,
		&i.InstalledAt,
		&i.InstalledVersion,
		&i.NeedsUpdate,
		&i.IsNsfw,
		&i.SupportsLatest,
	)
	return i, err
}

const getExtensionsByIDs = `-- name: GetExtensionsByIDs :many

SELECT id, repository_id, package_name, name, version, content_type, lang, icon_url, icon_local_path, apk_url, jar_url, jar_path, installed, enabled, discovered_at, installed_at, installed_version, needs_update, is_nsfw, supports_latest FROM extensions WHERE id IN (/*SLICE:ids*/?)
`

func (q *Queries) GetExtensionsByIDs(ctx context.Context, ids []int64) ([]Extension, error) {
	query := getExtensionsByIDs
	var queryParams []interface{}
	if len(ids) > 0 {
		for _, v := range ids {
			queryParams = append(queryParams, v)
		}
		query = strings.Replace(query, "/*SLICE:ids*/?", strings.Repeat(",?", len(ids))[1:], 1)
	} else {
		query = strings.Replace(query, "/*SLICE:ids*/?", "NULL", 1)
	}
	rows, err := q.db.QueryContext(ctx, query, queryParams...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Extension{}
	for rows.Next() {
		var i Extension
		if err := rows.Scan(
			&i.ID,
			&i.RepositoryID,
			&i.PackageName,
			&i.Name,
			&i.Version,
			&i.ContentType,
			&i.Lang,
			&i.IconUrl,
			&i.IconLocalPath,
			&i.ApkUrl,
			&i.JarUrl,
			&i.JarPath,
			&i.Installed,
			&i.Enabled,
			&i.DiscoveredAt,
			&i.InstalledAt,
			&i.InstalledVersion,
			&i.NeedsUpdate,
			&i.IsNsfw,
			&i.SupportsLatest,
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

const listExtensionsByRepository = `-- name: ListExtensionsByRepository :many
SELECT id, repository_id, package_name, name, version, content_type, lang, icon_url, icon_local_path, apk_url, jar_url, jar_path, installed, enabled, discovered_at, installed_at, installed_version, needs_update, is_nsfw, supports_latest FROM extensions WHERE repository_id = ? ORDER BY name
`

func (q *Queries) ListExtensionsByRepository(ctx context.Context, repositoryID int64) ([]Extension, error) {
	rows, err := q.db.QueryContext(ctx, listExtensionsByRepository, repositoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Extension{}
	for rows.Next() {
		var i Extension
		if err := rows.Scan(
			&i.ID,
			&i.RepositoryID,
			&i.PackageName,
			&i.Name,
			&i.Version,
			&i.ContentType,
			&i.Lang,
			&i.IconUrl,
			&i.IconLocalPath,
			&i.ApkUrl,
			&i.JarUrl,
			&i.JarPath,
			&i.Installed,
			&i.Enabled,
			&i.DiscoveredAt,
			&i.InstalledAt,
			&i.InstalledVersion,
			&i.NeedsUpdate,
			&i.IsNsfw,
			&i.SupportsLatest,
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

const listInstalledExtensions = `-- name: ListInstalledExtensions :many
SELECT id, repository_id, package_name, name, version, content_type, lang, icon_url, icon_local_path, apk_url, jar_url, jar_path, installed, enabled, discovered_at, installed_at, installed_version, needs_update, is_nsfw, supports_latest FROM extensions WHERE installed = TRUE AND enabled = TRUE ORDER BY name
`

func (q *Queries) ListInstalledExtensions(ctx context.Context) ([]Extension, error) {
	rows, err := q.db.QueryContext(ctx, listInstalledExtensions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Extension{}
	for rows.Next() {
		var i Extension
		if err := rows.Scan(
			&i.ID,
			&i.RepositoryID,
			&i.PackageName,
			&i.Name,
			&i.Version,
			&i.ContentType,
			&i.Lang,
			&i.IconUrl,
			&i.IconLocalPath,
			&i.ApkUrl,
			&i.JarUrl,
			&i.JarPath,
			&i.Installed,
			&i.Enabled,
			&i.DiscoveredAt,
			&i.InstalledAt,
			&i.InstalledVersion,
			&i.NeedsUpdate,
			&i.IsNsfw,
			&i.SupportsLatest,
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

const markExtensionInstalled = `-- name: MarkExtensionInstalled :one
UPDATE extensions
SET installed = TRUE, jar_path = ?, installed_version = version, installed_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING id, repository_id, package_name, name, version, content_type, lang, icon_url, icon_local_path, apk_url, jar_url, jar_path, installed, enabled, discovered_at, installed_at, installed_version, needs_update, is_nsfw, supports_latest
`

type MarkExtensionInstalledParams struct {
	JarPath sql.NullString `json:"jar_path"`
	ID      int64          `json:"id"`
}

func (q *Queries) MarkExtensionInstalled(ctx context.Context, arg MarkExtensionInstalledParams) (Extension, error) {
	row := q.db.QueryRowContext(ctx, markExtensionInstalled, arg.JarPath, arg.ID)
	var i Extension
	err := row.Scan(
		&i.ID,
		&i.RepositoryID,
		&i.PackageName,
		&i.Name,
		&i.Version,
		&i.ContentType,
		&i.Lang,
		&i.IconUrl,
		&i.IconLocalPath,
		&i.ApkUrl,
		&i.JarUrl,
		&i.JarPath,
		&i.Installed,
		&i.Enabled,
		&i.DiscoveredAt,
		&i.InstalledAt,
		&i.InstalledVersion,
		&i.NeedsUpdate,
		&i.IsNsfw,
		&i.SupportsLatest,
	)
	return i, err
}

const markExtensionUninstalled = `-- name: MarkExtensionUninstalled :one
UPDATE extensions
SET installed = FALSE, jar_path = NULL, installed_version = NULL, installed_at = NULL
WHERE id = ?
RETURNING id, repository_id, package_name, name, version, content_type, lang, icon_url, icon_local_path, apk_url, jar_url, jar_path, installed, enabled, discovered_at, installed_at, installed_version, needs_update, is_nsfw, supports_latest
`

func (q *Queries) MarkExtensionUninstalled(ctx context.Context, id int64) (Extension, error) {
	row := q.db.QueryRowContext(ctx, markExtensionUninstalled, id)
	var i Extension
	err := row.Scan(
		&i.ID,
		&i.RepositoryID,
		&i.PackageName,
		&i.Name,
		&i.Version,
		&i.ContentType,
		&i.Lang,
		&i.IconUrl,
		&i.IconLocalPath,
		&i.ApkUrl,
		&i.JarUrl,
		&i.JarPath,
		&i.Installed,
		&i.Enabled,
		&i.DiscoveredAt,
		&i.InstalledAt,
		&i.InstalledVersion,
		&i.NeedsUpdate,
		&i.IsNsfw,
		&i.SupportsLatest,
	)
	return i, err
}

const markExtensionUpdated = `-- name: MarkExtensionUpdated :one
UPDATE extensions
SET jar_path = ?, installed_version = version, installed_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING id, repository_id, package_name, name, version, content_type, lang, icon_url, icon_local_path, apk_url, jar_url, jar_path, installed, enabled, discovered_at, installed_at, installed_version, needs_update, is_nsfw, supports_latest
`

type MarkExtensionUpdatedParams struct {
	JarPath sql.NullString `json:"jar_path"`
	ID      int64          `json:"id"`
}

func (q *Queries) MarkExtensionUpdated(ctx context.Context, arg MarkExtensionUpdatedParams) (Extension, error) {
	row := q.db.QueryRowContext(ctx, markExtensionUpdated, arg.JarPath, arg.ID)
	var i Extension
	err := row.Scan(
		&i.ID,
		&i.RepositoryID,
		&i.PackageName,
		&i.Name,
		&i.Version,
		&i.ContentType,
		&i.Lang,
		&i.IconUrl,
		&i.IconLocalPath,
		&i.ApkUrl,
		&i.JarUrl,
		&i.JarPath,
		&i.Installed,
		&i.Enabled,
		&i.DiscoveredAt,
		&i.InstalledAt,
		&i.InstalledVersion,
		&i.NeedsUpdate,
		&i.IsNsfw,
		&i.SupportsLatest,
	)
	return i, err
}

const setExtensionEnabled = `-- name: SetExtensionEnabled :one
UPDATE extensions SET enabled = ? WHERE id = ?
RETURNING id, repository_id, package_name, name, version, content_type, lang, icon_url, icon_local_path, apk_url, jar_url, jar_path, installed, enabled, discovered_at, installed_at, installed_version, needs_update, is_nsfw, supports_latest
`

type SetExtensionEnabledParams struct {
	Enabled bool  `json:"enabled"`
	ID      int64 `json:"id"`
}

func (q *Queries) SetExtensionEnabled(ctx context.Context, arg SetExtensionEnabledParams) (Extension, error) {
	row := q.db.QueryRowContext(ctx, setExtensionEnabled, arg.Enabled, arg.ID)
	var i Extension
	err := row.Scan(
		&i.ID,
		&i.RepositoryID,
		&i.PackageName,
		&i.Name,
		&i.Version,
		&i.ContentType,
		&i.Lang,
		&i.IconUrl,
		&i.IconLocalPath,
		&i.ApkUrl,
		&i.JarUrl,
		&i.JarPath,
		&i.Installed,
		&i.Enabled,
		&i.DiscoveredAt,
		&i.InstalledAt,
		&i.InstalledVersion,
		&i.NeedsUpdate,
		&i.IsNsfw,
		&i.SupportsLatest,
	)
	return i, err
}

const updateExtensionIconLocalPath = `-- name: UpdateExtensionIconLocalPath :exec
UPDATE extensions SET icon_local_path = ? WHERE id = ?
`

type UpdateExtensionIconLocalPathParams struct {
	IconLocalPath sql.NullString `json:"icon_local_path"`
	ID            int64          `json:"id"`
}

func (q *Queries) UpdateExtensionIconLocalPath(ctx context.Context, arg UpdateExtensionIconLocalPathParams) error {
	_, err := q.db.ExecContext(ctx, updateExtensionIconLocalPath, arg.IconLocalPath, arg.ID)
	return err
}

const updateExtensionSupportsLatest = `-- name: UpdateExtensionSupportsLatest :one
UPDATE extensions SET supports_latest = ? WHERE id = ?
RETURNING id, repository_id, package_name, name, version, content_type, lang, icon_url, icon_local_path, apk_url, jar_url, jar_path, installed, enabled, discovered_at, installed_at, installed_version, needs_update, is_nsfw, supports_latest
`

type UpdateExtensionSupportsLatestParams struct {
	SupportsLatest bool  `json:"supports_latest"`
	ID             int64 `json:"id"`
}

func (q *Queries) UpdateExtensionSupportsLatest(ctx context.Context, arg UpdateExtensionSupportsLatestParams) (Extension, error) {
	row := q.db.QueryRowContext(ctx, updateExtensionSupportsLatest, arg.SupportsLatest, arg.ID)
	var i Extension
	err := row.Scan(
		&i.ID,
		&i.RepositoryID,
		&i.PackageName,
		&i.Name,
		&i.Version,
		&i.ContentType,
		&i.Lang,
		&i.IconUrl,
		&i.IconLocalPath,
		&i.ApkUrl,
		&i.JarUrl,
		&i.JarPath,
		&i.Installed,
		&i.Enabled,
		&i.DiscoveredAt,
		&i.InstalledAt,
		&i.InstalledVersion,
		&i.NeedsUpdate,
		&i.IsNsfw,
		&i.SupportsLatest,
	)
	return i, err
}

const upsertExtension = `-- name: UpsertExtension :one
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
RETURNING id, repository_id, package_name, name, version, content_type, lang, icon_url, icon_local_path, apk_url, jar_url, jar_path, installed, enabled, discovered_at, installed_at, installed_version, needs_update, is_nsfw, supports_latest
`

type UpsertExtensionParams struct {
	RepositoryID int64          `json:"repository_id"`
	PackageName  string         `json:"package_name"`
	Name         string         `json:"name"`
	Version      string         `json:"version"`
	ContentType  string         `json:"content_type"`
	Lang         string         `json:"lang"`
	IconUrl      sql.NullString `json:"icon_url"`
	ApkUrl       string         `json:"apk_url"`
	JarUrl       sql.NullString `json:"jar_url"`
	IsNsfw       bool           `json:"is_nsfw"`
}

func (q *Queries) UpsertExtension(ctx context.Context, arg UpsertExtensionParams) (Extension, error) {
	row := q.db.QueryRowContext(ctx, upsertExtension,
		arg.RepositoryID,
		arg.PackageName,
		arg.Name,
		arg.Version,
		arg.ContentType,
		arg.Lang,
		arg.IconUrl,
		arg.ApkUrl,
		arg.JarUrl,
		arg.IsNsfw,
	)
	var i Extension
	err := row.Scan(
		&i.ID,
		&i.RepositoryID,
		&i.PackageName,
		&i.Name,
		&i.Version,
		&i.ContentType,
		&i.Lang,
		&i.IconUrl,
		&i.IconLocalPath,
		&i.ApkUrl,
		&i.JarUrl,
		&i.JarPath,
		&i.Installed,
		&i.Enabled,
		&i.DiscoveredAt,
		&i.InstalledAt,
		&i.InstalledVersion,
		&i.NeedsUpdate,
		&i.IsNsfw,
		&i.SupportsLatest,
	)
	return i, err
}
