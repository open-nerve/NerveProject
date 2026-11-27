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

// fakeWorkspaceLabels answers its list, logs each call, and fails with err,
// answering nothing then, as the store does.
type fakeWorkspaceLabels struct {
	log  *callLog
	list []domain.Label
	err  error
}

func (f *fakeWorkspaceLabels) ListWorkspaceLabels(ctx context.Context, workspaceID, userID uuid.UUID) ([]domain.Label, error) {
	f.log.add(ctx, "ListWorkspaceLabels %s for %s", workspaceID, userID)
	if f.err != nil {
		return nil, f.err
	}
	return f.list, nil
}

// newListWorkspaceLabels is ListWorkspaceLabels over fakes sharing one log:
// in acme, alice is a member, bob an admin, carol a guest. The store's list
// is newLabels' web's Feature, ops's Docs, then web's UI, under Bug: in no
// order a sort by project, by sort order or by id, either way, gives
// (TestListWorkspaceLabels), so a use case that sorted would answer
// another.
func newListWorkspaceLabels() (*app.ListWorkspaceLabels, *fakeDirectory, *fakeWorkspaceLabels, *fakeAuthorizer) {
	log := &callLog{}
	workspaces := &fakeDirectory{log: log, workspaces: map[string]app.Workspace{"acme": acme}}
	_, l := newLabels()
	labels := &fakeWorkspaceLabels{log: log, list: []domain.Label{l.labels[webFeature], l.labels[opsDocs], l.labels[webUI]}}
	auth := &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{{alice, acme.ID}: {WorkspaceRole: shared.RoleMember},
		{bob, acme.ID}: {WorkspaceRole: shared.RoleAdmin}, {carol, acme.ID}: {WorkspaceRole: shared.RoleGuest}}}
	return app.NewListWorkspaceLabels(workspaces, labels, auth), workspaces, labels, auth
}

// workspaceLabelsListed are the calls of user's list of acme's labels: the
// workspace, without a lock, the decision in it, the list of the labels of
// the projects he is a member of; none in a transaction.
func workspaceLabelsListed(user uuid.UUID) []string {
	return []string{"WorkspaceBySlug acme outside tx",
		fmt.Sprintf("Authorize %s %s on %s/%s outside tx", user, domain.ActionWorkspaceLabelList, acme.ID, uuid.UUID{}),
		fmt.Sprintf("ListWorkspaceLabels %s for %s outside tx", acme.ID, user)}
}

// The use case finds the workspace, decides workspace_label.list in it, then
// answers the store's list for the caller as it is, every label whole, UI's
// parent too, whatever his role.
func TestListWorkspaceLabels(t *testing.T) {
	_, _, fake, _ := newListWorkspaceLabels()
	sortedByNone(t, fake.list, []sortOrder[domain.Label]{
		{"the project's", func(a, b domain.Label) int { return slices.Compare(a.ProjectID[:], b.ProjectID[:]) }},
		{"the project's, reversed", func(a, b domain.Label) int { return slices.Compare(b.ProjectID[:], a.ProjectID[:]) }},
		{"the sort order's", func(a, b domain.Label) int { return cmp.Compare(a.SortOrder, b.SortOrder) }},
		{"the sort order's, reversed", func(a, b domain.Label) int { return cmp.Compare(b.SortOrder, a.SortOrder) }},
		{"the id's", func(a, b domain.Label) int { return slices.Compare(a.ID[:], b.ID[:]) }},
		{"the id's, reversed", func(a, b domain.Label) int { return slices.Compare(b.ID[:], a.ID[:]) }},
	})
	for _, user := range []uuid.UUID{alice, bob, carol} {
		uc, _, labels, _ := newListWorkspaceLabels()
		got, err := uc.Execute(as(user), "acme")
		if want := workspaceLabelsListed(user); err != nil || !reflect.DeepEqual(got, labels.list) || !slices.Equal(labels.log.calls, want) {
			t.Errorf("Execute() as %s = %+v, %v, calls %q; want %+v, calls %q", user, got, err, labels.log.calls, labels.list, want)
		}
	}
}

// A workspace not there, or not visible, is workspace.not_found and lists
// nothing; every failure comes back as itself, after the calls before it
// and none after; without a caller nothing is read.
func TestListWorkspaceLabelsRefuses(t *testing.T) {
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
		{"a workspace he is not in", as(dave), "acme", nil, nil, nil, domain.ErrWorkspaceNotFound, workspaceLabelsListed(dave)[:2]},
		{"the directory failing", as(alice), "acme", errDisk, nil, nil, errDisk, workspaceLabelsListed(alice)[:1]},
		{"the decision failing", as(alice), "acme", nil, errDisk, nil, errDisk, workspaceLabelsListed(alice)[:2]},
		{"the list failing", as(alice), "acme", nil, nil, errDisk, errDisk, workspaceLabelsListed(alice)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, workspaces, labels, auth := newListWorkspaceLabels()
			workspaces.err, labels.err = tt.dirErr, tt.listErr
			if tt.authErr != nil {
				auth.errs = map[grantKey]error{{alice, acme.ID}: tt.authErr}
			}
			got, err := uc.Execute(tt.ctx, tt.slug)
			if !refusedAs(err, tt.want) || got != nil || !slices.Equal(labels.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, labels.log.calls, tt.want, tt.calls)
			}
		})
	}
}
