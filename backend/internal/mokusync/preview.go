package mokusync

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// checkCode tests a pairing code, spending it when consume is set. Wrong codes count toward voiding all open ones.
func (e *Engine) checkCode(ctx context.Context, code string, consume bool) error {
	hash, now := hashSecret(strings.ToUpper(strings.TrimSpace(code))), time.Now().UnixMilli()
	var n int64
	if consume {
		res, err := e.db.ExecContext(ctx, `DELETE FROM sync_pairing_codes WHERE code_hash = ? AND expires_at >= ?`, hash, now)
		if err != nil {
			return err
		}
		n, _ = res.RowsAffected()
	} else if err := e.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sync_pairing_codes WHERE code_hash = ? AND expires_at >= ?`,
		hash, now).Scan(&n); err != nil {
		return err
	}
	if n != 1 {
		if e.pairFailures.Add(1) >= maxPairFailures {
			e.pairFailures.Store(0)
			if _, err := e.db.ExecContext(ctx, `DELETE FROM sync_pairing_codes`); err != nil {
				return err
			}
		}
		return errors.New("pairing code is wrong or expired")
	}
	e.pairFailures.Store(0)
	return nil
}

type previewRequest struct {
	Code string `json:"code"`
}

// previewResponse is what a code holder may see before pairing: the same library pairing would hand over.
type previewResponse struct {
	Node         string   `json:"node"`
	Name         string   `json:"name"`
	Library      []string `json:"library"`
	Folders      []string `json:"folders"`
	ReadChapters int      `json:"readChapters"`
}

func (h *Handler) preview(w http.ResponseWriter, r *http.Request) {
	var req previewRequest
	if err := readJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.E.checkCode(r.Context(), req.Code, false); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	resp, err := h.E.summary(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, resp)
}

func (e *Engine) summary(ctx context.Context) (previewResponse, error) {
	if _, err := e.Capture(ctx); err != nil {
		return previewResponse{}, err
	}
	library, err := elements(ctx, e.db, KindLibrary, "")
	if err != nil {
		return previewResponse{}, err
	}
	folders, err := customFolderNames(ctx, e.db)
	if err != nil {
		return previewResponse{}, err
	}
	resp := previewResponse{Node: e.node, Name: e.Name(), Library: library, Folders: folders}
	if resp.Library == nil {
		resp.Library = []string{}
	}
	err = e.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sync_registers WHERE kind = ? AND field = 'completed' AND value = 'true'`,
		KindChapter).Scan(&resp.ReadChapters)
	return resp, err
}

func customFolderNames(ctx context.Context, q querier) ([]string, error) {
	rows, err := q.QueryContext(ctx, `SELECT name FROM folders WHERE kind = 'custom' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// PairPreview says what pairing with a device would merge, so the user can back out before anything mixes.
type PairPreview struct {
	Node string `json:"node"`
	Name string `json:"name"`
	// Titles in their library, and how many of those this device doesn't have yet.
	Titles    int `json:"titles"`
	NewTitles int `json:"newTitles"`
	// SentTitles are this device's titles they don't have yet.
	SentTitles   int `json:"sentTitles"`
	Folders      int `json:"folders"`
	ReadChapters int `json:"readChapters"`
	// MergingFolders are their folders that will merge with a same-name folder here.
	MergingFolders []string `json:"mergingFolders"`
}

// Preview asks the instance at address what pairing would bring in, without spending the code.
func (e *Engine) Preview(ctx context.Context, client *http.Client, address, code string) (PairPreview, error) {
	if !e.Named() {
		return PairPreview{}, ErrUnnamed
	}
	address, err := peerAddress(address)
	if err != nil {
		return PairPreview{}, err
	}
	var theirs previewResponse
	if err := postJSON(ctx, client, address+previewPath, "", previewRequest{Code: code}, &theirs); err != nil {
		return PairPreview{}, fmt.Errorf("preview %s: %w", address, err)
	}
	if theirs.Node == e.node {
		return PairPreview{}, errors.New("that address is this device")
	}
	mine, err := e.summary(ctx)
	if err != nil {
		return PairPreview{}, err
	}
	p := PairPreview{Node: theirs.Node, Name: theirs.Name, Titles: len(theirs.Library), Folders: len(theirs.Folders),
		ReadChapters: theirs.ReadChapters, MergingFolders: []string{}}
	have := map[string]bool{}
	for _, k := range mine.Library {
		have[k] = true
	}
	given := map[string]bool{}
	for _, k := range theirs.Library {
		given[k] = true
		if !have[k] {
			p.NewTitles++
		}
	}
	for _, k := range mine.Library {
		if !given[k] {
			p.SentTitles++
		}
	}
	local := map[string]bool{}
	for _, f := range mine.Folders {
		local[strings.ToLower(strings.TrimSpace(f))] = true
	}
	for _, f := range theirs.Folders {
		if local[strings.ToLower(strings.TrimSpace(f))] {
			p.MergingFolders = append(p.MergingFolders, f)
		}
	}
	sort.Strings(p.MergingFolders)
	return p, nil
}
