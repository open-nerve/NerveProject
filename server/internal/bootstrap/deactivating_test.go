package bootstrap

import (
	"context"
	"log/slog"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// deactivating is `nerve users deactivate` as bootstrap's Users wires it
// (users.go): identity's deactivation over its store, its memberships'
// step workspace's Deactivator, with project's cascade (project.NewCascade).
// sessions is identity's store, or one a gate stops; memberships the
// workspace store, or one a gate stops. Its commit and its rollback have 2s
// each; the caller's context bounds its waits.
func deactivating(pool *pgxpool.Pool, sessions identityapp.SessionRevoker, memberships workspaceapp.AllMembershipsEnder) *identityapp.Deactivate {
	store := identitypg.New(pool)
	return identityapp.NewDeactivate(identityapp.DeactivateDeps{
		Accounts: store, Users: store, Profiles: store, Sessions: sessions,
		Memberships: workspaceapp.NewDeactivator(memberships, project.NewCascade(project.CascadeDeps{Pool: pool}), clock.System{}),
		Tx:          postgres.NewTxManager(pool, 2*time.Second), Clock: clock.System{}, Logger: slog.New(slog.DiscardHandler),
	})
}

// deletedInvitationsTo stops a deactivation once it has deleted the
// invitations to the account's address, before those of the workspaces it
// leaves with no active member: it holds the account row, every workspace
// of his FOR NO KEY UPDATE, and every invitation the two deletes write,
// locked before either.
type deletedInvitationsTo struct {
	*workspacepg.Store
	gate *gate
}

func (m deletedInvitationsTo) DeleteInvitationsTo(ctx context.Context, email string, by uuid.UUID, now time.Time) error {
	if err := m.Store.DeleteInvitationsTo(ctx, email, by, now); err != nil {
		return err
	}
	return m.gate.wait(ctx)
}

// endedHoldingAll stops a deactivation once it has ended the account's
// workspace memberships, before the projects' step: it holds the account
// row, every workspace of his FOR NO KEY UPDATE, every invitation it
// deletes and his memberships' rows.
type endedHoldingAll struct {
	*workspacepg.Store
	gate *gate
}

func (m endedHoldingAll) EndWorkspaceMemberships(ctx context.Context, workspaceIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	if err := m.Store.EndWorkspaceMemberships(ctx, workspaceIDs, userID, by, now); err != nil {
		return err
	}
	return m.gate.wait(ctx)
}
