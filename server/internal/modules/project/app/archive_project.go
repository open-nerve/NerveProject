package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ArchiveProject archives a project, or unarchives it: POST
// /api/v0/projects/{project_id}/archive and /unarchive (M3 design 3.4,
// 3.19).
type ArchiveProject struct {
	projects ProjectArchiver
	locks    Locks
	tx       shared.TxManager
	clock    Clock
	archive  bool
}

// NewArchiveProject returns the use case that archives.
func NewArchiveProject(projects ProjectArchiver, locks Locks, tx shared.TxManager, clock Clock) *ArchiveProject {
	return &ArchiveProject{projects: projects, locks: locks, tx: tx, clock: clock, archive: true}
}

// NewUnarchiveProject returns the use case that unarchives.
func NewUnarchiveProject(projects ProjectArchiver, locks Locks, tx shared.TxManager, clock Clock) *ArchiveProject {
	return &ArchiveProject{projects: projects, locks: locks, tx: tx, clock: clock}
}

// Execute, in one transaction (M3 design 3.6): the project's locks (Locks:
// its workspace FOR SHARE, then the project FOR NO KEY UPDATE) and the
// decision on project.archive or project.unarchive; archived_at set to the
// clock's time read under the locks, or cleared. An archived project
// archived again takes the new time, as Plane's does. The answer is the
// project as stored, as the caller sees it.
func (u *ArchiveProject) Execute(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Project{}, err
	}
	action := domain.ActionUnarchive
	if u.archive {
		action = domain.ActionArchive
	}
	var stored domain.Project
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := u.locks.lockAndDecide(ctx, actor, write{project: id, action: action}); err != nil {
			return err
		}
		if err := u.projects.SetArchived(ctx, id, u.archive, actor.UserID, u.clock.Now()); err != nil {
			return err
		}
		stored, err = answer(ctx, u.projects, id, actor.UserID)
		return err
	})
	if err != nil {
		return domain.Project{}, err
	}
	return stored, nil
}
