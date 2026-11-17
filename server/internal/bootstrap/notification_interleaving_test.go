package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// eventNotifications counts the notifications of the event, deleted ones too, and the live ones among them.
func eventNotifications(t *testing.T, pool *pgxpool.Pool, event uuid.UUID) (all, live int) {
	t.Helper()
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT count(*), count(*) FILTER (WHERE deleted_at IS NULL) FROM notifications WHERE event_id = $1",
		event).Scan(&all, &live); err != nil {
		t.Fatal(err)
	}
	return all, live
}

// deleting sends alice's deletion of path through the API in the background; deleted checks that it answered 204.
func (w issueWorld) deleting(t *testing.T, path string) func(t *testing.T) {
	t.Helper()
	req := newRequest(t, http.MethodDelete, w.base+path, w.tokens["alice"], nil)
	answered := sendInBackground(req)
	return func(t *testing.T) {
		t.Helper()
		got := receiveWithin(t, answered, 10*time.Second, "the deletion of "+path)
		if got.err != nil {
			t.Fatal(got.err)
		}
		w.contract.CheckResponse(t, req, got.res)
		if got.res.StatusCode != http.StatusNoContent {
			t.Fatalf("DELETE %s = %d %s; want 204", path, got.res.StatusCode, got.body)
		}
	}
}

// A fan-out and the deletion of its work item or of its project come one after the other, through the work item's row
// (M7 P1b spec 2.8, 2.9, R3), each ordering made by the statement probes, never by waiting a while. alice renames a work
// item bob is assigned to, mentioning him in its description, which sends him his mention; its fan-out is held until
// the test holds what sets the order, then released.
//
// The fan-out first: a transaction holds bob's account row FOR UPDATE, which the fan-out's insert, its foreign key's
// FOR KEY SHARE, waits for after it has locked the work item's row FOR SHARE; alice's deletion of the work item then
// waits for that row (DeleteIssueRows). Once the account's row is let go, the fan-out writes bob's notification and
// commits, and the deletion's transaction deletes it with the work item's others, at its moment, by alice: none of the
// work item's is left live.
//
// The deletion first: a transaction holds one of the work item's notifications FOR UPDATE, which alice's deletion, in
// its transaction, waits for once it has deleted the work item's row (DeleteIssueNotifications); the fan-out, released,
// then waits for that row (ShareIssue). Once the notification is let go the deletion commits, and the fan-out, reading
// the row again, finds the work item deleted, writes nothing and completes. bob's mention is the event's, not a
// subscription the deletion deletes too: only the fan-out's recheck keeps it from being sent.
//
// A project's deletion: the fan-out waits for bob's account row as above, holding the work item's row, when alice
// deletes Ops; issue.delete_rows waits for the work item's row (DeleteProjectIssueRows), and notification.delete_rows,
// finding the project's work items not deleted, snoozes. Once the account's row is let go the fan-out commits its
// notification, issue.delete_rows deletes the work item, and notification.delete_rows deletes the project's
// notifications, that one too, at the project's deletion, in its first attempt: its snoozes consumed none. No
// notification of Ops is left live.
func TestAFanoutAndTheDeletionOfItsWorkItemOrProject(t *testing.T) {
	w := newHeldFanoutWorld(t, func(*config.Config) {})
	bob := w.ids["bob"]
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.makeMember(t, "bob", ops, shared.RoleMember)
	assigned := fmt.Sprintf(`{"name":"Fix","assignee_ids":[%q]}`, bob)
	first, second, third := w.newWorkItemIn(t, w.web, assigned), w.newWorkItemIn(t, w.web, assigned), w.newWorkItemIn(t, ops, assigned)
	releaseFanouts(t, w.pool)
	waitForFanouts(t, w.pool)
	// renamed is the fan-out of alice's renaming of item mentioning bob, held: the event it carries, once it has completed
	// in its first attempt.
	renaming := fmt.Sprintf(`{"name":"Fix it","description_html":%q}`, mentionHTML(bob))
	renamed := func(t *testing.T, item uuid.UUID) func(t *testing.T) uuid.UUID {
		t.Helper()
		w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+item.String(), renaming, http.StatusOK)
		return func(t *testing.T) uuid.UUID {
			t.Helper()
			waitForFanouts(t, w.pool)
			if runs := fanoutRunsOf(t, w.pool, "issue.updated", item); !reflect.DeepEqual(runs, []fanoutRun{{"completed", 1, 0}}) {
				t.Fatalf("the renaming's fan-outs %+v; want one completed in its first attempt", runs)
			}
			return fanoutOf(t, w.pool, "issue.updated", item).EventID
		}
	}
	release := func(t *testing.T) {
		t.Helper()
		if n := releaseFanouts(t, w.pool); n != 1 {
			t.Fatalf("released %d fan-outs; want the renaming's", n)
		}
	}
	letGo := func(t *testing.T, tx interface{ Rollback(context.Context) error }) {
		t.Helper()
		if err := tx.Rollback(pgtest.Soon(t)); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("the fan-out first", func(t *testing.T) {
		completed := renamed(t, first)
		account := holding(t, w.pool, "SELECT 1 FROM users WHERE id = $1 FOR UPDATE", bob)
		release(t)
		pgtest.WaitForStatementLockWaitBehind(t, w.pool, account.Conn().PgConn(), 5*time.Second, "InsertNotifications")
		deleted := w.deleting(t, "/api/v0/issues/"+first.String())
		pgtest.WaitForStatementLockWaitOn(t, w.pool, "issues", 5*time.Second, "DeleteIssueRows")
		letGo(t, account)
		deleted(t)
		event := completed(t)
		if all, live := eventNotifications(t, w.pool, event); all != 1 || live != 0 {
			t.Errorf("the renaming's notifications: %d, %d live; want bob's, deleted", all, live)
		}
		if bad, all := unstampedNotifications(t, w.pool, first); bad != 0 || all != 3 {
			t.Errorf("the work item's notifications: %d of %d not deleted at its deletion by alice; want none of 3", bad, all)
		}
	})
	t.Run("the deletion first", func(t *testing.T) {
		completed := renamed(t, second)
		notification := holding(t, w.pool, "SELECT 1 FROM notifications WHERE issue_id = $1 AND change_seq = -1 FOR UPDATE", second)
		deleted := w.deleting(t, "/api/v0/issues/"+second.String())
		pgtest.WaitForStatementLockWaitBehind(t, w.pool, notification.Conn().PgConn(), 5*time.Second, "DeleteIssueNotifications")
		release(t)
		pgtest.WaitForStatementLockWaitOn(t, w.pool, "issues", 5*time.Second, "ShareIssue")
		letGo(t, notification)
		deleted(t)
		if all, _ := eventNotifications(t, w.pool, completed(t)); all != 0 {
			t.Errorf("the renaming's notifications: %d; want none", all)
		}
		if bad, all := unstampedNotifications(t, w.pool, second); bad != 0 || all != 2 {
			t.Errorf("the work item's notifications: %d of %d not deleted at its deletion by alice; want none of 2", bad, all)
		}
	})
	t.Run("a project's deletion", func(t *testing.T) {
		completed := renamed(t, third)
		account := holding(t, w.pool, "SELECT 1 FROM users WHERE id = $1 FOR UPDATE", bob)
		release(t)
		pgtest.WaitForStatementLockWaitBehind(t, w.pool, account.Conn().PgConn(), 5*time.Second, "InsertNotifications")
		w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+ops.String(), "", http.StatusNoContent)
		pgtest.WaitForStatementLockWaitOn(t, w.pool, "issues", 5*time.Second, "DeleteProjectIssueRows")
		requestEventuallyWithin(t, "the notifications' deletion waiting for the work items", 10*time.Second, func(ctx context.Context) bool {
			job, ok := notificationDeletionJob(ctx, t, w.pool, ops)
			return ok && (job.Snoozes > 0 || job.State == "completed")
		})
		letGo(t, account)
		waitProjectDeletion(t, w.pool, ops)
		event := completed(t)
		if job, _ := notificationDeletionJob(pgtest.Soon(t), t, w.pool, ops); job.Snoozes == 0 || job.Attempt != 1 {
			t.Errorf("the notifications' deletion %+v; want it to have snoozed in its first attempt", job)
		}
		var bad, renaming, all int
		if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*) FILTER (WHERE n.deleted_at IS DISTINCT FROM p.deleted_at OR n.updated_at IS DISTINCT FROM p.deleted_at
				OR n.updated_by_id IS DISTINCT FROM $2), count(*) FILTER (WHERE n.event_id = $3), count(*)
			FROM notifications n JOIN projects p ON p.id = n.project_id WHERE n.project_id = $1`, ops, w.ids["alice"], event).Scan(&bad, &renaming, &all); err != nil {
			t.Fatal(err)
		}
		if bad != 0 || renaming != 1 || all != 3 {
			t.Errorf("Ops' notifications: %d of %d not deleted at its deletion by alice, %d of the renaming; want none of 3, 1", bad, all, renaming)
		}
	})
}
