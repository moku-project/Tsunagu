package mokusync

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/dns/dnsmessage"
	"golang.org/x/net/ipv4"
)

func query(id uint16, qs ...dnsmessage.Question) []byte {
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{ID: id})
	_ = b.StartQuestions()
	for _, q := range qs {
		_ = b.Question(q)
	}
	msg, _ := b.Finish()
	return msg
}

func TestAdvertAnswersABrowse(t *testing.T) {
	ad := newAdvert("0123456789abcdef", "Living room PC", 6008)
	ips := [][4]byte{{192, 168, 1, 20}}
	reply := ad.answer(browseQuery(), ips, true)
	if reply == nil {
		t.Fatal("no answer to a _moku._tcp browse")
	}
	s := newBrowseState()
	s.add(reply)
	got := s.found("someone-else")
	want := Nearby{Node: "0123456789abcdef", Name: "Living room PC", Address: "http://192.168.1.20:6008"}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("found %+v, want %+v", got, want)
	}
	if found := s.found("0123456789abcdef"); len(found) != 0 {
		t.Fatalf("an instance found itself: %+v", found)
	}
}

func TestAdvertAnswersOnlyWhatsAsked(t *testing.T) {
	ad := newAdvert("0123456789abcdef", "PC", 6008)
	other := dnsmessage.Question{Name: dnsmessage.MustNewName("_http._tcp.local."), Type: dnsmessage.TypePTR, Class: dnsmessage.ClassINET}
	if ad.answer(query(1, other), nil, false) != nil {
		t.Fatal("answered a query for another service")
	}
	host := dnsmessage.Question{Name: dnsmessage.MustNewName("MOKU-0123456789AB.local."), Type: dnsmessage.TypeA, Class: dnsmessage.ClassINET}
	reply := ad.answer(query(7, host), [][4]byte{{10, 0, 0, 2}}, true)
	var p dnsmessage.Parser
	h, err := p.Start(reply)
	if err != nil || h.ID != 7 {
		t.Fatalf("legacy reply header %+v, %v", h, err)
	}
	_ = p.SkipAllQuestions()
	rh, err := p.AnswerHeader()
	if err != nil || rh.Type != dnsmessage.TypeA || rh.TTL > 10 || rh.Class != dnsmessage.ClassINET {
		t.Fatalf("answer %+v, %v", rh, err)
	}
	if ad.answer(reply, nil, false) != nil {
		t.Fatal("answered a response")
	}
}

func TestGoodbyeRemovesTheInstance(t *testing.T) {
	ad := newAdvert("0123456789abcdef", "PC", 6008)
	s := newBrowseState()
	s.add(ad.announcement([][4]byte{{10, 0, 0, 2}}, -1))
	if len(s.found("")) != 1 {
		t.Fatal("announcement not found")
	}
	s.add(ad.announcement([][4]byte{{10, 0, 0, 2}}, 0))
	if got := s.found(""); len(got) != 0 {
		t.Fatalf("still found after goodbye: %+v", got)
	}
}

func TestAdvertNamesStayValid(t *testing.T) {
	long := strings.Repeat("é", 40) + ".lan"
	ad := newAdvert("abc", long, 1)
	label := strings.TrimSuffix(ad.instance, "."+serviceName)
	if len(label) > 63 || strings.Contains(label, ".") {
		t.Fatalf("instance label %q", label)
	}
	if ad.announcement(nil, -1) == nil {
		t.Fatal("couldn't build an announcement")
	}
	if newAdvert("abc", "  ", 1).instance != "Tsunagu."+serviceName {
		t.Fatal("blank name not replaced")
	}
}

func TestOnlyAutomaticAddressesAreReplaced(t *testing.T) {
	for addr, want := range map[string]bool{
		"http://192.168.1.4:6008":     true,
		"http://desktop.local:6008":   true,
		"https://192.168.1.4":         false,
		"http://sync.example.com":     false,
		"http://[fe80::1]:6008":       true,
		"not a url\x7f":               false,
		"https://moku.tailnet.ts.net": false,
	} {
		if got := replaceable(addr); got != want {
			t.Errorf("replaceable(%q) = %v, want %v", addr, got, want)
		}
	}
}

func TestPeerListenerServesOnlyPeerRoutes(t *testing.T) {
	hub := newNode(t, "desktop", nil)
	srv := httptest.NewServer(&Handler{E: hub.e, PeerOnly: true})
	defer srv.Close()
	res, err := srv.Client().Get(srv.URL + "/api/sync/v1/status")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status on the peer listener = %d, want 404", res.StatusCode)
	}
	phone := newNode(t, "phone", nil)
	code, _, _ := hub.e.PairingCode(hub.ctx)
	if _, err := phone.e.Connect(phone.ctx, srv.Client(), srv.URL, code); err != nil {
		t.Fatalf("pairing through the peer listener: %v", err)
	}
}

func TestGuessingVoidsOpenPairingCodes(t *testing.T) {
	hub := newNode(t, "desktop", nil)
	srv := serve(t, hub)
	code, _, _ := hub.e.PairingCode(hub.ctx)
	guesser := newNode(t, "guesser", nil)
	for range maxPairFailures {
		if _, err := guesser.e.Connect(guesser.ctx, srv.Client(), srv.URL, "AAAAAAAA"); err == nil {
			t.Fatal("a wrong code paired")
		}
	}
	phone := newNode(t, "phone", nil)
	if _, err := phone.e.Connect(phone.ctx, srv.Client(), srv.URL, code); err == nil {
		t.Fatal("a code still worked after repeated wrong guesses")
	}
	fresh, _, _ := hub.e.PairingCode(hub.ctx)
	if _, err := phone.e.Connect(phone.ctx, srv.Client(), srv.URL, fresh); err != nil {
		t.Fatalf("a new code after the lockout: %v", err)
	}
}

func TestRunnerFollowsAPeerToItsNewAddress(t *testing.T) {
	defer func(w time.Duration) { watchWait = w }(watchWait)
	watchWait = time.Second
	hub := newNode(t, "hub", nil)
	phone := newNode(t, "phone", nil)
	hub.e.wall, phone.e.wall = nil, nil
	before := serve(t, hub)
	code, _, _ := hub.e.PairingCode(hub.ctx)
	if _, err := phone.e.Connect(phone.ctx, before.Client(), before.URL, code); err != nil {
		t.Fatal(err)
	}
	before.Close()
	after := serve(t, hub)
	phone.e.SetDiscovery(func(ctx context.Context, self string) ([]Nearby, error) {
		return []Nearby{{Node: hub.e.Node(), Name: "hub", Address: after.URL}}, nil
	})

	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	go hub.e.Run(ctx, after.Client(), 50*time.Millisecond, time.Hour)
	go phone.e.Run(ctx, after.Client(), 50*time.Millisecond, time.Hour)
	hub.setInLibrary(2, true)
	deadline := time.Now().Add(5 * time.Second)
	for !phone.inLibrary(2) {
		if time.Now().After(deadline) {
			t.Fatal("phone never reached the hub at its new address")
		}
		time.Sleep(25 * time.Millisecond)
	}
	nearby, err := phone.e.Nearby(phone.ctx)
	if err != nil || len(nearby) != 1 || !nearby[0].Paired {
		t.Fatalf("nearby %+v, %v", nearby, err)
	}
}

// Real multicast needs a LAN interface, so it only runs when asked: MOKU_SYNC_MDNS=1.
func TestAdvertiseAndBrowseOnTheLAN(t *testing.T) {
	if os.Getenv("MOKU_SYNC_MDNS") == "" {
		t.Skip("set MOKU_SYNC_MDNS=1 to use the real network")
	}
	hub := newNode(t, "mdns test hub", nil)
	ctx, stop := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- hub.e.Advertise(ctx, 6123) }()
	defer func() {
		stop()
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	want := func(found []Nearby) bool {
		for _, n := range found {
			if n.Node == hub.e.Node() && n.Name == "mdns test hub" && strings.HasSuffix(n.Address, ":6123") {
				return true
			}
		}
		return false
	}
	t.Run("one-shot query", func(t *testing.T) {
		for try := 0; try < 5; try++ {
			found, err := Browse(context.Background(), time.Second, "")
			if err != nil {
				t.Fatal(err)
			}
			if want(found) {
				return
			}
		}
		t.Fatal("the advertised instance wasn't found")
	})
	// System resolvers query from port 5353 and expect the answer multicast back.
	t.Run("multicast query", func(t *testing.T) {
		lc := net.ListenConfig{Control: reuseAddr}
		c, err := lc.ListenPacket(context.Background(), "udp4", "0.0.0.0:5353")
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		pc := ipv4.NewPacketConn(c)
		for _, ifi := range multicastInterfaces() {
			_ = pc.JoinGroup(&ifi, mdnsGroup)
		}
		_ = pc.SetMulticastLoopback(true)
		if _, err := c.WriteTo(browseQuery(), mdnsGroup); err != nil {
			t.Fatal(err)
		}
		_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
		s := newBrowseState()
		buf := make([]byte, 9000)
		for {
			n, _, err := c.ReadFrom(buf)
			if err != nil {
				break
			}
			if s.add(buf[:n]); want(s.found("")) {
				return
			}
		}
		t.Fatal("no multicast answer")
	})
}
