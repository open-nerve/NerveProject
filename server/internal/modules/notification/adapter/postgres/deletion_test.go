package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// itemNotification inserts the live notification id of the work item issue of project for the world's receiver.
func itemNotification(t *testing.T, pool *pgxpool.Pool, w world, project, issue, id uuid.UUID) {
	t.Helper()
	exec(t, pool, `INSERT INTO notifications (id, workspace_id, project_id, receiver_id, issue_id, sender, data, event_id, change_seq, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'subscribed', '{"issue": {}, "change": {}}', gen_random_uuid(), 0, $6, $6)`, id, w.workspace, project, w.receiver, issue, now)
}

// stamps reads the deletion, the last write and its writer of each notification, in the order of ids, as text:
// "live" for no deletion, "nerve" for no writer.
func stamps(t *testing.T, pool *pgxpool.Pool, ids ...uuid.UUID) []string {
	t.Helper()
	got := make([]string, len(ids))
	for n, id := range ids {
		var deleted *time.Time
		var updated time.Time
		var by *uuid.UUID
		if err := pool.QueryRow(context.Background(), "SELECT deleted_at, updated_at, updated_by_id FROM notifications WHERE id = $1", id).Scan(&deleted, &updated, &by); err != nil {
			t.Fatal(err)
		}
		got[n] = stamp(deleted, updated, by)
	}
	return got
}

// stamp is stamps' text of one notification.
func stamp(deleted *time.Time, updated time.Time, by *uuid.UUID) string {
	text := "live"
	if deleted != nil {
		text = deleted.UTC().Format(time.RFC3339Nano)
	}
	text += " " + updated.UTC().Format(time.RFC3339Nano)
	if by == nil {
		return text + " nerve"
	}
	return text + " " + by.String()
}

func newDeleter(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, 'dave@corp.com', 'x', 'dave')", id)
	return id
}

// A work item's deletion soft-deletes its undeleted notifications at the deletion's moment by its deleter, nerve's
// own deletion (no deleter) too; a notification deleted before keeps its deletion, and another work item's in the
// same project stays (M7 P1b spec 2.8). It writes in the transaction ctx carries.
func TestDeleteIssueNotifications(t *testing.T) {
	s, pool := newStore(t)
	w := newWorld(t, pool)
	dave := newDeleter(t, pool)
	item, sibling, auto := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	ids := []uuid.UUID{purgeID(1), purgeID(2), purgeID(3)}
	for _, id := range slices.Backward(ids) {
		itemNotification(t, pool, w, w.project, item, id)
	}
	earlier := purgeID(4)
	itemNotification(t, pool, w, w.project, item, earlier)
	exec(t, pool, "UPDATE notifications SET deleted_at = $2, updated_at = $2 WHERE id = $1", earlier, now.Add(time.Hour))
	kept := purgeID(5)
	itemNotification(t, pool, w, w.project, sibling, kept)
	nerve := purgeID(6)
	itemNotification(t, pool, w, w.project, auto, nerve)
	before := stamps(t, pool, earlier, kept)

	at := now.Add(2 * time.Hour)
	rollback := errors.New("rollback the caller")
	err := postgres.NewTxManager(pool, time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		if err := s.DeleteIssueNotifications(ctx, w.workspace, item, app.DeletionStamp{At: at, By: &dave}); err != nil {
			t.Fatal(err)
		}
		return rollback
	})
	live := stamp(nil, now, nil)
	if got := stamps(t, pool, ids...); !errors.Is(err, rollback) || !slices.Equal(got, []string{live, live, live}) {
		t.Fatalf("a deletion the caller rolled back: %v %v; want the notifications as they were: %s", err, got, live)
	}
	if err := s.DeleteIssueNotifications(context.Background(), w.workspace, item, app.DeletionStamp{At: at, By: &dave}); err != nil {
		t.Fatal(err)
	}
	want := stamp(&at, at, &dave)
	if got := stamps(t, pool, ids...); !slices.Equal(got, []string{want, want, want}) {
		t.Fatalf("the work item's notifications=%v; want each %s", got, want)
	}
	if got := stamps(t, pool, earlier, kept); !slices.Equal(got, before) {
		t.Fatalf("deleted before and another work item's=%v; want them as they were: %v", got, before)
	}
	if err := s.DeleteIssueNotifications(context.Background(), w.workspace, auto, app.DeletionStamp{At: at}); err != nil {
		t.Fatal(err)
	}
	if got := stamps(t, pool, nerve)[0]; got != stamp(&at, at, nil) {
		t.Fatalf("nerve's own deletion=%s; want no deleter", got)
	}
	for _, bad := range []app.DeletionStamp{{}, {At: at, By: &uuid.UUID{}}} {
		if err := s.DeleteIssueNotifications(context.Background(), w.workspace, sibling, bad); err == nil {
			t.Fatalf("accepted the stamp %+v", bad)
		}
	}
	if got := stamps(t, pool, kept)[0]; got != before[1] {
		t.Fatalf("a refused stamp wrote %s", got)
	}
}

// A deleted project's notifications are soft-deleted a batch at a time after the cursor, in id order, at the
// deletion's moment by its deleter, and the batch answers their ids; another project's and one deleted before stay.
func TestDeleteProjectNotifications(t *testing.T) {
	s, pool := newStore(t)
	w := newWorld(t, pool)
	dave := newDeleter(t, pool)
	other := uuid.NewV7()
	exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, 'Ops', 'OPS')", other, w.workspace)
	for _, n := range []int{5, 4, 3, 2, 1} {
		itemNotification(t, pool, w, w.project, uuid.NewV7(), purgeID(n))
	}
	exec(t, pool, "UPDATE notifications SET deleted_at = $2, updated_at = $2 WHERE id = $1", purgeID(2), now.Add(time.Hour))
	itemNotification(t, pool, w, other, uuid.NewV7(), purgeID(6))
	before := stamps(t, pool, purgeID(2), purgeID(6))
	at := now.Add(2 * time.Hour)
	batch := func(after uuid.UUID, want ...uuid.UUID) {
		t.Helper()
		got, err := s.DeleteProjectNotifications(context.Background(), app.RowBatch{ProjectID: w.project, After: after, By: dave, At: at, Limit: 2})
		slices.SortFunc(got, uuid.UUID.Compare)
		if err != nil || !slices.Equal(got, want) {
			t.Fatalf("batch after %s=%v,%v; want %v", after, got, err, want)
		}
	}
	batch(uuid.UUID{}, purgeID(1), purgeID(3))
	batch(purgeID(3), purgeID(4), purgeID(5))
	batch(purgeID(5))
	want := stamp(&at, at, &dave)
	if got := stamps(t, pool, purgeID(1), purgeID(3), purgeID(4), purgeID(5)); !slices.Equal(got, []string{want, want, want, want}) {
		t.Fatalf("the project's notifications=%v; want each %s", got, want)
	}
	if got := stamps(t, pool, purgeID(2), purgeID(6)); !slices.Equal(got, before) {
		t.Fatalf("deleted before and another project's=%v; want %v", got, before)
	}
	for _, limit := range []int{0, 2147483648} {
		if _, err := s.DeleteProjectNotifications(context.Background(), app.RowBatch{ProjectID: w.project, By: dave, At: at, Limit: limit}); err == nil {
			t.Fatalf("accepted the batch %d", limit)
		}
	}
}

// Both deletions lock their rows in id order before they write them (M3 design 3.6, convention 5): with the rows
// written in the reverse of their id order and the middle one held, each waits on it holding the lower one and not
// yet the higher one. A statement that locked in its scan's order would hold the higher one.
func TestNotificationDeletionsLockInIDOrder(t *testing.T) {
	for _, tt := range []struct {
		name   string
		delete func(*postgresadapter.Store, context.Context, world, uuid.UUID) error
	}{
		{"work item", func(s *postgresadapter.Store, ctx context.Context, w world, item uuid.UUID) error {
			return s.DeleteIssueNotifications(ctx, w.workspace, item, app.DeletionStamp{At: now})
		}},
		{"project", func(s *postgresadapter.Store, ctx context.Context, w world, _ uuid.UUID) error {
			_, err := s.DeleteProjectNotifications(ctx, app.RowBatch{ProjectID: w.project, By: w.receiver, At: now, Limit: 10})
			return err
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, pool := newStore(t)
			w := newWorld(t, pool)
			item := uuid.NewV7()
			for _, n := range []int{3, 2, 1} {
				itemNotification(t, pool, w, w.project, item, purgeID(n))
			}
			holder, err := pool.Begin(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = holder.Rollback(context.Background()) }()
			if _, err := holder.Exec(context.Background(), "SELECT 1 FROM notifications WHERE id = $1 FOR UPDATE", purgeID(2)); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				done <- postgres.NewTxManager(pool, 10*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
					return tt.delete(s, ctx, w, item)
				})
			}()
			pgtest.WaitForLockWaitBehind(t, pool, holder.Conn().PgConn(), 10*time.Second)
			if !pgtest.RowWaits(t, pool, "notifications", purgeID(1), "FOR UPDATE") || pgtest.RowWaits(t, pool, "notifications", purgeID(3), "FOR UPDATE") {
				t.Fatal("the deletion waits on the middle row without holding the lowest id alone; want the rows locked in id order")
			}
			if err := holder.Rollback(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			var live int
			if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM notifications WHERE deleted_at IS NULL").Scan(&live); err != nil || live != 0 {
				t.Fatalf("live notifications after the deletion=%d,%v; want none", live, err)
			}
		})
	}
}
