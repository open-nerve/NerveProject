package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteProject deletes a project: DELETE /api/v0/projects/{project_id}
// (M3 design 3.3, 3.6).
type DeleteProject struct {
	projects ProjectsDeleter
	events   shared.EventPublisher
	locks    Locks
	tx       shared.TxManager
	clock    Clock
}

// NewDeleteProject returns the use case.
func NewDeleteProject(projects ProjectsDeleter, events shared.EventPublisher, locks Locks, tx shared.TxManager, clock Clock) *DeleteProject {
	return &DeleteProject{projects: projects, events: events, locks: locks, tx: tx, clock: clock}
}

// Execute locks and decides once, then soft-deletes the M3 project rows and
// publishes the issue-row deletion task in the same transaction. All rows use
// the caller and the one clock reading taken after the locks.
func (u *DeleteProject) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.locks.lockAndDecide(ctx, actor, write{project: id, action: domain.ActionDelete})
		if err != nil {
			return err
		}
		now := u.clock.Now()
		if err := deleteProjects(ctx, u.projects, Deletion{WorkspaceID: h.project.WorkspaceID, ProjectID: &id, By: actor.UserID, Now: now}); err != nil {
			return err
		}
		return u.events.Publish(ctx, shared.Event{
			ID: uuid.NewV7(), Type: "project.deleted", Entity: shared.EntityRef{Kind: "project", ID: id},
			WorkspaceID: h.project.WorkspaceID, ProjectID: id, ActorID: &actor.UserID, OccurredAt: now,
		})
	})
}
