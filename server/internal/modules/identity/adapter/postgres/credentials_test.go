package postgresadapter_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func TestFindLoginAccount(t *testing.T) {
	s, _ := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)

	got, err := s.FindLoginAccount(context.Background(), "alice@corp.com")

	if want := (app.LoginAccount{ID: u.ID, PasswordHash: u.PasswordHash}); err != nil || got != want {
		t.Errorf("FindLoginAccount() = %+v, %v; want %+v", got, err, want)
	}
	// The address is matched as given: the use case normalizes it first.
	for _, email := range []string{"bob@corp.com", "Alice@corp.com", " alice@corp.com"} {
		if _, err := s.FindLoginAccount(context.Background(), email); !errors.Is(err, app.ErrNotFound) {
			t.Errorf("FindLoginAccount(%q) = %v, want app.ErrNotFound", email, err)
		}
	}
}

func TestLockForCredentialsReadsTheRow(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	tx := postgres.NewTxManager(pool, 2*time.Second)

	var got app.LockedAccount
	var unknown error
	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		var err error
		got, err = s.LockForCredentials(ctx, u.ID)
		_, unknown = s.LockForCredentials(ctx, uuid.NewV7())
		return err
	})

	if want := (app.LockedAccount{PasswordHash: u.PasswordHash, Active: true}); err != nil || got != want {
		t.Errorf("LockForCredentials() = %+v, %v; want %+v", got, err, want)
	}
	if !errors.Is(unknown, app.ErrNotFound) {
		t.Errorf("LockForCredentials(unknown) = %v, want app.ErrNotFound", unknown)
	}
}

// FOR NO KEY UPDATE (M2 design 3.5): while one transaction holds the lock,
// a second lock of the row waits, and inserting a session that references
// the account does not (interleaving 6). lock_timeout turns a wait into a
// failure.
func TestTheCredentialLockBlocksLocksNotInserts(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	locked, release := make(chan struct{}), make(chan struct{})
	held := make(chan error, 1)
	go func() {
		held <- tx.WithinTx(context.Background(), func(ctx context.Context) error {
			if _, err := s.LockForCredentials(ctx, u.ID); err != nil {
				return err
			}
			close(locked)
			<-release
			return nil
		})
	}()
	<-locked
	defer func() {
		close(release)
		if err := <-held; err != nil {
			t.Errorf("the transaction holding the lock: %v", err)
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
	hash := sha256.Sum256([]byte("secret"))

	insert := withTimeout(func(ctx context.Context) error {
		return s.CreateSession(ctx, app.NewSession{ID: uuid.NewV7(), UserID: u.ID, TokenHash: hash[:], ExpiresAt: now.Add(time.Hour), Now: now})
	})
	secondLock := withTimeout(func(ctx context.Context) error {
		_, err := s.LockForCredentials(ctx, u.ID)
		return err
	})

	if insert != nil {
		t.Errorf("inserting a session under the lock: %v, want no wait", insert)
	}
	var pgErr *pgconn.PgError
	if !errors.As(secondLock, &pgErr) || pgErr.Code != "55P03" {
		t.Errorf("a second lock: %v, want lock_not_available after waiting", secondLock)
	}
}

func TestUpdatePasswordHash(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	later := now.Add(time.Hour)

	if err := s.UpdatePasswordHash(context.Background(), u.ID, "$argon2id$new", later); err != nil {
		t.Fatal(err)
	}

	var password string
	var created, updated time.Time
	if err := pool.QueryRow(context.Background(), "SELECT password, created_at, updated_at FROM users WHERE id = $1", u.ID).
		Scan(&password, &created, &updated); err != nil {
		t.Fatal(err)
	}
	if password != "$argon2id$new" || !created.Equal(now) || !updated.Equal(later) {
		t.Errorf("row = %q, created %v, updated %v; want the new hash, updated at %v", password, created, updated, later)
	}
}
