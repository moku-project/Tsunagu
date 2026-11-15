package mokusync

import (
	"context"
	"log"
	"net/http"
	"time"
)

// watchWait is how long a peer holds a watcher's exchange open waiting for a change.
var watchWait = 25 * time.Second

// Run captures local changes every interval, keeps one watcher per instance this one dials so remote
// changes arrive within seconds, and fetches missing library titles after remote changes or every idle.
func (e *Engine) Run(ctx context.Context, client *http.Client, interval, idle time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	watchers := map[string]context.CancelFunc{}
	defer func() {
		for _, stop := range watchers {
			stop()
		}
	}()
	var lastResolve, lastDiscover time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		n, err := e.Capture(ctx)
		if err != nil {
			log.Printf("moku-sync: capture: %v", err)
			continue
		}
		if _, err := e.ProjectPending(ctx); err != nil {
			log.Printf("moku-sync: apply pending: %v", err)
		}
		peers, nodes, err := e.dialPeers(ctx)
		if err != nil {
			log.Printf("moku-sync: list peers: %v", err)
			continue
		}
		live := map[string]bool{}
		for i, p := range peers {
			p.Client = client
			key := nodes[i] + " " + p.Address
			live[key] = true
			if _, ok := watchers[key]; !ok {
				wctx, stop := context.WithCancel(ctx)
				watchers[key] = stop
				go e.watch(wctx, p, nodes[i], interval)
			}
			// A watcher only learns of remote changes; local ones are pushed right away.
			if n > 0 {
				if err := e.SyncWith(ctx, p, nodes[i]); err != nil && ctx.Err() == nil {
					log.Printf("moku-sync: push to %s: %v", p.Address, err)
				}
			}
		}
		for key, stop := range watchers {
			if !live[key] {
				stop()
				delete(watchers, key)
			}
		}
		if e.unreachable.Load() > 0 && time.Since(lastDiscover) >= time.Minute {
			lastDiscover = time.Now()
			if n, err := e.rediscover(ctx); err != nil && ctx.Err() == nil {
				log.Printf("moku-sync: look for moved devices: %v", err)
			} else if n > 0 {
				log.Printf("moku-sync: %d paired device(s) moved to a new address", n)
			}
		}
		if e.resolver != nil && (e.remoteChanged.Swap(false) || time.Since(lastResolve) >= idle) {
			lastResolve = time.Now()
			if _, err := e.ResolveMissing(ctx, e.resolver); err != nil && ctx.Err() == nil {
				log.Printf("moku-sync: fetch missing titles: %v", err)
			}
		}
	}
}

// watch keeps a held exchange open with one peer, backing off while it's unreachable.
func (e *Engine) watch(ctx context.Context, p HTTPPeer, node string, interval time.Duration) {
	backoff := interval
	failing := false
	defer func() {
		if failing {
			e.unreachable.Add(-1)
		}
	}()
	for ctx.Err() == nil {
		start := time.Now()
		err := e.syncWith(ctx, p, node, watchWait)
		if ctx.Err() != nil {
			return
		}
		wait := interval
		if err != nil {
			// Peers are often asleep or away; log once per outage, not on every retry.
			if !failing {
				log.Printf("moku-sync: watch %s: %v", p.Address, err)
				e.unreachable.Add(1)
			}
			failing = true
			backoff = min(backoff*2, 2*time.Minute)
			wait = backoff
		} else {
			if failing {
				e.unreachable.Add(-1)
			}
			failing, backoff = false, interval
			// A peer that answers at once without holding still gets polled at a steady pace.
			if time.Since(start) >= interval {
				wait = 0
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}
