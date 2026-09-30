// Package workspace is the workspaces module (M3 design 3.3, 6.2):
// workspaces and their members. It brings creating, listing, reading,
// changing and deleting workspaces, checking a slug, listing the members
// and changing their roles, and each member's display settings, and offers
// the other modules its reads through ports.
package workspace

import (
	"context"
	"log/slog"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/http"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// WorkspaceRoles reads a caller's active role in a workspace: access's port
// (M3 design 6.5). ok is false unless the membership is active, its row not
// deleted and the workspace not deleted. It reads in the transaction ctx
// carries.
type WorkspaceRoles interface {
	ActiveRole(ctx context.Context, workspaceID, userID uuid.UUID) (role shared.Role, ok bool, err error)
}

// Provided are the adapters workspace offers the other modules. They depend
// on the pool alone, so bootstrap builds them before any module (M3 design
// 6.6, step 2).
type Provided struct {
	WorkspaceRoles WorkspaceRoles
}

// Provide builds workspace's adapters for the other modules.
func Provide(pool *pgxpool.Pool) Provided {
	return Provided{WorkspaceRoles: postgresadapter.New(pool)}
}

// AccountState is the account state the Accounts port hands over: bootstrap
// converts identity's into it (M3 design 6.5).
type AccountState = app.AccountState

// PublicProfile is the profile the MemberProfiles port hands over: bootstrap
// converts identity's into it (M3 design 6.5).
type PublicProfile = app.PublicProfile

// Deps are what bootstrap gives the module (M3 design 6.6, step 5).
type Deps struct {
	Pool       *pgxpool.Pool
	Tx         shared.TxManager
	Clock      app.Clock
	Logger     *slog.Logger
	Authorizer shared.Authorizer
	// Accounts is identity's Accounts, converted (bootstrap/ports.go).
	Accounts app.Accounts
	// Profiles is identity's PublicProfiles, converted (bootstrap/ports.go).
	Profiles app.MemberProfiles
	// CreationEnabled is workspace.creation_enabled (M3 design 3.11).
	CreationEnabled bool
}

// Module is the wired workspace module.
type Module struct {
	uc httpadapter.UseCases
}

// New wires the module's use cases and its HTTP side from d; nothing is
// registered or injected after it (M3 design 6.6).
func New(d Deps) *Module {
	store := postgresadapter.New(d.Pool)
	return &Module{uc: httpadapter.UseCases{
		ListWorkspaces: app.NewListWorkspaces(store),
		CreateWorkspace: app.NewCreateWorkspace(app.CreateWorkspaceDeps{
			Accounts: d.Accounts, Workspaces: store, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger, Enabled: d.CreationEnabled,
		}),
		GetWorkspace:      app.NewGetWorkspace(store, d.Authorizer),
		UpdateWorkspace:   app.NewUpdateWorkspace(store, d.Authorizer, d.Tx, d.Clock),
		DeleteWorkspace:   app.NewDeleteWorkspace(store, d.Authorizer, d.Tx, d.Clock, d.Logger),
		ListMembers:       app.NewListWorkspaceMembers(store, d.Profiles, d.Authorizer),
		UpdateMember:      app.NewUpdateWorkspaceMember(store, d.Profiles, d.Authorizer, d.Tx, d.Clock),
		CheckSlug:         app.NewCheckSlug(store),
		GetPreferences:    app.NewGetWorkspacePreferences(store, d.Authorizer),
		UpdatePreferences: app.NewUpdateWorkspacePreferences(store, d.Authorizer, d.Tx, d.Clock),
	}}
}

// Register mounts the module's API on router behind api's middlewares.
func (m *Module) Register(router *httpserver.Router, api *httpserver.API) {
	httpadapter.Register(router, api, m.uc)
}

// Actions lists the module's actions: bootstrap's test holds the union of
// every module's actions equal to access's rule table (M3 design 3.4).
func Actions() []shared.Action {
	return domain.Actions()
}

// ReservedSlugs is the reserved list (M3 design 3.10): bootstrap holds its
// server section equal to the top-level paths the server answers itself.
func ReservedSlugs() domain.ReservedSlugs {
	return domain.Reserved()
}
