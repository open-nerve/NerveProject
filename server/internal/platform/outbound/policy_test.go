package outbound

import (
	"errors"
	"net/netip"
	"testing"
)

func TestPolicyEmbeddedAndLocal(t *testing.T) {
	t.Run("public_and_local", func(t *testing.T) {
		p := mustPolicy(t, nil, []netip.Addr{netip.MustParseAddr("8.8.4.4"), netip.MustParseAddr("2606:4700::1001")})
		for _, literal := range []string{"8.8.8.8", "1.1.1.1", "2001:4860:4860::8888", "2606:4700::1111"} {
			t.Run(literal, func(t *testing.T) { wantAllowed(t, p, netip.MustParseAddr(literal)) })
		}
		for _, literal := range []string{"8.8.4.4", "2606:4700::1001", "2001:4860::8888%en0", "fe80::1%en0"} {
			t.Run(literal, func(t *testing.T) { wantBlocked(t, p, netip.MustParseAddr(literal)) })
		}
		wantBlocked(t, p, netip.Addr{})
	})
	t.Run("literal_allow_only", func(t *testing.T) {
		p := mustPolicy(t, []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}, nil)
		wantAllowed(t, p, netip.MustParseAddr("127.0.0.1"))
		wantBlocked(t, p, netip.MustParseAddr("127.0.0.2"))
		wantBlocked(t, p, netip.MustParseAddr("::ffff:127.0.0.1"))
	})
	t.Run("local_allow", func(t *testing.T) {
		p := mustPolicy(t, []netip.Prefix{netip.MustParsePrefix("8.8.4.4/32")}, []netip.Addr{netip.MustParseAddr("8.8.4.4")})
		wantAllowed(t, p, netip.MustParseAddr("8.8.4.4"))
	})
	t.Run("zone_is_never_allowed", func(t *testing.T) {
		p := mustPolicy(t, []netip.Prefix{netip.MustParsePrefix("::/0")}, nil)
		wantBlocked(t, p, netip.MustParseAddr("2001:4860::8888%en0"))
		wantBlocked(t, p, netip.Addr{})
	})
	for _, row := range []struct {
		name, prefix, public, private, local string
	}{
		{"mapped", "::ffff:0:0/96", "::ffff:8.8.8.8", "::ffff:10.0.0.7", "::ffff:8.8.4.4"},
		{"nat64", "64:ff9b::/96", "64:ff9b::808:808", "64:ff9b::a00:7", "64:ff9b::808:404"},
		{"6to4", "2002::/16", "2002:808:808::1", "2002:a00:7::1", "2002:808:404::1"},
	} {
		t.Run(row.name, func(t *testing.T) {
			plain := mustPolicy(t, nil, nil)
			// Even public embedded payloads cannot bypass their denied outer range.
			wantBlocked(t, plain, netip.MustParseAddr(row.public))
			allow := []netip.Prefix{netip.MustParsePrefix(row.prefix), netip.MustParsePrefix("10.0.0.7/32")}
			p := mustPolicy(t, allow, []netip.Addr{netip.MustParseAddr("8.8.4.4")})
			t.Run("public", func(t *testing.T) { wantAllowed(t, p, netip.MustParseAddr(row.public)) })
			t.Run("private", func(t *testing.T) { wantBlocked(t, p, netip.MustParseAddr(row.private)) })
			t.Run("local", func(t *testing.T) { wantBlocked(t, p, netip.MustParseAddr(row.local)) })
		})
	}
	t.Run("teredo", func(t *testing.T) {
		// Server 8.8.8.8 and complemented client 1.1.1.1, with nonzero flags/port.
		public := netip.MustParseAddr("2001:0:808:808:8000:63bf:fefe:fefe")
		wantBlocked(t, mustPolicy(t, nil, nil), public)
		p := mustPolicy(t, []netip.Prefix{netip.MustParsePrefix("2001::/32"), netip.MustParsePrefix("10.0.0.7/32")},
			[]netip.Addr{netip.MustParseAddr("8.8.4.4")})
		wantAllowed(t, p, public)
		for _, row := range []struct{ name, literal string }{
			{"private_server", "2001:0:a00:7:8000:63bf:fefe:fefe"},
			{"private_client", "2001:0:808:808:8000:63bf:f5ff:fff8"},
			{"local_server", "2001:0:808:404:8000:63bf:fefe:fefe"},
			{"local_client", "2001:0:808:808:8000:63bf:f7f7:fbfb"},
		} {
			t.Run(row.name, func(t *testing.T) { wantBlocked(t, p, netip.MustParseAddr(row.literal)) })
		}
	})
	t.Run("nat64_local_allocation", func(t *testing.T) {
		p := mustPolicy(t, nil, nil)
		// The entire opaque local allocation is denied without guessing a layout.
		wantBlocked(t, p, netip.MustParseAddr("64:ff9b:1:808:8:800::"))
		wantBlocked(t, p, netip.MustParseAddr("64:ff9b:1:abcd::808:808"))
	})
}

func TestPolicyCopiesInputs(t *testing.T) {
	allow := []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}
	local := []netip.Addr{netip.MustParseAddr("8.8.4.4")}
	p := mustPolicy(t, allow, local)
	allow[0] = netip.MustParsePrefix("127.0.0.2/32")
	local[0] = netip.MustParseAddr("8.8.8.8")
	wantAllowed(t, p, netip.MustParseAddr("127.0.0.1"))
	wantBlocked(t, p, netip.MustParseAddr("127.0.0.2"))
	wantBlocked(t, p, netip.MustParseAddr("8.8.4.4"))
	wantAllowed(t, p, netip.MustParseAddr("8.8.8.8"))
}

func TestPolicyRejectsInvalidInputs(t *testing.T) {
	for _, row := range []struct {
		name  string
		allow []netip.Prefix
		local []netip.Addr
	}{
		{"invalid_allow", []netip.Prefix{{}}, nil},
		{"invalid_local", nil, []netip.Addr{{}}},
		{"zoned_local", nil, []netip.Addr{netip.MustParseAddr("fe80::1%en0")}},
	} {
		t.Run(row.name, func(t *testing.T) {
			if _, err := newPolicy(row.allow, row.local); err == nil {
				t.Fatal("invalid policy input accepted")
			}
		})
	}
}

func mustPolicy(t *testing.T, allow []netip.Prefix, local []netip.Addr) policy {
	t.Helper()
	p, err := newPolicy(allow, local)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func wantBlocked(t *testing.T, p policy, addr netip.Addr) {
	t.Helper()
	if err := p.check(addr); !errors.Is(err, ErrBlocked) {
		t.Errorf("check(%s) = %v, want ErrBlocked", addr, err)
	}
}

func wantAllowed(t *testing.T, p policy, addr netip.Addr) {
	t.Helper()
	if err := p.check(addr); err != nil {
		t.Errorf("check(%s) = %v, want allowed", addr, err)
	}
}
