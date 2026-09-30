package bootstrap

import (
	"log/slog"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// testInvitationMAC is the invitation MAC of the tests' signing key, as
// newApp takes it: invitationMAC of the keys loaded from testKeyPEM.
func testInvitationMAC(t testing.TB) identity.MAC {
	t.Helper()
	keys, err := identity.LoadKeys([]byte(testKeyPEM), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	mac, err := invitationMAC(keys)
	if err != nil {
		t.Fatal(err)
	}
	return mac
}

// invitationToken is the token of the invitation id under
// testInvitationMAC: the one an app wired on the tests' key gives it.
func invitationToken(t testing.TB, id uuid.UUID) string {
	t.Helper()
	return domain.FormatToken(testInvitationMAC(t).Tag(domain.InvitationMessage(id)))
}

// invitationMAC, the MAC newApp gives the workspace module, is the
// design's (M3 design 3.8, 6.6): under the tests' key, the token of the id
// below is the known answer computed by hand from the design's derivation,
// HKDF info "nerve workspace-invitation mac v1" (spec P3, appendix). So the
// purpose the composition root asks identity for is pinned here, not only
// the constant: another purpose would void every link made before it.
func TestTheInvitationMACIsTheDesigns(t *testing.T) {
	id := uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	if got := invitationToken(t, id); got != "nrv_inv_kvqyKBh-bANMT6JAIYzolA" {
		t.Errorf("the token of %s = %s, want nrv_inv_kvqyKBh-bANMT6JAIYzolA", id, got)
	}
}
