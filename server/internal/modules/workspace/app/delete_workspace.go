package app

import (
	"context"
	"log/slog"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteWorkspace deletes a workspace: DELETE /api/v0/workspaces/{slug}.
type DeleteWorkspace struct {
	workspaces WorkspaceDeleter
	projects   ProjectCascade
	auth       shared.Authorizer
	tx         shared.TxManager
	clock      Clock
	logger     *slog.Logger
}

// NewDeleteWorkspace returns the use case.
func NewDeleteWorkspace(workspaces WorkspaceDeleter, projects ProjectCascade, auth shared.Authorizer, tx shared.TxManager, clock Clock,
	logger *slog.Logger) *DeleteWorkspace {
	return &DeleteWorkspace{workspaces: workspaces, projects: projects, auth: auth, tx: tx, clock: clock, logger: logger}
}

// cascade is what deleting a workspace soft-deletes, in the order of M3
// design 3.6: the workspace row, then the rows under it, each step one
// statement at the same moment, and last the projects and the rows under
// them, through ProjectCascade (M3 design 3.3).
func (u *DeleteWorkspace) cascade() []func(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return []func(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error{
		u.workspaces.DeleteWorkspace,
		u.workspaces.DeleteWorkspaceInvitations,
		u.workspaces.DeleteWorkspaceMembers,
		u.workspaces.DeleteWorkspacePreferences,
		u.projects.DeleteWorkspaceProjects,
	}
}

// Execute deletes the workspace in one transaction (M3 design 3.6): the
// workspace row FOR NO KEY UPDATE, the decision on workspace.delete, then
// the cascade, every step at the one moment the clock gives under the lock.
// Nobody's last_workspace_id is cleared (M3 design 3.14).
func (u *DeleteWorkspace) Execute(ctx context.Context, slug string) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	var deleted uuid.UUID
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		id, _, err := lockAndDecide(ctx, u.workspaces.LockWorkspaceBySlug, u.auth, actor, slug, domain.ActionDelete)
		if err != nil {
			return err
		}
		now := u.clock.Now()
		for _, step := range u.cascade() {
			if err := step(ctx, id, actor.UserID, now); err != nil {
				return err
			}
		}
		deleted = id
		return nil
	})
	if err != nil {
		return err
	}
	u.logger.InfoContext(ctx, "workspace deleted", slog.String("workspace_id", deleted.String()), slog.String("user_id", actor.UserID.String()))
	return nil
}
