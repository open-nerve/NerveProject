package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// NotificationPage is a page of the list (spec 2.3): its notifications,
// none for none, and the cursor of the next page, nil on the last.
type NotificationPage struct {
	Data       []NotificationView
	NextCursor *string
}

// ListNotificationsDeps supply the list: its inbox, its read snapshot
// (REPEATABLE READ, READ ONLY), the store's list and the work items'
// names.
type ListNotificationsDeps struct {
	InboxDeps
	ReadTx shared.TxManager
	Lists  NotificationLists
	Refs   IssueRefs
}

// ListNotifications lists the caller's notifications in a workspace,
// newest first (spec 2.2, 2.3): GET /workspaces/{slug}/notifications.
type ListNotifications struct{ d ListNotificationsDeps }

// NewListNotifications constructs the list.
func NewListNotifications(d ListNotificationsDeps) *ListNotifications {
	return &ListNotifications{d: d}
}

// Execute checks the request's own values first, as listDrafts does,
// whatever the workspace: the filter of q, then limit, then cursor, each a
// 422 or the cursor's 400. Then, in one read snapshot (spec 2.3): the
// inbox of the workspace slug names, for notification.read; the store's
// rows the filter selects after the cursor's, one more than the page,
// with the clock's moment the inbox read; the page trimmed to limit, whose
// last row is the next cursor when a row was left over; and each answered
// with its view, the work items the page names asked for once.
func (u *ListNotifications) Execute(ctx context.Context, slug string, q domain.ListQuery, limit *int, cursor *string) (NotificationPage, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return NotificationPage{}, err
	}
	filter, err := domain.ParseListFilter(q)
	if err != nil {
		return NotificationPage{}, err
	}
	size, err := shared.PageSize(limit)
	if err != nil {
		return NotificationPage{}, err
	}
	var after *domain.ListCursor
	if cursor != nil {
		c, err := domain.DecodeListCursor(*cursor)
		if err != nil {
			return NotificationPage{}, err
		}
		after = &c
	}
	var page NotificationPage
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error {
		inbox, err := u.d.open(ctx, actor, slug, domain.ActionNotificationRead)
		if err != nil {
			return err
		}
		rows, err := u.d.Lists.ListNotifications(ctx, ListRead{Inbox: inbox, Filter: filter, After: after, Limit: size + 1})
		if err != nil {
			return err
		}
		if len(rows) > size {
			rows = rows[:size]
			last := rows[len(rows)-1]
			next, err := domain.EncodeListCursor(domain.ListCursor{CreatedAt: last.CreatedAt, ID: last.ID})
			if err != nil {
				return err
			}
			page.NextCursor = &next
		}
		page.Data, err = viewsOf(ctx, u.d.Refs, actor.UserID, inbox.Grants, rows)
		return err
	})
	if err != nil {
		return NotificationPage{}, err
	}
	return page, nil
}

// GetUnreadCountDeps supply the unread counts: their inbox, their read
// snapshot (REPEATABLE READ, READ ONLY) and the store's count.
type GetUnreadCountDeps struct {
	InboxDeps
	ReadTx shared.TxManager
	Lists  NotificationLists
}

// GetUnreadCount counts the caller's unread notifications in a workspace,
// of each tab (spec 2.2, R3): GET
// /workspaces/{slug}/notifications/unread-count.
type GetUnreadCount struct{ d GetUnreadCountDeps }

// NewGetUnreadCount constructs the unread counts.
func NewGetUnreadCount(d GetUnreadCountDeps) *GetUnreadCount {
	return &GetUnreadCount{d: d}
}

// Execute, in one read snapshot: the inbox of the workspace slug names,
// for notification.read, and the store's count of its unread, unarchived
// notifications not snoozed at the clock's moment the inbox read, of the
// all tab's senders and of the mentions tab's (domain.TabSenders).
func (u *GetUnreadCount) Execute(ctx context.Context, slug string) (UnreadCounts, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return UnreadCounts{}, err
	}
	var counts UnreadCounts
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error {
		inbox, err := u.d.open(ctx, actor, slug, domain.ActionNotificationRead)
		if err != nil {
			return err
		}
		counts, err = u.d.Lists.CountUnread(ctx, UnreadRead{Inbox: inbox, All: domain.TabSenders(domain.TabAll), Mentions: domain.TabSenders(domain.TabMentions)})
		return err
	})
	if err != nil {
		return UnreadCounts{}, err
	}
	return counts, nil
}
