package bootstrap

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"
)

// The rows of the permission matrix of a work item's relations (M4 design
// 3.10, 9.2), naming the work item by its id: its column's work item
// (issueOf). A row of 9.2's table has the issue table's columns, and so
// has a variant on an archived work item, which each cell's prepare
// archives on its copy (inDone); the other variants, the project level's,
// as updateIssue's (permission_matrix_issue_updates_test.go), or the
// archived project's.

// relatesTo is the request of a row whose callers each relate their
// column's work item to the one other gives, as blocking it.
func relatesTo(other func(c caller, s seeded) uuid.UUID) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return toWorkItem(http.MethodPost, "/relations", relating("blocking", other(c, s)))(c, s)
	}
}

// beforesItem is the work item of another project that a column's cells
// relate their column's work item to: acme's private project's of the
// member before, which its admins and members see (PA, PM and PM+WA, who
// answer alike in both projects).
func beforesItem(_ caller, s seeded) uuid.UUID {
	return s.issue("acme/private", issueOf(callerBefore))
}

// archivedItem is the work item of acme's archived project, which may be
// related to none.
func archivedItem(_ caller, s seeded) uuid.UUID {
	return s.issue("acme/archived", matrixIssue)
}

func relationMatrixRows() []matrixRow {
	// The workspace admin who is only a guest of the project may relate his
	// own work item, as its admin (Grant.ProjectAdmin), but not to PM's,
	// which the guest rule hides from him: refused at its place (M4 design
	// 3.4).
	relateCells := onAWorkItem(cellCreated, cellCreated, cellCreated, cellForbidden)
	relateCells[callerAdminGuestOwn] = cellValidationFailed
	list := toWorkItem(http.MethodGet, "/relations", "")
	return []matrixRow{
		// Whom getIssue answers the work item (M4 design 3.4, 9.2): its
		// project's admins, members and the guests who see it.
		{op: "listIssueRelations", columns: issueColumns, request: list, cells: onAWorkItem(cellOK, cellOK, cellOK, cellOK),
			check: listsRelations},
		// An archived project's work items' are listed as any other's.
		{op: "listIssueRelations", variant: "archived", columns: archivedColumns, request: list, cells: issueRows.ofArchived(cellOK, cellForbidden),
			check: listsRelations},
		// The project's admins and members, the member who made it too, not
		// its guests, as updateIssue (M4 design 3.4, 9.2).
		{op: "createIssueRelations", write: true, columns: issueColumns, request: relatesTo(otherItemOf), cells: relateCells,
			refusal: "issue_ids[0] not_allowed", check: relatedTo(otherItemOf)},
		// An archived project's work items relate to none (M4 design 3.8),
		// after the decision.
		{op: "createIssueRelations", variant: "archived", write: true, columns: archivedColumns, request: relatesTo(otherItemOf),
			cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// Nor do its archived work items: the project's 409 comes first.
		{op: "createIssueRelations", variant: "archived, its work item archived", write: true, columns: archivedColumns,
			prepare: inDone(true), request: relatesTo(otherItemOf), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// Nor does an archived work item, after the decision, before the work
		// items asked for are looked at.
		{op: "createIssueRelations", variant: "archived work item", write: true, columns: issueColumns, prepare: inDone(true),
			request: relatesTo(otherItemOf), cells: onAWorkItem(cellIssueArchived, cellIssueArchived, cellIssueArchived, cellForbidden)},
		// A work item of another project of the workspace, which the caller
		// sees: related (M4 design 3.10).
		{op: "createIssueRelations", variant: "another project's work item", write: true, columns: projectColumns,
			request: relatesTo(beforesItem), cells: issueRows.of(cellCreated, cellCreated, cellIssueNotFound, cellCreated, cellForbidden, cellForbidden),
			check: relatedTo(beforesItem)},
		// The archived project's work item, after the decision: refused at
		// its place.
		{op: "createIssueRelations", variant: "an archived project's work item", write: true, columns: projectColumns,
			request: relatesTo(archivedItem), refusal: "issue_ids[0] not_allowed",
			cells: issueRows.of(cellValidationFailed, cellValidationFailed, cellIssueNotFound, cellValidationFailed, cellForbidden, cellForbidden)},
	}
}

// relationsListed are the seeded relations (seededRelations) a column
// lists of its work item (issueOf), each as "<its type as the work item
// reads it> <the other's project key>|<the other's name>", in the order
// they were made. PA, PM and PM+WA, members of acme's public and private
// projects, list both of the public project's matrixIssue; the member who
// made one, his own's; the guest who sees every work item, of the private
// project's matrixIssue, the member before's alone: the public project's
// guest rule hides its matrixIssue and the workspace admin's own from
// him. Every other column lists none: the guest's own's other end the
// guest rule hides from him, the workspace admin who is a guest's own's is
// in a project he is none of, and no seeded relation is of the archived
// project's work item.
var relationsListed = map[caller][]string{
	callerProjectAdmin:   {"blocked_by acme/private|" + matrixIssue, "relates_to acme/public|" + issueOf(callerGuestOwn)},
	callerProjectMember:  {"blocked_by acme/private|" + matrixIssue, "relates_to acme/public|" + issueOf(callerGuestOwn)},
	callerMemberAndAdmin: {"blocked_by acme/private|" + matrixIssue, "relates_to acme/public|" + issueOf(callerGuestOwn)},
	callerCreator:        {"duplicate acme/private|" + issueOf(callerBefore)},
	callerGuestAll:       {"blocking acme/private|" + issueOf(callerBefore)},
}

// listsRelations: the relations of the column's work item it sees, as
// relationsListed has them, each other work item by its seeded row's key
// (keyOfRow), or by its id when no seeded row is it.
func listsRelations(t *testing.T, c caller, s seeded, answer string) {
	listed := []string{}
	for _, r := range listIn[answeredRelation](t, answer) {
		other, ok := keyOfRow(s.issues, r.Issue.ID)
		if !ok {
			other = r.Issue.ID.String()
		}
		listed = append(listed, r.RelationType+" "+other)
	}
	if !slices.Equal(listed, relationsListed[c]) {
		t.Errorf("%s lists %s, that is %q; want %q", c, answer, listed, relationsListed[c])
	}
}

// relatedTo is the check of a row whose callers relate their column's work
// item to the one other gives (relatesTo): the answer is that relation
// alone, blocking it as the column's work item reads it, the other work
// item changed last by the caller.
func relatedTo(other func(c caller, s seeded) uuid.UUID) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		got, me := listIn[answeredRelation](t, answer), s.account(accountOf(c))
		if len(got) != 1 || got[0].RelationType != "blocking" || got[0].Issue.ID != other(c, s) ||
			!reflect.DeepEqual(got[0].Issue.UpdatedByID, &me) {
			t.Errorf("%s relates %s; want blocking %s alone, changed last by the caller", c, answer, other(c, s))
		}
	}
}
