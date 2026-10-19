package bootstrap

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"
)

// The rows of the permission matrix of changing a work item (M4 design
// 9.2), naming it by its id (/issues/{issue_id}): its column's project's
// seeded work item, matrixIssue. Each cell writes on a copy of its own.

// toIssue is the request of a row whose callers each change their column's
// project's seeded work item with the body body gives for their column.
func toIssue(body func(c caller, s seeded) string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return issueRows.to(http.MethodPatch, "", matrixIssue, body(c, s))(c, s)
	}
}

func issueUpdateMatrixRows() []matrixRow {
	ok := issueRows.of(cellOK, cellOK, cellForbidden, cellOK, cellForbidden, cellForbidden)
	refused := issueRows.of(cellValidationFailed, cellValidationFailed, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden)
	return []matrixRow{
		// The project's admins and members, as createIssue (M4 design 3.4).
		{op: "updateIssue", write: true, columns: projectColumns, request: issueRows.to(http.MethodPatch, "", matrixIssue, `{"name":"Mend"}`),
			cells: ok, check: renamesTheIssue},
		// Each field given: Done, high, PA's account for PM's and the removed
		// member's, UI for Bug.
		{op: "updateIssue", variant: "each field given", write: true, columns: projectColumns,
			request: toIssue(func(c caller, s seeded) string {
				return `{"name":"Mend","state_id":"` + s.state(projectOf(c), "Done").String() + `","priority":"high","assignee_ids":["` +
					s.account(callerProjectAdmin).String() + `"],"label_ids":["` + s.label(projectOf(c), "UI").String() + `"]}`
			}),
			cells: ok, check: changesEachField},
		// A change that changes nothing answers the work item as it is.
		{op: "updateIssue", variant: "nothing changed", write: true, columns: projectColumns,
			request: issueRows.to(http.MethodPatch, "", matrixIssue, `{}`), cells: ok, check: readsTheIssue},
		// An archived project's work items are not changed (M4 design 3.8),
		// after the decision: who may not change one learns nothing of its
		// project's state. The body changes nothing: the archived project is
		// refused before what a change changes is weighed, so even a change
		// of nothing is refused.
		{op: "updateIssue", variant: "archived", write: true, columns: archivedColumns,
			request: issueRows.to(http.MethodPatch, "", matrixIssue, `{}`), cells: issueRows.ofArchived(cellProjectArchived, cellForbidden)},
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
		// The removed member, whose membership of acme and of its public
		// project ended, stays assigned while PM is unassigned: an assignee
		// who has left is kept (M4 design 3.6).
		{op: "updateIssue", variant: "an assignee who left kept", write: true, columns: projectColumns,
			request: toIssue(func(_ caller, s seeded) string {
				return `{"assignee_ids":["` + s.account(callerRemoved).String() + `"]}`
			}),
			cells: ok, check: keepsTheAssigneeWhoLeft},
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
	CompletedAt *string     `json:"completed_at"`
	AssigneeIDs []uuid.UUID `json:"assignee_ids"`
	LabelIDs    []uuid.UUID `json:"label_ids"`
	CreatedByID *uuid.UUID  `json:"created_by_id"`
	UpdatedByID *uuid.UUID  `json:"updated_by_id"`
}

// seededIssue is the column's project's seeded work item as issueSeed
// stored it, as an answer shows it: its first, in Todo, medium, not
// completed, assigned to PM's account and the removed member's, in id
// order, labelled Bug, made and changed last by acme's admin.
func seededIssue(c caller, s seeded) answeredIssue {
	key, admin := projectOf(c), s.account(callerAdmin)
	return answeredIssue{ID: s.issue(key, matrixIssue), ProjectID: s.project(key), SequenceID: 1, Name: matrixIssue,
		StateID: s.state(key, "Todo"), Priority: "medium",
		AssigneeIDs: slices.SortedFunc(slices.Values([]uuid.UUID{s.account(callerProjectMember), s.account(callerRemoved)}), uuid.UUID.Compare),
		LabelIDs:    []uuid.UUID{s.label(key, "Bug")}, CreatedByID: &admin, UpdatedByID: &admin}
}

// answered checks that answer, the work item c's cell changes, is the
// seeded one with change made to it, the caller its last writer, and
// completed when completed; completedAt is not compared, but present or
// not.
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

// readsTheIssue: the column's project's seeded work item as issueSeed
// stored it, in each field answeredIssue holds (seededIssue).
func readsTheIssue(t *testing.T, c caller, s seeded, answer string) {
	var got answeredIssue
	decodeAnswer(t, answer, &got)
	if want := seededIssue(c, s); !reflect.DeepEqual(got, want) {
		t.Errorf("%s reads %s; want %+v", c, answer, want)
	}
}

// renamesTheIssue: the seeded work item named Mend, the rest as it was.
func renamesTheIssue(t *testing.T, c caller, s seeded, answer string) {
	answered(t, c, s, answer, false, func(i *answeredIssue) { i.Name = "Mend" })
}

// changesEachField: the seeded work item named Mend, in Done and so
// completed, high, assigned to PA's account alone, labelled UI alone.
func changesEachField(t *testing.T, c caller, s seeded, answer string) {
	answered(t, c, s, answer, true, func(i *answeredIssue) {
		i.Name, i.StateID, i.Priority = "Mend", s.state(projectOf(c), "Done"), "high"
		i.AssigneeIDs, i.LabelIDs = []uuid.UUID{s.account(callerProjectAdmin)}, []uuid.UUID{s.label(projectOf(c), "UI")}
	})
}

// keepsTheAssigneeWhoLeft: the seeded work item assigned to the removed
// member alone.
func keepsTheAssigneeWhoLeft(t *testing.T, c caller, s seeded, answer string) {
	answered(t, c, s, answer, false, func(i *answeredIssue) { i.AssigneeIDs = []uuid.UUID{s.account(callerRemoved)} })
}
