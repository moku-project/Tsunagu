// Package netguard keeps URL proxies away from local and private addresses,
// checked on every dial since DNS and redirects can point anywhere.
package netguard

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"syscall"
	"time"
)

var ErrBlocked = errors.New("destination address not allowed")

// Internal reports whether ip belongs to this machine or a private network.
func Internal(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

// proxyPorts are the ports of HTTP proxies set in the environment.
var proxyPorts = func() map[string]bool {
	ports := map[string]bool{}
	for _, k := range []string{"HTTP_PROXY", "http_proxy", "HTTPS_PROXY", "https_proxy", "ALL_PROXY", "all_proxy"} {
		if u, err := url.Parse(os.Getenv(k)); err == nil && u.Port() != "" {
			ports[u.Port()] = true
		}
	}
	return ports
}()

// Control is a net.Dialer Control hook that refuses internal addresses.
// allowLoopback, when non-nil, admits loopback ports it approves.
func Control(allowLoopback func(port string) bool) func(network, address string, c syscall.RawConn) error {
	return func(_, address string, _ syscall.RawConn) error {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return ErrBlocked
		}
		ip := net.ParseIP(host)
		if ip == nil {
			return ErrBlocked
		}
		if !Internal(ip) || proxyPorts[port] {
			return nil
		}
		if ip.IsLoopback() && allowLoopback != nil && allowLoopback(port) {
			return nil
		}
		return ErrBlocked
	}
}

// Transport is an HTTP transport whose dials pass through Control.
func Transport(allowLoopback func(port string) bool) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
			Control:   Control(allowLoopback),
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   8,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		ExpectContinueTimeout: time.Second,
	}
}
