package identity_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Interleavings 1-3 of M2 design 3.5: a login, a login that hashes the
// password again, and a token creation, each against the administrator's
// reset of the password. Each runs both ways round on a real database: the
// operation that goes first stops at a gate inside its transaction, after
// it locked the account row and before its first write; the test starts
// the other, waits until pg_stat_activity shows it waiting for a lock, and
// only then opens the gate. Without the lock, or with the lock taken
// outside the transaction, nothing waits and the test fails.

// gatedPasswords stops a hash write inside its transaction, before the
// UPDATE, which would lock the account row by itself.
type gatedPasswords struct {
	app.PasswordHashWriter
	gate *gate
}

func (p gatedPasswords) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, now time.Time) error {
	if err := p.gate.stop(); err != nil {
		return err
	}
	return p.PasswordHashWriter.UpdatePasswordHash(ctx, id, hash, now)
}

// gatedTokens stops a token creation inside its transaction, after the
// credential check, before the INSERT.
type gatedTokens struct {
	app.APITokenCreator
	gate *gate
}

func (c gatedTokens) CreateAPIToken(ctx context.Context, t app.NewAPIToken) error {
	if err := c.gate.stop(); err != nil {
		return err
	}
	return c.APITokenCreator.CreateAPIToken(ctx, t)
}

// reset sets alice's password to N3w-Passw0rd!, hashed as
// "hashed:N3w-Passw0rd!:reset".
func (a *account) reset(passwords app.PasswordHashWriter) (app.ResetPasswordResult, error) {
	return app.NewResetPassword(app.ResetPasswordDeps{
		Accounts: a.store, Passwords: passwords, Sessions: a.store, APITokens: a.store,
		Hasher: &gatedHasher{salt: "reset"}, Rules: domain.NewPasswordRules(), Tx: a.tx,
		Clock: clock.System{}, Logger: slog.New(slog.DiscardHandler),
	}).Execute(context.Background(), "alice@corp.com", "N3w-Passw0rd!")
}

// createToken creates a token as actor.
func (a *account) createToken(tokens app.APITokenCreator, actor shared.Actor) error {
	_, err := app.NewCreateAPIToken(app.CreateAPITokenDeps{
		Lock:   app.CredentialLock{Locker: a.store, Sessions: a.store, APITokens: a.store},
		Tokens: tokens, Tx: a.tx, Clock: clock.System{}, Logger: slog.New(slog.DiscardHandler),
	}).Execute(shared.WithActor(context.Background(), actor), domain.APITokenSpec{})
	return err
}

// credentials of alice that create a token: her session of before, or a
// token of hers made before, which the reset revokes as well. The lock is
// the account's row, whichever authenticates.
var creators = []struct {
	name   string
	actor  func(t *testing.T, a *account) shared.Actor
	tokens int // alice's tokens before the creation
}{
	{"with a session", func(_ *testing.T, a *account) shared.Actor {
		return shared.Actor{UserID: a.id, SessionID: a.session}
	}, 0},
	{"with a token", func(t *testing.T, a *account) shared.Actor {
		id := uuid.NewV7()
		hash := make([]byte, 32)
		hash[0] = 1
		if err := a.store.CreateAPIToken(context.Background(), app.NewAPIToken{ID: id, UserID: a.id, TokenHash: hash, Label: "script", Now: time.Now()}); err != nil {
			t.Fatal(err)
		}
		return shared.Actor{UserID: a.id, APITokenID: id}
	}, 1},
}

// signIn logs alice in with Tr0ub4dor&3.
func signIn(uc *app.Login) error {
	_, err := uc.Execute(context.Background(), app.LoginInput{Email: "alice@corp.com", Password: "Tr0ub4dor&3"})
	return err
}

// contend runs first until it stops at g, then second until Postgres shows
// a statement waiting for a lock, then opens g; it returns both errors.
func contend(t *testing.T, a *account, g *gate, first, second func() error) (error, error) {
	t.Helper()
	firstDone := run(first)
	g.await(t)
	secondDone := run(second)
	pgtest.WaitForLockWait(t, a.pool, waitLimit)
	close(g.opened)
	return await(t, firstDone), await(t, secondDone)
}

func run(f func() error) <-chan error {
	done := make(chan error, 1)
	go func() { done <- f() }()
	return done
}

// credentials is what the account holds: its stored hash, its live
// sessions, its sessions revoked by a reset, its live and its revoked
// tokens.
type credentials struct {
	hash                            string
	liveSessions, resetSessions     int
	liveAPITokens, revokedAPITokens int
}

func (a *account) credentials(t *testing.T) credentials {
	t.Helper()
	var c credentials
	err := a.pool.QueryRow(context.Background(), `SELECT password,
		(SELECT count(*) FROM auth_sessions WHERE user_id = $1 AND revoked_at IS NULL),
		(SELECT count(*) FROM auth_sessions WHERE user_id = $1 AND revoke_reason = 'password_reset'),
		(SELECT count(*) FROM api_tokens WHERE user_id = $1 AND deleted_at IS NULL),
		(SELECT count(*) FROM api_tokens WHERE user_id = $1 AND deleted_at IS NOT NULL)
		FROM users WHERE id = $1`, a.id).Scan(&c.hash, &c.liveSessions, &c.resetSessions, &c.liveAPITokens, &c.revokedAPITokens)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// Interleaving 1: a login verified the old password; the reset holds the
// lock. The login's transaction waits for it, then finds the reset's hash,
// verifies the password against it and fails: no new session.
func TestALoginWaitingForAResetFails(t *testing.T) {
	a := newAccount(t, "hashed:Tr0ub4dor&3:0")
	g := newGate()
	hasher := &gatedHasher{salt: "login"}
	var reset app.ResetPasswordResult

	resetErr, loginErr := contend(t, a, g,
		func() (err error) { reset, err = a.reset(gatedPasswords{a.store, g}); return err },
		func() error { return signIn(a.login(hasher, a.store, a.store)) })

	got := a.credentials(t)
	if resetErr != nil || !errors.Is(loginErr, domain.ErrInvalidCredentials) || reset.Sessions != 1 ||
		got != (credentials{hash: "hashed:N3w-Passw0rd!:reset", resetSessions: 1}) {
		t.Errorf("reset %+v, %v; login %v; credentials %+v; want the reset, 401 and no new session", reset, resetErr, loginErr, got)
	}
	if want := []string{"hashed:Tr0ub4dor&3:0", "hashed:N3w-Passw0rd!:reset"}; !slices.Equal(hasher.verified, want) {
		t.Errorf("the login verified %q, want %q", hasher.verified, want)
	}
}

// Interleaving 1 the other way round: the login holds the lock, so the
// reset waits for it and then revokes the login's session with the rest.
func TestAResetWaitsForALoginThatHoldsTheLock(t *testing.T) {
	a := newAccount(t, "hashed:Tr0ub4dor&3:0")
	g := newGate()
	var reset app.ResetPasswordResult

	loginErr, resetErr := contend(t, a, g,
		func() error { return signIn(a.login(&gatedHasher{salt: "login"}, a.store, gatedSessions{a.store, g})) },
		func() (err error) { reset, err = a.reset(a.store); return err })

	got := a.credentials(t)
	if loginErr != nil || resetErr != nil || reset.Sessions != 2 ||
		got != (credentials{hash: "hashed:N3w-Passw0rd!:reset", resetSessions: 2}) {
		t.Errorf("login %v; reset %+v, %v; credentials %+v; want both, the login's session revoked by the reset", loginErr, reset, resetErr, got)
	}
}

// Interleaving 2: a login verified the password against a hash of old
// parameters and hashed it again; the reset holds the lock. The login's
// transaction waits for it, finds the reset's hash and fails without
// writing its own over it.
func TestALoginWaitingForAResetDoesNotWriteItsNewHash(t *testing.T) {
	a := newAccount(t, "old:Tr0ub4dor&3")
	g := newGate()
	hasher := &gatedHasher{salt: "login"}
	var reset app.ResetPasswordResult

	resetErr, loginErr := contend(t, a, g,
		func() (err error) { reset, err = a.reset(gatedPasswords{a.store, g}); return err },
		func() error { return signIn(a.login(hasher, a.store, a.store)) })

	got := a.credentials(t)
	if resetErr != nil || !errors.Is(loginErr, domain.ErrInvalidCredentials) || reset.Sessions != 1 ||
		got != (credentials{hash: "hashed:N3w-Passw0rd!:reset", resetSessions: 1}) {
		t.Errorf("reset %+v, %v; login %v; credentials %+v; want the reset's hash, 401 and no new session", reset, resetErr, loginErr, got)
	}
	if want := []string{"old:Tr0ub4dor&3", "hashed:N3w-Passw0rd!:reset"}; !slices.Equal(hasher.verified, want) {
		t.Errorf("the login verified %q, want %q", hasher.verified, want)
	}
}

// Interleaving 2 the other way round: the login holds the lock before it
// writes its new hash, so the reset waits, then writes its own hash over
// the login's and revokes the login's session.
func TestAResetWaitsForALoginThatHashesThePasswordAgain(t *testing.T) {
	a := newAccount(t, "old:Tr0ub4dor&3")
	g := newGate()
	var reset app.ResetPasswordResult

	loginErr, resetErr := contend(t, a, g,
		func() error { return signIn(a.login(&gatedHasher{salt: "login"}, gatedPasswords{a.store, g}, a.store)) },
		func() (err error) { reset, err = a.reset(a.store); return err })

	got := a.credentials(t)
	if loginErr != nil || resetErr != nil || reset.Sessions != 2 ||
		got != (credentials{hash: "hashed:N3w-Passw0rd!:reset", resetSessions: 2}) {
		t.Errorf("login %v; reset %+v, %v; credentials %+v; want both, the reset's hash, the login's session revoked", loginErr, reset, resetErr, got)
	}
}

// Interleaving 3: a token creation with a credential of before; the reset
// holds the lock. The creation's transaction waits for it, finds the
// credential revoked and fails with 401: no new token.
func TestATokenCreationWaitingForAResetFails(t *testing.T) {
	for _, c := range creators {
		t.Run(c.name, func(t *testing.T) {
			a := newAccount(t, "hashed:Tr0ub4dor&3:0")
			actor := c.actor(t, a)
			g := newGate()
			var reset app.ResetPasswordResult

			resetErr, createErr := contend(t, a, g,
				func() (err error) { reset, err = a.reset(gatedPasswords{a.store, g}); return err },
				func() error { return a.createToken(a.store, actor) })

			var se *shared.Error
			got := a.credentials(t)
			if resetErr != nil || !errors.As(createErr, &se) || se.Code != shared.CodeUnauthorized || reset.APITokens != c.tokens ||
				got != (credentials{hash: "hashed:N3w-Passw0rd!:reset", resetSessions: 1, revokedAPITokens: c.tokens}) {
				t.Errorf("reset %+v, %v; creation %v; credentials %+v; want the reset, 401 and no new token", reset, resetErr, createErr, got)
			}
		})
	}
}

// Interleaving 3 the other way round: the creation holds the lock, so the
// reset waits for it and then revokes the new token with the rest.
func TestAResetWaitsForATokenCreationThatHoldsTheLock(t *testing.T) {
	for _, c := range creators {
		t.Run(c.name, func(t *testing.T) {
			a := newAccount(t, "hashed:Tr0ub4dor&3:0")
			actor := c.actor(t, a)
			g := newGate()
			var reset app.ResetPasswordResult

			createErr, resetErr := contend(t, a, g,
				func() error { return a.createToken(gatedTokens{a.store, g}, actor) },
				func() (err error) { reset, err = a.reset(a.store); return err })

			got := a.credentials(t)
			if createErr != nil || resetErr != nil || reset.APITokens != c.tokens+1 ||
				got != (credentials{hash: "hashed:N3w-Passw0rd!:reset", resetSessions: 1, revokedAPITokens: c.tokens + 1}) {
				t.Errorf("creation %v; reset %+v, %v; credentials %+v; want both, the new token revoked by the reset", createErr, reset, resetErr, got)
			}
		})
	}
}
