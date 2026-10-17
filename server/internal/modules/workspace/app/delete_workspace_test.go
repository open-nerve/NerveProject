package app_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// deleteFixture is DeleteWorkspace over fakes sharing one log: alice is
// acme's admin, bob beta's.
type deleteFixture struct {
	log        *callLog
	tx         *fakeTx
	workspaces *fakeWorkspaces
	projects   *fakeProjects
	auth       *fakeAuthorizer
	logs       *strings.Builder
}

// lockedInvitations and lockedProjects are what the deletion's two locks
// answer: the ids, in id order, of the invitations and the projects it
// deletes.
var lockedInvitations, lockedProjects = []uuid.UUID{uuid.NewV7(), uuid.NewV7()}, []uuid.UUID{uuid.NewV7(), uuid.NewV7()}

func newDelete() (*app.DeleteWorkspace, *deleteFixture) {
	log := &callLog{}
	f := &deleteFixture{log: log, tx: &fakeTx{}, workspaces: &fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme, beta},
		invitations: lockedInvitations}, projects: &fakeProjects{log: log, locked: lockedProjects},
		auth: &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{
			{alice.ID, acme.ID}: {WorkspaceRole: shared.RoleAdmin},
			{bob.ID, beta.ID}:   {WorkspaceRole: shared.RoleAdmin},
		}}, logs: &strings.Builder{}}
	return app.NewDeleteWorkspace(f.workspaces, f.projects, f.auth, f.tx, ticking{clocktest.At(now)}, slog.New(slog.NewTextHandler(f.logs, nil))), f
}

// ticking moves a microsecond on each read: a deletion that read the clock
// for each step would delete its rows at three moments.
type ticking struct{ c *clocktest.Fixed }

func (t ticking) Now() time.Time {
	n := t.c.Now()
	t.c.Advance(time.Microsecond)
	return n
}

// deletionLocks are the deletion's locks of w's rows after its decision, in
// order: its invitations, then its projects, through the project module.
func deletionLocks(w domain.Workspace) []string {
	return []string{"LockWorkspaceInvitations " + w.ID.String(), "LockWorkspaceProjects " + w.ID.String()}
}

// cascadeCalls are the deletion's steps on w by user, in order: the
// workspace's own, the invitations locked among them, then the projects',
// through the project module.
func cascadeCalls(user app.AccountState, w domain.Workspace) []string {
	var calls []string
	at := " by " + user.ID.String() + " at " + now.Format(time.RFC3339Nano)
	for _, step := range []string{"DeleteWorkspace", "DeleteInvitations", "DeleteWorkspaceMembers", "DeleteWorkspacePreferences",
		"DeleteWorkspaceProjects"} {
		if step == "DeleteInvitations" {
			calls = append(calls, fmt.Sprintf("%s %v%s", step, lockedInvitations, at))
			continue
		}
		calls = append(calls, step+" "+w.ID.String()+at)
	}
	return calls
}

// DeleteWorkspace locks the workspace FOR NO KEY UPDATE, decides, locks its
// invitations and then its projects through the project module, then
// soft-deletes the workspace, the invitations it locked, its members and
// their settings, and last its projects through the project module, by
// the caller at one moment, in one transaction (M3 design 3.6, M4 design
// 3.5), and logs it. Two callers, two workspaces.
func TestDeleteWorkspaceLocksThenDecidesThenCascades(t *testing.T) {
	for _, tt := range []struct {
		user app.AccountState
		w    domain.Workspace
	}{{alice, acme}, {bob, beta}} {
		uc, f := newDelete()
		if err := uc.Execute(as(tt.user), tt.w.Slug); err != nil {
			t.Errorf("%s, %s: Execute() = %v", tt.user.Email, tt.w.Slug, err)
		}
		want := slices.Concat(lockedDecision(tt.user, tt.w, "LockWorkspaceBySlug", domain.ActionDelete), deletionLocks(tt.w),
			cascadeCalls(tt.user, tt.w))
		if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 {
			t.Errorf("%s, %s: calls = %q in %d transactions, want %q in one", tt.user.Email, tt.w.Slug, f.log.calls, f.tx.calls, want)
		}
		wantLog := `level=INFO msg="workspace deleted" workspace_id=` + tt.w.ID.String() + " user_id=" + tt.user.ID.String() + "\n"
		if _, got, _ := strings.Cut(f.logs.String(), " "); got != wantLog {
			t.Errorf("%s, %s: log %q, want %q", tt.user.Email, tt.w.Slug, got, wantLog)
		}
	}
}

// Each refusal and failure is the answer: nothing is deleted after it, and
// nothing is logged. A step that fails ends the cascade there, and its
// transaction ends with the error, so the database rolls back the steps
// before it; a commit that fails after every step logs nothing either, as
// the log comes after the commit. Never a 404 for a failure.
func TestDeleteWorkspaceRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	decided := lockedDecision(alice, acme, "LockWorkspaceBySlug", domain.ActionDelete)
	locks := deletionLocks(acme)
	locked := slices.Concat(decided, locks)
	steps := cascadeCalls(alice, acme)
	tests := []struct {
		name  string
		user  app.AccountState
		slug  string
		set   func(f *deleteFixture)
		want  error
		calls []string
	}{
		{"no such workspace", alice, "nothing", nil, domain.ErrNotFound, []string{"LockWorkspaceBySlug nothing"}},
		{"not visible", bob, "acme", nil, domain.ErrNotFound, lockedDecision(bob, acme, "LockWorkspaceBySlug", domain.ActionDelete)},
		{"forbidden", carol, "acme", func(f *deleteFixture) { f.auth.errs = map[grantKey]error{{carol.ID, acme.ID}: shared.Forbidden()} },
			shared.Forbidden(), lockedDecision(carol, acme, "LockWorkspaceBySlug", domain.ActionDelete)},
		{"the lock failed", alice, "acme", func(f *deleteFixture) { f.workspaces.lockErrs = map[string]error{"acme": failure} }, failure,
			[]string{"LockWorkspaceBySlug acme"}},
		{"the Authorizer failed", alice, "acme", func(f *deleteFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} },
			failure, decided},
		{"the invitations' lock failed", alice, "acme",
			func(f *deleteFixture) {
				f.workspaces.deleteErrs = map[string]error{"LockWorkspaceInvitations": failure}
			},
			failure, append(slices.Clone(decided), locks[0])},
		{"the projects' lock failed", alice, "acme",
			func(f *deleteFixture) { f.projects.errs = map[string]error{"LockWorkspaceProjects": failure} },
			failure, locked},
		{"the workspace row failed", alice, "acme", func(f *deleteFixture) { f.workspaces.deleteErrs = map[string]error{"DeleteWorkspace": failure} },
			failure, append(slices.Clone(locked), steps[0])},
		{"the invitations failed", alice, "acme", func(f *deleteFixture) { f.workspaces.endErrs = map[string]error{"DeleteInvitations": failure} },
			failure, append(slices.Clone(locked), steps[:2]...)},
		{"the members failed", alice, "acme", func(f *deleteFixture) { f.workspaces.deleteErrs = map[string]error{"DeleteWorkspaceMembers": failure} },
			failure, append(slices.Clone(locked), steps[:3]...)},
		{"the settings failed", alice, "acme",
			func(f *deleteFixture) {
				f.workspaces.deleteErrs = map[string]error{"DeleteWorkspacePreferences": failure}
			},
			failure, append(slices.Clone(locked), steps[:4]...)},
		{"the projects failed", alice, "acme",
			func(f *deleteFixture) { f.projects.errs = map[string]error{"DeleteWorkspaceProjects": failure} },
			failure, append(slices.Clone(locked), steps...)},
		{"the commit failed", alice, "acme", func(f *deleteFixture) { f.tx.commitErr = failure }, failure,
			append(slices.Clone(locked), steps...)},
	}
	for _, tt := range tests {
		uc, f := newDelete()
		if tt.set != nil {
			tt.set(f)
		}
		err := uc.Execute(as(tt.user), tt.slug)
		if !errors.Is(err, tt.want) || (tt.want == failure && errors.Is(err, domain.ErrNotFound)) {
			t.Errorf("%s: Execute() = %v, want %v", tt.name, err, tt.want)
		}
		if !slices.Equal(f.log.calls, tt.calls) || f.tx.calls != 1 || f.logs.Len() != 0 {
			t.Errorf("%s: calls = %q in %d transactions, log %q; want %q in one, no log", tt.name, f.log.calls, f.tx.calls, f.logs, tt.calls)
		}
	}
	uc, f := newDelete()
	if err := uc.Execute(context.Background(), "acme"); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 || f.tx.calls != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and nothing", err, f.log.calls)
	}
}
