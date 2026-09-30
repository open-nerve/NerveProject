// Package workspace is the workspaces module (M3 design 3.3, 6.2):
// workspaces, their members and their invitations. It brings creating,
// listing, reading, changing and deleting workspaces, checking a slug,
// listing the members and changing their roles, each member's display
// settings, and the invitations, and offers the other modules its reads
// through ports.
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

// InvitationMACPurpose is the purpose of the invitation MAC that bootstrap
// asks identity's keys for: its key's HKDF info is "nerve
// workspace-invitation mac v1" (M3 design 3.8).
const InvitationMACPurpose = "workspace-invitation"

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
	// InvitationMAC is identity's MAC of InvitationMACPurpose.
	InvitationMAC app.InvitationMAC
	// CallerLock is identity's credential lock (identity.Provide).
	CallerLock app.CallerLock
	// CreationEnabled is workspace.creation_enabled (M3 design 3.11).
	CreationEnabled bool
}

// Module is the wired workspace module.
type Module struct {
	uc     httpadapter.UseCases
	signup *app.SignupInvitations
}

// SignupInvitations checks the invitation a registration names while
// sign-up is closed (M3 design 3.8): bootstrap's signup policy asks it.
type SignupInvitations interface {
	// Allows reports whether token is the link of the invitation id,
	// pending, to email, normalized. Every other case is the same false.
	Allows(ctx context.Context, email string, id uuid.UUID, token string) (bool, error)
}

// New wires the module's use cases and its HTTP side from d; nothing is
// registered or injected after it (M3 design 6.6).
func New(d Deps) *Module {
	store := postgresadapter.New(d.Pool)
	return &Module{signup: app.NewSignupInvitations(store, d.InvitationMAC), uc: httpadapter.UseCases{
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
		ListInvitations:   app.NewListWorkspaceInvitations(store, d.Authorizer, d.InvitationMAC),
		CreateInvitations: app.NewCreateWorkspaceInvitations(app.CreateInvitationsDeps{
			Caller: d.CallerLock, Invitations: store, Profiles: d.Profiles, Auth: d.Authorizer, Tx: d.Tx, Clock: d.Clock, MAC: d.InvitationMAC,
		}),
		GetInvitation:    app.NewGetWorkspaceInvitation(store, d.InvitationMAC),
		UpdateInvitation: app.NewUpdateWorkspaceInvitation(store, d.Authorizer, d.Tx, d.Clock, d.InvitationMAC),
		DeleteInvitation: app.NewDeleteWorkspaceInvitation(store, d.Authorizer, d.Tx, d.Clock),
		AcceptInvitation: app.NewAcceptWorkspaceInvitation(app.AcceptInvitationDeps{
			Accounts: d.Accounts, Invitations: store, Tx: d.Tx, Clock: d.Clock, MAC: d.InvitationMAC,
		}),
		DeclineInvitation: app.NewDeclineWorkspaceInvitation(d.Accounts, store, d.Tx, d.Clock, d.InvitationMAC),
	}}
}

// Register mounts the module's API on router behind api's middlewares.
func (m *Module) Register(router *httpserver.Router, api *httpserver.API) {
	httpadapter.Register(router, api, m.uc)
}

// PublicOperations are the module's routes that need no token.
func (m *Module) PublicOperations() []string {
	return httpadapter.PublicOperations()
}

// SignupInvitations is the check of a registration's invitation, for
// identity's SignupPolicy (M3 design 6.6 step 6).
func (m *Module) SignupInvitations() SignupInvitations {
	return m.signup
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
