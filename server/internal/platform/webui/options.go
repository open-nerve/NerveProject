package webui

import (
	"errors"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

// Options contains only the additional origin needed by file transfers.
type Options struct{ StorageOrigin string }

// validate accepts one origin, never a CSP expression or a URL path. The
// composition supplies the storage adapter's resolved origin (including
// virtual-host buckets), not the configured endpoint or an arbitrary list.
func (o Options) validate() error {
	if o.StorageOrigin == "" {
		return nil
	}
	u, err := url.Parse(o.StorageOrigin)
	invalid := errors.New("web storage origin must be one exact HTTP(S) DNS or IPv4 origin")
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.Path != "" || u.RawPath != "" || u.Opaque != "" || u.Scheme+"://"+u.Host != o.StorageOrigin || strings.ContainsAny(o.StorageOrigin, "?#") {
		return invalid
	}
	if strings.HasSuffix(u.Host, ":") {
		return invalid
	}
	if port := u.Port(); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return invalid
		}
	}
	host := u.Hostname()
	if ip, err := netip.ParseAddr(host); err == nil {
		if !ip.Is4() {
			return invalid
		}
		return nil
	}
	if len(host) > 253 {
		return invalid
	}
	for _, label := range strings.Split(strings.TrimSuffix(host, "."), ".") {
		if len(label) < 1 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return invalid
		}
		for _, c := range label {
			switch {
			case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-':
			default:
				return invalid
			}
		}
	}
	return nil
}
