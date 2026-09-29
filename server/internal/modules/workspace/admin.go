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

// AdminDeps are what the server administrator's commands need: the pool and
// identity's Accounts, and no Authorizer, signing key or jobs client (M3
// design 6.6).
type AdminDeps struct {
	Pool     *pgxpool.Pool
	Tx       shared.TxManager
	Clock    app.Clock
	Logger   *slog.Logger
	Accounts app.Accounts
}

// Admin is the server administrator's use cases behind `nerve workspaces`
// (M3 design 3.11).
type Admin struct {
	create *app.CreateWorkspace
}

// NewAdmin wires the administrator's use cases on the pool alone: the
// command line builds no HTTP server, so it is the module's own minimal
// composition, not New's.
func NewAdmin(d AdminDeps) *Admin {
	return &Admin{create: app.NewCreateWorkspace(app.CreateWorkspaceDeps{
		Accounts: d.Accounts, Workspaces: postgresadapter.New(d.Pool), Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
	})}
}

// CreateWorkspace is `nerve workspaces create`: the workspace named name
// with slug, with the account of adminEmail as its admin, whatever
// workspace.creation_enabled says.
func (a *Admin) CreateWorkspace(ctx context.Context, slug, name, adminEmail string) (domain.Workspace, error) {
	return a.create.ExecuteForAdmin(ctx, adminEmail, domain.NewWorkspace{Name: name, Slug: slug})
}
