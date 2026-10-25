package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The cascade of deleteProject against the catalog (M3 design 3.3, 3.6,
// 4), as deleteWorkspace's (workspace_deletion_test.go): a project deleted
// through the wired app leaves no undeleted row under it, in any table with
// a foreign key to projects; deletes each at the project's instant, by its
// deleter; and changes no row of the workspace's other project. The tables
// come from pg_constraint (keysTo): a phase that adds a table under
// projects fails this test until createProject or the seed here writes a
// row of it and the deletion deletes it.

// twoProjects is the wired app on a database of its own, and acme with two
// projects alice created, Web and Ops: each has her membership, her display
// settings in it and its states, through createProject, a label of hers,
// Bug, with its child, UI, and a work item of hers with a row of each
// table under it, written directly (seedLabels, seedIssues).
func twoProjects(t *testing.T) (contract *apitest.Contract, base string, pool *pgxpool.Pool, alice string, aliceID, web, ops uuid.UUID) {
	t.Helper()
	contract = apitest.Load(t)
	url := pgtest.NewDatabase(t)
	base = startApp(t, testConfig(t, url, false), migrations.FS())
	pool = openPool(t, url)
	alice = registerAccount(t, contract, base, "alice@example.com").AccessToken
	aliceID = accountID(t, contract, base, alice)
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice, `{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	web, ops = createdProject(t, contract, base, alice, "acme", "Web", "WEB"), createdProject(t, contract, base, alice, "acme", "Ops", "OPS")
	for _, project := range []uuid.UUID{web, ops} {
		seedLabels(t, pool, project, aliceID)
		seedIssues(t, pool, project, aliceID)
		for _, id := range queryIDs(t, pool, "SELECT id FROM issues WHERE project_id=$1", project) {
			status, body := call(t, contract, http.MethodPatch, base+"/api/v0/issues/"+id.String(), alice, fmt.Sprintf(`{"description_html":%q}`, mentionHTML(aliceID)))
			if status != http.StatusOK {
				t.Fatal(status, body)
			}
			status, body = call(t, contract, http.MethodPost, base+reactionsOf(id), alice, `{"reaction":"128077"}`)
			if status != http.StatusCreated {
				t.Fatal(status, body)
			}
			status, body = call(t, contract, http.MethodPost, base+commentsOf(id), alice, `{"comment_html":"<p>Seeded</p>"}`)
			if status != http.StatusCreated {
				t.Fatal(status, body)
			}
			var comment answeredComment
			decodeAnswer(t, body, &comment)
			status, body = call(t, contract, http.MethodPost, base+commentReactionsOf(comment.ID), alice, `{"reaction":"128077"}`)
			if status != http.StatusCreated {
				t.Fatal(status, body)
			}
		}
	}
	return contract, base, pool, alice, aliceID, web, ops
}

// seedLabels writes a label of project, Bug, and its child, UI, by by,
// directly: the seed does not depend on which of the project module's
// writes exist.
func seedLabels(t *testing.T, pool *pgxpool.Pool, project, by uuid.UUID) {
	t.Helper()
	if tag, err := pool.Exec(pgtest.Soon(t), `INSERT INTO labels (id, workspace_id, project_id, parent_id, name, created_by_id, updated_by_id)
		SELECT $1::uuid, workspace_id, id, NULL::uuid, 'Bug', $2::uuid, $2::uuid FROM projects WHERE id = $3
		UNION ALL SELECT $4::uuid, workspace_id, id, $1::uuid, 'UI', $2::uuid, $2::uuid FROM projects WHERE id = $3`,
		uuid.NewV7(), by, project, uuid.NewV7()); err != nil || tag.RowsAffected() != 2 {
		t.Fatalf("seeding the labels of %s: %d rows, %v; want 2", project, tag.RowsAffected(), err)
	}
}

// seedIssues writes a work item of project in its default state, by by,
// and a row of each of the issue module's tables under it: by its assignee
// and its subscriber, the project's label Bug (seedLabels) on it, its
// relation to another work item of the project, its blocker (relate), and
// its activity's first row, directly: the seed does not depend on which of
// the issue module's writes exist. Each number is the project's counter,
// moved on, as a creation moves it.
func seedIssues(t *testing.T, pool *pgxpool.Pool, project, by uuid.UUID) {
	t.Helper()
	issue, blocker := uuid.NewV7(), uuid.NewV7()
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{`UPDATE projects SET last_issue_sequence = last_issue_sequence + 1 WHERE id = $1`, []any{project}},
		{`INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id, created_by_id, updated_by_id)
			SELECT $1, p.workspace_id, p.id, s.id, 'Seeded', p.last_issue_sequence, $3, $3
			FROM projects p JOIN states s ON s.project_id = p.id AND s."default" AND s.deleted_at IS NULL WHERE p.id = $2`,
			[]any{issue, project, by}},
		{`INSERT INTO issue_assignees (id, workspace_id, project_id, issue_id, assignee_id, created_by_id, updated_by_id)
			SELECT $1, workspace_id, project_id, id, $3, $3, $3 FROM issues WHERE id = $2`, []any{uuid.NewV7(), issue, by}},
		{`INSERT INTO issue_labels (id, workspace_id, project_id, issue_id, label_id, created_by_id, updated_by_id)
			SELECT $1, i.workspace_id, i.project_id, i.id, l.id, $3, $3 FROM issues i JOIN labels l ON l.project_id = i.project_id AND l.name = 'Bug'
			WHERE i.id = $2`, []any{uuid.NewV7(), issue, by}},
		{`UPDATE projects SET last_issue_sequence = last_issue_sequence + 1 WHERE id = $1`, []any{project}},
		{`INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id, created_by_id, updated_by_id)
			SELECT $1, workspace_id, project_id, state_id, 'Blocker', (SELECT last_issue_sequence FROM projects WHERE id = $2), $3, $3
			FROM issues WHERE id = $4`, []any{blocker, project, by, issue}},
		{`INSERT INTO issue_subscribers (id, workspace_id, project_id, issue_id, subscriber_id, created_by_id, updated_by_id)
			SELECT $1, workspace_id, project_id, id, $3, $3, $3 FROM issues WHERE id = $2`, []any{uuid.NewV7(), issue, by}},
		{`INSERT INTO issue_activities (id, workspace_id, project_id, issue_id, comment, actor_id, created_by_id, updated_by_id)
			SELECT $1, workspace_id, project_id, id, 'created the work item', $3, $3, $3 FROM issues WHERE id = $2`, []any{uuid.NewV7(), issue, by}},
	} {
		if tag, err := pool.Exec(pgtest.Soon(t), stmt.sql, stmt.args...); err != nil || tag.RowsAffected() != 1 {
			t.Fatalf("seeding the work items of %s: %s: %d rows, %v; want 1", project, stmt.sql, tag.RowsAffected(), err)
		}
	}
	relate(t, pool, issue, blocker, "blocked_by", by)
}

// noRowLastWrittenBy fails the test when a row under the parent row id, by
// one of keys, the foreign keys to its table (keysTo), the row itself
// included, was last written by by: a deletion by by that kept a row's
// writer would not show on that row. The deletion tests read it before
// they delete.
func noRowLastWrittenBy(t *testing.T, pool *pgxpool.Pool, keys []foreignKey, id, by uuid.UUID) {
	t.Helper()
	for _, k := range keys {
		var n int
		if err := pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM "+k.table+" r WHERE "+k.scope()+" AND updated_by_id = $2", id, by).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s: %d rows under the %s %s last written by %s, %v; want none", k, n, rowOf(k.parent), id, by, err)
		}
	}
}

// Deleting a project through the API soft-deletes every row under it in
// every table the catalog ties to projects, and the project row, each at
// the project's deleted_at, a time within the request, and by the account
// that deleted it, and changes nothing under the workspace's other project.
// The deleted project is not found any more. A work item of Web that alice
// deleted before keeps its rows as they were, their deletion hers (M4/P1a
// review §6: the statements pass over a row deleted before). Web is
// archived then: an archived project is deleted as any other. Its deleter
// is dave, acme's admin, whom alice adds as Web's admin: no row under Web
// is of his writing before, so a deletion that kept a row's writer would
// show.
func TestDeletingAProjectLeavesNoUndeletedRowUnderIt(t *testing.T) {
	contract, base, pool, alice, aliceID, web, ops := twoProjects(t)
	gone := deletedWorkItem(t, contract, base, pool, alice, aliceID, web)
	goneKeys := keysTo(t, pool, "issues")
	goneRows := make([]string, len(goneKeys))
	for i, k := range goneKeys {
		if goneRows[i] = k.rows(t, pool, gone); goneRows[i] == "" {
			t.Fatalf("%s: no row of the work item deleted before Web: deletedWorkItem writes one", k)
		}
	}
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/projects/"+web.String()+"/archive", alice, ""); status != http.StatusOK {
		t.Fatalf("archiving Web = %d %s, want 200", status, body)
	}
	dave := registerAccount(t, contract, base, "dave@example.com").AccessToken
	daveID := accountID(t, contract, base, dave)
	inWorkspaceOf(t, pool, web, daveID, aliceID, shared.RoleAdmin)
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/projects/"+web.String()+"/members", alice,
		`{"members":[{"member_id":"`+daveID.String()+`","role":20}]}`); status != http.StatusCreated {
		t.Fatalf("adding dave to Web = %d %s, want 201", status, body)
	}
	keys := keysTo(t, pool, "projects")
	under, recorded := make([]rowsUnder, len(keys)), make([][]uuid.UUID, len(keys))
	for i, k := range keys {
		recorded[i] = k.undeleted(t, pool, web)
		under[i] = rowsUnder{key: k.String(), deletedBefore: len(recorded[i]), keptBefore: k.rows(t, pool, ops)}
	}
	noRowLastWrittenBy(t, pool, keys, web, daveID)

	before := time.Now().Truncate(time.Microsecond)
	if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/projects/"+web.String(), dave, ""); status != http.StatusNoContent {
		t.Fatalf("deleting Web = %d %s, want 204", status, body)
	}
	after := time.Now()

	for i, k := range keys {
		under[i].deletedAfter, under[i].keptAfter = len(k.undeleted(t, pool, web)), k.rows(t, pool, ops)
	}
	for _, v := range deletionViolations("projects", under, nil) {
		t.Error(v)
	}
	for i, k := range keys {
		if rows := k.unstamped(t, pool, recorded[i], web, daveID); rows != "" {
			t.Errorf("%s: rows not deleted at the project's deleted_at by its deleter %s:\n%s", k, daveID, rows)
		}
	}
	for i, k := range goneKeys {
		if rows := k.rows(t, pool, gone); rows != goneRows[i] {
			t.Errorf("%s: the rows of the work item deleted before Web:\n%s\nwant them as they were:\n%s", k, rows, goneRows[i])
		}
	}
	var at time.Time
	if err := pool.QueryRow(context.Background(), "SELECT deleted_at FROM projects WHERE id = $1", web).Scan(&at); err != nil ||
		at.Before(before) || at.After(after) {
		t.Errorf("Web's deleted_at = %v, %v; want within the request, %v to %v", at, err, before, after)
	}
	if status, body := call(t, contract, http.MethodGet, base+"/api/v0/projects/"+web.String(), alice, ""); status != http.StatusNotFound {
		t.Errorf("reading Web deleted = %d %s, want 404", status, body)
	}
}

// Deleting a project deletes the relations of its work items either way
// (M4 design 3.6, 3.10), each at the project's deleted_at, whichever
// project's row it is, of each stored type: beside the seeded relations
// (blocked_by), Web's Seeded related to Ops' Seeded (duplicate), a row of
// Web's, and Ops' Blocker related to Web's Seeded (relates_to), a row of
// Ops', written directly (relate). The second is no row under Web, which
// the catalog's test of the deletion reads
// (TestDeletingAProjectLeavesNoUndeletedRowUnderIt). Ops' own relation, of
// its Seeded to its Blocker, stays as it was.
func TestDeletingAProjectDeletesItsWorkItemsRelationsEitherWay(t *testing.T) {
	contract, base, pool, alice, aliceID, web, ops := twoProjects(t)
	const items = "SELECT id FROM issues WHERE project_id = $1 ORDER BY sequence_id"
	webItems, opsItems := queryIDs(t, pool, items, web), queryIDs(t, pool, items, ops)
	if len(webItems) != 2 || len(opsItems) != 2 {
		t.Fatalf("the work items %v of Web and %v of Ops; want Seeded and Blocker of each", webItems, opsItems)
	}
	seeded := webItems[0]
	relate(t, pool, seeded, opsItems[0], "duplicate", aliceID)
	relate(t, pool, opsItems[1], seeded, "relates_to", aliceID)
	const opsOwn = `SELECT coalesce(string_agg(id::text || ' ' || coalesce(deleted_at::text, 'undeleted'), ', ' ORDER BY id), '')
		FROM issue_relations WHERE project_id = $1 AND related_issue_id <> $2`
	var before, after string
	if err := pool.QueryRow(pgtest.Soon(t), opsOwn, ops, seeded).Scan(&before); err != nil || before == "" {
		t.Fatalf("Ops' own relations: %q, %v; want its seeded one", before, err)
	}

	if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/projects/"+web.String(), alice, ""); status != http.StatusNoContent {
		t.Fatalf("deleting Web = %d %s, want 204", status, body)
	}

	var left, all int
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT count(*) FILTER (WHERE r.deleted_at IS DISTINCT FROM p.deleted_at), count(*)
		FROM issue_relations r, projects p WHERE p.id = $1 AND $2 IN (r.issue_id, r.related_issue_id)`, web, seeded).Scan(&left, &all); err != nil ||
		left != 0 || all != 3 {
		t.Errorf("Web's Seeded's relations: %d of %d not deleted at Web's deleted_at, %v; want none of 3", left, all, err)
	}
	if err := pool.QueryRow(pgtest.Soon(t), opsOwn, ops, seeded).Scan(&after); err != nil || after != before {
		t.Errorf("Ops' own relations: %q, %v; want them as they were: %q", after, err, before)
	}
}

// Every statement of the deletion runs in its one transaction (M3 design
// 3.3, 9.3): a deletion refused at its commit, after every statement ran,
// changes no row under either project. A deferred constraint trigger
// refuses the commits that wrote one table under projects, each table the
// catalog ties to projects in turn: a step that ran in a transaction of
// its own commits its rows deleted unless it wrote that table, so it is
// seen while another table's commit is refused, wherever it comes in the
// deletion.
func TestAProjectDeletionRefusedAtItsCommitChangesNoRow(t *testing.T) {
	contract, base, pool, alice, _, web, ops := twoProjects(t)
	keys := keysTo(t, pool, "projects")
	rowsOf := func() []string {
		var all []string
		for _, k := range keys {
			for _, id := range []uuid.UUID{web, ops} {
				all = append(all, k.String()+":\n"+k.rows(t, pool, id))
			}
		}
		return all
	}
	before := rowsOf()
	for _, k := range keys {
		restore := refusingCommits(t, pool, k.table)
		if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/projects/"+web.String(), alice, ""); status != http.StatusInternalServerError {
			t.Errorf("deleting Web with the commits of %s refused = %d %s, want 500", k.table, status, body)
		}
		if after := rowsOf(); !slices.Equal(after, before) {
			t.Fatalf("the rows after the deletion refused at its commit of %s:\n%q\nwant\n%q", k.table, after, before)
		}
		restore()
	}
}
