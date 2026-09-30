package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	declined = now.Add(time.Hour)
	// acme's invitations: carol's pending, dave's declined; beta's: erin's.
	carolToAcme = domain.Invitation{ID: uuid.NewV7(), WorkspaceID: acme.ID, Email: "carol@corp.com", Role: shared.RoleMember, CreatedAt: now,
		CreatedByID: &alice.ID}
	daveToAcme = domain.Invitation{ID: uuid.NewV7(), WorkspaceID: acme.ID, Email: "dave@corp.com", Role: shared.RoleGuest, RespondedAt: &declined,
		CreatedAt: now}
	erinToBeta = domain.Invitation{ID: uuid.NewV7(), WorkspaceID: beta.ID, Email: "erin@corp.com", Role: shared.RoleAdmin, CreatedAt: now,
		CreatedByID: &bob.ID}
)

// invitationsFixture is the invitations' use cases over fakes sharing one
// log: acme and beta with carolToAcme, daveToAcme and erinToBeta; alice is
// acme's admin, bob acme's member and beta's admin.
type invitationsFixture struct {
	log         *callLog
	tx          *fakeTx
	invitations *fakeInvitations
	auth        *fakeAuthorizer
	mac         fakeMAC
}

func newInvitations() *invitationsFixture {
	log := &callLog{}
	return &invitationsFixture{log: log, tx: &fakeTx{},
		invitations: &fakeInvitations{fakeWorkspaces: &fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme, beta}},
			invitations: []domain.Invitation{carolToAcme, daveToAcme, erinToBeta}},
		auth: &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{
			{alice.ID, acme.ID}: {WorkspaceRole: shared.RoleAdmin},
			{bob.ID, acme.ID}:   {WorkspaceRole: shared.RoleMember},
			{bob.ID, beta.ID}:   {WorkspaceRole: shared.RoleAdmin},
		}}, mac: fakeMAC{log: log, key: "the instance's key"}}
}

// withToken is inv with its token under mac.
func withToken(mac fakeMAC, inv domain.Invitation) domain.InvitationWithToken {
	return domain.InvitationWithToken{Invitation: inv, Token: tokenOf(mac, inv.ID)}
}

// sameInvitations compares two lists, the pointers by value.
func sameInvitations(a, b []domain.InvitationWithToken) bool {
	return slices.EqualFunc(a, b, func(x, y domain.InvitationWithToken) bool {
		rx, ry, cx, cy := x.RespondedAt, y.RespondedAt, x.CreatedByID, y.CreatedByID
		x.RespondedAt, y.RespondedAt, x.CreatedByID, y.CreatedByID = nil, nil, nil, nil
		return x == y && (rx == nil) == (ry == nil) && (rx == nil || rx.Equal(*ry)) && (cx == nil) == (cy == nil) && (cx == nil || *cx == *cy)
	})
}

// The list is the workspace's invitations in the store's order, each with
// the token the MAC gives its id, read without a transaction after the
// decision on workspace_invitation.list for the caller. Two admins, two
// workspaces; acme's two invitations in either order the store answers
// them, so that a use case that sorted them would answer one of the two
// wrong.
func TestListWorkspaceInvitationsListsEachWithItsToken(t *testing.T) {
	for _, tt := range []struct {
		user     app.AccountState
		w        domain.Workspace
		reversed bool // the store holds and answers its invitations in the reverse order
		want     []domain.Invitation
	}{{alice, acme, false, []domain.Invitation{carolToAcme, daveToAcme}}, {alice, acme, true, []domain.Invitation{daveToAcme, carolToAcme}},
		{bob, beta, false, []domain.Invitation{erinToBeta}}} {
		f := newInvitations()
		if tt.reversed {
			slices.Reverse(f.invitations.invitations)
		}
		got, err := app.NewListWorkspaceInvitations(f.invitations, f.auth, f.mac).Execute(as(tt.user), tt.w.Slug)
		var want []domain.InvitationWithToken
		for _, inv := range tt.want {
			want = append(want, withToken(f.mac, inv))
		}
		if err != nil || !sameInvitations(got, want) {
			t.Errorf("%s lists %s: %+v, %v; want %+v", tt.user.Email, tt.w.Slug, got, err, want)
		}
		wantCalls := []string{"WorkspaceBySlug " + tt.w.Slug + " outside tx",
			"Authorize " + tt.user.ID.String() + " workspace_invitation.list on " + tt.w.ID.String() + "/" + uuid.Nil().String() + " outside tx",
			"ListInvitations " + tt.w.ID.String() + " outside tx"}
		if !slices.Equal(f.log.calls, wantCalls) {
			t.Errorf("%s lists %s: calls = %q, want %q", tt.user.Email, tt.w.Slug, f.log.calls, wantCalls)
		}
	}
}

// Another key gives other tokens: the token is the MAC's, of the
// invitation's id alone.
func TestTheTokensAreTheMACs(t *testing.T) {
	f := newInvitations()
	other := fakeMAC{log: f.log, key: "another instance's key"}
	got, err := app.NewListWorkspaceInvitations(f.invitations, f.auth, other).Execute(as(alice), "acme")
	if err != nil || len(got) != 2 || got[0].Token != tokenOf(other, carolToAcme.ID) || got[0].Token == tokenOf(f.mac, carolToAcme.ID) ||
		got[1].Token != tokenOf(other, daveToAcme.ID) {
		t.Errorf("Execute() under another key = %+v, %v; want its tokens", got, err)
	}
}

// Each refusal and failure is the answer, and no invitation is read after
// it: a workspace not there or not visible is workspace.not_found, the
// Authorizer's forbidden comes before the list. A failure is itself, never a
// 404; without a caller it is 401 and nothing is read.
func TestListWorkspaceInvitationsRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	decided := func(user app.AccountState) []string {
		return []string{"WorkspaceBySlug acme outside tx",
			"Authorize " + user.ID.String() + " workspace_invitation.list on " + acme.ID.String() + "/" + uuid.Nil().String() + " outside tx"}
	}
	tests := []struct {
		name  string
		user  app.AccountState
		slug  string
		set   func(f *invitationsFixture)
		want  error
		calls []string
	}{
		{"no such workspace", alice, "nothing", nil, domain.ErrNotFound, []string{"WorkspaceBySlug nothing outside tx"}},
		{"not visible", carol, "acme", nil, domain.ErrNotFound, decided(carol)},
		{"forbidden", bob, "acme", func(f *invitationsFixture) { f.auth.errs = map[grantKey]error{{bob.ID, acme.ID}: shared.Forbidden()} },
			shared.Forbidden(), decided(bob)},
		{"the workspace's read failed", alice, "acme", func(f *invitationsFixture) { f.invitations.slugErrs = map[string]error{"acme": failure} },
			failure, []string{"WorkspaceBySlug acme outside tx"}},
		{"the Authorizer failed", alice, "acme", func(f *invitationsFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} },
			failure, decided(alice)},
		{"the list failed", alice, "acme", func(f *invitationsFixture) { f.invitations.listErr = failure }, failure,
			append(decided(alice), "ListInvitations "+acme.ID.String()+" outside tx")},
	}
	for _, tt := range tests {
		f := newInvitations()
		if tt.set != nil {
			tt.set(f)
		}
		got, err := app.NewListWorkspaceInvitations(f.invitations, f.auth, f.mac).Execute(as(tt.user), tt.slug)
		if !errors.Is(err, tt.want) || got != nil || (tt.want == failure && errors.Is(err, domain.ErrNotFound)) {
			t.Errorf("%s: Execute() = %+v, %v; want %v", tt.name, got, err, tt.want)
		}
		if !slices.Equal(f.log.calls, tt.calls) {
			t.Errorf("%s: calls = %q, want %q", tt.name, f.log.calls, tt.calls)
		}
	}
	f := newInvitations()
	if _, err := app.NewListWorkspaceInvitations(f.invitations, f.auth, f.mac).Execute(context.Background(), "acme"); !errors.Is(err, shared.Unauthenticated()) ||
		len(f.log.calls) != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and nothing", err, f.log.calls)
	}
}
