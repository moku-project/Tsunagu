package server

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"tsunagu/backend/internal/mokusync"
	"tsunagu/backend/internal/sandbox"
	"tsunagu/backend/internal/sync"
)

// syncResolver fetches a synced library title through the same path as opening it in the app.
type syncResolver struct {
	sy *sync.Syncer
	sc *sandbox.SupervisedClient
}

func (r syncResolver) Resolve(ctx context.Context, packageName, externalID string) error {
	c, err := r.sc.Ensure(ctx)
	if err != nil {
		return err
	}
	_, err = r.sy.ResolveMedia(ctx, c, packageName, externalID, true)
	return err
}

// listenForPeers serves pairing and sync on addr, apart from the rest of the API, and advertises it over mDNS
// unless the embedder advertises it natively.
func (s *Server) listenForPeers(ctx context.Context, e *mokusync.Engine, addr string, advertise bool) error {
	if !strings.Contains(addr, ":") {
		addr = ":" + addr
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		host, _, _ := net.SplitHostPort(addr)
		if ln, err = net.Listen("tcp", net.JoinHostPort(host, "0")); err != nil {
			return err
		}
		log.Printf("moku-sync: %s unavailable, listening on %s instead", addr, ln.Addr())
	}
	srv := &http.Server{
		Handler:           &mokusync.Handler{E: e, PeerOnly: true},
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	s.cleanups = append(s.cleanups, func() { _ = srv.Close() })
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("moku-sync: peer listener: %v", err)
		}
	}()
	port := ln.Addr().(*net.TCPAddr).Port
	e.SetSyncPort(port)
	log.Printf("moku-sync: other devices pair and sync on port %d", port)
	if !advertise {
		return nil
	}
	go func() {
		if err := e.Advertise(ctx, port); err != nil {
			log.Printf("moku-sync: not advertised on the local network: %v", err)
		}
	}()
	return nil
}
