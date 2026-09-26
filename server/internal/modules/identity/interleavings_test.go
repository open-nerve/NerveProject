package identity_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/signing"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The interleavings of the account row lock protocol (M2 design 3.5) run
// the use cases on a real database. A gated hasher stops one of them in
// argon2, outside its transaction, while the test commits another; a gated
// session insert stops a login inside its transaction, holding the lock,
// until another waits for the lock. Every wait has a deadline, so a test
// fails rather than hangs.

// waitLimit bounds every wait of these tests.
const waitLimit = 10 * time.Second

// gatedHasher stands in for argon2: Hash gives "hashed:<password>:<salt>",
// and Verify takes that for any salt, and "old:<password>" as a hash of
// other parameters, which it asks to rehash. With a gate, its first Verify
// stops until the test opens the gate. It records the hashes it verified.
type gatedHasher struct {
	salt     string
	gate     *gate // nil: never stops
	mu       sync.Mutex
	verified []string
}

func (h *gatedHasher) Hash(_ context.Context, password string) (string, error) {
	return "hashed:" + password + ":" + h.salt, nil
}

func (h *gatedHasher) Verify(_ context.Context, password, hash string) (bool, bool, error) {
	h.mu.Lock()
	h.verified = append(h.verified, hash)
	first := len(h.verified) == 1
	h.mu.Unlock()
	if first && h.gate != nil {
		if err := h.gate.stop(); err != nil {
			return false, false, err
		}
	}
	if hash == "old:"+password {
		return true, true, nil
	}
	return strings.HasPrefix(hash, "hashed:"+password+":"), false, nil
}

// gate stops a Verify until the test opens it.
type gate struct {
	reached, opened chan struct{}
}

func newGate() *gate { return &gate{reached: make(chan struct{}), opened: make(chan struct{})} }

// stop tells the test that a Verify reached the gate and waits until it is
// opened.
func (g *gate) stop() error {
	close(g.reached)
	select {
	case <-g.opened:
		return nil
	case <-time.After(waitLimit):
		return errors.New("the gate was never opened")
	}
}

// await waits until a Verify stops at the gate.
func (g *gate) await(t *testing.T) {
	t.Helper()
	select {
	case <-g.reached:
	case <-time.After(waitLimit):
		t.Fatal("nothing reached the gate")
	}
}

// gatedSessions stops a login inside its transaction, after it locked the
// account row and found the hash unchanged, before it inserts its session.
type gatedSessions struct {
	app.SessionCreator
	gate *gate
}

func (s gatedSessions) CreateSession(ctx context.Context, session app.NewSession) error {
	if err := s.gate.stop(); err != nil {
		return err
	}
	return s.SessionCreator.CreateSession(ctx, session)
}

// account is alice@corp.com, whose password is Tr0ub4dor&3 and whose
// stored hash is given, with a live session, on a database of its own.
type account struct {
	store   *postgresadapter.Store
	pool    *pgxpool.Pool
	tx      shared.TxManager
	id      uuid.UUID
	session uuid.UUID
}

func newAccount(t *testing.T, hash string) *account {
	t.Helper()
	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, config.DatabaseConfig{URL: pgtest.NewDatabase(t), MaxConns: 8})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	a := &account{store: postgresadapter.New(pool), pool: pool, tx: postgres.NewTxManager(pool, 2*time.Second), id: uuid.NewV7(), session: uuid.NewV7()}
	now := time.Now()
	err = errors.Join(
		a.store.CreateUser(ctx, app.NewUser{ID: a.id, Email: "alice@corp.com", PasswordHash: hash, DisplayName: "alice", Now: now}),
		a.store.CreateDefaultProfile(ctx, uuid.NewV7(), a.id, now),
		a.store.CreateSession(ctx, app.NewSession{ID: a.session, UserID: a.id, TokenHash: make([]byte, 32), ExpiresAt: now.Add(time.Hour), Now: now}),
	)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func (a *account) login(h app.PasswordHasher, sessions app.SessionCreator) *app.Login {
	keys := signing.EphemeralKeys()
	return app.NewLogin(app.LoginDeps{
		Accounts: a.store, Locker: a.store, Passwords: a.store, Sessions: sessions, Hasher: h, Tx: a.tx,
		Issuance: app.Issuance{Tokens: signing.NewAccessTokens(keys), MAC: signing.NewRefreshTokenMAC(keys), AccessTTL: time.Minute, SessionTTL: time.Hour},
		Clock:    clock.System{}, Logger: slog.New(slog.DiscardHandler), DummyHash: "hashed:dummy:0",
	})
}

func (a *account) changePassword(h app.PasswordHasher) *app.ChangePassword {
	return app.NewChangePassword(app.ChangePasswordDeps{
		Accounts: a.store, Lock: app.CredentialLock{Locker: a.store, Sessions: a.store, APITokens: a.store},
		Passwords: a.store, Sessions: a.store, Hasher: h, Rules: domain.NewPasswordRules(), Tx: a.tx,
		Clock: clock.System{}, Logger: slog.New(slog.DiscardHandler),
	})
}

// loginAsync starts a login with Tr0ub4dor&3 and returns where its error
// arrives.
func loginAsync(uc *app.Login) <-chan error {
	done := make(chan error, 1)
	go func() {
		_, err := uc.Execute(context.Background(), app.LoginInput{Email: "alice@corp.com", Password: "Tr0ub4dor&3"})
		done <- err
	}()
	return done
}

func await(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(waitLimit):
		t.Fatal("the login did not finish")
		return nil
	}
}

// sessionsAndHash are the account's session count and its stored hash.
func (a *account) sessionsAndHash(t *testing.T) (int, string) {
	t.Helper()
	var n int
	var hash string
	err := a.pool.QueryRow(context.Background(),
		`SELECT (SELECT count(*) FROM auth_sessions WHERE user_id = $1), password FROM users WHERE id = $1`, a.id).Scan(&n, &hash)
	if err != nil {
		t.Fatal(err)
	}
	return n, hash
}

// Interleaving 4: a login verified the old password; the password change
// commits before the login's transaction begins, so no transaction waits
// for the lock. The login's transaction finds a hash other than its
// snapshot; the login verifies the password again, against that hash, and
// fails: no session. It pins the snapshot comparison and the second verify;
// TestAPasswordChangeWaitsForALoginThatHoldsTheLock pins the lock.
func TestALoginWithTheOldPasswordFailsWhenThePasswordChangesMeanwhile(t *testing.T) {
	a := newAccount(t, "hashed:Tr0ub4dor&3:0")
	g := newGate()
	loginHasher := &gatedHasher{salt: "login", gate: g}

	done := loginAsync(a.login(loginHasher, a.store))
	g.await(t)
	changed := a.changePassword(&gatedHasher{salt: "change"}).Execute(
		shared.WithActor(context.Background(), shared.Actor{UserID: a.id, SessionID: a.session}),
		app.ChangePasswordInput{Current: "Tr0ub4dor&3", New: "N3w-Passw0rd!"})
	close(g.opened)
	err := await(t, done)

	sessions, hash := a.sessionsAndHash(t)
	if changed != nil || !errors.Is(err, domain.ErrInvalidCredentials) || sessions != 1 || hash != "hashed:N3w-Passw0rd!:change" {
		t.Errorf("change %v; login %v; %d sessions, hash %q; want the change, 401, only the session of before, the new hash", changed, err, sessions, hash)
	}
	if want := []string{"hashed:Tr0ub4dor&3:0", "hashed:N3w-Passw0rd!:change"}; !slices.Equal(loginHasher.verified, want) {
		t.Errorf("the login verified %q, want %q", loginHasher.verified, want)
	}
}

// Interleaving 5: two logins read the hash of old parameters; the second
// hashes the password again and commits before the first's transaction
// begins, so no transaction waits for the lock. The first's transaction
// finds the new hash instead of its snapshot; the first verifies the
// password against it once more and signs in, without writing its own hash
// over the new one. It pins the snapshot comparison, the second verify and
// that a stale rehash is not written.
func TestTwoLoginsWhileOneHashesThePasswordAgain(t *testing.T) {
	a := newAccount(t, "old:Tr0ub4dor&3")
	g := newGate()
	first := &gatedHasher{salt: "first", gate: g}

	done := loginAsync(a.login(first, a.store))
	g.await(t)
	_, second := a.login(&gatedHasher{salt: "second"}, a.store).Execute(context.Background(), app.LoginInput{Email: "alice@corp.com", Password: "Tr0ub4dor&3"})
	close(g.opened)
	err := await(t, done)

	sessions, hash := a.sessionsAndHash(t)
	if second != nil || err != nil || sessions != 3 || hash != "hashed:Tr0ub4dor&3:second" {
		t.Errorf("logins %v, %v; %d sessions, hash %q; want both signed in beside the session of before, the second's hash", err, second, sessions, hash)
	}
	if want := []string{"old:Tr0ub4dor&3", "hashed:Tr0ub4dor&3:second"}; !slices.Equal(first.verified, want) {
		t.Errorf("the first login verified %q, want %q", first.verified, want)
	}
}

// Interleaving 4 the other way round pins the account row lock: a password
// change must wait for the transaction of a login that holds the lock, and
// then revoke the session that login created, so signing in with the old
// password leaves no live session behind. The order is forced: the login
// stops inside its transaction, holding the lock, before it inserts its
// session; the test waits until pg_stat_activity shows the change waiting
// for a lock, and only then lets the login go on. Every wait has a
// deadline.
func TestAPasswordChangeWaitsForALoginThatHoldsTheLock(t *testing.T) {
	a := newAccount(t, "hashed:Tr0ub4dor&3:0")
	g := newGate()

	done := loginAsync(a.login(&gatedHasher{salt: "login"}, gatedSessions{a.store, g}))
	g.await(t)
	changed := make(chan error, 1)
	go func() {
		changed <- a.changePassword(&gatedHasher{salt: "change"}).Execute(
			shared.WithActor(context.Background(), shared.Actor{UserID: a.id, SessionID: a.session}),
			app.ChangePasswordInput{Current: "Tr0ub4dor&3", New: "N3w-Passw0rd!"})
	}()
	pgtest.WaitForLockWait(t, a.pool, waitLimit)
	close(g.opened)
	err := await(t, done)
	var changeErr error
	select {
	case changeErr = <-changed:
	case <-time.After(waitLimit):
		t.Fatal("the change did not finish")
	}

	var live, revoked int
	var hash string
	if err := a.pool.QueryRow(context.Background(), `SELECT
		(SELECT count(*) FROM auth_sessions WHERE user_id = $1 AND revoked_at IS NULL),
		(SELECT count(*) FROM auth_sessions WHERE user_id = $1 AND revoke_reason = 'password_changed'),
		password FROM users WHERE id = $1`, a.id).Scan(&live, &revoked, &hash); err != nil {
		t.Fatal(err)
	}
	if err != nil || changeErr != nil || live != 1 || revoked != 1 || hash != "hashed:N3w-Passw0rd!:change" {
		t.Errorf("login %v; change %v; %d live sessions, %d revoked for password_changed, hash %q; want both done, the login's session revoked, the session of before live, the new hash",
			err, changeErr, live, revoked, hash)
	}
}
