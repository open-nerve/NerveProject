package bootstrap

import (
	"net/http"
	"slices"
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
// under the work item, and the deletion deletes it.

// subItems is the key of a work item's sub-items, which come with P3, as
// does their deletion with their parent's (M4 design 3.6, 3.9): no work
// item has one before.
const subItems = "issues.parent_id"

// workItemKeys are the foreign keys to issues (keysTo), the work item's
// own row included, but its sub-items' (subItems).
func workItemKeys(t testing.TB, pool *pgxpool.Pool) []foreignKey {
	t.Helper()
	return slices.DeleteFunc(keysTo(t, pool, "issues"), func(k foreignKey) bool { return k.String() == subItems })
}

// deletedWorkItem is a work item that the caller of token, account, makes
// in project through the API, assigned to him and labelled the project's
// Bug, and deletes through the API: a row of each table under it,
// deleted.
func deletedWorkItem(t *testing.T, contract *apitest.Contract, base string, pool *pgxpool.Pool, token string, account,
	project uuid.UUID) uuid.UUID {
	t.Helper()
	var bug uuid.UUID
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT id FROM labels WHERE project_id = $1 AND name = 'Bug'", project).Scan(&bug); err != nil {
		t.Fatal(err)
	}
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/projects/"+project.String()+"/issues", token,
		`{"name":"Gone","assignee_ids":["`+account.String()+`"],"label_ids":["`+bug.String()+`"]}`)
	if status != http.StatusCreated {
		t.Fatalf("creating the work item = %d %s, want 201", status, body)
	}
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, body, &created)
	if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/issues/"+created.ID.String(), token, ""); status != http.StatusNoContent {
		t.Fatalf("deleting the work item = %d %s, want 204", status, body)
	}
	return created.ID
}

// Deleting a work item through the API soft-deletes every row under it in
// every table the catalog ties to issues, and the work item's row, each at
// its deleted_at, a time within the request, and by the account that
// deleted it; its rows deleted before keep their deletion, and nothing
// under its project's other work item changes. The work item is bob's,
// assigned to him and labelled Bug (fullIssue), whom he unassigned and
// unlabelled, then assigned and labelled again: a row of each table under
// it, and of its assignees and its labels one deleted before, all of
// bob's writing. alice, Web's admin, deletes it: no row under it is of her
// writing before, so a deletion that kept a row's writer would show. The
// deleted work item is not found any more.
func TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt(t *testing.T) {
	w := newIssueWorld(t)
	deleted, kept := w.newWorkItemBy(t, "bob", w.web, w.fullIssue()), w.newWorkItemBy(t, "bob", w.web, w.fullIssue())
	for _, body := range []string{`{"assignee_ids":[],"label_ids":[]}`,
		`{"assignee_ids":["` + w.ids["bob"].String() + `"],"label_ids":["` + w.bug.String() + `"]}`} {
		w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+deleted.String(), body, http.StatusOK)
	}
	keys := workItemKeys(t, w.pool)
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
