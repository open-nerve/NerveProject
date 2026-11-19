package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// CountPendingIntake counts the pending intake work items of each of the
// projects as actor sees them, at now (M7 P2b spec 2.6), in the read
// transaction ctx carries; see app.IntakeCounter.
func (s *Store) CountPendingIntake(ctx context.Context, actor uuid.UUID, projects []app.VisibleProject, now time.Time) ([]app.IntakeCount, error) {
	p := gen.CountPendingIntakeIssuesParams{ProjectIds: make([]uuid.UUID, len(projects)), ProjectOwn: make([]bool, len(projects)), Now: now, ActorID: actor}
	for n, project := range projects {
		p.ProjectIds[n], p.ProjectOwn[n] = project.ID, project.OwnOnly
	}
	rows, err := s.queries(ctx).CountPendingIntakeIssues(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("count the pending intake work items of %d projects: %w", len(projects), err)
	}
	out := make([]app.IntakeCount, len(rows))
	for n, r := range rows {
		out[n] = app.IntakeCount{ProjectID: r.ProjectID, Pending: r.Pending}
	}
	return out, nil
}
