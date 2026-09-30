package postgresadapter_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// lock is one of the store's parent locks of a workspace (M3 design 3.6
// convention 2), which names it by its slug or by its id: take answers the
// workspace's id, the one the lock returned or, for the id locks, which
// return none, the one they were given.
type lock struct {
	name string
	take func(ctx context.Context, s *postgresadapter.Store, w named) (uuid.UUID, error)
}

// named is how a lock names a workspace.
type named struct {
	slug string
	id   uuid.UUID
}

var (
	noKeyUpdate = lock{"LockWorkspaceBySlug", func(ctx context.Context, s *postgresadapter.Store, w named) (uuid.UUID, error) {
		return s.LockWorkspaceBySlug(ctx, w.slug)
	}}
	forShare = lock{"ShareWorkspaceBySlug", func(ctx context.Context, s *postgresadapter.Store, w named) (uuid.UUID, error) {
		return s.ShareWorkspaceBySlug(ctx, w.slug)
	}}
	noKeyUpdateByID = lock{"LockWorkspace", func(ctx context.Context, s *postgresadapter.Store, w named) (uuid.UUID, error) {
		if err := s.LockWorkspace(ctx, w.id); err != nil {
			return uuid.UUID{}, err
		}
		return w.id, nil
	}}
	forShareByID = lock{"ShareWorkspace", func(ctx context.Context, s *postgresadapter.Store, w named) (uuid.UUID, error) {
		if err := s.ShareWorkspace(ctx, w.id); err != nil {
			return uuid.UUID{}, err
		}
		return w.id, nil
	}}
	locks = []lock{noKeyUpdate, forShare, noKeyUpdateByID, forShareByID}
)

// hold runs fn in a transaction of its own and keeps it open until end is
// called, which commits it and returns its error. fn not done within 10s
// fails the test. The test's cleanup calls end too, so a failure still ends
// the transaction: the pool's Close waits for its connection.
func hold(t *testing.T, tx *postgres.TxManager, fn func(ctx context.Context) error) (end func() error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	done, release := make(chan struct{}), make(chan struct{})
	held := make(chan error, 1)
	go func() {
		held <- tx.WithinTx(ctx, func(ctx context.Context) error {
			if err := fn(ctx); err != nil {
				return err
			}
			close(done)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	var once sync.Once
	var ended error
	end = func() error {
		once.Do(func() {
			close(release)
			select {
			case ended = <-held:
			case <-ctx.Done():
				ended = errors.New("the transaction holding the row did not end within 10s")
			}
			cancel()
		})
		return ended
	}
	t.Cleanup(func() { _ = end() })
	select {
	case <-done:
	case err := <-held:
		once.Do(cancel) // the transaction has ended: nothing to wait for
		t.Fatalf("taking the lock: %v", err)
	case <-ctx.Done():
		t.Fatal("the lock was not taken within 10s")
	}
	return end
}

// withLockTimeout runs fn in a transaction whose lock waits end after
// 300ms with lock_not_available (55P03).
func withLockTimeout(tx *postgres.TxManager, pool *pgxpool.Pool, fn func(ctx context.Context) error) error {
	return tx.WithinTx(context.Background(), func(ctx context.Context) error {
		if _, err := postgres.DB(ctx, pool).Exec(ctx, "SET LOCAL lock_timeout = '300ms'"); err != nil {
			return err
		}
		return fn(ctx)
	})
}

// The locks conflict as convention 2 wants: FOR NO KEY UPDATE, by slug or
// by id, waits for any of them, FOR SHARE waits for FOR NO KEY UPDATE and
// not for another FOR SHARE, and none waits for a lock of another
// workspace. A lock that waits ends with lock_not_available under a
// lock_timeout; one that does not answers its workspace's id.
func TestTheWorkspaceLocksConflictAsConvention2Says(t *testing.T) {
	tests := []struct {
		held, then lock
		slug       string // then's
		waits      bool
	}{
		{noKeyUpdate, noKeyUpdate, "acme", true},
		{noKeyUpdate, forShare, "acme", true},
		{forShare, noKeyUpdate, "acme", true},
		{forShare, forShare, "acme", false},
		{noKeyUpdate, noKeyUpdateByID, "acme", true},
		{noKeyUpdateByID, noKeyUpdate, "acme", true},
		{forShare, noKeyUpdateByID, "acme", true},
		{noKeyUpdateByID, forShare, "acme", true},
		{noKeyUpdate, forShareByID, "acme", true},
		{forShareByID, noKeyUpdate, "acme", true},
		{forShareByID, noKeyUpdateByID, "acme", true},
		{noKeyUpdateByID, forShareByID, "acme", true},
		{forShare, forShareByID, "acme", false},
		{forShareByID, forShare, "acme", false},
		{noKeyUpdate, noKeyUpdate, "beta", false},
		{noKeyUpdate, forShare, "beta", false},
		{noKeyUpdate, noKeyUpdateByID, "beta", false},
		{noKeyUpdate, forShareByID, "beta", false},
	}
	for _, tt := range tests {
		t.Run(tt.held.name+" held, "+tt.then.name+" of "+tt.slug, func(t *testing.T) {
			s, pool := newStore(t)
			alice := newAccount(t, pool, "alice@corp.com")
			ids := map[string]uuid.UUID{"acme": newWorkspace(t, s, "Acme", "acme", alice).ID, "beta": newWorkspace(t, s, "Beta", "beta", alice).ID}
			tx := postgres.NewTxManager(pool, 2*time.Second)
			hold(t, tx, func(ctx context.Context) error {
				_, err := tt.held.take(ctx, s, named{"acme", ids["acme"]})
				return err
			})

			var got uuid.UUID
			err := withLockTimeout(tx, pool, func(ctx context.Context) error {
				var err error
				got, err = tt.then.take(ctx, s, named{tt.slug, ids[tt.slug]})
				return err
			})

			var pgErr *pgconn.PgError
			switch {
			case tt.waits && (!errors.As(err, &pgErr) || pgErr.Code != "55P03"):
				t.Errorf("%s() = %s, %v; want lock_not_available after waiting", tt.then.name, got, err)
			case !tt.waits && (err != nil || got != ids[tt.slug]):
				t.Errorf("%s() = %s, %v; want %s's id %s without waiting", tt.then.name, got, err, tt.slug, ids[tt.slug])
			}
		})
	}
}

// A foreign key's check takes the workspace row FOR KEY SHARE when a row
// under the workspace is inserted: none of the locks holds it off, as a FOR
// UPDATE, stronger than convention 2 asks, would. The statement locks acme's
// row, one row, so it cannot pass by matching none.
func TestAForeignKeyCheckDoesNotWaitForTheWorkspaceLocks(t *testing.T) {
	for _, l := range locks {
		t.Run(l.name, func(t *testing.T) {
			s, pool := newStore(t)
			alice := newAccount(t, pool, "alice@corp.com")
			acme := newWorkspace(t, s, "Acme", "acme", alice)
			tx := postgres.NewTxManager(pool, 2*time.Second)
			hold(t, tx, func(ctx context.Context) error {
				_, err := l.take(ctx, s, named{"acme", acme.ID})
				return err
			})
			var locked int64
			err := withLockTimeout(tx, pool, func(ctx context.Context) error {
				tag, err := postgres.DB(ctx, pool).Exec(ctx, "SELECT id FROM workspaces WHERE id = $1 FOR KEY SHARE", acme.ID)
				locked = tag.RowsAffected()
				return err
			})
			if err != nil || locked != 1 {
				t.Errorf("FOR KEY SHARE of acme while %s holds it: %d rows, %v; want acme's row without a wait", l.name, locked, err)
			}
		})
	}
}

// A lock finds the undeleted workspace it names, whose id take answers;
// app.ErrNotFound for a deleted workspace, a slug or an id no workspace has,
// or a slug of another case.
func TestTheWorkspaceLocksFindOnlyAnUndeletedWorkspace(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	gone := newWorkspace(t, s, "Gone", "gone", alice)
	exec(t, pool, "UPDATE workspaces SET deleted_at = $2 WHERE id = $1", gone.ID, now)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	for _, l := range locks {
		for _, tt := range []struct {
			w    named
			want uuid.UUID
		}{
			{named{"acme", acme.ID}, acme.ID}, {named{"beta", beta.ID}, beta.ID}, {named{"gone", gone.ID}, uuid.UUID{}},
			{named{"nothing", uuid.NewV7()}, uuid.UUID{}}, {named{"ACME", uuid.NewV7()}, uuid.UUID{}},
		} {
			var got uuid.UUID
			err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
				var err error
				got, err = l.take(ctx, s, tt.w)
				return err
			})
			if tt.want == (uuid.UUID{}) && (!errors.Is(err, app.ErrNotFound) || got != tt.want) {
				t.Errorf("%s(%+v) = %s, %v; want app.ErrNotFound", l.name, tt.w, got, err)
			}
			if tt.want != (uuid.UUID{}) && (err != nil || got != tt.want) {
				t.Errorf("%s(%+v) = %s, %v; want %s", l.name, tt.w, got, err, tt.want)
			}
		}
	}
}

// A lock that waits for the transaction deleting the workspace reads no row
// once that one commits (convention 2): Postgres evaluates deleted_at IS
// NULL again on the committed version. WaitForLockWaitOn sees the lock
// waiting on the workspace row before the deletion commits.
func TestTheWorkspaceLocksSkipAWorkspaceDeletedWhileTheyWait(t *testing.T) {
	for _, l := range locks {
		t.Run(l.name, func(t *testing.T) {
			s, pool := newStore(t)
			alice := newAccount(t, pool, "alice@corp.com")
			acme := newWorkspace(t, s, "Acme", "acme", alice)
			tx := postgres.NewTxManager(pool, 2*time.Second)
			commit := hold(t, tx, func(ctx context.Context) error {
				if _, err := s.LockWorkspaceBySlug(ctx, "acme"); err != nil {
					return err
				}
				_, err := postgres.DB(ctx, pool).Exec(ctx, "UPDATE workspaces SET deleted_at = $2 WHERE id = $1", acme.ID, now)
				return err
			})
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			type answer struct {
				id  uuid.UUID
				err error
			}
			answered := make(chan answer, 1)
			go func() {
				var a answer
				a.err = tx.WithinTx(ctx, func(ctx context.Context) error {
					var err error
					a.id, err = l.take(ctx, s, named{"acme", acme.ID})
					return err
				})
				answered <- a
			}()
			pgtest.WaitForLockWaitOn(t, pool, "workspaces", 5*time.Second)
			if err := commit(); err != nil {
				t.Fatalf("the deletion: %v", err)
			}
			select {
			case a := <-answered:
				if !errors.Is(a.err, app.ErrNotFound) || a.id != (uuid.UUID{}) {
					t.Errorf("%s() after the deletion = %s, %v; want app.ErrNotFound", l.name, a.id, a.err)
				}
			case <-ctx.Done():
				t.Fatal("the lock did not end within 10s")
			}
		})
	}
}

// A lock that fails answers its error, never app.ErrNotFound, which the use
// case would turn into workspace.not_found: a cancelled context.
func TestAFailedWorkspaceLockIsAnErrorNotAnAnswer(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, l := range locks {
		if id, err := l.take(cancelled, s, named{"acme", acme.ID}); !errors.Is(err, context.Canceled) || errors.Is(err, app.ErrNotFound) || id != (uuid.UUID{}) {
			t.Errorf("%s() = %s, %v; want context.Canceled, not app.ErrNotFound", l.name, id, err)
		}
	}
}
