package postgresadapter

import (
	"context"
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
)

// MarkReadBatch marks a batch of mark-all-read read in the transaction ctx
// carries and answers how many rows it marked and the greatest of their
// ids, which RETURNING gives in no order (app.ReadMarks, mark_read.sql).
func (s *Store) MarkReadBatch(ctx context.Context, b app.ReadBatch) (app.MarkedBatch, error) {
	if err := checkLimit("mark-all-read batch", b.Limit); err != nil {
		return app.MarkedBatch{}, err
	}
	ids, own := grantArrays(b.Grants)
	marked, err := s.queries(ctx).MarkNotificationsRead(ctx, gen.MarkNotificationsReadParams{Now: b.Now, ReceiverID: b.ReceiverID, WorkspaceID: b.WorkspaceID,
		ProjectIds: ids, ProjectOwn: own, Senders: senderTexts(b.Filter.Senders), Archived: b.Filter.Archived, Snoozed: b.Filter.Snoozed, After: b.After,
		Batch: int32(b.Limit)})
	if err != nil {
		return app.MarkedBatch{}, fmt.Errorf("mark a batch of the notifications of %s in workspace %s read: %w", b.ReceiverID, b.WorkspaceID, err)
	}
	if len(marked) == 0 {
		return app.MarkedBatch{Last: b.After}, nil
	}
	return app.MarkedBatch{Marked: len(marked), Last: slices.MaxFunc(marked, uuid.UUID.Compare)}, nil
}
