package postgresadapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
)

// InsertNotifications inserts a fan-out's notifications in one statement,
// in the transaction ctx carries, each snapshot as its JSON; one of the
// same event, change and receiver as an undeleted one is not inserted
// (app.NotificationWriter, notifications.sql).
func (s *Store) InsertNotifications(ctx context.Context, n app.Notifications) error {
	p := gen.InsertNotificationsParams{WorkspaceID: n.WorkspaceID, ProjectID: n.ProjectID, TriggeredByID: n.TriggeredByID, IssueID: n.IssueID,
		IssueCreatedByID: n.IssueCreatedByID, EventID: n.EventID, At: n.At}
	for _, r := range n.Rows {
		data, err := json.Marshal(r.Data)
		if err != nil {
			return fmt.Errorf("encode the snapshot of notification %s: %w", r.ID, err)
		}
		p.Ids, p.ReceiverIds = append(p.Ids, r.ID), append(p.ReceiverIds, r.ReceiverID)
		p.Senders, p.Data, p.ChangeSeqs = append(p.Senders, string(r.Sender)), append(p.Data, data), append(p.ChangeSeqs, r.ChangeSeq)
	}
	if err := s.queries(ctx).InsertNotifications(ctx, p); err != nil {
		return fmt.Errorf("insert the notifications of work item %s: %w", n.IssueID, err)
	}
	return nil
}
