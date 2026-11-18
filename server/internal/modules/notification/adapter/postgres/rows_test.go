package postgresadapter_test

import (
	"context"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// stored17 is note 17 as a read of it alone answers it.
func stored17(w readWorld) app.StoredNotification {
	later := now.Add(time.Hour)
	return app.StoredNotification{Notification: app.Notification{ID: purgeID(17), WorkspaceID: w.workspace, ProjectID: w.project, IssueID: purgeID(1017),
		Sender: domain.SenderMentioned, TriggeredByID: &w.alice, Data: noteData(17), State: domain.State{ReadAt: &minuteAgo, ArchivedAt: &minuteAgo,
			SnoozedTill: &later}, CreatedAt: now.Add(-17 * time.Minute)}, IssueCreatedByID: &w.alice}
}

// A receiver's notification is read alone, unlocked or locked in the transaction ctx carries, with its work item's
// creator, nil when not known, whatever project it is of (M7 P1c spec 2.7, steps 1 and 4: visibility is the use
// case's); another receiver's, a deleted one and none at all are alike not found.
func TestReadANotificationAlone(t *testing.T) {
	s, pool := newStore(t)
	w := newReadWorld(t, pool)
	put(t, pool, w, readNotes(w)...)
	tx := postgres.NewTxManager(pool, time.Second)
	for name, read := range map[string]func(context.Context, uuid.UUID, uuid.UUID) (app.StoredNotification, bool, error){
		"unlocked": s.GetNotification, "locked": s.LockNotification} {
		t.Run(name, func(t *testing.T) {
			err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
				got, found, err := read(ctx, purgeID(17), w.receiver)
				if want := stored17(w); err != nil || !found || !reflect.DeepEqual(got, want) {
					t.Fatalf("read %+v, %v, %v; want %+v", got, found, err, want)
				}
				if got, found, err := read(ctx, purgeID(11), w.receiver); err != nil || !found || got.IssueCreatedByID != nil || got.ProjectID != w.ops {
					t.Fatalf("an unknown creator's in an own-only project: %+v, %v, %v; want it, of no creator", got, found, err)
				}
				for _, miss := range []struct {
					what         string
					id, receiver uuid.UUID
				}{{"carol's", purgeID(15), w.receiver}, {"his, as carol", purgeID(1), w.carol}, {"deleted", purgeID(14), w.receiver},
					{"none", uuid.NewV7(), w.receiver}} {
					if got, found, err := read(ctx, miss.id, miss.receiver); err != nil || found || !reflect.DeepEqual(got, app.StoredNotification{}) {
						t.Fatalf("%s: %+v, %v, %v; want not found", miss.what, got, found, err)
					}
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

// The locked read locks the row alone FOR NO KEY UPDATE until its transaction ends, which the unlocked one does not;
// a row deleted while the lock waited for it is rechecked as the lock reads it, and not found (spec 2.7, step 4).
func TestLockNotification(t *testing.T) {
	s, pool := newStore(t)
	w := newReadWorld(t, pool)
	put(t, pool, w, readNotes(w)...)
	tx := postgres.NewTxManager(pool, time.Second)
	held := func(id uuid.UUID) bool { return pgtest.RowWaits(t, pool, "notifications", id, "FOR NO KEY UPDATE") }
	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		if _, found, err := s.GetNotification(ctx, purgeID(1), w.receiver); err != nil || !found || held(purgeID(1)) {
			t.Fatalf("the unlocked read: %v, %v, held %v; want it found and not held", found, err, held(purgeID(1)))
		}
		if _, found, err := s.LockNotification(ctx, purgeID(1), w.receiver); err != nil || !found {
			t.Fatalf("the locked read: %v, %v", found, err)
		}
		if !held(purgeID(1)) || held(purgeID(2)) {
			t.Fatal("the locked read does not hold its row alone FOR NO KEY UPDATE")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if held(purgeID(1)) {
		t.Fatal("the row is still held after the transaction")
	}
	lockWhileDeleted(t, s, pool, w)
}

// lockWhileDeleted locks note 2 while another transaction holds it to soft-delete it, which commits once the lock
// waits: the lock answers it not found.
func lockWhileDeleted(t *testing.T, s *postgresadapter.Store, pool *pgxpool.Pool, w readWorld) {
	t.Helper()
	deleter, err := pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = deleter.Rollback(context.Background()) }()
	if _, err := deleter.Exec(pgtest.Soon(t), "UPDATE notifications SET deleted_at = $2, updated_at = $2 WHERE id = $1", purgeID(2), now); err != nil {
		t.Fatal(err)
	}
	type answer struct {
		found bool
		err   error
	}
	done, soon := make(chan answer, 1), pgtest.Soon(t)
	go func() {
		var a answer
		a.err = postgres.NewTxManager(pool, time.Second).WithinTx(soon, func(ctx context.Context) error {
			var err error
			_, a.found, err = s.LockNotification(ctx, purgeID(2), w.receiver)
			return err
		})
		done <- a
	}()
	pgtest.WaitForLockWaitBehind(t, pool, deleter.Conn().PgConn(), 5*time.Second)
	if err := deleter.Commit(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
	if a := <-done; a.err != nil || a.found {
		t.Fatalf("locked while deleted: %v, %v; want not found", a.found, a.err)
	}
}

// A changed notification's three states are written as given, at the change's moment by its receiver, in the
// transaction ctx carries, and the write answers the row as it now stores it; the rest of the row stays (spec 2.7,
// step 7). Another receiver's and a deleted one are not written.
func TestUpdateNotification(t *testing.T) {
	s, pool := newStore(t)
	w := newReadWorld(t, pool)
	put(t, pool, w, readNotes(w)...)
	at, read, till := now.Add(time.Hour), now.Add(time.Minute), now.Add(2*time.Hour)
	want := stored17(w).Notification
	want.State = domain.State{ReadAt: &read, SnoozedTill: &till}
	err := postgres.NewTxManager(pool, time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		if _, found, err := s.LockNotification(ctx, purgeID(17), w.receiver); err != nil || !found {
			t.Fatalf("lock: %v, %v", found, err)
		}
		got, err := s.UpdateNotification(ctx, app.StateWrite{ID: purgeID(17), ReceiverID: w.receiver, State: want.State, At: at})
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("updated %+v, %v; want %+v", got, err, want)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := stamps(t, pool, purgeID(17))[0]; got != stamp(nil, at, &w.receiver) {
		t.Fatalf("stamped %s; want the change's moment and its receiver", got)
	}
	stored, found, err := s.GetNotification(context.Background(), purgeID(17), w.receiver)
	if wantStored := (app.StoredNotification{Notification: want, IssueCreatedByID: &w.alice}); err != nil || !found || !reflect.DeepEqual(stored, wantStored) {
		t.Fatalf("stored %+v, %v, %v; want %+v", stored, found, err, wantStored)
	}
	want.State = domain.State{ArchivedAt: &read}
	if got, err := s.UpdateNotification(context.Background(), app.StateWrite{ID: purgeID(17), ReceiverID: w.receiver, State: want.State, At: at}); err != nil ||
		!reflect.DeepEqual(got, want) {
		t.Fatalf("updated %+v, %v; want read and snoozed cleared, archived: %+v", got, err, want)
	}
	before := stamps(t, pool, purgeID(15), purgeID(1), purgeID(14))
	for _, bad := range []app.StateWrite{{ID: purgeID(15), ReceiverID: w.receiver, At: at}, {ID: purgeID(1), ReceiverID: w.carol, At: at},
		{ID: purgeID(14), ReceiverID: w.receiver, At: at}} {
		if _, err := s.UpdateNotification(context.Background(), bad); err == nil {
			t.Fatalf("wrote %+v", bad)
		}
	}
	if got := stamps(t, pool, purgeID(15), purgeID(1), purgeID(14)); !reflect.DeepEqual(got, before) {
		t.Fatalf("carol's, bob's as carol and the deleted one %v; want them as they were: %v", got, before)
	}
}
