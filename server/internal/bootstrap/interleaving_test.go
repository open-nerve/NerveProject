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

// Interleaving 8 of M3 design 9.3: creating a workspace against the
// deactivation of its admin's account, in both orders, on a real database
// (3.6 convention 6), through the API's creation and the command line's
// (`nerve workspaces create`), against `nerve users deactivate` as
// bootstrap wires it, its memberships' step included. Each side runs its
// real use case; a gate inside its transaction, after its lock of the
// account row, holds the transaction open, and pgtest.WaitForLockWaitOn
// proves that the other side waits on that row before the gate opens. Every
// wait has a deadline.

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
// the account row: a deactivation before its memberships' step, a change of
// address after its write.
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

// creating is one entry of the creation of acme by alice, and what it
// answers when it finds her account deactivated under its lock.
type creating struct {
	name        string
	create      func(ctx context.Context, uc *workspaceapp.CreateWorkspace, alice uuid.UUID) error
	deactivated error
}

var creatings = []creating{
	{"createWorkspace", func(ctx context.Context, uc *workspaceapp.CreateWorkspace, alice uuid.UUID) error {
		_, err := uc.Execute(shared.WithActor(ctx, shared.Actor{UserID: alice}), domain.NewWorkspace{Name: "Acme", Slug: "acme"})
		return err
	}, shared.Unauthenticated()},
	{"nerve workspaces create", func(ctx context.Context, uc *workspaceapp.CreateWorkspace, _ uuid.UUID) error {
		_, err := uc.ExecuteForAdmin(ctx, "alice@example.com", domain.NewWorkspace{Name: "Acme", Slug: "acme"})
		return err
	}, domain.ErrAccountDeactivated},
}

// state is whether alice's account is active, how many workspaces there
// are and how many memberships, and how many of those are active.
func (r race) state(t *testing.T) (active bool, workspaces, members, activeMembers int) {
	t.Helper()
	if err := r.pool.QueryRow(pgtest.Soon(t), `SELECT is_active, (SELECT count(*) FROM workspaces), (SELECT count(*) FROM workspace_members),
		(SELECT count(*) FROM workspace_members WHERE is_active) FROM users WHERE id = $1`, r.alice).
		Scan(&active, &workspaces, &members, &activeMembers); err != nil {
		t.Fatal(err)
	}
	return active, workspaces, members, activeMembers
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
// FOR SHARE, then reads the account deactivated under the lock and is
// refused: the API's 401, the command's workspace.account_deactivated. No
// workspace.
func TestDeactivationFirstRefusesTheWorkspace(t *testing.T) {
	for _, c := range creatings {
		t.Run(c.name, func(t *testing.T) {
			r := newRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			deactivated := run(func() error {
				_, err := deactivating(r.pool, gatedSessions{identitypg.New(r.pool), g}, workspacepg.New(r.pool)).
					ExecuteByEmail(ctx, "alice@example.com")
				return err
			})
			held(t, ctx, g, deactivated, "the deactivation")
			created := run(func() error { return c.create(ctx, r.creation(workspacepg.New(r.pool)), r.alice) })
			pgtest.WaitForLockWaitOn(t, r.pool, "users", 5*time.Second)
			close(g.open)

			if err := result(t, ctx, deactivated, "the deactivation"); err != nil {
				t.Fatalf("the deactivation: %v", err)
			}
			if err := result(t, ctx, created, "the creation"); !errors.Is(err, c.deactivated) {
				t.Errorf("the creation = %v, want %v", err, c.deactivated)
			}
			if active, workspaces, members, _ := r.state(t); active || workspaces != 0 || members != 0 {
				t.Errorf("alice active %v, %d workspaces, %d memberships; want deactivated and none", active, workspaces, members)
			}
		})
	}
}

// Creation first: it holds the account row FOR SHARE; the deactivation
// waits on its FOR NO KEY UPDATE until the workspace is committed, then
// deactivates the account and ends its membership of the new workspace,
// whose only member, and so admin, it is (3.7 rule 2 allows it). Both
// succeed.
func TestCreationFirstHoldsOffTheDeactivation(t *testing.T) {
	for _, c := range creatings {
		t.Run(c.name, func(t *testing.T) {
			r := newRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			created := run(func() error { return c.create(ctx, r.creation(gatedWorkspaces{workspacepg.New(r.pool), g}), r.alice) })
			held(t, ctx, g, created, "the creation")
			deactivated := run(func() error {
				_, err := deactivating(r.pool, identitypg.New(r.pool), workspacepg.New(r.pool)).ExecuteByEmail(ctx, "alice@example.com")
				return err
			})
			pgtest.WaitForLockWaitOn(t, r.pool, "users", 5*time.Second)
			if active, workspaces, members, _ := r.state(t); !active || workspaces != 0 || members != 0 {
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
			if active, workspaces, members, activeMembers := r.state(t); active || workspaces != 1 || members != 1 || activeMembers != 0 {
				t.Errorf("alice active %v, %d workspaces, %d memberships, %d active; want deactivated after one workspace and its admin, "+
					"her membership ended", active, workspaces, members, activeMembers)
			}
		})
	}
}
