package postgresadapter

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// HistoricalIssueProjects reads active historical memberships in the caller's
// transaction, retaining soft-deleted and archived projects and memberships.
func (s *Store) HistoricalIssueProjects(ctx context.Context, workspaceID, actorID uuid.UUID) ([]app.HistoricalIssueProject, error) {
	rows, err := s.queries(ctx).HistoricalIssueProjects(ctx, gen.HistoricalIssueProjectsParams{WorkspaceID: workspaceID, ActorID: actorID})
	if err != nil {
		return nil, err
	}
	out := make([]app.HistoricalIssueProject, len(rows))
	for i, row := range rows {
		out[i] = app.HistoricalIssueProject{ID: row.ID, Role: shared.Role(row.Role), OwnOnly: row.OwnOnly}
	}
	return out, nil
}
