package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"uuid"
)

// The rows of the permission matrix of a project's states (M3 design 9.2):
// listing and creating them, under the project of each column, the writes
// on one, naming it by its id (/states/{state_id}) among its column's
// project's seeded states, those of matrixStates, and listing a
// workspace's, under the workspace of each column.

var (
	cellStateNameTaken   = cell{http.StatusConflict, "project.state_name_taken"}
	cellStateNotFound    = cell{http.StatusNotFound, "project.state_not_found"}
	cellStateLastInGroup = cell{http.StatusConflict, "project.state_last_in_group"}
	cellStateDefault     = cell{http.StatusConflict, "project.state_default"}
)

// ofState are the cells of a row of a write on a state: the answers of PA,
// PM, PG, PM+WA, WA- and WM-公, and project.state_not_found for the
// columns that do not see their project: WM-私, WG-, P-前 and X, whose
// state in gone's project is deleted with it.
func ofState(pa, pm, pg, pmwa, wa, wm cell) map[caller]cell {
	return map[caller]cell{callerProjectAdmin: pa, callerProjectMember: pm, callerProjectGuest: pg, callerMemberAndAdmin: pmwa,
		callerAdminOnly: wa, callerMemberPublic: wm, callerMemberPrivate: cellStateNotFound, callerGuestOnly: cellStateNotFound,
		callerBefore: cellStateNotFound, callerNever: cellStateNotFound, callerRemoved: cellStateNotFound, callerDeleted: cellStateNotFound}
}

// ofArchivedState are the cells of a row of a write on a state of the
// archived project: the answers of PA and of the workspace's member, who
// sees the project, and project.state_not_found for X, who does not.
func ofArchivedState(pa, wm cell) map[caller]cell {
	return map[caller]cell{callerArchivedAdmin: pa, callerArchivedMember: wm, callerArchivedNever: cellStateNotFound}
}

// toState is the request of a row whose callers each send method, with
// body, to the state name of their column's project, after its id.
func toState(method, after, name, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return method, "/api/v0/states/" + s.state(projectOf(c), name).String() + after, body
	}
}

// newState is the body of the state the rows create: QA, of the completed
// group, a name no seeded state has.
const newState = `{"name":"QA","color":"#0EA5E9","group":"completed"}`

// listedStates are the names of each matrix project's states as a list
// answers them: matrixStates by sequence, the triage state left out.
var listedStates = []string{"Backlog", "Todo", "In Progress", "Review", "Done", "Cancelled"}

func stateMatrixRows() []matrixRow {
	return []matrixRow{
		// Every active member of the project (M3 design 9.2): PM+WA as its
		// member, and not WA-, who is none.
		{op: "listStates", columns: projectColumns, request: toProject(http.MethodGet, "/states", ""),
			cells: ofProject(cellOK, cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: listsTheStates},
		// An archived project lists none (M3 design 3.17), to who may list.
		{op: "listStates", variant: "archived", columns: archivedColumns, request: toProject(http.MethodGet, "/states", ""),
			cells: ofArchived(cellOK, cellForbidden), check: listsTheStates},
		// The project's admins, and its members who are the workspace's
		// admins (M3 design 3.4): not its guests, whom Plane lets change
		// states.
		{op: "createState", write: true, columns: projectColumns, request: toProject(http.MethodPost, "/states", newState),
			cells: ofProject(cellCreated, cellForbidden, cellForbidden, cellCreated, cellForbidden, cellForbidden), check: createsTheState},
		// The name's 409 comes after the decision: who may not create states
		// learns nothing of the project's.
		{op: "createState", variant: "a name taken", write: true, columns: projectColumns,
			request: toProject(http.MethodPost, "/states", `{"name":"Todo","color":"#0EA5E9","group":"completed"}`),
			cells:   ofProject(cellStateNameTaken, cellForbidden, cellForbidden, cellStateNameTaken, cellForbidden, cellForbidden)},
		// An archived project's states are created as any other's (M3 design
		// 3.19).
		{op: "createState", variant: "archived", write: true, columns: archivedColumns, request: toProject(http.MethodPost, "/states", newState),
			cells: ofArchived(cellCreated, cellForbidden), check: createsTheState},
		// As createState: Todo renamed.
		{op: "updateState", write: true, columns: projectColumns, request: toState(http.MethodPatch, "", "Todo", `{"name":"Next"}`),
			cells: ofState(cellOK, cellForbidden, cellForbidden, cellOK, cellForbidden, cellForbidden), check: renamesTheState},
		// Todo, the only state of its group, moved to another: the 409 comes
		// after the decision (M3 design 6.7).
		{op: "updateState", variant: "the last of its group moved", write: true, columns: projectColumns,
			request: toState(http.MethodPatch, "", "Todo", `{"group":"started"}`),
			cells:   ofState(cellStateLastInGroup, cellForbidden, cellForbidden, cellStateLastInGroup, cellForbidden, cellForbidden)},
		{op: "updateState", variant: "a name taken", write: true, columns: projectColumns, request: toState(http.MethodPatch, "", "Todo", `{"name":"Done"}`),
			cells: ofState(cellStateNameTaken, cellForbidden, cellForbidden, cellStateNameTaken, cellForbidden, cellForbidden)},
		// The intake's triage state is none of the states (M3 design 3.17):
		// every column's 404, its project's admins' too.
		{op: "updateState", variant: "the triage state", write: true, columns: projectColumns,
			request: toState(http.MethodPatch, "", "Triage", `{"name":"Next"}`), cells: ofState(cellStateNotFound, cellStateNotFound,
				cellStateNotFound, cellStateNotFound, cellStateNotFound, cellStateNotFound)},
		// A value refused before the state is looked at (M3 design 6.7): the
		// same 422 in every column, on the triage state too.
		{op: "updateState", variant: "a value refused", write: true, columns: projectColumns,
			request: toState(http.MethodPatch, "", "Triage", `{"name":""}`), cells: func() map[caller]cell {
				cells := map[caller]cell{}
				for _, c := range projectColumns {
					cells[c] = cellValidationFailed
				}
				return cells
			}(), refusal: "name too_short"},
		{op: "updateState", variant: "archived", write: true, columns: archivedColumns, request: toState(http.MethodPatch, "", "Todo", `{"name":"Next"}`),
			cells: ofArchivedState(cellOK, cellForbidden), check: renamesTheState},
		// As createState: Review deleted, In Progress kept in its group.
		{op: "deleteState", write: true, columns: projectColumns, request: toState(http.MethodDelete, "", "Review", ""),
			cells: ofState(cellNoContent, cellForbidden, cellForbidden, cellNoContent, cellForbidden, cellForbidden)},
		// The default, and Done, the only state of its group: each 409 comes
		// after the decision (M3 design 3.17). Backlog is both the default and
		// its group's only state, and answers project.state_default: the
		// guarded deletion comes before the group's count (P7a spec 3 item
		// 13).
		{op: "deleteState", variant: "the default", write: true, columns: projectColumns, request: toState(http.MethodDelete, "", "Backlog", ""),
			cells: ofState(cellStateDefault, cellForbidden, cellForbidden, cellStateDefault, cellForbidden, cellForbidden)},
		{op: "deleteState", variant: "the last of its group", write: true, columns: projectColumns, request: toState(http.MethodDelete, "", "Done", ""),
			cells: ofState(cellStateLastInGroup, cellForbidden, cellForbidden, cellStateLastInGroup, cellForbidden, cellForbidden)},
		{op: "deleteState", variant: "the triage state", write: true, columns: projectColumns, request: toState(http.MethodDelete, "", "Triage", ""),
			cells: ofState(cellStateNotFound, cellStateNotFound, cellStateNotFound, cellStateNotFound, cellStateNotFound, cellStateNotFound)},
		{op: "deleteState", variant: "archived", write: true, columns: archivedColumns, request: toState(http.MethodDelete, "", "Review", ""),
			cells: ofArchivedState(cellNoContent, cellForbidden)},
		// As createState: Todo made the default.
		{op: "markDefaultState", write: true, columns: projectColumns, request: toState(http.MethodPost, "/mark-default", "Todo", ""),
			cells: ofState(cellNoContent, cellForbidden, cellForbidden, cellNoContent, cellForbidden, cellForbidden)},
		{op: "markDefaultState", variant: "the triage state", write: true, columns: projectColumns,
			request: toState(http.MethodPost, "/mark-default", "Triage", ""), cells: ofState(cellStateNotFound, cellStateNotFound, cellStateNotFound,
				cellStateNotFound, cellStateNotFound, cellStateNotFound)},
		{op: "markDefaultState", variant: "archived", write: true, columns: archivedColumns,
			request: toState(http.MethodPost, "/mark-default", "Todo", ""), cells: ofArchivedState(cellNoContent, cellForbidden)},
		// Every active member of the workspace (M3 design 9.2), each the
		// states of the projects he is a member of.
		{op: "listWorkspaceStates", request: toWorkspace(http.MethodGet, "/states", ""), cells: inWorkspace(cellOK, cellOK, cellOK),
			check: listsTheWorkspaceStates},
	}
}

// listsTheStates: the column's project's states but its triage state, by
// sequence: Review after In Progress; none of the archived project's.
func listsTheStates(t *testing.T, c caller, s seeded, answer string) {
	var list struct {
		Data []struct {
			ID        uuid.UUID `json:"id"`
			ProjectID uuid.UUID `json:"project_id"`
			Name      string    `json:"name"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	var got []string
	for _, st := range list.Data {
		if st.ProjectID != s.project(projectOf(c)) || st.ID != s.state(projectOf(c), st.Name) {
			t.Errorf("%s lists %s, not its project's", c, answer)
		}
		got = append(got, st.Name)
	}
	want := listedStates
	if projectOf(c) == "acme/archived" {
		want = nil
	}
	if !slices.Equal(got, want) {
		t.Errorf("%s lists %q, want %q", c, got, want)
	}
}

// createsTheState: QA in the column's project, after its Cancelled, the
// greatest sequence of its states but its triage state, and not its
// default.
func createsTheState(t *testing.T, c caller, s seeded, answer string) {
	var st struct {
		ProjectID uuid.UUID `json:"project_id"`
		Name      string    `json:"name"`
		Group     string    `json:"group"`
		Default   bool      `json:"default"`
		Sequence  float64   `json:"sequence"`
	}
	decodeAnswer(t, answer, &st)
	if st.ProjectID != s.project(projectOf(c)) || st.Name != "QA" || st.Group != "completed" || st.Default || st.Sequence != 70000 {
		t.Errorf("%s creates %s; want QA in %s, completed, at 70000, not the default", c, answer, projectOf(c))
	}
}

// renamesTheState: the column's project's Todo, renamed Next, as stored.
func renamesTheState(t *testing.T, c caller, s seeded, answer string) {
	var st struct {
		ID        uuid.UUID `json:"id"`
		ProjectID uuid.UUID `json:"project_id"`
		Name      string    `json:"name"`
		Group     string    `json:"group"`
	}
	decodeAnswer(t, answer, &st)
	if st.ID != s.state(projectOf(c), "Todo") || st.ProjectID != s.project(projectOf(c)) || st.Name != "Next" || st.Group != "unstarted" {
		t.Errorf("%s renames %s; want %s's Todo, unstarted, named Next", c, answer, projectOf(c))
	}
}

// listsTheWorkspaceStates: the states of acme's unarchived projects that
// the column's account is an active member of, by project id, then sequence:
// none for the admin and the member, members of none of them, and the
// public and the private project's for the guest, PG's account.
func listsTheWorkspaceStates(t *testing.T, c caller, s seeded, answer string) {
	var list struct {
		Data []struct {
			ID uuid.UUID `json:"id"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	var got, want []uuid.UUID
	for _, st := range list.Data {
		got = append(got, st.ID)
	}
	if c == callerGuest {
		for _, key := range byProjectID(s, "acme/public", "acme/private") {
			for _, name := range listedStates {
				want = append(want, s.state(key, name))
			}
		}
	}
	if !slices.Equal(got, want) {
		t.Errorf("%s lists %v, want %v", c, got, want)
	}
}

// byProjectID are the projects keys names, in the order of their ids.
func byProjectID(s seeded, keys ...string) []string {
	return slices.SortedFunc(slices.Values(keys), func(a, b string) int {
		pa, pb := s.project(a), s.project(b)
		return slices.Compare(pa[:], pb[:])
	})
}
