package bootstrap

import (
	"context"
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
// method and its path as the contract has them; target, the path it
// sends, a path of that pattern (pathOf), to Web of the world or to a row
// made in Web for it, made anew for each of alice's sends, before any lock
// is held, so that a write that ends what it names finds it there; body,
// its body, which names the world's rows, read as the test sends it, with
// a plain read, which no lock the test holds stops; the answer it gets;
// waits, the table it waits for under its lock before any write of its
// own: the first it reads there that its precheck did not (held ACCESS
// EXCLUSIVE), or, when it reads none, the first it writes (held SHARE,
// which its precheck's plain read does not wait for); and mode, the lock
// another transaction holds of that table: ACCESS EXCLUSIVE, which a plain
// read waits for too, or SHARE, which a write waits for and a plain read
// does not, so that the write waits there and its precheck does not.
type issueWrite struct {
	op, method, path string
	target           func(t *testing.T, w issueWorld) string
	body             func(t *testing.T, w issueWorld) string
	want             int
	waits, mode      string
}

// issueWrites are the work-item side's writes, each with its row: every
// operation of the contract tagged issue but GET (writesOnIssues).
var issueWrites = []issueWrite{
	{op: "createIssueLink", method: http.MethodPost, path: "/api/v0/issues/{issue_id}/links", target: func(t *testing.T, w issueWorld) string { return linksOf(w.newWorkItem(t)) }, body: func(*testing.T, issueWorld) string { return `{"url":"http://169.254.169.254/locks"}` }, want: http.StatusCreated, waits: "issues", mode: "SHARE"},
	{op: "addCommentReaction", method: http.MethodPost, path: "/api/v0/issue-comments/{comment_id}/reactions", target: func(t *testing.T, w issueWorld) string {
		return commentReactionsOf(w.newComment(t, "alice", w.newWorkItem(t)).ID)
	}, body: func(*testing.T, issueWorld) string { return `{"reaction":"128077"}` }, want: http.StatusCreated, waits: "comment_reactions", mode: "SHARE"},
	{op: "removeCommentReaction", method: http.MethodDelete, path: "/api/v0/issue-comments/{comment_id}/reactions/{reaction}", target: func(t *testing.T, w issueWorld) string { return w.newCommentReactionPath(t) }, body: func(*testing.T, issueWorld) string { return "" }, want: http.StatusNoContent, waits: "comment_reactions", mode: "SHARE"},
	{op: "addIssueReaction", method: http.MethodPost, path: "/api/v0/issues/{issue_id}/reactions", target: func(t *testing.T, w issueWorld) string { return reactionsOf(w.newWorkItem(t)) }, body: func(*testing.T, issueWorld) string { return `{"reaction":"128077"}` }, want: http.StatusCreated, waits: "issues", mode: "SHARE"},
	{op: "removeIssueReaction", method: http.MethodDelete, path: "/api/v0/issues/{issue_id}/reactions/{reaction}", target: func(t *testing.T, w issueWorld) string { return w.newReactionPath(t) }, body: func(*testing.T, issueWorld) string { return "" }, want: http.StatusNoContent, waits: "issues", mode: "SHARE"},
	{op: "updateIssueComment", method: http.MethodPatch, path: "/api/v0/issue-comments/{comment_id}", target: func(t *testing.T, w issueWorld) string {
		return commentPath(w.newComment(t, "alice", w.newWorkItem(t)).ID)
	}, body: func(*testing.T, issueWorld) string { return `{"comment_html":"<p>Updated</p>"}` }, want: http.StatusOK, waits: "issues", mode: "SHARE"},
	{op: "deleteIssueComment", method: http.MethodDelete, path: "/api/v0/issue-comments/{comment_id}", target: func(t *testing.T, w issueWorld) string {
		return commentPath(w.newComment(t, "alice", w.newWorkItem(t)).ID)
	}, body: func(*testing.T, issueWorld) string { return "" }, want: http.StatusNoContent, waits: "issues", mode: "SHARE"},
	{op: "createIssueComment", method: http.MethodPost, path: "/api/v0/issues/{issue_id}/comments", target: func(t *testing.T, w issueWorld) string { return commentsOf(w.newWorkItem(t)) }, body: func(*testing.T, issueWorld) string { return `{"comment_html":"<p>Hello</p>"}` }, want: http.StatusCreated, waits: "issues", mode: "SHARE"},
	{op: "createIssue", method: http.MethodPost, path: "/api/v0/projects/{project_id}/issues",
		target: func(_ *testing.T, w issueWorld) string { return "/api/v0/projects/" + w.web.String() + "/issues" },
		body:   func(_ *testing.T, w issueWorld) string { return w.fullIssue() }, want: http.StatusCreated, waits: "states", mode: "ACCESS EXCLUSIVE"},
	// A work item in Todo, which the send moves to Done: a change of
	// state, so the states are read under the lock.
	{op: "updateIssue", method: http.MethodPatch, path: "/api/v0/issues/{issue_id}",
		target: func(t *testing.T, w issueWorld) string { return "/api/v0/issues/" + w.newWorkItem(t).String() },
		body: func(_ *testing.T, w issueWorld) string {
			return `{"sort_order":0,"description_html":"<p>Under lock</p>","state_id":"` + w.done.String() + `"}`
		}, want: http.StatusOK,
		waits: "states", mode: "ACCESS EXCLUSIVE"},
	// A work item the send deletes: it reads nothing under the lock that
	// its precheck did not, so it waits for the work items' table, which
	// it writes first.
	{op: "deleteIssue", method: http.MethodDelete, path: "/api/v0/issues/{issue_id}",
		target: func(t *testing.T, w issueWorld) string { return "/api/v0/issues/" + w.newWorkItem(t).String() },
		body:   func(*testing.T, issueWorld) string { return "" }, want: http.StatusNoContent, waits: "issues", mode: "SHARE"},
	// A work item in Done, which the send archives: its state is read
	// under the lock.
	{op: "archiveIssue", method: http.MethodPost, path: "/api/v0/issues/{issue_id}/archive",
		target: func(t *testing.T, w issueWorld) string {
			return "/api/v0/issues/" + w.doneWorkItem(t).String() + "/archive"
		},
		body: func(*testing.T, issueWorld) string { return "" }, want: http.StatusOK, waits: "states", mode: "ACCESS EXCLUSIVE"},
	// An archived work item, which the send unarchives: it reads nothing
	// under the lock that its precheck did not, so it waits for the work
	// items' table, which it writes first.
	{op: "unarchiveIssue", method: http.MethodPost, path: "/api/v0/issues/{issue_id}/unarchive",
		target: func(t *testing.T, w issueWorld) string {
			id := w.doneWorkItem(t)
			w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+id.String()+"/archive", "", http.StatusOK)
			return "/api/v0/issues/" + id.String() + "/unarchive"
		},
		body: func(*testing.T, issueWorld) string { return "" }, want: http.StatusOK, waits: "issues", mode: "SHARE"},
	// A new work item, which the send puts Part under: Web's one work item
	// named so, made with the first target; each send moves it under the
	// next. It reads nothing under the lock that its precheck did not, so it
	// waits for the work items' table, which it writes first.
	{op: "addSubIssues", method: http.MethodPost, path: "/api/v0/issues/{issue_id}/sub-issues",
		target: func(t *testing.T, w issueWorld) string {
			if len(queryIDs(t, w.pool, "SELECT id FROM issues WHERE project_id = $1 AND name = 'Part'", w.web)) == 0 {
				w.newWorkItemIn(t, w.web, `{"name":"Part"}`)
			}
			return subIssues(w.newWorkItem(t))
		},
		body: func(t *testing.T, w issueWorld) string { return subIssueIDs(w.workItemNamed(t, "Part")) }, want: http.StatusOK,
		waits: "issues", mode: "SHARE"},
	// A new work item, which the send relates to Far: Web's one work item
	// named so, made with the first target. Its precheck reads no relation,
	// so it waits for the relations' table, which it reads first under
	// the lock.
	{op: "createIssueRelations", method: http.MethodPost, path: "/api/v0/issues/{issue_id}/relations",
		target: func(t *testing.T, w issueWorld) string {
			if len(queryIDs(t, w.pool, "SELECT id FROM issues WHERE project_id = $1 AND name = 'Far'", w.web)) == 0 {
				w.newWorkItemIn(t, w.web, `{"name":"Far"}`)
			}
			return relationsOf(w.newWorkItem(t))
		},
		body: func(t *testing.T, w issueWorld) string { return relating("relates_to", w.workItemNamed(t, "Far")) }, want: http.StatusCreated,
		waits: "issue_relations", mode: "ACCESS EXCLUSIVE"},
	// A relation of two new work items, which the send deletes: it reads
	// nothing under the lock that its precheck did not, so it waits for the
	// relations' table, which it writes first.
	{op: "deleteIssueRelation", method: http.MethodDelete, path: "/api/v0/issue-relations/{relation_id}",
		target: func(t *testing.T, w issueWorld) string {
			related := listIn[answeredRelation](t, w.send(t, "alice", http.MethodPost, relationsOf(w.newWorkItem(t)),
				relating("relates_to", w.newWorkItem(t)), http.StatusCreated))
			return relationPath(related[0].ID)
		},
		body: func(*testing.T, issueWorld) string { return "" }, want: http.StatusNoContent, waits: "issue_relations", mode: "SHARE"},
}

// issueTables are the tables a write on the work-item side reads or
// writes but users, which it reads without a lock and key-shares only
// through its foreign keys, after its project's lock: its project's, the
// memberships its decision reads, the states and labels it checks, and
// the work-item side's own.
const issueTables = "workspaces, workspace_members, projects, project_members, states, labels, issues, issue_assignees, issue_labels, " +
	"issue_links, issue_relations, issue_subscribers, issue_mentions, issue_comments, issue_reactions, comment_reactions, issue_activities, issue_description_versions"

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
// (issueWrites), which a write without one fails before any database; a
// row whose target is not a path of its operation (pathOf) fails before
// it sends, so that no row tests another operation under its name.
// Each write, on Web of the world (newIssueWorld):
//   - Refused, no lock: another transaction holds every table it reads or
//     writes but users (issueTables) IN EXCLUSIVE MODE, which a row lock
//     and a write wait for and a plain read does not. walt and wanda, who
//     see Web but may not write on its work items, are answered 403, and
//     olga, who does not see it, 404, while it holds them.
//   - Its project first: alice's write, under the same locks, waits for
//     the projects table, no other.
//   - FOR NO KEY UPDATE by its lock, and nothing of the workspace: another
//     transaction holds the table the write waits for under its lock
//     (waits), in its mode (mode): the first it reads there that its
//     precheck did not, or, when it reads none, the first it writes.
//     alice's write waits for it before any write of its own: it holds no
//     table's ROW EXCLUSIVE lock, which every write takes (writtenTables),
//     so what it holds of Web is its lock's, not an UPDATE's. It holds
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
			return o.ID == iw.op && o.Method == iw.method && o.Path == iw.path
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
			// alice's three sends' targets; the callers refused send her
			// first's.
			var targets []string
			for range 3 {
				target := iw.target(t, w)
				if !pathOf(iw.path, target) {
					t.Fatalf("%s's row sends %s %s, not a path of %s", iw.op, iw.method, target, iw.path)
				}
				targets = append(targets, target)
			}
			alices := 0
			send := func(name string) <-chan answer {
				target := targets[0]
				if name == "alice" {
					target = targets[alices]
					alices++
				}
				req := newRequest(t, iw.method, w.base+target, w.tokens[name], []byte(iw.body(t, w)))
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
				if iw.op == "updateIssue" && want == http.StatusOK && a.res.StatusCode == want {
					got := sortAnswer(t, string(a.body))
					stored := sortStored(t, w, got.ID)
					if got.SortOrder != 0 || stored.SortOrder != 0 {
						t.Fatal("locked update lost sort", got, stored)
					}
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

			waited := lockedTables(t, w.pool, iw.waits, iw.mode)
			answered = send("alice")
			pgtest.WaitForLockWaitBehind(t, w.pool, waited.Conn().PgConn(), 10*time.Second)
			if table := waitedTable(t, w.pool, waited.Conn().PgConn()); table != iw.waits {
				t.Fatalf("%s waits for %s, not for %s", iw.op, table, iw.waits)
			}
			if written := writtenTables(t, w.pool, waited.Conn().PgConn()); len(written) > 0 {
				t.Errorf("%s has written %q when it waits for %s: what it holds of Web is no longer its lock's alone", iw.op, written, iw.waits)
			}
			if lock := lockOn(t, w.pool, "projects WHERE id = $1", w.web); lock != "FOR NO KEY UPDATE" {
				t.Errorf("%s holds Web %s when it waits for %s, not FOR NO KEY UPDATE", iw.op, lock, iw.waits)
			}
			if lock := lockOn(t, w.pool, "workspaces WHERE id = $1", w.acme); lock != "no lock" {
				t.Errorf("%s holds acme %s when it waits for %s", iw.op, lock, iw.waits)
			}
			release(t, waited)
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

// writtenTables are the relations the backend blocked behind holder has
// written in its transaction, sorted: the tables and indexes it holds ROW
// EXCLUSIVE, the lock every INSERT, UPDATE and DELETE takes of them and no
// read or row lock does.
func writtenTables(t *testing.T, pool *pgxpool.Pool, holder *pgconn.PgConn) []string {
	t.Helper()
	const written = `SELECT coalesce(array_agg(DISTINCT h.relation::regclass::text ORDER BY h.relation::regclass::text), '{}')
		FROM pg_locks h
		WHERE h.granted AND h.locktype = 'relation' AND h.mode = 'RowExclusiveLock'
		  AND h.pid IN (SELECT l.pid FROM pg_locks l WHERE NOT l.granted AND $1::int = ANY (pg_blocking_pids(l.pid)))`
	var tables []string
	if err := pool.QueryRow(pgtest.Soon(t), written, int64(holder.PID())).Scan(&tables); err != nil {
		t.Fatalf("the tables written behind backend %d: %v", holder.PID(), err)
	}
	return tables
}
