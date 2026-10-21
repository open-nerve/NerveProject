package bootstrap

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The deletion of a work item against the catalog (M4 design 3.6), as a
// project's (project_deletion_test.go): a work item deleted through the
// wired app leaves no undeleted row under it, in any table with a foreign
// key to issues (keysTo); deletes each at the work item's deleted_at, a
// time within the request, by its deleter; leaves its rows deleted before
// as they were; and changes no row of another work item of its project.
// The tables come from pg_constraint: a phase that adds a table under
// issues fails this test until a creation or a change writes a row of it
// under the work item, or the test puts one there (underWorkItem), and the
// deletion deletes it.

// underWorkItem puts each of items under the work item parent, directly:
// a work item's sub-items (M4 design 3.9), seeded so that the test does
// not depend on which of the issue module's writes exist.
func underWorkItem(t testing.TB, pool *pgxpool.Pool, parent uuid.UUID, items ...uuid.UUID) {
	t.Helper()
	if tag, err := pool.Exec(pgtest.Soon(t), "UPDATE issues SET parent_id = $1 WHERE id = ANY ($2)", parent, items); err != nil ||
		tag.RowsAffected() != int64(len(items)) {
		t.Fatalf("putting %v under %s: %d rows, %v; want %d", items, parent, tag.RowsAffected(), err, len(items))
	}
}

// deletedWorkItem is a work item that the caller of token, account, makes
// in project through the API, assigned to him and labelled the project's
// Bug, with another he makes under it (underWorkItem), and deletes through
// the API: a row of each table under it, deleted.
func deletedWorkItem(t *testing.T, contract *apitest.Contract, base string, pool *pgxpool.Pool, token string, account,
	project uuid.UUID) uuid.UUID {
	t.Helper()
	var bug uuid.UUID
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT id FROM labels WHERE project_id = $1 AND name = 'Bug'", project).Scan(&bug); err != nil {
		t.Fatal(err)
	}
	var made [2]uuid.UUID
	for i, body := range []string{`{"name":"Gone","assignee_ids":["` + account.String() + `"],"label_ids":["` + bug.String() + `"]}`,
		`{"name":"Part"}`} {
		status, answer := call(t, contract, http.MethodPost, base+"/api/v0/projects/"+project.String()+"/issues", token, body)
		if status != http.StatusCreated {
			t.Fatalf("creating the work item = %d %s, want 201", status, answer)
		}
		var created struct {
			ID uuid.UUID `json:"id"`
		}
		decodeAnswer(t, answer, &created)
		made[i] = created.ID
	}
	gone := made[0]
	underWorkItem(t, pool, gone, made[1])
	if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/issues/"+gone.String(), token, ""); status != http.StatusNoContent {
		t.Fatalf("deleting the work item = %d %s, want 204", status, body)
	}
	return gone
}

// Deleting a work item through the API soft-deletes every row under it in
// every table the catalog ties to issues, and the work item's row, each at
// its deleted_at, a time within the request, and by the account that
// deleted it; its rows deleted before keep their deletion, and nothing
// under its project's other work item changes. The work item is bob's,
// assigned to him and labelled Bug (fullIssue), whom he unassigned and
// unlabelled, then assigned and labelled again, with a work item of his
// under it (underWorkItem): a row of each table under it, and of its
// assignees and its labels one deleted before, all of bob's writing. The
// other work item has the same rows. alice, Web's admin, deletes it: no
// row under it is of her writing before, so a deletion that kept a row's
// writer would show. The deleted work item is not found any more.
func TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt(t *testing.T) {
	w := newIssueWorld(t)
	deleted, kept := w.newWorkItemBy(t, "bob", w.web, w.fullIssue()), w.newWorkItemBy(t, "bob", w.web, w.fullIssue())
	for _, parent := range []uuid.UUID{deleted, kept} {
		underWorkItem(t, w.pool, parent, w.newWorkItemBy(t, "bob", w.web, `{"name":"Part"}`))
	}
	for _, body := range []string{`{"assignee_ids":[],"label_ids":[]}`,
		`{"assignee_ids":["` + w.ids["bob"].String() + `"],"label_ids":["` + w.bug.String() + `"]}`} {
		w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+deleted.String(), body, http.StatusOK)
	}
	keys := keysTo(t, w.pool, "issues")
	under, recorded := make([]rowsUnder, len(keys)), make([][]uuid.UUID, len(keys))
	earlier, earlierRows := make([][]uuid.UUID, len(keys)), make([]string, len(keys))
	for i, k := range keys {
		recorded[i], earlier[i] = k.undeleted(t, w.pool, deleted), k.deleted(t, w.pool, deleted)
		earlierRows[i] = k.asText(t, w.pool, earlier[i])
		under[i] = rowsUnder{key: k.String(), deletedBefore: len(recorded[i]), keptBefore: k.rows(t, w.pool, kept)}
		if (k.table == "issue_assignees" || k.table == "issue_labels") && len(earlier[i]) == 0 {
			t.Fatalf("%s: no row of the work item deleted before: bob's change removes one", k)
		}
	}
	noRowLastWrittenBy(t, w.pool, keys, deleted, w.ids["alice"])

	before := time.Now().Truncate(time.Microsecond)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+deleted.String(), "", http.StatusNoContent)
	after := time.Now()

	for i, k := range keys {
		under[i].deletedAfter, under[i].keptAfter = len(k.undeleted(t, w.pool, deleted)), k.rows(t, w.pool, kept)
	}
	for _, v := range deletionViolations("issues", under, nil) {
		t.Error(v)
	}
	for i, k := range keys {
		if rows := k.unstamped(t, w.pool, recorded[i], deleted, w.ids["alice"]); rows != "" {
			t.Errorf("%s: rows not deleted at the work item's deleted_at by its deleter alice:\n%s", k, rows)
		}
		if rows := k.asText(t, w.pool, earlier[i]); rows != earlierRows[i] {
			t.Errorf("%s: the rows deleted before:\n%s\nwant them as they were:\n%s", k, rows, earlierRows[i])
		}
	}
	var at time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM issues WHERE id = $1", deleted).Scan(&at); err != nil ||
		at.Before(before) || at.After(after) {
		t.Errorf("the work item's deleted_at = %v, %v; want within the request, %v to %v", at, err, before, after)
	}
	w.send(t, "alice", http.MethodGet, "/api/v0/issues/"+deleted.String(), "", http.StatusNotFound)
}

// Deleting a work item through the API deletes its subtree at one moment,
// by one deleter (M4 design 3.6, 3.9): the work item and each one under
// it, however deep, an archived one too, and every row under each. In
// Web, bob's root, assigned and labelled (fullIssue), is under alice's p
// and has under it alice's a, assigned and labelled too, and bob's b; a
// has c under it, in Done, archived; alice's s has her d under it. alice
// deletes root: root, a, b and c are deleted, and changed last by her, at
// root's deleted_at, and so is every row under each (workItemStamps: each
// changed by alice and deleted at the work item's last change); p, s and d
// keep every row as it was. A deleted work item of the subtree is not
// found any more.
func TestDeletingAWorkItemDeletesItsSubtreeAtOneMoment(t *testing.T) {
	w := newIssueWorld(t)
	p := w.newWorkItem(t)
	root := w.newWorkItemBy(t, "bob", w.web, w.fullIssue())
	a, b, c := w.newWorkItem(t), w.newWorkItemBy(t, "bob", w.web, `{"name":"Fix"}`), w.doneWorkItem(t)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+c.String()+"/archive", "", http.StatusOK)
	s, d := w.newWorkItem(t), w.newWorkItem(t)
	underWorkItem(t, w.pool, p, root)
	underWorkItem(t, w.pool, root, a, b)
	underWorkItem(t, w.pool, a, c)
	underWorkItem(t, w.pool, s, d)
	kept := map[uuid.UUID][]string{}
	for _, id := range []uuid.UUID{p, s, d} {
		kept[id] = w.workItemStamps(t, id)
	}

	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+root.String(), "", http.StatusNoContent)

	var at time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM issues WHERE id = $1", root).Scan(&at); err != nil {
		t.Fatal(err)
	}
	for name, id := range map[string]uuid.UUID{"root": root, "a": a, "b": b, "c": c} {
		var deletedAt, changedAt *time.Time
		var by *uuid.UUID
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at, updated_at, updated_by_id FROM issues WHERE id = $1", id).Scan(&deletedAt,
			&changedAt, &by); err != nil || deletedAt == nil || !deletedAt.Equal(at) || !changedAt.Equal(at) || by == nil || *by != w.ids["alice"] {
			t.Errorf("%s: deleted at %v, changed at %v by %v, %v; want both at root's %v, by alice", name, deletedAt, changedAt, by, err, at)
		}
		for _, line := range w.workItemStamps(t, id) {
			if !strings.Contains(line, "changed by alice changed, deleted changed") {
				t.Errorf("%s: %s; want it changed by alice and deleted at the deletion", name, line)
			}
		}
		w.send(t, "alice", http.MethodGet, "/api/v0/issues/"+id.String(), "", http.StatusNotFound)
	}
	for id, rows := range kept {
		if got := w.workItemStamps(t, id); !slices.Equal(got, rows) {
			t.Errorf("the rows of %s:\n%q\nwant them as they were:\n%q", id, got, rows)
		}
	}
}
