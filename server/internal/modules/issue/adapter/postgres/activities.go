package postgresadapter

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Activities filters both historical reference endpoints before keyset pagination.
func (s *Store) Activities(ctx context.Context, issueID, actorID uuid.UUID, projects []app.VisibleProject, after *domain.ActivityCursor, limit int) ([]app.ActivityRecord, error) {
	p := gen.ActivitiesParams{IssueID: issueID, ActorID: actorID, PageLimit: int32(limit), ProjectIds: []uuid.UUID{}, OwnOnlyProjectIds: []uuid.UUID{}}
	for _, project := range projects {
		if project.OwnOnly {
			p.OwnOnlyProjectIds = append(p.OwnOnlyProjectIds, project.ID)
		} else {
			p.ProjectIds = append(p.ProjectIds, project.ID)
		}
	}
	if after != nil {
		p.HasAfter = true
		p.AfterAt = after.CreatedAt
		p.AfterID = after.ID
	}
	rows, err := s.queries(ctx).Activities(ctx, p)
	if err != nil {
		return nil, err
	}
	result := make([]app.ActivityRecord, len(rows))
	for n, r := range rows {
		result[n] = app.ActivityRecord{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, ActorID: r.ActorID, Activity: domain.Activity{Field: r.Field, Verb: domain.Verb(r.Verb), OldValue: r.OldValue, NewValue: r.NewValue, OldIdentifier: r.OldIdentifier, NewIdentifier: r.NewIdentifier, Comment: r.Comment}, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
	}
	return result, nil
}
