package embedded

import (
	"context"
	"testing"
)

type fakeDiscovery string

func (f fakeDiscovery) Browse(int64) (string, error) { return string(f), nil }

func TestNativeDiscoveryDropsThisDeviceAndIncompleteEntries(t *testing.T) {
	SetSyncDiscovery(fakeDiscovery(`[{"node":"me","name":"Phone","address":"http://10.0.0.2:6008"},
		{"node":"desk","name":"Desk","address":"http://10.0.0.5:6008"},{"node":"half","name":"No address"}]`))
	defer SetSyncDiscovery(nil)
	found, err := discoveryOption()(context.Background(), "me")
	if err != nil || len(found) != 1 || found[0].Node != "desk" {
		t.Fatalf("found %+v, %v", found, err)
	}
	if SyncAdvert() != "" {
		t.Fatal("advertised while stopped")
	}
}
