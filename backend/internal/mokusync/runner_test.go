package mokusync

import (
	"context"
	"testing"
	"time"
)

func TestHeldExchangeAnswersWhenSomethingChanges(t *testing.T) {
	hub := newNode(t, "hub", nil)
	phone := newNode(t, "phone", nil)
	phone.syncWith(hub)

	done := make(chan error, 1)
	start := time.Now()
	go func() { done <- phone.e.syncWith(phone.ctx, LocalPeer{hub.e}, hub.e.Node(), 10*time.Second) }()
	time.Sleep(100 * time.Millisecond)
	hub.setInLibrary(2, true)
	hub.capture()

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("held exchange didn't answer after the hub changed")
	}
	if took := time.Since(start); took > 3*time.Second {
		t.Fatalf("change took %v to arrive", took)
	}
	if !phone.inLibrary(2) {
		t.Fatal("the change the exchange woke for didn't reach the phone")
	}
}

func TestHeldExchangeTimesOutQuietly(t *testing.T) {
	hub := newNode(t, "hub", nil)
	phone := newNode(t, "phone", nil)
	phone.syncWith(hub)
	start := time.Now()
	if err := phone.e.syncWith(phone.ctx, LocalPeer{hub.e}, hub.e.Node(), 200*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if took := time.Since(start); took < 150*time.Millisecond || took > 2*time.Second {
		t.Fatalf("idle exchange returned after %v, want about 200ms", took)
	}
}

func TestRunnerKeepsPairedInstancesInStep(t *testing.T) {
	defer func(w time.Duration) { watchWait = w }(watchWait)
	watchWait = 2 * time.Second
	hub := newNode(t, "hub", nil)
	phone := newNode(t, "phone", []int{5, 4, 3, 2, 1, 0})
	// Real clocks here: the runner stamps changes as they happen.
	hub.e.wall, phone.e.wall = nil, nil
	srv := serve(t, hub)
	code, _, err := hub.e.PairingCode(hub.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := phone.e.Connect(phone.ctx, srv.Client(), srv.URL, code); err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	go hub.e.Run(ctx, srv.Client(), 50*time.Millisecond, time.Hour)
	go phone.e.Run(ctx, srv.Client(), 50*time.Millisecond, time.Hour)

	eventually := func(what string, ok func() bool) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for !ok() {
			if time.Now().After(deadline) {
				t.Fatalf("%s didn't happen within 5s", what)
			}
			time.Sleep(25 * time.Millisecond)
		}
	}
	hub.setInLibrary(1, true)
	eventually("hub's library add reaching the phone", func() bool { return phone.inLibrary(1) })
	phone.setRead(1, 3, true)
	eventually("phone's read reaching the hub", func() bool { read, _, _ := hub.chapterRow(1, 3); return read })

	if err := phone.e.RemovePeer(phone.ctx, hub.e.Node()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	hub.setInLibrary(4, true)
	time.Sleep(500 * time.Millisecond)
	if phone.inLibrary(4) {
		t.Fatal("phone kept syncing after unpairing")
	}
}
