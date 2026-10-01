package bootstrap

import (
	"net/http"
	"testing"
	"uuid"
)

// The project module's rows of the permission matrix (M3 design 9.2).

func projectMatrixRows() []matrixRow {
	return []matrixRow{
		{op: "createProject", write: true, request: toWorkspace(http.MethodPost, "/projects", `{"name":"New","identifier":"new"}`),
			cells: inWorkspace(cellCreated, cellCreated, cellForbidden), check: createsItsProject},
		// A lead who is no member of the workspace is refused after the
		// decision (M3 design 3.6 convention 3): the guest is still refused
		// as a guest, and learns nothing of the lead.
		{op: "createProject", variant: "a lead who is no member", write: true,
			request: toWorkspace(http.MethodPost, "/projects", `{"name":"New","identifier":"NEW","project_lead_id":"`+uuid.Nil().String()+`"}`),
			cells:   inWorkspace(cellValidationFailed, cellValidationFailed, cellForbidden)},
	}
}

// createsItsProject: the project is created as asked, with its caller as
// its admin and only member.
func createsItsProject(t *testing.T, c caller, _ seeded, answer string) {
	var p struct {
		Identifier string      `json:"identifier"`
		MemberRole *int        `json:"member_role"`
		MemberIDs  []uuid.UUID `json:"member_ids"`
	}
	decodeAnswer(t, answer, &p)
	if p.Identifier != "NEW" || p.MemberRole == nil || *p.MemberRole != 20 || len(p.MemberIDs) != 1 {
		t.Errorf("%s creates %s; want NEW, with him its admin and only member", c, answer)
	}
}
