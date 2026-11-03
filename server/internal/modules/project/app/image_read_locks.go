package app

import (
	"context"
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ImageProjectSharer locks only the candidate projects, in ascending ID order,
// FOR SHARE, and returns the still-undeleted rows' workspace IDs.
type ImageProjectSharer interface {
	ShareImageProjects(context.Context, []uuid.UUID) (map[uuid.UUID]uuid.UUID, error)
}

// ImageReadLocks is the account -> workspace -> project boundary used only by
// project DTO reads. It does not alter the lock protocol of unrelated actions.
type ImageReadLocks struct {
	caller     CallerLock
	workspaces WorkspaceSharer
	projects   ImageProjectSharer
	clock      Clock
}

func NewImageReadLocks(caller CallerLock, workspaces WorkspaceSharer, projects ImageProjectSharer, clock Clock) ImageReadLocks {
	return ImageReadLocks{caller: caller, workspaces: workspaces, projects: projects, clock: clock}
}

func (l ImageReadLocks) lock(ctx context.Context, actor shared.Actor, workspaceID uuid.UUID, ids []uuid.UUID, notFound error) (map[uuid.UUID]uuid.UUID, error) {
	if err := l.caller.LockCaller(ctx, actor, l.clock.Now()); err != nil {
		return nil, err
	}
	w, found, err := l.workspaces.ShareWorkspaceByID(ctx, workspaceID)
	switch {
	case err != nil:
		return nil, err
	case !found:
		return nil, notFound
	case w.ID != workspaceID:
		return nil, fmt.Errorf("image workspace %s locked as %s", workspaceID, w.ID)
	}
	return l.projects.ShareImageProjects(ctx, ids)
}

func imageProjectIDs(projects []domain.Project) []uuid.UUID {
	ids := make([]uuid.UUID, len(projects))
	for i, p := range projects {
		ids[i] = p.ID
	}
	slices.SortFunc(ids, uuid.UUID.Compare)
	return slices.Compact(ids)
}
