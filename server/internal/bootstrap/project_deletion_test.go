package bootstrap

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
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
// settings in it and its states, through createProject.
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
	return contract, base, pool, alice, aliceID, createdProject(t, contract, base, alice, "acme", "Web", "WEB"),
		createdProject(t, contract, base, alice, "acme", "Ops", "OPS")
}

// Deleting a project through the API soft-deletes every row under it in
// every table the catalog ties to projects, and the project row, each at
// the project's deleted_at, a time within the request, and by the account
// that deleted it, and changes nothing under the workspace's other project.
// The deleted project is not found any more.
func TestDeletingAProjectLeavesNoUndeletedRowUnderIt(t *testing.T) {
	contract, base, pool, alice, aliceID, web, ops := twoProjects(t)
	keys := keysTo(t, pool, "projects")
	under, recorded := make([]rowsUnder, len(keys)), make([][]uuid.UUID, len(keys))
	for i, k := range keys {
		recorded[i] = k.undeleted(t, pool, web)
		under[i] = rowsUnder{key: k.String(), deletedBefore: len(recorded[i]), keptBefore: k.rows(t, pool, ops)}
	}

	before := time.Now().Truncate(time.Microsecond)
	if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/projects/"+web.String(), alice, ""); status != http.StatusNoContent {
		t.Fatalf("deleting Web = %d %s, want 204", status, body)
	}
	after := time.Now()

	for i, k := range keys {
		under[i].deletedAfter, under[i].keptAfter = len(k.undeleted(t, pool, web)), k.rows(t, pool, ops)
	}
	for _, v := range deletionViolations("project", under, nil) {
		t.Error(v)
	}
	for i, k := range keys {
		if rows := k.unstamped(t, pool, recorded[i], web, aliceID); rows != "" {
			t.Errorf("%s: rows not deleted at the project's deleted_at by its deleter %s:\n%s", k, aliceID, rows)
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
// changes no row under either project. A deferred constraint trigger on
// states refuses it: the last step's table, so a step that committed on
// its own, before it, would leave its rows deleted.
func TestAProjectDeletionRefusedAtItsCommitChangesNoRow(t *testing.T) {
	contract, base, pool, alice, _, web, ops := twoProjects(t)
	rowsOf := func() []string {
		var all []string
		for _, k := range keysTo(t, pool, "projects") {
			for _, id := range []uuid.UUID{web, ops} {
				all = append(all, k.String()+":\n"+k.rows(t, pool, id))
			}
		}
		return all
	}
	before := rowsOf()
	for _, sql := range []string{
		`CREATE FUNCTION refuse_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'the commit is refused'; END $$`,
		`CREATE CONSTRAINT TRIGGER refuse_commit AFTER UPDATE ON states DEFERRABLE INITIALLY DEFERRED
			FOR EACH ROW EXECUTE FUNCTION refuse_commit()`,
	} {
		if _, err := pool.Exec(context.Background(), sql); err != nil {
			t.Fatal(err)
		}
	}
	if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/projects/"+web.String(), alice, ""); status != http.StatusInternalServerError {
		t.Fatalf("deleting Web with its commit refused = %d %s, want 500", status, body)
	}
	if after := rowsOf(); !slices.Equal(after, before) {
		t.Errorf("the rows after the refused deletion:\n%q\nwant\n%q", after, before)
	}
}
