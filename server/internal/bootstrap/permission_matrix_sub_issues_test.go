package bootstrap

import (
	"net/http"
	"reflect"
	"testing"
	"uuid"
)

// The rows of the permission matrix of a work item's sub-items (M4 design
// 3.9, 9.2), naming the work item by its id: its column's work item
// (issueOf). A row of 9.2's table has the issue table's columns, and so
// has a variant on an archived work item, which each cell's prepare
// archives on its copy (inDone); the other variants, the project level's,
// as updateIssue's (permission_matrix_issue_updates_test.go), or the
// archived project's.

// subItemOf is the work item a column's cells put under their column's
// work item: acme's public project's work item of PM, the member who made
// it, or, for his own column, whose work item that is, the project's
// matrixIssue, acme's admin's. Only the public project's cells weigh it.
func subItemOf(c caller, s seeded) uuid.UUID {
	name := issueOf(callerCreator)
	if c == callerCreator {
		name = matrixIssue
	}
	return s.issue("acme/public", name)
}

// addsSubItem is the request of a row whose callers each put a work item
// under their column's work item: subItemOf, or the one item gives.
func addsSubItem(item func(c caller, s seeded) uuid.UUID) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return toWorkItem(http.MethodPost, "/sub-issues", `{"sub_issue_ids":["`+item(c, s).String()+`"]}`)(c, s)
	}
}

func subIssueMatrixRows() []matrixRow {
	list := toWorkItem(http.MethodGet, "/sub-issues", "")
	// The workspace admin who is only a guest of the project may put work
	// items under his own, as its admin (Grant.ProjectAdmin), but not PM's,
	// which the guest rule hides from him: refused at its place (M4 design
	// 3.4).
	adds := onAWorkItem(cellOK, cellOK, cellOK, cellForbidden)
	adds[callerAdminGuestOwn] = cellValidationFailed
	refused := issueRows.of(cellValidationFailed, cellValidationFailed, cellIssueNotFound, cellValidationFailed, cellForbidden, cellForbidden)
	return []matrixRow{
		// Whom getIssue answers the work item (M4 design 3.4, 9.2): its
		// project's admins, members and the guests who see it.
		{op: "listSubIssues", columns: issueColumns, request: list, cells: onAWorkItem(cellOK, cellOK, cellOK, cellOK), check: listsNone},
		// An archived project's work items' are listed as any other's.
		{op: "listSubIssues", variant: "archived", columns: archivedColumns, request: list, cells: issueRows.ofArchived(cellOK, cellForbidden),
			check: listsNone},
		// The project's admins and members, the member who made it too, not
		// its guests, as updateIssue (M4 design 3.4, 9.2).
		{op: "addSubIssues", write: true, columns: issueColumns, request: addsSubItem(subItemOf), cells: adds,
			refusal: "sub_issue_ids[0] not_allowed", check: putsUnder},
		// An archived project's work items take none under them (M4 design
		// 3.8), after the decision.
		{op: "addSubIssues", variant: "archived", write: true, columns: archivedColumns, request: addsSubItem(subItemOf),
			cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// Nor do its archived work items: the project's 409 comes first.
		{op: "addSubIssues", variant: "archived, its work item archived", write: true, columns: archivedColumns, prepare: inDone(true),
			request: addsSubItem(subItemOf), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// Nor does an archived work item, after the decision, before the
		// work items asked for are looked at.
		{op: "addSubIssues", variant: "archived work item", write: true, columns: issueColumns, prepare: inDone(true),
			request: addsSubItem(subItemOf), cells: onAWorkItem(cellIssueArchived, cellIssueArchived, cellIssueArchived, cellForbidden)},
		// The archived project's work item, another project's of the same
		// workspace: the work items asked for are checked after the
		// decision.
		{op: "addSubIssues", variant: "another project's work item", write: true, columns: projectColumns,
			request: addsSubItem(func(_ caller, s seeded) uuid.UUID { return s.issue("acme/archived", matrixIssue) }), cells: refused,
			refusal: "sub_issue_ids[0] not_allowed"},
	}
}

// listsNone: no seeded work item has a sub-item.
func listsNone(t *testing.T, c caller, _ seeded, answer string) {
	if got := listIn[storedIssue](t, answer); len(got) != 0 {
		t.Errorf("%s lists %s; want none", c, answer)
	}
}

// putsUnder: the column's sub-item (subItemOf) alone, under the column's
// work item, changed last by the caller.
func putsUnder(t *testing.T, c caller, s seeded, answer string) {
	got := listIn[storedIssue](t, answer)
	parent, me := s.issue(projectOf(c), issueOf(c)), s.account(accountOf(c))
	if len(got) != 1 || got[0].ID != subItemOf(c, s) || !reflect.DeepEqual(got[0].ParentID, &parent) ||
		!reflect.DeepEqual(got[0].UpdatedByID, &me) {
		t.Errorf("%s puts %s; want %s alone, under %s, changed last by the caller", c, answer, subItemOf(c, s), parent)
	}
}
