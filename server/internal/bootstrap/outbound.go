package bootstrap

import (
	"fmt"
	"net/netip"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/outbound"
)

// newOutbound constructs the client's fixed address policy from validated
// configuration. Parsing here also rejects invalid directly supplied values.
func newOutbound(cfg config.Config, version string) (*outbound.Client, error) {
	prefixes := make([]netip.Prefix, len(cfg.Outbound.AllowCIDRs))
	for i, value := range cfg.Outbound.AllowCIDRs {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return nil, fmt.Errorf("parse outbound.allow_cidrs entry %d: %w", i+1, err)
		}
		prefixes[i] = prefix
	}
	return outbound.New(outbound.Options{AllowCIDRs: prefixes, Version: version})
}
