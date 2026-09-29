package bootstrap

import (
	"context"
	"io"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/logging"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// WorkspaceCommand is one `nerve workspaces` command on the administrator's
// use cases: it runs and returns the line it prints.
type WorkspaceCommand func(ctx context.Context, admin *workspace.Admin) (string, error)

// Workspaces runs cmd on the minimal composition of M3 design 6.6: a pool,
// identity's Accounts and workspace's administrator use cases; no HTTP
// server, no Authorizer, no signing key, no jobs client. The command's line
// goes to out, the logs to logOut. An error is one line for the
// administrator, and the database is unchanged.
func Workspaces(ctx context.Context, cfg config.Config, logOut, out io.Writer, cmd WorkspaceCommand) error {
	logger, err := logging.New(logOut, cfg.Log)
	if err != nil {
		return err
	}
	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()
	admin := workspace.NewAdmin(workspace.AdminDeps{
		Pool:     pool,
		Tx:       postgres.NewTxManager(pool, cfg.Database.CommitTimeout),
		Clock:    clock.System{},
		Logger:   logger,
		Accounts: workspaceAccounts{accounts: identity.Provide(pool).Accounts},
	})
	line, err := cmd(ctx, admin)
	if err != nil {
		return commandError(err)
	}
	return writeLine(out, line)
}

// CreateWorkspace is `nerve workspaces create` (M3 design 3.11).
func CreateWorkspace(slug, name, adminEmail string) WorkspaceCommand {
	return func(ctx context.Context, admin *workspace.Admin) (string, error) {
		w, err := admin.CreateWorkspace(ctx, slug, name, adminEmail)
		return "created workspace " + w.Slug + " with admin " + shared.NormalizeEmail(adminEmail), err
	}
}
