package bootstrap

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"
)

// The rows of the permission matrix of changing and deleting a work item
// (M4 design 9.2), naming it by its id (/issues/{issue_id}): its column's
// work item (issueOf). Each cell writes on a copy of its own. A row of
// 9.2's table has the issue table's columns, and so has a variant on an
// archived work item, which each cell's prepare archives on its copy
// (inDone); the other variants, the project level's, as createIssue's
// (permission_matrix_issues_test.go), or the archived project's.

var cellIssueArchived = cell{http.StatusConflict, "issue.archived"}

// toIssue is the request of a row whose callers each change their column's
// work item with the body body gives for their column.
func toIssue(body func(c caller, s seeded) string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return toWorkItem(http.MethodPatch, "", body(c, s))(c, s)
	}
}

func issueUpdateMatrixRows() []matrixRow {
	// PG, the guest of a project that does not show its guests every work
	// item, on acme's admin's: issue.not_found, whatever his role may do.
	ok := issueRows.of(cellOK, cellOK, cellIssueNotFound, cellOK, cellForbidden, cellForbidden)
	refused := issueRows.of(cellValidationFailed, cellValidationFailed, cellIssueNotFound, cellValidationFailed, cellForbidden, cellForbidden)
	return []matrixRow{
		// The project's admins and members, as createIssue (M4 design 3.4),
		// not a guest who sees the work item; a guest of a project that does
		// not show its guests every work item, the workspace admin who is
		// only its guest too, finds none another made, whatever his role may
		// do: issue.not_found, not forbidden.
		{op: "updateIssue", write: true, columns: issueColumns, request: toWorkItem(http.MethodPatch, "", `{"name":"Mend"}`),
			cells: onAWorkItem(cellOK, cellOK, cellOK, cellForbidden), check: renamesTheIssue},
		// Each field given: Done, high, PA's account for PM's and the removed
		// member's, UI for Bug, the 12th to the 19th of October for no dates.
		{op: "updateIssue", variant: "each field given", write: true, columns: projectColumns,
			request: toIssue(func(c caller, s seeded) string {
				return `{"name":"Mend","state_id":"` + s.state(projectOf(c), "Done").String() + `","priority":"high","assignee_ids":["` +
					s.account(callerProjectAdmin).String() + `"],"label_ids":["` + s.label(projectOf(c), "UI").String() +
					`"],"start_date":"2026-10-12","target_date":"2026-10-19"}`
			}),
			cells: ok, check: changesEachField},
		// A change that changes nothing answers the work item as it is.
		{op: "updateIssue", variant: "nothing changed", write: true, columns: projectColumns,
			request: toWorkItem(http.MethodPatch, "", `{}`), cells: ok, check: readsTheIssue},
		// An archived project's work items are not changed (M4 design 3.8),
		// after the decision: who may not change one learns nothing of its
		// project's state. The body changes nothing: the archived project is
		// refused before what a change changes is weighed, so even a change
		// of nothing is refused.
		{op: "updateIssue", variant: "archived", write: true, columns: archivedColumns,
			request: toWorkItem(http.MethodPatch, "", `{}`), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// Nor are its archived work items: the project's 409 comes first.
		{op: "updateIssue", variant: "archived, its work item archived", write: true, columns: archivedColumns, prepare: inDone(true),
			request: toWorkItem(http.MethodPatch, "", `{}`), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// An archived work item is not changed either (M4 design 3.8), after
		// the decision, by a change of nothing too.
		{op: "updateIssue", variant: "archived work item", write: true, columns: issueColumns, prepare: inDone(true),
			request: toWorkItem(http.MethodPatch, "", `{}`), cells: onAWorkItem(cellIssueArchived, cellIssueArchived, cellIssueArchived, cellForbidden)},
		// The archived project's Backlog, a state of another project of the
		// same workspace: the state is checked after the decision.
		{op: "updateIssue", variant: "another project's state", write: true, columns: projectColumns,
			request: toIssue(func(_ caller, s seeded) string {
				return `{"state_id":"` + s.state("acme/archived", "Backlog").String() + `"}`
			}),
			cells: refused, refusal: "state_id not_allowed"},
		// The archived project's UI, a label of another project of the same
		// workspace, checked after the decision too.
		{op: "updateIssue", variant: "another project's label", write: true, columns: projectColumns,
			request: toIssue(func(_ caller, s seeded) string {
				return `{"label_ids":["` + s.label("acme/archived", "UI").String() + `"]}`
			}),
			cells: refused, refusal: "label_ids[0] not_allowed"},
		// The project's triage state takes no work item (M4 design 3.7).
		{op: "updateIssue", variant: "the triage state", write: true, columns: projectColumns,
			request: toIssue(func(c caller, s seeded) string {
				return `{"state_id":"` + s.state(projectOf(c), "Triage").String() + `"}`
			}),
			cells: refused, refusal: "state_id not_allowed"},
		// PG's account, the project's guest, added to the work item's
		// assignees, who are kept: only he is checked, and refused at his
		// place (M4 design 3.6, 3.7).
		{op: "updateIssue", variant: "a guest assigned", write: true, columns: projectColumns,
			request: toIssue(func(_ caller, s seeded) string {
				return `{"assignee_ids":["` + s.account(callerProjectMember).String() + `","` + s.account(callerRemoved).String() + `","` +
					s.account(accountOf(callerProjectGuest)).String() + `"]}`
			}),
			cells: refused, refusal: "assignee_ids[2] not_allowed"},
		// Dates out of order, checked after the decision with the dependent
		// rows (M4 design 3.7): who may not change the work item learns
		// nothing of its dates.
		{op: "updateIssue", variant: "dates out of order", write: true, columns: projectColumns,
			request: toWorkItem(http.MethodPatch, "", `{"start_date":"2026-10-20","target_date":"2026-10-19"}`), cells: refused,
			refusal: "target_date not_allowed"},
		// The removed member, whose membership of acme and of its public
		// project ended, stays assigned while PM is unassigned: an assignee
		// who has left is kept (M4 design 3.6).
		{op: "updateIssue", variant: "an assignee who left kept", write: true, columns: projectColumns,
			request: toIssue(func(_ caller, s seeded) string {
				return `{"assignee_ids":["` + s.account(callerRemoved).String() + `"]}`
			}),
			cells: ok, check: keepsTheAssigneeWhoLeft},
		// The project's admins, and the member who made the work item
		// (AllowCreator, M4 design 3.4), not another member; a guest of the
		// project may not, also on his own, which he made as the member he
		// was (9.2: made it, then a guest, 403); the member before, who made
		// his and has left the project, does not see it (made it, then left,
		// 404); a guest who sees only his own finds none another made.
		{op: "deleteIssue", write: true, columns: issueColumns, request: toWorkItem(http.MethodDelete, "", ""),
			cells: onAWorkItem(cellNoContent, cellForbidden, cellNoContent, cellForbidden)},
		// An archived project's work items are not deleted (M4 design 3.8),
		// after the decision, as updateIssue's.
		{op: "deleteIssue", variant: "archived", write: true, columns: archivedColumns, request: toWorkItem(http.MethodDelete, "", ""),
			cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		{op: "deleteIssue", variant: "archived, its work item archived", write: true, columns: archivedColumns, prepare: inDone(true),
			request: toWorkItem(http.MethodDelete, "", ""), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
		// An archived work item is deleted as any other (M4 design 3.8).
		{op: "deleteIssue", variant: "archived work item", write: true, columns: issueColumns, prepare: inDone(true),
			request: toWorkItem(http.MethodDelete, "", ""), cells: onAWorkItem(cellNoContent, cellForbidden, cellNoContent, cellForbidden)},
	}
}

// answeredIssue is a work item as an answer shows it: the fields the
// checks read.
type answeredIssue struct {
	ID          uuid.UUID   `json:"id"`
	ProjectID   uuid.UUID   `json:"project_id"`
	SequenceID  int         `json:"sequence_id"`
	Name        string      `json:"name"`
	StateID     uuid.UUID   `json:"state_id"`
	Priority    string      `json:"priority"`
	StartDate   *string     `json:"start_date"`
	TargetDate  *string     `json:"target_date"`
	CompletedAt *string     `json:"completed_at"`
	AssigneeIDs []uuid.UUID `json:"assignee_ids"`
	LabelIDs    []uuid.UUID `json:"label_ids"`
	CreatedByID *uuid.UUID  `json:"created_by_id"`
	UpdatedByID *uuid.UUID  `json:"updated_by_id"`
}

// seededIssue is the column's work item (issueOf) as issueSeed stored it,
// as an answer shows it: of its number (numberOf), in Todo, medium, without
// dates, not completed, assigned to PM's account and the removed member's,
// in id order, labelled Bug, made and changed last by its maker.
func seededIssue(c caller, s seeded) answeredIssue {
	key, name := projectOf(c), issueOf(c)
	number, by := numberOf(key, name)
	maker := s.account(by)
	return answeredIssue{ID: s.issue(key, name), ProjectID: s.project(key), SequenceID: number, Name: name, StateID: s.state(key, "Todo"),
		Priority:    "medium",
		AssigneeIDs: slices.SortedFunc(slices.Values([]uuid.UUID{s.account(callerProjectMember), s.account(callerRemoved)}), uuid.UUID.Compare),
		LabelIDs:    []uuid.UUID{s.label(key, "Bug")}, CreatedByID: &maker, UpdatedByID: &maker}
}

// answered checks that answer, the work item c's cell changes, is the one
// seeded (seededIssue) with change made to it, the caller its last writer
// unless change says otherwise, and completed when completed; completedAt
// is not compared, but present or not.
func answered(t *testing.T, c caller, s seeded, answer string, completed bool, change func(i *answeredIssue)) {
	t.Helper()
	var got answeredIssue
	decodeAnswer(t, answer, &got)
	want := seededIssue(c, s)
	me := s.account(accountOf(c))
	want.UpdatedByID = &me
	change(&want)
	if (got.CompletedAt != nil) != completed {
		t.Errorf("%s changes %s; want it completed %v", c, answer, completed)
	}
	got.CompletedAt = nil
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s changes %s; want %+v", c, answer, want)
	}
}

// readsTheIssue: the column's work item as issueSeed stored it, in each
// field answeredIssue holds (seededIssue).
func readsTheIssue(t *testing.T, c caller, s seeded, answer string) {
	var got answeredIssue
	decodeAnswer(t, answer, &got)
	if want := seededIssue(c, s); !reflect.DeepEqual(got, want) {
		t.Errorf("%s reads %s; want %+v", c, answer, want)
	}
}

// renamesTheIssue: the column's work item named Mend, the rest as it was.
func renamesTheIssue(t *testing.T, c caller, s seeded, answer string) {
	answered(t, c, s, answer, false, func(i *answeredIssue) { i.Name = "Mend" })
}

// changesEachField: the column's work item named Mend, in Done and so
// completed, high, assigned to PA's account alone, labelled UI alone, from
// the 12th to the 19th of October.
func changesEachField(t *testing.T, c caller, s seeded, answer string) {
	answered(t, c, s, answer, true, func(i *answeredIssue) {
		start, target := "2026-10-12", "2026-10-19"
		i.Name, i.StateID, i.Priority, i.StartDate, i.TargetDate = "Mend", s.state(projectOf(c), "Done"), "high", &start, &target
		i.AssigneeIDs, i.LabelIDs = []uuid.UUID{s.account(callerProjectAdmin)}, []uuid.UUID{s.label(projectOf(c), "UI")}
	})
}

// keepsTheAssigneeWhoLeft: the column's work item assigned to the removed
// member alone.
func keepsTheAssigneeWhoLeft(t *testing.T, c caller, s seeded, answer string) {
	answered(t, c, s, answer, false, func(i *answeredIssue) { i.AssigneeIDs = []uuid.UUID{s.account(callerRemoved)} })
}
