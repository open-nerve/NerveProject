package bootstrap

import (
	"net/http"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
)

// favoritesDeletionWindow is the favorites' deletion window (M7 P3c spec
// 2.6, R14): a project's deletion, and a workspace's, enqueue the deletion
// of their favorites behind their views', which the test holds
// (holdNavigationDeletion, or a row lock on the workspace's own view), so
// that the favorites stay undeleted; meanwhile alice, whose they are, lists
// none of them, a change and a deletion of each answer as an id no
// favorite has, and a favorite of each target in acme is the one 422 of a
// target she may not favor, byte for byte: what is deleted is not seen,
// whatever the job has done. Released, the job deletes them at the
// deletion's moment, by its deleter. The start-time runs have run first,
// so none queues on a lock the test holds (P2a review §4).
func favoritesDeletionWindow(t *testing.T, w favoritesWorld) {
	waitForStartupRuns(t, w.pool)
	t.Run("project", func(t *testing.T) {
		dropped := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Dropped", "DRP")
		w.turnCyclesOn(t, dropped)
		w.changeProject(t, dropped, `{"issue_views_view":true}`)
		// The job waits for Held, the first of Dropped's views by id
		// (holdNavigationDeletion); a favorite of v takes v's row FOR SHARE
		// (spec 2.8), which no one holds meanwhile.
		w.newView(t, "alice", projectViewsPath(dropped), `{"name":"Held"}`)
		v, _ := w.newView(t, "alice", projectViewsPath(dropped), `{"name":"Dropped's"}`)
		var c cycleAnswer
		decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+dropped.String()+"/cycles", `{"name":"Dropped's"}`, http.StatusCreated), &c)
		targets := []string{favoriteBody("project", dropped), favoriteBody("cycle", c.ID), favoriteBody("view", v.ID)}
		favorites := make([]uuid.UUID, len(targets))
		for n, body := range targets {
			favorites[n] = w.favorID(t, "alice", "acme", body)
		}
		holder := holdNavigationDeletion(t, w.pool, dropped)
		w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+dropped.String(), "", http.StatusNoContent)
		w.favoritesAnsweredAsAbsent(t, "acme", favorites, targets)
		requestRelease(t, holder)
		waitNavigationDeletion(t, w.pool, "project", dropped)
		at, by := stampOf(t, w.pool, "projects", "deleted_at", dropped)
		favoritesDeletedAt(t, w.pool, at, by, favorites...)
	})
	t.Run("workspace", func(t *testing.T) {
		var doomed struct{ ID uuid.UUID }
		decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Doomed","slug":"doomed"}`, http.StatusCreated), &doomed)
		gone := createdProject(t, w.contract, w.base, w.tokens["alice"], "doomed", "Gone", "GNE")
		w.changeProject(t, gone, `{"issue_views_view":true}`)
		pv, _ := w.newView(t, "alice", projectViewsPath(gone), `{"name":"Gone's"}`)
		wv, _ := w.newView(t, "alice", workspaceViewsPath("doomed"), `{"name":"Doomed's"}`)
		folder := w.favorID(t, "alice", "doomed", folderBody("Doomed's"))
		targets := []string{favoriteBody("project", gone), favoriteBody("view", pv.ID), favoriteBody("view", wv.ID)}
		inProject := []uuid.UUID{w.favorID(t, "alice", "doomed", targets[0]), w.favorID(t, "alice", "doomed", favoriteBody("view", pv.ID, folder))}
		ofWorkspace := []uuid.UUID{folder, w.favorID(t, "alice", "doomed", favoriteBody("view", wv.ID, folder))}
		holders := []pgx.Tx{holdNavigationDeletion(t, w.pool, gone), holding(t, w.pool, "SELECT 1 FROM issue_views WHERE id=$1 FOR UPDATE", wv.ID)}
		w.send(t, "alice", http.MethodDelete, "/api/v0/workspaces/doomed", "", http.StatusNoContent)
		w.answeredAsAbsent(t, "alice", http.MethodGet, favoritesPath("doomed"), favoritesPath("absent"), "", "workspace.not_found")
		w.answeredAsAbsent(t, "alice", http.MethodPost, favoritesPath("doomed"), favoritesPath("absent"), folderBody("New"), "workspace.not_found")
		w.favoritesAnsweredAsAbsent(t, "acme", append(inProject, ofWorkspace...), targets)
		for _, holder := range holders {
			requestRelease(t, holder)
		}
		waitNavigationDeletion(t, w.pool, "project", gone)
		waitNavigationDeletion(t, w.pool, "workspace", doomed.ID)
		at, by := stampOf(t, w.pool, "projects", "deleted_at", gone)
		favoritesDeletedAt(t, w.pool, at, by, inProject...)
		at, by = stampOf(t, w.pool, "workspaces", "deleted_at", doomed.ID)
		favoritesDeletedAt(t, w.pool, at, by, ofWorkspace...)
	})
}

// favoritesAnsweredAsAbsent checks, inside a deletion window, that alice's
// favorites ids are undeleted rows she does not list in acme, each of
// whose change and deletion answers as an id no favorite has; and that a
// favorite in the workspace slug of each target, of bodies, is the one 422
// of a target she may not favor; the rows still undeleted after.
func (w favoritesWorld) favoritesAnsweredAsAbsent(t *testing.T, slug string, ids []uuid.UUID, bodies []string) {
	t.Helper()
	favoritesLive(t, w.pool, ids...)
	if got := w.listedFavoriteIDs(t, "alice", "acme"); containsAny(got, ids...) {
		t.Errorf("alice lists %v; want none of %v", got, ids)
	}
	for _, id := range ids {
		w.favoriteAnsweredAsAbsent(t, "alice", id)
	}
	w.refusedAlike(t, "alice", slug, w.noTarget(t, "alice", slug), bodies...)
	favoritesLive(t, w.pool, ids...)
}
