package mokusync

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// batchLimit caps the ops in one exchange; a bigger backlog continues in further rounds.
var batchLimit = 5000

// ExchangeRequest carries the caller's version vector and the ops it believes the receiver lacks.
// Base is the receiver's version vector those ops were chosen against.
type ExchangeRequest struct {
	Node     string        `json:"node"`
	Name     string        `json:"name"`
	VV       VersionVector `json:"vv"`
	Base     VersionVector `json:"base"`
	Ops      []Op          `json:"ops"`
	Complete bool          `json:"complete"`
}

// ExchangeResponse carries every op the caller lacks (up to batchLimit) and the receiver's version vector.
type ExchangeResponse struct {
	Node string        `json:"node"`
	Name string        `json:"name"`
	VV   VersionVector `json:"vv"`
	Ops  []Op          `json:"ops"`
	More bool          `json:"more"`
}

// Peer is another Tsunagu this instance can exchange ops with.
type Peer interface {
	Exchange(ctx context.Context, req ExchangeRequest) (ExchangeResponse, error)
}

// Serve answers an exchange: it merges the caller's ops, then returns what the caller lacks.
func (e *Engine) Serve(ctx context.Context, req ExchangeRequest) (ExchangeResponse, error) {
	resp := ExchangeResponse{Node: e.node, Name: e.name}
	err := e.write(ctx, func(tx *sql.Tx) error {
		mine, err := versionVector(ctx, tx)
		if err != nil {
			return err
		}
		// The caller's ops are a complete delta only if nothing it assumed we had has gone missing.
		complete := req.Complete && !mine.Lacks(req.Base)
		if err := e.apply(ctx, tx, req.Ops); err != nil {
			return err
		}
		if complete {
			if err := mergeVV(ctx, tx, req.VV); err != nil {
				return err
			}
		}
		if resp.Ops, resp.More, err = opsSince(ctx, tx, req.VV, batchLimit); err != nil {
			return err
		}
		if resp.VV, err = versionVector(ctx, tx); err != nil {
			return err
		}
		// After a full reply the caller holds everything we do; after a partial one, only what it said.
		known := resp.VV
		if resp.More {
			known = req.VV
		}
		return savePeer(ctx, tx, req.Node, req.Name, known)
	})
	return resp, err
}

// SyncWith exchanges ops with peer until neither side lacks anything the other has.
func (e *Engine) SyncWith(ctx context.Context, peer Peer, peerNode string) error {
	for round := 0; round < 64; round++ {
		if _, err := e.Capture(ctx); err != nil {
			return err
		}
		known, err := e.peerVV(ctx, peerNode)
		if err != nil {
			return err
		}
		mine, err := e.VersionVector(ctx)
		if err != nil {
			return err
		}
		ops, more, err := opsSince(ctx, e.db, known, batchLimit)
		if err != nil {
			return err
		}
		resp, err := peer.Exchange(ctx, ExchangeRequest{Node: e.node, Name: e.name, VV: mine, Base: known, Ops: ops, Complete: !more})
		if err != nil {
			return err
		}
		if peerNode == "" {
			peerNode = resp.Node
		}
		if resp.Node != peerNode {
			return fmt.Errorf("peer answered as %s, expected %s", resp.Node, peerNode)
		}
		err = e.write(ctx, func(tx *sql.Tx) error {
			if err := e.apply(ctx, tx, resp.Ops); err != nil {
				return err
			}
			// The response held everything we lacked, so its version vector now describes us too.
			if !resp.More {
				if err := mergeVV(ctx, tx, resp.VV); err != nil {
					return err
				}
			}
			return savePeer(ctx, tx, resp.Node, resp.Name, resp.VV)
		})
		if err != nil {
			return err
		}
		if !more && !resp.More && !resp.VV.Lacks(mine) {
			return nil
		}
	}
	return errors.New("sync did not settle after 64 rounds")
}

func (e *Engine) peerVV(ctx context.Context, node string) (VersionVector, error) {
	vv := VersionVector{}
	if node == "" {
		return vv, nil
	}
	var raw string
	err := e.db.QueryRowContext(ctx, `SELECT vv FROM sync_peers WHERE node = ?`, node).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return vv, nil
	}
	if err != nil {
		return nil, err
	}
	return vv, json.Unmarshal([]byte(raw), &vv)
}

func savePeer(ctx context.Context, tx *sql.Tx, node, name string, vv VersionVector) error {
	if node == "" {
		return nil
	}
	b, err := json.Marshal(vv)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO sync_peers (node, name, vv, last_seen_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(node) DO UPDATE SET name = excluded.name, vv = excluded.vv, last_seen_at = excluded.last_seen_at`,
		node, name, string(b), nowMillis())
	return err
}

// LocalPeer connects two engines in one process; tests and same-host setups use it.
type LocalPeer struct{ E *Engine }

func (p LocalPeer) Exchange(ctx context.Context, req ExchangeRequest) (ExchangeResponse, error) {
	// A JSON round trip keeps in-process syncs honest about what crosses the wire.
	b, err := json.Marshal(req)
	if err != nil {
		return ExchangeResponse{}, err
	}
	var in ExchangeRequest
	if err := json.Unmarshal(b, &in); err != nil {
		return ExchangeResponse{}, err
	}
	resp, err := p.E.Serve(ctx, in)
	if err != nil {
		return resp, err
	}
	if b, err = json.Marshal(resp); err != nil {
		return resp, err
	}
	var out ExchangeResponse
	return out, json.Unmarshal(b, &out)
}
