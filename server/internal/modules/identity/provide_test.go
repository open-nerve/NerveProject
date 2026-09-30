package identity_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// rowLockedByAnother reports whether alice's account row is locked against
// FOR NO KEY UPDATE by another transaction: NOWAIT answers lock_not_available
// (55P03) at once instead of waiting.
func (a *account) rowLockedByAnother(t *testing.T) bool {
	t.Helper()
	_, err := a.pool.Exec(context.Background(), "SELECT 1 FROM users WHERE id = $1 FOR NO KEY UPDATE NOWAIT", a.id)
	var pgErr *pgconn.PgError
	if err != nil && (!errors.As(err, &pgErr) || pgErr.Code != "55P03") {
		t.Fatal(err)
	}
	return err != nil
}

// Provide's CredentialLock is M2's credential lock (M3 design 6.5, 6.6): a
// valid session or personal access token takes it, and the account row stays
// locked until the transaction ends; a revoked session, a revoked token, a
// deactivated account and an unknown one are 401 unauthorized.
func TestProvidedCredentialLockIsTheCredentialLock(t *testing.T) {
	a := newAccount(t, "hashed:Tr0ub4dor&3:s")
	lock := identity.Provide(a.pool).CredentialLock
	ctx, now := context.Background(), time.Now()
	session := shared.Actor{UserID: a.id, SessionID: a.session}
	pat := shared.Actor{UserID: a.id, APITokenID: uuid.NewV7()}
	if err := a.store.CreateAPIToken(ctx, app.NewAPIToken{ID: pat.APITokenID, UserID: a.id, TokenHash: make([]byte, 32), Label: "ci", Now: now}); err != nil {
		t.Fatal(err)
	}

	for name, actor := range map[string]shared.Actor{"a session": session, "a personal access token": pat} {
		var held bool
		err := a.tx.WithinTx(ctx, func(ctx context.Context) error {
			if err := lock.LockCaller(ctx, actor, now); err != nil {
				return err
			}
			held = a.rowLockedByAnother(t)
			return nil
		})
		if err != nil || !held || a.rowLockedByAnother(t) {
			t.Errorf("%s: LockCaller() = %v, row held %v; want the row locked until the transaction ended", name, err, held)
		}
	}

	if _, err := a.store.RevokeAPIToken(ctx, pat.APITokenID, a.id, now); err != nil {
		t.Fatal(err)
	}
	if err := lock.LockCaller(ctx, pat, now); !errors.Is(err, shared.Unauthenticated()) {
		t.Errorf("a revoked token: LockCaller() = %v, want 401 unauthorized", err)
	}
	if _, err := a.store.RevokeSessions(ctx, a.id, uuid.Nil(), domain.RevokePasswordReset, now); err != nil {
		t.Fatal(err)
	}
	if err := lock.LockCaller(ctx, session, now); !errors.Is(err, shared.Unauthenticated()) {
		t.Errorf("a revoked session: LockCaller() = %v, want 401 unauthorized", err)
	}
	live := shared.Actor{UserID: a.id, SessionID: uuid.NewV7()}
	if err := a.store.CreateSession(ctx, app.NewSession{ID: live.SessionID, UserID: a.id, TokenHash: make([]byte, 32),
		ExpiresAt: now.Add(time.Hour), Now: now}); err != nil {
		t.Fatal(err)
	}
	if err := a.store.DeactivateUser(ctx, a.id, now); err != nil {
		t.Fatal(err)
	}
	if err := lock.LockCaller(ctx, live, now); !errors.Is(err, shared.Unauthenticated()) {
		t.Errorf("a deactivated account: LockCaller() = %v, want 401 unauthorized", err)
	}
	if err := lock.LockCaller(ctx, shared.Actor{UserID: uuid.NewV7(), SessionID: live.SessionID}, now); !errors.Is(err, shared.Unauthenticated()) {
		t.Errorf("an unknown account: LockCaller() = %v, want 401 unauthorized", err)
	}
}
