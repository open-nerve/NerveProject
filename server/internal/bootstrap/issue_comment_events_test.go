package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func wantCollaborationEnvelope(t *testing.T, w issueWorld, e shared.Event, kind, entity string, id, actor uuid.UUID, at time.Time, changes []shared.Change) {
	t.Helper()
	if e.ID == (uuid.UUID{}) || e.ID == id || e.Type != kind || e.Entity != (shared.EntityRef{Kind: entity, ID: id}) || e.WorkspaceID != w.acme || e.ProjectID != w.web || e.ActorID == nil || *e.ActorID != actor || !e.OccurredAt.Equal(at) || !reflect.DeepEqual(e.Changes, changes) {
		t.Errorf("event=%+v changes=%+v want=%+v", e, e.Changes, changes)
	}
}

func TestCommentEventsComposition(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItem(t)
	at := time.Date(2044, 5, 6, 7, 8, 9, 123456000, time.UTC)
	c := clockAt(at)
	clock := &eventCountingClock{base: c}
	var events []shared.Event
	var failure error
	route := issueEventRouteFor(t, w, clock, &events, &failure, issue.EventTypes(), func(ctx context.Context, e shared.Event) error {
		var html string
		var deleted bool
		if err := postgres.DB(ctx, w.pool).QueryRow(ctx, "SELECT comment_html,deleted_at IS NOT NULL FROM issue_comments WHERE id=$1", e.Entity.ID).Scan(&html, &deleted); err != nil {
			return err
		}
		wantHTML := "<p>Clean</p>"
		if e.Type != "issue.comment_created" {
			wantHTML = "<p>Edited</p>"
		}
		if html != wantHTML || deleted != (e.Type == "issue.comment_deleted") {
			t.Error("comment event preceded business write", html, deleted)
		}
		if deleted {
			var live int
			if err := postgres.DB(ctx, w.pool).QueryRow(ctx, "SELECT count(*) FROM comment_reactions WHERE comment_id=$1 AND deleted_at IS NULL", e.Entity.ID).Scan(&live); err != nil {
				return err
			}
			if live != 0 {
				t.Error("comment event preceded child reaction deletion")
			}
		}
		var stamp bool
		if err := postgres.DB(ctx, w.pool).QueryRow(ctx, "SELECT updated_at=$2 AND updated_by_id=$3 FROM issues WHERE id=$1", id, e.OccurredAt, e.ActorID).Scan(&stamp); err != nil {
			return err
		}
		if !stamp {
			t.Error("comment event did not share issue audit stamp")
		}
		return nil
	})
	var comment answeredComment
	decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPost, commentsOf(id), w.ids["bob"], `{"comment_html":"<p>Clean</p><script>bad()</script>"}`, 201), &comment)
	c.advance(time.Minute)
	route.answerWithin(t, w.contract, http.MethodPatch, commentPath(comment.ID), w.ids["bob"], `{"comment_html":"<p>Edited</p><script>bad()</script>"}`, 200)
	before := collaborationRows(t, w, id)
	route.answerWithin(t, w.contract, http.MethodPatch, commentPath(comment.ID), w.ids["bob"], `{"comment_html":"<p>Edited</p><script>different()</script>"}`, 200)
	if !maps.Equal(before, collaborationRows(t, w, id)) || len(events) != 2 || clock.calls.Load() != 2 {
		t.Fatal("same sanitized HTML wrote/published/read clock")
	}
	addCommentReaction(t, w, "bob", comment.ID, "128077")
	c.advance(time.Minute)
	route.answerWithin(t, w.contract, http.MethodDelete, commentPath(comment.ID), w.ids["alice"], "", 204)
	if len(events) != 3 || issueEventCount(t, w) != 3 || clock.calls.Load() != 3 {
		t.Fatalf("comment event/job/clock count=%d/%d/%d", len(events), issueEventCount(t, w), clock.calls.Load())
	}
	changes := [][]shared.Change{
		{{Field: "comment_html", New: eventValue("<p>Clean</p>")}, {Field: "issue", NewID: &id}},
		{{Field: "comment_html", Old: eventValue("<p>Clean</p>"), New: eventValue("<p>Edited</p>")}, {Field: "issue", NewID: &id}},
		{{Field: "comment_html", Old: eventValue("<p>Edited</p>")}, {Field: "issue", OldID: &id}},
	}
	types := []string{"issue.comment_created", "issue.comment_updated", "issue.comment_deleted"}
	seen := map[uuid.UUID]bool{}
	for n, e := range events {
		actor := w.ids["bob"]
		if n == 2 {
			actor = w.ids["alice"]
		}
		wantCollaborationEnvelope(t, w, e, types[n], "issue_comment", comment.ID, actor, at.Add(time.Duration(n)*time.Minute), changes[n])
		if seen[e.ID] {
			t.Error("event ID reused")
		}
		seen[e.ID] = true
	}
}

// A comment's mentions in its events (M7 P1a spec 2.2): its creation's are
// the members its body mentions, walt, a guest who sees only his own, among
// them, olga, no member, not, in id order, before the work item; an edit's
// only those the body it replaces does not mention, an account that is no
// one's not among them, and none when it mentions no one new. An edit
// subscribes only the caller and the visible newly mentioned (R7): bob,
// still mentioned after his subscription row was deleted (as P1a's
// unsubscribe deletes it), is not subscribed again; wanda is.
func TestCommentEventMentions(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	w.makeMember(t, "wanda", w.web, shared.RoleAdmin)
	id := w.newWorkItemIn(t, w.web, `{"name":"Comments"}`)
	var events []shared.Event
	var failure error
	route := issueEventRouteFor(t, w, clockAt(time.Date(2044, 5, 6, 7, 8, 9, 0, time.UTC)), &events, &failure, issue.EventTypes(), nil)
	mentions := func(names ...string) []shared.Change {
		ids := make([]uuid.UUID, 0, len(names))
		for _, name := range names {
			ids = append(ids, w.ids[name])
		}
		slices.SortFunc(ids, uuid.UUID.Compare)
		changes := make([]shared.Change, 0, len(ids)+1)
		for _, account := range ids {
			changes = append(changes, shared.Change{Field: "mention", NewID: &account})
		}
		return append(changes, shared.Change{Field: "issue", NewID: &id})
	}
	var c answeredComment
	decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPost, commentsOf(id), w.ids["alice"], fmt.Sprintf(`{"comment_html":%q}`, mentionHTML(w.ids["olga"], w.ids["walt"], w.ids["bob"], w.ids["bob"])), 201), &c)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issue_subscribers SET deleted_at=now() WHERE issue_id=$1 AND subscriber_id=$2", id, w.ids["bob"]); err != nil {
		t.Fatal(err)
	}
	stranger := uuid.NewV7()
	for _, html := range []string{mentionHTML(w.ids["bob"], stranger, w.ids["olga"], w.ids["wanda"], w.ids["walt"]), mentionHTML(w.ids["bob"])} {
		route.answerWithin(t, w.contract, http.MethodPatch, commentPath(c.ID), w.ids["alice"], fmt.Sprintf(`{"comment_html":%q}`, html), 200)
	}
	if len(events) != 3 {
		t.Fatalf("comment events=%d, want three", len(events))
	}
	for n, want := range [][]shared.Change{mentions("bob", "walt"), mentions("wanda"), mentions()} {
		if e := events[n]; len(e.Changes) != len(want)+1 || e.Changes[0].Field != "comment_html" || !reflect.DeepEqual(e.Changes[1:], want) {
			t.Errorf("comment event %d changes=%+v, want comment_html then %+v", n, e.Changes, want)
		}
	}
	want := []uuid.UUID{w.ids["alice"], w.ids["wanda"]}
	slices.SortFunc(want, uuid.UUID.Compare)
	if got := queryIDs(t, w.pool, "SELECT subscriber_id FROM issue_subscribers WHERE issue_id=$1 AND deleted_at IS NULL ORDER BY subscriber_id", id); !slices.Equal(got, want) {
		t.Fatalf("subscribers=%v, want %v: an edit subscribes only the visible newly mentioned", got, want)
	}
}

func TestCommentEventsPublicationRollback(t *testing.T) {
	for _, operation := range []string{"create comment", "update comment", "delete comment", "add issue reaction", "remove issue reaction", "add comment reaction", "remove comment reaction"} {
		t.Run(operation, func(t *testing.T) {
			w := newIssueWorld(t)
			id := w.newWorkItem(t)
			comment := w.newComment(t, "bob", id)
			addCommentReaction(t, w, "bob", comment.ID, "128077")
			w.send(t, "bob", http.MethodPost, reactionsOf(id), `{"reaction":"128077"}`, 201)
			before := collaborationRows(t, w, id)
			at := time.Date(2044, 5, 6, 7, 8, 9, 123456000, time.UTC)
			var events []shared.Event
			failure := errors.New("subscriber rejected after real River insert")
			route := issueEventRouteFor(t, w, clockAt(at), &events, &failure, issue.EventTypes(), func(ctx context.Context, e shared.Event) error {
				// This read uses the request transaction: the subscriber must see all SQL
				// completed even though a separate pool cannot see its River insert yet.
				table := "issue_comments"
				switch e.Entity.Kind {
				case "issue_reaction":
					table = "issue_reactions"
				case "comment_reaction":
					table = "comment_reactions"
				}
				var complete bool
				if err := postgres.DB(ctx, w.pool).QueryRow(ctx, "SELECT updated_at=$2 AND updated_by_id=$3 FROM "+table+" WHERE id=$1", e.Entity.ID, at, w.ids["bob"]).Scan(&complete); err != nil {
					return err
				}
				if !complete {
					t.Error("publication preceded completed business SQL")
				}
				return nil
			})
			method, path, body := http.MethodPost, commentsOf(id), `{"comment_html":"<p>Rejected</p>"}`
			switch operation {
			case "update comment":
				method, path = http.MethodPatch, commentPath(comment.ID)
			case "delete comment":
				method, path, body = http.MethodDelete, commentPath(comment.ID), ""
			case "add issue reaction":
				path, body = reactionsOf(id), `{"reaction":"10084-65039"}`
			case "remove issue reaction":
				method, path, body = http.MethodDelete, reactionsOf(id)+"/128077", ""
			case "add comment reaction":
				path, body = commentReactionsOf(comment.ID), `{"reaction":"10084-65039"}`
			case "remove comment reaction":
				method, path, body = http.MethodDelete, commentReactionsOf(comment.ID)+"/128077", ""
			}
			route.answerWithin(t, w.contract, method, path, w.ids["bob"], body, 500)
			if len(events) != 1 {
				t.Fatalf("failed publication invoked %d events", len(events))
			}
			if !maps.Equal(before, collaborationRows(t, w, id)) || issueEventCount(t, w) != 0 {
				t.Fatal("publication failure committed business/activity/subscriber/reaction rows or earlier River insert")
			}
		})
	}
}

func TestReactionEventsComposition(t *testing.T) {
	for _, commentTarget := range []bool{false, true} {
		t.Run(fmt.Sprint("comment=", commentTarget), func(t *testing.T) {
			w := newIssueWorld(t)
			id := w.newWorkItem(t)
			comment := w.newComment(t, "alice", id)
			container, path, entity, field, table := id, reactionsOf(id), "issue_reaction", "issue", "issue_reactions"
			if commentTarget {
				container, path, entity, field, table = comment.ID, commentReactionsOf(comment.ID), "comment_reaction", "comment", "comment_reactions"
			}
			beforeIssue, beforeComment := issueRow(t, w, id), commentRow(t, w, comment.ID)
			activities := descriptionRows(t, w, "issue_activities", id)
			at := time.Date(2044, 5, 6, 7, 8, 9, 123456000, time.UTC)
			c := clockAt(at)
			clock := &eventCountingClock{base: c}
			var events []shared.Event
			var failure error
			route := issueEventRouteFor(t, w, clock, &events, &failure, issue.EventTypes(), func(ctx context.Context, e shared.Event) error {
				var reaction string
				var deleted bool
				if err := postgres.DB(ctx, w.pool).QueryRow(ctx, "SELECT reaction,deleted_at IS NOT NULL FROM "+table+" WHERE id=$1", e.Entity.ID).Scan(&reaction, &deleted); err != nil {
					return err
				}
				if reaction != "10084-65039" || deleted != (e.Type == "issue.reaction_removed") {
					t.Error("event preceded reaction write", reaction, deleted)
				}
				return nil
			})
			var got answeredReaction
			decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPost, path, w.ids["bob"], `{"reaction":"10084-65039"}`, 201), &got)
			if got.ID == container || got.ID == w.ids["bob"] || container == w.ids["bob"] {
				t.Fatal("fixture IDs must differ")
			}
			route.answerWithin(t, w.contract, http.MethodPost, path, w.ids["bob"], `{"reaction":"10084-65039"}`, 409)
			route.answerWithin(t, w.contract, http.MethodDelete, path+"/10084-65039", w.ids["alice"], "", 404)
			c.advance(time.Minute)
			route.answerWithin(t, w.contract, http.MethodDelete, path+"/10084-65039", w.ids["bob"], "", 204)
			route.answerWithin(t, w.contract, http.MethodDelete, path+"/10084-65039", w.ids["bob"], "", 404)
			if len(events) != 2 || issueEventCount(t, w) != 2 || clock.calls.Load() != 2 {
				t.Fatalf("reaction events/jobs/clock=%d/%d/%d", len(events), issueEventCount(t, w), clock.calls.Load())
			}
			wantCollaborationEnvelope(t, w, events[0], "issue.reaction_added", entity, got.ID, w.ids["bob"], at, []shared.Change{{Field: "reaction", New: eventValue("10084-65039")}, {Field: field, NewID: &container}})
			wantCollaborationEnvelope(t, w, events[1], "issue.reaction_removed", entity, got.ID, w.ids["bob"], at.Add(time.Minute), []shared.Change{{Field: "reaction", Old: eventValue("10084-65039")}, {Field: field, OldID: &container}})
			if events[0].ID == events[1].ID {
				t.Error("reaction events reused identity")
			}
			if activities != descriptionRows(t, w, "issue_activities", id) {
				t.Error("reaction created activity")
			}
			if commentTarget && (beforeIssue != issueRow(t, w, id) || beforeComment != commentRow(t, w, comment.ID)) {
				t.Error("comment reaction advanced parent audit")
			}
		})
	}
}
