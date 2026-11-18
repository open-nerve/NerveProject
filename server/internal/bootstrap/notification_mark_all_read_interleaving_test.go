package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	notificationriver "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// markPromptly bounds the answer to a mark-all-read that passes over the
// rows other transactions hold: well inside the request's deadline
// (testConfig's 5s) and the lock timeout of a work item's deletion
// (testIssueConfig's 4s), which waits meanwhile.
const markPromptly = 2 * time.Second

// sendingMark sends the account name's mark-all-read with body in the
// background, the request checked against the contract.
func (w issueWorld) sendingMark(t *testing.T, name, body string) (*http.Request, <-chan answer) {
	t.Helper()
	req := newRequest(t, http.MethodPost, w.base+inboxPath+"/mark-all-read", w.tokens[name], []byte(body))
	w.contract.CheckRequest(t, req)
	return req, sendInBackground(req)
}

// markedOf is the marked of a, the answer to the mark-all-read req, as
// the contract declares it: -1, and the test's error, for an answer other
// than 200.
func (w issueWorld) markedOf(t *testing.T, what string, req *http.Request, a answer) int {
	t.Helper()
	if a.err != nil {
		t.Fatal(a.err)
	}
	w.contract.CheckResponse(t, req, a.res)
	if a.res.StatusCode != http.StatusOK {
		t.Errorf("%s = %d %s; want 200", what, a.res.StatusCode, a.body)
		return -1
	}
	var marked int
	decodeAnswer(t, string(listKeys(t, a.body, "marked")["marked"]), &marked)
	return marked
}

// answeredUnlessWaiting is the answer answered yields, true; or false as
// soon as a backend of pool's database that runs statement waits for a row
// before it is answered: for a row's tuple lock, or for the transaction
// holding the row. Neither within 5s fails the test.
func answeredUnlessWaiting(t *testing.T, pool *pgxpool.Pool, answered <-chan answer, statement string) (answer, bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(5 * time.Millisecond) {
		select {
		case a := <-answered:
			return a, true
		default:
		}
		var waiting int
		if err := pool.QueryRow(pgtest.Soon(t), `SELECT count(*) FROM pg_stat_activity WHERE datname = current_database()
			AND wait_event_type = 'Lock' AND wait_event IN ('tuple', 'transactionid') AND `+pgtest.StatementIn(1), []string{statement}).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting > 0 {
			return answer{}, false
		}
		if time.Now().After(deadline) {
			t.Fatalf("no answer, and no %s waiting for a row, within 5s", statement)
		}
	}
}

// bobsOf are the ids of bob's notifications of item, in id order.
func (w issueWorld) bobsOf(t *testing.T, item uuid.UUID) []string {
	t.Helper()
	var ids []string
	for _, id := range queryIDs(t, w.pool, "SELECT id FROM notifications WHERE receiver_id = $1 AND issue_id = $2 ORDER BY id", w.ids["bob"], item) {
		ids = append(ids, id.String())
	}
	return ids
}

// Mark-all-read passes over the notifications a work item's deletion
// holds, never waiting for them (M7 P1c spec 2.7, R5; re-84 I1). The
// deletion soft-deletes its subtree's notifications in its transaction
// work item by work item, in the subtree's id order (Subtree, an
// issue.deleted each, DeleteIssueNotifications), each work item's in id
// order: not in one id order over the subtree. alice makes Side and
// Parent, then Child under Parent, Side and Child assigned to bob, and
// only then assigns Parent to bob as she renames it. So Parent's id is
// before Child's, and the deletion of Parent locks Parent's notifications
// before Child's; but bob's of Child, sent first, have ids before his of
// Parent. bob reads the last of Parent's, which a transaction then holds:
// alice's deletion of Parent locks Parent's others and waits for it,
// holding them. A mark-all-read that waited for rows, locking bob's
// unread ones in one id order, would lock Child's and wait for Parent's;
// let go, the deletion would wait for Child's: a cycle, 40P01, one of the
// two answered 500. bob's mark-all-read instead answers at once, having
// waited for no row, marks Side's and Child's, and leaves Parent's unread.
// Let go, the deletion deletes Parent's and Child's at its moment by
// alice, read or not.
//
// A fan-out in flight meanwhile: walt's comment on Side, made before the
// mark, has its fan-out write bob's notification and wait, uncommitted,
// for walt's account row, which a transaction holds FOR UPDATE (its
// foreign key's FOR KEY SHARE). The mark's batch neither sees that row
// nor waits for it, and does not mark it; once the fan-out commits, in its
// first attempt, the row is unread, of an event before the mark's moment,
// and the next mark-all-read marks it. With T11's late row, committed
// after the mark read its moment and before its batch, which is marked: a
// batch marks the rows committed when it reads, of an event no later than
// that moment (re-84 I2).
func TestMarkAllReadPassesOverTheNotificationsAWorkItemsDeletionHolds(t *testing.T) {
	w := newHeldFanoutWorld(t, func(*config.Config) {})
	settle := func() {
		t.Helper()
		releaseFanouts(t, w.pool)
		waitForFanouts(t, w.pool)
	}
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	toBob := fmt.Sprintf(`"assignee_ids":%s}`, idArray(w.ids["bob"]))
	side, parent := w.newWorkItemIn(t, w.web, `{"name":"Side",`+toBob), w.newWorkItemIn(t, w.web, `{"name":"Parent"}`)
	settle()
	child := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Child","parent_id":%q,`, parent)+toBob)
	settle()
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+parent.String(), `{"name":"Parent renamed",`+toBob, http.StatusOK)
	settle()
	ofSide, ofParent, ofChild := w.bobsOf(t, side), w.bobsOf(t, parent), w.bobsOf(t, child)
	lastOfParent := queryIDs(t, w.pool, "SELECT id FROM notifications WHERE issue_id = $1 ORDER BY id DESC LIMIT 1", parent)
	if len(ofSide) == 0 || len(ofChild) == 0 || len(ofParent) < 2 || parent.String() >= child.String() ||
		ofChild[len(ofChild)-1] >= ofParent[0] || lastOfParent[0].String() != ofParent[len(ofParent)-1] {
		t.Fatalf("bob's notifications of Side %v, of Child %v, of Parent %v, Parent's last %v; Parent %s, Child %s: want Parent's id before "+
			"Child's, bob's of Child before his of Parent, two of Parent at least, the last of which is the last of Parent's",
			ofSide, ofChild, ofParent, lastOfParent, parent, child)
	}
	last, held := ofParent[len(ofParent)-1], ofParent[:len(ofParent)-1]
	w.changeNote(t, "bob", last, `{"read":true}`)
	if got, want := w.unreadRows(t, "bob"), sortedIDs(slices.Concat(ofSide, ofChild, held)...); !slices.Equal(got, want) {
		t.Fatalf("bob's unread notifications %v; want %v", got, want)
	}

	w.send(t, "walt", http.MethodPost, commentsOf(side), `{"comment_html":"<p>Meanwhile</p>"}`, http.StatusCreated)
	account := holding(t, w.pool, "SELECT 1 FROM users WHERE id = $1 FOR UPDATE", w.ids["walt"])
	if n := releaseFanouts(t, w.pool); n != 1 {
		t.Fatalf("released %d fan-outs; want walt's comment's", n)
	}
	pgtest.WaitForStatementLockWaitBehind(t, w.pool, account.Conn().PgConn(), 5*time.Second, "InsertNotifications")
	row := holding(t, w.pool, "SELECT 1 FROM notifications WHERE id = $1 FOR UPDATE", last)
	deleted := w.deleting(t, "/api/v0/issues/"+parent.String())
	pgtest.WaitForStatementLockWaitBehind(t, w.pool, row.Conn().PgConn(), 5*time.Second, "DeleteIssueNotifications")

	req, answered := w.sendingMark(t, "bob", `{}`)
	sent := time.Now()
	a, unwaited := answeredUnlessWaiting(t, w.pool, answered, "MarkNotificationsRead")
	took := time.Since(sent)
	if unwaited {
		if left := w.unreadRows(t, "bob"); !slices.Equal(left, held) {
			t.Errorf("bob's unread notifications after his mark-all-read %v; want those of Parent the deletion holds, %v", left, held)
		}
	}
	release(t, row)
	if !unwaited {
		t.Errorf("bob's mark-all-read waited for a row while alice's deletion of Parent held Parent's notifications; want it to pass over them")
		a = receiveWithin(t, answered, 5*time.Second, "answer to bob's mark-all-read")
		took = time.Since(sent)
	}
	if marked := w.markedOf(t, "bob's mark-all-read", req, a); marked != len(ofSide)+len(ofChild) || took > markPromptly {
		t.Errorf("bob's mark-all-read marked %d in %v; want Side's and Child's, %d, within %v", marked, took, len(ofSide)+len(ofChild), markPromptly)
	}
	t.Logf("bob's mark-all-read answered in %v", took)
	deleted(t)
	release(t, account)
	settle()

	var retried int
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*) FROM river_job WHERE kind = $1 AND (attempt > 1 OR coalesce(cardinality(errors), 0) > 0)`,
		notificationriver.FanoutKind).Scan(&retried); err != nil || retried != 0 {
		t.Errorf("%d fan-outs failed or retried, %v; want none", retried, err)
	}
	if bad, n := unstampedNotifications(t, w.pool, parent, child); bad != 0 || n != len(ofParent)+len(ofChild) {
		t.Errorf("Parent's and Child's notifications: %d of %d not deleted at the deletion by alice; want none of %d", bad, n, len(ofParent)+len(ofChild))
	}
	var unreadHeld, readChild int
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*) FILTER (WHERE id = ANY ($1::uuid[]) AND read_at IS NULL),
			count(*) FILTER (WHERE id = ANY ($2::uuid[]) AND read_at IS NOT NULL) FROM notifications`, held, ofChild).Scan(&unreadHeld, &readChild); err != nil {
		t.Fatal(err)
	}
	if unreadHeld != len(held) || readChild != len(ofChild) {
		t.Errorf("of Parent's the deletion held %d unread, of Child's %d read; want %d and %d", unreadHeld, readChild, len(held), len(ofChild))
	}
	comment := w.noteIDOf(t, "bob", side, "comment created")
	var before bool
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT coalesce(c.created_at < s.read_at, false) FROM notifications c, notifications s WHERE c.id = $1 AND s.id = $2",
		comment, ofSide[0]).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if unread := w.unreadRows(t, "bob"); !slices.Equal(unread, []string{comment}) || !before {
		t.Errorf("bob's unread notifications %v, walt's comment's before the mark's moment %t; want the comment's alone, %s, before it", unread, before, comment)
	}
	if marked := w.markedBy(t, "bob", `{}`); marked != 1 {
		t.Errorf("bob's next mark-all-read marked %d; want walt's comment's", marked)
	}
}

// Mark-all-read and a project's deletion of its notifications
// (notification.delete_rows) both complete with no deadlock (M7 P1c spec
// 2.7, R5; P1b spec 2.8): each batch of either locks its rows in one id
// order, and mark-all-read never waits for a row. Here the mark holds them
// first, and they are marked, counted, and deleted after it. alice makes
// Fix in Web and Far in Ops, each assigned to bob, Ops' member too, who is
// sent the creation of each and his assignment. A transaction holds bob's
// account row FOR UPDATE: bob's mark-all-read, which read the projects he
// reads before Ops is deleted, locks and marks his four and waits at its
// foreign key's check of updated_by_id (FOR KEY SHARE). alice deletes Ops,
// and its notification.delete_rows, once Far is deleted, waits for Far's
// notifications the mark holds. Let go, the mark answers 4, and the
// deletion of the notifications completes in its first attempt with no
// failure: Far's are read at the mark's moment and deleted after it, at
// Ops' deletion by alice; Fix's are read and live. The other order needs
// no test of its own: the rows a batch of notification.delete_rows holds
// are passed over as a work item's deletion's are (the test above), and a
// mark-all-read that reads the projects once Ops is deleted reads none of
// its notifications (spec 2.5).
func TestMarkAllReadAndAProjectsDeletionOfItsNotifications(t *testing.T) {
	w := newIssueWorldWith(t, func(*config.Config) {})
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.makeMember(t, "bob", ops, shared.RoleMember)
	toBob := fmt.Sprintf(`"assignee_ids":%s}`, idArray(w.ids["bob"]))
	fix, far := w.newWorkItemIn(t, w.web, `{"name":"Fix",`+toBob), w.newWorkItemIn(t, ops, `{"name":"Far",`+toBob)
	waitForFanouts(t, w.pool)
	ofFix, ofFar := w.bobsOf(t, fix), w.bobsOf(t, far)
	if got, want := w.unreadRows(t, "bob"), sortedIDs(slices.Concat(ofFix, ofFar)...); len(ofFix) != 2 || len(ofFar) != 2 || !slices.Equal(got, want) {
		t.Fatalf("bob's unread notifications %v; want Fix's two and Far's two, %v", got, want)
	}

	account := holding(t, w.pool, "SELECT 1 FROM users WHERE id = $1 FOR UPDATE", w.ids["bob"])
	req, answered := w.sendingMark(t, "bob", `{}`)
	pgtest.WaitForStatementLockWaitBehind(t, w.pool, account.Conn().PgConn(), 5*time.Second, "MarkNotificationsRead")
	w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+ops.String(), "", http.StatusNoContent)
	pgtest.WaitForStatementLockWaitOn(t, w.pool, "notifications", 10*time.Second, "DeleteProjectNotifications")
	release(t, account)
	if marked := w.markedOf(t, "bob's mark-all-read", req, receiveWithin(t, answered, 5*time.Second, "answer to bob's mark-all-read")); marked != 4 {
		t.Errorf("bob's mark-all-read marked %d; want Fix's and Far's, 4", marked)
	}
	waitNotificationDeletion(t, w.pool, ops)

	var bad, live int
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT
			count(*) FILTER (WHERE n.project_id = $3 AND (n.read_at IS NULL OR n.read_at >= p.deleted_at OR n.deleted_at IS DISTINCT FROM p.deleted_at
				OR n.updated_at IS DISTINCT FROM p.deleted_at OR n.updated_by_id IS DISTINCT FROM $4)),
			count(*) FILTER (WHERE n.project_id <> $3 AND n.read_at IS NOT NULL AND n.deleted_at IS NULL)
		FROM notifications n JOIN projects p ON p.id = $3 WHERE n.id = ANY ($1::uuid[]) OR n.id = ANY ($2::uuid[])`,
		ofFar, ofFix, ops, w.ids["alice"]).Scan(&bad, &live); err != nil {
		t.Fatal(err)
	}
	if bad != 0 || live != len(ofFix) {
		t.Errorf("of Far's notifications %d not read before Ops' deletion and deleted at it by alice, of Fix's %d read and live; want none, %d",
			bad, live, len(ofFix))
	}
	if marked := w.markedBy(t, "bob", `{}`); marked != 0 {
		t.Errorf("bob's mark-all-read once Ops is deleted marked %d; want none", marked)
	}
}
