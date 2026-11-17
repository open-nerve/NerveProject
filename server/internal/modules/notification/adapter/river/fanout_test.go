package riveradapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func text(s string) *string       { return &s }
func ref(id uuid.UUID) *uuid.UUID { return &id }

// fanoutEvent is a comment's creation by an actor as the subscriber hands it over, its HTML blanked: a value "" apart
// from none, ids, and the work item last.
func fanoutEvent() shared.Event {
	actor, item := uuid.NewV7(), uuid.NewV7()
	return shared.Event{ID: uuid.NewV7(), Type: "issue.comment_created", OccurredAt: time.Date(2026, 11, 17, 9, 30, 0, 123456789, time.UTC),
		WorkspaceID: uuid.NewV7(), ProjectID: uuid.NewV7(), ActorID: &actor, Entity: shared.EntityRef{Kind: "issue_comment", ID: uuid.NewV7()},
		Changes: []shared.Change{{Field: "comment_html"}, {Field: "mention", NewID: ref(uuid.NewV7())}, {Field: "reply", Old: text(""), New: text("x")},
			{Field: "issue", NewID: &item}}}
}

// insertManyCalls is an Inserter that answers InsertManyTx with itself; a
// fan-out never inserts a job alone.
type insertManyCalls func(context.Context, []river.InsertManyParams) error

func (f insertManyCalls) InsertTx(context.Context, river.JobArgs, *river.InsertOpts) error {
	return errors.New("a fan-out inserted a job alone")
}

func (f insertManyCalls) InsertManyTx(ctx context.Context, params []river.InsertManyParams) error {
	return f(ctx, params)
}

// fanoutPayload is the JSON fanoutEvent's e is carried as: its id, type, moment, workspace, project, actor, entity and
// every change in order, a nil value or id left out, so "" stays apart from none.
func fanoutPayload(e shared.Event) map[string]any {
	var actor any
	if e.ActorID != nil {
		actor = e.ActorID.String()
	}
	return map[string]any{"event_id": e.ID.String(), "type": e.Type, "occurred_at": "2026-11-17T09:30:00.123456789Z",
		"workspace_id": e.WorkspaceID.String(), "project_id": e.ProjectID.String(), "actor_id": actor,
		"entity_kind": "issue_comment", "entity_id": e.Entity.ID.String(), "changes": []any{
			map[string]any{"field": "comment_html"},
			map[string]any{"field": "mention", "new_id": e.Changes[1].NewID.String()},
			map[string]any{"field": "reply", "old": "", "new": "x"},
			map[string]any{"field": "issue", "new_id": e.Changes[3].NewID.String()}}}
}

// The events are enqueued at once, with one InsertManyTx in the caller's context (its transaction, M7 P1b spec R12),
// a job each, in their order, each carrying its event as the subscriber hands it over, on the default queue with five
// attempts, as each was alone before; nerve's own event, without an actor, too. None are enqueued with no insert,
// and the insert's failure is the enqueueing's.
func TestFanoutsEnqueueTheEventsAtOnce(t *testing.T) {
	ctx := t.Context()
	nerve := fanoutEvent()
	nerve.ActorID = nil
	events := []shared.Event{fanoutEvent(), nerve, fanoutEvent()}
	failure := errors.New("insert failed")
	calls := 0
	q := NewFanouts(insertManyCalls(func(got context.Context, params []river.InsertManyParams) error {
		calls++
		if got != ctx || len(params) != len(events) {
			t.Fatalf("context replaced or %d jobs for %d events", len(params), len(events))
		}
		for n, p := range params {
			if p.Args.Kind() != "notification.fanout" {
				t.Fatalf("job %d kind %s", n, p.Args.Kind())
			}
			if p.InsertOpts == nil || !reflect.DeepEqual(*p.InsertOpts, river.InsertOpts{Queue: "default", MaxAttempts: 5}) {
				t.Fatalf("job %d opts=%+v", n, p.InsertOpts)
			}
			raw, err := json.Marshal(p.Args)
			if err != nil {
				t.Fatal(err)
			}
			var payload map[string]any
			if err := json.Unmarshal(raw, &payload); err != nil {
				t.Fatal(err)
			}
			if want := fanoutPayload(events[n]); !reflect.DeepEqual(payload, want) {
				t.Fatalf("job %d payload=%v\nwant     %v", n, payload, want)
			}
		}
		return failure
	}))
	if err := q.Enqueue(ctx, events); !errors.Is(err, failure) || calls != 1 {
		t.Fatalf("Enqueue=%v calls=%d; want the inserter's failure once", err, calls)
	}
	for _, none := range [][]shared.Event{nil, {}} {
		if err := q.Enqueue(ctx, none); err != nil || calls != 1 {
			t.Fatalf("Enqueue of none = %v after %d calls; want no insert", err, calls)
		}
	}
}

type fanoutCall func(context.Context, shared.Event) error

func (f fanoutCall) Execute(ctx context.Context, e shared.Event) error { return f(ctx, e) }

// The worker rebuilds the event the job carries, as River hands back its arguments, and runs the fan-out in the job's
// context: nerve's own event, without an actor, too. A failure retries the job and its last failed attempt is logged.
// The job's timeout is thirty seconds.
func TestFanoutWorkerRunsTheRebuiltEvent(t *testing.T) {
	failure := errors.New("lock timeout")
	nerve := fanoutEvent()
	nerve.ActorID = nil
	for _, tt := range []struct {
		name    string
		event   shared.Event
		err     error
		attempt int
	}{
		{"sent", fanoutEvent(), nil, 5},
		{"nerve's", nerve, nil, 1},
		{"retry", fanoutEvent(), failure, 4},
		{"exhausted", fanoutEvent(), failure, 5},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var raw []byte
			if err := NewFanouts(insertManyCalls(func(_ context.Context, params []river.InsertManyParams) (err error) {
				raw, err = json.Marshal(params[0].Args)
				return err
			})).Enqueue(t.Context(), []shared.Event{tt.event}); err != nil {
				t.Fatal(err)
			}
			var args FanoutArgs
			if err := json.Unmarshal(raw, &args); err != nil {
				t.Fatal(err)
			}
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, nil))
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			calls := 0
			worker := NewFanoutWorker(fanoutCall(func(got context.Context, e shared.Event) error {
				calls++
				if got != ctx {
					t.Error("context replaced")
				}
				if !reflect.DeepEqual(e, tt.event) {
					t.Errorf("rebuilt %+v\nwant    %+v", e, tt.event)
				}
				return tt.err
			}), logger)
			job := &river.Job[FanoutArgs]{JobRow: &rivertype.JobRow{ID: 73, Attempt: tt.attempt, MaxAttempts: 5}, Args: args}
			if worker.Timeout(job) != 30*time.Second {
				t.Fatal("timeout")
			}
			if err := worker.Work(ctx, job); err != tt.err || calls != 1 {
				t.Fatalf("Work = %v after %d calls; want %v after one", err, calls, tt.err)
			}
			if tt.name == "exhausted" {
				var got map[string]any
				if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if got["msg"] != "notification fan-out exhausted" || got["level"] != "ERROR" || got["job_id"] != float64(73) ||
					got["event_id"] != tt.event.ID.String() || got["type"] != tt.event.Type || got["error"] != failure.Error() {
					t.Fatal(got)
				}
			} else if buf.Len() != 0 {
				t.Fatalf("unexpected exhausted log %s", buf.String())
			}
			if declaration := FanoutJob(worker.uc, logger); declaration.Worker == nil || declaration.Periodic != nil {
				t.Fatal(declaration)
			}
		})
	}
}
