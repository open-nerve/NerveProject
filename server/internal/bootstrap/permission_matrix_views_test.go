package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
)

// The rows of the permission matrix of the saved views (M7 P3b spec 2.2,
// R7): a project's, listed and created under the project of each column
// of the issue table (issueColumns), whose member who made it is the
// view's owner, and read, changed and deleted by its id (/views/{view_id}),
// its column's (viewOf); a workspace's own, listed and created under the
// workspace of each column of the workspace level, and read, changed and
// deleted by its id, a member's or an admin's of its column's workspace,
// for the level has no column of an owner. /views/{view_id} names either
// kind, and the view's kind chooses its action (spec 2.5): getView,
// updateView and deleteView have a row of each. The views they rest on
// are in permission_matrix_seed_views_test.go; acme's public project's
// views are turned on (viewsOn), so that a cell that passes the decision
// writes.

var cellViewNotFound = cell{http.StatusNotFound, "navigation.view_not_found"}

// viewRows are the project views the rows name by id: a column that does
// not see its project answers navigation.view_not_found.
var viewRows = rowsByID{path: "views", notFound: cellViewNotFound, find: seeded.projectView}

// toView is the request of a row whose callers each send method, with
// body, to their column's view (viewOf).
func toView(method, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return viewRows.to(method, "", viewOf(c), body)(c, s)
	}
}

// toWorkspaceView is the request of a row whose callers each send method,
// with body, to whose view of their column's workspace's own.
func toWorkspaceView(whose, method, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return method, "/api/v0/views/" + s.workspaceView(workspaceOf(c), whose).String(), body
	}
}

// onAView are the cells of a row of the issue table on a project's view,
// named by its id (viewRows): a project-level row's on one (viewRows.of),
// but PG's, whose cases are the issue table's own columns (issueTable).
// The answers of a caller who acts on another's as an admin of its project
// (PA, PM+WA: Grant.ProjectAdmin), of PM, on another's, of its owner (the
// member who made it, and the workspace admin who is a guest, on his own),
// and of the project's guest who sees it, every view or his own;
// navigation.view_not_found for each column that does not see it: the
// guest and the workspace admin who is a guest, on another's, whom the
// guest rule hides it from before any role (spec 2.2, DP-15), and WM-私,
// WG-, P-前 and X; forbidden for WA- and WM-公, who see the project and
// are none of its members, WA- though he owns its matrixView.
func onAView(admin, member, owner, guest cell) map[caller]cell {
	return issueTable(viewRows.of(admin, member, guest, admin, cellForbidden, cellForbidden), map[caller]cell{callerCreator: owner,
		callerGuestAll: guest, callerGuestOwn: guest, callerGuestOthers: cellViewNotFound, callerAdminGuest: cellViewNotFound,
		callerAdminGuestOwn: owner})
}

// ofAWorkspaceView are the cells of a workspace-level row on a workspace's
// own view, named by its id: the answers of the workspace's admin, member
// and guest, and navigation.view_not_found for the callers the workspace
// is not visible to.
func ofAWorkspaceView(admin, member, guest cell) map[caller]cell {
	return workspaceCells(admin, member, guest, cellViewNotFound, cellViewNotFound, cellViewNotFound)
}

// The bodies of the rows that create a view and change one's name.
const (
	newView     = `{"name":"Mine"}`
	renamedView = `{"name":"Renamed"}`
)

func viewMatrixRows() []matrixRow {
	member, admin := columnsWorkspaceView(aMembersView), columnsWorkspaceView(anAdminsView)
	return []matrixRow{
		// The project's admins, members and guests, each guest who sees
		// only his own work items his own views alone; whether its views
		// are turned on or not.
		{op: "listProjectViews", columns: issueColumns, request: toProject(http.MethodGet, "/views", ""),
			cells: onTheIssues(cellOK, cellOK, cellOK), check: listsTheProjectViews},
		// Its admins and members, the workspace admin who is a guest as its
		// admin; not its other guests.
		{op: "createProjectView", write: true, columns: issueColumns, request: toProject(http.MethodPost, "/views", newView),
			cells: onTheIssues(cellCreated, cellCreated, cellForbidden), check: createsTheProjectView},
		{op: "getView", variant: "a project's view", columns: issueColumns, request: toView(http.MethodGet, ""),
			cells: onAView(cellOK, cellOK, cellOK, cellOK), check: expectView(columnsProjectView, false)},
		// Its owner alone (D29), a member or an admin of the project.
		{op: "updateView", variant: "a project's view", write: true, columns: issueColumns, request: toView(http.MethodPatch, renamedView),
			cells: onAView(cellForbidden, cellForbidden, cellOK, cellForbidden), check: expectView(columnsProjectView, true)},
		// Its owner, or its project's admin (Grant.ProjectAdmin).
		{op: "deleteView", variant: "a project's view", write: true, columns: issueColumns, request: toView(http.MethodDelete, ""),
			cells: onAView(cellNoContent, cellForbidden, cellNoContent, cellForbidden)},
		// The workspace's admins and members, not its guests (R1).
		{op: "listWorkspaceViews", request: toWorkspace(http.MethodGet, "/views", ""), cells: inWorkspace(cellOK, cellOK, cellForbidden),
			check: listsTheWorkspaceViews},
		{op: "createWorkspaceView", write: true, request: toWorkspace(http.MethodPost, "/views", newView),
			cells: inWorkspace(cellCreated, cellCreated, cellForbidden), check: createsTheWorkspaceView},
		{op: "getView", variant: "a workspace's view", request: toWorkspaceView(aMembersView, http.MethodGet, ""),
			cells: ofAWorkspaceView(cellOK, cellOK, cellForbidden), check: expectView(member, false)},
		// Its owner alone.
		{op: "updateView", variant: "a workspace's view, " + aMembersView, write: true,
			request: toWorkspaceView(aMembersView, http.MethodPatch, renamedView), cells: ofAWorkspaceView(cellForbidden, cellOK, cellForbidden),
			check: expectView(member, true)},
		{op: "updateView", variant: "a workspace's view, " + anAdminsView, write: true,
			request: toWorkspaceView(anAdminsView, http.MethodPatch, renamedView), cells: ofAWorkspaceView(cellOK, cellForbidden, cellForbidden),
			check: expectView(admin, true)},
		// Its owner, or the workspace's admin.
		{op: "deleteView", variant: "a workspace's view, " + aMembersView, write: true,
			request: toWorkspaceView(aMembersView, http.MethodDelete, ""), cells: ofAWorkspaceView(cellNoContent, cellNoContent, cellForbidden)},
		{op: "deleteView", variant: "a workspace's view, " + anAdminsView, write: true,
			request: toWorkspaceView(anAdminsView, http.MethodDelete, ""), cells: ofAWorkspaceView(cellNoContent, cellForbidden, cellForbidden)},
	}
}

// answeredView is a view as the rows' checks read an answer.
type answeredView struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	ProjectID   *uuid.UUID `json:"project_id"`
	Name        string     `json:"name"`
	OwnedByID   uuid.UUID  `json:"owned_by_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// matrixViewRow is a seeded view as a check expects it: its id, its
// workspace, its project (nil for a workspace's own), its name and its
// owner's account.
type matrixViewRow struct {
	id, workspace uuid.UUID
	project       *uuid.UUID
	name          string
	owner         uuid.UUID
}

// columnsProjectView is the project view a column's cells name (viewOf),
// as seeded.
func columnsProjectView(c caller, s seeded) matrixViewRow {
	for _, v := range seededViews() {
		if v.key == projectOf(c) && v.name == viewOf(c) {
			slug, _, _ := strings.Cut(v.key, "/")
			project := s.project(v.key)
			return matrixViewRow{id: s.projectView(v.key, v.name), workspace: s.workspace(slug), project: &project, name: v.name, owner: s.account(v.by)}
		}
	}
	s.t.Fatalf("no view %s of %s is seeded", viewOf(c), projectOf(c))
	return matrixViewRow{}
}

// columnsWorkspaceView is whose view of a column's workspace's own, as
// seeded.
func columnsWorkspaceView(whose string) func(c caller, s seeded) matrixViewRow {
	return func(c caller, s seeded) matrixViewRow {
		for _, v := range matrixWorkspaceViews {
			if v.slug == workspaceOf(c) && v.whose == whose {
				return matrixViewRow{id: s.workspaceView(v.slug, whose), workspace: s.workspace(v.slug), name: v.slug + ", " + whose,
					owner: s.account(v.owner)}
			}
		}
		s.t.Fatalf("no view of %s's own, %s, is seeded", workspaceOf(c), whose)
		return matrixViewRow{}
	}
}

// expectView is the check of a row's answers on a view: the view of its
// column that named gives, as seeded; when renamed, named Renamed and
// changed after its creation.
func expectView(named func(caller, seeded) matrixViewRow, renamed bool) func(*testing.T, caller, seeded, string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		var got answeredView
		decodeAnswer(t, answer, &got)
		want := named(c, s)
		if renamed {
			want.name = "Renamed"
		}
		if got.ID != want.id || got.WorkspaceID != want.workspace || !sameID(got.ProjectID, want.project) || got.Name != want.name ||
			got.OwnedByID != want.owner || got.UpdatedAt.After(got.CreatedAt) != renamed {
			t.Errorf("%s answers %s; want %+v, changed %v", c, answer, want, renamed)
		}
	}
}

// sameID reports whether a and b are both nil or name the same id.
func sameID(a, b *uuid.UUID) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

// listsTheProjectViews: the views of the column's project that it sees,
// all of them, or his own alone for a guest the project shows only his
// own work items (Grant.OwnOnly), the workspace admin who is its guest
// too; seeded at one moment, by their ids, descending.
func listsTheProjectViews(t *testing.T, c caller, s seeded, answer string) {
	ownOnly := c == callerGuestOwn || c == callerGuestOthers || c == callerAdminGuest || c == callerAdminGuestOwn
	var want []uuid.UUID
	for _, v := range seededViews() {
		if v.key == projectOf(c) && (!ownOnly || v.by == accountOf(c)) {
			want = append(want, s.projectView(v.key, v.name))
		}
	}
	listsTheViews(t, c, answer, want)
}

// listsTheWorkspaceViews: the views of the column's workspace's own, a
// member's and an admin's; seeded at one moment, by their ids, descending.
func listsTheWorkspaceViews(t *testing.T, c caller, s seeded, answer string) {
	var want []uuid.UUID
	for _, v := range matrixWorkspaceViews {
		if v.slug == workspaceOf(c) {
			want = append(want, s.workspaceView(v.slug, v.whose))
		}
	}
	listsTheViews(t, c, answer, want)
}

// listsTheViews checks that answer lists the views want, by their ids
// descending, the order of views made at one moment (spec 2.4).
func listsTheViews(t *testing.T, c caller, answer string, want []uuid.UUID) {
	var page struct {
		Data []answeredView `json:"data"`
	}
	decodeAnswer(t, answer, &page)
	got := make([]uuid.UUID, len(page.Data))
	for n, v := range page.Data {
		got[n] = v.ID
	}
	slices.SortFunc(want, func(a, b uuid.UUID) int { return b.Compare(a) })
	if !slices.Equal(got, want) {
		t.Errorf("%s lists %s; want %v", c, answer, want)
	}
}

// createsTheProjectView: Mine, in the column's project, owned by its
// account, made at one moment.
func createsTheProjectView(t *testing.T, c caller, s seeded, answer string) {
	var got answeredView
	decodeAnswer(t, answer, &got)
	project := s.project(projectOf(c))
	if got.ID == uuid.Nil() || got.WorkspaceID != s.workspace(workspaceOf(c)) || !sameID(got.ProjectID, &project) || got.Name != "Mine" ||
		got.OwnedByID != s.account(accountOf(c)) || !got.CreatedAt.Equal(got.UpdatedAt) {
		t.Errorf("%s creates %s; want Mine in %s, owned by %s, made at one moment", c, answer, projectOf(c), accountOf(c))
	}
}

// createsTheWorkspaceView: Mine, of the column's workspace's own, owned by
// its account, made at one moment.
func createsTheWorkspaceView(t *testing.T, c caller, s seeded, answer string) {
	var got answeredView
	decodeAnswer(t, answer, &got)
	if got.ID == uuid.Nil() || got.WorkspaceID != s.workspace(workspaceOf(c)) || got.ProjectID != nil || got.Name != "Mine" ||
		got.OwnedByID != s.account(c) || !got.CreatedAt.Equal(got.UpdatedAt) {
		t.Errorf("%s creates %s; want Mine of %s's own, owned by %s, made at one moment", c, answer, workspaceOf(c), c)
	}
}

// A view named by its id ({view_id}) is of its kind's level, in its
// column's place (targetViolation): a project's view is its column's own
// (viewOf) in its column's project, from a column of a project table; a
// workspace's own view is one of its column's workspace, from a column of
// the workspace level. Each of the three fails: a workspace column at a
// project's view, a project column at a workspace's view, a workspace
// column at a view of another workspace; and so does a guest's own column
// at another's view.
func TestMatrixViolationsCatchesAViewOfAnotherKind(t *testing.T) {
	s := newSeeded().in(t)
	getView := apitest.Operation{ID: "getView", Tags: []string{"navigation"}, Method: http.MethodGet, Path: "/api/v0/views/{view_id}"}
	ops := []apitest.Operation{getView}
	project := matrixRow{op: getView.ID, variant: "a project's view", columns: issueColumns, request: toView(http.MethodGet, ""),
		cells: onAView(cellOK, cellOK, cellOK, cellOK)}
	workspace := matrixRow{op: getView.ID, variant: "a workspace's view", request: toWorkspaceView(aMembersView, http.MethodGet, ""),
		cells: ofAWorkspaceView(cellOK, cellOK, cellForbidden)}
	if got := matrixViolations(ops, matrixExemptions{}, []matrixRow{project, workspace}, s, nil); len(got) != 0 {
		t.Fatalf("a matching matrix: %q, want none", got)
	}
	// sends is row, but the cell of at sends GET to the view id.
	sends := func(row matrixRow, at caller, id uuid.UUID) matrixRow {
		r := row
		r.request = func(c caller, s seeded) (string, string, string) {
			if c == at {
				return http.MethodGet, "/api/v0/views/" + id.String(), ""
			}
			return row.request(c, s)
		}
		return r
	}
	public, guests := s.projectView("acme/public", matrixView), s.projectView("acme/public", "Guest's")
	acme, gone := s.workspaceView("acme", aMembersView), s.workspaceView("gone", aMembersView)
	for _, tt := range []struct {
		name string
		row  matrixRow
		want string
	}{
		{"a workspace column at a project's view", sends(workspace, callerAdmin, public),
			"row getView, a workspace's view, admin: {view_id} from a column of no project table (projectTables): a project's row names its columns"},
		{"a project column at a workspace's view", sends(project, callerProjectAdmin, acme),
			fmt.Sprintf("row getView, a project's view, project admin: {view_id} %s is a workspace's view, from a column of no workspace level: "+
				"a workspace's row names its columns", acme)},
		{"a workspace column at a view of another workspace", sends(workspace, callerMember, gone),
			fmt.Sprintf("row getView, a workspace's view, member: {view_id} %s is a view of gone's own, not of its column's workspace acme", gone)},
		{"the guest's own column at another's view", sends(project, callerGuestOwn, public),
			fmt.Sprintf("row getView, a project's view, %s: {view_id} %s is not its column's view Guest's, %s", callerGuestOwn, public, guests)},
	} {
		if got := matrixViolations(ops, matrixExemptions{}, []matrixRow{tt.row}, s, nil); !slices.Equal(got, []string{tt.want}) {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
}
