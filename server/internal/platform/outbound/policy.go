package outbound

import (
	"fmt"
	"net/netip"
	"slices"
)

type policy struct {
	deny  []netip.Prefix
	allow []netip.Prefix
	local []netip.Addr
}

func newPolicy(allow []netip.Prefix, local []netip.Addr) (policy, error) {
	for _, prefix := range allow {
		if !prefix.IsValid() {
			return policy{}, fmt.Errorf("invalid outbound allow prefix: %s", prefix)
		}
	}
	for _, addr := range local {
		if !addr.IsValid() || addr.Zone() != "" {
			return policy{}, fmt.Errorf("invalid outbound local address: %s", addr)
		}
	}
	return policy{deny: specialRanges(), allow: slices.Clone(allow), local: slices.Clone(local)}, nil
}

func (p policy) check(addr netip.Addr) error {
	return p.checkAddr(addr, true)
}

func (p policy) checkAddr(addr netip.Addr, original bool) error {
	if !addr.IsValid() || addr.Zone() != "" {
		return ErrBlocked
	}
	allowed := false
	if original {
		for _, prefix := range p.allow {
			if prefix.Contains(addr) {
				allowed = true
				break
			}
		}
	}
	if !allowed {
		for _, prefix := range p.deny {
			if prefix.Contains(addr) {
				return ErrBlocked
			}
		}
		if slices.Contains(p.local, addr) {
			return ErrBlocked
		}
	}
	// Check the original outer address before decoding any embedded address.
	// Allow rules apply only to that literal, never to its embedded payloads.
	if addr.Is4In6() {
		return p.checkAddr(addr.Unmap(), false)
	}
	if !addr.Is6() {
		return nil
	}
	bytes := addr.As16()
	switch {
	case netip.MustParsePrefix("64:ff9b::/96").Contains(addr):
		return p.checkAddr(netip.AddrFrom4([4]byte(bytes[12:16])), false)
	case netip.MustParsePrefix("2002::/16").Contains(addr):
		return p.checkAddr(netip.AddrFrom4([4]byte(bytes[2:6])), false)
	case netip.MustParsePrefix("2001::/32").Contains(addr):
		server := netip.AddrFrom4([4]byte(bytes[4:8]))
		if err := p.checkAddr(server, false); err != nil {
			return err
		}
		client := netip.AddrFrom4([4]byte{^bytes[12], ^bytes[13], ^bytes[14], ^bytes[15]})
		return p.checkAddr(client, false)
	}
	// RFC 8215 reserves 64:ff9b:1::/48 without prescribing an encoding.
	// Its entire allocation is denied above; do not guess embedded bit positions.
	return nil
}
