package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The notification module declares its request kind and registers its
// worker, with no periodic schedule; leaving the worker out fails the
// registration.
func TestNewAppNotificationJobCatalog(t *testing.T) {
	kinds := []string{"notification.delete_rows"}
	if got := notification.EnqueuedKinds(); !slices.Equal(got, kinds) {
		t.Fatalf("enqueued kinds=%v; want %v", got, kinds)
	}
	pool, err := postgres.NewPool(t.Context(), config.DatabaseConfig{URL: unreachableDB, MaxConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	definitions := notification.New(notification.Deps{Pool: pool, DeleteBatch: 1000, DeleteBudget: 45 * time.Second, RuntimeNow: time.Now,
		Logger: slog.New(slog.DiscardHandler)}).Jobs()
	if len(definitions) != 1 || definitions[0].Worker == nil || definitions[0].Periodic != nil {
		t.Fatalf("notification jobs=%+v; want the deletion's worker alone", definitions)
	}
	cfg := jobs.Config{DefaultWorkers: 1, PeriodicWorkers: 1, Logger: slog.New(slog.DiscardHandler)}
	if _, err := jobs.New(pool, cfg, definitions, kinds); err != nil {
		t.Fatalf("module request worker registration: %v", err)
	}
	if _, err := jobs.New(pool, cfg, nil, kinds); err == nil {
		t.Fatal("notification.delete_rows declared without its worker registered")
	}
}

// notificationsOf is the notifications of the work items, deleted ones too,
// each as text, in order.
func notificationsOf(t *testing.T, pool *pgxpool.Pool, items ...uuid.UUID) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT coalesce(string_agg(n::text, E'\\n' ORDER BY n.id), '') FROM notifications n WHERE n.issue_id = ANY ($1)", items).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// unstampedNotifications counts the notifications of the deleted work items
// not deleted at their work item's deleted_at by its deleter, and all of
// them.
func unstampedNotifications(t *testing.T, pool *pgxpool.Pool, items ...uuid.UUID) (bad, all int) {
	t.Helper()
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT count(*) FILTER (WHERE i.deleted_at IS NULL OR n.deleted_at IS DISTINCT FROM i.deleted_at OR n.updated_at IS DISTINCT FROM i.deleted_at
			OR n.updated_by_id IS DISTINCT FROM i.updated_by_id), count(*)
		FROM notifications n JOIN issues i ON i.id = n.issue_id WHERE n.issue_id = ANY ($1)`, items).Scan(&bad, &all); err != nil {
		t.Fatal(err)
	}
	return bad, all
}

// A work item's deletion deletes the notifications of each work item it
// deletes, in its transaction (M7 P1b spec 2.8): a subscriber after
// notification's sees them deleted in the transaction at the event's moment
// by its actor and live outside it, and its failure rolls them back with
// the work item. Through the wired app, a subtree's deletion, Parent with
// Child, deletes both's at the work items' deleted_at by alice, and leaves
// Sibling's, of the same project; then Sibling's own deletion deletes its.
func TestDeletingAWorkItemDeletesItsNotificationsInItsTransaction(t *testing.T) {
	w := newIssueWorld(t)
	parent, child := w.newWorkItemIn(t, w.web, `{"name":"Parent"}`), w.newWorkItemIn(t, w.web, `{"name":"Child"}`)
	sibling := w.newWorkItemIn(t, w.web, `{"name":"Sibling"}`)
	underWorkItem(t, w.pool, parent, child)
	for _, item := range []uuid.UUID{parent, child, sibling} {
		seedNotification(t, w.pool, item, w.ids["bob"])
	}
	seedNotification(t, w.pool, parent, w.ids["walt"])
	before, siblings := notificationsOf(t, w.pool, parent, child, sibling), notificationsOf(t, w.pool, sibling)

	client, err := jobs.NewInserter(w.pool)
	if err != nil {
		t.Fatal(err)
	}
	fault := errors.New("later subscriber failed")
	seen := 0
	subscribers := append(notification.Subscribers(notification.Provide(w.pool), jobInserter{client: client}), shared.EventSubscriber{
		Name: "fail_after_notification", Types: []string{"issue.deleted"}, Handle: func(ctx context.Context, e shared.Event) error {
			seen++
			var inside, outside, all int
			if err := postgres.DB(ctx, w.pool).QueryRow(ctx, "SELECT count(*) FILTER (WHERE deleted_at = $2 AND updated_at = $2 AND updated_by_id = $3), count(*) FROM notifications WHERE issue_id = $1",
				e.Entity.ID, e.OccurredAt, w.ids["alice"]).Scan(&inside, &all); err != nil {
				return err
			}
			if err := w.pool.QueryRow(ctx, "SELECT count(*) FROM notifications WHERE issue_id = $1 AND deleted_at IS NULL", e.Entity.ID).Scan(&outside); err != nil {
				return err
			}
			if all == 0 || inside != all || outside != all {
				t.Errorf("work item %s's notifications: %d of %d deleted by the event in its transaction, %d live outside it; want all, all", e.Entity.ID, inside, all, outside)
			}
			return fault
		}})
	events, err := newDispatcher(eventTypes(), subscribers)
	if err != nil {
		t.Fatal(err)
	}
	route := newIssueRouteWithPublisher(t, w.pool, postgres.NewTxManager(w.pool, time.Second), clock.System{}, events)
	var response exchange
	if err := w.sending(&response, route, "alice", http.MethodDelete, "/api/v0/issues/"+parent.String(), "")(); err != nil {
		t.Fatal(err)
	}
	if response.rec.Code != http.StatusInternalServerError || seen != 1 {
		t.Fatalf("a deletion whose later subscriber failed = %d, seen %d; want 500 after the first work item's event", response.rec.Code, seen)
	}
	if got := notificationsOf(t, w.pool, parent, child, sibling); got != before {
		t.Fatalf("the notifications escaped the rollback:\n%s\nwant\n%s", got, before)
	}

	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+parent.String(), "", http.StatusNoContent)
	if bad, all := unstampedNotifications(t, w.pool, parent, child); bad != 0 || all != 3 {
		t.Fatalf("the subtree's notifications: %d of %d not deleted at their work item's deleted_at by its deleter; want none of 3", bad, all)
	}
	if got := notificationsOf(t, w.pool, sibling); got != siblings {
		t.Fatalf("Sibling's notification after Parent's deletion:\n%s\nwant it as it was:\n%s", got, siblings)
	}
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+sibling.String(), "", http.StatusNoContent)
	if bad, all := unstampedNotifications(t, w.pool, sibling); bad != 0 || all != 1 {
		t.Fatalf("Sibling's notification: %d of %d not deleted at its deleted_at by its deleter; want none of 1", bad, all)
	}
}

// While issue.delete_rows has not deleted a deleted project's work items,
// its notification.delete_rows only waits (M7 P1b spec 2.8, R3, acceptance
// "交错"): the job's snoozes grow, each run its first attempt, with no
// failure, and no notification is written; each snooze schedules the next
// run the wait's half second after the run started at the earliest, read
// as the database holds both moments, with half that as the margin for the
// clocks of River and PostgreSQL (the budget's JobSnooze(0) would schedule
// it as the run ends, and River's fetch interval alone hides the
// difference in the snoozes' rate); once issue.delete_rows runs,
// the job completes in its first attempt and the notifications are deleted
// at the project's deleted_at by its deleter. issue.delete_rows is held by
// deferring it as it is inserted (setDeletionControl), so no lock is held
// and nothing times out meanwhile.
func TestAProjectDeletionDeletesItsNotificationsOnceItsWorkItemsAre(t *testing.T) {
	f := newDeletionLifecycle(t, func(*config.Config) {}, time.Now, nil)
	w := f.w
	item := w.newWorkItemIn(t, w.web, `{"name":"Held"}`)
	seedNotification(t, w.pool, item, w.ids["bob"])
	seedNotification(t, w.pool, item, w.ids["walt"])
	before := notificationsOf(t, w.pool, item)
	setDeletionControl(t, w.pool, true, false)
	w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+w.web.String(), "", http.StatusNoContent)
	held := projectDeletionJob(t, w.pool, w.web)

	var job notificationJob
	var snoozes []int
	var waits []time.Duration
	requestEventuallyWithin(t, "notification deletion snoozing", 20*time.Second, func(ctx context.Context) bool {
		var ok bool
		if job, ok = notificationDeletionJob(ctx, t, w.pool, w.web); !ok {
			return false
		}
		if job.Attempt > 1 || job.Failures != 0 || !slices.Contains([]string{"available", "running"}, job.State) {
			t.Fatalf("notification deletion job while the work items wait: %+v; want it snoozing in its first attempt", job)
		}
		if got := notificationsOf(t, w.pool, item); got != before {
			t.Fatalf("the notifications changed while the work items wait:\n%s\nwant\n%s", got, before)
		}
		if len(snoozes) == 0 || job.Snoozes > snoozes[len(snoozes)-1] {
			snoozes = append(snoozes, job.Snoozes)
		}
		var wait time.Duration
		if err := w.pool.QueryRow(ctx, "SELECT scheduled_at - attempted_at FROM river_job WHERE id = $1 AND state = 'available' AND attempted_at IS NOT NULL", job.ID).Scan(&wait); err == nil {
			if wait < 250*time.Millisecond {
				t.Fatalf("a snooze scheduled the next run %s after the run started; want the wait's half second", wait)
			}
			waits = append(waits, wait)
		} else if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatal(err)
		}
		return len(snoozes) >= 3 && len(waits) > 0
	})
	for _, tr := range jobTransitions(t, w.pool, job.ID) {
		if tr.To == "running" && tr.Attempt != 1 {
			t.Fatalf("a run of the waiting job in attempt %d: %+v", tr.Attempt, jobTransitions(t, w.pool, job.ID))
		}
	}
	t.Logf("notification deletion job=%d snoozes observed %v, last wait %s, while issue.delete_rows %d was held", job.ID, snoozes, waits[len(waits)-1], held.ID)

	setDeletionControl(t, w.pool, false, false)
	releaseDeletionJob(t, w.pool, held, 0)
	waitDeletionJob(t, w.pool, held, "completed")
	waitNotificationDeletion(t, w.pool, w.web)
	var bad, all int
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*) FILTER (WHERE n.deleted_at IS DISTINCT FROM p.deleted_at OR n.updated_at IS DISTINCT FROM p.deleted_at
			OR n.updated_by_id IS DISTINCT FROM $2), count(*) FROM notifications n JOIN projects p ON p.id = n.project_id WHERE n.issue_id = $1`, item, w.ids["alice"]).Scan(&bad, &all); err != nil ||
		bad != 0 || all != 2 {
		t.Fatalf("the project's notifications: %d of %d not deleted at its deleted_at by alice, %v; want none of 2", bad, all, err)
	}
}
