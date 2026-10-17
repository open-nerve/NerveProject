package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// issueWrite is a write on the work-item side as
// TestEachIssueWriteLocksItsProjectFirst sends it: its operation, its
// method and its path as the contract has them, with %s for Web's id; its
// body, which names the world's rows; the answer it gets; and reads, the
// first table it reads under its project's lock that its precheck did not
// read, before any write of its own.
type issueWrite struct {
	op, method, path string
	body             func(w issueWorld) string
	want             int
	reads            string
}

// issueWrites are the work-item side's writes, each with its row: every
// operation of the contract tagged issue but GET (writesOnIssues).
var issueWrites = []issueWrite{
	{op: "createIssue", method: http.MethodPost, path: "/api/v0/projects/%s/issues", body: issueWorld.fullIssue, want: http.StatusCreated,
		reads: "states"},
}

// issueTables are the tables a write on the work-item side reads or
// writes but users, whose display names it reads without a lock: its
// project's, the memberships its decision reads, the states and labels it
// checks, and the work-item side's own.
const issueTables = "workspaces, workspace_members, projects, project_members, states, labels, issues, issue_assignees, issue_labels, " +
	"issue_subscribers, issue_activities"

// writesOnIssues lists the operations of ops tagged issue, but GET, sorted.
func writesOnIssues(ops []apitest.Operation) []string {
	var writes []string
	for _, op := range ops {
		if op.Method != http.MethodGet && slices.Contains(op.Tags, "issue") {
			writes = append(writes, op.ID)
		}
	}
	return slices.Sorted(slices.Values(writes))
}

// Each write on the work-item side, as bootstrap wires it, takes its
// project's row FOR NO KEY UPDATE as its first lock, and its workspace's
// row no more than a foreign key's FOR KEY SHARE; a caller it refuses
// takes no lock at all (M4 design 3.5, decision point 1: B; 9.3,
// interleaving 21). Every write of the contract has its row here
// (issueWrites), which a write without one fails before any database.
// Each write, on Web of the world (newIssueWorld):
//   - Refused, no lock: another transaction holds every table it reads or
//     writes but users (issueTables) IN EXCLUSIVE MODE, which a row lock
//     and a write wait for and a plain read does not. walt and wanda, who
//     see Web but may not write on its work items, are answered 403, and
//     olga, who does not see it, 404, while it holds them.
//   - Its project first: alice's write, under the same locks, waits for
//     the projects table, no other.
//   - FOR NO KEY UPDATE by its lock, and nothing of the workspace: another
//     transaction holds the first table the write reads under its lock
//     that its precheck did not (reads) IN ACCESS EXCLUSIVE MODE, which a
//     plain read waits for too.
//     alice's write waits for it before any write of its own, holding
//     Web's row FOR NO KEY UPDATE, no weaker and no stronger, and acme's
//     row not at all (lockOn).
//   - No lock of the workspace that a cascade's waits for: another
//     transaction holds acme's row FOR NO KEY UPDATE, as every cascade over
//     its projects does (M3 design 3.3); alice's write answers meanwhile.
//
// Each wait has a deadline; a write that waited where it may not fails
// there.
func TestEachIssueWriteLocksItsProjectFirst(t *testing.T) {
	contract := apitest.Load(t)
	ops := make([]string, len(issueWrites))
	for i, iw := range issueWrites {
		ops[i] = iw.op
		if !slices.ContainsFunc(contract.Operations(), func(o apitest.Operation) bool {
			return o.ID == iw.op && o.Method == iw.method && o.Path == fmt.Sprintf(iw.path, "{project_id}")
		}) {
			t.Fatalf("%s's row sends %s %s, not the contract's %s", iw.op, iw.method, iw.path, iw.op)
		}
	}
	if want := writesOnIssues(contract.Operations()); !slices.Equal(slices.Sorted(slices.Values(ops)), want) {
		t.Fatalf("the writes here are %q; the writes on the work-item side are %q: each has its row, in issueWrites", ops, want)
	}
	w := newIssueWorld(t)
	for _, iw := range issueWrites {
		t.Run(iw.op, func(t *testing.T) {
			path, body := w.base+fmt.Sprintf(iw.path, w.web), iw.body(w)
			send := func(name string) <-chan answer {
				req := newRequest(t, iw.method, path, w.tokens[name], []byte(body))
				w.contract.CheckRequest(t, req)
				return sendInBackground(req)
			}
			answers := func(name string, answered <-chan answer, want int) {
				t.Helper()
				a := receiveWithin(t, answered, 10*time.Second, "the answer to "+iw.op+" as "+name)
				if a.err != nil {
					t.Fatal(a.err)
				}
				if a.res.StatusCode != want {
					t.Errorf("%s as %s = %d %s, want %d", iw.op, name, a.res.StatusCode, a.body, want)
				}
			}

			tables := lockedTables(t, w.pool, issueTables, "EXCLUSIVE")
			for name, want := range map[string]int{"walt": http.StatusForbidden, "wanda": http.StatusForbidden, "olga": http.StatusNotFound} {
				answers(name, send(name), want)
			}
			answered := send("alice")
			pgtest.WaitForLockWaitBehind(t, w.pool, tables.Conn().PgConn(), 10*time.Second)
			if table := waitedTable(t, w.pool, tables.Conn().PgConn()); table != "projects" {
				t.Errorf("%s first waits for %s, not for projects", iw.op, table)
			}
			release(t, tables)
			answers("alice", answered, iw.want)

			read := lockedTables(t, w.pool, iw.reads, "ACCESS EXCLUSIVE")
			answered = send("alice")
			pgtest.WaitForLockWaitBehind(t, w.pool, read.Conn().PgConn(), 10*time.Second)
			if table := waitedTable(t, w.pool, read.Conn().PgConn()); table != iw.reads {
				t.Fatalf("%s waits for %s, not for %s", iw.op, table, iw.reads)
			}
			if lock := lockOn(t, w.pool, "projects WHERE id = $1", w.web); lock != "FOR NO KEY UPDATE" {
				t.Errorf("%s holds Web %s when it reads %s, not FOR NO KEY UPDATE", iw.op, lock, iw.reads)
			}
			if lock := lockOn(t, w.pool, "workspaces WHERE id = $1", w.acme); lock != "no lock" {
				t.Errorf("%s holds acme %s when it reads %s", iw.op, lock, iw.reads)
			}
			release(t, read)
			answers("alice", answered, iw.want)

			cascade := holding(t, w.pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", w.acme)
			answers("alice", send("alice"), iw.want)
			release(t, cascade)
		})
	}
}

// lockedTables begins a transaction on pool that holds tables, a list,
// in mode until it is released or the test ends.
func lockedTables(t *testing.T, pool *pgxpool.Pool, tables, mode string) pgx.Tx {
	t.Helper()
	tx, err := pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	if _, err := tx.Exec(pgtest.Soon(t), "LOCK TABLE "+tables+" IN "+mode+" MODE"); err != nil {
		t.Fatal(err)
	}
	return tx
}

// release rolls tx back.
func release(t *testing.T, tx pgx.Tx) {
	t.Helper()
	if err := tx.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
}

// waitedTable is the table whose lock the backend blocked behind holder
// waits for.
func waitedTable(t *testing.T, pool *pgxpool.Pool, holder *pgconn.PgConn) string {
	t.Helper()
	var table string
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT l.relation::regclass::text FROM pg_locks l
		WHERE NOT l.granted AND l.locktype = 'relation' AND $1::int = ANY (pg_blocking_pids(l.pid))`, int64(holder.PID())).Scan(&table); err != nil {
		t.Fatalf("the table waited for behind backend %d: %v", holder.PID(), err)
	}
	return table
}
