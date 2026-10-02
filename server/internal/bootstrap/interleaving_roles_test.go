package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The interleaving 2 of M3 design 9.3: two admins demote each other at once,
// in both orders, on a real database, through the real use case and the
// Authorizer as bootstrap wires them. The first side's gate, inside its
// transaction after its decision and before its write, holds the workspace's
// lock; pgtest.WaitForLockWaitOn proves that the other side waits on the
// workspace row before the gate opens. The first side succeeds; the other
// decides once the first has committed, as a member, and is refused
// forbidden: the workspace keeps an admin (3.6 convention 2, spike S1b).

// gatedMembers stops a role change before its write, after its decision,
// holding the workspace row's lock.
type gatedMembers struct {
	*workspacepg.Store
	gate *gate
}

func (m gatedMembers) UpdateMemberRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (
	workspacedomain.Membership, error) {
	if err := m.gate.wait(ctx); err != nil {
		return workspacedomain.Membership{}, err
	}
	return m.Store.UpdateMemberRole(ctx, id, role, by, now)
}

// adminRace is a database with acme, whose admins are alice and bob, and the
// ids of their accounts and memberships. bob's account has its default
// profile, as alice's (newRace) and every account production creates.
type adminRace struct {
	race
	bob            uuid.UUID
	aliceIn, bobIn uuid.UUID
}

func newAdminRace(t *testing.T) adminRace {
	t.Helper()
	r := adminRace{race: newRace(t), bob: uuid.NewV7(), aliceIn: uuid.NewV7(), bobIn: uuid.NewV7()}
	now := time.Now()
	users := identitypg.New(r.pool)
	if err := users.CreateUser(context.Background(), identityapp.NewUser{
		ID: r.bob, Email: "bob@example.com", PasswordHash: "x", DisplayName: "bob", Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := users.CreateDefaultProfile(context.Background(), uuid.NewV7(), r.bob, now); err != nil {
		t.Fatal(err)
	}
	store := workspacepg.New(r.pool)
	w, err := store.CreateWorkspace(context.Background(), workspaceapp.WorkspaceRow{
		ID: uuid.NewV7(), Name: "Acme", Slug: "acme", Timezone: "UTC", CreatedBy: r.alice, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	for id, user := range map[uuid.UUID]uuid.UUID{r.aliceIn: r.alice, r.bobIn: r.bob} {
		if err := store.CreateMember(context.Background(), workspaceapp.MemberRow{
			ID: id, WorkspaceID: w.ID, MemberID: user, Role: shared.RoleAdmin, CreatedBy: r.alice, Now: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

// change is updateWorkspaceMember over members, with project's cascade,
// identity's profiles and the Authorizer as bootstrap wires them.
func (r adminRace) change(members workspaceapp.MemberUpdater) *workspaceapp.UpdateWorkspaceMember {
	return workspaceapp.NewUpdateWorkspaceMember(members, project.New(project.Deps{Pool: r.pool}).Cascade(),
		workspaceProfiles{profiles: identity.Provide(r.pool).PublicProfiles},
		authorizerOn(r.pool), postgres.NewTxManager(r.pool, 2*time.Second), clocktest.At(time.Now()))
}

// roles are alice's and bob's roles in acme.
func (r adminRace) roles(t *testing.T) (alice, bob shared.Role) {
	t.Helper()
	if err := r.pool.QueryRow(soon(t),
		"SELECT (SELECT role FROM workspace_members WHERE id = $1), (SELECT role FROM workspace_members WHERE id = $2)", r.aliceIn, r.bobIn).
		Scan(&alice, &bob); err != nil {
		t.Fatal(err)
	}
	return alice, bob
}

func TestTwoAdminsDemotingEachOtherLeaveAnAdmin(t *testing.T) {
	for _, aliceFirst := range []bool{true, false} {
		name := "bob first"
		if aliceFirst {
			name = "alice first"
		}
		t.Run(name, func(t *testing.T) {
			r := newAdminRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			first, second := r.alice, r.bob
			firstTarget, secondTarget := r.bobIn, r.aliceIn
			if !aliceFirst {
				first, second, firstTarget, secondTarget = second, first, secondTarget, firstTarget
			}
			g := newGate()
			demoted := run(func() error {
				_, err := r.change(gatedMembers{workspacepg.New(r.pool), g}).Execute(shared.WithActor(ctx, shared.Actor{UserID: first}),
					firstTarget, shared.RoleMember)
				return err
			})
			held(t, ctx, g, demoted, "the first demotion")
			refused := run(func() error {
				_, err := r.change(workspacepg.New(r.pool)).Execute(shared.WithActor(ctx, shared.Actor{UserID: second}),
					secondTarget, shared.RoleMember)
				return err
			})
			pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
			if alice, bob := r.roles(t); alice != shared.RoleAdmin || bob != shared.RoleAdmin {
				t.Errorf("while the first holds the lock: alice %d, bob %d; want both admins", alice, bob)
			}
			close(g.open)

			if err := result(t, ctx, demoted, "the first demotion"); err != nil {
				t.Errorf("the first demotion = %v, want it done", err)
			}
			if err := result(t, ctx, refused, "the second demotion"); !errors.Is(err, shared.Forbidden()) {
				t.Errorf("the second demotion = %v, want 403 forbidden", err)
			}
			wantAlice, wantBob := shared.RoleAdmin, shared.RoleMember
			if !aliceFirst {
				wantAlice, wantBob = shared.RoleMember, shared.RoleAdmin
			}
			if alice, bob := r.roles(t); alice != wantAlice || bob != wantBob {
				t.Errorf("alice %d, bob %d; want %d, %d: the first one's change, and an admin left", alice, bob, wantAlice, wantBob)
			}
		})
	}
}
