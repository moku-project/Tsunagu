package mokusync

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// errUnauthorized means the peer no longer accepts this instance's token.
var errUnauthorized = errors.New("this device is not paired there")

// defaultClient gives up on an unreachable address quickly, since a moved device is then looked for on the LAN.
var defaultClient = &http.Client{
	Timeout:   60 * time.Second,
	Transport: &http.Transport{DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext, IdleConnTimeout: 90 * time.Second},
}

// Routes under PeerPrefix authenticate with peer tokens instead of the user's session.
const (
	PeerPrefix    = "/api/sync/v1/peer/"
	exchangePath  = PeerPrefix + "exchange"
	pairPath      = PeerPrefix + "pair"
	previewPath   = PeerPrefix + "preview"
	pairingTTL    = 10 * time.Minute
	maxBodyBytes  = 64 << 20
	pairingSymbol = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	// maxPairFailures wrong codes void every open code, so the LAN port can't be used to guess one.
	maxPairFailures = 20
)

// Handler serves the sync API. Peer routes check peer tokens; the rest sit behind Tsunagu's own auth.
type Handler struct {
	E      *Engine
	Client *http.Client
	// PeerOnly serves just the peer routes, for the listener other devices reach.
	PeerOnly bool
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.PeerOnly && !strings.HasPrefix(r.URL.Path, PeerPrefix) {
		http.NotFound(w, r)
		return
	}
	switch {
	case r.URL.Path == exchangePath && r.Method == http.MethodPost:
		h.exchange(w, r)
	case r.URL.Path == pairPath && r.Method == http.MethodPost:
		h.pair(w, r)
	case r.URL.Path == previewPath && r.Method == http.MethodPost:
		h.preview(w, r)
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
	if err := h.E.learnAddress(r.Context(), req.Node, callerHost(r), req.Port); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	if !e.Named() {
		return "", time.Time{}, ErrUnnamed
	}
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

// pairRequest also hands over a token for dialing the caller back, so either device can start a sync.
type pairRequest struct {
	Code   string `json:"code"`
	Node   string `json:"node"`
	Name   string `json:"name"`
	Expect string `json:"expect,omitempty"`
	Token  string `json:"token,omitempty"`
	Port   int    `json:"port,omitempty"`
}

type pairResponse struct {
	Node  string `json:"node"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

// acceptPairing spends a pairing code and returns a fresh token for the new peer. host is where the request
// came from, used with the caller's port to dial it back.
func (e *Engine) acceptPairing(ctx context.Context, req pairRequest, host string) (string, error) {
	if req.Node == "" || req.Node == e.node {
		return "", errors.New("invalid peer node")
	}
	// The other device checked who it previewed; pairing with anyone else would merge a library it never saw.
	if req.Expect != "" && req.Expect != e.node {
		return "", errors.New("this is not the device that was previewed")
	}
	if err := e.checkCode(ctx, req.Code, true); err != nil {
		return "", err
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	address := ""
	if req.Token != "" {
		address = dialBack(host, req.Port)
	}
	_, err := e.db.ExecContext(ctx, `INSERT INTO sync_peers (node, name, token_hash, dial_token, address) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(node) DO UPDATE SET name = excluded.name, token_hash = excluded.token_hash,
		dial_token = excluded.dial_token, address = excluded.address`, req.Node, req.Name, hashSecret(token), req.Token, address)
	return token, err
}

// callerHost is the IP a request came from, or "" when a proxy sits between, since that IP isn't the device's.
func callerHost(r *http.Request) string {
	for _, h := range []string{"Forwarded", "X-Forwarded-For", "X-Real-Ip"} {
		if r.Header.Get(h) != "" {
			return ""
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return ""
	}
	return host
}

// dialBack is the address a caller at host listens on, or "" when it can't be reached that way.
func dialBack(host string, port int) string {
	ip := net.ParseIP(host)
	if ip == nil || port <= 0 || port > 65535 || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
		return ""
	}
	return "http://" + net.JoinHostPort(ip.String(), strconv.Itoa(port))
}

// learnAddress follows a peer that dialed in from a new address, the way DHCP moves phones around.
func (e *Engine) learnAddress(ctx context.Context, node, host string, port int) error {
	address := dialBack(host, port)
	if address == "" {
		return nil
	}
	var old string
	err := e.db.QueryRowContext(ctx, `SELECT address FROM sync_peers WHERE node = ? AND dial_token != ''`, node).Scan(&old)
	if errors.Is(err, sql.ErrNoRows) || old == address || (old != "" && !replaceable(old)) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = e.db.ExecContext(ctx, `UPDATE sync_peers SET address = ? WHERE node = ?`, address, node)
	return err
}

func (h *Handler) pair(w http.ResponseWriter, r *http.Request) {
	var req pairRequest
	if err := readJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	token, err := h.E.acceptPairing(r.Context(), req, callerHost(r))
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
		Node    string `json:"node"`
	}
	if err := readJSON(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	node, err := h.E.ConnectTo(r.Context(), h.Client, req.Address, req.Code, req.Node)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, r, map[string]string{"node": node})
}

// Connect pairs with the instance at address and remembers it, so the runner keeps syncing with it.
func (e *Engine) Connect(ctx context.Context, client *http.Client, address, code string) (string, error) {
	return e.ConnectTo(ctx, client, address, code, "")
}

// ConnectTo pairs only if the instance at address is node, the one a preview showed; "" skips the check.
func (e *Engine) ConnectTo(ctx context.Context, client *http.Client, address, code, node string) (string, error) {
	if !e.Named() {
		return "", ErrUnnamed
	}
	address = strings.TrimRight(strings.TrimSpace(address), "/")
	if address == "" {
		return "", errors.New("missing address")
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	back := hex.EncodeToString(b)
	req := pairRequest{Code: code, Node: e.node, Name: e.Name(), Expect: node, Token: back, Port: int(e.syncPort.Load())}
	var resp pairResponse
	if err := postJSON(ctx, client, address+pairPath, "", req, &resp); err != nil {
		return "", fmt.Errorf("pair with %s: %w", address, err)
	}
	if resp.Node == "" || resp.Token == "" {
		return "", errors.New("peer sent an incomplete pairing reply")
	}
	if node != "" && resp.Node != node {
		return "", errors.New("a different device answered than the one previewed")
	}
	_, err := e.db.ExecContext(ctx, `INSERT INTO sync_peers (node, name, address, dial_token, token_hash) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(node) DO UPDATE SET name = excluded.name, address = excluded.address, dial_token = excluded.dial_token,
		token_hash = excluded.token_hash`, resp.Node, resp.Name, address, resp.Token, hashSecret(back))
	return resp.Node, err
}

// PeerStatus describes one paired instance.
type PeerStatus struct {
	Node       string `json:"node"`
	Name       string `json:"name"`
	Address    string `json:"address,omitempty"`
	LastSeenAt int64  `json:"lastSeenAt,omitempty"`
	// Online means the two are connected now, so changes on either reach the other within seconds.
	Online bool `json:"online"`
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
		_, failing := e.failing.Load(p.Node)
		p.Online = e.watchers.watching(p.Node) || (!failing && nowMillis()-p.LastSeenAt < (watchWait+20*time.Second).Milliseconds())
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
		client = defaultClient
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
	if res.StatusCode == http.StatusUnauthorized && token != "" {
		return errUnauthorized
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
