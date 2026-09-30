package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	identitydomain "github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Creating a workspace against M2's deactivation of its admin's account, in
// both orders, on a real database (M3 design 3.6 convention 6; the
// interleaving 8 of 9.3 without the membership's end, which the
// deactivation adds with its port). Each side runs its real use case; a
// gate inside its transaction, after its lock of the account row, holds the
// transaction open, and pgtest.WaitForLockWait proves that the other side
// waits on that row before the gate opens. Every wait has a deadline.

// gate holds a transaction open: the first call of wait signals held and
// waits until open is closed, the caller's context ends, or 10 seconds
// after the gate was made, whichever comes first. The gate's own deadline
// is for a side that lost the test's context, such as a use case that runs
// a statement on context.Background(): its wait would never end, and the
// test's cleanup, closing the pool, would wait for its connection forever.
type gate struct {
	held, open chan struct{}
	deadline   time.Time
}

func newGate() *gate {
	return &gate{held: make(chan struct{}), open: make(chan struct{}), deadline: time.Now().Add(10 * time.Second)}
}

func (g *gate) wait(ctx context.Context) error {
	close(g.held)
	expired := time.NewTimer(time.Until(g.deadline))
	defer expired.Stop()
	select {
	case <-g.open:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-expired.C:
		return errors.New("the gate was not opened within 10s")
	}
}

// gatedSessions stops a use case before it revokes the sessions, holding
// the account row: a deactivation at its last write, a change of address
// after its write.
type gatedSessions struct {
	identityapp.SessionRevoker
	gate *gate
}

func (s gatedSessions) RevokeSessions(ctx context.Context, userID, keep uuid.UUID, reason identitydomain.RevokeReason, now time.Time) (int, error) {
	if err := s.gate.wait(ctx); err != nil {
		return 0, err
	}
	return s.SessionRevoker.RevokeSessions(ctx, userID, keep, reason, now)
}

// gatedWorkspaces stops the creation before its first insert, holding the
// account row's lock.
type gatedWorkspaces struct {
	*workspacepg.Store
	gate *gate
}

func (w gatedWorkspaces) CreateWorkspace(ctx context.Context, row workspaceapp.WorkspaceRow) (domain.Workspace, error) {
	if err := w.gate.wait(ctx); err != nil {
		return domain.Workspace{}, err
	}
	return w.Store.CreateWorkspace(ctx, row)
}

type race struct {
	pool  *pgxpool.Pool
	alice uuid.UUID
}

// newRace is a database with alice's account.
func newRace(t *testing.T) race {
	t.Helper()
	pool := openPool(t, pgtest.NewDatabase(t))
	users := identitypg.New(pool)
	r := race{pool: pool, alice: uuid.NewV7()}
	now := time.Now()
	if err := users.CreateUser(context.Background(), identityapp.NewUser{
		ID: r.alice, Email: "alice@example.com", PasswordHash: "x", DisplayName: "alice", Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := users.CreateDefaultProfile(context.Background(), uuid.NewV7(), r.alice, now); err != nil {
		t.Fatal(err)
	}
	return r
}

// deactivation is M2's `nerve users deactivate` over sessions.
func (r race) deactivation(sessions identityapp.SessionRevoker) *identityapp.Deactivate {
	store := identitypg.New(r.pool)
	return identityapp.NewDeactivate(identityapp.DeactivateDeps{
		Accounts: store, Users: store, Profiles: store, Sessions: sessions,
		Tx: postgres.NewTxManager(r.pool, 2*time.Second), Clock: clocktest.At(time.Now()), Logger: slog.New(slog.DiscardHandler),
	})
}

// creation is createWorkspace over workspaces, with identity's Accounts as
// bootstrap wires it.
func (r race) creation(workspaces workspaceapp.WorkspaceCreator) *workspaceapp.CreateWorkspace {
	return workspaceapp.NewCreateWorkspace(workspaceapp.CreateWorkspaceDeps{
		Accounts:   workspaceAccounts{accounts: identity.Provide(r.pool).Accounts},
		Workspaces: workspaces,
		Tx:         postgres.NewTxManager(r.pool, 2*time.Second),
		Clock:      clocktest.At(time.Now()),
		Logger:     slog.New(slog.DiscardHandler),
		Enabled:    true,
	})
}

// state is whether alice's account is active, and how many workspaces and
// memberships there are.
func (r race) state(t *testing.T) (active bool, workspaces, members int) {
	t.Helper()
	if err := r.pool.QueryRow(context.Background(),
		"SELECT is_active, (SELECT count(*) FROM workspaces), (SELECT count(*) FROM workspace_members) FROM users WHERE id = $1", r.alice).
		Scan(&active, &workspaces, &members); err != nil {
		t.Fatal(err)
	}
	return active, workspaces, members
}

// run starts fn and returns the channel of its result.
func run(fn func() error) <-chan error {
	done := make(chan error, 1)
	go func() { done <- fn() }()
	return done
}

func held(t *testing.T, ctx context.Context, g *gate, done <-chan error, what string) {
	t.Helper()
	select {
	case <-g.held:
	case err := <-done:
		t.Fatalf("%s ended before its gate: %v", what, err)
	case <-ctx.Done():
		t.Fatalf("%s did not reach its gate within 10s", what)
	}
}

func result(t *testing.T, ctx context.Context, done <-chan error, what string) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		t.Fatalf("%s did not end within 10s", what)
		return nil
	}
}

// Deactivation first: it holds the account row; the creation waits on its
// FOR SHARE, then reads the account deactivated under the lock and answers
// 401. No workspace.
func TestDeactivationFirstRefusesTheWorkspace(t *testing.T) {
	r := newRace(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	g := newGate()
	deactivated := run(func() error {
		_, err := r.deactivation(gatedSessions{identitypg.New(r.pool), g}).ExecuteByEmail(ctx, "alice@example.com")
		return err
	})
	held(t, ctx, g, deactivated, "the deactivation")
	created := run(func() error {
		_, err := r.creation(workspacepg.New(r.pool)).Execute(shared.WithActor(ctx, shared.Actor{UserID: r.alice}),
			domain.NewWorkspace{Name: "Acme", Slug: "acme"})
		return err
	})
	pgtest.WaitForLockWait(t, r.pool, 5*time.Second)
	close(g.open)

	if err := result(t, ctx, deactivated, "the deactivation"); err != nil {
		t.Fatalf("the deactivation: %v", err)
	}
	if err := result(t, ctx, created, "the creation"); !errors.Is(err, shared.Unauthenticated()) {
		t.Errorf("the creation = %v, want 401 unauthorized", err)
	}
	if active, workspaces, members := r.state(t); active || workspaces != 0 || members != 0 {
		t.Errorf("alice active %v, %d workspaces, %d memberships; want deactivated and none", active, workspaces, members)
	}
}

// Creation first: it holds the account row FOR SHARE; the deactivation
// waits on its FOR NO KEY UPDATE until the workspace is committed, then
// deactivates the account. Both succeed.
func TestCreationFirstHoldsOffTheDeactivation(t *testing.T) {
	r := newRace(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	g := newGate()
	created := run(func() error {
		_, err := r.creation(gatedWorkspaces{workspacepg.New(r.pool), g}).Execute(shared.WithActor(ctx, shared.Actor{UserID: r.alice}),
			domain.NewWorkspace{Name: "Acme", Slug: "acme"})
		return err
	})
	held(t, ctx, g, created, "the creation")
	deactivated := run(func() error {
		_, err := r.deactivation(identitypg.New(r.pool)).ExecuteByEmail(ctx, "alice@example.com")
		return err
	})
	pgtest.WaitForLockWait(t, r.pool, 5*time.Second)
	if active, workspaces, members := r.state(t); !active || workspaces != 0 || members != 0 {
		t.Errorf("while the creation holds the row: alice active %v, %d workspaces, %d memberships; want active and none committed",
			active, workspaces, members)
	}
	close(g.open)

	if err := result(t, ctx, created, "the creation"); err != nil {
		t.Errorf("the creation = %v, want it done", err)
	}
	if err := result(t, ctx, deactivated, "the deactivation"); err != nil {
		t.Errorf("the deactivation = %v, want it done", err)
	}
	if active, workspaces, members := r.state(t); active || workspaces != 1 || members != 1 {
		t.Errorf("alice active %v, %d workspaces, %d memberships; want deactivated after one workspace and its admin", active, workspaces, members)
	}
}
