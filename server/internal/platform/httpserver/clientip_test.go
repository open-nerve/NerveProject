package httpserver

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func clientsTrusting(logger *slog.Logger, cidrs ...string) *clientIPs {
	c := &clientIPs{logger: logger, v6Prefix: 64}
	for _, s := range cidrs {
		c.trusted = append(c.trusted, netip.MustParsePrefix(s))
	}
	return c
}

func requestFrom(remote string, forwarded ...string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api/v0/instance", nil)
	r.RemoteAddr = remote
	for _, f := range forwarded {
		r.Header.Add("X-Forwarded-For", f)
	}
	return r
}

// The client is the peer, unless the peer is a trusted proxy: then it is the
// first address of X-Forwarded-For, from the right, that is not a trusted
// proxy (M2 design 3.10).
func TestClientIP(t *testing.T) {
	tests := []struct {
		name      string
		remote    string
		forwarded []string
		want      string
	}{
		{"untrusted peer: its forwarding is ignored", "203.0.113.7:5555", []string{"198.51.100.1"}, "203.0.113.7"},
		{"trusted peer without forwarding", "10.0.0.1:5555", nil, "10.0.0.1"},
		{"trusted peer forwards the client", "10.0.0.1:5555", []string{"198.51.100.1"}, "198.51.100.1"},
		{"trusted hops are skipped", "10.0.0.1:5555", []string{"198.51.100.1, 10.0.0.2"}, "198.51.100.1"},
		{"what the client wrote on the left is not believed", "10.0.0.1:5555", []string{"6.6.6.6,198.51.100.1 , 10.0.0.2"}, "198.51.100.1"},
		{"several header lines are one list", "10.0.0.1:5555", []string{"6.6.6.6", "198.51.100.1, 10.0.0.2"}, "198.51.100.1"},
		{"every hop trusted: the leftmost", "10.0.0.1:5555", []string{"10.0.0.3, 10.0.0.2"}, "10.0.0.3"},
		{"a malformed entry ends at the hop that forwarded it", "10.0.0.1:5555", []string{"198.51.100.1, bogus, 10.0.0.2"}, "10.0.0.2"},
		{"an empty entry ends the walk too", "10.0.0.1:5555", []string{"198.51.100.1,"}, "10.0.0.1"},
		{"IPv6 proxy and client", "[fd00::1]:443", []string{"2001:db8::5"}, "2001:db8::5"},
		{"a mapped client is IPv4", "10.0.0.1:5555", []string{"::ffff:198.51.100.1"}, "198.51.100.1"},
		{"a mapped peer is IPv4, and trusted", "[::ffff:10.0.0.1]:80", []string{"198.51.100.1"}, "198.51.100.1"},
		{"the zone is dropped", "[fe80::1%en0]:80", nil, "fe80::1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := clientsTrusting(slog.New(slog.DiscardHandler), "10.0.0.0/8", "fd00::/8")

			if got := c.of(requestFrom(tt.remote, tt.forwarded...)); got != netip.MustParseAddr(tt.want) {
				t.Errorf("client = %v, want %s", got, tt.want)
			}
		})
	}
}

func TestClientIPOfAnUnparsablePeerIsZero(t *testing.T) {
	c := clientsTrusting(slog.New(slog.DiscardHandler))
	if got := c.of(requestFrom("@unix-socket")); got.IsValid() || c.key(got) != "" {
		t.Errorf("client = %v with key %q, want the zero Addr and the empty key", got, c.key(got))
	}
}

// X-Forwarded-For from a peer that is not trusted most likely means a proxy
// missing from server.trusted_proxies: warned once per process, never again.
func TestUntrustedForwardingIsWarnedOnce(t *testing.T) {
	logger, logs := captureLogs(t)
	c := clientsTrusting(logger, "10.0.0.0/8")

	c.of(requestFrom("203.0.113.7:5555", "198.51.100.1"))
	c.of(requestFrom("203.0.113.8:5555", "198.51.100.2"))
	c.of(requestFrom("10.0.0.1:5555", "198.51.100.3"))

	var warnings []map[string]any
	for _, e := range logs() {
		if e["level"] == "WARN" {
			warnings = append(warnings, e)
		}
	}
	if len(warnings) != 1 || warnings[0]["peer"] != "203.0.113.7" {
		t.Errorf("warnings = %v, want one, naming the peer 203.0.113.7", warnings)
	}
}

// The per-IP buckets count an IPv4 client by its address and an IPv6 client
// by its prefix: a host usually holds a whole /64 (M2 design 3.10).
func TestIPKey(t *testing.T) {
	tests := []struct {
		ip     string
		prefix int
		want   string
	}{
		{"198.51.100.1", 64, "198.51.100.1"},
		{"2001:db8:1:2::1", 64, "2001:db8:1:2::/64"},
		{"2001:db8:1:2:ffff:ffff:ffff:ffff", 64, "2001:db8:1:2::/64"},
		{"2001:db8:1:3::1", 64, "2001:db8:1:3::/64"},
		{"2001:db8:1:2::1", 48, "2001:db8:1::/48"},
		{"2001:db8:1:2::1", 128, "2001:db8:1:2::1/128"},
	}
	for _, tt := range tests {
		c := &clientIPs{v6Prefix: tt.prefix}
		if got := c.key(netip.MustParseAddr(tt.ip)); got != tt.want {
			t.Errorf("key(%s) with /%d = %q, want %q", tt.ip, tt.prefix, got, tt.want)
		}
	}
}

// The request meta middleware puts both in the context: the full address
// for logs and sessions, the key for the buckets.
func TestRequestMetaCarriesTheClientAndItsKey(t *testing.T) {
	auth := &fakeAuth{}
	api := newTestAPI(auth, slog.New(slog.DiscardHandler))
	router, _ := mount(t, api, slog.New(slog.DiscardHandler))
	req := post("/api/v0/things", "tok", `{"name":"a"}`)
	req.RemoteAddr = "[2001:db8:1:2::7]:443"

	serve(router, req)

	meta := RequestMetaFrom(auth.ctx)
	if meta.ClientIP != netip.MustParseAddr("2001:db8:1:2::7") || meta.IPKey != "2001:db8:1:2::/64" {
		t.Errorf("meta = %+v, want the full address and its /64", meta)
	}
}
