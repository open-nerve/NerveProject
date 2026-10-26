package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type answeredLink struct {
	ID          uuid.UUID           `json:"id"`
	IssueID     uuid.UUID           `json:"issue_id"`
	URL         string              `json:"url"`
	Title       *string             `json:"title"`
	Metadata    domain.LinkMetadata `json:"metadata"`
	CreatedByID *uuid.UUID          `json:"created_by_id"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

func linksOf(id uuid.UUID) string { return "/api/v0/issues/" + id.String() + "/links" }
func linkBody(url string, title *string) string {
	b, _ := json.Marshal(struct {
		URL   string  `json:"url"`
		Title *string `json:"title"`
	}{url, title})
	return string(b)
}
func createLink(t *testing.T, w issueWorld, who string, id uuid.UUID, url string) answeredLink {
	t.Helper()
	var got answeredLink
	decodeAnswer(t, w.send(t, who, http.MethodPost, linksOf(id), linkBody(url, nil), 201), &got)
	return got
}

// High-entropy Unicode data exceeds PostgreSQL's inline index tuple limit
// while remaining exactly 2048 characters; no network destination is queried.
func longAPILinkURL() string {
	prefix := []rune("http://169.254.169.254/")
	out := append([]rune{}, prefix...)
	for n := 0; len(out) < 2048; n++ {
		h := sha256.Sum256([]byte(fmt.Sprintf("P7 API original %d", n)))
		for i := 0; i < 32 && len(out) < 2048; i += 2 {
			out = append(out, rune(0x4e00+(uint16(h[i])<<8|uint16(h[i+1]))%20000))
		}
	}
	return string(out)
}
func TestCreateIssueLinkAPI(t *testing.T) {
	w := newIssueWorld(t)
	id, other := w.newWorkItem(t), w.newWorkItem(t)
	long := longAPILinkURL()
	got := createLink(t, w, "bob", id, long)
	if got.ID == (uuid.UUID{}) || got.IssueID != id || got.URL != long || got.Title != nil || got.CreatedByID == nil || *got.CreatedByID != w.ids["bob"] || got.Metadata == nil || len(got.Metadata) != 0 || !got.CreatedAt.Equal(got.UpdatedAt) {
		t.Fatalf("creation=%+v", got)
	}
	var stored string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT url FROM issue_links WHERE id=$1", got.ID).Scan(&stored); err != nil || stored != long {
		t.Fatal("stored original differs", err)
	}
	w.send(t, "bob", http.MethodPost, linksOf(id), linkBody(long, nil), 409)
	createLink(t, w, "bob", other, long)
	before := descriptionRows(t, w, "issue_links", id)
	audit := descriptionAudit(t, w, id)
	answer := w.send(t, "bob", http.MethodPost, linksOf(id), linkBody(long+"界", nil), 422)
	if reason := refusalOf(t, []byte(answer), false); reason != "validation_failed url not_allowed" {
		t.Fatal(reason)
	}
	if before != descriptionRows(t, w, "issue_links", id) || audit != descriptionAudit(t, w, id) {
		t.Fatal("invalid URL wrote rows")
	}
	// Duplicate lookup must use the same normalized string as the insert.
	normalized := createLink(t, w, "bob", id, "  169.254.169.254/normalize  ")
	if normalized.URL != "http://169.254.169.254/normalize" {
		t.Fatal(normalized.URL)
	}
	w.send(t, "bob", http.MethodPost, linksOf(id), `{"url":"  169.254.169.254/normalize "}`, 409)
	title := "  User title  "
	var titled answeredLink
	decodeAnswer(t, w.send(t, "bob", http.MethodPost, linksOf(id), linkBody("http://169.254.169.254/title", &title), 201), &titled)
	if titled.Title == nil || *titled.Title != title {
		t.Fatal(titled)
	}
}
func TestListIssueLinksAPI(t *testing.T) {
	w := newIssueWorld(t)
	id, other := w.newWorkItem(t), w.newWorkItem(t)
	a := createLink(t, w, "bob", id, "http://169.254.169.254/a")
	b := createLink(t, w, "bob", id, "http://169.254.169.254/b")
	c := createLink(t, w, "bob", id, "http://169.254.169.254/c")
	// The smaller id is newer; tied older ids must still descend.
	at := time.Date(2045, 1, 2, 3, 4, 5, 0, time.UTC)
	for _, l := range []answeredLink{a, b, c} {
		when := at
		if l.ID == a.ID {
			when = at.Add(time.Minute)
		}
		if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issue_links SET created_at=$2,created_by_id=NULL WHERE id=$1", l.ID, when); err != nil {
			t.Fatal(err)
		}
	}
	got := listIn[answeredLink](t, w.send(t, "bob", http.MethodGet, linksOf(id), "", 200))
	ids := []uuid.UUID{}
	for _, l := range got {
		ids = append(ids, l.ID)
		if l.Title != nil || l.CreatedByID != nil || l.Metadata == nil {
			t.Fatal("null shape", l)
		}
	}
	if !slices.Equal(ids, []uuid.UUID{a.ID, c.ID, b.ID}) {
		t.Fatal("descending order", ids)
	}
	if answer := w.send(t, "alice", http.MethodGet, linksOf(other), "", 200); answer != "{\"data\":[]}\n" {
		t.Fatal(answer)
	}
}
func TestListIssueLinksGuestVisibility(t *testing.T) {
	w := newIssueWorld(t)
	other := w.newWorkItem(t)
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	own := w.newWorkItemBy(t, "walt", w.web, `{"name":"Guest own"}`)
	createLink(t, w, "bob", own, "http://169.254.169.254/own")
	createLink(t, w, "bob", other, "http://169.254.169.254/other")
	membership := queryIDs(t, w.pool, "SELECT id FROM project_members WHERE project_id=$1 AND member_id=$2", w.web, w.ids["walt"])[0]
	w.send(t, "alice", http.MethodPatch, "/api/v0/project-members/"+membership.String(), `{"role":5}`, 200)
	w.send(t, "walt", http.MethodGet, linksOf(own), "", 200)
	hidden := w.send(t, "walt", http.MethodGet, linksOf(other), "", 404)
	absent := w.send(t, "walt", http.MethodGet, linksOf(uuid.NewV7()), "", 404)
	if hidden != absent {
		t.Fatal("hidden differs from missing")
	}
	w.send(t, "walt", http.MethodPost, linksOf(own), `{"url":"http://169.254.169.254/no"}`, 403)
	w.send(t, "walt", http.MethodPost, linksOf(other), `{"url":"http://169.254.169.254/no"}`, 404)
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"guest_view_all_features":true}`, 200)
	rows := listIn[answeredLink](t, w.send(t, "walt", http.MethodGet, linksOf(other), "", 200))
	if len(rows) != 1 || rows[0].IssueID != other {
		t.Fatal(rows)
	}
	w.send(t, "walt", http.MethodPost, linksOf(other), `{"url":"http://169.254.169.254/no"}`, 403)
}
func TestCreateIssueLinkEventComposition(t *testing.T) {
	for n, title := range []*string{nil, eventValue("  Named  ")} {
		t.Run([]string{"null title", "title value"}[n], func(t *testing.T) {
			w := newIssueWorld(t)
			id := w.newWorkItem(t)
			at := time.Date(2044, 5, 6, 7, 8, 9, 123456000, time.UTC)
			clock := &eventCountingClock{base: clockAt(at)}
			var events []shared.Event
			var failure error
			route := issueEventRouteFor(t, w, clock, &events, &failure, issue.EventTypes(), func(ctx context.Context, e shared.Event) error {
				var ok bool
				err := postgres.DB(ctx, w.pool).QueryRow(ctx, `SELECT l.created_at=$2 AND l.updated_at=$2 AND l.created_by_id=$3 AND l.updated_by_id=$3 AND i.updated_at=$2 AND i.updated_by_id=$3 AND a.created_at=$2 AND a.updated_at=$2 AND a.actor_id=$3 AND a.created_by_id=$3 AND a.updated_by_id=$3 FROM issue_links l JOIN issues i ON i.id=l.issue_id JOIN issue_activities a ON a.issue_id=i.id AND a.field='link' WHERE l.id=$1`, e.Entity.ID, at, w.ids["bob"]).Scan(&ok)
				if err == nil && !ok {
					t.Error("link/issue/activity did not share one stamp")
				}
				return err
			})
			var got answeredLink
			decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPost, linksOf(id), w.ids["bob"], linkBody("http://169.254.169.254/event", title), 201), &got)
			route.answerWithin(t, w.contract, http.MethodPost, linksOf(id), w.ids["bob"], linkBody("  169.254.169.254/event ", title), 409)
			if len(events) != 1 || issueEventCount(t, w) != 1 || clock.calls.Load() != 1 {
				t.Fatalf("events/jobs/clock=%d/%d/%d", len(events), issueEventCount(t, w), clock.calls.Load())
			}
			wantCollaborationEnvelope(t, w, events[0], "issue.link_created", "issue_link", got.ID, w.ids["bob"], at, []shared.Change{{Field: "url", New: eventValue(got.URL)}, {Field: "title", New: title}})
			var field, verb, comment string
			var old, new *string
			var oldID, newID *uuid.UUID
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT field,verb,old_value,new_value,old_identifier,new_identifier,comment FROM issue_activities WHERE issue_id=$1 AND field='link'", id).Scan(&field, &verb, &old, &new, &oldID, &newID, &comment); err != nil {
				t.Fatal(err)
			}
			if field != "link" || verb != "created" || old != nil || new == nil || *new != got.URL || oldID != nil || newID == nil || *newID != got.ID || comment != "created a link" {
				t.Fatalf("activity slots=%s %s %v %v %v %v %s", field, verb, old, new, oldID, newID, comment)
			}
			if !got.CreatedAt.Equal(at) || !got.UpdatedAt.Equal(at) || !reflect.DeepEqual(got.Title, title) {
				t.Fatal(got)
			}
		})
	}
}

func TestCreateIssueLinkPublicationRollback(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItem(t)
	beforeIssue := issueRow(t, w, id)
	beforeActivity := descriptionRows(t, w, "issue_activities", id)
	var events []shared.Event
	failure := fmt.Errorf("subscriber rejected link creation")
	route := issueEventRouteFor(t, w, clockAt(time.Date(2044, 5, 6, 7, 8, 9, 0, time.UTC)), &events, &failure, issue.EventTypes(), nil)
	route.answerWithin(t, w.contract, http.MethodPost, linksOf(id), w.ids["bob"], `{"url":"http://169.254.169.254/rollback"}`, 500)
	if len(events) != 1 || issueEventCount(t, w) != 0 || beforeIssue != issueRow(t, w, id) || beforeActivity != descriptionRows(t, w, "issue_activities", id) {
		t.Fatal("publication failure committed issue/activity/job rows")
	}
	if rows := queryIDs(t, w.pool, "SELECT id FROM issue_links WHERE issue_id=$1", id); len(rows) != 0 {
		t.Fatal("publication failure committed link", rows)
	}
}
