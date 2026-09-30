package app_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func (f *invitationsFixture) get() *app.GetWorkspaceInvitation {
	return app.NewGetWorkspaceInvitation(f.invitations, f.mac)
}

// The link shows the invitation's role, whether it was declined, and its
// workspace's name and slug: the token checked by the MAC, then the
// invitation read, without a transaction and without a caller. Two
// workspaces; a pending and a declined invitation.
func TestGetWorkspaceInvitationShowsTheLinksInvitation(t *testing.T) {
	for _, tt := range []struct {
		inv  domain.Invitation
		want domain.InvitationPreview
	}{
		{carolToAcme, domain.InvitationPreview{ID: carolToAcme.ID, Role: shared.RoleMember, WorkspaceName: "Acme", WorkspaceSlug: "acme"}},
		{daveToAcme, domain.InvitationPreview{ID: daveToAcme.ID, Role: shared.RoleGuest, Declined: true, WorkspaceName: "Acme", WorkspaceSlug: "acme"}},
		{erinToBeta, domain.InvitationPreview{ID: erinToBeta.ID, Role: shared.RoleAdmin, WorkspaceName: "Beta", WorkspaceSlug: "beta"}},
	} {
		f := newInvitations()
		got, err := f.get().Execute(context.Background(), tt.inv.ID, tokenOf(f.mac, tt.inv.ID))
		if err != nil || got != tt.want {
			t.Errorf("%s: Execute() = %+v, %v; want %+v", tt.inv.Email, got, err, tt.want)
		}
		if want := []string{"Verify " + tt.inv.ID.String(), "InvitationPreview " + tt.inv.ID.String() + " outside tx"}; !slices.Equal(f.log.calls, want) {
			t.Errorf("%s: calls = %q, want %q", tt.inv.Email, f.log.calls, want)
		}
	}
}

// Every token that is not the invitation's is workspace.invitation_not_found
// and reads nothing: another invitation's, one under another key, one with
// a character changed, and every string not of the format. A token not of
// the format is not even given to the MAC.
func TestGetWorkspaceInvitationRefusesAWrongTokenBeforeReading(t *testing.T) {
	f := newInvitations()
	right := tokenOf(f.mac, carolToAcme.ID)
	// A character of the tag's middle, whose six bits are all the tag's
	// (the last character's low bits are padding the format refuses).
	changed := []byte(right)
	changed[len("nrv_inv_")+10] = 'A'
	if string(changed) == right {
		changed[len("nrv_inv_")+10] = 'B'
	}
	tests := []struct {
		name, token string
		verified    bool
	}{
		{"another invitation's", tokenOf(f.mac, daveToAcme.ID), true},
		{"another workspace's invitation's", tokenOf(f.mac, erinToBeta.ID), true},
		{"under another key", tokenOf(fakeMAC{log: f.log, key: "another instance's key"}, carolToAcme.ID), true},
		{"a character changed", string(changed), true},
		{"empty", "", false},
		{"without its prefix", strings.TrimPrefix(right, "nrv_inv_"), false},
		{"the prefix in upper case", "NRV_INV_" + strings.TrimPrefix(right, "nrv_inv_"), false},
		{"one character short", right[:len(right)-1], false},
		{"one character long", right + "A", false},
		{"with a space after", right + " ", false},
		{"padded", right + "==", false},
		{"of the standard alphabet", strings.NewReplacer("-", "+", "_", "/").Replace(right) + "+", false},
	}
	for _, tt := range tests {
		f := newInvitations()
		got, err := f.get().Execute(context.Background(), carolToAcme.ID, tt.token)
		if !errors.Is(err, domain.ErrInvitationNotFound) || got != (domain.InvitationPreview{}) {
			t.Errorf("%s: Execute() = %+v, %v; want workspace.invitation_not_found", tt.name, got, err)
		}
		var want []string
		if tt.verified {
			want = []string{"Verify " + carolToAcme.ID.String()}
		}
		if !slices.Equal(f.log.calls, want) {
			t.Errorf("%s: calls = %q, want %q", tt.name, f.log.calls, want)
		}
	}
}

// An invitation the store does not find, with the token of its id, is the
// same workspace.invitation_not_found; a failed read is itself, never a
// 404.
func TestGetWorkspaceInvitationRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	stranger := uuid.NewV7()
	for _, tt := range []struct {
		name string
		id   uuid.UUID
		set  func(f *invitationsFixture)
		want error
	}{
		{"no such invitation", stranger, nil, domain.ErrInvitationNotFound},
		{"deleted", carolToAcme.ID, func(f *invitationsFixture) { f.invitations.invitations = []domain.Invitation{daveToAcme} },
			domain.ErrInvitationNotFound},
		{"the read failed", carolToAcme.ID, func(f *invitationsFixture) { f.invitations.readErr = failure }, failure},
	} {
		f := newInvitations()
		if tt.set != nil {
			tt.set(f)
		}
		got, err := f.get().Execute(context.Background(), tt.id, tokenOf(f.mac, tt.id))
		if !errors.Is(err, tt.want) || got != (domain.InvitationPreview{}) || (tt.want == failure && errors.Is(err, domain.ErrInvitationNotFound)) {
			t.Errorf("%s: Execute() = %+v, %v; want %v", tt.name, got, err, tt.want)
		}
		if want := []string{"Verify " + tt.id.String(), "InvitationPreview " + tt.id.String() + " outside tx"}; !slices.Equal(f.log.calls, want) {
			t.Errorf("%s: calls = %q, want %q", tt.name, f.log.calls, want)
		}
	}
}
