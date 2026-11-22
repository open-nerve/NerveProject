package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// NavigationIssuePlaces reads, in one statement and without a lock, where
// the undeleted work items among ids are, archived or not, of any
// workspace, by id (navigation_issues.sql; M7 P3d spec 2.7), in the
// transaction ctx carries or on the pool. A deleted work item and an id of
// none are not among them.
func (s *Store) NavigationIssuePlaces(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]app.NavigationIssuePlace, error) {
	rows, err := s.queries(ctx).NavigationIssuePlaces(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("read where %d work items of recent visits are: %w", len(ids), err)
	}
	places := make(map[uuid.UUID]app.NavigationIssuePlace, len(rows))
	for _, r := range rows {
		places[r.ID] = app.NavigationIssuePlace{WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID}
	}
	return places, nil
}

// NavigationIssueRows reads, in one statement and without a lock, the
// undeleted work items among ids, archived or not, of any project, in id
// order, each with what a recent visit is answered with, its creator and
// its undeleted intake row's status (navigation_issues.sql; spec 2.4, 2.5),
// in the transaction ctx carries or on the pool.
func (s *Store) NavigationIssueRows(ctx context.Context, ids []uuid.UUID) ([]app.NavigationIssueRow, error) {
	rows, err := s.queries(ctx).NavigationIssueRows(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("read %d work items of recent visits: %w", len(ids), err)
	}
	out := make([]app.NavigationIssueRow, len(rows))
	for n, r := range rows {
		out[n] = app.NavigationIssueRow{ID: r.ID, ProjectID: r.ProjectID, SequenceID: int(r.SequenceID), Name: r.Name, StateID: r.StateID,
			Priority: domain.Priority(r.Priority), AssigneeIDs: r.AssigneeIds, CreatedByID: r.CreatedByID, Intake: intakeOf(r.IntakeStatus, nil, nil)}
	}
	return out, nil
}
