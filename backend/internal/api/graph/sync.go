package graph

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"tsunagu/backend/internal/api/graph/model"
	"tsunagu/backend/internal/mokusync"
)

func (r *Resolver) syncStatus(ctx context.Context) (*model.SyncStatus, error) {
	st, err := r.Ms.Status(ctx)
	if err != nil {
		return nil, err
	}
	out := &model.SyncStatus{Node: st.Node, Peers: []*model.SyncPeer{}, PendingTitles: int32(st.PendingTitles), Addresses: st.Addresses, MissingExtensions: toMissingExtensions(st.MissingExtensions)}
	if st.Name != "" {
		out.Name = &st.Name
	}
	for _, p := range st.Peers {
		out.Peers = append(out.Peers, toSyncPeer(p))
	}
	return out, nil
}

func toSyncPeer(p mokusync.PeerStatus) *model.SyncPeer {
	out := &model.SyncPeer{Node: p.Node, Name: p.Name, Online: p.Online}
	if p.Address != "" {
		out.Address = &p.Address
	}
	if p.LastSeenAt > 0 {
		t := time.UnixMilli(p.LastSeenAt)
		out.LastSeenAt = &t
	}
	return out
}

func (r *Resolver) syncNearby(ctx context.Context) ([]*model.SyncNearbyDevice, error) {
	found, err := r.Ms.Nearby(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*model.SyncNearbyDevice, 0, len(found))
	for _, d := range found {
		out = append(out, &model.SyncNearbyDevice{Node: d.Node, Name: d.Name, Address: d.Address, Paired: d.Paired})
	}
	return out, nil
}

func (r *Resolver) previewSyncPeer(ctx context.Context, address, code string) (*model.SyncPairPreview, error) {
	p, err := r.Ms.Preview(ctx, nil, address, code)
	if err != nil {
		return nil, err
	}
	return &model.SyncPairPreview{
		Node: p.Node, Name: p.Name, Titles: int32(p.Titles), NewTitles: int32(p.NewTitles), SentTitles: int32(p.SentTitles),
		Folders: int32(p.Folders), ReadChapters: int32(p.ReadChapters), MergingFolders: p.MergingFolders,
	}, nil
}

func toMissingExtensions(in []mokusync.MissingExtension) []*model.SyncMissingExtension {
	out := make([]*model.SyncMissingExtension, 0, len(in))
	for _, m := range in {
		out = append(out, &model.SyncMissingExtension{
			PackageName: m.Package, Name: m.Name, Lang: m.Lang, ContentType: m.ContentType, Repository: m.Repository,
			Available: m.Available, Titles: int32(m.Titles), Examples: m.Examples,
		})
	}
	return out
}

func (r *Resolver) installSyncExtension(ctx context.Context, pkg string) (*model.Extension, error) {
	var listed bool
	if err := r.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM extensions WHERE package_name = ?)`, pkg).Scan(&listed); err != nil {
		return nil, err
	}
	if !listed {
		repo, err := r.Ms.ExtensionRepository(ctx, pkg)
		if err != nil {
			return nil, err
		}
		if repo == "" {
			return nil, fmt.Errorf("no paired device said where %s comes from; add its repository by hand", pkg)
		}
		if _, err := r.Sy.AddRepository(ctx, repo, ""); err != nil {
			return nil, fmt.Errorf("add repository %s: %w", repo, err)
		}
	}
	return (&mutationResolver{r}).InstallExtension(ctx, pkg)
}

// folderName trims what the user typed, so "Reading " and "Reading" merge across devices and look the same.
func folderName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("folder name can't be empty")
	}
	return name, nil
}
