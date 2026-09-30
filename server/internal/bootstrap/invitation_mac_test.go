package bootstrap

import (
	"log/slog"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// bootstrap derives the invitation MAC from the signing key for
// workspace.InvitationMACPurpose (M3 design 3.8, 6.6). Its token of the id
// below is the known answer computed by hand from the design's derivation,
// HKDF info "nerve workspace-invitation mac v1" (spec P3, appendix): the
// purpose is part of the key, and a purpose other than the design's would
// void every link made before it.
func TestTheInvitationMACIsTheDesigns(t *testing.T) {
	keys, err := identity.LoadKeys([]byte(testKeyPEM), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	mac, err := keys.MAC(workspace.InvitationMACPurpose)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	if got := domain.FormatToken(mac.Tag(domain.InvitationMessage(id))); got != "nrv_inv_kvqyKBh-bANMT6JAIYzolA" {
		t.Errorf("the token of %s = %s, want nrv_inv_kvqyKBh-bANMT6JAIYzolA", id, got)
	}
}
