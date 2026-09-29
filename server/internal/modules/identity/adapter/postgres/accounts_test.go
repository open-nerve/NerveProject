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

// The other order (M3 design 3.6 convention 6): while deactivation holds the
// row (its FOR NO KEY UPDATE taken, is_active set false, not committed), both
// reads wait for its lock, then read the account as deactivation committed
// it. WaitForLockWait sees the read waiting before the deactivation commits;
// every wait has a 10s deadline, so the test fails, not hangs. Each read has
// its own database, so the only lock wait there is its own.
func TestTheShareWaitsForDeactivationAndReadsItsResult(t *testing.T) {
	reads := map[string]func(ctx context.Context, s *postgresadapter.Store, id uuid.UUID) (app.AccountState, bool, error){
		"ShareAccount": func(ctx context.Context, s *postgresadapter.Store, id uuid.UUID) (app.AccountState, bool, error) {
			return s.ShareAccount(ctx, id)
		},
		"ShareAccountByEmail": func(ctx context.Context, s *postgresadapter.Store, _ uuid.UUID) (app.AccountState, bool, error) {
			return s.ShareAccountByEmail(ctx, "alice@corp.com")
		},
	}
	for name, read := range reads {
		t.Run(name, func(t *testing.T) {
			s, pool := newStore(t)
			u := newUser("alice@corp.com")
			mustCreate(t, s, u)
			tx := postgres.NewTxManager(pool, 2*time.Second)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			locked, release := make(chan struct{}), make(chan struct{})
			// A failure below must still end the deactivation: the pool's
			// Close in the cleanup waits for its connection.
			var releaseOnce sync.Once
			commit := func() { releaseOnce.Do(func() { close(release) }) }
			defer commit()
			deactivated := make(chan error, 1)
			go func() {
				deactivated <- tx.WithinTx(ctx, func(ctx context.Context) error {
					if _, err := s.LockForCredentials(ctx, u.ID); err != nil {
						return err
					}
					if err := s.DeactivateUser(ctx, u.ID, now); err != nil {
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
			case err := <-deactivated:
				t.Fatalf("deactivating: %v", err)
			case <-ctx.Done():
				t.Fatal("the deactivation did not hold the row within 10s")
			}

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
					a.state, a.found, err = read(ctx, s, u.ID)
					return err
				})
				answered <- a
			}()
			pgtest.WaitForLockWait(t, pool, 5*time.Second)
			commit()

			select {
			case err := <-deactivated:
				if err != nil {
					t.Fatalf("the deactivation: %v", err)
				}
			case <-ctx.Done():
				t.Fatal("the deactivation did not end within 10s")
			}
			select {
			case a := <-answered:
				want := app.AccountState{ID: u.ID, Email: "alice@corp.com", Active: false}
				if a.err != nil || !a.found || a.state != want {
					t.Errorf("%s() after the deactivation = %+v, found %v, %v; want %+v, found", name, a.state, a.found, a.err, want)
				}
			case <-ctx.Done():
				t.Fatal("the read did not end within 10s")
			}
		})
	}
}
