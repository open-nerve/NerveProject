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
	locks    Locks
	tx       shared.TxManager
	clock    Clock
}

// NewDeleteProject returns the use case.
func NewDeleteProject(projects ProjectsDeleter, locks Locks, tx shared.TxManager, clock Clock) *DeleteProject {
	return &DeleteProject{projects: projects, locks: locks, tx: tx, clock: clock}
}

// Execute, in one transaction (M3 design 3.6): the project's locks (Locks:
// its workspace FOR SHARE, then the project FOR NO KEY UPDATE) and the
// decision on project.delete; the clock read under the locks; then the
// project and the rows under it soft-deleted, each at that one moment, by
// the caller (deleteProjects). An archived project is deleted as any
// other.
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
		return deleteProjects(ctx, u.projects, Deletion{WorkspaceID: h.project.WorkspaceID, ProjectID: &id, By: actor.UserID, Now: u.clock.Now()})
	})
}
