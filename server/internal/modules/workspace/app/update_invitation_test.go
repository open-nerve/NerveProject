package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// lockedInvitationCalls are the calls up to the decision on action over
// the invitation inv for user: the read, the workspace's FOR SHARE, the
// invitation's lock, the decision.
func lockedInvitationCalls(user app.AccountState, inv domain.Invitation, action shared.Action) []string {
	return []string{
		"InvitationByID " + inv.ID.String(),
		"ShareWorkspace " + inv.WorkspaceID.String(),
		"LockInvitation " + inv.ID.String(),
		"Authorize " + user.ID.String() + " " + string(action) + " on " + inv.WorkspaceID.String() + "/" + uuid.Nil().String(),
	}
}

func (f *invitationsFixture) update() *app.UpdateWorkspaceInvitation {
	return app.NewUpdateWorkspaceInvitation(f.invitations, f.auth, f.tx, clockAt{at: clockNow}, f.mac)
}

// UpdateWorkspaceInvitation reads the invitation, locks its workspace FOR
// SHARE, locks the invitation, decides, then writes the role, all in one
// transaction (M3 design 3.6); the answer is the row as stored, with its
// token. Two admins, two workspaces.
func TestUpdateWorkspaceInvitationLocksThenDecidesThenWrites(t *testing.T) {
	for _, tt := range []struct {
		user app.AccountState
		inv  domain.Invitation
		role shared.Role
	}{{alice, carolToAcme, shared.RoleGuest}, {alice, carolToAcme, shared.RoleAdmin}, {bob, erinToBeta, shared.RoleMember}} {
		f := newInvitations()
		got, err := f.update().Execute(as(tt.user), tt.inv.ID, tt.role)
		want := tt.inv
		want.Role = tt.role
		if err != nil || !sameInvitations([]domain.InvitationWithToken{got}, []domain.InvitationWithToken{withToken(f.mac, want)}) {
			t.Errorf("%s to %d: Execute() = %+v, %v; want %+v", tt.inv.Email, tt.role, got, err, withToken(f.mac, want))
		}
		wantCalls := append(lockedInvitationCalls(tt.user, tt.inv, domain.ActionInvitationUpdate),
			fmt.Sprintf("UpdateInvitationRole %s to %d by %s at %s", tt.inv.ID, tt.role, tt.user.ID, clockNow.Format(time.RFC3339Nano)))
		if !slices.Equal(f.log.calls, wantCalls) || f.tx.calls != 1 {
			t.Errorf("%s to %d: calls = %q in %d transactions, want %q in one", tt.inv.Email, tt.role, f.log.calls, f.tx.calls, wantCalls)
		}
	}
}

// Each refusal and failure is the answer, and no role is written. The
// role's check comes first. An invitation not there, deleted meanwhile, of
// a workspace deleted meanwhile or not visible is
// workspace.invitation_not_found, and so, before any decision, is one of
// another workspace when read again under the lock (M3 design 3.6
// convention 2), though the caller is that one's admin too; a member's
// forbidden comes before any check of the invitation; then a declined
// invitation, also declined while the lock waited, is
// workspace.invitation_responded. A failure is itself, never a 404.
func TestUpdateWorkspaceInvitationRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	forbidBob := func(f *invitationsFixture) { f.auth.errs = map[grantKey]error{{bob.ID, acme.ID}: shared.Forbidden()} }
	decided := lockedInvitationCalls(alice, carolToAcme, domain.ActionInvitationUpdate)
	stranger := uuid.NewV7()
	tests := []struct {
		name  string
		user  app.AccountState
		id    uuid.UUID
		role  shared.Role
		set   func(f *invitationsFixture)
		want  error
		calls []string
	}{
		{"a role outside the three", alice, carolToAcme.ID, 10, nil,
			shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldInvalidFormat, Message: "is not 5, 15 or 20"}), nil},
		{"no such invitation", alice, stranger, shared.RoleGuest, nil, domain.ErrInvitationNotFound, []string{"InvitationByID " + stranger.String()}},
		{"acme deleted while the lock waited", alice, carolToAcme.ID, shared.RoleGuest,
			func(f *invitationsFixture) { f.invitations.lockErrs = map[string]error{"acme": app.ErrNotFound} }, domain.ErrInvitationNotFound,
			decided[:2]},
		{"deleted while the lock waited", alice, carolToAcme.ID, shared.RoleGuest,
			func(f *invitationsFixture) {
				f.invitations.onLock = func() { f.invitations.invitations = []domain.Invitation{daveToAcme, erinToBeta} }
			},
			domain.ErrInvitationNotFound, decided[:3]},
		{"of beta when read again under the lock", alice, carolToAcme.ID, shared.RoleGuest,
			func(f *invitationsFixture) {
				f.auth.grants[grantKey{alice.ID, beta.ID}] = shared.Grant{WorkspaceRole: shared.RoleAdmin}
				f.invitations.onLock = func() { f.invitations.invitations[0].WorkspaceID = beta.ID }
			},
			domain.ErrInvitationNotFound, decided[:3]},
		{"not visible", carol, carolToAcme.ID, shared.RoleGuest, nil, domain.ErrInvitationNotFound,
			lockedInvitationCalls(carol, carolToAcme, domain.ActionInvitationUpdate)},
		{"a member", bob, carolToAcme.ID, shared.RoleGuest, forbidBob, shared.Forbidden(),
			lockedInvitationCalls(bob, carolToAcme, domain.ActionInvitationUpdate)},
		{"a member, of a declined one", bob, daveToAcme.ID, shared.RoleGuest, forbidBob, shared.Forbidden(),
			lockedInvitationCalls(bob, daveToAcme, domain.ActionInvitationUpdate)},
		{"a declined one", alice, daveToAcme.ID, shared.RoleMember, nil, domain.ErrInvitationResponded,
			lockedInvitationCalls(alice, daveToAcme, domain.ActionInvitationUpdate)},
		{"declined while the lock waited", alice, carolToAcme.ID, shared.RoleGuest,
			func(f *invitationsFixture) {
				f.invitations.onLock = func() { f.invitations.invitations[0].RespondedAt = &declined }
			},
			domain.ErrInvitationResponded, decided},
		{"the read failed", alice, carolToAcme.ID, shared.RoleGuest, func(f *invitationsFixture) { f.invitations.readErr = failure }, failure,
			decided[:1]},
		{"the workspace's lock failed", alice, carolToAcme.ID, shared.RoleGuest,
			func(f *invitationsFixture) { f.invitations.lockErrs = map[string]error{"acme": failure} }, failure, decided[:2]},
		{"the invitation's lock failed", alice, carolToAcme.ID, shared.RoleGuest,
			func(f *invitationsFixture) { f.invitations.onLock = func() { f.invitations.readErr = failure } }, failure, decided[:3]},
		{"the Authorizer failed", alice, carolToAcme.ID, shared.RoleGuest,
			func(f *invitationsFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} }, failure, decided},
		{"the write failed", alice, carolToAcme.ID, shared.RoleGuest, func(f *invitationsFixture) { f.invitations.writeErr = failure }, failure,
			append(decided, fmt.Sprintf("UpdateInvitationRole %s to %d by %s at %s", carolToAcme.ID, shared.RoleGuest, alice.ID,
				clockNow.Format(time.RFC3339Nano)))},
	}
	for _, tt := range tests {
		f := newInvitations()
		if tt.set != nil {
			tt.set(f)
		}
		got, err := f.update().Execute(as(tt.user), tt.id, tt.role)
		if !errors.Is(err, tt.want) || got != (domain.InvitationWithToken{}) {
			t.Errorf("%s: Execute() = %+v, %v; want no invitation and %v", tt.name, got, err, tt.want)
		}
		if tt.want == failure && errors.Is(err, domain.ErrInvitationNotFound) {
			t.Errorf("%s: Execute() = %v, which is also workspace.invitation_not_found", tt.name, err)
		}
		wantTx := 1
		if tt.calls == nil {
			wantTx = 0
		}
		if !slices.Equal(f.log.calls, tt.calls) || f.tx.calls != wantTx {
			t.Errorf("%s: calls = %q in %d transactions, want %q in %d", tt.name, f.log.calls, f.tx.calls, tt.calls, wantTx)
		}
	}
	f := newInvitations()
	if _, err := f.update().Execute(context.Background(), carolToAcme.ID, shared.RoleGuest); !errors.Is(err, shared.Unauthenticated()) ||
		len(f.log.calls) != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and no call", err, f.log.calls)
	}
}
