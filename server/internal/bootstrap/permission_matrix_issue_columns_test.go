package bootstrap

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
)

// The issue table of the permission matrix (M4 design 9.2): the project
// level's columns, its PG as the project's guest in each of his three
// cases, beside the member who made the work item and the workspace admin
// who is only a guest of the project; and the work item each column's
// cells name.

// The issue table's columns that are no other table's. Each calls as an
// account of matrixAccounts (accountOf), and names the work item its case
// is about (issueOf): acme's public project does not show its guests every
// work item, as a project starts, and its private one does (guestViewAll),
// so that the guest's three cases are one account's.
const (
	callerCreator       caller = "project member who made it"                 // 建了它的项目成员: PM, on his own
	callerGuestAll      caller = "project guest, every work item"             // 访客·全部: PG, in the private project
	callerGuestOwn      caller = "project guest, his own"                     // 访客·自己: PG, on his own
	callerGuestOthers   caller = "project guest, another's"                   // 访客·别人: PG, on acme's admin's
	callerAdminGuest    caller = "workspace admin and project guest"          // 管理员·访客: on acme's admin's
	callerAdminGuestOwn caller = "workspace admin and project guest, his own" // the same account, on his own
)

// issueColumns are the columns of the issue table, in the order of 9.2's
// table: PA, PM and the member who made the work item; the guest who sees
// every work item, and the guest who sees only his own, on his own and on
// another's; the workspace admin who is a guest of the project, on
// another's and on his own; then the project level's others, as
// projectColumns has them.
var issueColumns = []caller{callerProjectAdmin, callerProjectMember, callerCreator, callerGuestAll, callerGuestOwn, callerGuestOthers,
	callerAdminGuest, callerAdminGuestOwn, callerMemberAndAdmin, callerAdminOnly, callerMemberPublic, callerMemberPrivate, callerGuestOnly,
	callerBefore, callerNever, callerRemoved, callerDeleted}

// issueOf is the name of the work item of its project (projectOf) that a
// column's cells name: its own, made by its account, for a column of
// ownIssues; its project's matrixIssue, made by the workspace's admin,
// for every other.
func issueOf(c caller) string {
	for _, o := range ownIssues {
		if o.c == c {
			return o.name
		}
	}
	return matrixIssue
}

// toWorkItem is the request of a row whose callers each send method, with
// body, to their column's work item (issueOf), after its id.
func toWorkItem(method, after, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return issueRows.to(method, after, issueOf(c), body)(c, s)
	}
}

// onAWorkItem are the cells of a row of the issue table on a work item: a
// project-level row's on one (issueRows.of), but PG's, whose cases are
// the issue table's own columns (issueTable). The answers of a caller who
// acts on it as an admin of its project (PA; PM+WA, and the workspace
// admin who is a guest, on his own: a workspace admin with a membership
// passes each rule of the project, Grant.ProjectAdmin), of PM, of the
// member who made it, and of the project's guest who sees it, every work
// item or his own; issue.not_found for each column that does not see it:
// the guest and the workspace admin who is a guest, on another's, whom
// the guest rule hides it from before any role (M4 design 3.4), and
// WM-私, WG-, P-前 and X; forbidden for WA- and WM-公, who see the project
// and are none of its members.
func onAWorkItem(admin, member, creator, guest cell) map[caller]cell {
	return issueTable(issueRows.of(admin, member, guest, admin, cellForbidden, cellForbidden), map[caller]cell{callerCreator: creator,
		callerGuestAll: guest, callerGuestOwn: guest, callerGuestOthers: cellIssueNotFound, callerAdminGuest: cellIssueNotFound,
		callerAdminGuestOwn: admin})
}

// onTheIssues are the cells of a row of the issue table on its project's
// work items, none named (createIssue): a project-level row's on the
// project (ofProject), but PG's, whose cases are the issue table's own
// columns (issueTable). The answers of a caller who acts as an admin of
// the project (PA, PM+WA, and the workspace admin who is a guest, in
// either column), of its members (PM, and the member who made a work
// item) and of its guests, in each of their columns; project.not_found
// for WM-私, WG-, P-前 and X, who do not see the project; forbidden for
// WA- and WM-公.
func onTheIssues(admin, member, guest cell) map[caller]cell {
	return issueTable(ofProject(admin, member, guest, admin, cellForbidden, cellForbidden), map[caller]cell{callerCreator: member,
		callerGuestAll: guest, callerGuestOwn: guest, callerGuestOthers: guest, callerAdminGuest: admin, callerAdminGuestOwn: admin})
}

// issueTable is a row of the issue table made from project's, a
// project-level row's cells: PG's taken out, and own's, the cells of the
// issue table's own columns, put in.
func issueTable(project, own map[caller]cell) map[caller]cell {
	delete(project, callerProjectGuest)
	maps.Copy(project, own)
	return project
}

// A cell names its column's own work item (issueOf), not only one of its
// column's project: the guest's own column naming acme's admin's work item
// would ask the guest rule's other case under its name, and its 404 could
// pass for his own.
func TestMatrixViolationsCatchesAnotherCasesWorkItem(t *testing.T) {
	s := newSeeded().in(t)
	getIssue := apitest.Operation{ID: "getIssue", Tags: []string{"issue"}, Method: http.MethodGet, Path: "/api/v0/issues/{issue_id}"}
	row := matrixRow{op: getIssue.ID, columns: issueColumns, request: toWorkItem(http.MethodGet, "", ""),
		cells: onAWorkItem(cellOK, cellOK, cellOK, cellOK)}
	ops := []apitest.Operation{getIssue}
	if got := matrixViolations(ops, matrixExemptions{}, []matrixRow{row}, s, nil); len(got) != 0 {
		t.Fatalf("a matching matrix: %q, want none", got)
	}
	another := row
	another.request = func(c caller, s seeded) (string, string, string) {
		if c == callerGuestOwn {
			return row.request(callerGuestOthers, s)
		}
		return row.request(c, s)
	}
	want := []string{fmt.Sprintf("row getIssue, %s: {issue_id} %s is not its column's work item Guest's, %s", callerGuestOwn,
		s.issue("acme/public", matrixIssue), s.issue("acme/public", "Guest's"))}
	if got := matrixViolations(ops, matrixExemptions{}, []matrixRow{another}, s, nil); !slices.Equal(got, want) {
		t.Errorf("the guest's own column at acme's admin's work item: %q, want %q", got, want)
	}
}
