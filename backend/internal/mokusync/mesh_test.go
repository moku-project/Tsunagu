package mokusync

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// listen gives a node its own peer listener, the way every device now has one.
func listen(t *testing.T, n *node) *httptest.Server {
	srv := httptest.NewServer(&Handler{E: n.e, PeerOnly: true})
	t.Cleanup(srv.Close)
	_, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	p, _ := strconv.Atoi(port)
	n.e.SetSyncPort(p)
	return srv
}

func eventually(t *testing.T, what string, within time.Duration, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(within)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("%s didn't happen within %s", what, within)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func run(t *testing.T, nodes ...*node) {
	ctx, stop := context.WithCancel(context.Background())
	t.Cleanup(stop)
	for _, n := range nodes {
		n.e.wall = nil
		go n.e.Run(ctx, nil, 50*time.Millisecond, time.Hour)
	}
}

func shortWatch(t *testing.T) {
	w, u := watchWait, unpairedAfter
	watchWait, unpairedAfter = time.Second, 300*time.Millisecond
	t.Cleanup(func() { watchWait, unpairedAfter = w, u })
}

func (n *node) peerAddress(node string) string {
	var a string
	_ = n.db.QueryRowContext(n.ctx, `SELECT address FROM sync_peers WHERE node = ?`, node).Scan(&a)
	return a
}

func TestTwoPhonesSyncWhicheverIsOpen(t *testing.T) {
	shortWatch(t)
	a := newNode(t, "phone a", nil)
	b := newNode(t, "phone b", nil)
	srvA, srvB := listen(t, a), listen(t, b)
	code, _, _ := b.e.PairingCode(b.ctx)
	if _, err := a.e.ConnectTo(a.ctx, nil, srvB.URL, code, b.e.Node()); err != nil {
		t.Fatal(err)
	}
	// Pairing hands each side what it needs to start a sync with the other.
	if got := b.peerAddress(a.e.Node()); got != srvA.URL {
		t.Fatalf("b dials a at %q, want %q", got, srvA.URL)
	}
	if got := a.peerAddress(b.e.Node()); got != srvB.URL {
		t.Fatalf("a dials b at %q, want %q", got, srvB.URL)
	}

	// Only the device that issued the code runs; it still reaches the one that entered it.
	ctx, stop := context.WithCancel(context.Background())
	b.e.wall = nil
	go b.e.Run(ctx, nil, 50*time.Millisecond, time.Hour)
	a.setInLibrary(2, true)
	eventually(t, "a's add reaching b with only b running", 3*time.Second, func() bool { return b.inLibrary(2) })
	stop()

	run(t, a)
	b.e.wall = nil
	go b.e.Run(t.Context(), nil, 50*time.Millisecond, time.Hour)
	b.setRead(2, 1, true)
	eventually(t, "b's read reaching a", 3*time.Second, func() bool { read, _, _ := a.chapterRow(2, 1); return read })
	a.setInLibrary(3, true)
	eventually(t, "a's add reaching b", 3*time.Second, func() bool { return b.inLibrary(3) })

	// Both dial at first; the higher node ID gives way so one connection carries both directions.
	single := func() bool { return holding(a, b) != holding(b, a) }
	eventually(t, "a single connection", 5*time.Second, single)
	time.Sleep(1500 * time.Millisecond)
	if !single() {
		t.Fatal("both devices went back to dialing each other")
	}
	st, err := a.e.Status(a.ctx)
	if err != nil || len(st.Peers) != 1 || !st.Peers[0].Online {
		t.Fatalf("a's status %+v %v", st.Peers, err)
	}
}

func TestPeersFollowEachOtherToNewAddresses(t *testing.T) {
	shortWatch(t)
	hub := newNode(t, "desktop", nil)
	phone := newNode(t, "phone", nil)
	srvHub := listen(t, hub)
	srvPhone := listen(t, phone)
	code, _, _ := hub.e.PairingCode(hub.ctx)
	if _, err := phone.e.Connect(phone.ctx, nil, srvHub.URL, code); err != nil {
		t.Fatal(err)
	}
	// The phone got a new address; the desktop learns it the next time the phone calls.
	srvPhone.Close()
	moved := listen(t, phone)
	if err := phone.e.SyncNow(phone.ctx, nil); err != nil {
		t.Fatal(err)
	}
	if got := hub.peerAddress(phone.e.Node()); got != moved.URL {
		t.Fatalf("desktop dials the phone at %q, want %q", got, moved.URL)
	}

	// A rename shows up on the other device with the next sync.
	if err := phone.e.SetName(phone.ctx, "Shozikan's Pixel"); err != nil {
		t.Fatal(err)
	}
	if err := phone.e.SyncNow(phone.ctx, nil); err != nil {
		t.Fatal(err)
	}
	if peers, _ := hub.e.Peers(hub.ctx); len(peers) != 1 || peers[0].Name != "Shozikan's Pixel" {
		t.Fatalf("desktop shows %+v after the rename", peers)
	}

	// An address the user typed, like a hostname behind a proxy, is left alone.
	hub.exec(`UPDATE sync_peers SET address = 'https://phone.example' WHERE node = ?`, phone.e.Node())
	if err := phone.e.SyncNow(phone.ctx, nil); err != nil {
		t.Fatal(err)
	}
	if got := hub.peerAddress(phone.e.Node()); got != "https://phone.example" {
		t.Fatalf("a chosen address was replaced with %q", got)
	}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = "192.168.1.20:5555"
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	if callerHost(req) != "" {
		t.Fatal("trusted the address of a proxy")
	}
}

func TestUnpairingReachesTheOtherDevice(t *testing.T) {
	shortWatch(t)
	hub := newNode(t, "desktop", nil)
	phone := newNode(t, "phone", nil)
	srv := listen(t, hub)
	code, _, _ := hub.e.PairingCode(hub.ctx)
	if _, err := phone.e.Connect(phone.ctx, nil, srv.URL, code); err != nil {
		t.Fatal(err)
	}
	if err := hub.e.RemovePeer(hub.ctx, phone.e.Node()); err != nil {
		t.Fatal(err)
	}
	run(t, phone)
	eventually(t, "the phone forgetting the desktop", 5*time.Second, func() bool {
		peers, _ := phone.e.Peers(phone.ctx)
		return len(peers) == 0
	})
}

func TestWakeRetriesAtOnce(t *testing.T) {
	shortWatch(t)
	hub := newNode(t, "desktop", nil)
	phone := newNode(t, "phone", nil)
	srv := listen(t, hub)
	code, _, _ := hub.e.PairingCode(hub.ctx)
	if _, err := phone.e.Connect(phone.ctx, nil, srv.URL, code); err != nil {
		t.Fatal(err)
	}
	addr := srv.Listener.Addr().String()
	srv.Close()
	run(t, phone)
	// Long enough for the retry wait to grow past what the check below allows.
	time.Sleep(3 * time.Second)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Skipf("port taken again: %v", err)
	}
	back := &httptest.Server{Listener: ln, Config: &http.Server{Handler: &Handler{E: hub.e, PeerOnly: true}}}
	back.Start()
	t.Cleanup(back.Close)
	hub.setInLibrary(4, true)
	hub.capture()
	phone.e.Wake()
	eventually(t, "a sync right after waking", 1500*time.Millisecond, func() bool { return phone.inLibrary(4) })
}

func TestDeletionRecordsGoOnceEveryDeviceHasThem(t *testing.T) {
	a := newNode(t, "a", nil)
	b := newNode(t, "b", nil)
	c := newNode(t, "c", nil)
	removed := func(n *node) int {
		var k int
		_ = n.db.QueryRowContext(n.ctx, `SELECT (SELECT COUNT(*) FROM sync_set_removed) + (SELECT COUNT(*) FROM sync_ops WHERE type = 'remove')`).Scan(&k)
		return k
	}
	compact := func(n *node) int {
		k, err := n.e.Compact(n.ctx)
		if err != nil {
			t.Fatal(err)
		}
		return k
	}
	a.setInLibrary(1, true)
	a.capture()
	a.syncWith(b)
	b.syncWith(c)
	a.wall++
	a.setInLibrary(1, false)
	a.capture()
	a.syncWith(b)
	if k := compact(a); k != 0 {
		t.Fatalf("dropped %d deletion records c hasn't seen", k)
	}
	b.syncWith(c)
	a.syncWith(b)
	if k := compact(a); k != 1 {
		t.Fatalf("dropped %d deletion records, want 1", k)
	}
	if k := removed(a); k != 0 {
		t.Fatalf("%d deletion rows left", k)
	}
	compact(b)
	compact(c)
	a.syncWith(b)
	b.syncWith(c)
	for _, n := range []*node{b, c} {
		if d := diff(a.crdtState(), n.crdtState()); d != "" {
			t.Fatalf("state differs from %s after compacting:\n%s", n.e.name, d)
		}
		if n.inLibrary(1) {
			t.Fatalf("%s brought the removed title back", n.e.name)
		}
	}
	if _, ok, _ := readTitleInfo(c.ctx, c.db, mediaKey(1)); ok {
		t.Fatal("title info stayed after the title left every library")
	}

	// A device nobody has heard from in a long time stops holding records back.
	d := newNode(t, "d", nil)
	a.syncWith(d)
	a.wall++
	a.setInLibrary(1, true)
	a.capture()
	a.wall++
	a.setInLibrary(1, false)
	a.capture()
	a.syncWith(b)
	b.syncWith(c)
	a.syncWith(b)
	if k := compact(a); k != 0 {
		t.Fatalf("dropped %d records d hasn't seen", k)
	}
	a.exec(`UPDATE sync_seen SET reported_at = 1 WHERE node = ?`, d.e.Node())
	if k := compact(a); k != 1 {
		t.Fatalf("dropped %d records with d long gone, want 1", k)
	}
}

func holding(n, peer *node) bool {
	n.e.watchers.mu.Lock()
	defer n.e.watchers.mu.Unlock()
	w := n.e.watchers.m[peer.e.Node()]
	return w != nil && w.holding > 0
}

func TestAStrangerAtAPeersAddressDoesNotUnpairIt(t *testing.T) {
	shortWatch(t)
	hub := newNode(t, "desktop", nil)
	phone := newNode(t, "phone", nil)
	srv := listen(t, hub)
	code, _, _ := hub.e.PairingCode(hub.ctx)
	if _, err := phone.e.Connect(phone.ctx, nil, srv.URL, code); err != nil {
		t.Fatal(err)
	}
	// Another Tsunagu now answers at the desktop's address and refuses the phone's token.
	stranger := listen(t, newNode(t, "someone else", nil))
	phone.exec(`UPDATE sync_peers SET address = ? WHERE node = ?`, stranger.URL, hub.e.Node())
	peers, nodes, _ := phone.e.dialPeers(phone.ctx)
	if err := phone.e.SyncWith(phone.ctx, peers[0], nodes[0]); !errors.Is(err, errWrongDevice) {
		t.Fatalf("syncing with a stranger: %v, want errWrongDevice", err)
	}
	run(t, phone)
	time.Sleep(3 * unpairedAfter)
	if got, _ := phone.e.Peers(phone.ctx); len(got) != 1 {
		t.Fatalf("a stranger's refusal unpaired the desktop: %+v", got)
	}
}
