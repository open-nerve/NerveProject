package bootstrap

import (
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// checkShareLocks observes a subscription row's declared shape (M7 P1a spec
// 2.4) at real PostgreSQL waits, on Web of the world, each send on a work
// item made for it before any lock is held: one of bob's for
// subscribeIssue, which alice does not subscribe to, one of hers for
// unsubscribeIssue, which she does (the creator's subscription).
//   - Refused, no lock: another transaction holds every table the work-item
//     side reads or writes but users (issueTables) IN EXCLUSIVE MODE, which a
//     row lock and a write wait for and a plain read does not. walt and
//     wanda, who see Web but are not of it, are answered 403, and olga, who
//     does not see it, 404, while it holds them.
//   - The work item's row first: alice's send, under the same locks, waits
//     for the work items' table, which its FOR SHARE takes, having written
//     nothing: not for the projects' table, nor the workspaces'.
//   - Its row FOR SHARE, and nothing of the project or the workspace:
//     another transaction holds the subscribers' table IN SHARE MODE, which
//     her write of her row there waits for and a plain read does not.
//     Waiting there, having written nothing, her send holds the work item's
//     row FOR SHARE, no weaker and no stronger, and neither Web's row nor
//     acme's (requestRowLock: her transaction's locks, no other's).
//   - No place in the gate: the observed sends' gate counts each place they
//     take, and they take none; each answers row.want, and her subscription
//     is then live (subscribeIssue), or not (unsubscribeIssue).
//   - Not behind the project's writes: other transactions hold Web's row FOR
//     NO KEY UPDATE, as every write on its work items does, and acme's row
//     FOR NO KEY UPDATE, as every cascade over its projects does; her send
//     answers meanwhile.
func (row subscriptionWriteLock) checkShareLocks(t *testing.T, w issueWorld) {
	t.Helper()
	subscribes := row.op == "subscribeIssue"
	target := func(t *testing.T) uuid.UUID {
		t.Helper()
		id := w.newWorkItemIn(t, w.web, `{"name":"Watched"}`)
		if subscribes {
			id = w.newWorkItemBy(t, "bob", w.web, `{"name":"Watched"}`)
		}
		if !pathOf(row.path, row.pathTo(id)) {
			t.Fatalf("%s's row sends %s, not a path of %s", row.op, row.pathTo(id), row.path)
		}
		if aliceSubscribes(t, w, id) == subscribes {
			t.Fatalf("%s's target %s: alice's subscription live=%v before the send", row.op, id, subscribes)
		}
		return id
	}
	answers := func(name string, id uuid.UUID, want int) {
		t.Helper()
		req := newRequest(t, row.method, w.base+row.pathTo(id), w.tokens[name], nil)
		w.contract.CheckRequest(t, req)
		a := receiveWithin(t, sendInBackground(req), 10*time.Second, "the answer to "+row.op+" as "+name)
		if a.err != nil {
			t.Fatal(a.err)
		}
		w.contract.CheckResponse(t, req, a.res)
		if a.res.StatusCode != want {
			t.Errorf("%s as %s = %d %s, want %d", row.op, name, a.res.StatusCode, a.body, want)
		}
	}
	observe := func(id uuid.UUID, held pgx.Tx) (*requestProbe, func()) {
		tx := requestTx(w.pool)
		resources := &collaborationWriteResources{}
		route := newIssueRouteWithGate(t, w.pool, tx, clock.System{}, discardIssueEvents{}, resources)
		var e exchange
		collect := requestBackground(t, w.sending(&e, route, "alice", row.method, row.pathTo(id), ""), func() { requestRelease(t, held) })
		return tx, func() {
			collect()
			w.answered(t, e, row.want)
			if resources.gates != 0 {
				t.Errorf("%s took %d places in the write gate", row.op, resources.gates)
			}
			if got := aliceSubscribes(t, w, id); got != subscribes {
				t.Errorf("%s: alice's subscription live=%v after it, want %v", row.op, got, subscribes)
			}
		}
	}

	first := target(t)
	// The earlier writes' and the targets' notification fan-outs lock their
	// work items' rows FOR SHARE and, inserting, acme's and Web's FOR KEY
	// SHARE: one still running as the tables are locked in turn would
	// deadlock with them (R10). Each hold below waits for them to end,
	// after its target's creation has enqueued its own.
	waitForFanouts(t, w.pool)
	tables := lockedTables(t, w.pool, issueTables, "EXCLUSIVE")
	for name, want := range map[string]int{"walt": http.StatusForbidden, "wanda": http.StatusForbidden, "olga": http.StatusNotFound} {
		answers(name, first, want)
	}
	tx, collect := observe(first, tables)
	pid := requestTableBlocked(t, w.pool, tx, tables.Conn().PgConn().PID(), "issues")
	if written := requestWrittenTables(t, w.pool, pid); len(written) > 0 {
		t.Errorf("%s has written %q when it waits for the work items' table", row.op, written)
	}
	requestRelease(t, tables)
	collect()

	second := target(t)
	waitForFanouts(t, w.pool)
	subscribers := lockedTables(t, w.pool, "issue_subscribers", "SHARE")
	tx, collect = observe(second, subscribers)
	pid = requestTableBlocked(t, w.pool, tx, subscribers.Conn().PgConn().PID(), "issue_subscribers")
	if written := requestWrittenTables(t, w.pool, pid); len(written) > 0 {
		t.Errorf("%s has written %q when it waits for the subscribers' table", row.op, written)
	}
	for _, lock := range []struct {
		name, from, want string
		id               uuid.UUID
	}{{"the work item", "issues WHERE id = $1", row.issue, second}, {"Web", "projects WHERE id = $1", row.project, w.web},
		{"acme", "workspaces WHERE id = $1", row.workspace, w.acme}} {
		if got := requestRowLock(t, w.pool, w.pool, tx, lock.from, lock.id); got != lock.want {
			t.Errorf("%s holds %s %s when it waits for the subscribers' table, want %s", row.op, lock.name, got, lock.want)
		}
	}
	requestRelease(t, subscribers)
	collect()

	third := target(t)
	waitForFanouts(t, w.pool)
	project := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", w.web)
	cascade := holding(t, w.pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", w.acme)
	answers("alice", third, row.want)
	release(t, cascade)
	release(t, project)
	if got := aliceSubscribes(t, w, third); got != subscribes {
		t.Errorf("%s: alice's subscription live=%v after it, want %v", row.op, got, subscribes)
	}
}

// pathTo is row's path to the work item id.
func (row subscriptionWriteLock) pathTo(id uuid.UUID) string {
	return strings.Replace(row.path, "{issue_id}", id.String(), 1)
}

// aliceSubscribes reports whether alice has a live subscription to the
// work item id.
func aliceSubscribes(t *testing.T, w issueWorld, id uuid.UUID) bool {
	t.Helper()
	var live bool
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT EXISTS (SELECT 1 FROM issue_subscribers
  WHERE issue_id = $1 AND subscriber_id = $2 AND deleted_at IS NULL)`, id, w.ids["alice"]).Scan(&live); err != nil {
		t.Fatal(err)
	}
	return live
}
