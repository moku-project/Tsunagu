package mokusync

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"
)

// watchWait is how long a peer holds a watcher's exchange open waiting for a change.
var watchWait = 25 * time.Second

var compactEvery = 10 * time.Minute

// unpairedAfter is how long a peer must keep refusing this one's token before it counts as unpaired, which
// covers the moment after pairing when the other side hasn't saved this one's token yet.
var unpairedAfter = time.Minute

// Run captures local changes every interval, keeps one watcher per paired instance so remote changes
// arrive within seconds, and fetches missing library titles after remote changes or every idle.
func (e *Engine) Run(ctx context.Context, client *http.Client, interval, idle time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	watchers := map[string]context.CancelFunc{}
	defer func() {
		for _, stop := range watchers {
			stop()
		}
	}()
	var lastResolve, lastDiscover, lastCompact time.Time
	for {
		woke := false
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		case <-e.wake.wait():
			// Held requests from before the device slept are likely dead; start over at once.
			woke = true
			for key, stop := range watchers {
				stop()
				delete(watchers, key)
			}
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
			if e.yields(nodes[i]) {
				continue
			}
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
		// After a wake the device may be on another network, so paired devices are looked for right away.
		if (woke && len(peers) > 0) || (e.unreachable.Load() > 0 && time.Since(lastDiscover) >= time.Minute) {
			lastDiscover = time.Now()
			if n, err := e.rediscover(ctx); err != nil && ctx.Err() == nil {
				log.Printf("moku-sync: look for moved devices: %v", err)
			} else if n > 0 {
				log.Printf("moku-sync: %d paired device(s) moved to a new address", n)
			}
		}
		if time.Since(lastCompact) >= compactEvery {
			lastCompact = time.Now()
			if _, err := e.Compact(ctx); err != nil && ctx.Err() == nil {
				log.Printf("moku-sync: drop old deletion records: %v", err)
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

// Wake makes the runner sync with every paired instance now, for when the app returns to the foreground.
func (e *Engine) Wake() { e.wake.broadcast() }

// yields skips dialing a peer that is already watching this instance: one held exchange carries changes both
// ways. When both dial, the higher node ID gives way, so exactly one connection remains.
func (e *Engine) yields(node string) bool {
	if !e.watchers.watching(node) {
		return false
	}
	_, failing := e.failing.Load(node)
	return failing || e.node > node
}

// watch keeps a held exchange open with one peer, backing off while it's unreachable.
func (e *Engine) watch(ctx context.Context, p HTTPPeer, node string, interval time.Duration) {
	backoff := interval
	failing := false
	var refusedSince time.Time
	defer func() {
		if failing {
			e.unreachable.Add(-1)
			e.failing.Delete(node)
		}
	}()
	for ctx.Err() == nil {
		start := time.Now()
		err := e.syncWith(ctx, p, node, watchWait)
		if ctx.Err() != nil {
			return
		}
		if !errors.Is(err, errUnauthorized) {
			refusedSince = time.Time{}
		} else if refusedSince.IsZero() {
			refusedSince = time.Now()
		} else if time.Since(refusedSince) >= unpairedAfter {
			log.Printf("moku-sync: %s unpaired this device; forgetting it", p.Address)
			if err := e.RemovePeer(ctx, node); err != nil {
				log.Printf("moku-sync: forget %s: %v", node, err)
			}
			return
		}
		wait := interval
		if err != nil {
			// Peers are often asleep or away; log once per outage, not on every retry.
			if !failing {
				log.Printf("moku-sync: watch %s: %v", p.Address, err)
				e.unreachable.Add(1)
				e.failing.Store(node, true)
			}
			failing = true
			backoff = min(backoff*2, 2*time.Minute)
			wait = backoff
		} else {
			if failing {
				e.unreachable.Add(-1)
				e.failing.Delete(node)
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
