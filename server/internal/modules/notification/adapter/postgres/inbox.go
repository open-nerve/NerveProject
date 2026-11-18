package postgresadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
)

// ListNotifications reads a page of the list in the transaction ctx
// carries (app.NotificationLists, inbox.sql).
func (s *Store) ListNotifications(ctx context.Context, r app.ListRead) ([]app.Notification, error) {
	if err := checkLimit("notification page", r.Limit); err != nil {
		return nil, err
	}
	ids, own := grantArrays(r.Grants)
	p := gen.ListNotificationsParams{ReceiverID: r.ReceiverID, WorkspaceID: r.WorkspaceID, ProjectIds: ids, ProjectOwn: own, Senders: senderTexts(r.Filter.Senders),
		Read: r.Filter.Read, Archived: r.Filter.Archived, Now: r.Now, Snoozed: r.Filter.Snoozed, PageLimit: int32(r.Limit)}
	if r.After != nil {
		p.HasAfter, p.AfterAt, p.AfterID = true, r.After.CreatedAt, r.After.ID
	}
	rows, err := s.queries(ctx).ListNotifications(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("list the notifications of %s in workspace %s: %w", r.ReceiverID, r.WorkspaceID, err)
	}
	page := make([]app.Notification, len(rows))
	for n, row := range rows {
		if page[n], err = notificationOf(row); err != nil {
			return nil, err
		}
	}
	return page, nil
}

// CountUnread reads the two unread counts in the transaction ctx carries
// (app.NotificationLists, inbox.sql).
func (s *Store) CountUnread(ctx context.Context, r app.UnreadRead) (app.UnreadCounts, error) {
	ids, own := grantArrays(r.Grants)
	c, err := s.queries(ctx).CountUnreadNotifications(ctx, gen.CountUnreadNotificationsParams{AllSenders: senderTexts(r.All), MentionSenders: senderTexts(r.Mentions),
		ReceiverID: r.ReceiverID, WorkspaceID: r.WorkspaceID, Now: r.Now, ProjectIds: ids, ProjectOwn: own})
	if err != nil {
		return app.UnreadCounts{}, fmt.Errorf("count the unread notifications of %s in workspace %s: %w", r.ReceiverID, r.WorkspaceID, err)
	}
	return app.UnreadCounts{All: int(c.AllUnread), Mentions: int(c.MentionsUnread)}, nil
}

// checkLimit refuses a LIMIT of what that is not from 1 to the int4 the
// statement takes.
func checkLimit(what string, limit int) error {
	if limit < 1 || limit > math.MaxInt32 {
		return fmt.Errorf("%s must be from 1 to 2147483647, got %d", what, limit)
	}
	return nil
}

// grantArrays are grants as the statements take them (spec R2): the paired
// arrays of project ids and own-only flags, empty, not NULL, for none.
func grantArrays(grants []app.Grant) ([]uuid.UUID, []bool) {
	ids, own := make([]uuid.UUID, len(grants)), make([]bool, len(grants))
	for n, g := range grants {
		ids[n], own[n] = g.ProjectID, g.OwnOnly
	}
	return ids, own
}

// senderTexts are senders as the statements take them, empty, not NULL,
// for none.
func senderTexts(senders []domain.Sender) []string {
	texts := make([]string, len(senders))
	for n, s := range senders {
		texts[n] = string(s)
	}
	return texts
}

// notificationOf is the notification a row of the list's columns stores,
// its snapshot decoded.
func notificationOf(r gen.ListNotificationsRow) (app.Notification, error) {
	var data domain.Data
	if err := json.Unmarshal(r.Data, &data); err != nil {
		return app.Notification{}, fmt.Errorf("decode the snapshot of notification %s: %w", r.ID, err)
	}
	return app.Notification{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, Sender: domain.Sender(r.Sender),
		TriggeredByID: r.TriggeredByID, Data: data, State: domain.State{ReadAt: r.ReadAt, ArchivedAt: r.ArchivedAt, SnoozedTill: r.SnoozedTill},
		CreatedAt: r.CreatedAt}, nil
}
