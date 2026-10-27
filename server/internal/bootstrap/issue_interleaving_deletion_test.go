package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// deletesAcme is alice's deletion of acme through the workspace module's
// route.
func deletesAcme(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
	return w.sending(e, newWorkspaceRoute(t, w.pool, tx), "alice", http.MethodDelete, "/api/v0/workspaces/acme", "")
}

// deletesIssue is name's deletion of Web's one work item (workItem), which
// sends no body: as an itemWrite's write, it ignores the body it is given.
func deletesIssue(name string, _ func(*testing.T, issueWorld) string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodDelete, "/api/v0/issues/"+w.workItem(t).String(), "")
	}
}

// fullBody is the world's fullIssue, as createsIssue takes a body.
func fullBody(_ *testing.T, w issueWorld) string {
	return w.fullIssue()
}

// deletedWith checks the deletion of the row id of table, the deleted work
// item's, project's or workspace's: at a moment no earlier than since,
// with the rows of Web's work-item side, n of them, each deleted and last
// written at that moment, by alice, and not before it was made.
func (w issueWorld) deletedWith(t *testing.T, table string, id uuid.UUID, since time.Time, n int) {
	t.Helper()
	var at time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM "+table+" WHERE id = $1", id).Scan(&at); err != nil || at.Before(since) {
		t.Errorf("the row of %s deleted at %v (%v); want no earlier than %v", table, at, err, since)
	}
	switch table {
	case "projects":
		waitProjectDeletion(t, w.pool, id)
	case "workspaces":
		waitWorkspaceDeletion(t, w.pool, id)
	}
	var all, hers int
	if err := w.pool.QueryRow(pgtest.Soon(t), `
		SELECT count(*), count(*) FILTER (WHERE r.deleted_at = d.at AND r.updated_at = d.at AND r.updated_by_id = $3 AND r.created_at <= d.at)
		FROM (SELECT deleted_at, updated_at, updated_by_id, created_at FROM issues WHERE project_id = $2
			UNION ALL SELECT deleted_at, updated_at, updated_by_id, created_at FROM issue_assignees WHERE project_id = $2
			UNION ALL SELECT deleted_at, updated_at, updated_by_id, created_at FROM issue_labels WHERE project_id = $2
			UNION ALL SELECT deleted_at, updated_at, updated_by_id, created_at FROM issue_relations WHERE project_id = $2
			UNION ALL SELECT deleted_at, updated_at, updated_by_id, created_at FROM issue_subscribers WHERE project_id = $2
			UNION ALL SELECT deleted_at, updated_at, updated_by_id, created_at FROM issue_activities WHERE project_id = $2) r,
			(SELECT deleted_at AS at FROM `+table+` WHERE id = $1) d`, id, w.web, w.ids["alice"]).Scan(&all, &hers); err != nil {
		t.Fatal(err)
	}
	if all != n || hers != n {
		t.Errorf("Web's work-item side has %d rows, %d of them deleted with the row of %s; want %d, each deleted with it", all, hers,
			table, n)
	}
}

// A work item's write and its project's deletion, or its workspace's,
// serialize on Web's row (M4 design 3.5, 3.6; interleavings 2 and 3, the
// creation's branch and the change's), in both orders. bob writes and alice
// deletes: the rows he wrote name him until the deletion stamps them as
// hers. Written first, holding Web FOR NO KEY UPDATE: a work item bob
// creates assigned to himself and labelled Bug (fullIssue), six rows, his
// one subscription, as its creator and its assignee, among them; or alice's
// Fix (aWorkItem) he changes so, which adds his assignment, his
// subscription, a label and three activities to its three rows. The
// deletion waits for Web's row, then deletes every row of Web's work-item
// side with Web, or with acme, by alice, at its one moment, no earlier than
// the write's commit: it read its clock under the locks it took once the
// write had committed (M4 design 3.5). Deleted first: the write, which
// passed the precheck, finds Web deleted under its lock: 404
// project.not_found, or issue.not_found for the change, and it wrote
// nothing; the rows there were are deleted.
func TestAWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize(t *testing.T) {
	for _, iw := range []struct {
		itemWrite
		written int // the rows of Web's work-item side once it has written
	}{{itemWrites[0], 6}, {itemWrites[1], 9}} {
		for _, deletion := range []struct {
			name  string
			write racingWrite
			table string                       // the deleted row's
			id    func(w issueWorld) uuid.UUID // its id
		}{
			{"Web", deletesWeb, "projects", func(w issueWorld) uuid.UUID { return w.web }},
			{"acme", deletesAcme, "workspaces", func(w issueWorld) uuid.UUID { return w.acme }},
		} {
			t.Run(iw.name+", "+deletion.name, func(t *testing.T) {
				inBothOrders(t, iw.prepare, "deleted", deletion.write, iw.name, iw.write("bob", fullBody), func(t *testing.T,
					w issueWorld, deletedFirst bool, deleted, written exchange) {
					w.answered(t, deleted, http.StatusNoContent)
					rows := iw.written
					if deletedFirst {
						w.refused(t, written, http.StatusNotFound, iw.gone)
						rows = iw.rows
					} else {
						w.answered(t, written, iw.done)
					}
					w.deletedWith(t, deletion.table, deletion.id(w), written.released, rows)
				})
			})
		}
	}
}

// Deleting a work item and changing it serialize on Web's row (M4 design
// 3.5, 3.6), in both orders. alice deletes her Fix (aWorkItem), whose
// three rows are hers; bob renames it Mend. Deleted first, holding Web FOR
// NO KEY UPDATE: the change, which passed its precheck on Fix as it was,
// finds no Fix under its lock: 404 issue.not_found, and it wrote nothing;
// the three rows are deleted. Changed first: the deletion waits for Web's
// row, then deletes the work item and its rows, bob's activity among them,
// four, by alice, at its one moment, no earlier than the change's commit:
// it read its clock under the lock it took once the change had committed.
func TestDeletingAWorkItemAndChangingItSerialize(t *testing.T) {
	inBothOrders(t, aWorkItem, "deleted", deletesIssue("alice", nil), "changed", changesIssue("bob", named("Mend")), func(t *testing.T,
		w issueWorld, deletedFirst bool, deleted, changed exchange) {
		w.answered(t, deleted, http.StatusNoContent)
		rows, said := 4, []string{"name: Fix → Mend"}
		if deletedFirst {
			w.refused(t, changed, http.StatusNotFound, "issue.not_found")
			rows, said = 3, nil
		} else {
			w.answered(t, changed, http.StatusOK)
		}
		if got := w.activityBy(t, w.workItem(t), w.ids["bob"]); !slices.Equal(got, said) {
			t.Errorf("bob's activity says %q, want %q", got, said)
		}
		w.deletedWith(t, "issues", w.workItem(t), changed.released, rows)
	})
}
