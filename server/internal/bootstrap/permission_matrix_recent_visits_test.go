package bootstrap

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"
)

// The rows of the permission matrix of one's recent visits (M7 P3d spec
// 2.2, 2.5, 2.7, R11), on the workspace level's columns alone: a row with a
// project table's column would make its operation a write on a project
// (writesOnAProject), and a recording is a write of the workspace's, whose
// locks TestRecordingARecentVisitTakesItsLocks pins. recent_visit.read and
// recent_visit.write let the workspace's admin, member and guest in (design
// 3.10); which targets each of them may record and sees listed are the
// projects he reads (Readable) and the work items the guest rule leaves him
// there (spec 2.5). In the prepared data acme's admin and member read none
// of acme's projects (the admin is a member of none, the member's membership
// of the private one is deleted, partingStates), and the guest reads the
// public one, which shows him only his own work items.
//
// A recording names its target alone, its workspace found from it (R7): its
// column's project (projectOf), acme's public one but for the deleted
// workspace's admin, whose gone's is deleted with it; or that project's
// matrixIssue, made by acme's admin. Each of its rows' cells is on a copy of
// its own, as it writes. Every 404 of a recording is the target's own,
// project.not_found or issue.not_found, one value for a target he does not
// see, one in a workspace he is not a member of and one deleted (spec 2.7):
// never's and the removed member's in acme, the deleted workspace's
// admin's in gone. So the public project, recorded as its member once the
// row's prepare makes acme's admin and member its members on the cell's
// copy (prepareDraftPublisher), is 204 to the three; without it, the
// admin's and the member's are 404 as the guest's is 204; and its
// matrixIssue, theirs to see once they are its members, the guest rule
// hides from the guest, 404.
//
// The list is a read, but its row is a write's (spec R11), each cell on a
// copy its prepare changes (visitingThePublicProject): its caller's visits
// of acme's public project and of a work item of it that is none. The
// guest lists his visit of the project alone; the admin and the member, who
// read no project of acme, none of theirs (listsTheVisitsHeSees). The
// callers the workspace is not visible to are workspace.not_found, their
// visits there or not.
func recentVisitMatrixRows() []matrixRow {
	record := func(kind string, target func(caller, seeded) uuid.UUID) func(caller, seeded) (string, string, string) {
		return func(c caller, s seeded) (string, string, string) {
			return http.MethodPost, "/api/v0/me/recent-visits", `{"entity_type":"` + kind + `","entity_id":"` + target(c, s).String() + `"}`
		}
	}
	project := func(c caller, s seeded) uuid.UUID { return s.project(projectOf(c)) }
	adminsItem := func(c caller, s seeded) uuid.UUID { return s.issue(projectOf(c), matrixIssue) }
	recorded := func(admin, member, guest, notFound cell) map[caller]cell {
		return workspaceCells(admin, member, guest, notFound, notFound, notFound)
	}
	return []matrixRow{
		{op: "listRecentVisits", variant: "his visits of the public project", write: true, prepare: visitingThePublicProject,
			request: toWorkspace(http.MethodGet, "/recent-visits", ""), cells: inWorkspace(cellOK, cellOK, cellOK), check: listsTheVisitsHeSees},
		{op: "recordRecentVisit", variant: "a project, as its member", write: true, prepare: prepareDraftPublisher,
			request: record("project", project), cells: recorded(cellNoContent, cellNoContent, cellNoContent, cellProjectNotFound)},
		{op: "recordRecentVisit", variant: "a project, as none of its members", write: true, request: record("project", project),
			cells: recorded(cellProjectNotFound, cellProjectNotFound, cellNoContent, cellProjectNotFound)},
		{op: "recordRecentVisit", variant: "a work item acme's admin made", write: true, prepare: prepareDraftPublisher,
			request: record("issue", adminsItem), cells: recorded(cellNoContent, cellNoContent, cellIssueNotFound, cellIssueNotFound)},
	}
}

// visitingThePublicProject is the list's prepare: the cell's caller's
// visits of acme's public project and of a work item of it whose id is no
// work item's, written directly on the cell's copy at url (seedRecentVisits),
// over a pool it closes before the cell's app starts.
func visitingThePublicProject(t *testing.T, c caller, s seeded, url string) {
	t.Helper()
	pool := openPool(t, url)
	defer pool.Close()
	seedRecentVisits(t, pool, s.project("acme/public"), s.account(accountOf(c)))
}

// recentVisitAnswer is a recent visit as an answer of the API holds it.
type recentVisitAnswer struct {
	ID         uuid.UUID              `json:"id"`
	EntityType string                 `json:"entity_type"`
	EntityID   uuid.UUID              `json:"entity_id"`
	VisitedAt  time.Time              `json:"visited_at"`
	Project    recentVisitAnswerOwner `json:"project"`
	Issue      json.RawMessage        `json:"issue"`
}

// recentVisitAnswerOwner is the project of a recent visit as an answer
// holds it.
type recentVisitAnswerOwner struct {
	ID         uuid.UUID      `json:"id"`
	Name       string         `json:"name"`
	Identifier string         `json:"identifier"`
	LogoProps  map[string]any `json:"logo_props"`
}

// listsTheVisitsHeSees: the guest's list holds his visit of acme's public
// project alone, which he reads, its work item null and its project as
// seeded, with no icon; his visit of a work item that is none is no one's
// to see. The admin's and the member's are empty: they read no project of
// acme. Each visit listed has an id and a moment of its own.
func listsTheVisitsHeSees(t *testing.T, c caller, s seeded, answer string) {
	var list struct {
		Data []recentVisitAnswer `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	want := []recentVisitAnswer{}
	if c == callerGuest {
		public := s.project("acme/public")
		want = append(want, recentVisitAnswer{EntityType: "project", EntityID: public, Issue: json.RawMessage("null"),
			Project: recentVisitAnswerOwner{ID: public, Name: "Web", Identifier: "WEB", LogoProps: map[string]any{}}})
	}
	for n := range min(len(list.Data), len(want)) {
		v := list.Data[n]
		if v.ID == uuid.Nil() || v.VisitedAt.IsZero() {
			t.Errorf("%s's recent visits in %s = %s; want each with its id and its moment", c, workspaceOf(c), answer)
		}
		want[n].ID, want[n].VisitedAt = v.ID, v.VisitedAt
	}
	if list.Data == nil || !reflect.DeepEqual(list.Data, want) {
		t.Errorf("%s's recent visits in %s = %s; want %+v", c, workspaceOf(c), answer, want)
	}
}
