package bootstrap

import (
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// viewsDeletionWindow is the views' deletion window (M7 P3b spec 2.7, R6):
// a project's deletion, and a workspace's, enqueue the deletion of their
// views, which the test holds (holdNavigationDeletion, or a row lock on
// the workspace's own view) so that the views stay undeleted; meanwhile
// every operation on them answers alice, who made them and would be
// answered 200, 201 or 204, as on an id, a project or a slug that never
// was, byte for byte: what is deleted is not seen, whatever the job has
// done. Released, the job deletes them at the deletion's moment, by its
// deleter. The start-time runs have run first, so none queues on a lock
// the test holds (P2a review §4).
func viewsDeletionWindow(t *testing.T, w issueWorld) {
	t.Helper()
	waitForStartupRuns(t, w.pool)
	t.Run("project", func(t *testing.T) {
		dropped := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Dropped", "DRP")
		w.changeProject(t, dropped, `{"issue_views_view":true}`)
		v, _ := w.newView(t, "alice", projectViewsPath(dropped), `{"name":"Dropped's"}`)
		holder := holdNavigationDeletion(t, w.pool, dropped)
		w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+dropped.String(), "", http.StatusNoContent)
		undeleted(t, w.pool, v.ID)
		w.viewAnsweredAsAbsent(t, v.ID)
		w.viewsAnsweredAsAbsent(t, projectViewsPath(dropped), projectViewsPath(uuid.NewV7()), "project.not_found")
		undeleted(t, w.pool, v.ID)
		requestRelease(t, holder)
		waitNavigationDeletion(t, w.pool, "project", dropped)
		deletedWith(t, w.pool, v.ID, "projects", dropped)
	})
	t.Run("workspace", func(t *testing.T) {
		var doomed struct{ ID uuid.UUID }
		decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Doomed","slug":"doomed"}`, http.StatusCreated), &doomed)
		gone := createdProject(t, w.contract, w.base, w.tokens["alice"], "doomed", "Gone", "GNE")
		w.changeProject(t, gone, `{"issue_views_view":true}`)
		pv, _ := w.newView(t, "alice", projectViewsPath(gone), `{"name":"Gone's"}`)
		wv, _ := w.newView(t, "alice", workspaceViewsPath("doomed"), `{"name":"Doomed's"}`)
		holders := []pgx.Tx{holdNavigationDeletion(t, w.pool, gone), holding(t, w.pool, "SELECT 1 FROM issue_views WHERE id=$1 FOR UPDATE", wv.ID)}
		w.send(t, "alice", http.MethodDelete, "/api/v0/workspaces/doomed", "", http.StatusNoContent)
		undeleted(t, w.pool, pv.ID, wv.ID)
		w.viewAnsweredAsAbsent(t, pv.ID)
		w.viewAnsweredAsAbsent(t, wv.ID)
		w.viewsAnsweredAsAbsent(t, workspaceViewsPath("doomed"), workspaceViewsPath("absent"), "workspace.not_found")
		w.viewsAnsweredAsAbsent(t, projectViewsPath(gone), projectViewsPath(uuid.NewV7()), "project.not_found")
		undeleted(t, w.pool, pv.ID, wv.ID)
		for _, holder := range holders {
			requestRelease(t, holder)
		}
		waitNavigationDeletion(t, w.pool, "project", gone)
		waitNavigationDeletion(t, w.pool, "workspace", doomed.ID)
		deletedWith(t, w.pool, pv.ID, "projects", gone)
		deletedWith(t, w.pool, wv.ID, "workspaces", doomed.ID)
	})
}

// viewAnsweredAsAbsent checks that alice's read, change and deletion of
// the view id answer as an id that never was.
func (w issueWorld) viewAnsweredAsAbsent(t *testing.T, id uuid.UUID) {
	t.Helper()
	absent := viewPath(uuid.NewV7())
	for _, op := range []struct{ method, body string }{{http.MethodGet, ""}, {http.MethodPatch, `{"name":"Renamed"}`}, {http.MethodDelete, ""}} {
		w.answeredAsAbsent(t, "alice", op.method, viewPath(id), absent, op.body, "navigation.view_not_found")
	}
}

// viewsAnsweredAsAbsent checks that alice's list of the views at path and
// her creation of one there answer as at absent, with code.
func (w issueWorld) viewsAnsweredAsAbsent(t *testing.T, path, absent, code string) {
	t.Helper()
	w.answeredAsAbsent(t, "alice", http.MethodGet, path, absent, "", code)
	w.answeredAsAbsent(t, "alice", http.MethodPost, path, absent, `{"name":"New"}`, code)
}

// undeleted checks that each of the views ids is not deleted.
func undeleted(t *testing.T, pool *pgxpool.Pool, ids ...uuid.UUID) {
	t.Helper()
	for _, id := range ids {
		if r := viewRowOf(t, pool, id); r.deletedAt != nil {
			t.Fatalf("the view %s is deleted at %s inside the window", id, r.deletedAt)
		}
	}
}

// deletedWith checks that the view id is deleted as its parent, the row
// id of the table parents, was: deleted and last changed at its deletion's
// moment, by its deleter (spec 2.7).
func deletedWith(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, parents string, parent uuid.UUID) {
	t.Helper()
	var at time.Time
	var by uuid.UUID
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at, updated_by_id FROM "+parents+" WHERE id = $1 AND deleted_at IS NOT NULL", parent).
		Scan(&at, &by); err != nil {
		t.Fatal(err)
	}
	if r := viewRowOf(t, pool, id); r.deletedAt == nil || !r.deletedAt.Equal(at) || !r.updatedAt.Equal(at) || r.updatedBy != by {
		t.Fatalf("the view %s: %+v; want deleted at %s by %s", id, r, at, by)
	}
}
