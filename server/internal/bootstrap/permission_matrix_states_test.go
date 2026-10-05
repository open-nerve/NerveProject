package bootstrap

import (
	"net/http"
	"testing"
	"uuid"
)

// The rows of the permission matrix of a project's states (M3 design 9.2):
// creating them, under the project of each column, its seeded states those
// of matrixStates.

var cellStateNameTaken = cell{http.StatusConflict, "project.state_name_taken"}

// newState is the body of the state the rows create: QA, of the completed
// group, a name no seeded state has.
const newState = `{"name":"QA","color":"#0EA5E9","group":"completed"}`

func stateMatrixRows() []matrixRow {
	return []matrixRow{
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
