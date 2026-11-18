package httpadapter

import (
	"context"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
)

// ListNotifications serves GET /api/v0/workspaces/{slug}/notifications:
// the filters go to the use case as the strings given, nil when left out
// (the domain parses them, P1c spec 2.3), with limit and cursor.
func (h handler) ListNotifications(ctx context.Context, req gen.ListNotificationsRequestObject) (gen.ListNotificationsResponseObject, error) {
	p := req.Params
	q := domain.ListQuery{Tab: p.Tab, Type: p.Type, Read: p.Read, Archived: p.Archived, Snoozed: p.Snoozed}
	page, err := h.uc.ListNotifications.Execute(ctx, req.Slug, q, p.Limit, p.Cursor)
	if err != nil {
		return nil, err
	}
	data := make([]gen.Notification, len(page.Data))
	for n, v := range page.Data {
		data[n] = notificationOf(v)
	}
	return gen.ListNotifications200JSONResponse{Data: data, NextCursor: orNull(page.NextCursor)}, nil
}

// GetNotificationUnreadCount serves GET
// /api/v0/workspaces/{slug}/notifications/unread-count: each tab's count
// under its name (R3).
func (h handler) GetNotificationUnreadCount(ctx context.Context, req gen.GetNotificationUnreadCountRequestObject) (gen.GetNotificationUnreadCountResponseObject, error) {
	c, err := h.uc.GetUnreadCount.Execute(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	return gen.GetNotificationUnreadCount200JSONResponse{All: int64(c.All), Mentions: int64(c.Mentions)}, nil
}

// MarkAllNotificationsRead serves POST
// /api/v0/workspaces/{slug}/notifications/mark-all-read: the body's values
// go to the use case as given, the tab and the types as the domain's tab
// and senders, unchecked (the domain checks them, spec 2.7); a value left
// out is nil, a type given empty is kept.
func (h handler) MarkAllNotificationsRead(ctx context.Context, req gen.MarkAllNotificationsReadRequestObject) (gen.MarkAllNotificationsReadResponseObject, error) {
	b := req.Body
	v := domain.FilterValues{Tab: (*domain.Tab)(b.Tab), Archived: b.Archived, Snoozed: b.Snoozed}
	if b.Type != nil {
		senders := make([]domain.Sender, len(*b.Type))
		for n, t := range *b.Type {
			senders[n] = domain.Sender(t)
		}
		v.Type = &senders
	}
	marked, err := h.uc.MarkAllRead.Execute(ctx, req.Slug, v)
	if err != nil {
		return nil, err
	}
	return gen.MarkAllNotificationsRead200JSONResponse{Marked: int64(marked)}, nil
}

// UpdateNotification serves PATCH /api/v0/notifications/{notification_id}:
// read and archived go to the use case as given, nil when left out;
// snoozed_till given or not, null given as nil, which ends the snooze
// (spec 2.7).
func (h handler) UpdateNotification(ctx context.Context, req gen.UpdateNotificationRequestObject) (gen.UpdateNotificationResponseObject, error) {
	b := req.Body
	p := domain.Patch{Read: b.Read, Archived: b.Archived}
	p.SetSnoozed, p.SnoozedTill = given(b.SnoozedTill)
	v, err := h.uc.UpdateNotification.Execute(ctx, req.NotificationID, p)
	if err != nil {
		return nil, err
	}
	return gen.UpdateNotification200JSONResponse(notificationOf(v)), nil
}

// notificationOf is v as the API shows it (spec 2.4): every key, null for
// each value it has not.
func notificationOf(v app.NotificationView) gen.Notification {
	i, c := v.Data.Issue, v.Data.Change
	return gen.Notification{ID: v.ID, WorkspaceID: v.WorkspaceID, ProjectID: v.ProjectID, IssueID: v.IssueID, Sender: gen.NotificationSender(v.Sender),
		TriggeredByID: orNull(v.TriggeredByID), CreatedAt: v.CreatedAt,
		ReadAt: orNull(v.State.ReadAt), ArchivedAt: orNull(v.State.ArchivedAt), SnoozedTill: orNull(v.State.SnoozedTill),
		Data: gen.NotificationData{
			Issue: gen.NotificationIssue{ID: i.ID, Name: i.Name, SequenceID: i.SequenceID, Identifier: i.Identifier, StateName: i.StateName,
				StateGroup: i.StateGroup},
			Change: gen.NotificationChange{Field: c.Field, Verb: gen.NotificationVerb(c.Verb), OldValue: orNull(c.OldValue), NewValue: orNull(c.NewValue),
				OldID: orNull(c.OldID), NewID: orNull(c.NewID), OldIssue: orNull(issueRefOf(c.OldIssue)), NewIssue: orNull(issueRefOf(c.NewIssue)),
				CommentID: orNull(c.CommentID), Excerpt: orNull(c.Excerpt)},
		}}
}

// issueRefOf is r as the API shows it, nil for nil.
func issueRefOf(r *domain.IssueRef) *gen.NotificationIssueRef {
	if r == nil {
		return nil
	}
	return &gen.NotificationIssueRef{ID: r.ID, SequenceID: r.SequenceID, Identifier: r.Identifier, Name: r.Name}
}

// orNull is v as a nullable value: null for nil, never left out.
func orNull[T any](v *T) nullable.Nullable[T] {
	if v == nil {
		return nullable.NewNullNullable[T]()
	}
	return nullable.NewNullableWithValue(*v)
}

// given is whether v is given, and its value, nil for null.
func given[T any](v nullable.Nullable[T]) (bool, *T) {
	switch {
	case !v.IsSpecified():
		return false, nil
	case v.IsNull():
		return true, nil
	}
	value := v.MustGet()
	return true, &value
}
