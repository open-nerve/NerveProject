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

type insertCalls func(context.Context, river.JobArgs, *river.InsertOpts) error

func (f insertCalls) InsertTx(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) error {
	return f(ctx, args, opts)
}

// The job carries the event as the subscriber hands it over, in the caller's context (its transaction), on the
// default queue with five attempts: its id, type, moment, workspace, project, actor, entity and every change in order,
// a nil value or id left out, so "" stays apart from none.
func TestFanoutsEnqueueTheEvent(t *testing.T) {
	ctx := t.Context()
	e := fanoutEvent()
	failure := errors.New("insert failed")
	calls := 0
	q := NewFanouts(insertCalls(func(got context.Context, args river.JobArgs, opts *river.InsertOpts) error {
		calls++
		if got != ctx || args.Kind() != "notification.fanout" {
			t.Fatal("context/kind")
		}
		if opts == nil || !reflect.DeepEqual(*opts, river.InsertOpts{Queue: "default", MaxAttempts: 5}) {
			t.Fatalf("opts=%+v", opts)
		}
		raw, err := json.Marshal(args)
		if err != nil {
			t.Fatal(err)
		}
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatal(err)
		}
		want := map[string]any{"event_id": e.ID.String(), "type": e.Type, "occurred_at": "2026-11-17T09:30:00.123456789Z",
			"workspace_id": e.WorkspaceID.String(), "project_id": e.ProjectID.String(), "actor_id": e.ActorID.String(),
			"entity_kind": "issue_comment", "entity_id": e.Entity.ID.String(), "changes": []any{
				map[string]any{"field": "comment_html"},
				map[string]any{"field": "mention", "new_id": e.Changes[1].NewID.String()},
				map[string]any{"field": "reply", "old": "", "new": "x"},
				map[string]any{"field": "issue", "new_id": e.Changes[3].NewID.String()}}}
		if !reflect.DeepEqual(payload, want) {
			t.Fatalf("payload=%v\nwant     %v", payload, want)
		}
		return failure
	}))
	if err := q.Enqueue(ctx, e); !errors.Is(err, failure) || calls != 1 {
		t.Fatalf("Enqueue=%v calls=%d; want the inserter's failure once", err, calls)
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
			if err := NewFanouts(insertCalls(func(_ context.Context, args river.JobArgs, _ *river.InsertOpts) (err error) {
				raw, err = json.Marshal(args)
				return err
			})).Enqueue(t.Context(), tt.event); err != nil {
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
