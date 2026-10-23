package netguard

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestControl(t *testing.T) {
	ephemeral := func(port string) bool { return port == "45221" }
	cases := map[string]bool{
		"93.184.216.34:443":     true,
		"127.0.0.1:45221":       true, // approved loopback port
		"127.0.0.1:6007":        false,
		"10.0.0.5:80":           false,
		"169.254.169.254:80":    false,
		"[::1]:8080":            false,
		"[fe80::1]:80":          false,
		"0.0.0.0:80":            false,
		"192.168.1.10:9000":     false,
		"[2606:4700::1111]:443": true,
	}
	ctl := Control(ephemeral)
	for addr, want := range cases {
		err := ctl("tcp", addr, nil)
		if got := err == nil; got != want {
			t.Errorf("%s: allowed=%v, want %v", addr, got, want)
		}
	}
}

// A public URL that redirects to an internal one must not be followed there.
func TestTransportBlocksInternalTargets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	client := &http.Client{Transport: Transport(nil)}
	_, err := client.Get(srv.URL)
	if !errors.Is(err, ErrBlocked) {
		t.Fatalf("loopback test server reached: %v", err)
	}
}
