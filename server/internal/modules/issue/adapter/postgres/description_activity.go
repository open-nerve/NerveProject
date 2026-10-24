package postgresadapter

import (
	"context"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// LatestActivity reads the latest live activity of any field under the work item.
func (s *Store) LatestActivity(ctx context.Context, issueID uuid.UUID) (app.ActivityRecord, bool, error) {
	r, err := s.queries(ctx).LatestActivity(ctx, issueID)
	if errors.Is(err, pgx.ErrNoRows) {
		return app.ActivityRecord{}, false, nil
	}
	if err != nil {
		return app.ActivityRecord{}, false, err
	}
	return app.ActivityRecord{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, ActorID: r.ActorID,
		Activity: domain.Activity{Field: r.Field, Verb: domain.Verb(r.Verb), OldValue: r.OldValue, NewValue: r.NewValue,
			OldIdentifier: r.OldIdentifier, NewIdentifier: r.NewIdentifier, Comment: r.Comment}, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}, true, nil
}

// MergeDescriptionActivity advances the content and audit time of an existing description row.
func (s *Store) MergeDescriptionActivity(ctx context.Context, m app.DescriptionActivityMerge) error {
	return s.queries(ctx).MergeDescriptionActivity(ctx, gen.MergeDescriptionActivityParams{ID: m.ID, By: m.By, NewHtml: &m.NewHTML, At: m.At})
}
