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
	events     shared.EventPublisher
	logger     *slog.Logger
}

// NewDeleteWorkspace returns the use case.
func NewDeleteWorkspace(workspaces WorkspaceDeleter, projects ProjectCascade, auth shared.Authorizer, tx shared.TxManager, clock Clock, events shared.EventPublisher,
	logger *slog.Logger) *DeleteWorkspace {
	return &DeleteWorkspace{workspaces: workspaces, projects: projects, auth: auth, tx: tx, clock: clock, events: events, logger: logger}
}

// cascade writes the synchronous M3 rows at the request moment.
func (u *DeleteWorkspace) cascade(ctx context.Context, id uuid.UUID, invitations []uuid.UUID, by uuid.UUID, now time.Time) error {
	for _, step := range []func() error{
		func() error { return u.workspaces.DeleteWorkspace(ctx, id, by, now) },
		func() error { return u.workspaces.DeleteInvitations(ctx, invitations, by, now) },
		func() error { return u.workspaces.DeleteWorkspaceMembers(ctx, id, by, now) },
		func() error { return u.workspaces.DeleteWorkspacePreferences(ctx, id, by, now) },
		func() error { return u.projects.DeleteWorkspaceProjects(ctx, id, by, now) },
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
		now := u.clock.Now()
		if err := u.cascade(ctx, id, invitations, actor.UserID, now); err != nil {
			return err
		}
		changes := make([]shared.Change, 0, len(projects))
		for _, project := range projects {
			changes = append(changes, shared.Change{Field: "project", OldID: &project})
		}
		if err := u.events.Publish(ctx, shared.Event{ID: uuid.NewV7(), Type: "workspace.deleted", Entity: shared.EntityRef{Kind: "workspace", ID: id}, WorkspaceID: id, ActorID: &actor.UserID, OccurredAt: now, Changes: changes}); err != nil {
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
