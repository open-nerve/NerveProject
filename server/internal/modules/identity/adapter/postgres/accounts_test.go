package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
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
