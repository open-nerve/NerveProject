package bootstrap

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
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
// settings in it and its states, through createProject, and a label of
// hers, Bug, with its child, UI, written directly (seedLabels).
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
	seedLabels(t, pool, web, aliceID)
	seedLabels(t, pool, ops, aliceID)
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

// Deleting a project through the API soft-deletes every row under it in
// every table the catalog ties to projects, and the project row, each at
// the project's deleted_at, a time within the request, and by the account
// that deleted it, and changes nothing under the workspace's other project.
// The deleted project is not found any more. Web is archived first: an
// archived project is deleted as any other. Its deleter is dave, acme's
// admin, whom alice adds as Web's admin: no row under Web is of his
// writing before, so a deletion that kept a row's writer would show.
func TestDeletingAProjectLeavesNoUndeletedRowUnderIt(t *testing.T) {
	contract, base, pool, alice, aliceID, web, ops := twoProjects(t)
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
		var his int
		if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+k.table+" WHERE "+pgx.Identifier{k.column}.Sanitize()+
			" = $1 AND updated_by_id = $2", web, daveID).Scan(&his); err != nil || his != 0 {
			t.Fatalf("%s: %d rows under Web last written by dave, %v; want none", k, his, err)
		}
	}

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
	var at time.Time
	if err := pool.QueryRow(context.Background(), "SELECT deleted_at FROM projects WHERE id = $1", web).Scan(&at); err != nil ||
		at.Before(before) || at.After(after) {
		t.Errorf("Web's deleted_at = %v, %v; want within the request, %v to %v", at, err, before, after)
	}
	if status, body := call(t, contract, http.MethodGet, base+"/api/v0/projects/"+web.String(), alice, ""); status != http.StatusNotFound {
		t.Errorf("reading Web deleted = %d %s, want 404", status, body)
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
