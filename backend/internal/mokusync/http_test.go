package mokusync

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serve(t *testing.T, n *node) *httptest.Server {
	srv := httptest.NewServer(&Handler{E: n.e})
	t.Cleanup(srv.Close)
	return srv
}

func TestPairThenSyncOverHTTP(t *testing.T) {
	hub := newNode(t, "desktop", nil)
	phone := newNode(t, "phone", []int{3, 1, 4, 0, 5, 2})
	srv := serve(t, hub)

	code, _, err := hub.e.PairingCode(hub.ctx)
	if err != nil {
		t.Fatal(err)
	}
	node, err := phone.e.Connect(phone.ctx, srv.Client(), srv.URL, strings.ToLower(code))
	if err != nil {
		t.Fatal(err)
	}
	if node != hub.e.Node() {
		t.Fatalf("paired with %s, want %s", node, hub.e.Node())
	}

	hub.setInLibrary(1, true)
	hub.setRead(1, 2, true)
	phone.setInLibrary(4, true)
	phone.setEpisodePosition(4, 0, 300, 1440)

	peers, nodes, err := phone.e.dialPeers(phone.ctx)
	if err != nil || len(peers) != 1 {
		t.Fatalf("dial peers: %v %v", peers, err)
	}
	peers[0].Client = srv.Client()
	if err := phone.e.SyncWith(phone.ctx, peers[0], nodes[0]); err != nil {
		t.Fatal(err)
	}
	if d := diff(hub.libraryState(), phone.libraryState()); d != "" {
		t.Fatalf("libraries differ:\n%s", d)
	}
	if d := diff(hub.crdtState(), phone.crdtState()); d != "" {
		t.Fatalf("sync state differs:\n%s", d)
	}
}

func TestPairingCodesWorkOnce(t *testing.T) {
	hub := newNode(t, "desktop", nil)
	a := newNode(t, "a", nil)
	b := newNode(t, "b", nil)
	srv := serve(t, hub)
	code, _, err := hub.e.PairingCode(hub.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.e.Connect(a.ctx, srv.Client(), srv.URL, "WRONGCODE"); err == nil {
		t.Fatal("a wrong code paired")
	}
	if _, err := a.e.Connect(a.ctx, srv.Client(), srv.URL, code); err != nil {
		t.Fatal(err)
	}
	if _, err := b.e.Connect(b.ctx, srv.Client(), srv.URL, code); err == nil {
		t.Fatal("a used code paired a second device")
	}
}

func TestExchangeRejectsUnknownTokens(t *testing.T) {
	hub := newNode(t, "desktop", nil)
	phone := newNode(t, "phone", nil)
	srv := serve(t, hub)
	code, _, _ := hub.e.PairingCode(hub.ctx)
	if _, err := phone.e.Connect(phone.ctx, srv.Client(), srv.URL, code); err != nil {
		t.Fatal(err)
	}
	stranger := newNode(t, "stranger", nil)
	for name, peer := range map[string]HTTPPeer{
		"no token":    {Address: srv.URL, Client: srv.Client()},
		"wrong token": {Address: srv.URL, Token: "deadbeef", Client: srv.Client()},
	} {
		err := stranger.e.SyncWith(stranger.ctx, peer, hub.e.Node())
		if err == nil || (!strings.Contains(err.Error(), "401") && !errors.Is(err, errUnauthorized)) {
			t.Errorf("%s: got %v, want 401", name, err)
		}
	}
	// The phone's token doesn't let another node speak as itself.
	peers, _, _ := phone.e.dialPeers(phone.ctx)
	peers[0].Client = srv.Client()
	if err := stranger.e.SyncWith(stranger.ctx, peers[0], hub.e.Node()); err == nil {
		t.Error("a token was accepted from a different node")
	}
}

func TestUnknownRoutesAreNotFound(t *testing.T) {
	hub := newNode(t, "desktop", nil)
	srv := serve(t, hub)
	res, err := srv.Client().Get(srv.URL + PeerPrefix + "exchange")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("GET exchange = %d, want 404", res.StatusCode)
	}
}
