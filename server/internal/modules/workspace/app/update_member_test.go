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

// newUpdateMember is UpdateWorkspaceMember over membersFixture's fakes: in
// acme alice is the admin, bob a member, carol's membership has ended.
func newUpdateMember() (*app.UpdateWorkspaceMember, *membersFixture, *fakeTx) {
	f := newMembers()
	tx := &fakeTx{}
	return app.NewUpdateWorkspaceMember(f.workspaces, f.projects, f.profiles, f.auth, tx, clockAt{at: clockNow}), f, tx
}

// lockedMemberCalls are the calls up to the decision on the membership m
// for user: the read, the workspace's lock, the read again, the decision.
func lockedMemberCalls(user app.AccountState, m domain.Membership) []string {
	return []string{
		"MemberByID " + m.ID.String(),
		"LockWorkspace " + m.WorkspaceID.String(),
		"MemberByID " + m.ID.String(),
		"Authorize " + user.ID.String() + " workspace_member.update on " + m.WorkspaceID.String() + "/" + uuid.Nil().String(),
	}
}

// UpdateWorkspaceMember reads the membership, locks its workspace FOR NO
// KEY UPDATE, reads it again, decides, then writes the role and reads the
// member's profile, all in one transaction (M3 design 3.6); a change to
// guest makes him a guest in acme's projects in between, by alice at the
// role's time (M3 design 3.3), a change to another role leaves them. The
// admin sees the address.
func TestUpdateWorkspaceMemberLocksThenDecidesThenWrites(t *testing.T) {
	at := clockNow.Format(time.RFC3339Nano)
	for _, tt := range []struct {
		role    shared.Role
		cascade []string
	}{
		{shared.RoleGuest, []string{fmt.Sprintf("DemoteToGuest %s %s by %s at %s", acme.ID, bob.ID, alice.ID, at)}},
		{shared.RoleMember, nil},
		{shared.RoleAdmin, nil},
	} {
		uc, f, tx := newUpdateMember()
		got, err := uc.Execute(as(alice), bobInAcme.ID, tt.role)
		want := bobInAcme
		want.Role = tt.role
		if err != nil || !sameMembers([]domain.Member{got}, []domain.Member{withUser(want, true)}) {
			t.Errorf("to %d: Execute() = %+v, %v; want %+v", tt.role, got, err, withUser(want, true))
		}
		wantCalls := slices.Concat(lockedMemberCalls(alice, bobInAcme),
			[]string{fmt.Sprintf("UpdateMemberRole %s to %d by %s at %s", bobInAcme.ID, tt.role, alice.ID, at)}, tt.cascade,
			[]string{fmt.Sprintf("PublicProfiles %v", []uuid.UUID{bob.ID})})
		if !slices.Equal(f.log.calls, wantCalls) || tx.calls != 1 {
			t.Errorf("to %d: calls = %q in %d transactions, want %q in one", tt.role, f.log.calls, tx.calls, wantCalls)
		}
	}
}

// The answer's address follows the caller's grant, not the member's role:
// a grant that does not see addresses (a guest's, which the rule never
// gives today) gets none.
func TestUpdateWorkspaceMemberShowsTheAddressByTheCallersRole(t *testing.T) {
	for role, seen := range map[shared.Role]bool{shared.RoleAdmin: true, shared.RoleMember: true, shared.RoleGuest: false} {
		uc, f, _ := newUpdateMember()
		f.auth.grants[grantKey{alice.ID, acme.ID}] = shared.Grant{WorkspaceRole: role}
		got, err := uc.Execute(as(alice), bobInAcme.ID, shared.RoleGuest)
		want := bobInAcme
		want.Role = shared.RoleGuest
		if err != nil || !sameMembers([]domain.Member{got}, []domain.Member{withUser(want, seen)}) {
			t.Errorf("granted as %d: Execute() = %+v, %v; want %+v", role, got, err, withUser(want, seen))
		}
	}
}

// Each refusal and failure is the answer, and no role is written. The
// role's check comes first. A membership that is not there or deleted
// meanwhile, of a workspace not there, deleted meanwhile or not visible is
// workspace.member_not_found, and so, before any decision, is one of
// another workspace when read again under the lock (M3 design 3.6
// convention 2), though the caller is that one's admin too; a member's
// forbidden comes before any check of the target, so he learns nothing
// about it; then an ended membership is workspace.member_not_found and the
// caller's own workspace.own_membership; a failure is never a 404.
func TestUpdateWorkspaceMemberRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	forbidBob := func(f *membersFixture) { f.auth.errs = map[grantKey]error{{bob.ID, acme.ID}: shared.Forbidden()} }
	stranger := domain.Membership{ID: uuid.NewV7(), WorkspaceID: uuid.NewV7(), MemberID: bob.ID, Role: shared.RoleMember, IsActive: true}
	decided := lockedMemberCalls(alice, bobInAcme)
	tests := []struct {
		name  string
		user  app.AccountState
		id    uuid.UUID
		role  shared.Role
		set   func(f *membersFixture)
		want  error
		calls []string
	}{
		{"a role outside the three", alice, bobInAcme.ID, 10, nil,
			shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldInvalidFormat, Message: "is not 5, 15 or 20"}), nil},
		{"no such membership", alice, stranger.ID, shared.RoleGuest, nil, domain.ErrMemberNotFound, []string{"MemberByID " + stranger.ID.String()}},
		{"no such workspace", alice, stranger.ID, shared.RoleGuest,
			func(f *membersFixture) {
				f.workspaces.memberships[stranger.WorkspaceID] = []domain.Membership{stranger}
			},
			domain.ErrMemberNotFound, []string{"MemberByID " + stranger.ID.String(), "LockWorkspace " + stranger.WorkspaceID.String()}},
		{"acme deleted while the lock waited", alice, bobInAcme.ID, shared.RoleGuest,
			func(f *membersFixture) { f.workspaces.lockErrs = map[string]error{"acme": app.ErrNotFound} }, domain.ErrMemberNotFound, decided[:2]},
		{"deleted while the lock waited", alice, bobInAcme.ID, shared.RoleGuest,
			func(f *membersFixture) {
				f.workspaces.onLock = func() { f.workspaces.memberships[acme.ID] = []domain.Membership{aliceInAcme, carolInAcme} }
			},
			domain.ErrMemberNotFound, decided[:3]},
		{"of beta when read again under the lock", alice, bobInAcme.ID, shared.RoleGuest,
			func(f *membersFixture) {
				f.auth.grants[grantKey{alice.ID, beta.ID}] = shared.Grant{WorkspaceRole: shared.RoleAdmin}
				f.workspaces.onLock = func() { f.workspaces.memberships[acme.ID][1].WorkspaceID = beta.ID }
			},
			domain.ErrMemberNotFound, decided[:3]},
		{"not visible", carol, bobInAcme.ID, shared.RoleGuest, nil, domain.ErrMemberNotFound, lockedMemberCalls(carol, bobInAcme)},
		{"a member", bob, aliceInAcme.ID, shared.RoleGuest, forbidBob, shared.Forbidden(), lockedMemberCalls(bob, aliceInAcme)},
		{"a member, of an ended membership", bob, carolInAcme.ID, shared.RoleGuest, forbidBob, shared.Forbidden(), lockedMemberCalls(bob, carolInAcme)},
		{"a member, of his own", bob, bobInAcme.ID, shared.RoleGuest, forbidBob, shared.Forbidden(), lockedMemberCalls(bob, bobInAcme)},
		{"an ended membership", alice, carolInAcme.ID, shared.RoleMember, nil, domain.ErrMemberNotFound, lockedMemberCalls(alice, carolInAcme)},
		{"ended while the lock waited", alice, bobInAcme.ID, shared.RoleGuest,
			func(f *membersFixture) {
				f.workspaces.onLock = func() { f.workspaces.memberships[acme.ID][1].IsActive = false }
			},
			domain.ErrMemberNotFound, decided},
		{"his own", alice, aliceInAcme.ID, shared.RoleMember, nil, domain.ErrOwnMembership, lockedMemberCalls(alice, aliceInAcme)},
		{"the read failed", alice, bobInAcme.ID, shared.RoleGuest, func(f *membersFixture) { f.workspaces.membersErr = failure }, failure,
			[]string{"MemberByID " + bobInAcme.ID.String()}},
		{"the lock failed", alice, bobInAcme.ID, shared.RoleGuest, func(f *membersFixture) { f.workspaces.lockErrs = map[string]error{"acme": failure} },
			failure, decided[:2]},
		{"the read under the lock failed", alice, bobInAcme.ID, shared.RoleGuest,
			func(f *membersFixture) { f.workspaces.onLock = func() { f.workspaces.membersErr = failure } }, failure, decided[:3]},
		{"the Authorizer failed", alice, bobInAcme.ID, shared.RoleGuest,
			func(f *membersFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} }, failure, decided},
	}
	for _, tt := range tests {
		uc, f, tx := newUpdateMember()
		if tt.set != nil {
			tt.set(f)
		}
		got, err := uc.Execute(as(tt.user), tt.id, tt.role)
		if !errors.Is(err, tt.want) || got != (domain.Member{}) {
			t.Errorf("%s: Execute() = %+v, %v; want no member and %v", tt.name, got, err, tt.want)
		}
		if tt.want == failure && errors.Is(err, domain.ErrMemberNotFound) {
			t.Errorf("%s: Execute() = %v, which is also workspace.member_not_found", tt.name, err)
		}
		wantTx := 1
		if tt.calls == nil {
			wantTx = 0
		}
		if !slices.Equal(f.log.calls, tt.calls) || tx.calls != wantTx {
			t.Errorf("%s: calls = %q in %d transactions, want %q in %d", tt.name, f.log.calls, tx.calls, tt.calls, wantTx)
		}
	}
}

// A failed write, a failed step of the projects, a failed read of the
// profile, and a member without an account each fail the transaction, which
// the database then rolls back, the role's change with it:
// the answer is the error, never a member and never a problem of the
// contract (so a 500); a failure is the one injected. The calls are the
// change's own, each once and in the transaction, up to the failing one:
// nothing runs after it, and it is not tried again.
func TestUpdateWorkspaceMemberFailsWithinTheTransaction(t *testing.T) {
	failure := errors.New("connection reset")
	at := clockNow.Format(time.RFC3339Nano)
	calls := slices.Concat(lockedMemberCalls(alice, bobInAcme), []string{
		fmt.Sprintf("UpdateMemberRole %s to %d by %s at %s", bobInAcme.ID, shared.RoleGuest, alice.ID, at),
		fmt.Sprintf("DemoteToGuest %s %s by %s at %s", acme.ID, bob.ID, alice.ID, at),
		fmt.Sprintf("PublicProfiles %v", []uuid.UUID{bob.ID}),
	})
	decided := len(lockedMemberCalls(alice, bobInAcme))
	tests := []struct {
		name  string
		set   func(f *membersFixture)
		want  error    // the injected failure; nil for the use case's own error
		calls []string // up to the failing call, the last one
	}{
		{"the write", func(f *membersFixture) { f.workspaces.roleErr = failure }, failure, calls[:decided+1]},
		{"the projects' step", func(f *membersFixture) { f.projects.errs = map[string]error{"DemoteToGuest": failure} }, failure,
			calls[:decided+2]},
		{"the profile", func(f *membersFixture) { f.profiles.err = failure }, failure, calls},
		{"a member without one", func(f *membersFixture) { f.profiles.profiles = profiles[:2] }, nil, calls},
	}
	for _, tt := range tests {
		uc, f, tx := newUpdateMember()
		tt.set(f)
		got, err := uc.Execute(as(alice), bobInAcme.ID, shared.RoleGuest)
		var se *shared.Error
		if err == nil || errors.As(err, &se) || got != (domain.Member{}) || tx.calls != 1 {
			t.Errorf("%s failing: Execute() = %+v, %v in %d transactions; want an error that is no *shared.Error, in one", tt.name, got, err, tx.calls)
		}
		if tt.want != nil && !errors.Is(err, tt.want) {
			t.Errorf("%s failing: Execute() = %v, want %v", tt.name, err, tt.want)
		}
		if !slices.Equal(f.log.calls, tt.calls) {
			t.Errorf("%s failing: calls\n%q\nwant\n%q", tt.name, f.log.calls, tt.calls)
		}
	}
	uc, f, _ := newUpdateMember()
	if _, err := uc.Execute(context.Background(), bobInAcme.ID, shared.RoleGuest); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and no call", err, f.log.calls)
	}
}
