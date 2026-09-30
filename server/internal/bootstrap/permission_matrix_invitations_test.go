package bootstrap

import (
	"log/slog"
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// The invitations' part of the workspace module's rows (M3 design 9.2).

var cellInvitationNotFound = cell{http.StatusNotFound, "workspace.invitation_not_found"}

// ofInvitation are the cells of a row that names an invitation: the answers
// of the workspace's admin, member and guest, and
// workspace.invitation_not_found for the callers the workspace is not
// visible to.
func ofInvitation(admin, member, guest cell) map[caller]cell {
	return map[caller]cell{callerAdmin: admin, callerMember: member, callerGuest: guest,
		callerNever: cellInvitationNotFound, callerRemoved: cellInvitationNotFound, callerDeleted: cellInvitationNotFound}
}

// toInvitation is the request of a row whose callers each send method and
// body to the invitation of email seeded in the workspace their column
// targets.
func toInvitation(method, email, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return method, "/api/v0/workspace-invitations/" + s.invitation(workspaceOf(c), email).String(), body
	}
}

// promotesTheNewcomer: the admin's answer is acme's invitation of
// newcomer@example.com, now as an admin, with the token of its id.
func promotesTheNewcomer(t *testing.T, c caller, answer string) {
	var inv struct {
		ID          uuid.UUID `json:"id"`
		WorkspaceID uuid.UUID `json:"workspace_id"`
		Email       string    `json:"email"`
		Role        int       `json:"role"`
		Token       string    `json:"token"`
	}
	decodeAnswer(t, answer, &inv)
	if inv.Email != "newcomer@example.com" || inv.Role != 20 || inv.Token != invitationToken(t, inv.ID) {
		t.Errorf("%s's change answers %+v, want newcomer@example.com's invitation as an admin, with its token", c, inv)
	}
}

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

// inviting is the body of a creation that invites email as a guest.
func inviting(email string) string {
	return `{"invitations":[{"email":"` + email + `","role":5}]}`
}

// invitesTheInvitee: the admin's answer is the one new invitation, of
// invitee@example.com as a guest, with the token of its id.
func invitesTheInvitee(t *testing.T, c caller, answer string) {
	var list struct {
		Data []struct {
			ID    uuid.UUID `json:"id"`
			Email string    `json:"email"`
			Role  int       `json:"role"`
			Token string    `json:"token"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	if len(list.Data) != 1 || list.Data[0].Email != "invitee@example.com" || list.Data[0].Role != 5 ||
		list.Data[0].Token != invitationToken(t, list.Data[0].ID) {
		t.Errorf("%s's invitation answers %+v, want invitee@example.com's, as a guest, with its token", c, list.Data)
	}
}
