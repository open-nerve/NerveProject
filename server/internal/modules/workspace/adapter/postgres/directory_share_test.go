package postgresadapter_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The directory's lock by id, the first lock of every write on a project
// (M3 design 3.6 convention 2), finds the undeleted workspace with the id,
// with its id and its time zone: each of two workspaces its own, so a lock
// that read the first row whatever its id would answer one for the other;
// not a deleted one, not an id no workspace has. A failing call answers its
// error, not "no such workspace", which the write would turn into a 404.
func TestShareWorkspaceByIDFindsTheUndeletedWorkspace(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	beta := newWorkspace(t, s, "Beta", "beta", alice)
	gone := newWorkspace(t, s, "Gone", "gone", alice)
	exec(t, pool, "UPDATE workspaces SET timezone = 'Asia/Shanghai' WHERE id = $1", beta.ID)
	exec(t, pool, "UPDATE workspaces SET deleted_at = $2 WHERE id = $1", gone.ID, now)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		for id, want := range map[uuid.UUID]app.DirectoryEntry{acme.ID: {ID: acme.ID, Timezone: "UTC"}, beta.ID: {ID: beta.ID, Timezone: "Asia/Shanghai"}} {
			if got, found, err := d.ShareWorkspaceByID(ctx, id); err != nil || !found || got != want {
				t.Errorf("ShareWorkspaceByID(%s) = %+v, %v, %v; want %+v", id, got, found, err, want)
			}
		}
		for _, id := range []uuid.UUID{gone.ID, uuid.NewV7()} {
			if got, found, err := d.ShareWorkspaceByID(ctx, id); err != nil || found || got != (app.DirectoryEntry{}) {
				t.Errorf("ShareWorkspaceByID(%s) = %+v, %v, %v; want not found", id, got, found, err)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, found, err := d.ShareWorkspaceByID(cancelled, acme.ID); !errors.Is(err, context.Canceled) || found || got != (app.DirectoryEntry{}) {
		t.Errorf("ShareWorkspaceByID() = %+v, %v, %v; want context.Canceled, not no workspace", got, found, err)
	}
}

// The lock by id is convention 2's FOR SHARE: it waits for the workspace's
// FOR NO KEY UPDATE, which every cascade over the workspace's projects runs
// under, and makes it wait; it does not wait for another FOR SHARE, which
// another write on a project of the workspace holds; and it locks no other
// workspace. A lock that waits ends with lock_not_available under a
// lock_timeout.
func TestTheDirectorysLockByIDIsForShare(t *testing.T) {
	const directory = "Directory.ShareWorkspaceByID"
	tests := []struct {
		held, then string
		slug       string // then's
		waits      bool
	}{
		{directory, noKeyUpdateByID.name, "acme", true},
		{noKeyUpdateByID.name, directory, "acme", true},
		{directory, forShareByID.name, "acme", false},
		{forShareByID.name, directory, "acme", false},
		{directory, noKeyUpdateByID.name, "beta", false},
		{noKeyUpdateByID.name, directory, "beta", false},
	}
	for _, tt := range tests {
		t.Run(tt.held+" held, "+tt.then+" of "+tt.slug, func(t *testing.T) {
			s, pool := newStore(t)
			d := postgresadapter.NewDirectory(pool)
			locks := map[string]lock{noKeyUpdateByID.name: noKeyUpdateByID, forShareByID.name: forShareByID, directory: {directory,
				func(ctx context.Context, _ *postgresadapter.Store, w named) (uuid.UUID, error) {
					e, found, err := d.ShareWorkspaceByID(ctx, w.id)
					if err == nil && !found {
						err = app.ErrNotFound
					}
					return e.ID, err
				}}}
			alice := newAccount(t, pool, "alice@corp.com")
			ids := map[string]uuid.UUID{"acme": newWorkspace(t, s, "Acme", "acme", alice).ID, "beta": newWorkspace(t, s, "Beta", "beta", alice).ID}
			tx := postgres.NewTxManager(pool, 2*time.Second)
			hold(t, tx, func(ctx context.Context) error {
				_, err := locks[tt.held].take(ctx, s, named{"acme", ids["acme"]})
				return err
			})

			var got uuid.UUID
			err := withLockTimeout(tx, pool, func(ctx context.Context) error {
				var err error
				got, err = locks[tt.then].take(ctx, s, named{tt.slug, ids[tt.slug]})
				return err
			})

			var pgErr *pgconn.PgError
			switch {
			case tt.waits && (!errors.As(err, &pgErr) || pgErr.Code != "55P03"):
				t.Errorf("%s() = %s, %v; want lock_not_available after waiting", tt.then, got, err)
			case !tt.waits && (err != nil || got != ids[tt.slug]):
				t.Errorf("%s() = %s, %v; want %s's id %s without waiting", tt.then, got, err, tt.slug, ids[tt.slug])
			}
		})
	}
}

// A workspace deleted while the lock by id waits is not found: the
// statement has deleted_at IS NULL, which Postgres evaluates again on the
// row's newest version after the wait.
func TestTheDirectorysLockByIDSeesADeletionItWaitedFor(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	end := hold(t, tx, func(ctx context.Context) error {
		if err := s.LockWorkspace(ctx, acme.ID); err != nil {
			return err
		}
		return s.DeleteWorkspace(ctx, acme.ID, alice, now)
	})
	type answer struct {
		w     app.DirectoryEntry
		found bool
		err   error
	}
	done := make(chan answer, 1)
	go func() {
		var a answer
		a.err = tx.WithinTx(context.Background(), func(ctx context.Context) error {
			var err error
			a.w, a.found, err = d.ShareWorkspaceByID(ctx, acme.ID)
			return err
		})
		done <- a
	}()
	pgtest.WaitForLockWaitOn(t, pool, "workspaces", 10*time.Second)
	if err := end(); err != nil {
		t.Fatal(err)
	}
	select {
	case a := <-done:
		if a.err != nil || a.found {
			t.Errorf("ShareWorkspaceByID() after the deletion = %+v, %v, %v; want not found", a.w, a.found, a.err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("ShareWorkspaceByID() did not end within 10s")
	}
}
