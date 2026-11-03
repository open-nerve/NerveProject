package postgresadapter

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func (s *Store) RecordAttachmentChange(ctx context.Context, row app.ActivityRow) error {
	if _, ok := postgres.Tx(ctx); !ok {
		return errors.New("attachment change requires a transaction")
	}
	item, found, err := s.GetIssue(ctx, row.IssueID)
	if err != nil {
		return err
	}
	if !found || item.WorkspaceID != row.WorkspaceID || item.ProjectID != row.ProjectID || row.ActorID == nil {
		return errors.New("attachment change does not match its issue")
	}
	if err := s.InsertActivities(ctx, []app.ActivityRow{row}); err != nil {
		return err
	}
	return s.MarkChanged(ctx, app.ChangedRows{IDs: []uuid.UUID{row.IssueID}, By: *row.ActorID, At: row.At})
}
