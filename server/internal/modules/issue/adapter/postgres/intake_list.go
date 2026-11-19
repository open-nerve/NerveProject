package postgresadapter

import (
	"context"
	"fmt"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListIntake reads a page of the project's intake list and the row after
// it (M7 P2b spec 2.3), in the read transaction ctx carries; see
// app.IntakeLister. A query the domain would not have parsed is an error.
func (s *Store) ListIntake(ctx context.Context, in app.IntakeListRead) ([]app.IntakeRecord, error) {
	q, f := in.Query, in.Query.Filters
	if !domain.IntakeOrderAllowed(q.Order) || q.Limit < 1 || q.Limit > shared.MaxPageSize || (q.Cursor != nil && q.Cursor.Order != q.Order) {
		return nil, fmt.Errorf("invalid intake list query")
	}
	p := gen.ListIntakeIssuesParams{
		ProjectID: in.ProjectID, ActorID: in.ActorID, OwnOnly: in.OwnOnly, Now: in.Now,
		Statuses: make([]string, len(f.Statuses)), Priorities: make([]string, len(f.Priorities)), Creators: listArray(f.CreatorIDs),
		Labels: listArray(f.Labels.IDs), LabelsNone: f.Labels.None, Assignees: listArray(f.Assignees.IDs), AssigneesNone: f.Assignees.None,
		CreatedFrom: f.Created.From, CreatedTo: f.Created.To, UpdatedFrom: f.Updated.From, UpdatedTo: f.Updated.To,
		OrderKey: q.Order.Key(), Descending: q.Order.Descending(), PageSize: int32(q.Limit + 1),
	}
	for n, status := range f.Statuses {
		p.Statuses[n] = string(status)
	}
	for n, priority := range f.Priorities {
		p.Priorities[n] = string(priority)
	}
	if c := q.Cursor; c != nil {
		key := c.Seq
		if q.Order.Key() != "sequence_id" {
			key = c.At.UnixMicro()
		}
		p.HasCursor, p.CursorKey, p.CursorID = true, &key, &c.ID
	}
	rows, err := s.queries(ctx).ListIntakeIssues(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("list the intake of project %s: %w", in.ProjectID, err)
	}
	out := make([]app.IntakeRecord, len(rows))
	for n, r := range rows {
		out[n] = app.IntakeRecord{Total: r.Total, Position: intakePosition(r, q.Order), Issue: domain.Issue{
			ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, SequenceID: int(r.SequenceID), Name: r.Name, StateID: r.StateID,
			Priority: domain.Priority(r.Priority), StartDate: dayOf(r.StartDate), TargetDate: dayOf(r.TargetDate), ParentID: r.ParentID,
			SortOrder: r.SortOrder, CompletedAt: r.CompletedAt, ArchivedAt: r.ArchivedAt, CreatedByID: r.CreatedByID, UpdatedByID: r.UpdatedByID,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, AssigneeIDs: r.AssigneeIds, LabelIDs: r.LabelIds, CycleID: r.CycleID, ModuleIDs: r.ModuleIds,
			SubIssuesCount: int(r.SubIssuesCount), LinkCount: r.LinkCount,
			Intake: intakeOf(&r.IntakeStatus, r.IntakeSnoozedTill, r.IntakeDuplicateToID),
		}}
	}
	return out, nil
}

// intakePosition is the row r's position in the order: its key, the work
// item's time or its number, and its id.
func intakePosition(r gen.ListIntakeIssuesRow, order domain.IntakeOrder) domain.IntakeCursor {
	c := domain.IntakeCursor{Order: order, ID: r.ID}
	switch order.Key() {
	case "created_at":
		c.At = r.CreatedAt
	case "updated_at":
		c.At = r.UpdatedAt
	default:
		c.Seq = int64(r.SequenceID)
	}
	return c
}
