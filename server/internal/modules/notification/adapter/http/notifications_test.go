package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	oldParentID = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000b2")
	newParentID = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000b3")
	commentID   = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000d1")
	mentionID   = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000c2")
)

// fix is the work item Fix as its notifications keep it.
var fix = domain.IssueSnapshot{ID: fixID, Name: "Fix", SequenceID: 3, Identifier: "WEB", StateName: "Todo", StateGroup: "unstarted"}

// fullView is alice's notification with every value it may have, each a
// value of its own: bob's change of Fix's parent, both work items named,
// read, archived and snoozed.
func fullView() app.NotificationView {
	oldValue, newValue, excerpt := "OPS-4", "WEB-9", "Looks good"
	read, archived, till := sent.Add(time.Minute), sent.Add(2*time.Minute), sent.Add(time.Hour)
	return app.NotificationView{ID: noteID, WorkspaceID: acmeID, ProjectID: webID, IssueID: fixID, Sender: domain.SenderAssigned,
		TriggeredByID: &bobID, CreatedAt: sent,
		Data: domain.View{Issue: fix, Change: domain.ChangeView{Field: "parent", Verb: domain.VerbUpdated, OldValue: &oldValue, NewValue: &newValue,
			OldID: &oldParentID, NewID: &newParentID, CommentID: &commentID, Excerpt: &excerpt,
			OldIssue: &domain.IssueRef{ID: oldParentID, SequenceID: 4, Identifier: "OPS", Name: "Old parent"},
			NewIssue: &domain.IssueRef{ID: newParentID, SequenceID: 9, Identifier: "WEB", Name: "New parent"}}},
		State: domain.State{ReadAt: &read, ArchivedAt: &archived, SnoozedTill: &till}}
}

// bareView is alice's notification with none of the values it may lack:
// nerve's mention of her in Fix's description, in none of the three states.
func bareView() app.NotificationView {
	return app.NotificationView{ID: mentionID, WorkspaceID: acmeID, ProjectID: webID, IssueID: fixID, Sender: domain.SenderMentioned, CreatedAt: sent,
		Data: domain.View{Issue: fix, Change: domain.ChangeView{Field: "description", Verb: domain.VerbMentioned}}}
}

const (
	fixJSON = `"issue":{"id":"0199a2b4-0000-7000-8000-0000000000b1","identifier":"WEB","name":"Fix","sequence_id":3,"state_group":"unstarted",` +
		`"state_name":"Todo"}`
	fullJSON = `{"archived_at":"2026-11-18T10:02:00.123456Z","created_at":"2026-11-18T10:00:00.123456Z","data":{"change":{` +
		`"comment_id":"0199a2b4-0000-7000-8000-0000000000d1","excerpt":"Looks good","field":"parent","new_id":"0199a2b4-0000-7000-8000-0000000000b3",` +
		`"new_issue":{"id":"0199a2b4-0000-7000-8000-0000000000b3","identifier":"WEB","name":"New parent","sequence_id":9},"new_value":"WEB-9",` +
		`"old_id":"0199a2b4-0000-7000-8000-0000000000b2","old_issue":{"id":"0199a2b4-0000-7000-8000-0000000000b2","identifier":"OPS",` +
		`"name":"Old parent","sequence_id":4},"old_value":"OPS-4","verb":"updated"},` + fixJSON + `},"id":"0199a2b4-0000-7000-8000-0000000000c1",` +
		`"issue_id":"0199a2b4-0000-7000-8000-0000000000b1","project_id":"0199a2b4-0000-7000-8000-0000000000a1",` +
		`"read_at":"2026-11-18T10:01:00.123456Z","sender":"assigned","snoozed_till":"2026-11-18T11:00:00.123456Z",` +
		`"triggered_by_id":"0199a2b4-0000-7000-8000-000000000002","workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	bareJSON = `{"archived_at":null,"created_at":"2026-11-18T10:00:00.123456Z","data":{"change":{"comment_id":null,"excerpt":null,` +
		`"field":"description","new_id":null,"new_issue":null,"new_value":null,"old_id":null,"old_issue":null,"old_value":null,"verb":"mentioned"},` +
		fixJSON + `},"id":"0199a2b4-0000-7000-8000-0000000000c2","issue_id":"0199a2b4-0000-7000-8000-0000000000b1",` +
		`"project_id":"0199a2b4-0000-7000-8000-0000000000a1","read_at":null,"sender":"mentioned","snoozed_till":null,"triggered_by_id":null,` +
		`"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
)

// refusal is a use case's error and the answer it is.
type refusal struct {
	name   string
	err    error
	status int
	want   string
}

// The refusals of the workspace operations, as the contract declares them,
// and a failure.
func workspaceRefusals(t *testing.T, invalid error) []refusal {
	t.Helper()
	return []refusal{
		{"no workspace", domain.ErrWorkspaceNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`},
		{"invalid", invalid, http.StatusUnprocessableEntity, `{"status":422,"code":"validation_failed","title":"Unprocessable Entity",` +
			`"detail":"The request has invalid values.","errors":[{"field":"type","code":"not_allowed","message":"is only allowed in the all tab"}]}`},
		{"a failure", errGone, http.StatusInternalServerError, failed},
	}
}

// typeInMentions is the domain's refusal of a type in the mentions tab.
func typeInMentions(t *testing.T) error {
	t.Helper()
	_, err := domain.NewFilter(domain.FilterValues{Tab: new(domain.TabMentions), Type: &[]domain.Sender{domain.SenderAssigned}})
	if err == nil {
		t.Fatal("a type in the mentions tab was not refused")
	}
	return err
}

// GET goes to the use case for the caller and the path's workspace with
// each query value as given, the strings unparsed, and limit and cursor;
// one left out is nil. The answer is 200 with the page's notifications in
// its order, each with every key, null for each value it has not, and the
// next cursor.
func TestListNotifications(t *testing.T) {
	next := "next-page"
	for _, tt := range []struct {
		query string
		want  listCall
	}{
		{"?tab=all&type=assigned,created&read=false&archived=true&snoozed=false&limit=2&cursor=abc",
			listCall{Caller: aliceID, Slug: "acme", Query: domain.ListQuery{Tab: new("all"), Type: new("assigned,created"), Read: new("false"),
				Archived: new("true"), Snoozed: new("false")}, Limit: new(2), Cursor: new("abc")}},
		{"", listCall{Caller: aliceID, Slug: "acme"}},
		{"?tab=inbox&type=&read=yes", listCall{Caller: aliceID, Slug: "acme", Query: domain.ListQuery{Tab: new("inbox"), Type: new(""), Read: new("yes")}}},
	} {
		f := newFakes()
		f.list.answer = app.NotificationPage{Data: []app.NotificationView{fullView(), bareView()}, NextCursor: &next}
		res, body := do(t, newServer(t, f.useCases()), request(http.MethodGet, listPath+tt.query, "alice", ""))
		if want := `{"data":[` + fullJSON + `,` + bareJSON + `],"next_cursor":"next-page"}`; res.StatusCode != http.StatusOK || body != want+"\n" {
			t.Errorf("GET %s = %d %s, want 200 %s", tt.query, res.StatusCode, body, want)
		}
		if !reflect.DeepEqual(f.list.calls, []listCall{tt.want}) || f.asked() != 1 {
			t.Errorf("GET %s: the use cases got %+v, want the list's %+v alone", tt.query, f.list.calls, tt.want)
		}
	}
}

// The last page's cursor is null, and a page of none is [], never null.
func TestListNotificationsLastAndEmptyPages(t *testing.T) {
	for _, tt := range []struct {
		page app.NotificationPage
		want string
	}{
		{app.NotificationPage{Data: []app.NotificationView{bareView()}}, `{"data":[` + bareJSON + `],"next_cursor":null}`},
		{app.NotificationPage{Data: []app.NotificationView{}}, `{"data":[],"next_cursor":null}`},
		{app.NotificationPage{}, `{"data":[],"next_cursor":null}`},
	} {
		f := newFakes()
		f.list.answer = tt.page
		if res, body := do(t, newServer(t, f.useCases()), request(http.MethodGet, listPath, "alice", "")); res.StatusCode != http.StatusOK ||
			body != tt.want+"\n" {
			t.Errorf("GET of %+v = %d %s, want 200 %s", tt.page, res.StatusCode, body, tt.want)
		}
	}
}

// The list's refusals, as the contract declares them: the workspace's, a
// filter's and a cursor's; and a failure.
func TestListNotificationsRefusals(t *testing.T) {
	_, cursor := domain.DecodeListCursor("nope")
	cases := append(workspaceRefusals(t, typeInMentions(t)), refusal{"a cursor", cursor, http.StatusBadRequest,
		`{"status":400,"code":"bad_request","title":"Bad Request","detail":"The cursor is not one this list can read.",` +
			`"errors":[{"field":"cursor","code":"invalid_format","message":"is not a cursor this list can read"}]}`})
	for _, tt := range cases {
		f := newFakes()
		f.list.err = tt.err
		if res, body := do(t, newServer(t, f.useCases()), request(http.MethodGet, listPath+"?cursor=nope", "alice", "")); res.StatusCode != tt.status ||
			body != tt.want+"\n" {
			t.Errorf("%s: GET = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// GET goes to the use case for the caller and the path's workspace; the
// answer is 200 with each tab's count under its own name.
func TestGetNotificationUnreadCount(t *testing.T) {
	f := newFakes()
	f.count.answer = app.UnreadCounts{All: 3, Mentions: 1}
	if res, body := do(t, newServer(t, f.useCases()), request(http.MethodGet, countPath, "alice", "")); res.StatusCode != http.StatusOK ||
		body != `{"all":3,"mentions":1}`+"\n" {
		t.Errorf("GET = %d %s, want 200 {all: 3, mentions: 1}", res.StatusCode, body)
	}
	if want := []countCall{{Caller: aliceID, Slug: "acme"}}; !reflect.DeepEqual(f.count.calls, want) || f.asked() != 1 {
		t.Errorf("the use cases got %+v, want the count's %+v alone", f.count.calls, want)
	}
	for _, tt := range workspaceRefusals(t, typeInMentions(t)) {
		if tt.status == http.StatusUnprocessableEntity {
			continue
		}
		f := newFakes()
		f.count.err = tt.err
		if res, body := do(t, newServer(t, f.useCases()), request(http.MethodGet, countPath, "alice", "")); res.StatusCode != tt.status ||
			body != tt.want+"\n" {
			t.Errorf("%s: GET = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// POST goes to the use case for the caller and the path's workspace with
// the body's values as given, the enums as the domain's tab and senders
// unchecked (the use case checks them), a value left out nil and a type
// given empty kept; the answer is 200 with how many it marked.
func TestMarkAllNotificationsRead(t *testing.T) {
	for _, tt := range []struct {
		body string
		want domain.FilterValues
	}{
		{`{}`, domain.FilterValues{}},
		{`{"tab":"all","type":["assigned","subscribed"],"archived":true,"snoozed":false}`, domain.FilterValues{Tab: new(domain.TabAll),
			Type: &[]domain.Sender{domain.SenderAssigned, domain.SenderSubscribed}, Archived: new(true), Snoozed: new(false)}},
		{`{"tab":"mentions","snoozed":true}`, domain.FilterValues{Tab: new(domain.TabMentions), Snoozed: new(true)}},
		{`{"tab":"inbox","type":["mentioned","created","created"]}`, domain.FilterValues{Tab: new(domain.Tab("inbox")),
			Type: &[]domain.Sender{domain.SenderMentioned, domain.SenderCreated, domain.SenderCreated}}},
		{`{"type":[]}`, domain.FilterValues{Type: &[]domain.Sender{}}},
	} {
		f := newFakes()
		f.mark.answer = 7
		res, body := do(t, newServer(t, f.useCases()), request(http.MethodPost, markPath, "alice", tt.body))
		if res.StatusCode != http.StatusOK || body != `{"marked":7}`+"\n" {
			t.Errorf("POST %s = %d %s, want 200 {marked: 7}", tt.body, res.StatusCode, body)
		}
		if want := []markCall{{Caller: aliceID, Slug: "acme", Values: tt.want}}; !reflect.DeepEqual(f.mark.calls, want) || f.asked() != 1 {
			t.Errorf("POST %s: the use cases got %+v, want mark-all-read's %+v alone", tt.body, f.mark.calls, want)
		}
	}
}

// Mark-all-read's refusals, as the contract declares them: the
// workspace's and a filter's; and a failure.
func TestMarkAllNotificationsReadRefusals(t *testing.T) {
	for _, tt := range workspaceRefusals(t, typeInMentions(t)) {
		f := newFakes()
		f.mark.err = tt.err
		if res, body := do(t, newServer(t, f.useCases()), request(http.MethodPost, markPath, "alice", `{"tab":"mentions","type":["assigned"]}`)); res.StatusCode != tt.status ||
			body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// samePatch tells whether a and b are the same change: each state given or
// not alike, snoozed_till the same instant.
func samePatch(a, b domain.Patch) bool {
	sameBool := func(x, y *bool) bool { return (x == nil) == (y == nil) && (x == nil || *x == *y) }
	sameTill := (a.SnoozedTill == nil) == (b.SnoozedTill == nil) && (a.SnoozedTill == nil || a.SnoozedTill.Equal(*b.SnoozedTill))
	return sameBool(a.Read, b.Read) && sameBool(a.Archived, b.Archived) && a.SetSnoozed == b.SetSnoozed && sameTill
}

// PATCH goes to the use case for the caller and the path's notification
// with the body's three states: read and archived as given, nil when left
// out; snoozed_till given or not, null given as nil. The answer is 200
// with the notification as the list answers it.
func TestUpdateNotification(t *testing.T) {
	till := time.Date(2026, 11, 19, 0, 0, 0, 500000000, time.UTC)
	for _, tt := range []struct {
		body string
		want domain.Patch
	}{
		{`{}`, domain.Patch{}},
		{`{"read":true}`, domain.Patch{Read: new(true)}},
		{`{"read":false,"archived":true}`, domain.Patch{Read: new(false), Archived: new(true)}},
		{`{"archived":false}`, domain.Patch{Archived: new(false)}},
		{`{"snoozed_till":"2026-11-19T08:00:00.5+08:00"}`, domain.Patch{SetSnoozed: true, SnoozedTill: &till}},
		{`{"snoozed_till":null}`, domain.Patch{SetSnoozed: true}},
		{`{"read":true,"snoozed_till":null}`, domain.Patch{Read: new(true), SetSnoozed: true}},
	} {
		f := newFakes()
		f.update.answer = fullView()
		res, body := do(t, newServer(t, f.useCases()), request(http.MethodPatch, updatePath, "alice", tt.body))
		if res.StatusCode != http.StatusOK || body != fullJSON+"\n" {
			t.Errorf("PATCH %s = %d %s, want 200 %s", tt.body, res.StatusCode, body, fullJSON)
		}
		if len(f.update.calls) != 1 || f.asked() != 1 || f.update.calls[0].Caller != aliceID || f.update.calls[0].ID != noteID ||
			!samePatch(f.update.calls[0].Patch, tt.want) {
			t.Errorf("PATCH %s: the use cases got %+v, want the change's %+v alone", tt.body, f.update.calls, tt.want)
		}
	}
	f := newFakes()
	f.update.answer = bareView()
	if res, body := do(t, newServer(t, f.useCases()), request(http.MethodPatch, updatePath, "alice", `{}`)); res.StatusCode != http.StatusOK ||
		body != bareJSON+"\n" {
		t.Errorf("PATCH of a bare notification = %d %s, want 200 %s", res.StatusCode, body, bareJSON)
	}
}

// A body that is not the change's structure, null for read or archived, a
// value of another type, a snoozed_till that is no moment, a field it does
// not have: refused as bad_request before the use case.
func TestUpdateNotificationHoldsTheBodyToItsStructure(t *testing.T) {
	f := newFakes()
	h := newServer(t, f.useCases())
	for _, body := range []string{`[]`, `{"read":null}`, `{"archived":null}`, `{"read":"true"}`, `{"archived":1}`, `{"snoozed_till":"tomorrow"}`,
		`{"snoozed_till":1}`, `{"snoozed":true}`} {
		res, got := do(t, h, request(http.MethodPatch, updatePath, "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("PATCH %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if f.asked() != 0 {
		t.Errorf("the use cases got %d calls, want none", f.asked())
	}
}

// The change's refusals, as the contract declares them: a notification
// the caller cannot reach, a snooze not in the future, a lock not had in
// time (with Retry-After: 1); and a failure.
func TestUpdateNotificationRefusals(t *testing.T) {
	_, _, past := domain.Patch{SetSnoozed: true, SnoozedTill: &sent}.Apply(domain.State{}, sent)
	for _, tt := range []refusal{
		{"not reached", domain.ErrNotificationNotFound, http.StatusNotFound,
			`{"status":404,"code":"notification.not_found","title":"Not Found","detail":"The notification does not exist, or you cannot see it."}`},
		{"a snooze not in the future", past, http.StatusUnprocessableEntity, `{"status":422,"code":"validation_failed","title":"Unprocessable Entity",` +
			`"detail":"The request has invalid values.","errors":[{"field":"snoozed_till","code":"must_be_future","message":"must be in the future"}]}`},
		{"a lock not had in time", shared.ServerBusy(time.Second), http.StatusServiceUnavailable, busy},
		{"a failure", errGone, http.StatusInternalServerError, failed},
	} {
		f := newFakes()
		f.update.err = tt.err
		res, body := do(t, newServer(t, f.useCases()), request(http.MethodPatch, updatePath, "alice", `{"snoozed_till":"2026-11-18T10:00:00Z"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" || tt.want == busy && res.Header.Get("Retry-After") != "1" {
			t.Errorf("%s: PATCH = %d %s, Retry-After %q; want %d %s", tt.name, res.StatusCode, body, res.Header.Get("Retry-After"), tt.status, tt.want)
		}
	}
}
