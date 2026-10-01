package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeLister answers its list, logs each call with the visibility asked for,
// and fails with err, answering nothing then, as the store does.
type fakeLister struct {
	log  *callLog
	list []domain.Project
	err  error
}

func (f *fakeLister) ListProjects(ctx context.Context, workspaceID, userID uuid.UUID, v domain.Visibility, archived bool) ([]domain.Project, error) {
	f.log.add(ctx, "ListProjects %s for %s %+v archived %v", workspaceID, userID, v, archived)
	if f.err != nil {
		return nil, f.err
	}
	return f.list, nil
}

// newList is ListProjects over fakes sharing one log: in acme, alice is a
// member, bob an admin, carol a guest.
func newList() (*app.ListProjects, *fakeDirectory, *fakeLister, *fakeAuthorizer) {
	log := &callLog{}
	workspaces := &fakeDirectory{log: log, workspaces: map[string]app.Workspace{"acme": acme}}
	projects := &fakeLister{log: log, list: []domain.Project{{Name: "Web"}}}
	auth := &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{{alice, acme.ID}: {WorkspaceRole: shared.RoleMember},
		{bob, acme.ID}: {WorkspaceRole: shared.RoleAdmin}, {carol, acme.ID}: {WorkspaceRole: shared.RoleGuest}}}
	return app.NewListProjects(workspaces, projects, auth), workspaces, projects, auth
}

// The use case finds the workspace without a lock, outside any
// transaction, decides project.list in it, then lists what the role the
// decision read sees, archived or not as asked; the answer is the list.
func TestListProjects(t *testing.T) {
	for _, tt := range []struct {
		user     uuid.UUID
		archived bool
		v        domain.Visibility
	}{
		{alice, false, domain.Visibility{Public: true}},
		{bob, true, domain.Visibility{All: true, Public: true}},
		{carol, false, domain.Visibility{}},
	} {
		uc, workspaces, _, _ := newList()
		got, err := uc.Execute(as(tt.user), "acme", tt.archived)
		want := []string{"WorkspaceBySlug acme outside tx",
			fmt.Sprintf("Authorize %s project.list on %s/%s outside tx", tt.user, acme.ID, uuid.UUID{}),
			fmt.Sprintf("ListProjects %s for %s %+v archived %v outside tx", acme.ID, tt.user, tt.v, tt.archived)}
		if err != nil || len(got) != 1 || got[0].Name != "Web" || !slices.Equal(workspaces.log.calls, want) {
			t.Errorf("Execute() as %s = %+v, %v, calls %q; want the list, calls %q", tt.user, got, err, workspaces.log.calls, want)
		}
	}
}

// A workspace not there, or not visible, is workspace.not_found and lists
// nothing; every failure comes back as itself; without a caller nothing is
// read.
func TestListProjectsRefuses(t *testing.T) {
	failure := errors.New("connection reset")
	tests := []struct {
		name     string
		ctx      context.Context
		slug     string
		dirErr   error
		authErr  error
		listErr  error
		want     error
		listRead bool
	}{
		{"no workspace", as(alice), "gone", nil, nil, nil, domain.ErrWorkspaceNotFound, false},
		{"a workspace he is not in", as(dave), "acme", nil, nil, nil, domain.ErrWorkspaceNotFound, false},
		{"the directory failing", as(alice), "acme", failure, nil, nil, failure, false},
		{"the decision failing", as(alice), "acme", nil, failure, nil, failure, false},
		{"the list failing", as(alice), "acme", nil, nil, failure, failure, true},
		{"no caller", context.Background(), "acme", nil, nil, nil, shared.Unauthenticated(), false},
	}
	for _, tt := range tests {
		uc, workspaces, projects, auth := newList()
		workspaces.err, projects.err = tt.dirErr, tt.listErr
		if tt.authErr != nil {
			auth.errs = map[grantKey]error{{alice, acme.ID}: tt.authErr}
		}
		got, err := uc.Execute(tt.ctx, tt.slug, false)
		listed := slices.ContainsFunc(workspaces.log.calls, func(c string) bool { return len(c) > 12 && c[:12] == "ListProjects" })
		if !errors.Is(err, tt.want) || got != nil || listed != tt.listRead {
			t.Errorf("%s: Execute() = %+v, %v, calls %q; want nothing, %v", tt.name, got, err, workspaces.log.calls, tt.want)
		}
	}
}
