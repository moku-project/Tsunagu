package mokusync

import (
	"context"
	"errors"
	"net/http"
)

// Status describes this instance, the instances it's paired with, and what's still on its way.
type Status struct {
	Node          string       `json:"node"`
	Name          string       `json:"name"`
	Peers         []PeerStatus `json:"peers"`
	PendingTitles int          `json:"pendingTitles"`
	// Addresses are where devices on the LAN reach this one; empty when it only dials out.
	Addresses         []string           `json:"addresses"`
	MissingExtensions []MissingExtension `json:"missingExtensions"`
}

func (e *Engine) Status(ctx context.Context) (Status, error) {
	peers, err := e.Peers(ctx)
	if err != nil {
		return Status{}, err
	}
	pending, err := e.PendingTitles(ctx)
	if err != nil {
		return Status{}, err
	}
	missing, err := e.MissingExtensions(ctx)
	if err != nil {
		return Status{}, err
	}
	return Status{Node: e.node, Name: e.Name(), Peers: peers, PendingTitles: pending, Addresses: e.Addresses(), MissingExtensions: missing}, nil
}

// RemovePeer unpairs an instance: its token stops working here, and this one stops dialing it.
func (e *Engine) RemovePeer(ctx context.Context, node string) error {
	res, err := e.db.ExecContext(ctx, `DELETE FROM sync_peers WHERE node = ?`, node)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errors.New("no paired device with that ID")
	}
	return nil
}

// SyncNow syncs with every instance this one dials and fetches missing titles, without waiting for the runner.
func (e *Engine) SyncNow(ctx context.Context, client *http.Client) error {
	peers, nodes, err := e.dialPeers(ctx)
	if err != nil {
		return err
	}
	var first error
	for i, p := range peers {
		p.Client = client
		if err := e.SyncWith(ctx, p, nodes[i]); err != nil && first == nil {
			first = err
		}
	}
	if e.resolver != nil {
		if _, err := e.ResolveMissing(ctx, e.resolver); err != nil && first == nil {
			first = err
		}
	}
	return first
}
