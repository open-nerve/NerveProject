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

func (f *invitationsFixture) delete() *app.DeleteWorkspaceInvitation {
	return app.NewDeleteWorkspaceInvitation(f.invitations, f.auth, f.tx, clockAt{at: clockNow})
}

// DeleteWorkspaceInvitation reads the invitation, locks its workspace FOR
// SHARE, locks the invitation, decides, then deletes it, all in one
// transaction (M3 design 3.6). A declined invitation is deleted as a
// pending one is (3.8). Two admins, two workspaces; the other invitations
// stay.
func TestDeleteWorkspaceInvitationLocksThenDecidesThenDeletes(t *testing.T) {
	for _, tt := range []struct {
		user app.AccountState
		inv  domain.Invitation
		left []domain.Invitation
	}{
		{alice, carolToAcme, []domain.Invitation{daveToAcme, erinToBeta}},
		{alice, daveToAcme, []domain.Invitation{carolToAcme, erinToBeta}},
		{bob, erinToBeta, []domain.Invitation{carolToAcme, daveToAcme}},
	} {
		f := newInvitations()
		err := f.delete().Execute(as(tt.user), tt.inv.ID)
		wantCalls := append(lockedInvitationCalls(tt.user, tt.inv, domain.ActionInvitationDelete),
			fmt.Sprintf("DeleteInvitation %s by %s at %s", tt.inv.ID, tt.user.ID, clockNow.Format(time.RFC3339Nano)))
		if err != nil || !slices.Equal(f.log.calls, wantCalls) || f.tx.calls != 1 {
			t.Errorf("%s: Execute() = %v, calls = %q in %d transactions; want %q in one", tt.inv.Email, err, f.log.calls, f.tx.calls, wantCalls)
		}
		if !slices.EqualFunc(f.invitations.invitations, tt.left, func(a, b domain.Invitation) bool { return a.ID == b.ID }) {
			t.Errorf("%s: left %+v, want %+v", tt.inv.Email, f.invitations.invitations, tt.left)
		}
	}
}

// Each refusal and failure is the answer, and nothing is deleted: an
// invitation not there, deleted meanwhile, of a workspace deleted meanwhile
// or not visible is workspace.invitation_not_found; a member's forbidden.
// A failure is itself, never a 404.
func TestDeleteWorkspaceInvitationRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	decided := lockedInvitationCalls(alice, carolToAcme, domain.ActionInvitationDelete)
	stranger := uuid.NewV7()
	tests := []struct {
		name  string
		user  app.AccountState
		id    uuid.UUID
		set   func(f *invitationsFixture)
		want  error
		calls []string
	}{
		{"no such invitation", alice, stranger, nil, domain.ErrInvitationNotFound, []string{"InvitationByID " + stranger.String()}},
		{"acme deleted while the lock waited", alice, carolToAcme.ID,
			func(f *invitationsFixture) { f.invitations.lockErrs = map[string]error{"acme": app.ErrNotFound} }, domain.ErrInvitationNotFound,
			decided[:2]},
		{"deleted while the lock waited", alice, carolToAcme.ID,
			func(f *invitationsFixture) {
				f.invitations.onLock = func() { f.invitations.invitations = []domain.Invitation{daveToAcme, erinToBeta} }
			},
			domain.ErrInvitationNotFound, decided[:3]},
		{"not visible", carol, carolToAcme.ID, nil, domain.ErrInvitationNotFound,
			lockedInvitationCalls(carol, carolToAcme, domain.ActionInvitationDelete)},
		{"a member", bob, daveToAcme.ID, func(f *invitationsFixture) { f.auth.errs = map[grantKey]error{{bob.ID, acme.ID}: shared.Forbidden()} },
			shared.Forbidden(), lockedInvitationCalls(bob, daveToAcme, domain.ActionInvitationDelete)},
		{"the read failed", alice, carolToAcme.ID, func(f *invitationsFixture) { f.invitations.readErr = failure }, failure, decided[:1]},
		{"the workspace's lock failed", alice, carolToAcme.ID,
			func(f *invitationsFixture) { f.invitations.lockErrs = map[string]error{"acme": failure} }, failure, decided[:2]},
		{"the invitation's lock failed", alice, carolToAcme.ID,
			func(f *invitationsFixture) { f.invitations.onLock = func() { f.invitations.readErr = failure } }, failure, decided[:3]},
		{"the Authorizer failed", alice, carolToAcme.ID,
			func(f *invitationsFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} }, failure, decided},
		{"the write failed", alice, carolToAcme.ID, func(f *invitationsFixture) { f.invitations.writeErr = failure }, failure,
			append(decided, fmt.Sprintf("DeleteInvitation %s by %s at %s", carolToAcme.ID, alice.ID, clockNow.Format(time.RFC3339Nano)))},
	}
	for _, tt := range tests {
		f := newInvitations()
		if tt.set != nil {
			tt.set(f)
		}
		err := f.delete().Execute(as(tt.user), tt.id)
		if !errors.Is(err, tt.want) || (tt.want == failure && errors.Is(err, domain.ErrInvitationNotFound)) {
			t.Errorf("%s: Execute() = %v, want %v", tt.name, err, tt.want)
		}
		if !slices.Equal(f.log.calls, tt.calls) || f.tx.calls != 1 {
			t.Errorf("%s: calls = %q in %d transactions, want %q in one", tt.name, f.log.calls, f.tx.calls, tt.calls)
		}
	}
	f := newInvitations()
	if err := f.delete().Execute(context.Background(), carolToAcme.ID); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and no call", err, f.log.calls)
	}
}
