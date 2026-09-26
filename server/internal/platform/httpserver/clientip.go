package httpserver

import (
	"log/slog"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"sync"
)

// clientIPs tells who the client of a request is (M2 design 3.10), and the
// key it counts under in the per-IP rate-limit buckets.
type clientIPs struct {
	logger   *slog.Logger
	trusted  []netip.Prefix // server.trusted_proxies
	v6Prefix int            // ratelimit.ipv6_prefix_len
	// warned makes the warning about X-Forwarded-For from an untrusted peer
	// once per process: bootstrap builds one API.
	warned sync.Once
}

// of returns the client of r: the connection's peer, unless the peer is a
// trusted proxy. Then it is the first address of X-Forwarded-For, from the
// right, that is not a trusted proxy: the proxies append the address they
// received from, so what lies left of the first untrusted one is the
// client's to write. When every address is a trusted proxy the leftmost is
// the client; a malformed entry ends the walk at the trusted hop that
// forwarded it. Addresses lose their zone, and an IPv4-mapped IPv6 address is
// its IPv4 address. The zero Addr when the peer address does not parse.
func (c *clientIPs) of(r *http.Request) netip.Addr {
	client := normalize(peerAddr(r.RemoteAddr))
	forwarded := r.Header.Values("X-Forwarded-For")
	if !c.isTrusted(client) {
		if len(forwarded) > 0 {
			c.warned.Do(func() {
				c.logger.WarnContext(r.Context(), "ignored X-Forwarded-For from a peer that is not a trusted proxy: "+
					"behind a reverse proxy, add its address to server.trusted_proxies, or every client counts as the proxy",
					slog.String("peer", client.String()))
			})
		}
		return client
	}
	hops := strings.Split(strings.Join(forwarded, ","), ",")
	for _, hop := range slices.Backward(hops) {
		addr, err := netip.ParseAddr(strings.TrimSpace(hop))
		if err != nil {
			break
		}
		client = normalize(addr)
		if !c.isTrusted(client) {
			break
		}
	}
	return client
}

func (c *clientIPs) isTrusted(ip netip.Addr) bool {
	return ip.IsValid() && slices.ContainsFunc(c.trusted, func(p netip.Prefix) bool { return p.Contains(ip) })
}

// key is the key of ip in the per-IP buckets: an IPv4 address is itself, an
// IPv6 address its prefix of ratelimit.ipv6_prefix_len bits, since a host
// usually holds a whole /64. "" for the zero Addr.
func (c *clientIPs) key(ip netip.Addr) string {
	switch {
	case !ip.IsValid():
		return ""
	case ip.Is4():
		return ip.String()
	}
	p, _ := ip.Prefix(c.v6Prefix) // NewAPI holds it to 1-128
	return p.String()
}

// peerAddr is the address of RemoteAddr without its port; the zero Addr
// when it does not parse.
func peerAddr(remote string) netip.Addr {
	ap, err := netip.ParseAddrPort(remote)
	if err != nil {
		return netip.Addr{}
	}
	return ap.Addr()
}

func normalize(ip netip.Addr) netip.Addr {
	return ip.Unmap().WithZone("")
}
