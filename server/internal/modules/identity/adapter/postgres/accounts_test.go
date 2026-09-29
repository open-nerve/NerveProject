package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// ShareAccount and ShareAccountByEmail read the account's state: two
// accounts, one deactivated, and an unknown one of each kind (M3 design
// 6.5). The address is matched as given: the caller normalizes it.
func TestShareAccountReadsTheState(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	exec(t, pool, "UPDATE users SET is_active = false WHERE id = $1", bob.ID)
	tx := postgres.NewTxManager(pool, 2*time.Second)

	type answer struct {
		state app.AccountState
		found bool
	}
	var byID, byEmail []answer
	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		for _, id := range []uuid.UUID{alice.ID, bob.ID, uuid.NewV7()} {
			state, found, err := s.ShareAccount(ctx, id)
			if err != nil {
				return err
			}
			byID = append(byID, answer{state, found})
		}
		for _, email := range []string{"alice@corp.com", "bob@corp.com", "Alice@corp.com", "carol@corp.com"} {
			state, found, err := s.ShareAccountByEmail(ctx, email)
			if err != nil {
				return err
			}
			byEmail = append(byEmail, answer{state, found})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	aliceState := answer{app.AccountState{ID: alice.ID, Email: "alice@corp.com", Active: true}, true}
	bobState := answer{app.AccountState{ID: bob.ID, Email: "bob@corp.com", Active: false}, true}
	if want := []answer{aliceState, bobState, {}}; !slices.Equal(byID, want) {
		t.Errorf("ShareAccount() = %+v, want %+v", byID, want)
	}
	if want := []answer{aliceState, bobState, {}, {}}; !slices.Equal(byEmail, want) {
		t.Errorf("ShareAccountByEmail() = %+v, want %+v", byEmail, want)
	}
}

// FOR SHARE (M3 design 3.6 convention 6): while a transaction holds it,
// deactivation's lock of the row (FOR NO KEY UPDATE, M2's credential lock)
// waits, and a second FOR SHARE does not. lock_timeout turns a wait into a
// failure; the holding transaction ends within 10s, so the test fails, not
// hangs. Both reads take the lock.
func TestTheShareLockBlocksDeactivationNotAnotherShare(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	holders := map[string]func(ctx context.Context) error{
		"ShareAccount": func(ctx context.Context) error {
			_, _, err := s.ShareAccount(ctx, u.ID)
			return err
		},
		"ShareAccountByEmail": func(ctx context.Context) error {
			_, _, err := s.ShareAccountByEmail(ctx, "alice@corp.com")
			return err
		},
	}
	for name, hold := range holders {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			locked, release := make(chan struct{}), make(chan struct{})
			held := make(chan error, 1)
			go func() {
				held <- tx.WithinTx(ctx, func(ctx context.Context) error {
					if err := hold(ctx); err != nil {
						return err
					}
					close(locked)
					select {
					case <-release:
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				})
			}()
			select {
			case <-locked:
			case err := <-held:
				t.Fatalf("taking the lock: %v", err)
			case <-ctx.Done():
				t.Fatal("the lock was not taken within 10s")
			}
			defer func() {
				close(release)
				select {
				case err := <-held:
					if err != nil {
						t.Errorf("the transaction holding the lock: %v", err)
					}
				case <-ctx.Done():
					t.Error("the transaction holding the lock did not end within 10s")
				}
			}()
			withTimeout := func(fn func(ctx context.Context) error) error {
				return tx.WithinTx(context.Background(), func(ctx context.Context) error {
					if _, err := postgres.DB(ctx, pool).Exec(ctx, "SET LOCAL lock_timeout = '500ms'"); err != nil {
						return err
					}
					return fn(ctx)
				})
			}

			share := withTimeout(func(ctx context.Context) error {
				_, _, err := s.ShareAccount(ctx, u.ID)
				return err
			})
			deactivation := withTimeout(func(ctx context.Context) error {
				_, err := s.LockForCredentials(ctx, u.ID)
				return err
			})

			if share != nil {
				t.Errorf("a second FOR SHARE: %v, want no wait", share)
			}
			var pgErr *pgconn.PgError
			if !errors.As(deactivation, &pgErr) || pgErr.Code != "55P03" {
				t.Errorf("deactivation's lock: %v, want lock_not_available after waiting", deactivation)
			}
		})
	}
}

// share is one of the two reads of Accounts: by id, or by alice's address.
type share func(ctx context.Context, s *postgresadapter.Store, id uuid.UUID) (app.AccountState, bool, error)

var (
	shareByID share = func(ctx context.Context, s *postgresadapter.Store, id uuid.UUID) (app.AccountState, bool, error) {
		return s.ShareAccount(ctx, id)
	}
	shareByEmail share = func(ctx context.Context, s *postgresadapter.Store, _ uuid.UUID) (app.AccountState, bool, error) {
		return s.ShareAccountByEmail(ctx, "alice@corp.com")
	}
)

// hold runs lock in a transaction of its own and keeps the transaction open
// until end is called, which commits it and returns its error. The lock not
// taken within 10s fails the test. The test's cleanup calls end too, so a
// failure still ends the transaction: the pool's Close waits for its
// connection.
func hold(t *testing.T, tx *postgres.TxManager, lock func(ctx context.Context) error) (end func() error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	locked, release := make(chan struct{}), make(chan struct{})
	held := make(chan error, 1)
	go func() {
		held <- tx.WithinTx(ctx, func(ctx context.Context) error {
			if err := lock(ctx); err != nil {
				return err
			}
			close(locked)
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
	case <-locked:
	case err := <-held:
		once.Do(cancel) // the transaction has ended: nothing to wait for
		t.Fatalf("taking the lock: %v", err)
	case <-ctx.Done():
		t.Fatal("the lock was not taken within 10s")
	}
	return end
}

// The other order (M3 design 3.6 convention 6): while a transaction holds
// the row with M2's lock and has changed it, not committed, a share waits
// for the lock, then reads the account as that transaction committed it.
// Deactivation (is_active set false) holds it for both reads; `nerve users
// set-email` (a new address) for the read by id, so the address it answers
// is the one read under the lock too. WaitForLockWait sees the read waiting
// before the holder commits; every wait has a 10s deadline, so the test
// fails, not hangs. Each case has its own database, so the only lock wait
// there is its own.
func TestTheShareWaitsForTheRowsLockAndReadsWhatWasCommitted(t *testing.T) {
	deactivates := func(ctx context.Context, s *postgresadapter.Store, id uuid.UUID) error {
		if _, err := s.LockForCredentials(ctx, id); err != nil {
			return err
		}
		return s.DeactivateUser(ctx, id, now)
	}
	setsEmail := func(ctx context.Context, s *postgresadapter.Store, _ uuid.UUID) error {
		id, err := s.LockAccount(ctx, "alice@corp.com")
		if err != nil {
			return err
		}
		return s.ChangeEmail(ctx, id, "alice@corp.org", now)
	}
	deactivated := app.AccountState{Email: "alice@corp.com", Active: false}
	moved := app.AccountState{Email: "alice@corp.org", Active: true}
	tests := []struct {
		name string
		hold func(ctx context.Context, s *postgresadapter.Store, id uuid.UUID) error
		read share
		want app.AccountState // and alice's id
	}{
		{"deactivation holds, ShareAccount", deactivates, shareByID, deactivated},
		{"deactivation holds, ShareAccountByEmail", deactivates, shareByEmail, deactivated},
		{"set-email holds, ShareAccount", setsEmail, shareByID, moved},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, pool := newStore(t)
			u := newUser("alice@corp.com")
			mustCreate(t, s, u)
			tx := postgres.NewTxManager(pool, 2*time.Second)
			commit := hold(t, tx, func(ctx context.Context) error { return tt.hold(ctx, s, u.ID) })
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			type answer struct {
				state app.AccountState
				found bool
				err   error
			}
			answered := make(chan answer, 1)
			go func() {
				var a answer
				a.err = tx.WithinTx(ctx, func(ctx context.Context) error {
					var err error
					a.state, a.found, err = tt.read(ctx, s, u.ID)
					return err
				})
				answered <- a
			}()
			pgtest.WaitForLockWait(t, pool, 5*time.Second)
			if err := commit(); err != nil {
				t.Fatalf("the holder: %v", err)
			}

			select {
			case a := <-answered:
				want := tt.want
				want.ID = u.ID
				if a.err != nil || !a.found || a.state != want {
					t.Errorf("after the holder = %+v, found %v, %v; want %+v, found", a.state, a.found, a.err, want)
				}
			case <-ctx.Done():
				t.Fatal("the read did not end within 10s")
			}
		})
	}
}

// A share that fails answers its error, never "no such account", which
// would become 401 or workspace.account_not_found instead of 500: while
// deactivation's lock holds the row, a share under a lock_timeout gets
// lock_not_available (55P03), and both reads return it.
func TestAShareThatFailsReturnsTheError(t *testing.T) {
	for name, read := range map[string]share{"ShareAccount": shareByID, "ShareAccountByEmail": shareByEmail} {
		t.Run(name, func(t *testing.T) {
			s, pool := newStore(t)
			u := newUser("alice@corp.com")
			mustCreate(t, s, u)
			tx := postgres.NewTxManager(pool, 2*time.Second)
			hold(t, tx, func(ctx context.Context) error {
				_, err := s.LockForCredentials(ctx, u.ID)
				return err
			})

			var state app.AccountState
			var found bool
			var shareErr error // the share's own answer, not the commit's
			err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
				if _, err := postgres.DB(ctx, pool).Exec(ctx, "SET LOCAL lock_timeout = '100ms'"); err != nil {
					return err
				}
				state, found, shareErr = read(ctx, s, u.ID)
				return shareErr
			})

			var pgErr *pgconn.PgError
			if !errors.As(shareErr, &pgErr) || pgErr.Code != "55P03" || found || state != (app.AccountState{}) {
				t.Errorf("%s() while the row is locked = %+v, found %v, %v (the transaction: %v); want lock_not_available and nothing found",
					name, state, found, shareErr, err)
			}
		})
	}
}
