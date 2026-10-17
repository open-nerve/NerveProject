package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"uuid"
)

// The rows of the permission matrix of a project's work items (M4 design
// 9.2): creating one, under the project of each column. Each cell writes
// on a copy of its own, so each creation is its project's first.

// newIssue is the body of the work item the rows create.
const newIssue = `{"name":"Fix the login"}`

// withIssueBody is the request of a row whose callers each create a work
// item with the body body gives for their column.
func withIssueBody(body func(c caller, s seeded) string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return toProject(http.MethodPost, "/issues", body(c, s))(c, s)
	}
}

func issueMatrixRows() []matrixRow {
	return []matrixRow{
		// The project's admins and members, not its guests (M4 design 3.4):
		// PM+WA as the member he is; a workspace admin or member who is no
		// member of the project sees it and may not.
		{op: "createIssue", write: true, columns: projectColumns, request: toProject(http.MethodPost, "/issues", newIssue),
			cells: ofProject(cellCreated, cellCreated, cellForbidden, cellCreated, cellForbidden, cellForbidden), check: createsTheIssue},
		// Each field given: in Done, high, assigned to PM's account, a
		// member of the project, and labelled UI; the activity names him by
		// his display name, which identity gives (bootstrap/ports.go).
		{op: "createIssue", variant: "each field given", write: true, columns: projectColumns,
			request: withIssueBody(func(c caller, s seeded) string {
				return `{"name":"Fix","state_id":"` + s.state(projectOf(c), "Done").String() + `","priority":"high","assignee_ids":["` +
					s.account(callerProjectMember).String() + `"],"label_ids":["` + s.label(projectOf(c), "UI").String() + `"]}`
			}),
			cells: ofProject(cellCreated, cellCreated, cellForbidden, cellCreated, cellForbidden, cellForbidden), check: createsTheGivenIssue},
		// An archived project takes no work items (M4 design 3.8), after the
		// decision: who may not create one learns nothing of its state.
		{op: "createIssue", variant: "archived", write: true, columns: archivedColumns, request: toProject(http.MethodPost, "/issues", newIssue),
			cells: ofArchived(cellProjectArchived, cellForbidden)},
		// The archived project's Backlog, a state of another project of the
		// same workspace: the state is checked after the decision.
		{op: "createIssue", variant: "another project's state", write: true, columns: projectColumns,
			request: withIssueBody(func(_ caller, s seeded) string {
				return `{"name":"Fix","state_id":"` + s.state("acme/archived", "Backlog").String() + `"}`
			}),
			cells:   ofProject(cellValidationFailed, cellValidationFailed, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "state_id not_allowed"},
		// The archived project's Bug, a label of another project of the same
		// workspace, checked after the decision too.
		{op: "createIssue", variant: "another project's label", write: true, columns: projectColumns,
			request: withIssueBody(func(_ caller, s seeded) string {
				return `{"name":"Fix","label_ids":["` + s.label("acme/archived", "Bug").String() + `"]}`
			}),
			cells:   ofProject(cellValidationFailed, cellValidationFailed, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "label_ids[0] not_allowed"},
		// The project's triage state, which the matrix seeds through the
		// project store, as createProject makes it, and which the API's lists
		// of states leave out (M4 design 9.3, M17): it takes no work item
		// before the intake (M7), checked after the decision too.
		{op: "createIssue", variant: "the triage state", write: true, columns: projectColumns,
			request: withIssueBody(func(c caller, s seeded) string {
				return `{"name":"Fix","state_id":"` + s.state(projectOf(c), "Triage").String() + `"}`
			}),
			cells:   ofProject(cellValidationFailed, cellValidationFailed, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "state_id not_allowed"},
		// The project's guest, PG's account, as the assignee: a guest is not
		// assigned (M4 design 3.7), and it is checked after the decision.
		{op: "createIssue", variant: "a guest assigned", write: true, columns: projectColumns,
			request: withIssueBody(func(_ caller, s seeded) string {
				return `{"name":"Fix","assignee_ids":["` + s.account(accountOf(callerProjectGuest)).String() + `"]}`
			}),
			cells:   ofProject(cellValidationFailed, cellValidationFailed, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "assignee_ids[0] not_allowed"},
	}
}

// createsTheIssue: the column's project's first work item, Fix the login,
// in its default state Backlog, of no priority, at a state's first sort
// order, not completed, assigned to no one, without labels, made and
// last changed by the caller.
func createsTheIssue(t *testing.T, c caller, s seeded, answer string) {
	var i struct {
		ProjectID   uuid.UUID   `json:"project_id"`
		SequenceID  int         `json:"sequence_id"`
		Name        string      `json:"name"`
		StateID     uuid.UUID   `json:"state_id"`
		Priority    string      `json:"priority"`
		SortOrder   float64     `json:"sort_order"`
		CompletedAt *string     `json:"completed_at"`
		AssigneeIDs []uuid.UUID `json:"assignee_ids"`
		LabelIDs    []uuid.UUID `json:"label_ids"`
		CreatedByID *uuid.UUID  `json:"created_by_id"`
		UpdatedByID *uuid.UUID  `json:"updated_by_id"`
	}
	decodeAnswer(t, answer, &i)
	me := s.account(accountOf(c))
	if i.ProjectID != s.project(projectOf(c)) || i.SequenceID != 1 || i.Name != "Fix the login" || i.StateID != s.state(projectOf(c), "Backlog") ||
		i.Priority != "none" || i.SortOrder != 65535 || i.CompletedAt != nil || len(i.AssigneeIDs) != 0 || len(i.LabelIDs) != 0 ||
		i.CreatedByID == nil || *i.CreatedByID != me || i.UpdatedByID == nil || *i.UpdatedByID != me {
		t.Errorf("%s creates %s; want %s's work item 1, Fix the login, in Backlog, none, at 65535, by %s", c, answer, projectOf(c), me)
	}
}

// createsTheGivenIssue: the column's project's first work item, Fix, in
// Done, high, at a state's first sort order, completed at its creation,
// assigned to PM's account, labelled UI.
func createsTheGivenIssue(t *testing.T, c caller, s seeded, answer string) {
	var i struct {
		SequenceID  int         `json:"sequence_id"`
		StateID     uuid.UUID   `json:"state_id"`
		Priority    string      `json:"priority"`
		SortOrder   float64     `json:"sort_order"`
		CompletedAt *string     `json:"completed_at"`
		AssigneeIDs []uuid.UUID `json:"assignee_ids"`
		LabelIDs    []uuid.UUID `json:"label_ids"`
	}
	decodeAnswer(t, answer, &i)
	pm, ui := s.account(callerProjectMember), s.label(projectOf(c), "UI")
	if i.SequenceID != 1 || i.StateID != s.state(projectOf(c), "Done") || i.Priority != "high" || i.SortOrder != 65535 || i.CompletedAt == nil ||
		!slices.Equal(i.AssigneeIDs, []uuid.UUID{pm}) || !slices.Equal(i.LabelIDs, []uuid.UUID{ui}) {
		t.Errorf("%s creates %s; want %s's work item 1 in Done, high, at 65535, completed, assigned to %s, labelled %s", c, answer,
			projectOf(c), pm, ui)
	}
}
