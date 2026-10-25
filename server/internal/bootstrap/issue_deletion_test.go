package bootstrap

import (
	"fmt"
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

// relate relates the work item from to the work item to, of any project,
// directly, by by: a relation row of from's project, of the stored type
// kind (M4 design 3.10), seeded so that the test does not depend on which
// of the issue module's writes exist.
func relate(t testing.TB, pool *pgxpool.Pool, from, to uuid.UUID, kind string, by uuid.UUID) {
	t.Helper()
	if tag, err := pool.Exec(pgtest.Soon(t), `INSERT INTO issue_relations (id, workspace_id, project_id, issue_id, related_issue_id,
		relation_type, created_by_id, updated_by_id) SELECT $1, workspace_id, project_id, id, $3, $4, $5, $5 FROM issues WHERE id = $2`,
		uuid.NewV7(), from, to, kind, by); err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("relating %s to %s: %d rows, %v; want 1", from, to, tag.RowsAffected(), err)
	}
}

// deletedWorkItem is a work item that the caller of token, account, makes
// in project through the API, assigned to him and labelled the project's
// Bug, with another he makes under it (underWorkItem) and a third, Kin; he
// relates it to Kin and the one under it to it (relate), and deletes it
// through the API: a row of each table under it, deleted.
func deletedWorkItem(t *testing.T, contract *apitest.Contract, base string, pool *pgxpool.Pool, token string, account,
	project uuid.UUID) uuid.UUID {
	t.Helper()
	var bug uuid.UUID
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT id FROM labels WHERE project_id = $1 AND name = 'Bug'", project).Scan(&bug); err != nil {
		t.Fatal(err)
	}
	var made [3]uuid.UUID
	for i, body := range []string{`{"name":"Gone","assignee_ids":["` + account.String() + `"],"label_ids":["` + bug.String() + `"]}`,
		`{"name":"Part"}`, `{"name":"Kin"}`} {
		body = strings.TrimSuffix(body, "}") + `,"description_html":"<mention-component entity_name=\"user_mention\" entity_identifier=\"` + account.String() + `\"></mention-component>"}`
		status, answer := call(t, contract, http.MethodPost, base+"/api/v0/projects/"+project.String()+"/issues", token, body)
		if status != http.StatusCreated {
			t.Fatalf("creating the work item = %d %s, want 201", status, answer)
		}
		var created struct {
			ID uuid.UUID `json:"id"`
		}
		decodeAnswer(t, answer, &created)
		made[i] = created.ID
		status, answer = call(t, contract, http.MethodPost, base+reactionsOf(created.ID), token, `{"reaction":"128077"}`)
		if status != http.StatusCreated {
			t.Fatal(status, answer)
		}
		status, answer = call(t, contract, http.MethodPost, base+commentsOf(created.ID), token, `{"comment_html":"<p>Seeded</p>"}`)
		if status != http.StatusCreated {
			t.Fatal(status, answer)
		}
		var comment answeredComment
		decodeAnswer(t, answer, &comment)
		status, answer = call(t, contract, http.MethodPost, base+commentReactionsOf(comment.ID), token, `{"reaction":"128077"}`)
		if status != http.StatusCreated {
			t.Fatal(status, answer)
		}
	}
	gone := made[0]
	underWorkItem(t, pool, gone, made[1])
	relate(t, pool, gone, made[2], "relates_to", account)
	relate(t, pool, made[1], gone, "relates_to", account)
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
// assignees and its labels one deleted before, all of bob's writing; and,
// of his writing too, written directly (relate), a relation of each stored
// type: its relation to a work item of Ops (duplicate), a row of Web's; a
// relation of another work item of Ops to it (blocked_by), a row of Ops';
// and its relation to another of Web's (relates_to), deleted before. The
// other work item, made and related the same way, has a row of each table
// under it, of which only its relation to another of Web's is deleted
// before. alice, Web's admin, deletes it: no row under it is of her
// writing before, so a deletion that kept a row's writer would show. The
// deleted work item is not found any more.
func TestDeletingAWorkItemLeavesNoUndeletedRowUnderIt(t *testing.T) {
	w := newIssueWorld(t)
	deleted, kept := w.newWorkItemBy(t, "bob", w.web, w.fullIssue()), w.newWorkItemBy(t, "bob", w.web, w.fullIssue())
	for _, parent := range []uuid.UUID{deleted, kept} {
		for _, html := range []string{mentionHTML(w.ids["bob"]), "<p>Removed</p>", mentionHTML(w.ids["bob"])} {
			w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+parent.String(), fmt.Sprintf(`{"description_html":%q}`, html), 200)
		}
		w.send(t, "bob", http.MethodPost, reactionsOf(parent), `{"reaction":"128077"}`, http.StatusCreated)
		w.send(t, "bob", http.MethodPost, reactionsOf(parent), `{"reaction":"1"}`, http.StatusCreated)
		w.send(t, "bob", http.MethodDelete, reactionsOf(parent)+"/1", "", http.StatusNoContent)
		comment := w.newComment(t, "bob", parent)
		addCommentReaction(t, w, "bob", comment.ID, "128077")
		addCommentReaction(t, w, "bob", comment.ID, "1")
		w.send(t, "bob", http.MethodDelete, commentReactionsOf(comment.ID)+"/1", "", 204)
		underWorkItem(t, w.pool, parent, w.newWorkItemBy(t, "bob", w.web, `{"name":"Part"}`))
	}
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	for _, item := range []uuid.UUID{deleted, kept} {
		relate(t, w.pool, item, w.newWorkItemIn(t, ops, `{"name":"Far"}`), "duplicate", w.ids["bob"])
		relate(t, w.pool, w.newWorkItemIn(t, ops, `{"name":"Farther"}`), item, "blocked_by", w.ids["bob"])
		near := w.newWorkItemBy(t, "bob", w.web, `{"name":"Near"}`)
		relate(t, w.pool, item, near, "relates_to", w.ids["bob"])
		if tag, err := w.pool.Exec(pgtest.Soon(t), `UPDATE issue_relations SET deleted_at = now(), updated_at = now() WHERE issue_id = $1
			AND related_issue_id = $2`, item, near); err != nil || tag.RowsAffected() != 1 {
			t.Fatalf("deleting the relation to Near: %d rows, %v; want 1", tag.RowsAffected(), err)
		}
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

// deletesAtOneMoment has the account by delete the work item root through
// the API, and checks that its subtree is deleted at one moment, by one
// deleter (M4 design 3.6, 3.9): each work item of subtree, root among
// them, by its name, is deleted, and changed last by by, at root's
// deleted_at, and so is every row under each (workItemStamps: each changed
// by by and deleted at the work item's last change); each is not found any
// more, by alice, who sees every work item of Web; and each work item of
// kept keeps every row as it was.
func (w issueWorld) deletesAtOneMoment(t *testing.T, by string, root uuid.UUID, subtree map[string]uuid.UUID, kept ...uuid.UUID) {
	t.Helper()
	before := make([][]string, len(kept))
	for i, id := range kept {
		before[i] = w.workItemStamps(t, id)
	}

	w.send(t, by, http.MethodDelete, "/api/v0/issues/"+root.String(), "", http.StatusNoContent)

	var at time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM issues WHERE id = $1", root).Scan(&at); err != nil {
		t.Fatal(err)
	}
	for name, id := range subtree {
		var deletedAt, changedAt *time.Time
		var writer *uuid.UUID
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at, updated_at, updated_by_id FROM issues WHERE id = $1", id).Scan(&deletedAt,
			&changedAt, &writer); err != nil || deletedAt == nil || !deletedAt.Equal(at) || !changedAt.Equal(at) || writer == nil ||
			*writer != w.ids[by] {
			t.Errorf("%s: deleted at %v, changed at %v by %v, %v; want both at the root's %v, by %s", name, deletedAt, changedAt, writer, err, at,
				by)
		}
		for _, line := range w.workItemStamps(t, id) {
			if !strings.Contains(line, "changed by "+by+" changed, deleted changed") {
				t.Errorf("%s: %s; want it changed by %s and deleted at the deletion", name, line, by)
			}
		}
		w.send(t, "alice", http.MethodGet, "/api/v0/issues/"+id.String(), "", http.StatusNotFound)
	}
	for i, id := range kept {
		if got := w.workItemStamps(t, id); !slices.Equal(got, before[i]) {
			t.Errorf("the rows of %s:\n%q\nwant them as they were:\n%q", id, got, before[i])
		}
	}
}

// Deleting a work item through the API deletes its subtree at one moment,
// by one deleter (deletesAtOneMoment): the work item and each one under
// it, however deep, an archived one too, and every row under each, the
// relations of each either way among them. In Web, bob's root, assigned
// and labelled (fullIssue), is under alice's p and has under it alice's a,
// assigned and labelled too, and bob's b; a has c under it, in Done,
// archived, which bob relates to alice's o, outside the subtree (relate):
// a relation that names no work item of the subtree but c, so that a
// deletion of the relations of root alone, the subtree's first in id
// order, shows. alice's s has her d under it. alice deletes root: root, a,
// b and c are deleted, by her, at root's deleted_at, with every row under
// each, c's relation to o among them; p, s and d keep every row as it was.
func TestDeletingAWorkItemDeletesItsSubtreeAtOneMoment(t *testing.T) {
	w := newIssueWorld(t)
	p := w.newWorkItem(t)
	root := w.newWorkItemBy(t, "bob", w.web, w.fullIssue())
	a, b, c := w.newWorkItem(t), w.newWorkItemBy(t, "bob", w.web, `{"name":"Fix"}`), w.doneWorkItem(t)
	for _, id := range []uuid.UUID{p, root, a, b, c} {
		w.send(t, "bob", http.MethodPost, reactionsOf(id), `{"reaction":"128077"}`, http.StatusCreated)
		comment := w.newComment(t, "bob", id)
		addCommentReaction(t, w, "bob", comment.ID, "128077")
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+c.String()+"/archive", "", http.StatusOK)
	s, d, o := w.newWorkItem(t), w.newWorkItem(t), w.newWorkItem(t)
	underWorkItem(t, w.pool, p, root)
	underWorkItem(t, w.pool, root, a, b)
	underWorkItem(t, w.pool, a, c)
	underWorkItem(t, w.pool, s, d)
	relate(t, w.pool, c, o, "relates_to", w.ids["bob"])
	w.deletesAtOneMoment(t, "alice", root, map[string]uuid.UUID{"root": root, "a": a, "b": b, "c": c}, p, s, d)
}

// A work item's subtree is deleted whole, the work items its deleter does
// not see among them (M4 design 3.6, 3.9: the subtree has no exception for
// the guest rule). walt, acme's admin and Web's guest (adminGuest), may
// delete his own work item and sees only his own. Under his mine,
// assigned and labelled (fullIssue), is alice's a, which he does not see
// (404), and under a is her b, both assigned and labelled too; alice's k
// is under none. walt deletes mine (deletesAtOneMoment): mine, a and b are
// deleted, by him, at mine's deleted_at, with every row under each; k
// keeps every row as it was.
func TestAnAdminGuestsDeletionTakesTheSubItemsHeDoesNotSee(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	mine := w.newWorkItemBy(t, "walt", w.web, w.fullIssue())
	a, b, k := w.newWorkItem(t), w.newWorkItem(t), w.newWorkItem(t)
	underWorkItem(t, w.pool, mine, a)
	underWorkItem(t, w.pool, a, b)
	w.send(t, "walt", http.MethodGet, "/api/v0/issues/"+a.String(), "", http.StatusNotFound)
	w.deletesAtOneMoment(t, "walt", mine, map[string]uuid.UUID{"mine": mine, "a": a, "b": b}, k)
}
