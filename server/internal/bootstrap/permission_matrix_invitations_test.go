package bootstrap

import (
	"log/slog"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// The invitations' part of the workspace module's rows (M3 design 9.2).

// invitationToken is the token of the invitation id under the matrix's
// signing key, as the app wired on it computes it: identity's MAC of the
// workspace module's purpose.
func invitationToken(t *testing.T, id uuid.UUID) string {
	t.Helper()
	keys, err := identity.LoadKeys([]byte(testKeyPEM), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	mac, err := keys.MAC(workspace.InvitationMACPurpose)
	if err != nil {
		t.Fatal(err)
	}
	return workspacedomain.FormatToken(mac.Tag(workspacedomain.InvitationMessage(id)))
}

// listsTheInvitations: the admin's list is acme's invitations, not gone's,
// each with the token of its id.
func listsTheInvitations(t *testing.T, c caller, answer string) {
	var list struct {
		Data []struct {
			ID    uuid.UUID `json:"id"`
			Email string    `json:"email"`
			Token string    `json:"token"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	var got, want []string
	for _, i := range list.Data {
		got = append(got, i.Email)
		if i.Token != invitationToken(t, i.ID) {
			t.Errorf("%s: the invitation of %s has the token %s, want its id's", c, i.Email, i.Token)
		}
	}
	for _, i := range matrixInvitations {
		if i.slug == "acme" {
			want = append(want, i.email)
		}
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("%s lists the invitations of %q, want %q", c, got, want)
	}
}
