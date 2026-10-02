package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newWorkspaceRoute is the workspace module (workspace.New) on pool, as
// bootstrap wires it for the endings: access's Authorizer, identity's
// Accounts and MemberProfiles, converted, and the project module's cascade,
// each on pool.
func newWorkspaceRoute(t *testing.T, pool *pgxpool.Pool) moduleRoute {
	t.Helper()
	tx, auth, identityPorts, provided := postgres.NewTxManager(pool, 2*time.Second), authorizerOn(pool), identity.Provide(pool), workspace.Provide(pool)
	projects := project.New(project.Deps{Pool: pool, Tx: tx, Clock: clock.System{}, Authorizer: auth,
		Workspaces: projectWorkspaces{directory: provided.WorkspaceDirectory}, Members: provided.WorkspaceMembers})
	return newRoute(t, workspace.New(workspace.Deps{Pool: pool, Tx: tx, Clock: clock.System{}, Logger: slog.New(slog.DiscardHandler),
		Authorizer: auth, Accounts: workspaceAccounts{accounts: identityPorts.Accounts},
		Profiles: workspaceProfiles{profiles: identityPorts.PublicProfiles}, Projects: projects.Cascade()}).Register)
}

// Each ending runs every statement on its transaction's connection (M3
// design 3.6 convention 2, 6.7): its locks, its reads (the membership, the
// decision's role, the other admin, the member's address through
// MemberProfiles, the projects' only admin), its writes and the projects'
// step. The workspace module and the project module's cascade are wired as
// bootstrap wires them, on a pool of one connection: a statement sent
// through the pool rather than the transaction would wait for a second
// connection that never comes, and its request fail at the request's
// deadline. In acme, alice and carol are the admins and bob a member;
// alice's Web has the three, carol its admin too; a pending invitation to
// each address of the three. alice removes bob, and leaves.
func TestTheEndingsRunOnTheirTransactionsConnection(t *testing.T) {
	r := newGrowthRace(t, false)
	carol := uuid.NewV7()
	ctx, now := context.Background(), time.Now()
	users := identitypg.New(r.pool)
	if err := users.CreateUser(ctx, identityapp.NewUser{ID: carol, Email: "carol@example.com", PasswordHash: "x", DisplayName: "carol",
		Now: now}); err != nil {
		t.Fatal(err)
	}
	if err := users.CreateDefaultProfile(ctx, uuid.NewV7(), carol, now); err != nil {
		t.Fatal(err)
	}
	var acme uuid.UUID
	if err := r.pool.QueryRow(ctx, "SELECT workspace_id FROM projects WHERE id = $1", r.web).Scan(&acme); err != nil {
		t.Fatal(err)
	}
	workspaces, projects := workspacepg.New(r.pool), projectpg.New(r.pool)
	if err := workspaces.CreateMember(ctx, workspaceapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, MemberID: carol, Role: shared.RoleAdmin,
		CreatedBy: r.alice, Now: now}); err != nil {
		t.Fatal(err)
	}
	for user, role := range map[uuid.UUID]shared.Role{r.bob: shared.RoleMember, carol: shared.RoleAdmin} {
		if err := projects.CreateMember(ctx, projectapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: r.web, MemberID: user, Role: role,
			CreatedBy: r.alice, Now: now}); err != nil {
			t.Fatal(err)
		}
	}
	for _, email := range []string{"alice@example.com", "bob@example.com", "carol@example.com"} {
		if _, err := workspaces.CreateInvitations(ctx, []workspaceapp.InvitationRow{{ID: uuid.NewV7(), WorkspaceID: acme, Email: email,
			Role: shared.RoleGuest, CreatedBy: r.alice, Now: now}}); err != nil {
			t.Fatal(err)
		}
	}
	route := newWorkspaceRoute(t, poolOfOne(t, r.pool.Config().ConnString()))
	contract := apitest.Load(t)

	route.answerWithin(t, contract, http.MethodDelete, "/api/v0/workspace-members/"+r.bobIn.String(), r.alice, "", http.StatusNoContent)
	route.answerWithin(t, contract, http.MethodPost, "/api/v0/workspaces/acme/leave", r.alice, "", http.StatusNoContent)

	var ended, pending int
	if err := r.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM project_members WHERE project_id = $1 AND NOT is_active),
		(SELECT count(*) FROM workspace_member_invites WHERE workspace_id = $2 AND deleted_at IS NULL)`, r.web, acme).Scan(&ended, &pending); err != nil {
		t.Fatal(err)
	}
	if ended != 2 || pending != 1 {
		t.Errorf("after the removal and the leaving: %d ended memberships of Web, %d pending invitations; want 2, carol's alone", ended, pending)
	}
}
