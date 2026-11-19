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
package network

import (
	"context"
	"net"
	"net/http"
	"time"
)

// fallbackDNS are asked when the system names no DNS server, in this order
// (Go's resolver tries the loopback entries one after the other).
var fallbackDNS = []string{"1.1.1.1:53", "8.8.8.8:53"}

// dialFunc is net.Dialer.DialContext's shape.
type dialFunc func(ctx context.Context, network, address string) (net.Conn, error)

// New returns an HTTP client whose requests end after timeout.
func New(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	dialer.Resolver = &net.Resolver{PreferGo: true, Dial: fallbackDial(dialer.DialContext, fallbackDNS)}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			DialContext:         ipv4Only(dialer.DialContext),
			TLSHandshakeTimeout: 15 * time.Second,
		},
	}
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
