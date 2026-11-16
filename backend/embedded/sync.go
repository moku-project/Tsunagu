package embedded

import (
	"context"
	"encoding/json"
	"time"

	"tsunagu/backend/internal/mokusync"
)

// SyncDiscovery is the app's own browser for the _moku._tcp service, for platforms where Tsunagu can't open
// mDNS sockets itself (iOS without the multicast entitlement). Set it and the app advertises SyncAdvert too.
type SyncDiscovery interface {
	// Browse returns what it found within waitMillis as a JSON array of {"node","name","address"}, with the
	// address as "http://<ip>:<port>" and node and name taken from the service's TXT record.
	Browse(waitMillis int64) (string, error)
}

// SetSyncDiscovery uses the app's service discovery from the next Start; nil goes back to Tsunagu's own mDNS.
func SetSyncDiscovery(d SyncDiscovery) {
	mu.Lock()
	defer mu.Unlock()
	discovery = d
}

func discoveryOption() func(ctx context.Context, self string) ([]mokusync.Nearby, error) {
	if discovery == nil {
		return nil
	}
	d := discovery
	return func(ctx context.Context, self string) ([]mokusync.Nearby, error) {
		wait := int64(1500)
		if dl, ok := ctx.Deadline(); ok {
			wait = min(wait, time.Until(dl).Milliseconds())
		}
		raw, err := d.Browse(wait)
		if err != nil {
			return nil, err
		}
		var found []mokusync.Nearby
		if err := json.Unmarshal([]byte(raw), &found); err != nil {
			return nil, err
		}
		out := found[:0]
		for _, n := range found {
			if n.Node != "" && n.Node != self && n.Address != "" {
				out = append(out, n)
			}
		}
		return out, nil
	}
}

// SyncAdvert is what the app publishes as a _moku._tcp service when it provides SyncDiscovery: JSON with the
// instance name, port and TXT entries; show people the TXT name. It's "" while stopped, not listening or not yet named; read it again after naming the device.
func SyncAdvert() string {
	mu.Lock()
	defer mu.Unlock()
	if running == nil || running.Sync == nil {
		return ""
	}
	e := running.Sync
	port := e.SyncPort()
	if port == 0 || !e.Named() {
		return ""
	}
	b, _ := json.Marshal(map[string]any{
		"type": "_moku._tcp",
		"name": e.ServiceInstance(),
		"port": port,
		"txt":  map[string]string{"v": "1", "node": e.Node(), "name": e.Name()},
	})
	return string(b)
}

// SyncWake syncs with paired devices right away; call it when the app comes back to the foreground.
func SyncWake() {
	mu.Lock()
	defer mu.Unlock()
	if running != nil && running.Sync != nil {
		running.Sync.Wake()
	}
}
