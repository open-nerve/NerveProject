package bootstrap

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	notificationdomain "github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// inboxPath is the path of the caller's notifications in acme.
const inboxPath = "/api/v0/workspaces/acme/notifications"

// inboxPage is a page of an account's notifications in acme as the API
// answers it: each notification decoded with every key it is answered
// with, and the next page's cursor, nil on the last.
type inboxPage struct {
	rows []map[string]any
	next *string
}

// inbox is the page of the account name's notifications in acme that
// query asks for: its two keys and no other. The request is checked
// against the contract on a copy, as kin-openapi's check writes the
// contract's defaults into the query it checks (tab, archived, snoozed,
// limit): sent so, a request would never leave a parameter to the
// server's own default, which this test holds.
func (w issueWorld) inbox(t *testing.T, name, query string) inboxPage {
	t.Helper()
	req := newRequest(t, http.MethodGet, w.base+inboxPath+query, w.tokens[name], nil)
	w.contract.CheckRequest(t, req.Clone(req.Context()))
	res, raw := send(t, req)
	w.contract.CheckResponse(t, req, res)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s as %s = %d %s, want 200", inboxPath+query, name, res.StatusCode, raw)
	}
	listKeys(t, raw, "data", "next_cursor")
	var page struct {
		Data []map[string]any `json:"data"`
		Next *string          `json:"next_cursor"`
	}
	decodeAnswer(t, string(raw), &page)
	return inboxPage{page.Data, page.Next}
}

// refusedList is the account bob's list of acme's notifications with
// query, sent without checking it against the contract, which may not
// take it: the answer's status and its refusal (refusalOf).
func (w issueWorld) refusedList(t *testing.T, query string) (int, string) {
	t.Helper()
	req := newRequest(t, http.MethodGet, w.base+inboxPath+query, w.tokens["bob"], nil)
	res, body := send(t, req)
	w.contract.CheckResponse(t, req, res)
	return res.StatusCode, refusalOf(t, body, false)
}

// noteIDs are the ids of rows, in their order; notesWhere, of those keep
// keeps.
func noteIDs(rows []map[string]any) []string {
	return notesWhere(rows, func(map[string]any) bool { return true })
}

func notesWhere(rows []map[string]any, keep func(map[string]any) bool) []string {
	ids := []string{}
	for _, r := range rows {
		if keep(r) {
			id, _ := r["id"].(string)
			ids = append(ids, id)
		}
	}
	return ids
}

// noteChange is the change of a notification as answered.
func noteChange(row map[string]any) map[string]any {
	data, _ := row["data"].(map[string]any)
	change, _ := data["change"].(map[string]any)
	return change
}

// noteKey is a notification's place in the list's order: its moment,
// created_at, and its id.
func noteKey(t *testing.T, row map[string]any) (time.Time, uuid.UUID) {
	t.Helper()
	at, err := time.Parse(time.RFC3339Nano, fmt.Sprint(row["created_at"]))
	if err != nil {
		t.Fatalf("created_at of %v: %v", row, err)
	}
	id, err := uuid.Parse(fmt.Sprint(row["id"]))
	if err != nil {
		t.Fatalf("id of %v: %v", row, err)
	}
	return at, id
}

// wantNotes checks that rows are the notifications want and no other, in
// any order, each with exactly want's keys and values but its id and its
// moment, which the test does not know: so a key answered absent, not
// null, differs.
func wantNotes(t *testing.T, who string, rows []map[string]any, want ...map[string]any) {
	t.Helper()
	shown := func(rows []map[string]any) []string {
		lines := make([]string, len(rows))
		for n, r := range rows {
			kept := maps.Clone(r)
			delete(kept, "id")
			delete(kept, "created_at")
			b, err := json.Marshal(kept)
			if err != nil {
				t.Fatal(err)
			}
			lines[n] = string(b)
		}
		return slices.Sorted(slices.Values(lines))
	}
	if got, wanted := shown(rows), shown(want); !slices.Equal(got, wanted) {
		t.Errorf("%s's notifications\n%s\nwant\n%s", who, strings.Join(got, "\n"), strings.Join(wanted, "\n"))
	}
}

// noteOf is a notification of the work item s, as the API answers it but
// its id and its moment (P1c spec 2.4): sent as sender, triggered by the
// account by, in Web of acme, unread, not archived nor snoozed, s its
// snapshot, its change of field and verb with the keys of set, every other
// key of the change null.
func (w issueWorld) noteOf(sender string, by uuid.UUID, s notificationdomain.IssueSnapshot, field, verb string, set map[string]any) map[string]any {
	change := map[string]any{"field": field, "verb": verb}
	for _, key := range []string{"old_value", "new_value", "old_id", "new_id", "old_issue", "new_issue", "comment_id", "excerpt"} {
		change[key] = nil
	}
	maps.Copy(change, set)
	return map[string]any{"workspace_id": w.acme.String(), "project_id": w.web.String(), "issue_id": s.ID.String(), "sender": sender,
		"triggered_by_id": by.String(), "read_at": nil, "archived_at": nil, "snoozed_till": nil, "data": map[string]any{
			"issue": map[string]any{"id": s.ID.String(), "name": s.Name, "sequence_id": float64(s.SequenceID), "identifier": s.Identifier,
				"state_name": s.StateName, "state_group": s.StateGroup},
			"change": change}}
}

// issueRefOf is the work item s as a change's old_issue or new_issue
// answers it (spec 2.6).
func issueRefOf(s notificationdomain.IssueSnapshot) map[string]any {
	return map[string]any{"id": s.ID.String(), "sequence_id": float64(s.SequenceID), "identifier": s.Identifier, "name": s.Name}
}

// The list of a receiver's notifications through the app (M7 P1c spec 2.3,
// 2.4; H2–H5), of the real writes' notifications once the app's fan-outs
// have run. bob is sent, by alice: Fix's creation assigned to him and his
// assignment; as a subscriber of Move, her one change of its name, parent,
// priority, state and labels, five notifications of one moment; as the
// creator of Talk, her comment; her comment mentioning him, only as a
// mention, in the mentions tab alone; as Fix's assignee, her relation of
// Fix to Move and her link; as a subscriber of Old, its archiving. Each is
// answered with every key, null for what it has not; the ids of an
// assignee, a label or a state as stored, a parent's and a relation's other
// end with its number and name, any other field's none. walt, Talk's
// subscriber, is sent both comments as subscribed; alice, who wrote all of
// that, only bob's comment on her Fix, and bob nothing of that comment nor
// of his creation of Talk. The tab, type, read, archived and snoozed
// filters select as the spec says, those left out by the server's
// defaults; a query it does not take is refused; and pages of three walk
// the list as one page, ties of one moment by id across the pages' edges.
func TestNotificationsListedThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	alice, bob := w.ids["alice"], w.ids["bob"]
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	var me struct {
		DisplayName string `json:"display_name"`
	}
	decodeAnswer(t, w.send(t, "bob", http.MethodGet, "/api/v0/me", "", http.StatusOK), &me)
	todo, done := webState{w.todo, "Todo", "unstarted"}, webState{w.done, "Done", "completed"}
	started := stateID(t, w.pool, w.web, "In Progress")
	parent, fix := w.itemMade(t, "alice", "Parent", todo), w.itemMade(t, "alice", "Fix", todo, bob)
	move, talk, old := w.itemMade(t, "alice", "Fix", todo), w.itemMade(t, "bob", "Talk", todo), w.itemMade(t, "alice", "Old", done)
	// A fan-out reads the work item's subscribers when it runs: the creations' have run before anyone else subscribes.
	waitForFanouts(t, w.pool)

	for _, s := range []struct {
		watcher string
		item    uuid.UUID
	}{{"bob", move.ID}, {"bob", old.ID}, {"walt", talk.ID}} {
		w.send(t, s.watcher, http.MethodPost, "/api/v0/issues/"+s.item.String()+"/subscribe", "", http.StatusNoContent)
	}
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+move.ID.String(), fmt.Sprintf(
		`{"name":"Fix it","parent_id":%q,"priority":"high","state_id":%q,"label_ids":[%q]}`, parent.ID, started, w.bug), http.StatusOK)
	comments := make([]answeredComment, 3)
	for n, c := range []struct {
		by   string
		item uuid.UUID
		html string
	}{{"alice", talk.ID, "<p>Looks good</p>"}, {"alice", talk.ID, mentionHTML(bob)}, {"bob", fix.ID, "<p>On it</p>"}} {
		body := fmt.Sprintf(`{"comment_html":%q}`, c.html)
		decodeAnswer(t, w.send(t, c.by, http.MethodPost, commentsOf(c.item), body, http.StatusCreated), &comments[n])
	}
	w.send(t, "alice", http.MethodPost, relationsOf(fix.ID), relating("relates_to", move.ID), http.StatusCreated)
	address := "http://169.254.169.254/notes"
	createLink(t, w, "alice", fix.ID, address)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+old.ID.String()+"/archive", "", http.StatusOK)
	waitForFanouts(t, w.pool)

	moved := move
	moved.Name, moved.StateName, moved.StateGroup = "Fix it", "In Progress", "started"
	looks, hello, onIt := comments[0].ID.String(), comments[1].ID.String(), comments[2].ID.String()
	bobs := []map[string]any{
		w.noteOf("assigned", alice, fix, "issue", "created", nil),
		w.noteOf("assigned", alice, fix, "assignees", "created", map[string]any{"new_value": me.DisplayName, "new_id": bob.String()}),
		w.noteOf("subscribed", alice, moved, "name", "updated", map[string]any{"old_value": "Fix", "new_value": "Fix it"}),
		w.noteOf("subscribed", alice, moved, "parent", "updated", map[string]any{"new_id": parent.ID.String(), "new_issue": issueRefOf(parent)}),
		w.noteOf("subscribed", alice, moved, "priority", "updated", map[string]any{"old_value": "none", "new_value": "high"}),
		w.noteOf("subscribed", alice, moved, "state", "updated", map[string]any{"old_value": "Todo", "new_value": "In Progress",
			"old_id": w.todo.String(), "new_id": started.String()}),
		w.noteOf("subscribed", alice, moved, "labels", "updated", map[string]any{"old_value": "", "new_value": "Bug", "new_id": w.bug.String()}),
		w.noteOf("created", alice, talk, "comment", "created", map[string]any{"comment_id": looks, "excerpt": "Looks good"}),
		w.noteOf("assigned", alice, fix, "relates_to", "created", map[string]any{"new_id": move.ID.String(), "new_issue": issueRefOf(moved)}),
		w.noteOf("assigned", alice, fix, "link", "created", map[string]any{"new_value": address}),
		w.noteOf("subscribed", alice, old, "archived_at", "updated", map[string]any{"new_value": "manual_archive"}),
	}
	mention := w.noteOf("mentioned", alice, talk, "comment", "mentioned", map[string]any{"comment_id": hello, "excerpt": "Hello"})
	all, mentions := w.inbox(t, "bob", ""), w.inbox(t, "bob", "?tab=mentions")

	t.Run("each write's notification", func(t *testing.T) {
		wantNotes(t, "bob's all tab", all.rows, bobs...)
		wantNotes(t, "bob's mentions tab", mentions.rows, mention)
		if all.next != nil || mentions.next != nil {
			t.Errorf("bob's tabs' next cursors %v and %v; want each a last page", all.next, mentions.next)
		}
	})
	t.Run("the actor is sent nothing he wrote", func(t *testing.T) {
		wantNotes(t, "alice", w.inbox(t, "alice", "").rows,
			w.noteOf("created", bob, fix, "comment", "created", map[string]any{"comment_id": onIt, "excerpt": "On it"}))
		for _, name := range []string{"alice", "bob", "walt"} {
			for _, tab := range []string{"", "?tab=mentions"} {
				if own := notesWhere(w.inbox(t, name, tab).rows, func(r map[string]any) bool { return r["triggered_by_id"] == w.ids[name].String() }); len(own) > 0 {
					t.Errorf("%s is sent %v, of his own writes", name, own)
				}
			}
		}
	})
	t.Run("a mention and a subscription", func(t *testing.T) {
		wantNotes(t, "walt", w.inbox(t, "walt", "").rows,
			w.noteOf("subscribed", alice, talk, "comment", "created", map[string]any{"comment_id": looks, "excerpt": "Looks good"}),
			w.noteOf("subscribed", alice, talk, "comment", "created", map[string]any{"comment_id": hello, "excerpt": "Hello"}))
		ofHello := func(r map[string]any) bool { return noteChange(r)["comment_id"] == hello }
		if got, mentioned := append(notesWhere(all.rows, ofHello), notesWhere(mentions.rows, ofHello)...), notesWhere(mentions.rows, ofHello); len(got) != 1 ||
			!slices.Equal(got, mentioned) {
			t.Errorf("bob, Talk's creator, is sent %v of the comment mentioning him; want its mention alone, of %v", got, mentioned)
		}
	})
	t.Run("tabs and types", func(t *testing.T) {
		of := func(senders ...string) []string {
			return notesWhere(all.rows, func(r map[string]any) bool { return slices.Contains(senders, fmt.Sprint(r["sender"])) })
		}
		for query, want := range map[string][]string{
			"?tab=all": noteIDs(all.rows), "?type=assigned": of("assigned"), "?tab=all&type=created,subscribed": of("created", "subscribed"),
			"?type=subscribed": of("subscribed"), "?type=subscribed,assigned,created": noteIDs(all.rows),
		} {
			if got := w.inbox(t, "bob", query); !slices.Equal(noteIDs(got.rows), want) || got.next != nil {
				t.Errorf("bob's %s = %v, next %v; want %v, the last page", query, noteIDs(got.rows), got.next, want)
			}
		}
		if senders := notesWhere(all.rows, func(r map[string]any) bool { return r["sender"] == "mentioned" }); len(senders) > 0 || len(mentions.rows) != 1 {
			t.Errorf("bob's all tab lists the mentions %v; want none, and the mentions tab its one", senders)
		}
	})
	t.Run("refused queries", func(t *testing.T) {
		for _, c := range []struct {
			query, want string
			status      int
		}{
			{"?tab=mentions&type=assigned", "validation_failed type not_allowed", 422},
			{"?type=assigned,assigned", "validation_failed type invalid_format", 422},
			{"?type=mentioned", "validation_failed type invalid_format", 422},
			{"?type=assigned,", "validation_failed type invalid_format", 422},
			{"?tab=inbox", "validation_failed tab invalid_format", 422},
			{"?read=yes", "validation_failed read invalid_format", 422},
			{"?archived=1", "validation_failed archived invalid_format", 422},
			{"?snoozed=maybe", "validation_failed snoozed invalid_format", 422},
			{"?unread=true", "validation_failed unread not_allowed", 422},
			{"?tab=all&tab=mentions", "validation_failed tab invalid_format", 422},
			{"?limit=0", "validation_failed limit out_of_range", 422},
			{"?limit=101", "validation_failed limit out_of_range", 422},
			{"?cursor=x", "bad_request cursor invalid_format", 400},
		} {
			if status, got := w.refusedList(t, c.query); status != c.status || got != c.want {
				t.Errorf("bob's %s = %d %s; want %d %s", c.query, status, got, c.status, c.want)
			}
		}
	})
	t.Run("pages", func(t *testing.T) {
		var walked []map[string]any
		query, edgeTies := "?limit=3", 0
		for page := 1; ; page++ {
			if page > len(all.rows) {
				t.Fatalf("bob's pages of three go on past %d: %v", len(all.rows), noteIDs(walked))
			}
			got := w.inbox(t, "bob", query)
			if len(walked) > 0 && len(got.rows) > 0 {
				before, _ := noteKey(t, walked[len(walked)-1])
				if after, _ := noteKey(t, got.rows[0]); after.Equal(before) {
					edgeTies++
				}
			}
			walked = append(walked, got.rows...)
			if got.next == nil {
				break
			}
			if len(got.rows) != 3 {
				t.Fatalf("bob's page %d of three has %d and a cursor", page, len(got.rows))
			}
			query = "?limit=3&cursor=" + url.QueryEscape(*got.next)
		}
		if !slices.Equal(noteIDs(walked), noteIDs(all.rows)) {
			t.Errorf("bob's pages of three walk %v; want %v, his one page", noteIDs(walked), noteIDs(all.rows))
		}
		if edgeTies == 0 {
			t.Errorf("no edge of bob's pages of three falls between notifications of one moment")
		}
		for n := 1; n < len(all.rows); n++ {
			at, id := noteKey(t, all.rows[n-1])
			if nextAt, nextID := noteKey(t, all.rows[n]); nextAt.After(at) || (nextAt.Equal(at) && nextID.Compare(id) >= 0) {
				t.Errorf("bob's notification %s at %s lists before %s at %s; want by created_at, then id, both descending", id, at, nextID, nextAt)
			}
		}
		for _, event := range []struct {
			what  string
			item  notificationdomain.IssueSnapshot
			field []string
		}{{"Move's change", move, []string{"name", "parent", "priority", "state", "labels"}}, {"Fix's creation", fix, []string{"issue", "assignees"}}} {
			var of []uuid.UUID
			moments := map[time.Time]bool{}
			for _, r := range all.rows {
				if r["issue_id"] == event.item.ID.String() && slices.Contains(event.field, fmt.Sprint(noteChange(r)["field"])) {
					at, id := noteKey(t, r)
					of, moments[at] = append(of, id), true
				}
			}
			if len(of) != len(event.field) || len(moments) != 1 {
				t.Errorf("%s's notifications to bob %v are of the moments %v; want %d of one", event.what, of, moments, len(event.field))
			}
		}
	})

	pick := func(field string) string {
		picked := notesWhere(all.rows, func(r map[string]any) bool { return noteChange(r)["field"] == field })
		if len(picked) != 1 {
			t.Fatalf("bob's all tab has %v of %s; want one", picked, field)
		}
		return picked[0]
	}
	link, relation, comment := pick("link"), pick("relates_to"), pick("comment")
	without := func(drop ...string) []string {
		return notesWhere(all.rows, func(r map[string]any) bool { return !slices.Contains(drop, fmt.Sprint(r["id"])) })
	}
	listsAs := func(t *testing.T, want map[string][]string) {
		t.Helper()
		for query, ids := range want {
			if got := noteIDs(w.inbox(t, "bob", query).rows); !slices.Equal(got, ids) {
				t.Errorf("bob's %q = %v; want %v", query, got, ids)
			}
		}
	}
	// Each of the three below changes one of bob's notifications, which the next keeps.
	t.Run("read", func(t *testing.T) {
		w.send(t, "bob", http.MethodPatch, "/api/v0/notifications/"+link, `{"read":true}`, http.StatusOK)
		listsAs(t, map[string][]string{"": noteIDs(all.rows), "?read=true": {link}, "?read=false": without(link),
			"?read=false&type=assigned": notesWhere(all.rows, func(r map[string]any) bool { return r["sender"] == "assigned" && r["id"] != link })})
	})
	t.Run("archived", func(t *testing.T) {
		w.send(t, "bob", http.MethodPatch, "/api/v0/notifications/"+relation, `{"archived":true}`, http.StatusOK)
		listsAs(t, map[string][]string{"": without(relation), "?archived=false": without(relation), "?archived=true": {relation},
			"?archived=true&read=true": {}})
	})
	t.Run("snoozed", func(t *testing.T) {
		till := time.Now().Add(time.Hour).UTC().Format(time.RFC3339Nano)
		w.send(t, "bob", http.MethodPatch, "/api/v0/notifications/"+comment, `{"snoozed_till":"`+till+`"}`, http.StatusOK)
		listsAs(t, map[string][]string{"": without(relation, comment), "?snoozed=false": without(relation, comment), "?snoozed=true": {comment},
			"?snoozed=true&type=subscribed": {}, "?archived=true": {relation}, "?tab=mentions": noteIDs(mentions.rows)})
	})
}
