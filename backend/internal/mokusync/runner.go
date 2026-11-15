package mokusync

import (
	"context"
	"log"
	"net/http"
	"time"
)

// Run captures local changes every interval and syncs with each paired instance this one dials.
// A dial runs right after a capture that found changes, and at least every idle interval otherwise.
func (e *Engine) Run(ctx context.Context, client *http.Client, interval, idle time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	var lastDial time.Time
	failing := map[string]bool{}
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
		if n == 0 && time.Since(lastDial) < idle {
			continue
		}
		lastDial = time.Now()
		peers, nodes, err := e.dialPeers(ctx)
		if err != nil {
			log.Printf("moku-sync: list peers: %v", err)
			continue
		}
		for i, p := range peers {
			p.Client = client
			if err := e.SyncWith(ctx, p, nodes[i]); err != nil {
				// Peers are often asleep or away; log a failure once, not every tick.
				if !failing[nodes[i]] {
					log.Printf("moku-sync: sync with %s: %v", p.Address, err)
				}
				failing[nodes[i]] = true
				continue
			}
			failing[nodes[i]] = false
		}
		if e.resolver != nil {
			if _, err := e.ResolveMissing(ctx, e.resolver); err != nil && ctx.Err() == nil {
				log.Printf("moku-sync: fetch missing titles: %v", err)
			}
		}
	}
}
