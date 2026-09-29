package app

import (
	"context"
	"log/slog"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateWorkspaceDeps are CreateWorkspace's collaborators.
type CreateWorkspaceDeps struct {
	Accounts   Accounts
	Workspaces WorkspaceCreator
	Tx         shared.TxManager
	Clock      Clock
	Logger     *slog.Logger
	// Enabled is workspace.creation_enabled: whether callers of the API may
	// create workspaces. The server's administrator always may (M3 design
	// 3.11).
	Enabled bool
}

// CreateWorkspace creates a workspace with its first admin (M3 design 3.11):
// the caller of POST /api/v0/workspaces, or the account the server's
// administrator names to `nerve workspaces create`. One use case, two
// entries, the same rules, uniqueness and admin membership.
type CreateWorkspace struct {
	d CreateWorkspaceDeps
}

// NewCreateWorkspace returns the use case.
func NewCreateWorkspace(d CreateWorkspaceDeps) *CreateWorkspace {
	return &CreateWorkspace{d: d}
}

// Execute creates w with the caller as its admin: workspace.creation_disabled
// while creation is off, before anything else (Plane views/workspace/
// base.py:83-96); 422 validation_failed; 401 unauthorized when the account,
// read under its lock, is no longer active (M3 design 3.6 convention 6);
// workspace.slug_taken.
func (u *CreateWorkspace) Execute(ctx context.Context, w domain.NewWorkspace) (domain.Workspace, error) {
	if !u.d.Enabled {
		return domain.Workspace{}, domain.ErrCreationDisabled
	}
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Workspace{}, err
	}
	return u.create(ctx, w, byAPI, func(ctx context.Context) (AccountState, error) {
		state, found, err := u.d.Accounts.ShareAccount(ctx, actor.UserID)
		switch {
		case err != nil:
			return AccountState{}, err
		case !found || !state.Active:
			return AccountState{}, shared.Unauthenticated()
		}
		return state, nil
	})
}

// ExecuteForAdmin creates w with the account of email as its admin, for the
// server's administrator: workspace.creation_enabled does not apply. The
// address is normalized as registration does it; no such account is
// workspace.account_not_found, a deactivated one, read under its lock,
// workspace.account_deactivated.
func (u *CreateWorkspace) ExecuteForAdmin(ctx context.Context, email string, w domain.NewWorkspace) (domain.Workspace, error) {
	email = shared.NormalizeEmail(email)
	return u.create(ctx, w, byCLI, func(ctx context.Context) (AccountState, error) {
		state, found, err := u.d.Accounts.ShareAccountByEmail(ctx, email)
		switch {
		case err != nil:
			return AccountState{}, err
		case !found:
			return AccountState{}, domain.ErrAccountNotFound
		case !state.Active:
			return AccountState{}, domain.ErrAccountDeactivated
		}
		return state, nil
	})
}

// The values of "by" in the logs: who asked.
const (
	byAPI = "api"
	byCLI = "cli"
)

// create checks w, then writes in one transaction in the order of M3 design
// 3.6: the admin's account row first, locked FOR SHARE by lock, which also
// decides on the account's state as read under the lock; then the workspace
// and its admin membership. Creating a workspace enqueues nothing: no demo
// data (M3 design 3.11).
func (u *CreateWorkspace) create(ctx context.Context, w domain.NewWorkspace, by string,
	lock func(ctx context.Context) (AccountState, error)) (domain.Workspace, error) {
	if err := domain.CheckNewWorkspace(w); err != nil {
		return domain.Workspace{}, err
	}
	zone := domain.DefaultTimezone
	if w.Timezone != nil {
		zone = *w.Timezone
	}
	now := u.d.Clock.Now()
	var admin AccountState
	var created domain.Workspace
	err := u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if admin, err = lock(ctx); err != nil {
			return err
		}
		created, err = u.d.Workspaces.CreateWorkspace(ctx, WorkspaceRow{
			ID: uuid.NewV7(), Name: w.Name, Slug: w.Slug, OrganizationSize: w.OrganizationSize, Timezone: zone,
			CreatedBy: admin.ID, Now: now,
		})
		if err != nil {
			return err
		}
		return u.d.Workspaces.CreateMember(ctx, MemberRow{
			ID: uuid.NewV7(), WorkspaceID: created.ID, MemberID: admin.ID, Role: shared.RoleAdmin, CreatedBy: admin.ID, Now: now,
		})
	})
	if err != nil {
		return domain.Workspace{}, err
	}
	created.Role, created.TotalMembers = shared.RoleAdmin, 1
	u.d.Logger.InfoContext(ctx, "workspace created", slog.String("workspace_id", created.ID.String()),
		slog.String("user_id", admin.ID.String()), slog.String("by", by))
	return created, nil
}
