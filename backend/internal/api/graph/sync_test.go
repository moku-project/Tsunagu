package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"

	"tsunagu/backend/internal/db"
	"tsunagu/backend/internal/mokusync"
)

func syncGraph(t *testing.T) (*mokusync.Engine, func(q string) map[string]any) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	e, err := mokusync.Open(context.Background(), conn)
	if err != nil {
		t.Fatal(err)
	}
	srv := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: &Resolver{Ms: e, DB: conn}}))
	return e, func(q string) map[string]any {
		body, _ := json.Marshal(map[string]string{"query": q})
		req := httptest.NewRequest(http.MethodPost, "/api/graphql", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		var out map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("%s: %v", rec.Body.String(), err)
		}
		return out
	}
}

func TestSyncStatusAndDeviceName(t *testing.T) {
	e, gql := syncGraph(t)
	out := gql(`mutation { setSyncDeviceName(name: "  Living room PC ") { node name peers { node } pendingTitles } }`)
	got, _ := json.Marshal(out["data"])
	want := `{"setSyncDeviceName":{"name":"Living room PC","node":"` + e.Node() + `","peers":[],"pendingTitles":0}}`
	if string(got) != want {
		t.Fatalf("got %s, want %s (errors %v)", got, want, out["errors"])
	}
	if out := gql(`mutation { setSyncDeviceName(name: " ") { name } }`); out["errors"] == nil {
		t.Fatal("an empty device name was accepted")
	}
}

func TestPairingThroughGraphQL(t *testing.T) {
	hub, hubGQL := syncGraph(t)
	_, phoneGQL := syncGraph(t)
	srv := httptest.NewServer(&mokusync.Handler{E: hub})
	defer srv.Close()

	out := hubGQL(`mutation { createSyncPairingCode { code expiresAt } }`)
	code := out["data"].(map[string]any)["createSyncPairingCode"].(map[string]any)["code"].(string)
	if len(code) != 8 {
		t.Fatalf("pairing code %q", code)
	}
	out = phoneGQL(`mutation { connectSyncPeer(address: "` + srv.URL + `", code: "` + code + `") { node name address } }`)
	peer, _ := json.Marshal(out["data"])
	if !strings.Contains(string(peer), hub.Node()) || !strings.Contains(string(peer), srv.URL) {
		t.Fatalf("connect returned %s (errors %v)", peer, out["errors"])
	}
	out = phoneGQL(`mutation { syncNow { peers { node lastSeenAt } } }`)
	if out["errors"] != nil {
		t.Fatalf("syncNow: %v", out["errors"])
	}
	out = phoneGQL(`mutation { removeSyncPeer(node: "` + hub.Node() + `") { peers { node } } }`)
	if got, _ := json.Marshal(out["data"]); string(got) != `{"removeSyncPeer":{"peers":[]}}` {
		t.Fatalf("after unpairing: %s (errors %v)", got, out["errors"])
	}
}

func TestSyncNearbyAndAddresses(t *testing.T) {
	hub, hubGQL := syncGraph(t)
	phone, phoneGQL := syncGraph(t)
	hub.SetSyncPort(6008)
	phone.SetDiscovery(func(ctx context.Context, self string) ([]mokusync.Nearby, error) {
		return []mokusync.Nearby{{Node: hub.Node(), Name: "Desk", Address: "http://192.168.1.9:6008"}}, nil
	})
	out := phoneGQL(`{ syncNearby { node name address paired } syncStatus { addresses } }`)
	got, _ := json.Marshal(out["data"])
	want := `{"syncNearby":[{"address":"http://192.168.1.9:6008","name":"Desk","node":"` + hub.Node() + `","paired":false}],"syncStatus":{"addresses":[]}}`
	if string(got) != want {
		t.Fatalf("got %s, want %s (errors %v)", got, want, out["errors"])
	}
	out = hubGQL(`{ syncStatus { addresses } }`)
	if out["errors"] != nil {
		t.Fatal(out["errors"])
	}
	for _, a := range out["data"].(map[string]any)["syncStatus"].(map[string]any)["addresses"].([]any) {
		if !strings.HasSuffix(a.(string), ":6008") {
			t.Fatalf("address %v", a)
		}
	}
	if out := hubGQL(`{ syncNearby { node } }`); out["errors"] == nil {
		t.Fatal("nearby worked with discovery off")
	}
}
