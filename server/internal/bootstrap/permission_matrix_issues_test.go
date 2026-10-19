package bootstrap

import (
	"maps"
	"net/http"
	"slices"
	"strconv"
	"testing"
	"uuid"
)

// The rows of the permission matrix of a project's work items (M4 design
// 9.2): creating one, under the project of each column, and reading one,
// naming it by its id (/issues/{issue_id}), or by its identifier in its
// column's workspace (/workspaces/{slug}/issues/{identifier}): its
// column's work item (issueOf). Each creating cell writes on a copy of its
// own, so each creation is its project's next work item after the seeded
// ones (nextNumber); the reading cells share one. A row of 9.2's table has
// the issue table's columns (issueColumns); a variant that asks what is
// checked after the decision, the project level's (projectColumns), whose
// PG, a guest, may create none; one on the archived project, its table's
// (archivedColumns).

var cellIssueNotFound = cell{http.StatusNotFound, "issue.not_found"}

// issueRows are the work items the rows name by id: a column that does not
// see its project answers issue.not_found.
var issueRows = rowsByID{path: "issues", notFound: cellIssueNotFound, find: seeded.issue}

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
		// PM+WA as the member he is, and the workspace admin who is only a
		// guest of the project as its admin (Grant.ProjectAdmin, 9.2); a
		// workspace admin or member who is no member of the project sees it
		// and may not.
		{op: "createIssue", write: true, columns: issueColumns, request: toProject(http.MethodPost, "/issues", newIssue),
			cells: onTheIssues(cellCreated, cellCreated, cellForbidden), check: createsTheIssue},
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
		// The project's admins, members and guests (M4 design 3.4); a guest
		// of a project that does not show its guests every work item, the
		// workspace admin who is only its guest too, finds none another made,
		// as one not there.
		{op: "getIssue", columns: issueColumns, request: toWorkItem(http.MethodGet, "", ""), cells: onAWorkItem(cellOK, cellOK, cellOK, cellOK),
			check: readsTheIssue},
		// An archived project's work items are read as any other's.
		{op: "getIssue", variant: "archived", columns: archivedColumns, request: toWorkItem(http.MethodGet, "", ""),
			cells: issueRows.ofArchived(cellOK, cellForbidden), check: readsTheIssue},
		// Every active member of the workspace finds it, then as getIssue;
		// X, who is none, finds no workspace (M4 design 3.4, 3.7).
		{op: "getIssueByIdentifier", columns: issueColumns, request: byIdentifier,
			cells: inItsWorkspace(onAWorkItem(cellOK, cellOK, cellOK, cellOK)), check: readsTheIssue},
		// An archived project's work items are found as any other's.
		{op: "getIssueByIdentifier", variant: "archived", columns: archivedColumns, request: byIdentifier,
			cells: inItsWorkspace(issueRows.ofArchived(cellOK, cellForbidden)), check: readsTheIssue},
	}
}

// identifierOf is the identifier of the column's work item (issueOf): its
// project's identifier, a hyphen and its number, as WEB-2.
func identifierOf(c caller) string {
	n, _ := numberOf(projectOf(c), issueOf(c))
	for _, p := range matrixProjects {
		if p.key == projectOf(c) {
			return p.identifier + "-" + strconv.Itoa(n)
		}
	}
	return ""
}

// byIdentifier is the request of a row whose callers each read their
// column's work item by its identifier, in their column's workspace.
func byIdentifier(c caller, _ seeded) (string, string, string) {
	return http.MethodGet, "/api/v0/workspaces/" + workspaceOf(c) + "/issues/" + identifierOf(c), ""
}

// inItsWorkspace are cells of a row that names its column's workspace by
// its slug: workspace.not_found for X, and for the archived project's
// column of an account never a member of acme, whom it does not let in;
// every other cell as cells has it.
func inItsWorkspace(cells map[caller]cell) map[caller]cell {
	out := maps.Clone(cells)
	for _, c := range []caller{callerNever, callerRemoved, callerDeleted, callerArchivedNever} {
		if _, ok := out[c]; ok {
			out[c] = cellWorkspaceNotFound
		}
	}
	return out
}

// createsTheIssue: the column's project's next work item (nextNumber), Fix
// the login, in its default state Backlog, of no priority, at a state's
// first sort order, not completed, assigned to no one, without labels,
// made and last changed by the caller.
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
	me, next := s.account(accountOf(c)), nextNumber(projectOf(c))
	if i.ProjectID != s.project(projectOf(c)) || i.SequenceID != next || i.Name != "Fix the login" ||
		i.StateID != s.state(projectOf(c), "Backlog") || i.Priority != "none" || i.SortOrder != 65535 || i.CompletedAt != nil ||
		len(i.AssigneeIDs) != 0 || len(i.LabelIDs) != 0 || i.CreatedByID == nil || *i.CreatedByID != me || i.UpdatedByID == nil ||
		*i.UpdatedByID != me {
		t.Errorf("%s creates %s; want %s's work item %d, Fix the login, in Backlog, none, at 65535, by %s", c, answer, projectOf(c), next, me)
	}
}

// createsTheGivenIssue: the column's project's next work item, Fix, in
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
	pm, ui, next := s.account(callerProjectMember), s.label(projectOf(c), "UI"), nextNumber(projectOf(c))
	if i.SequenceID != next || i.StateID != s.state(projectOf(c), "Done") || i.Priority != "high" || i.SortOrder != 65535 ||
		i.CompletedAt == nil || !slices.Equal(i.AssigneeIDs, []uuid.UUID{pm}) || !slices.Equal(i.LabelIDs, []uuid.UUID{ui}) {
		t.Errorf("%s creates %s; want %s's work item %d in Done, high, at 65535, completed, assigned to %s, labelled %s", c, answer,
			projectOf(c), next, pm, ui)
	}
}
