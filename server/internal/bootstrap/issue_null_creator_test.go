package bootstrap

import (
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// NULL is neither an OwnOnly caller's property nor an AllowCreator grant.
// Accounts remain alive; only the nullable creator column is changed.
func TestNullCreatorRemainsHiddenAndUnowned(t *testing.T) {
	f := newDeletionWindowFixture(t)
	w := f.w
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issues SET created_by_id=NULL WHERE id=ANY($1)", []uuid.UUID{f.x.issue, f.x.archived, f.x.other}); err != nil {
		t.Fatal(err)
	}
	// The member cannot inherit the vanished creator's deletion permission.
	before := registrationRows(t, w.pool)
	w.send(t, "bob", "DELETE", "/api/v0/issues/"+f.x.issue.String(), "", 403)
	if registrationRows(t, w.pool) != before {
		t.Fatal("NULL creator granted member a write")
	}
	bobTree := nullCreatorTree(t, w, "bob")
	w.send(t, "alice", "PATCH", bobsMembership(t, w), `{"role":5}`, 200)
	w.adminGuest(t)
	waltTree := nullCreatorTree(t, w, "walt")
	absent := f.x.absent(false)
	for _, actor := range []string{"bob", "walt"} {
		tree := bobTree
		if actor == "walt" {
			tree = waltTree
		}
		checkNullCreatorTree(t, w, actor, tree)
		for _, row := range deletionWindowCases() {
			if slices.Contains([]string{"searchIssues", "listDrafts", "getDraft", "listMemberIssues", "listIssues", "createDraft", "updateDraft", "deleteDraft", "publishDraft", "createIssue"}, row.op) {
				continue
			}
			t.Run(actor+"/"+row.op, func(t *testing.T) {
				path, body := row.request(f.x)
				missing, missingBody := row.request(absent)
				r := deletionWindowRequest{name: row.op, actor: actor, method: row.method, path: path, body: body, want: 404, absentPath: missing, absentBody: missingBody}
				before := registrationRows(t, w.pool)
				r.checked(t, w)
				if registrationRows(t, w.pool) != before {
					t.Fatal("hidden NULL creator request wrote rows")
				}
			})
		}
		for _, path := range []string{"/api/v0/projects/" + w.web.String() + "/issues", "/api/v0/workspaces/acme/issue-search?q=Target"} {
			body := w.send(t, actor, "GET", path, "", 200)
			for _, row := range listIn[storedIssue](t, body) {
				if row.ID == f.x.issue || row.ID == f.x.archived || row.ID == f.x.other {
					t.Fatal("NULL creator leaked through directory", body)
				}
			}
		}
	}
	w.send(t, "alice", "DELETE", "/api/v0/issues/"+f.x.issue.String(), "", 204)
	var deleted bool
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at IS NOT NULL AND created_by_id IS NULL AND updated_by_id=$2 FROM issues WHERE id=$1", f.x.issue, w.ids["alice"]).Scan(&deleted); err != nil || !deleted {
		t.Fatal("admin cannot delete unowned issue", deleted, err)
	}
}

// Two visible generations surround a hidden middle generation. This catches
// count, parent_id and nested-relation leaks, not just direct target refusal.
func nullCreatorTree(t *testing.T, w issueWorld, actor string) [3]uuid.UUID {
	t.Helper()
	parent := w.newWorkItemBy(t, actor, w.web, `{"name":"Visible parent"}`)
	middle := w.newWorkItemBy(t, actor, w.web, `{"name":"Unknown creator","parent_id":"`+parent.String()+`"}`)
	child := w.newWorkItemBy(t, actor, w.web, `{"name":"Visible child","parent_id":"`+middle.String()+`"}`)
	grandchild := w.newWorkItemBy(t, actor, w.web, `{"name":"Visible grandchild","parent_id":"`+child.String()+`"}`)
	w.send(t, "alice", "POST", relationsOf(parent), relating("relates_to", child), 201)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issues SET created_by_id=NULL WHERE id=$1", middle); err != nil {
		t.Fatal(err)
	}
	return [3]uuid.UUID{parent, child, grandchild}
}

func checkNullCreatorTree(t *testing.T, w issueWorld, actor string, tree [3]uuid.UUID) {
	t.Helper()
	var parent, child countedIssue
	decodeAnswer(t, w.send(t, actor, "GET", "/api/v0/issues/"+tree[0].String(), "", 200), &parent)
	decodeAnswer(t, w.send(t, actor, "GET", "/api/v0/issues/"+tree[1].String(), "", 200), &child)
	if parent.SubIssuesCount != 0 || child.ParentID != nil || child.SubIssuesCount != 1 {
		t.Fatal("hidden hierarchy leaked", parent, child)
	}
	windowIDs()(t, w.send(t, actor, "GET", subIssues(tree[0]), "", 200))
	windowIDs(tree[2])(t, w.send(t, actor, "GET", subIssues(tree[1]), "", 200))
	rows := listIn[answeredRelation](t, w.send(t, actor, "GET", relationsOf(tree[0]), "", 200))
	if len(rows) != 1 || rows[0].Issue.ID != tree[1] || rows[0].Issue.ParentID != nil || rows[0].Issue.SubIssuesCount != 1 {
		t.Fatal("nested hierarchy leaked", rows)
	}
}
