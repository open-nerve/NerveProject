package app_test

import (
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// reactivateFixture is ReactivateMember over membersFixture's fakes, with
// the accounts of alice, bob and carol, hers deactivated, and 2 ended
// project memberships to count.
type reactivateFixture struct {
	*membersFixture
	tx       *fakeTx
	accounts *fakeAccounts
	counts   *fakeCounts
	logs     *strings.Builder
}

func newReactivate() (*app.ReactivateMember, *reactivateFixture) {
	m := newMembers()
	f := &reactivateFixture{membersFixture: m, tx: &fakeTx{}, accounts: &fakeAccounts{log: m.log, accounts: []app.AccountState{alice, bob, carol}},
		counts: &fakeCounts{log: m.log, n: 2}, logs: &strings.Builder{}}
	return app.NewReactivateMember(f.accounts, f.workspaces, f.counts, f.tx, clockAt{clockNow, m.log}, slog.New(slog.NewTextHandler(f.logs, nil))), f
}

// reactivationCalls are the calls of user's reactivation in w up to the
// read of his membership: his account's lock by his address, the
// workspace's by its slug, the membership.
func reactivationCalls(user app.AccountState, w domain.Workspace) []string {
	return []string{"ShareAccountByEmail " + user.Email, "LockWorkspaceBySlug " + w.Slug, "MemberOf " + w.ID.String() + " " + user.ID.String()}
}

// reactivatedLog is the one line a reactivation logs.
func reactivatedLog(w domain.Workspace, user app.AccountState) string {
	return `level=INFO msg="workspace member reactivated" workspace_id=` + w.ID.String() + " user_id=" + user.ID.String() + " by=cli\n"
}

// The account row first, by the address normalized, then the workspace's
// lock, the membership, the clock under the lock, the reactivation and the
// count of his ended project memberships, all in one transaction (M3
// design 3.6's lock table, 3.11); one line logged. carol's account is
// deactivated, and is reactivated all the same; bob's membership of acme,
// ended for the case, is reactivated with his account active. Each keeps
// its role.
func TestReactivateMemberRestoresTheEndedMembership(t *testing.T) {
	at := clockNow.Format(time.RFC3339Nano)
	tests := []struct {
		name  string
		user  app.AccountState
		email string
		set   func(f *reactivateFixture)
		want  app.Reactivation
	}{
		{"carol, deactivated", carol, " Carol@Corp.COM ", nil,
			app.Reactivation{Email: carol.Email, Role: shared.RoleGuest, EndedProjectMemberships: 2, AccountActive: false}},
		{"bob, active", bob, bob.Email, func(f *reactivateFixture) { f.workspaces.memberships[acme.ID][1].IsActive = false },
			app.Reactivation{Email: bob.Email, Role: shared.RoleMember, EndedProjectMemberships: 2, AccountActive: true}},
	}
	for _, tt := range tests {
		uc, f := newReactivate()
		if tt.set != nil {
			tt.set(f)
		}
		got, err := uc.Execute(t.Context(), "acme", tt.email)
		want := append(reactivationCalls(tt.user, acme), "Now", "ReactivateMember "+acme.ID.String()+" "+tt.user.ID.String()+" at "+at,
			"CountInactive "+acme.ID.String()+" "+tt.user.ID.String())
		if err != nil || got != tt.want || !slices.Equal(f.log.calls, want) || f.tx.calls != 1 {
			t.Errorf("%s: Execute() = %+v, %v, calls\n%q\nin %d transactions; want %+v,\n%q\nin one", tt.name, got, err, f.log.calls, f.tx.calls,
				tt.want, want)
		}
		if logs := logged(f.logs); logs != reactivatedLog(acme, tt.user) {
			t.Errorf("%s: logs = %q, want %q", tt.name, logs, reactivatedLog(acme, tt.user))
		}
	}
}

// An active membership is reported and nothing changes: no clock, no
// write, no count, no log line (M3 design 3.11, as Plane's command).
func TestReactivateMemberLeavesAnActiveMembership(t *testing.T) {
	uc, f := newReactivate()
	got, err := uc.Execute(t.Context(), "acme", bob.Email)
	want := app.Reactivation{Email: bob.Email, Role: shared.RoleMember, AlreadyActive: true, AccountActive: true}
	if err != nil || got != want || !slices.Equal(f.log.calls, reactivationCalls(bob, acme)) || f.tx.calls != 1 || f.logs.Len() != 0 {
		t.Errorf("Execute() = %+v, %v, calls %q in %d transactions, logs %q; want %+v, %q in one, no log", got, err, f.log.calls, f.tx.calls,
			f.logs, want, reactivationCalls(bob, acme))
	}
}

// Each refusal and each failure is the answer, and nothing is logged;
// nothing is reactivated before a refusal. Each but the commit's failure
// is what the transaction's function returned, so that the transaction
// rolls back on it, a failure after the write too; the commit's failure
// comes once the function returned nil: no account of the address is
// workspace.account_not_found, a workspace not there, or deleted while the
// lock waited, workspace.slug_not_found, an account with no membership of
// it workspace.never_a_member, each as itself and as no other problem; a
// failure is no problem at all. The clock logs its reads among the calls:
// no refusal reads it.
func TestReactivateMemberRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	at := clockNow.Format(time.RFC3339Nano)
	reactivating := append(reactivationCalls(carol, acme), "Now", "ReactivateMember "+acme.ID.String()+" "+carol.ID.String()+" at "+at)
	tests := []struct {
		name        string
		slug, email string
		set         func(f *reactivateFixture)
		want        error
		calls       []string
	}{
		{"no such account", "acme", "nobody@corp.com", nil, domain.ErrAccountNotFound, []string{"ShareAccountByEmail nobody@corp.com"}},
		{"no such workspace", "gone", carol.Email, nil, domain.ErrSlugNotFound, []string{"ShareAccountByEmail " + carol.Email, "LockWorkspaceBySlug gone"}},
		{"deleted while the lock waited", "acme", carol.Email, func(f *reactivateFixture) {
			f.workspaces.lockErrs = map[string]error{"acme": app.ErrNotFound}
		}, domain.ErrSlugNotFound, []string{"ShareAccountByEmail " + carol.Email, "LockWorkspaceBySlug acme"}},
		{"never a member", "beta", alice.Email, nil, domain.ErrNeverAMember, reactivationCalls(alice, beta)},
		{"the account's lock failed", "acme", carol.Email, func(f *reactivateFixture) { f.accounts.err = failure }, failure,
			[]string{"ShareAccountByEmail " + carol.Email}},
		{"the workspace's lock failed", "acme", carol.Email, func(f *reactivateFixture) {
			f.workspaces.lockErrs = map[string]error{"acme": failure}
		}, failure, []string{"ShareAccountByEmail " + carol.Email, "LockWorkspaceBySlug acme"}},
		{"the membership's read failed", "acme", carol.Email, func(f *reactivateFixture) { f.workspaces.membersErr = failure }, failure,
			reactivationCalls(carol, acme)},
		{"the reactivation failed", "acme", carol.Email, func(f *reactivateFixture) { f.workspaces.restoreErr = failure }, failure, reactivating},
		{"the count failed", "acme", carol.Email, func(f *reactivateFixture) { f.counts.err = failure }, failure,
			append(reactivating, "CountInactive "+acme.ID.String()+" "+carol.ID.String())},
		{"the commit failed", "acme", carol.Email, func(f *reactivateFixture) { f.tx.commitErr = failure }, failure,
			append(reactivating, "CountInactive "+acme.ID.String()+" "+carol.ID.String())},
	}
	for _, tt := range tests {
		uc, f := newReactivate()
		if tt.set != nil {
			tt.set(f)
		}
		got, err := uc.Execute(t.Context(), tt.slug, tt.email)
		if !errors.Is(err, tt.want) || got != (app.Reactivation{}) {
			t.Errorf("%s: Execute() = %+v, %v; want %v", tt.name, got, err, tt.want)
		}
		if !f.tx.answered(err) {
			t.Errorf("%s: Execute() = %v, the transaction's function returned %v, its commit failing with %v; want the answer to come out "+
				"of the transaction as itself", tt.name, err, f.tx.returned, f.tx.commitErr)
		}
		answeredAs(t, tt.name, err, tt.want)
		if !slices.Equal(f.log.calls, tt.calls) || f.tx.calls != 1 || f.logs.Len() != 0 {
			t.Errorf("%s: calls = %q in %d transactions, logs %q; want %q in one, no log", tt.name, f.log.calls, f.tx.calls, f.logs, tt.calls)
		}
	}
}
