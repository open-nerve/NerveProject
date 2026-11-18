package app_test

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func listNotifications(f *inboxPorts) *app.ListNotifications {
	return app.NewListNotifications(app.ListNotificationsDeps{InboxDeps: f.inbox(), ReadTx: f.readTx(), Lists: f, Refs: f})
}

func getUnreadCount(f *inboxPorts) *app.GetUnreadCount {
	return app.NewGetUnreadCount(app.GetUnreadCountDeps{InboxDeps: f.inbox(), ReadTx: f.readTx(), Lists: f})
}

// inboxOpened are the calls that open alice's inbox in acme for action in the read snapshot (spec 2.2, 2.5): the
// workspace by its slug, the decision on it, the projects she reads, then the clock.
func inboxOpened(action shared.Action) []string {
	return []string{"begin read", callIn("read", "WorkspaceBySlug", "acme"), callIn("read", "Authorize", alice, action, shared.Target{WorkspaceID: acme}),
		callIn("read", "Readable", alice, acme), "Now"}
}

// alicesInbox is the Inbox newInboxPorts open for alice in acme at the clock's moment.
func alicesInbox(f *inboxPorts) app.Inbox {
	return app.Inbox{ReceiverID: alice, WorkspaceID: acme, Grants: f.grants, Now: inboxNow}
}

func sameCalls(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("calls\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// A page is read in one snapshot (spec 2.3): the inbox opened once, the clock read, the store asked for one more row
// than the page, by the default filter, and the work items the page names asked for once, with the projects she reads;
// each notification is answered with its view, the parent she reads named.
func TestListNotificationsReadsAPageInOneSnapshot(t *testing.T) {
	f := newInboxPorts()
	f.rows = []app.Notification{note(anHourAgo, parentChange), note(twoHoursAgo, renamedChange)}
	page, err := listNotifications(f).Execute(readerOf(alice), "acme", domain.ListQuery{}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	sameCalls(t, f.calls, append(inboxOpened(domain.ActionNotificationRead), callIn("read", "ListNotifications", shared.DefaultPageSize+1),
		callIn("read", "IssueRefs", alice, f.grants, []uuid.UUID{parentItem}), "end read"))
	want := app.ListRead{Inbox: alicesInbox(f), Filter: domain.ListFilter{Filter: domain.Filter{Senders: domain.TabSenders(domain.TabAll)}},
		Limit: shared.DefaultPageSize + 1}
	if !reflect.DeepEqual(f.listed, []app.ListRead{want}) {
		t.Fatalf("read %+v, want %+v", f.listed, want)
	}
	if page.NextCursor != nil || len(page.Data) != 2 {
		t.Fatalf("page %+v, want both rows and no cursor", page)
	}
	for n, row := range f.rows {
		if v, want := page.Data[n], viewOf(row, map[uuid.UUID]domain.IssueRef{parentItem: parentRef}); !reflect.DeepEqual(v, want) {
			t.Errorf("row %d answered %+v, want %+v", n, v, want)
		}
	}
}

// viewOf is n as its reader is answered, refs the work items he reads.
func viewOf(n app.Notification, refs map[uuid.UUID]domain.IssueRef) app.NotificationView {
	return app.NotificationView{ID: n.ID, WorkspaceID: n.WorkspaceID, ProjectID: n.ProjectID, IssueID: n.IssueID, Sender: n.Sender,
		TriggeredByID: n.TriggeredByID, Data: domain.ViewOf(n.Data, refs), State: n.State, CreatedAt: n.CreatedAt}
}

// The filter, the limit and the cursor are the request's (spec 2.3): the store is asked for limit+1 rows after the
// cursor's; of limit+1 rows the extra one is dropped and the next cursor is the last answered row's, and a page of no
// more than limit rows is the last, without a cursor.
func TestListNotificationsTrimsThePageToItsLimit(t *testing.T) {
	rows := []app.Notification{note(inboxNow, renamedChange), note(anHourAgo, renamedChange), note(anHourAgo, renamedChange)}
	after := domain.ListCursor{CreatedAt: inboxNow, ID: uuid.NewV7()}
	cursor, err := domain.EncodeListCursor(after)
	if err != nil {
		t.Fatal(err)
	}
	unreadMentions := domain.ListQuery{Tab: str("mentions"), Read: str("false")}
	for _, c := range []struct {
		name  string
		rows  []app.Notification
		limit int
		next  *domain.ListCursor
	}{
		{"one row more than the page", rows, 2, &domain.ListCursor{CreatedAt: rows[1].CreatedAt, ID: rows[1].ID}},
		{"exactly the page", rows[:2], 2, nil},
		{"fewer than the page", rows[:1], 2, nil},
		{"none", nil, 2, nil},
		{"a page of one", rows, 1, &domain.ListCursor{CreatedAt: rows[0].CreatedAt, ID: rows[0].ID}},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newInboxPorts()
			f.rows = c.rows
			page, err := listNotifications(f).Execute(readerOf(alice), "acme", unreadMentions, &c.limit, &cursor)
			if err != nil {
				t.Fatal(err)
			}
			want := app.ListRead{Inbox: alicesInbox(f), Filter: domain.ListFilter{Filter: domain.Filter{Senders: domain.TabSenders(domain.TabMentions)}, Read: new(false)},
				After: &after, Limit: c.limit + 1}
			if !reflect.DeepEqual(f.listed, []app.ListRead{want}) {
				t.Fatalf("read %+v, want %+v", f.listed, want)
			}
			kept := c.rows[:min(len(c.rows), c.limit)]
			var ids []uuid.UUID
			for _, v := range page.Data {
				ids = append(ids, v.ID)
			}
			if page.Data == nil || len(ids) != len(kept) || !slices.EqualFunc(kept, ids, func(n app.Notification, id uuid.UUID) bool { return n.ID == id }) {
				t.Fatalf("answered %v (nil %v), want the first %d rows", ids, page.Data == nil, len(kept))
			}
			if c.next == nil {
				if page.NextCursor != nil {
					t.Fatalf("next cursor %q, want none on the last page", *page.NextCursor)
				}
				return
			}
			got, err := domain.DecodeListCursor(*page.NextCursor)
			if err != nil || got != *c.next {
				t.Fatalf("next cursor %+v (%v), want %+v", got, err, *c.next)
			}
		})
	}
}

// A page that names no work item, an empty one too, does not ask for any (spec 2.6); one whose work items alice
// does not read asks for them and answers them as nil.
func TestListNotificationsAsksForTheWorkItemsOnlyWhenThePageNamesSome(t *testing.T) {
	for _, c := range []struct {
		name string
		rows []app.Notification
		asks bool
	}{
		{"no row", nil, false},
		{"rows naming no work item", []app.Notification{note(anHourAgo, renamedChange), note(twoHoursAgo, renamedChange)}, false},
		{"a row naming one she does not read", []app.Notification{note(anHourAgo, blockedChange)}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newInboxPorts()
			f.rows = c.rows
			page, err := listNotifications(f).Execute(readerOf(alice), "acme", domain.ListQuery{}, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			want := append(inboxOpened(domain.ActionNotificationRead), callIn("read", "ListNotifications", shared.DefaultPageSize+1))
			if c.asks {
				want = append(want, callIn("read", "IssueRefs", alice, f.grants, []uuid.UUID{blockedItem}))
			}
			sameCalls(t, f.calls, append(want, "end read"))
			for _, v := range page.Data {
				if v.Data.Change.NewID != nil || v.Data.Change.NewIssue != nil {
					t.Fatalf("answered %+v, want the work item she does not read as nil", v.Data.Change)
				}
			}
		})
	}
}

// The filter, the limit and the cursor are checked before anything is read (listDrafts' order): a 422 or a 400 of the
// request's own values, whatever the workspace; the filter's before the limit's, the limit's before the cursor's.
func TestListNotificationsRefusesItsParametersBeforeReading(t *testing.T) {
	zero, bad := 0, "nope"
	for _, c := range []struct {
		name   string
		q      domain.ListQuery
		limit  *int
		cursor *string
		field  string
	}{
		{"a tab", domain.ListQuery{Tab: &bad}, nil, nil, "tab"},
		{"a limit", domain.ListQuery{}, &zero, nil, "limit"},
		{"a cursor", domain.ListQuery{}, nil, &bad, "cursor"},
		{"a tab and a limit", domain.ListQuery{Tab: &bad}, &zero, &bad, "tab"},
		{"a limit and a cursor", domain.ListQuery{}, &zero, &bad, "limit"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newInboxPorts()
			_, err := listNotifications(f).Execute(readerOf(alice), "nowhere", c.q, c.limit, c.cursor)
			var problem *shared.Error
			if !errors.As(err, &problem) || len(problem.Fields) == 0 || problem.Fields[0].Field != c.field || len(f.calls) != 0 {
				t.Fatalf("Execute = %v, calls %v; want a problem of %s and no call", err, f.calls, c.field)
			}
		})
	}
}

// The workspace is the slug's, which alice must be a member of (spec 2.2): a slug that names none and one she cannot
// see are alike workspace.not_found, the same value, and nothing is read after; any other refusal or failure is
// answered as it is.
func TestTheInboxOfAWorkspaceNotSeenIsNotFound(t *testing.T) {
	read := func(f *inboxPorts, slug string) error {
		_, err := listNotifications(f).Execute(readerOf(alice), slug, domain.ListQuery{}, nil, nil)
		return err
	}
	count := func(f *inboxPorts, slug string) error {
		_, err := getUnreadCount(f).Execute(readerOf(alice), slug)
		return err
	}
	for _, c := range []struct {
		name    string
		execute func(*inboxPorts, string) error
		ports   []string
	}{
		{"list", read, []string{"WorkspaceBySlug", "Authorize", "Readable", "ListNotifications"}},
		{"unread count", count, []string{"WorkspaceBySlug", "Authorize", "Readable", "CountUnread"}},
	} {
		execute := c.execute
		t.Run(c.name, func(t *testing.T) {
			f := newInboxPorts()
			if err := execute(f, "nowhere"); err != domain.ErrWorkspaceNotFound {
				t.Fatalf("an unknown slug: %v, want workspace.not_found", err)
			}
			sameCalls(t, f.calls, []string{"begin read", callIn("read", "WorkspaceBySlug", "nowhere"), "end read failed"})
			f = newInboxPorts()
			f.authErr = shared.ErrNotVisible
			if err := execute(f, "acme"); err != domain.ErrWorkspaceNotFound {
				t.Fatalf("a workspace not seen: %v, want workspace.not_found", err)
			}
			sameCalls(t, f.calls, append(inboxOpened(domain.ActionNotificationRead)[:3], "end read failed"))
			f = newInboxPorts()
			f.authErr = shared.Forbidden()
			if err := execute(f, "acme"); !errors.Is(err, shared.Forbidden()) {
				t.Fatalf("a refused role: %v, want forbidden", err)
			}
			for _, port := range c.ports {
				f = newInboxPorts()
				f.fail = port
				if err := execute(f, "acme"); !errors.Is(err, errInboxPort) {
					t.Fatalf("%s failing: %v, want its error", port, err)
				}
			}
		})
	}
}

// The list's views ask for the work items once and fail as that read.
func TestListNotificationsFailsAsTheWorkItemsRead(t *testing.T) {
	f := newInboxPorts()
	f.rows = []app.Notification{note(anHourAgo, parentChange)}
	f.fail = "IssueRefs"
	if _, err := listNotifications(f).Execute(readerOf(alice), "acme", domain.ListQuery{}, nil, nil); !errors.Is(err, errInboxPort) {
		t.Fatalf("Execute = %v, want the read's error", err)
	}
}

// The unread counts are read in one snapshot (spec 2.2, R3): the inbox opened once, the clock read, and one count of
// each tab's senders, answered as the store counts them.
func TestGetUnreadCountCountsBothTabsInOneSnapshot(t *testing.T) {
	f := newInboxPorts()
	f.counts = app.UnreadCounts{All: 7, Mentions: 2}
	counts, err := getUnreadCount(f).Execute(readerOf(alice), "acme")
	if err != nil {
		t.Fatal(err)
	}
	sameCalls(t, f.calls, append(inboxOpened(domain.ActionNotificationRead), callIn("read", "CountUnread"), "end read"))
	want := app.UnreadRead{Inbox: alicesInbox(f), All: domain.TabSenders(domain.TabAll), Mentions: domain.TabSenders(domain.TabMentions)}
	if !reflect.DeepEqual(f.counted, []app.UnreadRead{want}) || counts != f.counts {
		t.Fatalf("counted %+v as %+v; want %+v as %+v", f.counted, counts, want, f.counts)
	}
}
