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
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
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

func TestPasswordAccount(t *testing.T) {
	s, _ := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)

	got, err := s.PasswordAccount(context.Background(), u.ID)

	if want := (app.PasswordAccount{Email: "alice@corp.com", PasswordHash: u.PasswordHash}); err != nil || got != want {
		t.Errorf("PasswordAccount() = %+v, %v; want %+v", got, err, want)
	}
	if _, err := s.PasswordAccount(context.Background(), uuid.NewV7()); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("PasswordAccount(unknown) = %v, want app.ErrNotFound", err)
	}
}

// RevokeSessions revokes the account's live sessions but the one kept, and
// only them: a session revoked or expired already keeps what it has, and
// another account's is untouched (M2 design 3.5).
func TestRevokeSessions(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	session := func(u app.NewUser, expires time.Time) uuid.UUID {
		t.Helper()
		n := app.NewSession{ID: uuid.NewV7(), UserID: u.ID, TokenHash: secretHash[:], ExpiresAt: expires, Now: now}
		if err := s.CreateSession(context.Background(), n); err != nil {
			t.Fatal(err)
		}
		return n.ID
	}
	kept, live, expired, loggedOut, bobs := session(alice, sessionEnd), session(alice, sessionEnd),
		session(alice, now.Add(time.Second)), session(alice, sessionEnd), session(bob, sessionEnd)
	exec(t, pool, `UPDATE auth_sessions SET revoked_at = $2, revoke_reason = 'logout' WHERE id = $1`, loggedOut, now)

	n, err := s.RevokeSessions(context.Background(), alice.ID, kept, domain.RevokePasswordChanged, later)

	if err != nil || n != 1 {
		t.Fatalf("RevokeSessions() = %d, %v; want 1", n, err)
	}
	tests := []struct {
		name            string
		id              uuid.UUID
		reason          string // "" for none
		revoked, update time.Time
	}{
		{"the kept session", kept, "", time.Time{}, now},
		{"a live session", live, "password_changed", later, later},
		{"an expired session", expired, "", time.Time{}, now},
		{"a session logged out", loggedOut, "logout", now, now},
		{"another account's session", bobs, "", time.Time{}, now},
	}
	for _, tt := range tests {
		r := readSession(t, pool, tt.id)
		var reason string
		var revoked time.Time
		if r.reason != nil {
			reason, revoked = *r.reason, *r.revoked
		}
		if reason != tt.reason || !revoked.Equal(tt.revoked) || !r.updated.Equal(tt.update) {
			t.Errorf("%s: reason %q revoked %v updated %v; want %q, %v, %v", tt.name, reason, revoked, r.updated, tt.reason, tt.revoked, tt.update)
		}
	}
	// uuid.Nil keeps none.
	if n, err := s.RevokeSessions(context.Background(), alice.ID, uuid.Nil(), domain.RevokePasswordChanged, later); err != nil || n != 1 {
		t.Errorf("RevokeSessions(keep none) = %d, %v; want the kept one revoked", n, err)
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
	select {
	case <-locked:
	case err := <-held:
		t.Fatalf("taking the lock: %v", err)
	}
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
	rehashed := now.Add(time.Hour)

	if err := s.UpdatePasswordHash(context.Background(), u.ID, "$argon2id$new", rehashed); err != nil {
		t.Fatal(err)
	}

	var password string
	var created, updated time.Time
	if err := pool.QueryRow(context.Background(), "SELECT password, created_at, updated_at FROM users WHERE id = $1", u.ID).
		Scan(&password, &created, &updated); err != nil {
		t.Fatal(err)
	}
	if password != "$argon2id$new" || !created.Equal(now) || !updated.Equal(rehashed) {
		t.Errorf("row = %q, created %v, updated %v; want the new hash, updated at %v", password, created, updated, rehashed)
	}
}
