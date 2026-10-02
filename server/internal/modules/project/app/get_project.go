package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetProject reads a project: GET /api/v0/projects/{project_id} (M3 design
// 3.19).
type GetProject struct {
	projects ProjectReader
	auth     shared.Authorizer
}

// NewGetProject returns the use case.
func NewGetProject(projects ProjectReader, auth shared.Authorizer) *GetProject {
	return &GetProject{projects: projects, auth: auth}
}

// Execute answers the undeleted project id, archived or not, as the caller
// sees it, when he may read it. It reads the project first, which names its
// workspace, then decides project.read on it; a project that is not there
// and one the caller does not see answer the same domain.ErrNotFound (M3
// design 8.2). A read opens no transaction (M3 design 6.7).
func (u *GetProject) Execute(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Project{}, err
	}
	p, found, err := u.projects.GetProject(ctx, id, actor.UserID)
	switch {
	case err != nil:
		return domain.Project{}, err
	case !found:
		return domain.Project{}, domain.ErrNotFound
	}
	if _, err = decide(ctx, u.auth, actor, domain.ActionRead, p.WorkspaceID, p.ID); err != nil {
		return domain.Project{}, err
	}
	return p, nil
}
