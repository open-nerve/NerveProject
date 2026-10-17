package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// GreatestSortOrder reads the greatest sort order of the project's
// undeleted work items in the state, archived ones too, nil when it has
// none (M4 design 3.7).
func (s *Store) GreatestSortOrder(ctx context.Context, projectID, stateID uuid.UUID) (*float64, error) {
	greatest, err := s.queries(ctx).GreatestSortOrder(ctx, gen.GreatestSortOrderParams{ProjectID: projectID, StateID: stateID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("read the greatest sort order of state %s: %w", stateID, err)
	}
	return &greatest, nil
}

// InsertIssue inserts the work item r. The use case checked every value
// and took its number under the project's lock, so any violation is a bug:
// an internal error (500), not a domain error.
func (s *Store) InsertIssue(ctx context.Context, r app.IssueRow) error {
	err := s.queries(ctx).InsertIssue(ctx, gen.InsertIssueParams{
		ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, StateID: r.StateID, Name: r.Name, Priority: string(r.Priority),
		SequenceID: int32(r.SequenceID), SortOrder: r.SortOrder, CompletedAt: r.CompletedAt, By: &r.By, At: r.At,
	})
	if err != nil {
		return fmt.Errorf("insert work item %s: %w", r.ID, err)
	}
	return nil
}

// InsertAssignees inserts the work item's assignees' rows r.
func (s *Store) InsertAssignees(ctx context.Context, r app.LinkRows) error {
	err := s.queries(ctx).InsertAssignees(ctx, gen.InsertAssigneesParams{
		Ids: r.IDs, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, AssigneeIds: r.Targets, By: &r.By, At: r.At,
	})
	return linked(err, "assignees", r.IssueID)
}

// InsertLabels inserts the work item's labels' rows r.
func (s *Store) InsertLabels(ctx context.Context, r app.LinkRows) error {
	err := s.queries(ctx).InsertLabels(ctx, gen.InsertLabelsParams{
		Ids: r.IDs, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, LabelIds: r.Targets, By: &r.By, At: r.At,
	})
	return linked(err, "labels", r.IssueID)
}

// InsertSubscribers inserts the work item's subscribers' rows r, but of an
// account that subscribes to it already or that r names before (M4 design
// 3.12).
func (s *Store) InsertSubscribers(ctx context.Context, r app.LinkRows) error {
	err := s.queries(ctx).InsertSubscribers(ctx, gen.InsertSubscribersParams{
		Ids: r.IDs, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, SubscriberIds: r.Targets, By: &r.By, At: r.At,
	})
	return linked(err, "subscribers", r.IssueID)
}

// linked is err, an insert of the work item issue's rows of table, as the
// insert's error; nil is nil.
func linked(err error, table string, issue uuid.UUID) error {
	if err != nil {
		return fmt.Errorf("insert the %s of work item %s: %w", table, issue, err)
	}
	return nil
}

// InsertActivities inserts rows, a statement each, in their order: a
// creation writes one more than it has assignees (M4 design 3.13).
func (s *Store) InsertActivities(ctx context.Context, rows []app.ActivityRow) error {
	for _, r := range rows {
		a := r.Activity
		err := s.queries(ctx).InsertActivity(ctx, gen.InsertActivityParams{
			ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, Verb: string(a.Verb), Field: a.Field,
			OldValue: a.OldValue, NewValue: a.NewValue, OldIdentifier: a.OldIdentifier, NewIdentifier: a.NewIdentifier, Comment: a.Comment,
			ActorID: &r.ActorID, At: r.At,
		})
		if err != nil {
			return fmt.Errorf("insert activity of work item %s: %w", r.IssueID, err)
		}
	}
	return nil
}

// GetIssue reads the undeleted work item id, archived or not, with its
// assignees and labels (M4 design 5.2); found is false when there is none.
func (s *Store) GetIssue(ctx context.Context, id uuid.UUID) (domain.Issue, bool, error) {
	r, err := s.queries(ctx).GetIssue(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Issue{}, false, nil
	case err != nil:
		return domain.Issue{}, false, fmt.Errorf("read work item %s: %w", id, err)
	}
	return domain.Issue{
		ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, SequenceID: int(r.SequenceID), Name: r.Name, StateID: r.StateID,
		Priority: domain.Priority(r.Priority), ParentID: r.ParentID, SortOrder: r.SortOrder, CompletedAt: r.CompletedAt,
		ArchivedAt: r.ArchivedAt, AssigneeIDs: r.AssigneeIds, LabelIDs: r.LabelIds, CreatedByID: r.CreatedByID,
		UpdatedByID: r.UpdatedByID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, true, nil
}
