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
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func ptr[T any](v T) *T { return &v }

// updateFixture is UpdateWorkspace over fakes sharing one log: alice is
// acme's admin, bob beta's.
type updateFixture struct {
	log        *callLog
	tx         *fakeTx
	workspaces *fakeWorkspaces
	auth       *fakeAuthorizer
}

func newUpdate() (*app.UpdateWorkspace, *updateFixture) {
	log := &callLog{}
	f := &updateFixture{log: log, tx: &fakeTx{}, workspaces: &fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme, beta}},
		auth: &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{
			{alice.ID, acme.ID}: {WorkspaceRole: shared.RoleAdmin},
			{bob.ID, beta.ID}:   {WorkspaceRole: shared.RoleAdmin},
		}}}
	return app.NewUpdateWorkspace(f.workspaces, f.auth, f.tx, clocktest.At(now)), f
}

// lockedDecision is the calls of the first two steps on w for user, in the
// transaction: the lock, then the decision on action.
func lockedDecision(user app.AccountState, w domain.Workspace, lock string, action shared.Action) []string {
	return []string{
		lock + " " + w.Slug,
		"Authorize " + user.ID.String() + " " + string(action) + " on " + w.ID.String() + "/" + uuid.Nil().String(),
	}
}

// UpdateWorkspace locks the workspace FOR NO KEY UPDATE, then decides, then
// writes, all in one transaction (M3 design 3.6): the patch is applied by
// the caller at the clock's now, and the answer carries the caller's role.
// Two callers, two workspaces.
func TestUpdateWorkspaceLocksThenDecidesThenWrites(t *testing.T) {
	size := "11-50"
	tests := []struct {
		user app.AccountState
		w    domain.Workspace
		p    domain.WorkspacePatch
		call string
	}{
		{alice, acme, domain.WorkspacePatch{Name: ptr("Acme Inc"), OrganizationSize: &size, Timezone: ptr("Asia/Shanghai")},
			`name="Acme Inc" size="11-50" timezone="Asia/Shanghai"`},
		{bob, beta, domain.WorkspacePatch{Timezone: ptr("Europe/Paris")}, `name=<nil> size=<nil> timezone="Europe/Paris"`},
		{alice, acme, domain.WorkspacePatch{}, `name=<nil> size=<nil> timezone=<nil>`},
	}
	for _, tt := range tests {
		uc, f := newUpdate()
		got, err := uc.Execute(as(tt.user), tt.w.Slug, tt.p)
		want := tt.w
		if tt.p.Name != nil {
			want.Name = *tt.p.Name
		}
		if tt.p.OrganizationSize != nil {
			want.OrganizationSize = tt.p.OrganizationSize
		}
		if tt.p.Timezone != nil {
			want.Timezone = *tt.p.Timezone
		}
		want.Role, want.UpdatedAt = shared.RoleAdmin, now
		if err != nil || got != want {
			t.Errorf("%s, %s: Execute() = %+v, %v; want %+v", tt.user.Email, tt.w.Slug, got, err, want)
		}
		wantCalls := append(lockedDecision(tt.user, tt.w, "LockWorkspaceBySlug", domain.ActionUpdate),
			"UpdateWorkspace "+tt.w.ID.String()+" "+tt.call+" by "+tt.user.ID.String()+" at "+now.Format(time.RFC3339Nano))
		if !slices.Equal(f.log.calls, wantCalls) || f.tx.calls != 1 {
			t.Errorf("%s, %s: calls = %q in %d transactions, want %q in one", tt.user.Email, tt.w.Slug, f.log.calls, f.tx.calls, wantCalls)
		}
	}
}

// Each refusal, and each failure, is the answer, with nothing written: the
// values' check first, without a transaction; then a workspace that is not
// there and one the caller cannot see, both workspace.not_found; a member's
// forbidden; a failed lock, decision or write, never turned into a 404.
func TestUpdateWorkspaceRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	invalid := shared.Invalid(shared.FieldError{Field: "name", Code: shared.FieldTooShort, Message: "must not be empty"})
	rename := domain.WorkspacePatch{Name: ptr("Renamed")}
	tests := []struct {
		name   string
		user   app.AccountState
		slug   string
		p      domain.WorkspacePatch
		set    func(f *updateFixture)
		want   error
		calls  []string
		inTxes int
	}{
		{"invalid values", alice, "acme", domain.WorkspacePatch{Name: ptr("")}, nil, invalid, nil, 0},
		{"no such workspace", alice, "nothing", rename, nil, domain.ErrNotFound, []string{"LockWorkspaceBySlug nothing"}, 1},
		{"not visible", bob, "acme", rename, nil, domain.ErrNotFound, lockedDecision(bob, acme, "LockWorkspaceBySlug", domain.ActionUpdate), 1},
		{"forbidden", carol, "acme", rename, func(f *updateFixture) { f.auth.errs = map[grantKey]error{{carol.ID, acme.ID}: shared.Forbidden()} },
			shared.Forbidden(), lockedDecision(carol, acme, "LockWorkspaceBySlug", domain.ActionUpdate), 1},
		{"the lock failed", alice, "acme", rename, func(f *updateFixture) { f.workspaces.lockErrs = map[string]error{"acme": failure} },
			failure, []string{"LockWorkspaceBySlug acme"}, 1},
		{"the Authorizer failed", alice, "acme", rename, func(f *updateFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} },
			failure, lockedDecision(alice, acme, "LockWorkspaceBySlug", domain.ActionUpdate), 1},
		{"the write failed", alice, "acme", rename, func(f *updateFixture) { f.workspaces.updateErr = failure }, failure,
			append(lockedDecision(alice, acme, "LockWorkspaceBySlug", domain.ActionUpdate),
				"UpdateWorkspace "+acme.ID.String()+` name="Renamed" size=<nil> timezone=<nil> by `+alice.ID.String()+" at "+now.Format(time.RFC3339Nano)), 1},
	}
	for _, tt := range tests {
		uc, f := newUpdate()
		if tt.set != nil {
			tt.set(f)
		}
		got, err := uc.Execute(as(tt.user), tt.slug, tt.p)
		if !errors.Is(err, tt.want) || got != (domain.Workspace{}) {
			t.Errorf("%s: Execute() = %+v, %v; want no workspace and %v", tt.name, got, err, tt.want)
		}
		if tt.want == failure && errors.Is(err, domain.ErrNotFound) {
			t.Errorf("%s: Execute() = %v, which is also workspace.not_found", tt.name, err)
		}
		if !slices.Equal(f.log.calls, tt.calls) || f.tx.calls != tt.inTxes {
			t.Errorf("%s: calls = %q in %d transactions, want %q in %d", tt.name, f.log.calls, f.tx.calls, tt.calls, tt.inTxes)
		}
	}
}

// Without a caller, UpdateWorkspace is 401 and touches nothing, whatever the
// values.
func TestUpdateWorkspaceWithoutACaller(t *testing.T) {
	for _, p := range []domain.WorkspacePatch{{Name: ptr("Renamed")}, {Name: ptr("")}} {
		uc, f := newUpdate()
		if _, err := uc.Execute(context.Background(), "acme", p); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 || f.tx.calls != 0 {
			t.Errorf("Execute(%+v) without an actor = %v, calls %q in %d transactions; want 401 unauthorized and nothing", p, err, f.log.calls, f.tx.calls)
		}
	}
}
