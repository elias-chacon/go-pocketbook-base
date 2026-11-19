// Package network is the reader's network as an app sees it: an HTTP client that
// works on the device's Linux, for any app of the ecosystem (the Launcher's
// catalog is the first user). It belongs to the device context because what it
// works around is a trait of the reader, not of an app.
//
// The reader's Linux has no /etc/resolv.conf and no IPv6, so Go's own resolver
// falls back to asking [::1]:53 and 127.0.0.1:53, which fail ("socket: address
// family not supported by protocol"). The client keeps whatever DNS servers the
// system does configure, and only when it configures none (the resolver then
// points at the loopback) asks public DNS servers over IPv4. Connections use IPv4.
//
// The reader also does not keep Wi-Fi connected for an app: before a request the
// client asks the system to connect (inkview.NetConnect), at most once in a while.
// Without it every request fails with "network is unreachable".
package network

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"go-pocketbook-base/internal/device/inkview"
)

// connectedFor is how long a successful connection is trusted before the system
// is asked again (asking when already connected is cheap, but not free).
const connectedFor = 30 * time.Second

// fallbackDNS are asked when the system names no DNS server, in this order
// (Go's resolver tries the loopback entries one after the other).
var fallbackDNS = []string{"1.1.1.1:53", "8.8.8.8:53"}

// dialFunc is net.Dialer.DialContext's shape.
type dialFunc func(ctx context.Context, network, address string) (net.Conn, error)

// Link brings the reader's connection up (Wi-Fi); it returns nil when it is up.
type Link func() error

// New returns an HTTP client whose requests end after timeout and that connects
// the reader first (inkview.NetConnect).
func New(timeout time.Duration) *http.Client { return NewWithLink(timeout, inkview.NetConnect) }

// NewWithLink is New with the connection supplied (tests).
func NewWithLink(timeout time.Duration, link Link) *http.Client {
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	dialer.Resolver = &net.Resolver{PreferGo: true, Dial: fallbackDial(dialer.DialContext, fallbackDNS)}
	return &http.Client{
		Timeout: timeout,
		Transport: &linked{link: link, now: time.Now, base: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			DialContext:         ipv4Only(dialer.DialContext),
			TLSHandshakeTimeout: 15 * time.Second,
		}},
	}
}

// linked is a RoundTripper that makes sure the connection is up first.
type linked struct {
	base http.RoundTripper
	link Link
	now  func() time.Time

	mu    sync.Mutex
	until time.Time // the connection is trusted until then
}

// RoundTrip implements http.RoundTripper.
func (l *linked) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := l.ensure(); err != nil {
		return nil, err
	}
	return l.base.RoundTrip(req)
}

func (l *linked) ensure() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.now().Before(l.until) {
		return nil
	}
	if err := l.link(); err != nil {
		return fmt.Errorf("network: %w", err)
	}
	l.until = l.now().Add(connectedFor)
	return nil
}

// ipv4Only makes TCP connections over IPv4: the reader has no IPv6, and a
// failing AAAA attempt only wastes time.
func ipv4Only(dial dialFunc) dialFunc {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if network == "tcp" {
			network = "tcp4"
		}
		return dial(ctx, network, address)
	}
}

// fallbackDial is the resolver's Dial: a DNS server of the system is used as it
// is, and the loopback entries Go falls back to when the system has none are
// replaced by servers (the first loopback by servers[0], the second by servers[1]).
func fallbackDial(dial dialFunc, servers []string) dialFunc {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return dial(ctx, network, address)
		}
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() && len(servers) > 0 {
			address = servers[0]
			if ip.To4() == nil && len(servers) > 1 { // ::1 is Go's second fallback
				address = servers[1]
			}
			switch network {
			case "udp":
				network = "udp4"
			case "tcp":
				network = "tcp4"
			}
		}
		return dial(ctx, network, address)
	}
}
