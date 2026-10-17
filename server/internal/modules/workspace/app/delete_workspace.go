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

// cascade soft-deletes, by by at now, in the order of M3 design 3.6: the
// workspace row, the invitations it locked, the memberships, the display
// settings, and last the projects and the rows under them, the work-item
// side of the projects it locked too, through ProjectCascade (M3 design
// 3.3, M4 design 3.5). A failing step comes back as itself and the steps
// after it do not run.
func (u *DeleteWorkspace) cascade(ctx context.Context, id uuid.UUID, invitations, projects []uuid.UUID, by uuid.UUID, now time.Time) error {
	for _, step := range []func() error{
		func() error { return u.workspaces.DeleteWorkspace(ctx, id, by, now) },
		func() error { return u.workspaces.DeleteInvitations(ctx, invitations, by, now) },
		func() error { return u.workspaces.DeleteWorkspaceMembers(ctx, id, by, now) },
		func() error { return u.workspaces.DeleteWorkspacePreferences(ctx, id, by, now) },
		func() error { return u.projects.DeleteWorkspaceProjects(ctx, id, projects, by, now) },
	} {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

// Execute deletes the workspace in one transaction (M3 design 3.6, M4
// design 3.5): the workspace row FOR NO KEY UPDATE, the decision on
// workspace.delete, then its invitations and then its projects FOR NO KEY
// UPDATE, each in id order, then the cascade, every step at the one moment
// the clock gives after those locks: a write that held one of the projects
// has committed by then, so the deletion's moment is never earlier than
// its. Nobody's last_workspace_id is cleared (M3 design 3.14).
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
		invitations, err := u.workspaces.LockWorkspaceInvitations(ctx, id)
		if err != nil {
			return err
		}
		projects, err := u.projects.LockWorkspaceProjects(ctx, id)
		if err != nil {
			return err
		}
		if err := u.cascade(ctx, id, invitations, projects, actor.UserID, u.clock.Now()); err != nil {
			return err
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
