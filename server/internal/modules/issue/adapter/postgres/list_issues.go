package postgresadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Flat reads a projected page and its lookahead in the caller's transaction.
// Each total counts all matched issues before the cursor; child counts already
// follow the caller's project scope and must not be recomputed by the caller.
func (s *Store) Flat(ctx context.Context, in app.ListRead) ([]app.ListRecord, error) {
	params, err := flatParameters(in)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries(ctx).FlatIssues(ctx, params)
	if err != nil {
		return nil, err
	}
	out := make([]app.ListRecord, len(rows))
	for n, r := range rows {
		position, err := listPosition(r, in.Query.Order)
		if err != nil {
			return nil, err
		}
		out[n] = app.ListRecord{Total: r.Total, Position: position, Issue: domain.Issue{
			ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, SequenceID: int(r.SequenceID), Name: r.Name, StateID: r.StateID,
			Priority: domain.Priority(r.Priority), StartDate: dayOf(r.StartDate), TargetDate: dayOf(r.TargetDate), ParentID: r.ParentID,
			SortOrder: r.SortOrder, CompletedAt: r.CompletedAt, ArchivedAt: r.ArchivedAt, CreatedByID: r.CreatedByID, UpdatedByID: r.UpdatedByID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
			AssigneeIDs: r.AssigneeIds, LabelIDs: r.LabelIds, SubIssuesCount: int(r.SubIssuesCount),
		}}
	}
	return out, nil
}
