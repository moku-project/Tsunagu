package mokusync

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

const browseWait = 1500 * time.Millisecond

type NearbyDevice struct {
	Nearby
	Paired bool `json:"paired"`
}

func (e *Engine) Nearby(ctx context.Context) ([]NearbyDevice, error) {
	if e.discover == nil {
		return nil, errors.New("local network discovery is off on this device")
	}
	found, err := e.discover(ctx, e.node)
	if err != nil {
		return nil, err
	}
	paired := map[string]bool{}
	rows, err := e.db.QueryContext(ctx, `SELECT node FROM sync_peers`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var node string
		if err := rows.Scan(&node); err != nil {
			return nil, err
		}
		paired[node] = true
	}
	out := make([]NearbyDevice, 0, len(found))
	for _, n := range found {
		out = append(out, NearbyDevice{Nearby: n, Paired: paired[n.Node]})
	}
	return out, rows.Err()
}

// rediscover moves dialed peers to the address they now advertise, for when DHCP gave them a new one.
func (e *Engine) rediscover(ctx context.Context) (int, error) {
	if e.discover == nil {
		return 0, nil
	}
	found, err := e.discover(ctx, e.node)
	if err != nil {
		return 0, err
	}
	moved := 0
	for _, n := range found {
		var old string
		err := e.db.QueryRowContext(ctx, `SELECT address FROM sync_peers WHERE node = ? AND dial_token != ''`, n.Node).Scan(&old)
		if err != nil || old == n.Address || !replaceable(old) {
			continue
		}
		if _, err := e.db.ExecContext(ctx, `UPDATE sync_peers SET address = ? WHERE node = ?`, n.Address, n.Node); err != nil {
			return moved, err
		}
		moved++
	}
	return moved, nil
}

// replaceable keeps addresses the user chose on purpose, like a hostname or HTTPS proxy, untouched.
func replaceable(address string) bool {
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "http" {
		return false
	}
	host := u.Hostname()
	return net.ParseIP(host) != nil || strings.HasSuffix(strings.ToLower(host), ".local")
}

// Addresses lists where other devices on the LAN can reach this instance's sync port.
func (e *Engine) Addresses() []string {
	port := e.syncPort.Load()
	if port == 0 {
		return []string{}
	}
	out := []string{}
	for _, ip := range interfaceIPv4s(nil) {
		out = append(out, fmt.Sprintf("http://%s", net.JoinHostPort(net.IP(ip[:]).String(), fmt.Sprint(port))))
	}
	return out
}

// BrowseLAN is the discovery Tsunagu uses outside tests.
func BrowseLAN(ctx context.Context, self string) ([]Nearby, error) {
	return Browse(ctx, browseWait, self)
}
