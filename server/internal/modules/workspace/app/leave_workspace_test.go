package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newLeave is LeaveWorkspace over membersFixture's fakes: in acme alice is
// the admin, bob a member, carol's membership has ended; in beta bob is a
// guest.
func newLeave() (*app.LeaveWorkspace, *membersFixture, *fakeTx) {
	f := newMembers()
	tx := &fakeTx{}
	return app.NewLeaveWorkspace(f.workspaces, f.profiles, f.projects, f.auth, tx, clockAt{at: clockNow}), f, tx
}

// leavingCalls are the calls up to the decision on user's leaving of w: the
// workspace's lock by its slug, the decision on workspace.leave.
func leavingCalls(user app.AccountState, w domain.Workspace) []string {
	return lockedDecision(user, w, "LockWorkspaceBySlug", domain.ActionLeave)
}

// otherAdmin is the call that asks whether w has an active admin other than
// user.
func otherAdmin(user app.AccountState, w domain.Workspace) string {
	return "HasOtherAdmin " + w.ID.String() + " " + user.ID.String()
}

// The caller leaves: the workspace locked FOR NO KEY UPDATE by its slug, the
// decision, then the ending of his own membership, by himself at the
// clock's one time, all in one transaction (M3 design 3.6, 3.8). A member
// and a guest are asked nothing more; an admin leaves when the workspace
// has another active admin: dave, made acme's admin for the case.
func TestLeaveWorkspaceLocksDecidesThenEnds(t *testing.T) {
	dave := app.AccountState{ID: uuid.NewV7(), Email: "dave@corp.com", Active: true}
	tests := []struct {
		name string
		user app.AccountState
		w    domain.Workspace
		set  func(f *membersFixture)
		want []string
	}{
		{"a member", bob, acme, nil, slices.Concat(leavingCalls(bob, acme), endingCalls(acme.ID, bob, bob.ID))},
		{"a guest", bob, beta, nil, slices.Concat(leavingCalls(bob, beta), endingCalls(beta.ID, bob, bob.ID))},
		{"an admin, beside another", alice, acme, func(f *membersFixture) {
			f.workspaces.memberships[acme.ID] = append(f.workspaces.memberships[acme.ID],
				domain.Membership{ID: uuid.NewV7(), WorkspaceID: acme.ID, MemberID: dave.ID, Role: shared.RoleAdmin, IsActive: true})
		}, slices.Concat(leavingCalls(alice, acme), []string{otherAdmin(alice, acme)}, endingCalls(acme.ID, alice, alice.ID))},
	}
	for _, tt := range tests {
		uc, f, tx := newLeave()
		if tt.set != nil {
			tt.set(f)
		}
		err := uc.Execute(as(tt.user), tt.w.Slug)
		if err != nil || !slices.Equal(f.log.calls, tt.want) || tx.calls != 1 {
			t.Errorf("%s: Execute() = %v, calls\n%q\nin %d transactions; want nil,\n%q\nin one", tt.name, err, f.log.calls, tx.calls, tt.want)
		}
	}
}

// Each refusal is the answer, and nothing is ended. A workspace not there,
// deleted while the lock waited, or of which the caller is not an active
// member is workspace.not_found; acme's only active admin is
// workspace.sole_admin, beside a member or alone (M3 design 3.7 rule 1),
// asked after the decision; a failure is never a 404.
func TestLeaveWorkspaceRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	tests := []struct {
		name  string
		user  app.AccountState
		slug  string
		set   func(f *membersFixture)
		want  error
		calls []string
	}{
		{"no such workspace", bob, "gone", nil, domain.ErrNotFound, []string{"LockWorkspaceBySlug gone"}},
		{"deleted while the lock waited", bob, "acme", func(f *membersFixture) { f.workspaces.lockErrs = map[string]error{"acme": app.ErrNotFound} },
			domain.ErrNotFound, []string{"LockWorkspaceBySlug acme"}},
		{"not a member", carol, "acme", nil, domain.ErrNotFound, leavingCalls(carol, acme)},
		{"the only admin", alice, "acme", nil, domain.ErrSoleAdmin, append(leavingCalls(alice, acme), otherAdmin(alice, acme))},
		{"the only admin, alone", alice, "acme",
			func(f *membersFixture) { f.workspaces.memberships[acme.ID] = []domain.Membership{aliceInAcme} }, domain.ErrSoleAdmin,
			append(leavingCalls(alice, acme), otherAdmin(alice, acme))},
		{"the lock failed", bob, "acme", func(f *membersFixture) { f.workspaces.lockErrs = map[string]error{"acme": failure} }, failure,
			[]string{"LockWorkspaceBySlug acme"}},
		{"the Authorizer failed", bob, "acme", func(f *membersFixture) { f.auth.errs = map[grantKey]error{{bob.ID, acme.ID}: failure} }, failure,
			leavingCalls(bob, acme)},
		{"the question of another admin failed", alice, "acme",
			func(f *membersFixture) { f.workspaces.endErrs = map[string]error{"HasOtherAdmin": failure} }, failure,
			append(leavingCalls(alice, acme), otherAdmin(alice, acme))},
	}
	for _, tt := range tests {
		uc, f, tx := newLeave()
		if tt.set != nil {
			tt.set(f)
		}
		err := uc.Execute(as(tt.user), tt.slug)
		if !errors.Is(err, tt.want) {
			t.Errorf("%s: Execute() = %v; want %v", tt.name, err, tt.want)
		}
		if tt.want == failure && errors.Is(err, domain.ErrNotFound) {
			t.Errorf("%s: Execute() = %v, which is also workspace.not_found", tt.name, err)
		}
		if !slices.Equal(f.log.calls, tt.calls) || tx.calls != 1 {
			t.Errorf("%s: calls = %q in %d transactions, want %q in one", tt.name, f.log.calls, tx.calls, tt.calls)
		}
	}
	uc, f, tx := newLeave()
	if err := uc.Execute(context.Background(), "acme"); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 || tx.calls != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q in %d transactions; want 401 unauthorized and no call", err, f.log.calls, tx.calls)
	}
}

// A failed read of the leaver's address, a member without an account, a
// failed step of the ending, the projects' refusal of the only admin of a
// project with other members, and a refused commit each fail the
// transaction: the answer is the error as it came, project.sole_admin
// itself (M3 design 3.7 rule 2); the calls are the leaving's own, each
// once, up to the failing one: nothing runs after it.
func TestLeaveWorkspaceFailsWithinTheTransaction(t *testing.T) {
	failure := errors.New("connection reset")
	soleAdmin := shared.NewError(shared.KindConflict, "project.sole_admin", "Ending the membership would leave a project without an admin.")
	calls := slices.Concat(leavingCalls(bob, acme), endingCalls(acme.ID, bob, bob.ID))
	decided := len(leavingCalls(bob, acme))
	tests := []struct {
		name  string
		set   func(f *membersFixture, tx *fakeTx)
		want  error // nil for the use case's own, which is no *shared.Error
		calls []string
	}{
		{"the address", func(f *membersFixture, _ *fakeTx) { f.profiles.err = failure }, failure, calls[:decided+1]},
		{"a member without an account", func(f *membersFixture, _ *fakeTx) { f.profiles.profiles = profiles[:2] }, nil, calls[:decided+1]},
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
		uc, f, tx := newLeave()
		tt.set(f, tx)
		err := uc.Execute(as(bob), "acme")
		var se *shared.Error
		switch {
		case tt.want == nil && (err == nil || errors.As(err, &se)):
			t.Errorf("%s failing: Execute() = %v; want an error that is no *shared.Error", tt.name, err)
		case tt.want != nil && !errors.Is(err, tt.want):
			t.Errorf("%s failing: Execute() = %v, want %v", tt.name, err, tt.want)
		}
		if !slices.Equal(f.log.calls, tt.calls) || tx.calls != 1 {
			t.Errorf("%s failing: calls\n%q\nin %d transactions; want\n%q\nin one", tt.name, f.log.calls, tx.calls, tt.calls)
		}
	}
}
