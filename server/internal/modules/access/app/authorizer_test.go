package app_test

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type ctxKey struct{}

// membership is one (workspace, user) pair of fakeRoles.
type membership struct{ workspace, user uuid.UUID }

// fakeRoles answers the role of each (workspace, user) it holds, and records
// every call with the value of ctxKey in its context, "(none)" when the
// context has none, so a context dropped on the way fails the call's
// assertion rather than the fake.
type fakeRoles struct {
	roles map[membership]shared.Role
	err   error
	calls []string
}

func (f *fakeRoles) ActiveRole(ctx context.Context, workspaceID, userID uuid.UUID) (shared.Role, bool, error) {
	value, ok := ctx.Value(ctxKey{}).(string)
	if !ok {
		value = "(none)"
	}
	f.calls = append(f.calls, workspaceID.String()+" "+userID.String()+" "+value)
	if f.err != nil {
		return 0, false, f.err
	}
	role, ok := f.roles[membership{workspaceID, userID}]
	return role, ok, nil
}

// projectKey is one (project, user) pair of fakeProjects.
type projectKey struct{ project, user uuid.UUID }

// fakeProjects answers the facts of each (project, user) pair it holds, and
// records every call as fakeRoles does.
type fakeProjects struct {
	facts map[projectKey]app.ProjectFacts
	err   error
	calls []string
}

func (f *fakeProjects) ProjectFacts(ctx context.Context, projectID, userID uuid.UUID) (app.ProjectFacts, bool, error) {
	value, ok := ctx.Value(ctxKey{}).(string)
	if !ok {
		value = "(none)"
	}
	f.calls = append(f.calls, projectID.String()+" "+userID.String()+" "+value)
	if f.err != nil {
		return app.ProjectFacts{}, false, f.err
	}
	p, ok := f.facts[projectKey{projectID, userID}]
	return p, ok, nil
}

var (
	w1, w2 = uuid.NewV7(), uuid.NewV7()
	a, b   = uuid.NewV7(), uuid.NewV7()
)

// Authorize reads the caller's role in the target's workspace, in the
// caller's context: two users, two workspaces, and each answer is the one of
// its own pair.
func TestAuthorizeReadsTheCallersRoleInTheTargetsWorkspace(t *testing.T) {
	roles := &fakeRoles{roles: map[membership]shared.Role{
		{w1, a}: shared.RoleAdmin, {w2, a}: shared.RoleGuest, {w1, b}: shared.RoleMember,
	}}
	projects := &fakeProjects{}
	auth := app.NewAuthorizer(roles, projects)
	tests := []struct {
		user, workspace uuid.UUID
		want            shared.Role // 0: not visible
	}{
		{a, w1, shared.RoleAdmin},
		{a, w2, shared.RoleGuest},
		{b, w1, shared.RoleMember},
		{b, w2, 0},
	}
	for _, tt := range tests {
		roles.calls = nil
		ctx := context.WithValue(context.Background(), ctxKey{}, "tx")
		grant, err := auth.Authorize(ctx, shared.Actor{UserID: tt.user}, "workspace.read", shared.Target{WorkspaceID: tt.workspace})
		want := tt.workspace.String() + " " + tt.user.String() + " tx"
		if len(roles.calls) != 1 || roles.calls[0] != want {
			t.Errorf("ActiveRole calls = %q, want [%q]", roles.calls, want)
		}
		if len(projects.calls) != 0 {
			t.Errorf("a workspace-level target read the projects %q, want none", projects.calls)
		}
		if tt.want == 0 {
			if !errors.Is(err, shared.ErrNotVisible) {
				t.Errorf("user %s in %s: Authorize() = %+v, %v; want ErrNotVisible", tt.user, tt.workspace, grant, err)
			}
			continue
		}
		if err != nil || grant != (shared.Grant{WorkspaceRole: tt.want}) {
			t.Errorf("user %s in %s: Authorize() = %+v, %v; want role %d", tt.user, tt.workspace, grant, err, tt.want)
		}
	}
}

// Nothing is cached: a membership that ends between two calls is not
// visible at the second.
func TestAuthorizeReadsOnEveryCall(t *testing.T) {
	roles := &fakeRoles{roles: map[membership]shared.Role{{w1, a}: shared.RoleAdmin}}
	auth := app.NewAuthorizer(roles, &fakeProjects{})
	ctx := context.WithValue(context.Background(), ctxKey{}, "request")
	target := shared.Target{WorkspaceID: w1}
	if _, err := auth.Authorize(ctx, shared.Actor{UserID: a}, "workspace.read", target); err != nil {
		t.Fatalf("first Authorize() = %v", err)
	}
	delete(roles.roles, membership{w1, a})
	if _, err := auth.Authorize(ctx, shared.Actor{UserID: a}, "workspace.read", target); !errors.Is(err, shared.ErrNotVisible) {
		t.Errorf("Authorize() after the membership ended = %v, want ErrNotVisible", err)
	}
}

// An action without a row fails closed, before anything is read, and is not
// a refusal the caller could act on: an internal error.
func TestAuthorizeRefusesAnActionWithoutARule(t *testing.T) {
	roles := &fakeRoles{roles: map[membership]shared.Role{{w1, a}: shared.RoleAdmin}}
	projects := &fakeProjects{}
	auth := app.NewAuthorizer(roles, projects)
	ctx := context.WithValue(context.Background(), ctxKey{}, "request")
	grant, err := auth.Authorize(ctx, shared.Actor{UserID: a}, "no.such.action", shared.Target{WorkspaceID: w1, ProjectID: uuid.NewV7()})
	var se *shared.Error
	if err == nil || errors.As(err, &se) || grant != (shared.Grant{}) {
		t.Errorf("Authorize() = %+v, %v; want an internal error", grant, err)
	}
	if len(roles.calls)+len(projects.calls) != 0 {
		t.Errorf("ActiveRole calls = %q, ProjectFacts calls %q; want none", roles.calls, projects.calls)
	}
}

// Each port's failure is Authorize's.
func TestAuthorizeReturnsThePortsError(t *testing.T) {
	failure := errors.New("connection reset")
	ctx := context.WithValue(context.Background(), ctxKey{}, "request")
	auth := app.NewAuthorizer(&fakeRoles{err: failure}, &fakeProjects{})
	if _, err := auth.Authorize(ctx, shared.Actor{UserID: a}, "workspace.read", shared.Target{WorkspaceID: w1}); !errors.Is(err, failure) {
		t.Errorf("Authorize() with the roles failing = %v, want %v", err, failure)
	}
	auth = app.NewAuthorizer(&fakeRoles{roles: map[membership]shared.Role{{w1, a}: shared.RoleAdmin}}, &fakeProjects{err: failure})
	if _, err := auth.Authorize(ctx, shared.Actor{UserID: a}, "project.read", shared.Target{WorkspaceID: w1, ProjectID: uuid.NewV7()}); !errors.Is(err, failure) {
		t.Errorf("Authorize() with the projects failing = %v, want %v", err, failure)
	}
}

// A target that names a project has the project's facts read too, for the
// caller and in the caller's context, and the decision takes them: the
// caller's project role is in the Grant, a workspace admin sees a private
// project he is not in. A project of another workspace than the target's,
// or one not found, is seen by no one: its facts do not count.
func TestAuthorizeReadsTheTargetsProject(t *testing.T) {
	p1, p2, gone := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	roles := &fakeRoles{roles: map[membership]shared.Role{{w1, a}: shared.RoleMember, {w1, b}: shared.RoleAdmin}}
	projects := &fakeProjects{facts: map[projectKey]app.ProjectFacts{
		{p1, a}: {WorkspaceID: w1, Member: true, Role: shared.RoleGuest},
		{p1, b}: {WorkspaceID: w1},
		{p2, a}: {WorkspaceID: w2, Public: true, Member: true, Role: shared.RoleAdmin},
	}}
	auth := app.NewAuthorizer(roles, projects)
	tests := []struct {
		user, project uuid.UUID
		want          *shared.Grant // nil: not visible
	}{
		{a, p1, &shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleGuest}},
		{b, p1, &shared.Grant{WorkspaceRole: shared.RoleAdmin}},
		{a, p2, nil},
		{a, gone, nil},
	}
	for _, tt := range tests {
		projects.calls = nil
		ctx := context.WithValue(context.Background(), ctxKey{}, "tx")
		grant, err := auth.Authorize(ctx, shared.Actor{UserID: tt.user}, "project.read", shared.Target{WorkspaceID: w1, ProjectID: tt.project})
		if want := tt.project.String() + " " + tt.user.String() + " tx"; len(projects.calls) != 1 || projects.calls[0] != want {
			t.Errorf("ProjectFacts calls = %q, want [%q]", projects.calls, want)
		}
		if tt.want == nil {
			if !errors.Is(err, shared.ErrNotVisible) {
				t.Errorf("user %s, project %s: Authorize() = %+v, %v; want ErrNotVisible", tt.user, tt.project, grant, err)
			}
			continue
		}
		if err != nil || grant != *tt.want {
			t.Errorf("user %s, project %s: Authorize() = %+v, %v; want %+v", tt.user, tt.project, grant, err, *tt.want)
		}
	}
}
