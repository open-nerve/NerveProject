package app_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// workspaceListProjects is the caller's scopes in acme: both are logged,
// so that reading the historical scope instead of the ordinary one shows.
type workspaceListProjects struct {
	log      *callLog
	projects []app.VisibleProject
}

func (f *workspaceListProjects) Visible(ctx context.Context, wid, uid uuid.UUID) ([]app.VisibleProject, error) {
	return f.projects, f.log.add(ctx, "Visible", wid, uid)
}
func (f *workspaceListProjects) Historical(ctx context.Context, wid, uid uuid.UUID) ([]app.VisibleProject, error) {
	return f.projects, f.log.add(ctx, "Historical", wid, uid)
}

// workspaceList is alice's list of acme: she sees only her own work items
// in web, every one in ops, acme's other project.
func workspaceList(t *testing.T) (*app.ListWorkspaceIssues, *world, *listingPorts, *workspaceListProjects) {
	t.Helper()
	_, w, _ := newGet()
	l, p, _ := listingFixture(t)
	p.pages = [][]app.ListRecord{{listRecord(carol, nil)}}
	p.pids = []uuid.UUID{web, ops}
	d := &workspaceListProjects{log: w.log, projects: []app.VisibleProject{{ID: web, Role: shared.RoleGuest, OwnOnly: true}, {ID: ops, Role: shared.RoleMember}}}
	u := app.NewListWorkspaceIssues(app.ListWorkspaceIssuesDeps{ReadTx: listSnapshot{fakeTx{log: w.log}, p}, Authorizer: w.auth,
		Workspaces: &fakeWorkspaces{log: w.log, slug: "acme", id: acme}, Projects: d, Listing: l})
	return u, w, p, d
}

// The workspace's list (M7 P3a spec 2.2, 2.3): in one read snapshot, the
// workspace, the decision on workspace_issue.list in it, the projects the
// caller sees there (Visible, each with its OwnOnly), and the engine's
// read of them, with the caller's query, no target member and no relation.
func TestListWorkspaceIssues(t *testing.T) {
	u, w, p, d := workspaceList(t)
	page, err := u.Execute(as(alice), "acme", map[string][]string{"assignee_id": {"me"}, "project_id": {ops.String()}, "order_by": {"sort_order"}})
	if err != nil || page.Total != 3 || len(page.Data) != 1 || page.Data[0].ID != carol {
		t.Fatal(page, err)
	}
	want := []string{"Begin", line("WorkspaceBySlug", "acme"), line("Authorize", alice, domain.ActionListWorkspaceIssues, acme, uuid.Nil()), line("Visible", acme, alice)}
	if !slices.Equal(w.log.calls, want) {
		t.Fatal(w.log.calls, want)
	}
	read := p.reads[0]
	if read.ActorID != alice || read.TargetMemberID != nil || read.Query.Relation != "" || read.Query.GroupBy != "" || read.Query.Order != "sort_order" ||
		!slices.Equal(read.Query.Filters.Assignees.IDs, []uuid.UUID{alice}) || !slices.Equal(read.Query.Filters.ProjectIDs, []uuid.UUID{ops}) ||
		!reflect.DeepEqual(read.Directories.Projects, d.projects) {
		t.Fatal(read)
	}
}

// No project seen in the workspace is the empty flat page: nothing is read
// after the scope.
func TestListWorkspaceIssuesWithoutProjects(t *testing.T) {
	u, w, p, d := workspaceList(t)
	d.projects = nil
	page, err := u.Execute(as(alice), "acme", map[string][]string{"subscriber_id": {"me"}})
	if err != nil || !reflect.DeepEqual(page, app.IssueListPage{Data: []domain.Issue{}}) || len(p.calls) != 0 || len(p.reads) != 0 {
		t.Fatal(page, err, p.calls)
	}
	if want := []string{"Begin", line("WorkspaceBySlug", "acme"), line("Authorize", alice, domain.ActionListWorkspaceIssues, acme, uuid.Nil()), line("Visible", acme, alice)}; !slices.Equal(w.log.calls, want) {
		t.Fatal(w.log.calls, want)
	}
}

// The refusal order (spec 2.3): the query before anything is read, so a
// bad one in a workspace not there or not seen is 422 still; then the
// workspace, not there or not seen alike.
func TestListWorkspaceIssuesRefusals(t *testing.T) {
	notAllowed := func(field string) error {
		return shared.Invalid(shared.FieldError{Field: field, Code: shared.FieldNotAllowed, Message: "is not allowed for this list"})
	}
	for _, tt := range []struct {
		name, fail, slug string
		ctx              context.Context
		raw              map[string][]string
		auth             error
		want             error
		calls            int
	}{
		{name: "no actor", ctx: context.Background(), want: shared.Unauthenticated()},
		{name: "relation", ctx: as(alice), raw: map[string][]string{"relation": {"assigned"}}, want: notAllowed("relation")},
		{name: "cycle", ctx: as(alice), raw: map[string][]string{"cycle_id": {"none"}}, want: notAllowed("cycle_id")},
		{name: "group", ctx: as(alice), raw: map[string][]string{"group_by": {"priority"}}, want: notAllowed("group_by")},
		{name: "bad query in a workspace not seen", ctx: as(alice), raw: map[string][]string{"archived": {"false"}}, auth: shared.ErrNotVisible, want: notAllowed("archived")},
		{name: "bad query in a workspace not there", ctx: as(alice), slug: "missing", raw: map[string][]string{"group_key": {"high"}}, want: notAllowed("group_key")},
		{name: "missing workspace", ctx: as(alice), slug: "missing", want: domain.ErrWorkspaceNotFound, calls: 2},
		{name: "hidden workspace", ctx: as(alice), auth: shared.ErrNotVisible, want: domain.ErrWorkspaceNotFound, calls: 3},
		{name: "workspace error", ctx: as(alice), fail: "WorkspaceBySlug", want: errDisk, calls: 2},
		{name: "auth error", ctx: as(alice), fail: "Authorize", want: errDisk, calls: 3},
		{name: "projects error", ctx: as(alice), fail: "Visible", want: errDisk, calls: 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			u, w, p, _ := workspaceList(t)
			w.log.fail = tt.fail
			w.auth.answers = []error{tt.auth}
			slug := tt.slug
			if slug == "" {
				slug = "acme"
			}
			page, err := u.Execute(tt.ctx, slug, tt.raw)
			if !reflect.DeepEqual(err, tt.want) || !reflect.DeepEqual(page, app.IssueListPage{}) || len(w.log.calls) != tt.calls || len(p.reads) != 0 {
				t.Fatal(page, err, tt.want, w.log.calls)
			}
		})
	}
	for _, fail := range []string{"states", "labels", "assignees", "profiles", "flat", "parents", "total"} {
		t.Run(fail, func(t *testing.T) {
			u, _, p, _ := workspaceList(t)
			p.fail = fail
			raw := map[string][]string{"order_by": {"sort_order"}}
			if fail == "parents" {
				p.pages[0][0].Issue.ParentID = &bob
			}
			if fail == "total" {
				p.pages = [][]app.ListRecord{nil, nil}
				cursor, _ := domain.EncodeListCursor(domain.ListCursor{Order: "sort_order", Key: []byte("1"), ID: bob})
				raw["cursor"] = []string{cursor}
			}
			if _, err := u.Execute(as(alice), "acme", raw); !errors.Is(err, errDisk) {
				t.Fatal(err)
			}
		})
	}
	_, w, p, d := workspaceList(t)
	l, _, _ := listingFixture(t)
	u := app.NewListWorkspaceIssues(app.ListWorkspaceIssuesDeps{ReadTx: activityReadFailureTx{}, Authorizer: w.auth, Projects: d, Listing: l})
	if _, err := u.Execute(as(alice), "acme", nil); !errors.Is(err, errDisk) || len(w.log.calls) != 0 || len(p.reads) != 0 {
		t.Fatal(err, w.log.calls)
	}
}
