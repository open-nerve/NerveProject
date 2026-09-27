package postgresadapter_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// holdLock runs lock in a transaction and keeps the transaction open until
// the test ends. Every wait, the holding transaction's too, ends within 10s:
// the test fails, not hangs.
func holdLock(t *testing.T, tx *postgres.TxManager, lock func(ctx context.Context) error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	locked, release, held := make(chan struct{}), make(chan struct{}), make(chan error, 1)
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
	select {
	case <-locked:
	case err := <-held:
		t.Fatalf("taking the lock: %v", err)
	case <-ctx.Done():
		t.Fatal("the lock was not taken within 10s")
	}
	t.Cleanup(func() {
		close(release)
		select {
		case err := <-held:
			if err != nil {
				t.Errorf("the transaction holding the lock: %v", err)
			}
		case <-ctx.Done():
			t.Error("the transaction holding the lock did not end within 10s")
		}
	})
}

// withLockTimeout runs fn in a transaction whose lock waits fail after
// 500ms.
func withLockTimeout(tx *postgres.TxManager, pool *pgxpool.Pool, fn func(ctx context.Context) error) error {
	return tx.WithinTx(context.Background(), func(ctx context.Context) error {
		if _, err := postgres.DB(ctx, pool).Exec(ctx, "SET LOCAL lock_timeout = '500ms'"); err != nil {
			return err
		}
		return fn(ctx)
	})
}

func isLockTimeout(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "55P03"
}

func TestLockAccountFindsTheAccountByAddress(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	tx := postgres.NewTxManager(pool, 2*time.Second)

	var got uuid.UUID
	var unknown error
	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		var err error
		got, err = s.LockAccount(ctx, "bob@corp.com")
		_, unknown = s.LockAccount(ctx, "carol@corp.com")
		return err
	})

	if err != nil || got != bob.ID || !errors.Is(unknown, app.ErrNotFound) {
		t.Errorf("LockAccount() = %v, %v, unknown address %v; want bob's id and app.ErrNotFound", got, err, unknown)
	}
}

// LockAccount takes the lock of M2 design 3.5 by address: while it is held,
// the credential lock of the same account waits, another account's does
// not, and inserting a session of the account does not either (FOR NO KEY
// UPDATE, not FOR UPDATE).
func TestLockAccountIsTheAccountRowLock(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	holdLock(t, tx, func(ctx context.Context) error {
		_, err := s.LockAccount(ctx, "alice@corp.com")
		return err
	})
	hash := sha256.Sum256([]byte("secret"))

	same := withLockTimeout(tx, pool, func(ctx context.Context) error {
		_, err := s.LockForCredentials(ctx, alice.ID)
		return err
	})
	byAddress := withLockTimeout(tx, pool, func(ctx context.Context) error {
		_, err := s.LockAccount(ctx, "alice@corp.com")
		return err
	})
	other := withLockTimeout(tx, pool, func(ctx context.Context) error {
		_, err := s.LockForCredentials(ctx, bob.ID)
		return err
	})
	insert := withLockTimeout(tx, pool, func(ctx context.Context) error {
		return s.CreateSession(ctx, app.NewSession{ID: uuid.NewV7(), UserID: alice.ID, TokenHash: hash[:], ExpiresAt: now.Add(time.Hour), Now: now})
	})

	if !isLockTimeout(same) || !isLockTimeout(byAddress) {
		t.Errorf("locking the held account by id: %v, by address: %v; want both to wait", same, byAddress)
	}
	if other != nil || insert != nil {
		t.Errorf("locking another account: %v; inserting a session of the held one: %v; want neither to wait", other, insert)
	}
}

func readUser(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) (email string, active bool, updated time.Time) {
	t.Helper()
	if err := pool.QueryRow(context.Background(), "SELECT email, is_active, updated_at FROM users WHERE id = $1", id).
		Scan(&email, &active, &updated); err != nil {
		t.Fatal(err)
	}
	return email, active, updated
}

func TestChangeEmail(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)

	changed := s.ChangeEmail(context.Background(), alice.ID, "alice@new.example", later)
	taken := s.ChangeEmail(context.Background(), alice.ID, "bob@corp.com", later.Add(time.Minute))

	if changed != nil || !errors.Is(taken, domain.ErrEmailTaken) {
		t.Errorf("ChangeEmail() = %v, to bob's address %v; want nil, then identity.email_taken", changed, taken)
	}
	if email, _, updated := readUser(t, pool, alice.ID); email != "alice@new.example" || !updated.Equal(later) {
		t.Errorf("alice = %s updated %v, want alice@new.example at %v", email, updated, later)
	}
	if email, _, updated := readUser(t, pool, bob.ID); email != "bob@corp.com" || !updated.Equal(now) {
		t.Errorf("bob = %s updated %v, want unchanged", email, updated)
	}
}

func TestActivateUser(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	for _, u := range []app.NewUser{alice, bob} {
		if err := s.DeactivateUser(context.Background(), u.ID, now); err != nil {
			t.Fatal(err)
		}
	}

	if err := s.ActivateUser(context.Background(), alice.ID, later); err != nil {
		t.Fatal(err)
	}

	if _, active, updated := readUser(t, pool, alice.ID); !active || !updated.Equal(later) {
		t.Errorf("alice active %v updated %v, want active at %v", active, updated, later)
	}
	if _, active, _ := readUser(t, pool, bob.ID); active {
		t.Error("bob is active, want him left inactive")
	}
}

// tokenRow is what the administrator's revocation writes.
type tokenRow struct {
	deleted   *time.Time
	updated   time.Time
	updatedBy *uuid.UUID
}

func readToken(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) tokenRow {
	t.Helper()
	var r tokenRow
	if err := pool.QueryRow(context.Background(), "SELECT deleted_at, updated_at, updated_by_id FROM api_tokens WHERE id = $1", id).
		Scan(&r.deleted, &r.updated, &r.updatedBy); err != nil {
		t.Fatal(err)
	}
	return r
}

// Every token of the account that is not revoked yet, expired or not, is
// revoked, by no account: updated_by_id NULL. A token revoked before keeps
// its revocation; another account's tokens stay.
func TestRevokeAllAPITokens(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	live, expired, revoked, bobs := newToken(alice.ID, "live", now), newToken(alice.ID, "expired", now),
		newToken(alice.ID, "revoked", now), newToken(bob.ID, "bobs", now)
	for _, n := range []app.NewAPIToken{live, expired, revoked, bobs} {
		mustCreateToken(t, s, n)
	}
	exec(t, pool, "UPDATE api_tokens SET expired_at = $2 WHERE id = $1", expired.ID, now.Add(-time.Minute))
	if ok, err := s.RevokeAPIToken(context.Background(), revoked.ID, alice.ID, now); err != nil || !ok {
		t.Fatalf("RevokeAPIToken() = %v, %v", ok, err)
	}

	n, err := s.RevokeAllAPITokens(context.Background(), alice.ID, later)

	if err != nil || n != 2 {
		t.Fatalf("RevokeAllAPITokens() = %d, %v; want 2", n, err)
	}
	for _, id := range []uuid.UUID{live.ID, expired.ID} {
		if r := readToken(t, pool, id); r.deleted == nil || !r.deleted.Equal(later) || !r.updated.Equal(later) || r.updatedBy != nil {
			t.Errorf("token %v = deleted %v updated %v by %v; want revoked at %v by nobody", id, r.deleted, r.updated, r.updatedBy, later)
		}
	}
	if r := readToken(t, pool, revoked.ID); r.deleted == nil || !r.deleted.Equal(now) || r.updatedBy == nil || *r.updatedBy != alice.ID {
		t.Errorf("the token revoked before = deleted %v by %v, want its own revocation kept", r.deleted, r.updatedBy)
	}
	if r := readToken(t, pool, bobs.ID); r.deleted != nil || !r.updated.Equal(now) {
		t.Errorf("bob's token = deleted %v updated %v, want untouched", r.deleted, r.updated)
	}
}

// Usable means unrevoked and unexpired at now, as authentication judges: a
// token that expires at now is not.
func TestCountUsableAPITokens(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	forever, expiring, expiresNow, revoked, bobs := newToken(alice.ID, "forever", now), newToken(alice.ID, "expiring", now),
		newToken(alice.ID, "now", now), newToken(alice.ID, "revoked", now), newToken(bob.ID, "bobs", now)
	for _, n := range []app.NewAPIToken{forever, expiring, expiresNow, revoked, bobs} {
		mustCreateToken(t, s, n)
	}
	exec(t, pool, "UPDATE api_tokens SET expired_at = $2 WHERE id = $1", expiring.ID, later)
	exec(t, pool, "UPDATE api_tokens SET expired_at = $2 WHERE id = $1", expiresNow.ID, now)
	exec(t, pool, "UPDATE api_tokens SET deleted_at = $2 WHERE id = $1", revoked.ID, now)

	n, err := s.CountUsableAPITokens(context.Background(), alice.ID, now)

	if err != nil || n != 2 {
		t.Errorf("CountUsableAPITokens() = %d, %v; want 2: the one that never expires and the one that expires later", n, err)
	}
}
