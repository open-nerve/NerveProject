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

// newRemoveMember is RemoveWorkspaceMember over membersFixture's fakes: in
// acme alice is the admin, bob a member, carol's membership has ended.
func newRemoveMember() (*app.RemoveWorkspaceMember, *membersFixture, *fakeTx) {
	f := newMembers()
	tx := &fakeTx{}
	return app.NewRemoveWorkspaceMember(f.workspaces, f.profiles, f.projects, f.auth, tx, clockAt{at: clockNow}), f, tx
}

// removalCalls are the calls up to the decision on removing the membership
// m for user: lockedMemberCalls, deciding workspace_member.remove.
func removalCalls(user app.AccountState, m domain.Membership) []string {
	calls := lockedMemberCalls(user, m)
	calls[len(calls)-1] = "Authorize " + user.ID.String() + " workspace_member.remove on " + m.WorkspaceID.String() + "/" + uuid.Nil().String()
	return calls
}

// endingCalls are the calls of the ending of user's membership of
// workspace by by at the clock's time (membershipEnd): his address read,
// the pending invitation to it deleted, his membership ended, then his
// project memberships, in that order.
func endingCalls(workspace uuid.UUID, user app.AccountState, by uuid.UUID) []string {
	at := clockNow.Format(time.RFC3339Nano)
	return []string{
		fmt.Sprintf("PublicProfiles %v", []uuid.UUID{user.ID}),
		fmt.Sprintf("DeletePendingInvitations %s %s by %s at %s", workspace, user.Email, by, at),
		fmt.Sprintf("EndMember %s %s by %s at %s", workspace, user.ID, by, at),
		fmt.Sprintf("EndMemberships %v %s by %s at %s", []uuid.UUID{workspace}, user.ID, by, at),
	}
}

// RemoveWorkspaceMember reads the membership, locks its workspace FOR NO
// KEY UPDATE, reads it again, decides, then ends it: the pending
// invitation to the member's address, read through MemberProfiles, his
// membership, his memberships of acme's projects, each by alice at the
// clock's one time, all in one transaction (M3 design 3.6, 3.8). bob's
// membership is ended.
func TestRemoveWorkspaceMemberLocksThenDecidesThenEnds(t *testing.T) {
	uc, f, tx := newRemoveMember()

	err := uc.Execute(as(alice), bobInAcme.ID)

	want := slices.Concat(removalCalls(alice, bobInAcme), endingCalls(acme.ID, bob, alice.ID))
	if err != nil || !slices.Equal(f.log.calls, want) || tx.calls != 1 {
		t.Errorf("Execute() = %v, calls\n%q\nin %d transactions; want nil,\n%q\nin one", err, f.log.calls, tx.calls, want)
	}
	if m, err := f.workspaces.MemberByID(context.Background(), bobInAcme.ID); err != nil || m.IsActive {
		t.Errorf("bob's membership of acme after the removal: %+v, %v; want it ended", m, err)
	}
}

// Each refusal is the answer, and nothing is ended. A membership that is
// not there or deleted meanwhile, of a workspace not there, deleted
// meanwhile or not visible is workspace.member_not_found, and so, before
// any decision, is one of another workspace when read again under the lock
// (M3 design 3.6 convention 2), though the caller is that one's admin too;
// a member's forbidden comes before any check of the target, so he learns
// nothing about it; then, in updateWorkspaceMember's order, an ended
// membership is workspace.member_not_found, also when it is the caller's
// own, and the caller's own active one workspace.own_membership; a failure
// is never a 404. The clock logs its reads among the calls: no refusal
// reads it.
func TestRemoveWorkspaceMemberRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	forbidBob := func(f *membersFixture) { f.auth.errs = map[grantKey]error{{bob.ID, acme.ID}: shared.Forbidden()} }
	stranger := domain.Membership{ID: uuid.NewV7(), WorkspaceID: uuid.NewV7(), MemberID: bob.ID, Role: shared.RoleMember, IsActive: true}
	decided := removalCalls(alice, bobInAcme)
	tests := []struct {
		name  string
		user  app.AccountState
		id    uuid.UUID
		set   func(f *membersFixture)
		want  error
		calls []string
	}{
		{"no such membership", alice, stranger.ID, nil, domain.ErrMemberNotFound, []string{"MemberByID " + stranger.ID.String()}},
		{"no such workspace", alice, stranger.ID,
			func(f *membersFixture) {
				f.workspaces.memberships[stranger.WorkspaceID] = []domain.Membership{stranger}
			},
			domain.ErrMemberNotFound, []string{"MemberByID " + stranger.ID.String(), "LockWorkspace " + stranger.WorkspaceID.String()}},
		{"acme deleted while the lock waited", alice, bobInAcme.ID,
			func(f *membersFixture) { f.workspaces.lockErrs = map[string]error{"acme": app.ErrNotFound} }, domain.ErrMemberNotFound, decided[:2]},
		{"deleted while the lock waited", alice, bobInAcme.ID,
			func(f *membersFixture) {
				f.workspaces.onLock = func() { f.workspaces.memberships[acme.ID] = []domain.Membership{aliceInAcme, carolInAcme} }
			},
			domain.ErrMemberNotFound, decided[:3]},
		{"of beta when read again under the lock", alice, bobInAcme.ID,
			func(f *membersFixture) {
				f.auth.grants[grantKey{alice.ID, beta.ID}] = shared.Grant{WorkspaceRole: shared.RoleAdmin}
				f.workspaces.onLock = func() { f.workspaces.memberships[acme.ID][1].WorkspaceID = beta.ID }
			},
			domain.ErrMemberNotFound, decided[:3]},
		{"not visible", carol, bobInAcme.ID, nil, domain.ErrMemberNotFound, removalCalls(carol, bobInAcme)},
		{"a member", bob, aliceInAcme.ID, forbidBob, shared.Forbidden(), removalCalls(bob, aliceInAcme)},
		{"a member, of an ended membership", bob, carolInAcme.ID, forbidBob, shared.Forbidden(), removalCalls(bob, carolInAcme)},
		{"a member, of his own", bob, bobInAcme.ID, forbidBob, shared.Forbidden(), removalCalls(bob, bobInAcme)},
		{"an ended membership", alice, carolInAcme.ID, nil, domain.ErrMemberNotFound, removalCalls(alice, carolInAcme)},
		{"ended while the lock waited", alice, bobInAcme.ID,
			func(f *membersFixture) {
				f.workspaces.onLock = func() { f.workspaces.memberships[acme.ID][1].IsActive = false }
			},
			domain.ErrMemberNotFound, decided},
		{"his own", alice, aliceInAcme.ID, nil, domain.ErrOwnMembership, removalCalls(alice, aliceInAcme)},
		{"his own, ended", alice, aliceInAcme.ID,
			func(f *membersFixture) { f.workspaces.memberships[acme.ID][0].IsActive = false }, domain.ErrMemberNotFound,
			removalCalls(alice, aliceInAcme)},
		{"the read failed", alice, bobInAcme.ID, func(f *membersFixture) { f.workspaces.membersErr = failure }, failure,
			[]string{"MemberByID " + bobInAcme.ID.String()}},
		{"the lock failed", alice, bobInAcme.ID, func(f *membersFixture) { f.workspaces.lockErrs = map[string]error{"acme": failure} },
			failure, decided[:2]},
		{"the read under the lock failed", alice, bobInAcme.ID,
			func(f *membersFixture) { f.workspaces.onLock = func() { f.workspaces.membersErr = failure } }, failure, decided[:3]},
		{"the Authorizer failed", alice, bobInAcme.ID,
			func(f *membersFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} }, failure, decided},
	}
	for _, tt := range tests {
		f := newMembers()
		tx := &fakeTx{}
		uc := app.NewRemoveWorkspaceMember(f.workspaces, f.profiles, f.projects, f.auth, tx, clockAt{clockNow, f.log})
		if tt.set != nil {
			tt.set(f)
		}
		err := uc.Execute(as(tt.user), tt.id)
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: Execute() = %v; want %v", tt.name, err, tt.want)
		}
		// The problem the API answers is the first *shared.Error in the
		// chain: the refusal wanted, or none for a failure (a 500).
		var se *shared.Error
		switch {
		case tt.want == failure && errors.As(err, &se):
			t.Errorf("%s: Execute() = %v, which is also %s", tt.name, err, se.Code)
		case tt.want != failure && (!errors.As(err, &se) || !se.Is(tt.want)):
			t.Errorf("%s: Execute() = %v, answered as another problem; want %v", tt.name, err, tt.want)
		}
		if !slices.Equal(f.log.calls, tt.calls) || tx.calls != 1 {
			t.Errorf("%s: calls = %q in %d transactions, want %q in one", tt.name, f.log.calls, tx.calls, tt.calls)
		}
	}
	uc, f, tx := newRemoveMember()
	if err := uc.Execute(context.Background(), bobInAcme.ID); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 || tx.calls != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q in %d transactions; want 401 unauthorized and no call", err, f.log.calls, tx.calls)
	}
}

// A failed read of the member's address, a member without an account or
// another account's profile answered for his (whose pending invitations
// would go), a failed step of the ending, the projects' refusal of the only
// active admin of a project with other active members, and a refused
// commit each fail the transaction, which the database then rolls back,
// the steps before with it: the answer is the error as it came, and
// project.sole_admin is itself, the 409 of the contract (M3 design 3.7 rule
// 2); the calls are the removal's own, each once and in the transaction,
// up to the failing one: nothing runs after it, and it is not tried again.
func TestRemoveWorkspaceMemberFailsWithinTheTransaction(t *testing.T) {
	failure := errors.New("connection reset")
	soleAdmin := shared.NewError(shared.KindConflict, "project.sole_admin", "Ending the membership would leave a project without an admin.")
	calls := slices.Concat(removalCalls(alice, bobInAcme), endingCalls(acme.ID, bob, alice.ID))
	decided := len(removalCalls(alice, bobInAcme))
	tests := []struct {
		name  string
		set   func(f *membersFixture, tx *fakeTx)
		want  error    // the error injected; nil for the use case's own, which is no *shared.Error
		calls []string // up to the failing call, the last one
	}{
		{"the address", func(f *membersFixture, _ *fakeTx) { f.profiles.err = failure }, failure, calls[:decided+1]},
		{"a member without an account", func(f *membersFixture, _ *fakeTx) { f.profiles.profiles = profiles[:2] }, nil, calls[:decided+1]},
		{"another account's profile for his", func(f *membersFixture, _ *fakeTx) { f.profiles.slipped = profiles[:1] }, nil,
			calls[:decided+1]},
		{"the invitations", func(f *membersFixture, _ *fakeTx) {
			f.workspaces.endErrs = map[string]error{"DeletePendingInvitations": failure}
		},
			failure, calls[:decided+2]},
		{"the membership", func(f *membersFixture, _ *fakeTx) { f.workspaces.endErrs = map[string]error{"EndMember": failure} }, failure,
			calls[:decided+3]},
		{"the projects' step", func(f *membersFixture, _ *fakeTx) { f.projects.errs = map[string]error{"EndMemberships": failure} }, failure,
			calls},
		{"the only admin of a project", func(f *membersFixture, _ *fakeTx) { f.projects.errs = map[string]error{"EndMemberships": soleAdmin} },
			soleAdmin, calls},
		{"the commit", func(_ *membersFixture, tx *fakeTx) { tx.commitErr = failure }, failure, calls},
	}
	for _, tt := range tests {
		uc, f, tx := newRemoveMember()
		tt.set(f, tx)
		err := uc.Execute(as(alice), bobInAcme.ID)
		var se *shared.Error
		switch {
		case tt.want == nil && (err == nil || errors.As(err, &se)):
			t.Errorf("%s failing: Execute() = %v; want an error that is no *shared.Error", tt.name, err)
		case tt.want != nil && !errors.Is(err, tt.want):
			t.Errorf("%s failing: Execute() = %v, want %v", tt.name, err, tt.want)
		}
		// The problem the API answers is the first *shared.Error in the
		// chain: the one injected, or none for a failure (a 500).
		if tt.want != nil && errors.As(err, &se) && error(se) != tt.want {
			t.Errorf("%s failing: Execute() = %v, answered as %s; want %v and no other problem", tt.name, err, se.Code, tt.want)
		}
		if !slices.Equal(f.log.calls, tt.calls) || tx.calls != 1 {
			t.Errorf("%s failing: calls\n%q\nin %d transactions; want\n%q\nin one", tt.name, f.log.calls, tx.calls, tt.calls)
		}
	}
}
