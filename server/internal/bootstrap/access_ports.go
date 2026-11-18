package bootstrap

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
)

// The reads project offers the access module, as access's ports (M3 design
// 6.5): each the same read, its facts converted.

// accessProjects is project's ProjectAccess as access's port: the same read,
// the facts converted.
type accessProjects struct {
	projects project.ProjectAccess
}

func (a accessProjects) ProjectFacts(ctx context.Context, projectID, userID uuid.UUID) (access.ProjectFacts, bool, error) {
	f, found, err := a.projects.ProjectFacts(ctx, projectID, userID)
	return access.ProjectFacts(f), found, err
}

// accessProjectsMany is project's ProjectAccessMany as access's port: the
// same read, the facts converted, the roles as they come.
type accessProjectsMany struct {
	projects project.ProjectAccessMany
}

func (a accessProjectsMany) ProjectFactsMany(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (access.ProjectFactsMany, bool, error) {
	f, found, err := a.projects.ProjectFactsMany(ctx, projectID, ids)
	return access.ProjectFactsMany(f), found, err
}

// accessMemberProjects is project's MemberProjectAccess as access's port:
// the same read, each project's facts converted.
type accessMemberProjects struct {
	projects project.MemberProjectAccess
}

func (a accessMemberProjects) MemberProjectFacts(ctx context.Context, workspaceID, userID uuid.UUID) ([]access.MemberProjectFacts, error) {
	facts, err := a.projects.MemberProjectFacts(ctx, workspaceID, userID)
	if err != nil {
		return nil, err
	}
	out := make([]access.MemberProjectFacts, len(facts))
	for i, f := range facts {
		out[i] = access.MemberProjectFacts(f)
	}
	return out, nil
}
