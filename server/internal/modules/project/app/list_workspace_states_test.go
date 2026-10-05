package app_test

import (
	"cmp"
	"context"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeWorkspaceStates answers its list, logs each call, and fails with err,
// answering nothing then, as the store does.
type fakeWorkspaceStates struct {
	log  *callLog
	list []domain.State
	err  error
}

func (f *fakeWorkspaceStates) ListWorkspaceStates(ctx context.Context, workspaceID, userID uuid.UUID) ([]domain.State, error) {
	f.log.add(ctx, "ListWorkspaceStates %s for %s", workspaceID, userID)
	if f.err != nil {
		return nil, f.err
	}
	return f.list, nil
}

// newListWorkspaceStates is ListWorkspaceStates over fakes sharing one log:
// in acme, alice is a member, bob an admin, carol a guest. The store's list
// is web's Review, ops's Backlog, then web's Todo: in no order a sort by
// project or by sequence, either way, gives (TestListWorkspaceStates), so a
// use case that sorted would answer another.
func newListWorkspaceStates() (*app.ListWorkspaceStates, *fakeDirectory, *fakeWorkspaceStates, *fakeAuthorizer) {
	log := &callLog{}
	workspaces := &fakeDirectory{log: log, workspaces: map[string]app.Workspace{"acme": acme}}
	states := &fakeWorkspaceStates{log: log, list: []domain.State{{ID: webReview, ProjectID: webID, Name: "Review", Sequence: 40000},
		{ID: opsBacklog, ProjectID: opsID, Name: "Backlog", Sequence: 15000}, {ID: webTodo, ProjectID: webID, Name: "Todo", Sequence: 25000}}}
	auth := &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{{alice, acme.ID}: {WorkspaceRole: shared.RoleMember},
		{bob, acme.ID}: {WorkspaceRole: shared.RoleAdmin}, {carol, acme.ID}: {WorkspaceRole: shared.RoleGuest}}}
	return app.NewListWorkspaceStates(workspaces, states, auth), workspaces, states, auth
}

// workspaceStatesListed are the calls of user's list of acme's states: the
// workspace, without a lock, the decision in it, the list of the states of
// the projects he is a member of; none in a transaction.
func workspaceStatesListed(user uuid.UUID) []string {
	return []string{"WorkspaceBySlug acme outside tx",
		fmt.Sprintf("Authorize %s %s on %s/%s outside tx", user, domain.ActionWorkspaceStateList, acme.ID, uuid.UUID{}),
		fmt.Sprintf("ListWorkspaceStates %s for %s outside tx", acme.ID, user)}
}

// The use case finds the workspace, decides workspace_state.list in it, then
// answers the store's list for the caller as it is, whatever his role.
func TestListWorkspaceStates(t *testing.T) {
	_, _, fake, _ := newListWorkspaceStates()
	for _, by := range []struct {
		name  string
		order func(a, b domain.State) int
	}{
		{"the project's", func(a, b domain.State) int { return slices.Compare(a.ProjectID[:], b.ProjectID[:]) }},
		{"the project's, reversed", func(a, b domain.State) int { return slices.Compare(b.ProjectID[:], a.ProjectID[:]) }},
		{"the sequence's", func(a, b domain.State) int { return cmp.Compare(a.Sequence, b.Sequence) }},
		{"the sequence's, reversed", func(a, b domain.State) int { return cmp.Compare(b.Sequence, a.Sequence) }},
	} {
		if slices.IsSortedFunc(fake.list, by.order) {
			t.Fatalf("the store's list %v is in %s order: a use case that sorted so would pass", fake.list, by.name)
		}
	}
	for _, user := range []uuid.UUID{alice, bob, carol} {
		uc, _, states, _ := newListWorkspaceStates()
		got, err := uc.Execute(as(user), "acme")
		if want := workspaceStatesListed(user); err != nil || !reflect.DeepEqual(got, states.list) || !slices.Equal(states.log.calls, want) {
			t.Errorf("Execute() as %s = %+v, %v, calls %q; want %+v, calls %q", user, got, err, states.log.calls, states.list, want)
		}
	}
}

// A workspace not there, or not visible, is workspace.not_found and lists
// nothing; every failure comes back as itself, after the calls before it
// and none after; without a caller nothing is read.
func TestListWorkspaceStatesRefuses(t *testing.T) {
	for _, tt := range []struct {
		name                     string
		ctx                      context.Context
		slug                     string
		dirErr, authErr, listErr error
		want                     error
		calls                    []string
	}{
		{"no caller", context.Background(), "acme", nil, nil, nil, shared.Unauthenticated(), nil},
		{"no workspace", as(alice), "gone", nil, nil, nil, domain.ErrWorkspaceNotFound, []string{"WorkspaceBySlug gone outside tx"}},
		{"a workspace he is not in", as(dave), "acme", nil, nil, nil, domain.ErrWorkspaceNotFound, workspaceStatesListed(dave)[:2]},
		{"the directory failing", as(alice), "acme", errDisk, nil, nil, errDisk, workspaceStatesListed(alice)[:1]},
		{"the decision failing", as(alice), "acme", nil, errDisk, nil, errDisk, workspaceStatesListed(alice)[:2]},
		{"the list failing", as(alice), "acme", nil, nil, errDisk, errDisk, workspaceStatesListed(alice)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, workspaces, states, auth := newListWorkspaceStates()
			workspaces.err, states.err = tt.dirErr, tt.listErr
			if tt.authErr != nil {
				auth.errs = map[grantKey]error{{alice, acme.ID}: tt.authErr}
			}
			got, err := uc.Execute(tt.ctx, tt.slug)
			if !refusedAs(err, tt.want) || got != nil || !slices.Equal(states.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, states.log.calls, tt.want, tt.calls)
			}
		})
	}
}
