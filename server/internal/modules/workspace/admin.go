package workspace

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AdminDeps are what the server administrator's commands need: the pool,
// identity's Accounts and project's ProjectMembershipCounts, and no
// Authorizer, signing key, project cascade or jobs client (M3 design 6.6).
type AdminDeps struct {
	Pool     *pgxpool.Pool
	Tx       shared.TxManager
	Clock    app.Clock
	Logger   *slog.Logger
	Accounts app.Accounts
	Counts   app.ProjectMembershipCounts
}

// Admin is the server administrator's use cases behind `nerve workspaces`
// (M3 design 3.11).
type Admin struct {
	create     *app.CreateWorkspace
	reactivate *app.ReactivateMember
}

// NewAdmin wires the administrator's use cases on the pool alone: the
// command line builds no HTTP server, so it is the module's own minimal
// composition, not New's.
func NewAdmin(d AdminDeps) *Admin {
	store := postgresadapter.New(d.Pool)
	return &Admin{
		create: app.NewCreateWorkspace(app.CreateWorkspaceDeps{
			Accounts: d.Accounts, Workspaces: store, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
		}),
		reactivate: app.NewReactivateMember(d.Accounts, store, d.Counts, d.Tx, d.Clock, d.Logger),
	}
}

// CreateWorkspace is `nerve workspaces create`: the workspace named name
// with slug, with the account of adminEmail as its admin, whatever
// workspace.creation_enabled says.
func (a *Admin) CreateWorkspace(ctx context.Context, slug, name, adminEmail string) (domain.Workspace, error) {
	return a.create.ExecuteForAdmin(ctx, adminEmail, domain.NewWorkspace{Name: name, Slug: slug})
}

// ReactivateMember is `nerve workspaces reactivate-member`: the ended
// membership of the account of email in the workspace slug active again,
// its role kept, also while the account is deactivated.
func (a *Admin) ReactivateMember(ctx context.Context, slug, email string) (app.Reactivation, error) {
	return a.reactivate.Execute(ctx, slug, email)
}
