package bootstrap

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// changingNote sends the account name's change of his notification id
// with body in the background, as wholeChange sends it: with its fixed
// X-Request-Id, checked against the contract.
func (w issueWorld) changingNote(t *testing.T, name, id, body string) (*http.Request, <-chan answer) {
	t.Helper()
	req := newRequest(t, http.MethodPatch, w.base+notePath(id), w.tokens[name], []byte(body))
	req.Header.Set("X-Request-Id", "00000000-0000-4000-8000-000000000001")
	w.contract.CheckRequest(t, req)
	return req, sendInBackground(req)
}

// A change of a notification and the deletion of its work item come one
// after the other, through the notification's row (M7 P1c spec 2.7, R5;
// P1b spec 2.8), each order made by the statement probes, never by
// waiting a while. alice makes Gone and Kept, each assigned to bob, who is
// sent the creation of each and his assignment.
//
// The deletion first: a transaction holds the later of Gone's two
// notifications, which alice's deletion of Gone waits for once it has
// locked the earlier (DeleteIssueNotifications). bob's change of the
// earlier, read, finds it his and undeleted without a lock (steps 1 to 3)
// and waits for it under the lock (step 4, LockNotification). Once the
// later is let go the deletion commits, and the change's lock reads the
// row's latest version, deleted: it is answered 404 notification.not_found,
// the very answer to an id that never was, and writes nothing.
//
// The change first: a transaction holds bob's account row FOR UPDATE,
// which bob's change of Kept's first notification, read, waits for once it
// has locked and written the row, at its foreign key's check of
// updated_by_id (UpdateNotification, FOR KEY SHARE); alice's deletion of
// Kept then waits for the row (DeleteIssueNotifications). Once the
// account's row is let go, the change commits, answered 200 and read, and
// the deletion deletes the row, read, with Kept's other, at Kept's
// deletion by alice.
//
// A change that waits for its row past the lock timeout is answered 503
// server_busy: TestTheNotificationAPIThroughTheApp.
func TestAChangeOfANotificationAndTheDeletionOfItsWorkItem(t *testing.T) {
	w := newIssueWorld(t)
	toBob := fmt.Sprintf(`"assignee_ids":%s}`, idArray(w.ids["bob"]))
	gone, kept := w.newWorkItemIn(t, w.web, `{"name":"Gone",`+toBob), w.newWorkItemIn(t, w.web, `{"name":"Kept",`+toBob)
	waitForFanouts(t, w.pool)
	ofGone, ofKept := w.bobsOf(t, gone), w.bobsOf(t, kept)
	if len(ofGone) != 2 || len(ofKept) != 2 {
		t.Fatalf("bob's notifications of Gone %v, of Kept %v; want two of each", ofGone, ofKept)
	}

	t.Run("the deletion first", func(t *testing.T) {
		row := holding(t, w.pool, "SELECT 1 FROM notifications WHERE id = $1 FOR UPDATE", ofGone[1])
		deleted := w.deleting(t, "/api/v0/issues/"+gone.String())
		pgtest.WaitForStatementLockWaitBehind(t, w.pool, row.Conn().PgConn(), 5*time.Second, "DeleteIssueNotifications")
		req, answered := w.changingNote(t, "bob", ofGone[0], `{"read":true}`)
		pgtest.WaitForStatementLockWaitOn(t, w.pool, "notifications", 5*time.Second, "LockNotification")
		release(t, row)
		deleted(t)
		a := receiveWithin(t, answered, 5*time.Second, "answer to bob's change of Gone's notification")
		if a.err != nil {
			t.Fatal(a.err)
		}
		w.contract.CheckResponse(t, req, a.res)
		header := a.res.Header.Clone()
		header.Del("Date")
		status, body, wantHeader := w.wholeChange(t, "bob", uuid.NewV7().String(), `{"read":true}`)
		if a.res.StatusCode != status || string(a.body) != body || !reflect.DeepEqual(header, wantHeader) || status != http.StatusNotFound ||
			refusalOf(t, a.body, false) != "notification.not_found" {
			t.Errorf("bob's change of a notification deleted as it waited = %d %s %v; want an id's that never was, %d %s %v",
				a.res.StatusCode, a.body, header, status, body, wantHeader)
		}
		if bad, n := unstampedNotifications(t, w.pool, gone); bad != 0 || n != 2 {
			t.Errorf("Gone's notifications: %d of %d not deleted at its deletion by alice; want none of 2", bad, n)
		}
		if got := w.storedNote(t, ofGone[0]); !strings.HasPrefix(got, "read_at=null ") {
			t.Errorf("bob's notification of Gone stored %s; want it unread", got)
		}
	})
	t.Run("the change first", func(t *testing.T) {
		account := holding(t, w.pool, "SELECT 1 FROM users WHERE id = $1 FOR UPDATE", w.ids["bob"])
		req, answered := w.changingNote(t, "bob", ofKept[0], `{"read":true}`)
		pgtest.WaitForStatementLockWaitBehind(t, w.pool, account.Conn().PgConn(), 5*time.Second, "UpdateNotification")
		deleted := w.deleting(t, "/api/v0/issues/"+kept.String())
		pgtest.WaitForStatementLockWaitOn(t, w.pool, "notifications", 5*time.Second, "DeleteIssueNotifications")
		release(t, account)
		a := receiveWithin(t, answered, 5*time.Second, "answer to bob's change of Kept's notification")
		if a.err != nil {
			t.Fatal(a.err)
		}
		w.contract.CheckResponse(t, req, a.res)
		var changed apiNotification
		if a.res.StatusCode == http.StatusOK {
			decodeAnswer(t, string(a.body), &changed)
		}
		if a.res.StatusCode != http.StatusOK || changed.ReadAt == nil {
			t.Fatalf("bob's change of a notification as its work item's deletion waits = %d %s; want 200, read", a.res.StatusCode, a.body)
		}
		deleted(t)
		if bad, n := unstampedNotifications(t, w.pool, kept); bad != 0 || n != 2 {
			t.Errorf("Kept's notifications: %d of %d not deleted at its deletion by alice; want none of 2", bad, n)
		}
		var read *time.Time
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT read_at FROM notifications WHERE id = $1", ofKept[0]).Scan(&read); err != nil {
			t.Fatal(err)
		}
		if read == nil || !read.Equal(*changed.ReadAt) {
			t.Errorf("bob's notification of Kept, deleted, read at %v; want as his change answered, %v", read, changed.ReadAt)
		}
	})
}
