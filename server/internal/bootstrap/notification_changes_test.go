package bootstrap

import (
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// notePath is the address of the change of the notification id.
func notePath(id string) string { return "/api/v0/notifications/" + id }

// changeNote is the account name's change of his notification id with
// body, answered 200: the notification as answered, with every key.
func (w issueWorld) changeNote(t *testing.T, name, id, body string) map[string]any {
	t.Helper()
	var note map[string]any
	decodeAnswer(t, w.send(t, name, http.MethodPatch, notePath(id), body, http.StatusOK), &note)
	return note
}

// noteIDOf is the id of the account name's one notification, deleted or
// not, of the work item item whose change is change, its field and its
// verb apart by a space.
func (w issueWorld) noteIDOf(t *testing.T, name string, item uuid.UUID, change string) string {
	t.Helper()
	ids := queryIDs(t, w.pool, `SELECT id FROM notifications WHERE receiver_id = $1 AND issue_id = $2
		AND concat_ws(' ', data->'change'->>'field', data->'change'->>'verb') = $3`, w.ids[name], item, change)
	if len(ids) != 1 {
		t.Fatalf("%s's notifications of %s's %s: %v; want one", name, item, change, ids)
	}
	return ids[0].String()
}

// storedNote is the row of the notification id as stored: its three
// states, and the moment and the writer of its last write, each as
// "name=value", null for none.
func (w issueWorld) storedNote(t *testing.T, id string) string {
	t.Helper()
	var read, archived, snoozed *time.Time
	var updated time.Time
	var by *uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT read_at, archived_at, snoozed_till, updated_at, updated_by_id FROM notifications WHERE id = $1`,
		id).Scan(&read, &archived, &snoozed, &updated, &by); err != nil {
		t.Fatal(err)
	}
	at := func(t *time.Time) string {
		if t == nil {
			return "null"
		}
		return t.UTC().Format(time.RFC3339Nano)
	}
	writer := "null"
	if by != nil {
		writer = w.names()[*by]
	}
	return fmt.Sprintf("read_at=%s archived_at=%s snoozed_till=%s updated_at=%s updated_by=%s", at(read), at(archived), at(snoozed), at(&updated), writer)
}

// answeredAt is a moment as an answer gives it, in RFC 3339.
func answeredAt(t *testing.T, value any) time.Time {
	t.Helper()
	at, err := time.Parse(time.RFC3339Nano, fmt.Sprint(value))
	if err != nil {
		t.Fatalf("the moment %v: %v", value, err)
	}
	return at
}

// snoozing is a change's body snoozing a notification till at.
func snoozing(at time.Time) string {
	return fmt.Sprintf(`{"snoozed_till":%q}`, at.Format(time.RFC3339Nano))
}

// wholeChange is the whole answer to the account name's change of the
// notification id with body: its status, its body and every header but
// Date, the transport's clock. The request carries one fixed
// X-Request-Id, which the answer echoes, so that two such answers can
// differ only by what they answer (the asset refusals' way of comparing a
// hidden one with a missing one).
func (w issueWorld) wholeChange(t *testing.T, name, id, body string) (int, string, http.Header) {
	t.Helper()
	req := newRequest(t, http.MethodPatch, w.base+notePath(id), w.tokens[name], []byte(body))
	req.Header.Set("X-Request-Id", "00000000-0000-4000-8000-000000000001")
	w.contract.CheckRequest(t, req)
	res, raw := send(t, req)
	w.contract.CheckResponse(t, req, res)
	header := res.Header.Clone()
	header.Del("Date")
	return res.StatusCode, string(raw), header
}

// A receiver's change of his notification through the app (M7 P1c spec
// 2.7, R5, R7; H6). bob is sent, by alice, the creation of Fix and of Two
// assigned to him, and of Gone, of Far in Ops and of Away in Site of beta,
// all assigned to him; alice, his comment on her Fix. read true stamps
// read_at with the change's moment and the change's writer, keeps it as it
// is when it is set, writing nothing, and false clears it; archived the
// same. snoozed_till stores a moment after now as the microsecond it is,
// refuses one before now 422 must_be_future and clears for null. {}
// answers the notification and writes nothing; read null breaks the body.
// Each state changes alone. alice's notification, his of Gone once she
// deletes it, of Away once beta no longer has him and of Far once he left
// Ops are each answered the very answer of an id that never was, and so
// with a snoozed_till before now too: never the 422. On a clock that
// stands still, the module wired as bootstrap wires it refuses a
// snoozed_till at now, or within its microsecond, and takes one a
// microsecond after it, and once the clock is at that moment the snooze
// is over: the notification is listed, and counted unread, again.
func TestChangingANotificationThroughTheApp(t *testing.T) {
	var cfg config.Config
	w := newIssueWorldWith(t, func(c *config.Config) { cfg = *c })
	alice, bob := w.ids["alice"], w.ids["bob"]
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.makeMember(t, "bob", ops, shared.RoleMember)
	w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Beta","slug":"beta"}`, http.StatusCreated)
	site := createdProject(t, w.contract, w.base, w.tokens["alice"], "beta", "Site", "SITE")
	beta := inWorkspaceOf(t, w.pool, site, bob, alice, shared.RoleMember)
	w.makeMember(t, "bob", site, shared.RoleMember)
	items := map[string]uuid.UUID{}
	for _, m := range []struct {
		name    string
		project uuid.UUID
	}{{"Fix", w.web}, {"Two", w.web}, {"Gone", w.web}, {"Far", ops}, {"Away", site}} {
		items[m.name] = w.newWorkItemIn(t, m.project, fmt.Sprintf(`{"name":%q,"assignee_ids":%s}`, m.name, idArray(bob)))
	}
	w.send(t, "bob", http.MethodPost, commentsOf(items["Fix"]), `{"comment_html":"<p>On it</p>"}`, http.StatusCreated)
	waitForFanouts(t, w.pool)
	snoozed, kept := w.noteIDOf(t, "bob", items["Two"], "issue created"), w.noteIDOf(t, "bob", items["Two"], "assignees created")

	t.Run("read and archived", func(t *testing.T) {
		for _, s := range []struct{ field, key, id string }{
			{"read", "read_at", w.noteIDOf(t, "bob", items["Fix"], "issue created")},
			{"archived", "archived_at", w.noteIDOf(t, "bob", items["Fix"], "assignees created")},
		} {
			before := time.Now().Truncate(time.Microsecond)
			set := w.changeNote(t, "bob", s.id, `{"`+s.field+`":true}`)
			at := answeredAt(t, set[s.key])
			stamp := at.UTC().Format(time.RFC3339Nano)
			stored := w.storedNote(t, s.id)
			if at.Before(before) || at.After(time.Now()) || !strings.Contains(stored, s.key+"="+stamp) ||
				!strings.HasSuffix(stored, "updated_at="+stamp+" updated_by=bob") {
				t.Errorf("bob's %s true answers %s %s, stored %s; want the change's moment, after %s, written by him then", s.field, s.key, stamp, stored, before)
			}
			if again := w.changeNote(t, "bob", s.id, `{"`+s.field+`":true}`); !reflect.DeepEqual(again, set) || w.storedNote(t, s.id) != stored {
				t.Errorf("bob's %s true again answers %v, stored %s; want %v, stored %s: kept as it was, nothing written", s.field, again, w.storedNote(t, s.id), set,
					stored)
			}
			cleared := w.changeNote(t, "bob", s.id, `{"`+s.field+`":false}`)
			if now := w.storedNote(t, s.id); cleared[s.key] != nil || !strings.Contains(now, s.key+"=null") || !strings.HasSuffix(now, "updated_by=bob") ||
				strings.Contains(now, "updated_at="+stamp) {
				t.Errorf("bob's %s false answers %s %v, stored %s; want it cleared, written by him later", s.field, s.key, cleared[s.key], now)
			}
		}
	})
	t.Run("snoozed_till", func(t *testing.T) {
		till := time.Now().Add(time.Hour).UTC().Truncate(time.Second).Add(123456789 * time.Nanosecond)
		micro := till.Truncate(time.Microsecond).Format(time.RFC3339Nano)
		note := w.changeNote(t, "bob", snoozed, snoozing(till))
		stored := w.storedNote(t, snoozed)
		if !strings.HasSuffix(micro, ".123456Z") || note["snoozed_till"] != micro || !strings.Contains(stored, "snoozed_till="+micro) ||
			!strings.HasSuffix(stored, "updated_by=bob") {
			t.Errorf("bob's snooze till %s answers %v, stored %s; want %s, the microsecond, written by him", till.Format(time.RFC3339Nano), note["snoozed_till"],
				stored, micro)
		}
		status, body := call(t, w.contract, http.MethodPatch, w.base+notePath(snoozed), w.tokens["bob"], snoozing(time.Now().Add(-time.Hour)))
		if got := refusalOf(t, []byte(body), false); status != http.StatusUnprocessableEntity || got != "validation_failed snoozed_till must_be_future" ||
			w.storedNote(t, snoozed) != stored {
			t.Errorf("bob's snooze till an hour ago = %d %s, stored %s; want 422 snoozed_till must_be_future, stored %s", status, got, w.storedNote(t, snoozed), stored)
		}
		if note := w.changeNote(t, "bob", snoozed, `{"snoozed_till":null}`); note["snoozed_till"] != nil ||
			!strings.Contains(w.storedNote(t, snoozed), "snoozed_till=null") {
			t.Errorf("bob's snooze null answers %v, stored %s; want it cleared", note["snoozed_till"], w.storedNote(t, snoozed))
		}
	})
	t.Run("nothing to change, and a broken body", func(t *testing.T) {
		before := w.storedNote(t, kept)
		var row map[string]any
		for _, r := range w.inbox(t, "bob", "").rows {
			if r["id"] == kept {
				row = r
			}
		}
		if note := w.changeNote(t, "bob", kept, `{}`); row == nil || !reflect.DeepEqual(note, row) || w.storedNote(t, kept) != before {
			t.Errorf("bob's {} answers %v, stored %s; want %v as listed, stored %s: nothing written", note, w.storedNote(t, kept), row, before)
		}
		req := newRequest(t, http.MethodPatch, w.base+notePath(kept), w.tokens["bob"], []byte(`{"read":null}`))
		res, body := send(t, req)
		w.contract.CheckResponse(t, req, res)
		if got := refusalOf(t, body, false); res.StatusCode != http.StatusBadRequest || got != "bad_request read invalid_format" ||
			w.storedNote(t, kept) != before {
			t.Errorf("bob's read null = %d %s, stored %s; want 400 bad_request read invalid_format, stored %s", res.StatusCode, got, w.storedNote(t, kept), before)
		}
	})
	t.Run("each state alone", func(t *testing.T) {
		till := time.Now().Add(time.Hour)
		last := w.changeNote(t, "bob", kept, fmt.Sprintf(`{"read":true,"snoozed_till":%q}`, till.Format(time.RFC3339Nano)))
		for _, c := range []struct{ body, key string }{
			{`{"archived":true}`, "archived_at"}, {`{"read":false}`, "read_at"}, {`{"snoozed_till":null}`, "snoozed_till"}, {`{"archived":false}`, "archived_at"},
		} {
			note := w.changeNote(t, "bob", kept, c.body)
			want := maps.Clone(last)
			want[c.key] = note[c.key]
			if reflect.DeepEqual(note[c.key], last[c.key]) || !reflect.DeepEqual(note, want) {
				t.Errorf("bob's %s answers %v; want %v but its %s changed", c.body, note, last, c.key)
			}
			last = note
		}
	})
	t.Run("refused alike", func(t *testing.T) {
		w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+items["Gone"].String(), "", http.StatusNoContent)
		w.send(t, "alice", http.MethodDelete, "/api/v0/workspace-members/"+w.memberRow(t, "workspace_members", "workspace_id", beta, "bob"), "",
			http.StatusNoContent)
		w.send(t, "bob", http.MethodPost, "/api/v0/projects/"+ops.String()+"/leave", "", http.StatusNoContent)
		hidden := []struct{ what, id, row string }{
			{"alice's", w.noteIDOf(t, "alice", items["Fix"], "comment created"), "alice false"},
			{"deleted", w.noteIDOf(t, "bob", items["Gone"], "issue created"), "bob true"},
			{"of beta, which has him no more", w.noteIDOf(t, "bob", items["Away"], "issue created"), "bob false"},
			{"of Ops, which he left", w.noteIDOf(t, "bob", items["Far"], "issue created"), "bob false"},
		}
		ids := make([]string, len(hidden))
		for n, h := range hidden {
			var receiver uuid.UUID
			var deleted bool
			if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT receiver_id, deleted_at IS NOT NULL FROM notifications WHERE id = $1`, h.id).Scan(&receiver,
				&deleted); err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprintf("%s %t", w.names()[receiver], deleted); got != h.row {
				t.Fatalf("the notification %s, %s, is the receiver's and deleted %s; want %s", h.id, h.what, got, h.row)
			}
			ids[n] = h.id
		}
		for _, body := range []string{`{"read":true}`, snoozing(time.Now().Add(-time.Hour))} {
			status, answer, header := w.wholeChange(t, "bob", uuid.NewV7().String(), body)
			if got := refusalOf(t, []byte(answer), false); status != http.StatusNotFound || got != "notification.not_found" {
				t.Fatalf("bob's %s of an id that never was = %d %s; want 404 notification.not_found", body, status, answer)
			}
			for _, h := range hidden {
				if gotStatus, got, gotHeader := w.wholeChange(t, "bob", h.id, body); gotStatus != status || got != answer || !reflect.DeepEqual(gotHeader, header) {
					t.Errorf("bob's %s of the notification %s = %d %s %v; want %d %s %v, an id that never was's", body, h.what, gotStatus, got, gotHeader, status,
						answer, header)
				}
			}
		}
		var written int
		if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*) FROM notifications WHERE id = ANY($1::uuid[])
			AND (read_at IS NOT NULL OR snoozed_till IS NOT NULL OR updated_by_id = $2)`, ids, bob).Scan(&written); err != nil || written != 0 {
			t.Errorf("the refused changes wrote %d of the notifications, %v; want none", written, err)
		}
	})
	t.Run("at the business clock's moment", func(t *testing.T) {
		clock := clockAt(time.Now())
		deps := notificationDeps(w.pool, cfg)
		deps.Clock = clock
		route := newRouteWithTimeout(t, notification.New(deps).Register, cfg.Server.RequestTimeout)
		listed := func(query string) bool {
			var page notificationPage
			decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodGet, inboxPath+query, bob, "", http.StatusOK), &page)
			return slices.ContainsFunc(page.Data, func(n apiNotification) bool { return n.ID.String() == snoozed })
		}
		unread := func() int {
			var counts unreadCounts
			decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodGet, inboxPath+"/unread-count", bob, "", http.StatusOK), &counts)
			return counts.All
		}
		now := clock.Now()
		for _, at := range []time.Time{now.Add(-time.Microsecond), now, now.Add(999 * time.Nanosecond)} {
			answer := route.answerWithin(t, w.contract, http.MethodPatch, notePath(snoozed), bob, snoozing(at), http.StatusUnprocessableEntity)
			if got := refusalOf(t, []byte(answer), false); got != "validation_failed snoozed_till must_be_future" {
				t.Errorf("bob's snooze till %s, the clock at %s = %s; want 422 snoozed_till must_be_future", at.Format(time.RFC3339Nano),
					now.Format(time.RFC3339Nano), got)
			}
		}
		till := now.Add(time.Microsecond)
		var note map[string]any
		decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPatch, notePath(snoozed), bob, snoozing(till), http.StatusOK), &note)
		snoozedUnread := unread()
		// The route's pool, the test's, scans a moment in time.Local, not in UTC as the app's does: the instant is compared.
		if !answeredAt(t, note["snoozed_till"]).Equal(till) || listed("") || !listed("?snoozed=true") {
			t.Errorf("bob's snooze till a microsecond on answers %v; want %s, listed as snoozed alone", note["snoozed_till"], till.Format(time.RFC3339Nano))
		}
		clock.advance(time.Microsecond)
		if got := unread(); got != snoozedUnread+1 || !listed("") || listed("?snoozed=true") {
			t.Errorf("the clock at the snooze's end: bob's unread %d, was %d; want it counted again, listed as not snoozed alone", got, snoozedUnread)
		}
	})
}
