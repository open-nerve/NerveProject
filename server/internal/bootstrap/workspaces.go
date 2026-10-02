package bootstrap

import (
	"context"
	"fmt"
	"io"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
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
// identity's Accounts, project's ProjectMembershipCounts and workspace's
// administrator use cases; no HTTP server, no Authorizer, no signing key,
// no project cascade, no jobs client. The command's line goes to out, the
// logs to logOut. An error is one line for the administrator, and the
// database is unchanged.
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
		Counts:   project.Provide(pool).ProjectMembershipCounts,
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

// roleNames are the workspace roles as the commands print them.
var roleNames = map[shared.Role]string{shared.RoleAdmin: "admin", shared.RoleMember: "member", shared.RoleGuest: "guest"}

// ReactivateMember is `nerve workspaces reactivate-member` (M3 design 3.11,
// as Plane's reactivate_workspace_member): the line says the role kept and
// how many of the member's project memberships stay ended, or that the
// membership was active already; for a deactivated account, that `nerve
// users activate` is next.
func ReactivateMember(slug, email string) WorkspaceCommand {
	return func(ctx context.Context, admin *workspace.Admin) (string, error) {
		r, err := admin.ReactivateMember(ctx, slug, email)
		line := fmt.Sprintf("reactivated %s in %s as %s; project memberships still ended: %d, each restored when the member joins or is added "+
			"to its project", r.Email, slug, roleNames[r.Role], r.EndedProjectMemberships)
		if r.AlreadyActive {
			line = fmt.Sprintf("%s is an active member of %s already; nothing changed", r.Email, slug)
		}
		if !r.AccountActive {
			line += "; the account is deactivated: run nerve users activate --email " + r.Email + " next"
		}
		return line, err
	}
}
