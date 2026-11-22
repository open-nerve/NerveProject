package bootstrap

import (
	"fmt"
	"net/http"
	"testing"
	"uuid"

	navigationdomain "github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
)

// capsAtTwenty is the cap's cell (spec 2.5, 2.8, D35, D43): carl, acme's
// and Web's member, records his visit of Own in his own workspace Carl's;
// then in acme his visits of Cap 0 and Cap 1, two of Web's work items, of
// pending, one of its intake items, which he records and does not list
// (keepsTriageOutUntilAccepted), and of Cap 2 to Cap 18, 20 in all; then
// Cap 0 again, made first and now visited last. He has 20 undeleted visits
// in acme and lists 19. His 21st target, Web, answers 204 and leaves him
// 20: the one visited first, Cap 1's, is deleted for good, no row of it
// left, not Cap 0's, made first; pending's, hidden, takes its place among
// the 20 and is kept; he lists 19, Web's first. His visit in Carl's is as
// it was, byte for byte.
func capsAtTwenty(t *testing.T, w recentVisitsWorld, pending uuid.UUID) {
	carl := w.ids["carl"]
	w.visit(t, "carl", "project", w.own)
	elsewhere := visitsText(t, w.pool, liveVisit(t, w.pool, carl, w.own))
	items := make([]uuid.UUID, 19)
	for n := range items {
		items[n] = w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Cap %d"}`, n))
	}
	for _, id := range append([]uuid.UUID{items[0], items[1], pending}, items[2:]...) {
		w.visit(t, "carl", "issue", id)
	}
	first, earliest := liveVisit(t, w.pool, carl, items[0]), liveVisit(t, w.pool, carl, items[1])
	w.visit(t, "carl", "issue", items[0])
	if n, listed := w.undeletedVisits(t, carl), len(w.listedVisits(t, "carl", "acme", "")); n != navigationdomain.RecentVisitCap || listed != 19 {
		t.Fatalf("carl has %d undeleted visits in acme and lists %d; want 20 and 19, pending's hidden", n, listed)
	}
	w.visit(t, "carl", "project", w.web)
	if n := w.undeletedVisits(t, carl); n != navigationdomain.RecentVisitCap {
		t.Errorf("carl's 21st target leaves him %d undeleted visits in acme; want 20", n)
	}
	if gone := visitRowsWhere(t, w.pool, "id = $1 OR (user_id = $2 AND entity_identifier = $3)", earliest.id, carl, items[1]); len(gone) != 0 {
		t.Errorf("carl's visit of Cap 1, visited first, past the cap: %v; want it deleted for good", gone)
	}
	if made := liveVisit(t, w.pool, carl, items[0]); made.id != first.id {
		t.Errorf("carl's visit of Cap 0, made first: %s; want %s kept", made, first)
	}
	liveVisit(t, w.pool, carl, pending)
	want := []visitRow{liveVisit(t, w.pool, carl, w.web), liveVisit(t, w.pool, carl, items[0])}
	for n := len(items) - 1; n >= 2; n-- {
		want = append(want, liveVisit(t, w.pool, carl, items[n]))
	}
	w.listsVisits(t, "carl", "acme", "", want...)
	if got := visitsText(t, w.pool, liveVisit(t, w.pool, carl, w.own)); got != elsewhere {
		t.Errorf("carl's visit in Carl's:\n%s\nwas\n%s", got, elsewhere)
	}
}

// visitsDeletionWindow is the recent visits' deletion window (spec 2.6,
// 2.7, R10, m4): a project's deletion, and a workspace's, enqueue the
// deletion of their visits behind their views', which the test holds
// (holdNavigationDeletion: the project's one view, its views turned on, a
// view none of the visits names), so that alice's visits stay undeleted;
// meanwhile her recording of each target answers as an id that never was,
// byte for byte, writing nothing: what is deleted is not seen, whatever
// the job has done. Each cell's deletion checks her list in its own
// workspace: acme's lists neither the deleted project nor its work item;
// the deleted workspace's answers as a workspace that is none, byte for
// byte. Released, the job deletes them at the project's deletion's moment,
// by its deleter. The start-time runs have run first, so none queues on a
// lock the test holds (P2a review §4).
func visitsDeletionWindow(t *testing.T, w recentVisitsWorld) {
	alice := []uuid.UUID{w.ids["alice"]}
	waitForStartupRuns(t, w.pool)
	window := func(t *testing.T, slug, name, identifier string, deletion func(project, item uuid.UUID)) {
		project := createdProject(t, w.contract, w.base, w.tokens["alice"], slug, name, identifier)
		w.changeProject(t, project, `{"issue_views_view":true}`)
		w.newView(t, "alice", projectViewsPath(project), `{"name":"Held"}`)
		item := w.newWorkItemIn(t, project, `{"name":"`+name+`'s"}`)
		w.visit(t, "alice", "project", project)
		w.visit(t, "alice", "issue", item)
		visits := liveVisits(t, w.pool, alice, project, item)
		kept := visitsText(t, w.pool, visits...)
		holder := holdNavigationDeletion(t, w.pool, project)
		deletion(project, item)
		w.visitAnsweredAsAbsent(t, "alice", "project", project)
		w.visitAnsweredAsAbsent(t, "alice", "issue", item)
		if got := visitsText(t, w.pool, visits...); got != kept {
			t.Errorf("alice's visits under %s, its deletion held:\n%s\nwere\n%s", name, got, kept)
		}
		requestRelease(t, holder)
		waitNavigationDeletion(t, w.pool, "project", project)
		at, by := stampOf(t, w.pool, "projects", "deleted_at", project)
		visitsDeletedAt(t, w.pool, at, by, visits...)
	}
	t.Run("project", func(t *testing.T) {
		window(t, "acme", "Dropped", "DRP", func(project, item uuid.UUID) {
			w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+project.String(), "", http.StatusNoContent)
			if listed := w.listedTargets(t, "alice", "acme"); containsAny(listed, project, item) {
				t.Errorf("alice lists %v in acme; want neither Dropped nor its work item", listed)
			}
		})
	})
	t.Run("workspace", func(t *testing.T) {
		w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Doomed","slug":"doomed"}`, http.StatusCreated)
		window(t, "doomed", "Gone", "GNE", func(uuid.UUID, uuid.UUID) {
			w.send(t, "alice", http.MethodDelete, "/api/v0/workspaces/doomed", "", http.StatusNoContent)
			w.answeredAsAbsent(t, "alice", http.MethodGet, recentVisitsPath("doomed"), recentVisitsPath("absent"), "", "workspace.not_found")
		})
	})
}
