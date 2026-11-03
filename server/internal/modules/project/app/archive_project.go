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
type ArchiveProjectDeps struct {
	Projects ProjectArchiver
	Locks    Locks
	Caller   CallerLock
	Images   ImageReader
	Tx       shared.TxManager
	Clock    Clock
}

type ArchiveProject struct {
	d       ArchiveProjectDeps
	archive bool
}

func NewArchiveProject(d ArchiveProjectDeps) *ArchiveProject {
	return &ArchiveProject{d: d, archive: true}
}
func NewUnarchiveProject(d ArchiveProjectDeps) *ArchiveProject { return &ArchiveProject{d: d} }

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
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := u.d.Caller.LockCaller(ctx, actor, u.d.Clock.Now()); err != nil {
			return err
		}
		if _, err := u.d.Locks.lockAndDecide(ctx, actor, write{project: id, action: action}); err != nil {
			return err
		}
		if err := u.d.Projects.SetArchived(ctx, id, u.archive, actor.UserID, u.d.Clock.Now()); err != nil {
			return err
		}
		stored, err = answer(ctx, u.d.Projects, id, actor.UserID)
		if err != nil {
			return err
		}
		images, err := withProjectImages(ctx, u.d.Images, []domain.Project{stored})
		if err != nil {
			return err
		}
		stored = images[0]
		return nil
	})
	if err != nil {
		return domain.Project{}, err
	}
	return stored, nil
}
