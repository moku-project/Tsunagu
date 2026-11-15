package mokusync

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Routes under PeerPrefix authenticate with peer tokens instead of the user's session.
const (
	PeerPrefix    = "/api/sync/v1/peer/"
	exchangePath  = PeerPrefix + "exchange"
	pairPath      = PeerPrefix + "pair"
	pairingTTL    = 10 * time.Minute
	maxBodyBytes  = 64 << 20
	pairingSymbol = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

// Handler serves the sync API. Peer routes check peer tokens; the rest sit behind Tsunagu's own auth.
type Handler struct {
	E      *Engine
	Client *http.Client
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == exchangePath && r.Method == http.MethodPost:
		h.exchange(w, r)
	case r.URL.Path == pairPath && r.Method == http.MethodPost:
		h.pair(w, r)
	case r.URL.Path == "/api/sync/v1/pairing-code" && r.Method == http.MethodPost:
		h.pairingCode(w, r)
	case r.URL.Path == "/api/sync/v1/connect" && r.Method == http.MethodPost:
		h.connect(w, r)
	case r.URL.Path == "/api/sync/v1/status" && r.Method == http.MethodGet:
		h.status(w, r)
	default:
		http.NotFound(w, r)
	}
}

func hashSecret(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func (h *Handler) exchange(w http.ResponseWriter, r *http.Request) {
	var req ExchangeRequest
	if err := readJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !h.authorized(r, req.Node) {
		http.Error(w, "unknown peer or token", http.StatusUnauthorized)
		return
	}
	resp, err := h.E.Serve(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, resp)
}

func (h *Handler) authorized(r *http.Request, node string) bool {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" || node == "" {
		return false
	}
	var want string
	err := h.E.db.QueryRowContext(r.Context(), `SELECT token_hash FROM sync_peers WHERE node = ?`, node).Scan(&want)
	return err == nil && want != "" && subtle.ConstantTimeCompare([]byte(want), []byte(hashSecret(token))) == 1
}

// PairingCode issues a one-time code another instance trades for a peer token.
func (e *Engine) PairingCode(ctx context.Context) (string, time.Time, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", time.Time{}, err
	}
	code := make([]byte, len(b))
	for i, v := range b {
		code[i] = pairingSymbol[int(v)%len(pairingSymbol)]
	}
	expires := time.Now().Add(pairingTTL)
	if _, err := e.db.ExecContext(ctx, `DELETE FROM sync_pairing_codes WHERE expires_at < ?`, time.Now().UnixMilli()); err != nil {
		return "", time.Time{}, err
	}
	if _, err := e.db.ExecContext(ctx, `INSERT INTO sync_pairing_codes (code_hash, expires_at) VALUES (?, ?)`,
		hashSecret(string(code)), expires.UnixMilli()); err != nil {
		return "", time.Time{}, err
	}
	return string(code), expires, nil
}

type pairRequest struct {
	Code string `json:"code"`
	Node string `json:"node"`
	Name string `json:"name"`
}

type pairResponse struct {
	Node  string `json:"node"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

// acceptPairing spends a pairing code and returns a fresh token for the new peer.
func (e *Engine) acceptPairing(ctx context.Context, req pairRequest) (string, error) {
	if req.Node == "" || req.Node == e.node {
		return "", errors.New("invalid peer node")
	}
	res, err := e.db.ExecContext(ctx, `DELETE FROM sync_pairing_codes WHERE code_hash = ? AND expires_at >= ?`,
		hashSecret(strings.ToUpper(strings.TrimSpace(req.Code))), time.Now().UnixMilli())
	if err != nil {
		return "", err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return "", errors.New("pairing code is wrong or expired")
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	_, err = e.db.ExecContext(ctx, `INSERT INTO sync_peers (node, name, token_hash) VALUES (?, ?, ?)
		ON CONFLICT(node) DO UPDATE SET name = excluded.name, token_hash = excluded.token_hash`, req.Node, req.Name, hashSecret(token))
	return token, err
}

func (h *Handler) pair(w http.ResponseWriter, r *http.Request) {
	var req pairRequest
	if err := readJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	token, err := h.E.acceptPairing(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	writeJSON(w, r, pairResponse{Node: h.E.node, Name: h.E.Name(), Token: token})
}

func (h *Handler) pairingCode(w http.ResponseWriter, r *http.Request) {
	code, expires, err := h.E.PairingCode(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, map[string]any{"code": code, "expiresAt": expires.UnixMilli(), "node": h.E.node, "name": h.E.Name()})
}

func (h *Handler) connect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Address string `json:"address"`
		Code    string `json:"code"`
	}
	if err := readJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	node, err := h.E.Connect(r.Context(), h.Client, req.Address, req.Code)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, r, map[string]string{"node": node})
}

// Connect pairs with the instance at address and remembers it, so the runner keeps syncing with it.
func (e *Engine) Connect(ctx context.Context, client *http.Client, address, code string) (string, error) {
	address = strings.TrimRight(strings.TrimSpace(address), "/")
	if address == "" {
		return "", errors.New("missing address")
	}
	var resp pairResponse
	if err := postJSON(ctx, client, address+pairPath, "", pairRequest{Code: code, Node: e.node, Name: e.Name()}, &resp); err != nil {
		return "", fmt.Errorf("pair with %s: %w", address, err)
	}
	if resp.Node == "" || resp.Token == "" {
		return "", errors.New("peer sent an incomplete pairing reply")
	}
	_, err := e.db.ExecContext(ctx, `INSERT INTO sync_peers (node, name, address, dial_token) VALUES (?, ?, ?, ?)
		ON CONFLICT(node) DO UPDATE SET name = excluded.name, address = excluded.address, dial_token = excluded.dial_token`,
		resp.Node, resp.Name, address, resp.Token)
	return resp.Node, err
}

// PeerStatus describes one paired instance.
type PeerStatus struct {
	Node       string `json:"node"`
	Name       string `json:"name"`
	Address    string `json:"address,omitempty"`
	LastSeenAt int64  `json:"lastSeenAt,omitempty"`
}

func (e *Engine) Peers(ctx context.Context) ([]PeerStatus, error) {
	rows, err := e.db.QueryContext(ctx, `SELECT node, name, address, COALESCE(last_seen_at, 0) FROM sync_peers ORDER BY name, node`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []PeerStatus{}
	for rows.Next() {
		var p PeerStatus
		if err := rows.Scan(&p.Node, &p.Name, &p.Address, &p.LastSeenAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	st, err := h.E.Status(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, r, st)
}

// HTTPPeer reaches another Tsunagu over its sync API.
type HTTPPeer struct {
	Address string
	Token   string
	Client  *http.Client
}

func (p HTTPPeer) Exchange(ctx context.Context, req ExchangeRequest) (ExchangeResponse, error) {
	var resp ExchangeResponse
	err := postJSON(ctx, p.Client, strings.TrimRight(p.Address, "/")+exchangePath, p.Token, req, &resp)
	return resp, err
}

func postJSON(ctx context.Context, client *http.Client, url, token string, body, out any) error {
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if err := json.NewEncoder(zw).Encode(body); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	var rd io.Reader = res.Body
	if res.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(res.Body)
		if err != nil {
			return err
		}
		defer gz.Close()
		rd = gz
	}
	if res.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(rd, 4096))
		return fmt.Errorf("%s: %s", res.Status, strings.TrimSpace(string(msg)))
	}
	return json.NewDecoder(io.LimitReader(rd, maxBodyBytes)).Decode(out)
}

func readJSON(r *http.Request, out any) error {
	var rd io.Reader = http.MaxBytesReader(nil, r.Body, maxBodyBytes)
	if r.Header.Get("Content-Encoding") == "gzip" {
		gz, err := gzip.NewReader(rd)
		if err != nil {
			return err
		}
		defer gz.Close()
		rd = io.LimitReader(gz, maxBodyBytes)
	}
	return json.NewDecoder(rd).Decode(out)
}

func writeJSON(w http.ResponseWriter, r *http.Request, v any) {
	w.Header().Set("Content-Type", "application/json")
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		zw := gzip.NewWriter(w)
		defer zw.Close()
		_ = json.NewEncoder(zw).Encode(v)
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

// dialPeers lists paired instances this one connects to.
func (e *Engine) dialPeers(ctx context.Context) ([]HTTPPeer, []string, error) {
	rows, err := e.db.QueryContext(ctx, `SELECT node, address, dial_token FROM sync_peers WHERE address != '' AND dial_token != ''`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var peers []HTTPPeer
	var nodes []string
	for rows.Next() {
		var node string
		var p HTTPPeer
		if err := rows.Scan(&node, &p.Address, &p.Token); err != nil {
			return nil, nil, err
		}
		peers = append(peers, p)
		nodes = append(nodes, node)
	}
	return peers, nodes, rows.Err()
}
