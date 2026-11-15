package graph

import (
	"context"
	"time"

	"tsunagu/backend/internal/api/graph/model"
	"tsunagu/backend/internal/mokusync"
)

func (r *Resolver) syncStatus(ctx context.Context) (*model.SyncStatus, error) {
	st, err := r.Ms.Status(ctx)
	if err != nil {
		return nil, err
	}
	out := &model.SyncStatus{Node: st.Node, Name: st.Name, Peers: []*model.SyncPeer{}, PendingTitles: int32(st.PendingTitles)}
	for _, p := range st.Peers {
		out.Peers = append(out.Peers, toSyncPeer(p))
	}
	return out, nil
}

func toSyncPeer(p mokusync.PeerStatus) *model.SyncPeer {
	out := &model.SyncPeer{Node: p.Node, Name: p.Name}
	if p.Address != "" {
		out.Address = &p.Address
	}
	if p.LastSeenAt > 0 {
		t := time.UnixMilli(p.LastSeenAt)
		out.LastSeenAt = &t
	}
	return out
}
