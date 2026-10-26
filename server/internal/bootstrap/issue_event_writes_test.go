package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type eventCountingClock struct {
	base  interface{ Now() time.Time }
	calls atomic.Int32
}

func (c *eventCountingClock) Now() time.Time { c.calls.Add(1); return c.base.Now() }

type issueEventProbeArgs struct {
	Event shared.Event `json:"event"`
}

func (issueEventProbeArgs) Kind() string { return "issue_test.event_probe" }
func eventValue[T any](v T) *T           { return &v }

// Real dispatcher -> transaction bridge -> River insert. The independent pool
// observes no job until the HTTP business write commits.
func issueEventRoute(t *testing.T, w issueWorld, c interface{ Now() time.Time }, received *[]shared.Event, fail *error) moduleRoute {
	t.Helper()
	client, err := jobs.NewInserter(w.pool)
	if err != nil {
		t.Fatal(err)
	}
	inserter := issueInserter{client: client}
	dispatcher, err := newDispatcher(issue.EventTypes(), []shared.EventSubscriber{{Name: "transaction probe", Types: []string{"issue.created", "issue.updated"}, Handle: func(ctx context.Context, e shared.Event) error {
		if _, ok := postgres.Tx(ctx); !ok {
			t.Error("event lost request transaction")
		}
		var html, version string
		if err := postgres.DB(ctx, w.pool).QueryRow(ctx, `SELECT i.description_html,v.description_html FROM issues i JOIN LATERAL (SELECT description_html FROM issue_description_versions WHERE issue_id=i.id AND deleted_at IS NULL ORDER BY last_saved_at DESC,id DESC LIMIT 1) v ON true WHERE i.id=$1`, e.Entity.ID).Scan(&html, &version); err != nil {
			return err
		}
		if html != version {
			t.Errorf("event preceded version write: issue=%q version=%q", html, version)
		}
		*received = append(*received, e)
		var before, after int
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM river_job WHERE kind='issue_test.event_probe'").Scan(&before); err != nil {
			return err
		}
		if err := inserter.InsertTx(ctx, issueEventProbeArgs{Event: e}, &river.InsertOpts{Queue: "issue_events"}); err != nil {
			return err
		}
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM river_job WHERE kind='issue_test.event_probe'").Scan(&after); err != nil {
			return err
		}
		if before != after {
			t.Errorf("uncommitted event job visible outside transaction: %d -> %d", before, after)
		}
		return *fail
	}}})
	if err != nil {
		t.Fatal(err)
	}
	return newIssueRouteWithPublisher(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), c, dispatcher)
}
func issueEventCount(t *testing.T, w issueWorld) int {
	t.Helper()
	var n int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM river_job WHERE kind='issue_test.event_probe'").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
func TestIssueCreatedEventComposition(t *testing.T) {
	w := newIssueWorld(t)
	parent := w.newWorkItem(t)
	var label struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/labels", `{"name":"UI"}`, 201), &label)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE projects SET default_assignee_id=$2 WHERE id=$1", w.web, w.ids["bob"]); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2044, 5, 6, 7, 8, 9, 123456000, time.UTC)
	c := clockAt(at)
	var received []shared.Event
	var failure error
	tracked := &eventCountingClock{base: c}
	route := issueEventRoute(t, w, tracked, &received, &failure)
	path := "/api/v0/projects/" + w.web.String() + "/issues"
	body := fmt.Sprintf(`{"name":"  Complete  ","description_html":"<p>Clean</p><script>bad()</script>","state_id":%q,"priority":"urgent","start_date":"2044-05-01","target_date":"2044-05-08","parent_id":%q,"label_ids":%s}`, w.done, parent, idArray(label.ID, w.bug))
	var got storedIssue
	decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPost, path, w.ids["alice"], body, 201), &got)
	if n := tracked.calls.Load(); n != 1 {
		t.Fatalf("create clock reads=%d, want one", n)
	}
	if len(received) != 1 {
		t.Fatalf("created events=%d, want one", len(received))
	}
	e := received[0]
	if e.ID == (uuid.UUID{}) || e.ID == got.ID || e.Type != "issue.created" || e.Entity != (shared.EntityRef{Kind: "issue", ID: got.ID}) || e.WorkspaceID != w.acme || e.ProjectID != w.web || e.ActorID == nil || *e.ActorID != w.ids["alice"] || !e.OccurredAt.Equal(at) {
		t.Fatalf("event envelope=%+v", e)
	}
	want := []shared.Change{{Field: "name", New: eventValue("Complete")}, {Field: "description", New: eventValue("<p>Clean</p>")}, {Field: "state", New: eventValue("Done"), NewID: &w.done}, {Field: "priority", New: eventValue("urgent")}, {Field: "start_date", New: eventValue("2044-05-01")}, {Field: "target_date", New: eventValue("2044-05-08")}, {Field: "parent", New: eventValue("WEB-1"), NewID: &parent}, {Field: "sort_order", New: eventValue("65535")}, {Field: "sequence_id", New: eventValue("2")}, {Field: "completed_at", New: eventValue("2044-05-06T07:08:09.123456Z")}, {Field: "assignees", New: eventValue("bob"), NewID: eventValue(w.ids["bob"])}}
	labels := []shared.Change{{Field: "labels", New: eventValue("Bug"), NewID: &w.bug}, {Field: "labels", New: eventValue("UI"), NewID: &label.ID}}
	if w.bug.Compare(label.ID) > 0 {
		labels[0], labels[1] = labels[1], labels[0]
	}
	want = append(want, labels...)
	if !reflect.DeepEqual(e.Changes, want) {
		t.Fatalf("created changes=%+v want=%+v", e.Changes, want)
	}
	var persisted uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT (args->'event'->>'ID')::uuid FROM river_job WHERE kind='issue_test.event_probe'`).Scan(&persisted); err != nil || persisted != e.ID {
		t.Fatalf("job event ID=%s err=%v", persisted, err)
	}
	// A subscriber failure rolls back the issue, sequence, all child rows and job.
	sentinel := errors.New("event subscriber rejected")
	failure = sentinel
	var sequenceBefore, sequenceAfter int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT last_issue_sequence FROM projects WHERE id=$1", w.web).Scan(&sequenceBefore); err != nil {
		t.Fatal(err)
	}
	route.answerWithin(t, w.contract, http.MethodPost, path, w.ids["alice"], body, 500)
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT last_issue_sequence FROM projects WHERE id=$1", w.web).Scan(&sequenceAfter); err != nil {
		t.Fatal(err)
	}
	if sequenceBefore != sequenceAfter || len(queryIDs(t, w.pool, "SELECT id FROM issues WHERE project_id=$1", w.web)) != 2 || issueEventCount(t, w) != 1 {
		t.Fatal("publisher failure committed business rows/sequence/job")
	}
	if len(received) != 2 || received[1].ID == e.ID {
		t.Fatal("separate create reused event identity")
	}
	// Version failure must stop before subscriber invocation, not merely undo jobs.
	failure = nil
	received = nil
	if _, err := w.pool.Exec(pgtest.Soon(t), `ALTER TABLE issue_description_versions ADD CONSTRAINT reject_event_description CHECK (description_html NOT LIKE '%Rejected%')`); err != nil {
		t.Fatal(err)
	}
	route.answerWithin(t, w.contract, http.MethodPost, path, w.ids["alice"], `{"name":"Rejected create","description_html":"<p>Rejected</p>"}`, 500)
	if len(received) != 0 || issueEventCount(t, w) != 1 {
		t.Fatal("create published before failed initial version")
	}
}

func TestIssueUpdatedEventComposition(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItemIn(t, w.web, `{"name":"Initial","description_html":"<p>Old</p>"}`)
	at := time.Date(2044, 5, 6, 7, 8, 9, 123456000, time.UTC)
	c := clockAt(at)
	var received []shared.Event
	var failure error
	tracked := &eventCountingClock{base: c}
	route := issueEventRoute(t, w, tracked, &received, &failure)
	path := "/api/v0/issues/" + id.String()
	send := func(body string, want int) {
		t.Helper()
		route.answerWithin(t, w.contract, http.MethodPatch, path, w.ids["alice"], body, want)
	}
	version := savedVersion(t, w, id)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issue_description_versions SET last_saved_at=$2 WHERE id=$1", version.ID, at.Add(-599*time.Second)); err != nil {
		t.Fatal(err)
	}
	send(`{"description_html":"<p>First</p>"}`, 200)
	if n := tracked.calls.Load(); n != 1 {
		t.Fatalf("update clock reads=%d, want one", n)
	}
	if got := savedVersion(t, w, id); got.ID != version.ID {
		t.Fatal("599 seconds did not merge version")
	}
	c.advance(600 * time.Second)
	send(`{"description_html":"<p>Second</p>"}`, 200)
	if got := savedVersion(t, w, id); got.ID != version.ID {
		t.Fatal("600 seconds did not merge version")
	}
	c.advance(600*time.Second + time.Microsecond)
	send(`{"description_html":"<p>Outside</p>"}`, 200)
	if got := savedVersion(t, w, id); got.ID == version.ID {
		t.Fatal("600 seconds plus 1us merged version")
	}
	// Sorting has an event although it has no activity row.
	beforeActivities := descriptionRows(t, w, "issue_activities", id)
	send(`{"sort_order":0}`, 200)
	send(`{"sort_order":1}`, 200)
	send(`{"sort_order":1.0000000000000002}`, 200)
	if beforeActivities != descriptionRows(t, w, "issue_activities", id) {
		t.Fatal("sorting inserted activity")
	}
	send(`{"name":"Initial"}`, 200)
	send(`{"sort_order":1.0000000000000002}`, 200)
	send(`{"description_html":"<p data-id=\"migration\">Outside</p>","skip_activity":true}`, 200)
	if len(received) != 6 || issueEventCount(t, w) != 6 {
		t.Fatalf("real changes/noop/skip events=%d jobs=%d want 6", len(received), issueEventCount(t, w))
	}
	send(`{"description_html":"<p>Third</p>","skip_activity":true}`, 200)
	want := [][]shared.Change{
		{{Field: "description", Old: eventValue("<p>Old</p>"), New: eventValue("<p>First</p>")}},
		{{Field: "description", Old: eventValue("<p>First</p>"), New: eventValue("<p>Second</p>")}},
		{{Field: "description", Old: eventValue("<p>Second</p>"), New: eventValue("<p>Outside</p>")}},
		{{Field: "sort_order", Old: eventValue("65535"), New: eventValue("0")}},
		{{Field: "sort_order", Old: eventValue("0"), New: eventValue("1")}},
		{{Field: "sort_order", Old: eventValue("1"), New: eventValue("1.0000000000000002")}},
		{{Field: "description", Old: eventValue(`<p data-id="migration">Outside</p>`), New: eventValue("<p>Third</p>")}},
	}
	if len(received) != len(want) {
		t.Fatalf("events=%d want %d", len(received), len(want))
	}
	seen := map[uuid.UUID]bool{}
	for n, e := range received {
		expectedAt := at
		if n > 0 {
			expectedAt = at.Add(600 * time.Second)
		}
		if n > 1 {
			expectedAt = at.Add(1200*time.Second + time.Microsecond)
		}
		if e.ID == (uuid.UUID{}) || e.ID == id || seen[e.ID] || e.Type != "issue.updated" || e.Entity != (shared.EntityRef{Kind: "issue", ID: id}) || e.ActorID == nil || *e.ActorID != w.ids["alice"] || e.WorkspaceID != w.acme || e.ProjectID != w.web || !e.OccurredAt.Equal(expectedAt) || !reflect.DeepEqual(e.Changes, want[n]) {
			t.Errorf("event %d=%+v want changes=%+v", n, e, want[n])
		}
		seen[e.ID] = true
	}
	tables := []string{"issues", "issue_assignees", "issue_labels", "issue_subscribers", "issue_mentions", "issue_activities", "issue_description_versions"}
	snapshot := map[string]string{}
	for _, table := range tables {
		snapshot[table] = descriptionRows(t, w, table, id)
	}
	failure = errors.New("subscriber failed")
	send(fmt.Sprintf(`{"name":"Rejected","description_html":"<p>Rejected</p>","assignee_ids":[%q],"label_ids":[%q]}`, w.ids["bob"], w.bug), 500)
	for _, table := range tables {
		if snapshot[table] != descriptionRows(t, w, table, id) {
			t.Error("subscriber failure did not rollback", table)
		}
	}
	if issueEventCount(t, w) != 7 {
		t.Fatal("subscriber failure committed job")
	}
	failure = nil
	received = nil
	if _, err := w.pool.Exec(pgtest.Soon(t), `ALTER TABLE issue_description_versions ADD CONSTRAINT reject_event_description CHECK (description_html NOT LIKE '%Rejected%')`); err != nil {
		t.Fatal(err)
	}
	send(`{"description_html":"<p>Rejected</p>"}`, 500)
	if len(received) != 0 || issueEventCount(t, w) != 7 {
		t.Fatal("update published before failed description version")
	}
}

func TestIssueUpdatedEventHiddenOldParent(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	hidden := w.newWorkItem(t)
	own := w.newWorkItemBy(t, "walt", w.web, `{"name":"Own"}`)
	child := w.newWorkItemBy(t, "walt", w.web, `{"name":"Child"}`)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+child.String(), fmt.Sprintf(`{"parent_id":%q}`, hidden), 200)
	at := time.Date(2044, 5, 6, 7, 8, 9, 0, time.UTC)
	var received []shared.Event
	var failure error
	route := issueEventRoute(t, w, clockAt(at), &received, &failure)
	route.answerWithin(t, w.contract, http.MethodPatch, "/api/v0/issues/"+child.String(), w.ids["walt"], fmt.Sprintf(`{"parent_id":%q}`, own), 200)
	want := []shared.Change{{Field: "parent", Old: eventValue("WEB-1"), New: eventValue("WEB-2"), OldID: &hidden, NewID: &own}}
	if len(received) != 1 || !reflect.DeepEqual(received[0].Changes, want) {
		t.Fatalf("hidden real old parent must survive response filtering: %+v", received)
	}
}
