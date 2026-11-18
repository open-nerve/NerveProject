package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// apiNotification is a notification as the API answers it, the keys a
// test reads.
type apiNotification struct {
	ID            uuid.UUID  `json:"id"`
	WorkspaceID   uuid.UUID  `json:"workspace_id"`
	ProjectID     uuid.UUID  `json:"project_id"`
	IssueID       uuid.UUID  `json:"issue_id"`
	Sender        string     `json:"sender"`
	TriggeredByID *uuid.UUID `json:"triggered_by_id"`
	ReadAt        *time.Time `json:"read_at"`
	ArchivedAt    *time.Time `json:"archived_at"`
	Data          struct {
		Issue struct {
			ID         uuid.UUID `json:"id"`
			Name       string    `json:"name"`
			Identifier string    `json:"identifier"`
		} `json:"issue"`
		Change struct {
			Field string `json:"field"`
		} `json:"change"`
	} `json:"data"`
}

// notificationPage is a page of the list as the API answers it.
type notificationPage struct {
	Data       []apiNotification `json:"data"`
	NextCursor *string           `json:"next_cursor"`
}

// unreadCounts is the unread counts as the API answers them.
type unreadCounts struct {
	All      int `json:"all"`
	Mentions int `json:"mentions"`
}

// The notification module's four operations through the app, as bootstrap
// wires them (M7 P1c spec 2.2–2.4, 2.7): alice creates Fix in Web,
// assigned to bob, who is notified of its creation and of his assignment.
// His list in acme pages them one by one, each his, of Fix, sent by alice,
// unread; the unread counts are two and none; he reads one, which leaves
// one of the all tab unread; mark-all-read marks that one, and then none.
// A change of his notification whose row another transaction holds waits
// at most the work items' lock timeout, a second here, and is answered 503
// server_busy, Retry-After 1, changing nothing (busyTx); once the row is
// let go, the same change archives it.
func TestTheNotificationAPIThroughTheApp(t *testing.T) {
	w := newIssueWorldWith(t, func(cfg *config.Config) { cfg.Issue.WriteLockTimeout = time.Second })
	fix := w.newWorkItem(t)
	waitForFanouts(t, w.pool)
	const list, count, mark = "/api/v0/workspaces/acme/notifications", "/api/v0/workspaces/acme/notifications/unread-count",
		"/api/v0/workspaces/acme/notifications/mark-all-read"

	var first, second notificationPage
	decodeAnswer(t, w.send(t, "bob", http.MethodGet, list+"?limit=1", "", http.StatusOK), &first)
	if len(first.Data) != 1 || first.NextCursor == nil {
		t.Fatalf("bob's first page of one = %+v; want one notification and a cursor", first)
	}
	decodeAnswer(t, w.send(t, "bob", http.MethodGet, list+"?limit=1&cursor="+*first.NextCursor, "", http.StatusOK), &second)
	if len(second.Data) != 1 || second.NextCursor != nil {
		t.Fatalf("bob's second page of one = %+v; want one notification and no cursor", second)
	}
	notes := slices.Concat(first.Data, second.Data)
	var fields []string
	for _, n := range notes {
		if n.WorkspaceID != w.acme || n.ProjectID != w.web || n.IssueID != fix || n.Data.Issue.ID != fix || n.Data.Issue.Name != "Fix" ||
			n.Sender != "assigned" || n.TriggeredByID == nil || *n.TriggeredByID != w.ids["alice"] || n.ReadAt != nil || n.ArchivedAt != nil {
			t.Errorf("bob's notification %+v; want Fix's in Web, assigned, by alice, unread and not archived", n)
		}
		fields = append(fields, n.Data.Change.Field)
	}
	if slices.Sort(fields); !slices.Equal(fields, []string{"assignees", "issue"}) || notes[0].ID == notes[1].ID {
		t.Errorf("bob's notifications are of %v, ids %s and %s; want two, of Fix's creation and of his assignment", fields, notes[0].ID, notes[1].ID)
	}
	counted := func(want unreadCounts) {
		t.Helper()
		var got unreadCounts
		if decodeAnswer(t, w.send(t, "bob", http.MethodGet, count, "", http.StatusOK), &got); got != want {
			t.Errorf("bob's unread counts = %+v; want %+v", got, want)
		}
	}
	counted(unreadCounts{All: 2})

	var read apiNotification
	decodeAnswer(t, w.send(t, "bob", http.MethodPatch, "/api/v0/notifications/"+notes[0].ID.String(), `{"read":true}`, http.StatusOK), &read)
	if read.ID != notes[0].ID || read.ReadAt == nil || read.ArchivedAt != nil {
		t.Errorf("bob's read notification = %+v; want %s, read and not archived", read, notes[0].ID)
	}
	counted(unreadCounts{All: 1})
	var marked struct {
		Marked int `json:"marked"`
	}
	if decodeAnswer(t, w.send(t, "bob", http.MethodPost, mark, `{}`, http.StatusOK), &marked); marked.Marked != 1 {
		t.Errorf("bob's mark-all-read marked %d; want the one left unread", marked.Marked)
	}
	counted(unreadCounts{})
	if decodeAnswer(t, w.send(t, "bob", http.MethodPost, mark, `{}`, http.StatusOK), &marked); marked.Marked != 0 {
		t.Errorf("bob's mark-all-read again marked %d; want none", marked.Marked)
	}

	path := "/api/v0/notifications/" + notes[0].ID.String()
	row := holding(t, w.pool, "SELECT 1 FROM notifications WHERE id = $1 FOR UPDATE", notes[0].ID)
	req := newRequest(t, http.MethodPatch, w.base+path, w.tokens["bob"], []byte(`{"archived":true}`))
	w.contract.CheckRequest(t, req)
	sentAt := time.Now()
	a := receiveWithin(t, sendInBackground(req), 4*time.Second, "bob's change of a held notification")
	if a.err != nil {
		t.Fatal(a.err)
	}
	if took := time.Since(sentAt); took < time.Second {
		t.Errorf("bob's change of a held notification answered %v after it was sent, before the lock timeout, 1s", took)
	}
	w.wantBusy(t, "bob's change of a held notification", req, a.res, a.body)
	release(t, row)
	var archived apiNotification
	decodeAnswer(t, w.send(t, "bob", http.MethodPatch, path, `{"archived":true}`, http.StatusOK), &archived)
	if archived.ArchivedAt == nil || archived.ReadAt == nil || !archived.ReadAt.Equal(*read.ReadAt) {
		t.Errorf("bob's archived notification = %+v; want archived, read as before at %s", archived, read.ReadAt)
	}
}
