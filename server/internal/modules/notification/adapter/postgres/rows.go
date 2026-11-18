package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
)

// GetNotification reads a receiver's undeleted notification without a
// lock (app.NotificationRows, rows.sql).
func (s *Store) GetNotification(ctx context.Context, id, receiverID uuid.UUID) (app.StoredNotification, bool, error) {
	r, err := s.queries(ctx).GetNotification(ctx, gen.GetNotificationParams{ID: id, ReceiverID: receiverID})
	return storedOf("read", id, r, err)
}

// LockNotification locks and reads a receiver's undeleted notification in
// the transaction ctx carries (app.NotificationRows, rows.sql).
func (s *Store) LockNotification(ctx context.Context, id, receiverID uuid.UUID) (app.StoredNotification, bool, error) {
	r, err := s.queries(ctx).LockNotification(ctx, gen.LockNotificationParams{ID: id, ReceiverID: receiverID})
	return storedOf("lock", id, gen.GetNotificationRow(r), err)
}

// UpdateNotification writes a receiver's notification's states in the
// transaction ctx carries, under LockNotification's lock
// (app.NotificationRows, rows.sql).
func (s *Store) UpdateNotification(ctx context.Context, w app.StateWrite) (app.Notification, error) {
	r, err := s.queries(ctx).UpdateNotification(ctx, gen.UpdateNotificationParams{ReadAt: w.State.ReadAt, ArchivedAt: w.State.ArchivedAt,
		SnoozedTill: w.State.SnoozedTill, At: w.At, ReceiverID: w.ReceiverID, ID: w.ID})
	if err != nil {
		return app.Notification{}, fmt.Errorf("update notification %s: %w", w.ID, err)
	}
	return notificationOf(gen.ListNotificationsRow(r))
}

// storedOf is the notification r stores, found when err is not
// pgx.ErrNoRows; the read's other error, as what of id.
func storedOf(what string, id uuid.UUID, r gen.GetNotificationRow, err error) (app.StoredNotification, bool, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return app.StoredNotification{}, false, nil
	}
	if err != nil {
		return app.StoredNotification{}, false, fmt.Errorf("%s notification %s: %w", what, id, err)
	}
	n, err := notificationOf(gen.ListNotificationsRow{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, Sender: r.Sender,
		TriggeredByID: r.TriggeredByID, Data: r.Data, ReadAt: r.ReadAt, ArchivedAt: r.ArchivedAt, SnoozedTill: r.SnoozedTill, CreatedAt: r.CreatedAt})
	if err != nil {
		return app.StoredNotification{}, false, err
	}
	return app.StoredNotification{Notification: n, IssueCreatedByID: r.IssueCreatedByID}, true, nil
}
