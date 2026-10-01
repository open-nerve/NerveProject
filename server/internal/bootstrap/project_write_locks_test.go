package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// projectWrite is a write on a project as
// TestEachWriteOnAProjectSharesItsWorkspaceFirst sends it, by its
// operationId: the request on the project, by alice.
type projectWrite struct {
	op, method, path, body string // path: %s the project's id
	want                   int
}

// projectWrites are the writes on a project, in the order they run on
// Web, then on Ops.
var projectWrites = []projectWrite{
	{op: "updateProject", method: http.MethodPatch, path: "/api/v0/projects/%s", body: `{"description":"Changed"}`, want: http.StatusOK},
	{op: "archiveProject", method: http.MethodPost, path: "/api/v0/projects/%s/archive", want: http.StatusOK},
	{op: "unarchiveProject", method: http.MethodPost, path: "/api/v0/projects/%s/unarchive", want: http.StatusOK},
	{op: "deleteProject", method: http.MethodDelete, path: "/api/v0/projects/%s", want: http.StatusNoContent},
}

// writesOnAProject are the operations whose matrix rows write and have the
// project level's columns: every write on a project of the contract.
func writesOnAProject() []string {
	var ops []string
	for _, r := range matrixRows() {
		if r.write && slices.Equal(r.columns, projectColumns) && !slices.Contains(ops, r.op) {
			ops = append(ops, r.op)
		}
	}
	return slices.Sorted(slices.Values(ops))
}

// holding begins a transaction on pool that runs sql with args, and keeps
// it open until the test rolls it back, or ends.
func holding(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) pgx.Tx {
	t.Helper()
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	if tag, err := tx.Exec(context.Background(), sql, args...); err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("%s: %v, %v; want one row held", sql, tag, err)
	}
	return tx
}

// heldBy reports whether a transaction holds the one row sql locks NOWAIT:
// it answers lock_not_available (55P03) at once while one does.
func heldBy(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) bool {
	t.Helper()
	tag, err := pool.Exec(context.Background(), sql, args...)
	var pgErr *pgconn.PgError
	switch {
	case errors.As(err, &pgErr) && pgErr.Code == "55P03":
		return true
	case err != nil || tag.RowsAffected() != 1:
		t.Fatalf("%s: %v, %v; want one row", sql, tag, err)
	}
	return false
}

// Every write on a project takes its workspace's row FOR SHARE first, in
// its transaction, before any other lock (M3 design 3.6 convention 2, the
// lock table), as bootstrap wires it: alice, acme's admin, writes on her
// projects Web and Ops, each write once on each. Every write on a project
// of the contract has its row here: the matrix's rows that write at the
// project level are the list.
//   - The workspace first: another transaction holds acme's row FOR NO KEY
//     UPDATE, as every cascade over its projects does (3.3). The write on
//     Web waits for that row, and meanwhile does not hold Web's row: a FOR
//     UPDATE NOWAIT of it succeeds.
//   - In its transaction, FOR SHARE, before its project: another
//     transaction holds Ops's row FOR NO KEY UPDATE. The write on Ops waits
//     for it, and meanwhile holds acme's row at FOR SHARE, no stronger,
//     which another write on a project of acme shares (a FOR NO KEY UPDATE
//     NOWAIT of it fails, a FOR SHARE NOWAIT succeeds).
//
// Once the other transaction ends, each write answers as it would alone.
// Each row sends its own operation's request, its method and path as the
// contract has them, so no write's row runs another write instead.
func TestEachWriteOnAProjectSharesItsWorkspaceFirst(t *testing.T) {
	contract := apitest.Load(t)
	ops := make([]string, len(projectWrites))
	for i, w := range projectWrites {
		ops[i] = w.op
		if !slices.ContainsFunc(contract.Operations(), func(o apitest.Operation) bool {
			return o.ID == w.op && o.Method == w.method && o.Path == fmt.Sprintf(w.path, "{project_id}")
		}) {
			t.Fatalf("%s's row sends %s %s, not the contract's %s", w.op, w.method, w.path, w.op)
		}
	}
	if want := writesOnAProject(); !slices.Equal(slices.Sorted(slices.Values(ops)), want) {
		t.Fatalf("the writes here are %q; the writes on a project are %q: each has its row, in projectWrites", slices.Sorted(slices.Values(ops)),
			want)
	}
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	pool := openPool(t, dbURL)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice, `{"name":"Acme","slug":"acme"}`)
	if status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	var acme struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, body, &acme)
	projects := [2]uuid.UUID{createdProject(t, contract, base, alice, "acme", "Web", "WEB"),
		createdProject(t, contract, base, alice, "acme", "Ops", "OPS")}
	for _, w := range projectWrites {
		for phase, project := range projects {
			req := newRequest(t, w.method, base+fmt.Sprintf(w.path, project), alice, []byte(w.body))
			contract.CheckRequest(t, req)
			var other pgx.Tx
			if phase == 0 {
				other = holding(t, pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", acme.ID)
			} else {
				other = holding(t, pool, "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", project)
			}
			answered := sendInBackground(req)
			if phase == 0 {
				pgtest.WaitForLockWaitOn(t, pool, "workspaces", 10*time.Second)
				if heldBy(t, pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE NOWAIT", project) {
					t.Errorf("%s holds its project while it waits for its workspace", w.op)
				}
			} else {
				pgtest.WaitForLockWaitOn(t, pool, "projects", 10*time.Second)
				if !heldBy(t, pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE NOWAIT", acme.ID) ||
					heldBy(t, pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR SHARE NOWAIT", acme.ID) {
					t.Errorf("%s does not hold its workspace FOR SHARE in its transaction while it waits for its project", w.op)
				}
			}
			if err := other.Rollback(context.Background()); err != nil {
				t.Fatal(err)
			}
			a := receiveWithin(t, answered, 10*time.Second, "the answer to "+w.op)
			if a.err != nil {
				t.Fatal(a.err)
			}
			contract.CheckResponse(t, req, a.res)
			if a.res.StatusCode != w.want {
				t.Errorf("%s on %s = %d %s, want %d", w.op, []string{"Web", "Ops"}[phase], a.res.StatusCode, a.body, w.want)
			}
		}
	}
}
