package bootstrap

import (
	"fmt"
	"net/http"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification"
	notificationdomain "github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A notification's snapshot names its work item's own project and state (M7 P1b spec 2.5, 2.6), each read alone by its
// id: in acme, whose Web, Ops and Docs each have the six default states, bob, assigned a work item of Ops in its In
// Progress state, is sent its notifications with OPS and In Progress, started; not Web's identifier nor Backlog, the
// first of acme's projects and of Ops' states by id. Through the reads newApp completes the snapshot with, Docs'
// triage state is read too, and Docs' identifier once it is archived; Web's In Progress asked of Ops is none, Ops
// asked of another workspace is none, and so is Gone once it is deleted.
func TestTheSnapshotNamesItsOwnProjectAndState(t *testing.T) {
	w := newIssueWorld(t)
	bob := w.ids["bob"]
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	docs := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Docs", "DOCS")
	gone := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Gone", "GONE")
	w.makeMember(t, "bob", ops, shared.RoleMember)
	item := w.newWorkItemIn(t, ops, fmt.Sprintf(`{"name":"Fix the login","state_id":%q,"assignee_ids":[%q]}`, stateID(t, w.pool, ops, "In Progress"), bob))
	waitForFanouts(t, w.pool)
	want := notificationdomain.IssueSnapshot{ID: item, Name: "Fix the login", SequenceID: 1, Identifier: "OPS", StateName: "In Progress",
		StateGroup: "started"}
	_, got := sentNotifications(t, w.pool, item)
	if len(got) == 0 {
		t.Fatal("no notification of the work item; want bob's")
	}
	for _, n := range got {
		if n.Receiver != bob || n.Data.Issue != want {
			t.Errorf("notification to %s of %+v; want bob's of %+v", n.Receiver, n.Data.Issue, want)
		}
	}

	names := notificationProjects{names: project.Provide(w.pool, nil).NotificationNames}
	if s, found, err := names.State(t.Context(), docs, stateID(t, w.pool, docs, "Triage")); err != nil || !found ||
		s != (notification.StateName{Name: "Triage", Group: "triage"}) {
		t.Errorf("State(Docs' Triage) = %+v, %v, %v; want Triage, triage", s, found, err)
	}
	if s, found, err := names.State(t.Context(), ops, stateID(t, w.pool, w.web, "In Progress")); err != nil || found || s != (notification.StateName{}) {
		t.Errorf("State(Web's In Progress, of Ops) = %+v, %v, %v; want none", s, found, err)
	}
	if id, found, err := names.Identifier(t.Context(), uuid.NewV7(), ops); err != nil || found || id != "" {
		t.Errorf("Identifier(Ops, of another workspace) = %q, %v, %v; want none", id, found, err)
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+docs.String()+"/archive", `{}`, http.StatusOK)
	if id, found, err := names.Identifier(t.Context(), w.acme, docs); err != nil || !found || id != "DOCS" {
		t.Errorf("Identifier(Docs, archived) = %q, %v, %v; want DOCS", id, found, err)
	}
	w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+gone.String(), "", http.StatusNoContent)
	if id, found, err := names.Identifier(t.Context(), w.acme, gone); err != nil || found || id != "" {
		t.Errorf("Identifier(Gone, deleted) = %q, %v, %v; want none", id, found, err)
	}
}
