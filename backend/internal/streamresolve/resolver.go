package streamresolve

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"tsunagu/backend/internal/sandbox"
	sandboxv1 "tsunagu/backend/internal/sandbox/gen/sandbox/v1"
)

const (
	cacheTTL = 30 * time.Second
	cacheCap = 128
)

type Resolver struct {
	sc *sandbox.SupervisedClient
	sf singleflight.Group

	mu sync.Mutex
	m  map[string]cached
}

type cached struct {
	info *sandboxv1.StreamInfo
	at   time.Time
}

func New(sc *sandbox.SupervisedClient) *Resolver {
	return &Resolver{sc: sc, m: map[string]cached{}}
}

func key(pkg, entryID, episodeID string) string {
	return pkg + "\x00" + entryID + "\x00" + episodeID
}

func (r *Resolver) Resolve(ctx context.Context, pkg, entryID, episodeID string) (*sandboxv1.StreamInfo, error) {
	k := key(pkg, entryID, episodeID)

	r.mu.Lock()
	if c, ok := r.m[k]; ok && time.Since(c.at) < cacheTTL {
		r.mu.Unlock()
		return c.info, nil
	}
	r.mu.Unlock()

	v, err, _ := r.sf.Do(k, func() (any, error) {
		client, err := r.sc.Ensure(ctx)
		if err != nil {
			return nil, err
		}
		info, err := client.GetVideoStream(ctx, pkg, entryID, episodeID)
		if err != nil {
			return nil, err
		}
		r.mu.Lock()
		now := time.Now()
		for mk, c := range r.m {
			if now.Sub(c.at) >= cacheTTL {
				delete(r.m, mk)
			}
		}
		if len(r.m) >= cacheCap {
			var oldK string
			var oldT time.Time
			for mk, c := range r.m {
				if oldK == "" || c.at.Before(oldT) {
					oldK, oldT = mk, c.at
				}
			}
			delete(r.m, oldK)
		}
		r.m[k] = cached{info: info, at: now}
		r.mu.Unlock()
		return info, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*sandboxv1.StreamInfo), nil
}

func (r *Resolver) Invalidate(pkg, entryID, episodeID string) {
	r.mu.Lock()
	delete(r.m, key(pkg, entryID, episodeID))
	r.mu.Unlock()
}
