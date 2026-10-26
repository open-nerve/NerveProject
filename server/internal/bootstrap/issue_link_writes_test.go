package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func linkPath(id uuid.UUID) string { return "/api/v0/issue-links/" + id.String() }

func TestUpdateIssueLinkAPI(t *testing.T) {
	w := newIssueWorld(t)
	id, other := w.newWorkItem(t), w.newWorkItem(t)
	a := createLink(t, w, "bob", id, "http://169.254.169.254/a")
	b := createLink(t, w, "bob", id, "http://169.254.169.254/b")
	if _, err := w.pool.Exec(pgtest.Soon(t), `UPDATE issue_links SET metadata='{"title":"fresh preview"}' WHERE id IN ($1,$2)`, a.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	// A different member may edit the author's link; omitted/null/empty titles differ.
	for _, tc := range []struct {
		body  string
		title *string
	}{{`{"title":"  Mine  "}`, eventValue("  Mine  ")}, {`{"url":"  169.254.169.254/a "}`, eventValue("  Mine  ")}, {`{"title":null}`, nil}, {`{"title":""}`, eventValue("")}} {
		var got answeredLink
		decodeAnswer(t, w.send(t, "alice", http.MethodPatch, linkPath(a.ID), tc.body, 200), &got)
		if !reflect.DeepEqual(got.Title, tc.title) || !reflect.DeepEqual(got.Metadata, domain.LinkMetadata{"title": eventValue("fresh preview")}) {
			t.Fatalf("title/metadata=%+v", got)
		}
	}
	var got answeredLink
	long := longAPILinkURL()
	decodeAnswer(t, w.send(t, "alice", http.MethodPatch, linkPath(a.ID), linkBody(long, eventValue("changed")), 200), &got)
	var stored string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT url FROM issue_links WHERE id=$1", a.ID).Scan(&stored); err != nil || stored != long || got.URL != long || len(got.Metadata) != 0 {
		t.Fatal("update lost original or retained metadata", err, got)
	}
	before, audit := descriptionRows(t, w, "issue_links", id), descriptionAudit(t, w, id)
	activities := descriptionRows(t, w, "issue_activities", id)
	if activities == "[]" {
		t.Fatal("rollback baseline must contain existing activities")
	}
	var rejected []shared.Event
	var failure error
	route := issueEventRouteFor(t, w, clockAt(time.Date(2044, 5, 6, 7, 8, 9, 0, time.UTC)), &rejected, &failure, issue.EventTypes(), nil)
	for _, tc := range []struct {
		link   uuid.UUID
		body   string
		status int
	}{{a.ID, linkBody(long+"界", nil), 422}, {b.ID, linkBody(long, nil), 409}} {
		route.answerWithin(t, w.contract, http.MethodPatch, linkPath(tc.link), w.ids["alice"], tc.body, tc.status)
		if before != descriptionRows(t, w, "issue_links", id) || audit != descriptionAudit(t, w, id) {
			t.Fatal("refusal changed metadata/audit")
		}
		if activities != descriptionRows(t, w, "issue_activities", id) {
			t.Fatal("refusal changed activity snapshot")
		}
	}
	if len(rejected) != 0 || issueEventCount(t, w) != 0 {
		t.Fatal("invalid/duplicate update published an event")
	}
	c := createLink(t, w, "bob", other, "http://169.254.169.254/c")
	w.send(t, "alice", http.MethodPatch, linkPath(c.ID), linkBody(long, nil), 200)
	w.send(t, "alice", http.MethodDelete, linkPath(a.ID), "", 204)
	w.send(t, "alice", http.MethodPatch, linkPath(b.ID), linkBody(long, nil), 200)
	w.send(t, "alice", http.MethodDelete, linkPath(b.ID), "", 204)
	createLink(t, w, "bob", id, long)
}

func TestUpdateIssueLinkEventsAndDeleteIssueLink(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItem(t)
	l := createLink(t, w, "bob", id, "http://169.254.169.254/before")
	at := time.Date(2044, 5, 6, 7, 8, 9, 123456000, time.UTC)
	clock := &eventCountingClock{base: clockAt(at)}
	var events []shared.Event
	var failure error
	route := issueEventRouteFor(t, w, clock, &events, &failure, issue.EventTypes(), func(ctx context.Context, e shared.Event) error {
		var ok bool
		err := postgres.DB(ctx, w.pool).QueryRow(ctx, `SELECT l.updated_at=$2 AND l.updated_by_id=$3 AND i.updated_at=$2 AND i.updated_by_id=$3 FROM issue_links l JOIN issues i ON i.id=l.issue_id WHERE l.id=$1`, e.Entity.ID, at, w.ids["alice"]).Scan(&ok)
		if err == nil && !ok {
			t.Error("link and issue do not share stamp")
		}
		return err
	})
	send := func(method, body string, want int) string {
		return route.answerWithin(t, w.contract, method, linkPath(l.ID), w.ids["alice"], body, want)
	}
	activity := descriptionRows(t, w, "issue_activities", id)
	send(http.MethodPatch, `{"title":"Named"}`, 200)
	if activity != descriptionRows(t, w, "issue_activities", id) {
		t.Fatal("title-only inserted URL activity")
	}
	send(http.MethodPatch, linkBody("http://169.254.169.254/after", nil), 200)
	before, audit := descriptionRows(t, w, "issue_links", id), descriptionAudit(t, w, id)
	send(http.MethodPatch, `{}`, 200)
	send(http.MethodPatch, `{"url":" 169.254.169.254/after ","title":null}`, 200)
	if before != descriptionRows(t, w, "issue_links", id) || audit != descriptionAudit(t, w, id) || clock.calls.Load() != 2 || len(events) != 2 || issueEventCount(t, w) != 2 {
		t.Fatal("noop changed rows/events/clock")
	}
	send(http.MethodDelete, "", 204)
	if len(events) != 3 || clock.calls.Load() != 3 || issueEventCount(t, w) != 3 {
		t.Fatal("write event/clock count")
	}
	want := [][]shared.Change{
		{{Field: "title", New: eventValue("Named")}},
		{{Field: "url", Old: eventValue(l.URL), New: eventValue("http://169.254.169.254/after")}, {Field: "title", Old: eventValue("Named")}},
		{{Field: "url", Old: eventValue("http://169.254.169.254/after")}, {Field: "title"}},
	}
	for i, e := range events {
		kind := "issue.link_updated"
		if i == 2 {
			kind = "issue.link_deleted"
		}
		wantCollaborationEnvelope(t, w, e, kind, "issue_link", l.ID, w.ids["alice"], at, want[i])
	}
	for _, verb := range []string{"updated", "deleted"} {
		var old, new *string
		var oldID, newID *uuid.UUID
		var comment string
		var stamped bool
		if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT old_value,new_value,old_identifier,new_identifier,comment,created_at=$3 AND updated_at=$3 AND actor_id=$4 AND created_by_id=$4 AND updated_by_id=$4 FROM issue_activities WHERE issue_id=$1 AND field='link' AND verb=$2`, id, verb, at, w.ids["alice"]).Scan(&old, &new, &oldID, &newID, &comment, &stamped); err != nil {
			t.Fatal(err)
		}
		if !stamped || old == nil || new == nil {
			t.Fatal("activity stamps or nullable slots", verb)
		}
		if verb == "updated" {
			if *old != l.URL || *new != "http://169.254.169.254/after" || oldID == nil || newID == nil || *oldID != l.ID || *newID != l.ID || comment != "updated a link" {
				t.Fatal("updated activity slots")
			}
		} else if *old != "http://169.254.169.254/after" || *new != "" || oldID != nil || newID != nil || comment != "deleted the link" {
			t.Fatal("delete activity must use empty string and nil identifiers")
		}
	}
	var deleted bool
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT deleted_at=$2 AND updated_at=$2 AND updated_by_id=$3 AND created_by_id=$4 FROM issue_links WHERE id=$1`, l.ID, at, w.ids["alice"], w.ids["bob"]).Scan(&deleted); err != nil || !deleted {
		t.Fatal("delete audit", err)
	}
	send(http.MethodDelete, "", 404)
}

func TestUpdateIssueLinkPublicationRollback(t *testing.T) {
	for _, method := range []string{http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			w := newIssueWorld(t)
			id := w.newWorkItem(t)
			l := createLink(t, w, "bob", id, "http://169.254.169.254/rollback")
			if _, err := w.pool.Exec(pgtest.Soon(t), `UPDATE issue_links SET metadata='{"title":"keep"}' WHERE id=$1`, l.ID); err != nil {
				t.Fatal(err)
			}
			before, audit := descriptionRows(t, w, "issue_links", id), descriptionAudit(t, w, id)
			activities := descriptionRows(t, w, "issue_activities", id)
			if activities == "[]" {
				t.Fatal("rollback baseline must contain existing activities")
			}
			var events []shared.Event
			failure := fmt.Errorf("subscriber rejected link write")
			route := issueEventRouteFor(t, w, clockAt(time.Date(2044, 5, 6, 7, 8, 9, 0, time.UTC)), &events, &failure, issue.EventTypes(), nil)
			body := ""
			if method == http.MethodPatch {
				body = linkBody("http://169.254.169.254/rejected", nil)
			}
			route.answerWithin(t, w.contract, method, linkPath(l.ID), w.ids["alice"], body, 500)
			if before != descriptionRows(t, w, "issue_links", id) || audit != descriptionAudit(t, w, id) || issueEventCount(t, w) != 0 || len(events) != 1 {
				t.Fatal("failed publication committed link/audit/job")
			}
			if activities != descriptionRows(t, w, "issue_activities", id) {
				t.Fatal("failed publication changed activity snapshot")
			}
		})
	}
}
