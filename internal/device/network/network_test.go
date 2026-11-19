package network

import (
	"context"
	"net"
	"testing"
	"time"
)

type call struct{ network, address string }

func spy(calls *[]call) dialFunc {
	return func(_ context.Context, network, address string) (net.Conn, error) {
		*calls = append(*calls, call{network, address})
		return nil, net.ErrClosed
	}
}

func TestTheLoopbackFallbackOfGoIsReplacedByPublicDNSOverIPv4(t *testing.T) {
	var calls []call
	dial := fallbackDial(spy(&calls), []string{"1.1.1.1:53", "8.8.8.8:53"})
	for _, addr := range []string{"127.0.0.1:53", "[::1]:53"} {
		dial(context.Background(), "udp", addr)
	}
	dial(context.Background(), "tcp", "127.0.0.1:53")

	want := []call{{"udp4", "1.1.1.1:53"}, {"udp4", "8.8.8.8:53"}, {"tcp4", "1.1.1.1:53"}}
	for i, w := range want {
		if calls[i] != w {
			t.Errorf("call %d = %v, want %v", i, calls[i], w)
		}
	}
}

func TestADNSServerOfTheSystemIsLeftAlone(t *testing.T) {
	var calls []call
	dial := fallbackDial(spy(&calls), []string{"1.1.1.1:53"})
	dial(context.Background(), "udp", "192.168.0.1:53")
	dial(context.Background(), "udp", "not-an-address")
	if calls[0] != (call{"udp", "192.168.0.1:53"}) || calls[1] != (call{"udp", "not-an-address"}) {
		t.Errorf("%v", calls)
	}
}

func TestWithoutFallbackServersTheLoopbackIsKept(t *testing.T) {
	var calls []call
	fallbackDial(spy(&calls), nil)(context.Background(), "udp", "127.0.0.1:53")
	if calls[0] != (call{"udp", "127.0.0.1:53"}) {
		t.Errorf("%v", calls)
	}
	// a single server answers for both loopback entries
	calls = nil
	one := fallbackDial(spy(&calls), []string{"9.9.9.9:53"})
	one(context.Background(), "udp", "[::1]:53")
	if calls[0].address != "9.9.9.9:53" {
		t.Errorf("%v", calls)
	}
}

func TestConnectionsUseIPv4(t *testing.T) {
	var calls []call
	dial := ipv4Only(spy(&calls))
	dial(context.Background(), "tcp", "api.github.com:443")
	dial(context.Background(), "udp", "x:1")
	if calls[0].network != "tcp4" || calls[1].network != "udp" {
		t.Errorf("%v", calls)
	}
}

func TestNewBuildsAClientWithATimeout(t *testing.T) {
	c := New(42 * time.Second)
	if c.Timeout != 42*time.Second || c.Transport == nil {
		t.Errorf("%+v", c)
	}
}
