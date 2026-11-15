package mokusync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
)

// KindTitle carries what a device without the title's extension needs to show it and find that extension.
const KindTitle = "title" // key: media key; field "info"

type titleInfo struct {
	Title     string        `json:"title"`
	Extension extensionInfo `json:"extension"`
}

type extensionInfo struct {
	Name        string `json:"name"`
	Lang        string `json:"lang,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	Repository  string `json:"repository,omitempty"`
}

func (e *Engine) captureTitleInfo(ctx context.Context, tx *sql.Tx, mediaKey string, wall int64) (int, error) {
	var info titleInfo
	err := tx.QueryRowContext(ctx, `SELECT m.title, e.name, e.lang, e.content_type, COALESCE(r.index_url, '')
		FROM sync_media_keys mk JOIN media m ON m.id = mk.media_id JOIN extensions e ON e.id = m.extension_id
		LEFT JOIN repositories r ON r.id = e.repository_id WHERE mk.key = ?`, mediaKey).
		Scan(&info.Title, &info.Extension.Name, &info.Extension.Lang, &info.Extension.ContentType, &info.Extension.Repository)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return e.emitSet(ctx, tx, KindTitle, mediaKey, "info", mustJSON(info), wall)
}

func readTitleInfo(ctx context.Context, q querier, mediaKey string) (titleInfo, bool, error) {
	v, _, ok, err := register(ctx, q, KindTitle, mediaKey, "info")
	if err != nil || !ok {
		return titleInfo{}, false, err
	}
	var info titleInfo
	return info, json.Unmarshal(v, &info) == nil, nil
}

type MissingExtension struct {
	Package     string `json:"package"`
	Name        string `json:"name"`
	Lang        string `json:"lang"`
	ContentType string `json:"contentType"`
	// Repository is where a paired device installed it from; empty when nobody said.
	Repository string `json:"repository"`
	// Available means this device already lists the extension, so installing needs no new repository.
	Available bool     `json:"available"`
	Titles    int      `json:"titles"`
	Examples  []string `json:"examples"`
}

const missingExamples = 3

// MissingExtensions groups library titles that can't be fetched here because their extension isn't installed.
func (e *Engine) MissingExtensions(ctx context.Context) ([]MissingExtension, error) {
	installed, known, err := localExtensions(ctx, e.db)
	if err != nil {
		return nil, err
	}
	keys, err := pendingLibraryTitles(ctx, e.db)
	if err != nil {
		return nil, err
	}
	byPackage := map[string]*MissingExtension{}
	for _, k := range keys {
		p, ok := splitKey(k, 2)
		if !ok || installed[p[0]] {
			continue
		}
		m := byPackage[p[0]]
		if m == nil {
			m = &MissingExtension{Package: p[0], Examples: []string{}}
			if name, ok := known[p[0]]; ok {
				m.Name, m.Available = name, true
			}
			byPackage[p[0]] = m
		}
		m.Titles++
		info, ok, err := readTitleInfo(ctx, e.db, k)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if info.Title != "" {
			m.Examples = append(m.Examples, info.Title)
		}
		if m.Name == "" {
			m.Name = info.Extension.Name
		}
		if m.Lang == "" {
			m.Lang, m.ContentType = info.Extension.Lang, info.Extension.ContentType
		}
		if m.Repository == "" {
			m.Repository = info.Extension.Repository
		}
	}
	out := make([]MissingExtension, 0, len(byPackage))
	for _, m := range byPackage {
		if m.Name == "" {
			m.Name = m.Package
		}
		sort.Strings(m.Examples)
		m.Examples = m.Examples[:min(len(m.Examples), missingExamples)]
		out = append(out, *m)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Titles != out[j].Titles {
			return out[i].Titles > out[j].Titles
		}
		return out[i].Package < out[j].Package
	})
	return out, nil
}

// ExtensionRepository is the repository a paired device installed pkg from, or "" when none said.
func (e *Engine) ExtensionRepository(ctx context.Context, pkg string) (string, error) {
	rows, err := e.db.QueryContext(ctx, `SELECT value FROM sync_registers WHERE kind = ? AND key > ? AND key < ? ORDER BY clock DESC`,
		KindTitle, pkg+KeySep, pkg+KeySep+"\U0010FFFF")
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		var info titleInfo
		if err := rows.Scan(&v); err != nil {
			return "", err
		}
		if json.Unmarshal([]byte(v), &info) == nil && info.Extension.Repository != "" {
			return info.Extension.Repository, nil
		}
	}
	return "", rows.Err()
}

// ExtensionsChanged retries titles that were waiting on an extension, on the runner's next pass.
func (e *Engine) ExtensionsChanged() {
	e.resolve.mu.Lock()
	clear(e.resolve.failed)
	e.resolve.mu.Unlock()
	e.remoteChanged.Store(true)
}

func localExtensions(ctx context.Context, q querier) (installed map[string]bool, known map[string]string, err error) {
	rows, err := q.QueryContext(ctx, `SELECT package_name, name, installed FROM extensions`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	installed, known = map[string]bool{}, map[string]string{}
	for rows.Next() {
		var pkg, name string
		var in bool
		if err := rows.Scan(&pkg, &name, &in); err != nil {
			return nil, nil, err
		}
		known[pkg], installed[pkg] = name, in
	}
	return installed, known, rows.Err()
}

func pendingLibraryTitles(ctx context.Context, q querier) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT DISTINCT p.field FROM sync_pending p JOIN sync_set_adds a
		ON a.kind = 'library' AND a.element = p.field WHERE p.kind = 'library' ORDER BY p.field`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}
