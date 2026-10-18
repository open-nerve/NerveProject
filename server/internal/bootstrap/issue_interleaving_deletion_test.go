package bootstrap

import (
	"net/http"
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

// fullBody is the world's fullIssue, as createsIssue takes a body.
func fullBody(_ *testing.T, w issueWorld) string {
	return w.fullIssue()
}

// deletedWith checks the deletion of the row id of table, the deleted
// project's or workspace's: at a moment no earlier than since, with the
// rows of Web's work-item side, n of them, each deleted and last written
// at that moment, by alice, and not before it was made.
func (w issueWorld) deletedWith(t *testing.T, table string, id uuid.UUID, since time.Time, n int) {
	t.Helper()
	var at time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM "+table+" WHERE id = $1", id).Scan(&at); err != nil || at.Before(since) {
		t.Errorf("the row of %s deleted at %v (%v); want no earlier than %v", table, at, err, since)
	}
	var all, hers int
	if err := w.pool.QueryRow(pgtest.Soon(t), `
		SELECT count(*), count(*) FILTER (WHERE r.deleted_at = d.at AND r.updated_at = d.at AND r.updated_by_id = $3 AND r.created_at <= d.at)
		FROM (SELECT deleted_at, updated_at, updated_by_id, created_at FROM issues WHERE project_id = $2
			UNION ALL SELECT deleted_at, updated_at, updated_by_id, created_at FROM issue_assignees WHERE project_id = $2
			UNION ALL SELECT deleted_at, updated_at, updated_by_id, created_at FROM issue_labels WHERE project_id = $2
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
// creation's branch and the change's), in both orders. Written first,
// holding Web FOR NO KEY UPDATE: a work item created assigned to bob and
// labelled Bug (fullIssue), seven rows; or alice's Fix (aWorkItem) changed
// so, which adds an assignee, his subscription, a label and three
// activities to its three rows. The deletion waits for Web's row, then
// deletes every row of Web's work-item side with Web, or with acme, by
// alice, at its one moment, no earlier than the write's commit: it read
// its clock under the locks it took once the write had committed (M4
// design 3.5). Deleted first: the write, which passed the precheck, finds
// Web deleted under its lock: 404 project.not_found, or issue.not_found
// for the change, and it wrote nothing; the rows there were are deleted.
func TestAWorkItemsWriteAndDeletingItsProjectOrWorkspaceSerialize(t *testing.T) {
	for _, iw := range []struct {
		itemWrite
		written int // the rows of Web's work-item side once it has written
	}{{itemWrites[0], 7}, {itemWrites[1], 9}} {
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
				inBothOrders(t, iw.prepare, "deleted", deletion.write, iw.name, iw.write("alice", fullBody), func(t *testing.T,
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
