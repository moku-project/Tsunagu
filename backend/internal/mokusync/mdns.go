package mokusync

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"golang.org/x/net/dns/dnsmessage"
	"golang.org/x/net/ipv4"
)

// Just enough mDNS (RFC 6762) and DNS-SD (RFC 6763) to advertise _moku._tcp and find other instances.
const (
	serviceName  = "_moku._tcp.local."
	servicesMeta = "_services._dns-sd._udp.local."
	serviceTTL   = 4500
	hostTTL      = 120
	cacheFlush   = 1 << 15
)

var mdnsGroup = &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251), Port: 5353}

type Nearby struct {
	Node    string `json:"node"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

type advert struct {
	instance, host, node, name string
	port                       uint16
}

func newAdvert(node, name string, port int) advert {
	label := strings.TrimSpace(strings.ReplaceAll(name, ".", " "))
	if label == "" {
		label = "Tsunagu"
	}
	label = truncate(label, 63)
	return advert{
		instance: label + "." + serviceName,
		host:     "moku-" + node[:min(12, len(node))] + ".local.",
		node:     node,
		name:     name,
		port:     uint16(port),
	}
}

type record int

const (
	recMeta record = iota
	recPTR
	recSRV
	recTXT
	recA
)

func sameName(a, b string) bool { return strings.EqualFold(a, b) }

// answer builds the reply to a query, or nil when it asks nothing about this instance. A legacy query
// comes from a port other than 5353 and wants a plain unicast DNS reply.
func (ad advert) answer(query []byte, ips [][4]byte, legacy bool) []byte {
	var p dnsmessage.Parser
	h, err := p.Start(query)
	if err != nil || h.Response {
		return nil
	}
	qs, err := p.AllQuestions()
	if err != nil {
		return nil
	}
	want := map[record]bool{}
	for _, q := range qs {
		name, all := q.Name.String(), q.Type == dnsmessage.TypeALL
		switch {
		case sameName(name, servicesMeta) && (q.Type == dnsmessage.TypePTR || all):
			want[recMeta] = true
		case sameName(name, serviceName) && (q.Type == dnsmessage.TypePTR || all):
			want[recPTR] = true
		case sameName(name, ad.instance) && (q.Type == dnsmessage.TypeSRV || all):
			want[recSRV] = true
		case sameName(name, ad.instance) && (q.Type == dnsmessage.TypeTXT || all):
			want[recTXT] = true
		case sameName(name, ad.host) && (q.Type == dnsmessage.TypeA || all):
			want[recA] = true
		}
	}
	if len(want) == 0 {
		return nil
	}
	extra := map[record]bool{}
	if want[recPTR] {
		extra[recSRV], extra[recTXT], extra[recA] = true, true, true
	}
	if want[recSRV] {
		extra[recA] = true
	}
	for r := range want {
		delete(extra, r)
	}
	head := dnsmessage.Header{Response: true, Authoritative: true}
	var questions []dnsmessage.Question
	if legacy {
		head.ID, questions = h.ID, qs
	}
	return ad.message(head, questions, want, extra, ips, -1, legacy)
}

// announcement lists every record unasked; ttl 0 is the goodbye that clears caches.
func (ad advert) announcement(ips [][4]byte, ttl int) []byte {
	all := map[record]bool{recPTR: true, recSRV: true, recTXT: true, recA: true}
	return ad.message(dnsmessage.Header{Response: true, Authoritative: true}, nil, all, nil, ips, ttl, false)
}

func (ad advert) message(head dnsmessage.Header, qs []dnsmessage.Question, answers, extra map[record]bool, ips [][4]byte, ttl int, legacy bool) []byte {
	b := dnsmessage.NewBuilder(make([]byte, 0, 512), head)
	b.EnableCompression()
	if err := b.StartQuestions(); err != nil {
		return nil
	}
	for _, q := range qs {
		if err := b.Question(q); err != nil {
			return nil
		}
	}
	if err := b.StartAnswers(); err != nil {
		return nil
	}
	if err := ad.records(&b, answers, ips, ttl, legacy); err != nil {
		return nil
	}
	if err := b.StartAdditionals(); err != nil {
		return nil
	}
	if err := ad.records(&b, extra, ips, ttl, legacy); err != nil {
		return nil
	}
	msg, err := b.Finish()
	if err != nil {
		return nil
	}
	return msg
}

func (ad advert) records(b *dnsmessage.Builder, set map[record]bool, ips [][4]byte, ttl int, legacy bool) error {
	instance, err := dnsmessage.NewName(ad.instance)
	if err != nil {
		return err
	}
	host, err := dnsmessage.NewName(ad.host)
	if err != nil {
		return err
	}
	service := dnsmessage.MustNewName(serviceName)
	header := func(name dnsmessage.Name, def int, unique bool) dnsmessage.ResourceHeader {
		t := def
		if ttl >= 0 {
			t = ttl
		}
		if legacy {
			t = min(t, 10)
		}
		class := dnsmessage.ClassINET
		// Unique records tell caches to drop older copies; a legacy resolver wouldn't understand the bit.
		if unique && !legacy {
			class |= cacheFlush
		}
		return dnsmessage.ResourceHeader{Name: name, Class: class, TTL: uint32(t)}
	}
	if set[recMeta] {
		if err := b.PTRResource(header(dnsmessage.MustNewName(servicesMeta), serviceTTL, false), dnsmessage.PTRResource{PTR: service}); err != nil {
			return err
		}
	}
	if set[recPTR] {
		if err := b.PTRResource(header(service, serviceTTL, false), dnsmessage.PTRResource{PTR: instance}); err != nil {
			return err
		}
	}
	if set[recSRV] {
		if err := b.SRVResource(header(instance, hostTTL, true), dnsmessage.SRVResource{Port: ad.port, Target: host}); err != nil {
			return err
		}
	}
	if set[recTXT] {
		txt := []string{"v=1", "node=" + ad.node, "name=" + truncate(ad.name, 200)}
		if err := b.TXTResource(header(instance, serviceTTL, true), dnsmessage.TXTResource{TXT: txt}); err != nil {
			return err
		}
	}
	if set[recA] {
		for _, ip := range ips {
			if err := b.AResource(header(host, hostTTL, true), dnsmessage.AResource{A: ip}); err != nil {
				return err
			}
		}
	}
	return nil
}

func truncate(s string, n int) string {
	for len(s) > n {
		_, size := utf8.DecodeLastRuneInString(s)
		s = s[:len(s)-size]
	}
	return s
}

func multicastInterfaces() []net.Interface {
	all, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []net.Interface
	for _, ifi := range all {
		if ifi.Flags&net.FlagUp != 0 && ifi.Flags&net.FlagMulticast != 0 && ifi.Flags&net.FlagLoopback == 0 {
			out = append(out, ifi)
		}
	}
	return out
}

func interfaceIPv4s(ifi *net.Interface) [][4]byte {
	var addrs []net.Addr
	var err error
	if ifi != nil {
		addrs, err = ifi.Addrs()
	} else {
		addrs, err = net.InterfaceAddrs()
	}
	if err != nil {
		return nil
	}
	var out [][4]byte
	for _, a := range addrs {
		n, ok := a.(*net.IPNet)
		if !ok || n.IP.IsLoopback() || n.IP.IsLinkLocalUnicast() {
			continue
		}
		if v4 := n.IP.To4(); v4 != nil {
			out = append(out, [4]byte(v4))
		}
	}
	return out
}

// Advertise answers mDNS queries for this instance's sync port until ctx ends, then says goodbye.
func (e *Engine) Advertise(ctx context.Context, port int) error {
	lc := net.ListenConfig{Control: reuseAddr}
	c, err := lc.ListenPacket(ctx, "udp4", "0.0.0.0:5353")
	if err != nil {
		return err
	}
	pc := ipv4.NewPacketConn(c)
	defer pc.Close()
	_ = pc.SetMulticastTTL(255)
	_ = pc.SetMulticastLoopback(true)
	_ = pc.SetControlMessage(ipv4.FlagInterface, true)

	var mu sync.Mutex
	joined := map[int]net.Interface{}
	join := func() (added []net.Interface) {
		mu.Lock()
		defer mu.Unlock()
		for _, ifi := range multicastInterfaces() {
			if _, ok := joined[ifi.Index]; ok {
				continue
			}
			if pc.JoinGroup(&ifi, mdnsGroup) == nil {
				joined[ifi.Index] = ifi
				added = append(added, ifi)
			}
		}
		return added
	}
	if len(join()) == 0 {
		return errors.New("no network interface takes multicast")
	}
	var current atomic.Pointer[advert]
	ad := newAdvert(e.node, e.Name(), port)
	current.Store(&ad)

	send := func(msg func(ips [][4]byte) []byte, to []net.Interface) {
		for _, ifi := range to {
			if b := msg(interfaceIPv4s(&ifi)); b != nil && pc.SetMulticastInterface(&ifi) == nil {
				_, _ = pc.WriteTo(b, nil, mdnsGroup)
			}
		}
	}
	interfaces := func() []net.Interface {
		mu.Lock()
		defer mu.Unlock()
		out := make([]net.Interface, 0, len(joined))
		for _, ifi := range joined {
			out = append(out, ifi)
		}
		return out
	}

	go func() {
		buf := make([]byte, 9000)
		for {
			n, cm, from, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			src, ok := from.(*net.UDPAddr)
			if !ok {
				continue
			}
			var ifi *net.Interface
			if cm != nil && cm.IfIndex != 0 {
				ifi, _ = net.InterfaceByIndex(cm.IfIndex)
			}
			legacy := src.Port != mdnsGroup.Port
			reply := current.Load().answer(buf[:n], interfaceIPv4s(ifi), legacy)
			if reply == nil {
				continue
			}
			if legacy {
				_, _ = pc.WriteTo(reply, nil, src)
			} else if ifi != nil {
				if pc.SetMulticastInterface(ifi) == nil {
					_, _ = pc.WriteTo(reply, nil, mdnsGroup)
				}
			} else {
				_, _ = pc.WriteTo(reply, nil, mdnsGroup)
			}
		}
	}()

	announce := func(ad advert, to []net.Interface) {
		send(func(ips [][4]byte) []byte { return ad.announcement(ips, -1) }, to)
	}
	// RFC 6762 §8.3: announce at least twice, a second apart.
	announce(ad, interfaces())
	again := time.After(time.Second)
	rejoin := time.NewTicker(30 * time.Second)
	defer rejoin.Stop()
	for {
		select {
		case <-ctx.Done():
			old := *current.Load()
			send(func(ips [][4]byte) []byte { return old.announcement(ips, 0) }, interfaces())
			return nil
		case <-again:
			announce(*current.Load(), interfaces())
		case <-e.renamed.wait():
			old := *current.Load()
			if name := e.Name(); name != old.name {
				next := newAdvert(e.node, name, port)
				current.Store(&next)
				send(func(ips [][4]byte) []byte { return old.announcement(ips, 0) }, interfaces())
				announce(next, interfaces())
				again = time.After(time.Second)
			}
		case <-rejoin.C:
			if added := join(); len(added) > 0 {
				announce(*current.Load(), added)
			}
		}
	}
}

func browseQuery() []byte {
	b := dnsmessage.NewBuilder(nil, dnsmessage.Header{})
	_ = b.StartQuestions()
	_ = b.Question(dnsmessage.Question{Name: dnsmessage.MustNewName(serviceName), Type: dnsmessage.TypePTR, Class: dnsmessage.ClassINET})
	msg, _ := b.Finish()
	return msg
}

type browseState struct {
	instances map[string]bool
	srv       map[string]dnsmessage.SRVResource
	txt       map[string][]string
	a         map[string][][4]byte
}

func newBrowseState() *browseState {
	return &browseState{instances: map[string]bool{}, srv: map[string]dnsmessage.SRVResource{}, txt: map[string][]string{}, a: map[string][][4]byte{}}
}

// add reads every record in a response; answers and additionals carry the same kinds of records.
func (s *browseState) add(msg []byte) {
	var p dnsmessage.Parser
	h, err := p.Start(msg)
	if err != nil || !h.Response {
		return
	}
	if err := p.SkipAllQuestions(); err != nil {
		return
	}
	sections := []struct {
		header func() (dnsmessage.ResourceHeader, error)
		skip   func() error
	}{{p.AnswerHeader, p.SkipAnswer}, {p.AuthorityHeader, p.SkipAuthority}, {p.AdditionalHeader, p.SkipAdditional}}
	for _, sec := range sections {
		for {
			rh, err := sec.header()
			if err != nil {
				break
			}
			name := strings.ToLower(rh.Name.String())
			switch rh.Type {
			case dnsmessage.TypePTR:
				r, err := p.PTRResource()
				if err != nil {
					return
				}
				if sameName(name, serviceName) {
					s.instances[strings.ToLower(r.PTR.String())] = rh.TTL > 0
				}
			case dnsmessage.TypeSRV:
				r, err := p.SRVResource()
				if err != nil {
					return
				}
				s.srv[name] = r
			case dnsmessage.TypeTXT:
				r, err := p.TXTResource()
				if err != nil {
					return
				}
				s.txt[name] = r.TXT
			case dnsmessage.TypeA:
				r, err := p.AResource()
				if err != nil {
					return
				}
				s.a[name] = append(s.a[name], r.A)
			default:
				if err := sec.skip(); err != nil {
					return
				}
			}
		}
	}
}

func (s *browseState) found(self string) []Nearby {
	seen := map[string]bool{}
	var out []Nearby
	for instance, live := range s.instances {
		srv, ok := s.srv[instance]
		if !live || !ok {
			continue
		}
		fields := map[string]string{}
		for _, kv := range s.txt[instance] {
			if k, v, ok := strings.Cut(kv, "="); ok {
				fields[k] = v
			}
		}
		node := fields["node"]
		ips := s.a[strings.ToLower(srv.Target.String())]
		if node == "" || node == self || seen[node] || len(ips) == 0 {
			continue
		}
		seen[node] = true
		ip := net.IP(ips[0][:])
		out = append(out, Nearby{Node: node, Name: fields["name"], Address: fmt.Sprintf("http://%s", net.JoinHostPort(ip.String(), fmt.Sprint(srv.Port)))})
	}
	return out
}

// Browse asks the local network for instances advertising _moku._tcp, listening for wait.
func Browse(ctx context.Context, wait time.Duration, self string) ([]Nearby, error) {
	c, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		return nil, err
	}
	defer c.Close()
	pc := ipv4.NewPacketConn(c)
	_ = pc.SetMulticastTTL(255)
	_ = pc.SetMulticastLoopback(true)
	q := browseQuery()
	query := func() error {
		sent := false
		for _, ifi := range multicastInterfaces() {
			if pc.SetMulticastInterface(&ifi) == nil {
				if _, err := pc.WriteTo(q, nil, mdnsGroup); err == nil {
					sent = true
				}
			}
		}
		if sent {
			return nil
		}
		_, err := c.WriteTo(q, mdnsGroup)
		return err
	}
	if err := query(); err != nil {
		return nil, err
	}
	// Multicast is easily dropped on Wi-Fi, so ask once more partway through.
	retry := time.AfterFunc(wait/3, func() { _ = query() })
	defer retry.Stop()
	deadline := time.Now().Add(wait)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = c.SetReadDeadline(deadline)
	stop := context.AfterFunc(ctx, func() { _ = c.SetReadDeadline(time.Now()) })
	defer stop()
	s := newBrowseState()
	buf := make([]byte, 9000)
	for {
		n, _, err := c.ReadFrom(buf)
		if err != nil {
			break
		}
		s.add(buf[:n])
	}
	return s.found(self), ctx.Err()
}
