package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification"
	notificationriver "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fanoutRun is a notification.fanout job as River holds it: its state, its attempt and its errors.
type fanoutRun struct {
	State           string
	Attempt, Errors int
}

// fanoutRunsOf reads, in order, the notification.fanout jobs of the events of type on the entity.
func fanoutRunsOf(t *testing.T, pool *pgxpool.Pool, eventType string, entity uuid.UUID) []fanoutRun {
	t.Helper()
	rows, err := pool.Query(pgtest.Soon(t), `SELECT state::text, attempt, coalesce(cardinality(errors), 0) FROM river_job
		WHERE kind = $1 AND args->>'type' = $2 AND args->>'entity_id' = $3 ORDER BY id`, notificationriver.FanoutKind, eventType, entity.String())
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var runs []fanoutRun
	for rows.Next() {
		var r fanoutRun
		if err := rows.Scan(&r.State, &r.Attempt, &r.Errors); err != nil {
			t.Fatal(err)
		}
		runs = append(runs, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return runs
}

// notified is a notification as the end-to-end test reads it: its receiver, sender, position and field.
type notified struct {
	Receiver uuid.UUID
	Sender   string
	Seq      int
	Field    string
}

// notifiedOf reads the work item's undeleted notifications, by position.
func notifiedOf(t *testing.T, pool *pgxpool.Pool, item uuid.UUID) []notified {
	t.Helper()
	rows, err := pool.Query(pgtest.Soon(t), `SELECT receiver_id, sender, change_seq, data->'change'->>'field' FROM notifications
		WHERE issue_id = $1 AND deleted_at IS NULL ORDER BY created_at, change_seq`, item)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var got []notified
	for rows.Next() {
		var n notified
		if err := rows.Scan(&n.Receiver, &n.Sender, &n.Seq, &n.Field); err != nil {
			t.Fatal(err)
		}
		got = append(got, n)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return got
}

// The app sends a write's notifications through its fan-out job (M7 P1b spec 2.9): alice creates a work item assigned
// to bob through the API, and its job completes in its first attempt, having sent bob its creation and his
// assignment. A write that notifies no one but its actor of anything enqueues no job: alice's change of the
// description alone, and her description mentioning herself alone; her change of its name enqueues one, which sends
// bob, its assignee, the change.
func TestTheAppSendsAWritesNotificationsThroughItsFanoutJob(t *testing.T) {
	w := newIssueWorld(t)
	alice, bob := w.ids["alice"], w.ids["bob"]
	item := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Fix","assignee_ids":[%q]}`, bob))
	waitForFanouts(t, w.pool)
	if runs := fanoutRunsOf(t, w.pool, "issue.created", item); !reflect.DeepEqual(runs, []fanoutRun{{"completed", 1, 0}}) {
		t.Fatalf("the creation's fan-outs %+v; want one completed in its first attempt", runs)
	}
	got := notifiedOf(t, w.pool, item)
	if len(got) != 2 || got[0] != (notified{bob, "assigned", -1, "issue"}) || got[1].Receiver != bob || got[1].Sender != "assigned" ||
		got[1].Seq < 0 || got[1].Field != "assignees" {
		t.Fatalf("the creation's notifications %+v; want bob's of the creation and of his assignment", got)
	}
	for _, write := range []struct{ name, body string }{
		{"the description alone", `{"description_html":"<p>Steps to reproduce</p>"}`},
		{"alice mentioning herself alone", fmt.Sprintf(`{"description_html":%q}`, mentionHTML(alice))},
	} {
		w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+item.String(), write.body, http.StatusOK)
		if runs := fanoutRunsOf(t, w.pool, "issue.updated", item); len(runs) != 0 {
			t.Fatalf("%s enqueued fan-outs %+v; want none", write.name, runs)
		}
	}
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+item.String(), `{"name":"Fix it now"}`, http.StatusOK)
	if runs := fanoutRunsOf(t, w.pool, "issue.updated", item); len(runs) != 1 {
		t.Fatalf("the name's change enqueued fan-outs %+v; want one", runs)
	}
	waitForFanouts(t, w.pool)
	if got := notifiedOf(t, w.pool, item); len(got) != 3 || got[2] != (notified{bob, "assigned", 0, "name"}) {
		t.Fatalf("the notifications %+v; want bob's of the name's change last", got)
	}
}

// The fan-out's job keeps each event a write publishes as it is but its HTML (M7 P1b spec 2.9, R4): alice's creation
// of a work item with a description mentioning walt and assigned to bob and walt, and bob's comment mentioning alice.
// Each job carries its event's id, type, moment, workspace, project, actor and entity, and every change at its
// position, so the two assignees' as the creation numbers them; the values of the description and of the comment's
// HTML are left out and river_job keeps no HTML; on the default queue with five attempts.
func TestTheFanoutJobKeepsTheEventsPositionsWithoutItsHTML(t *testing.T) {
	w := newIssueWorld(t)
	alice, bob, walt := w.ids["alice"], w.ids["bob"], w.ids["walt"]
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	client, err := jobs.NewInserter(w.pool)
	if err != nil {
		t.Fatal(err)
	}
	var events []shared.Event
	capture := shared.EventSubscriber{Name: "capture", Types: []string{"issue.created", "issue.comment_created"},
		Handle: func(_ context.Context, e shared.Event) error {
			events = append(events, e)
			return nil
		}}
	dispatcher, err := newDispatcher(eventTypes(), append([]shared.EventSubscriber{capture},
		notification.Subscribers(notification.Provide(w.pool), jobInserter{client: client})...))
	if err != nil {
		t.Fatal(err)
	}
	route := newIssueRouteWithPublisher(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), clock.System{}, dispatcher)
	var item struct {
		ID uuid.UUID `json:"id"`
	}
	body := fmt.Sprintf(`{"name":"Fix","assignee_ids":[%q,%q],"description_html":%q}`, bob, walt, mentionHTML(walt))
	decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", alice, body, 201), &item)
	route.answerWithin(t, w.contract, http.MethodPost, commentsOf(item.ID), bob, fmt.Sprintf(`{"comment_html":%q}`, mentionHTML(alice)), 201)
	if len(events) != 2 {
		t.Fatalf("events %+v; want the creation and the comment", events)
	}
	for _, e := range events {
		html := map[string]bool{}
		for _, c := range e.Changes {
			if (c.Field == "description" || c.Field == "comment_html") && c.New != nil && strings.Contains(*c.New, "<mention-component") {
				html[c.Field] = true
			}
		}
		if !html["description"] && !html["comment_html"] {
			t.Fatalf("%s's changes %+v; want its HTML among them", e.Type, e.Changes)
		}
		var raw, queue string
		var attempts int
		if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT args::text, queue, max_attempts FROM river_job WHERE kind = $1 AND args->>'event_id' = $2`,
			notificationriver.FanoutKind, e.ID.String()).Scan(&raw, &queue, &attempts); err != nil {
			t.Fatalf("%s's one fan-out: %v", e.Type, err)
		}
		if strings.Contains(raw, "<") || queue != "default" || attempts != 5 {
			t.Fatalf("%s's fan-out %s on %s with %d attempts; want no HTML, on default with 5", e.Type, raw, queue, attempts)
		}
		var args notificationriver.FanoutArgs
		var shape struct {
			Changes []map[string]any `json:"changes"`
		}
		if err := json.Unmarshal([]byte(raw), &args); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(raw), &shape); err != nil {
			t.Fatal(err)
		}
		if args.EventID != e.ID || args.Type != e.Type || !args.OccurredAt.Equal(e.OccurredAt) || args.WorkspaceID != e.WorkspaceID ||
			args.ProjectID != e.ProjectID || !reflect.DeepEqual(args.ActorID, e.ActorID) || args.EntityKind != e.Entity.Kind || args.EntityID != e.Entity.ID ||
			len(args.Changes) != len(e.Changes) {
			t.Fatalf("%s's fan-out %s; want the event %+v", e.Type, raw, e)
		}
		for n, c := range e.Changes {
			want := notificationriver.FanoutChange{Field: c.Field, Old: c.Old, New: c.New, OldID: c.OldID, NewID: c.NewID}
			if html[c.Field] {
				want.Old, want.New = nil, nil
				if !reflect.DeepEqual(shape.Changes[n], map[string]any{"field": c.Field}) {
					t.Errorf("%s's change %d kept %v; want its field alone", e.Type, n, shape.Changes[n])
				}
			}
			if !reflect.DeepEqual(args.Changes[n], want) {
				t.Errorf("%s's change %d = %+v; want %+v", e.Type, n, args.Changes[n], want)
			}
		}
		if e.Type != "issue.created" {
			continue
		}
		for _, assignee := range []uuid.UUID{bob, walt} {
			at := slices.IndexFunc(args.Changes, func(c notificationriver.FanoutChange) bool {
				return c.Field == "assignees" && c.NewID != nil && *c.NewID == assignee
			})
			if want := positionOf(t, e, "assignees", assignee); at != int(want) {
				t.Errorf("the fan-out has the assignee %s at %d; want %d, the creation's", assignee, at, want)
			}
		}
	}
}
