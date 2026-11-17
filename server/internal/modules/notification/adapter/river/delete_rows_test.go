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

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
)

type deleteRowsCall func(context.Context, uuid.UUID, time.Time, uuid.UUID, time.Time) (app.DeleteRowsResult, error)

func (f deleteRowsCall) Execute(ctx context.Context, project uuid.UUID, at time.Time, by uuid.UUID, enqueued time.Time) (app.DeleteRowsResult, error) {
	return f(ctx, project, at, by, enqueued)
}

// A complete run ends the job; a run waiting for the project's work items snoozes it for half a second, a fixed
// positive wait apart from the budget's (M7 P1b spec 2.8); a run stopped at its budget snoozes it to continue at once;
// a failure retries it and the last failed attempt is logged. The job's enqueueing reaches the use case, which bounds
// the wait. The job's timeout is its budget and five seconds.
func TestDeleteRowsWorkerMapsWaitBudgetAndTimeout(t *testing.T) {
	args := DeleteRowsArgs{ProjectID: uuid.NewV7(), At: time.Now().Add(-61 * 24 * time.Hour), By: uuid.NewV7()}
	created := time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)
	failure := errors.New("database unavailable")
	for _, tt := range []struct {
		name    string
		result  app.DeleteRowsResult
		err     error
		attempt int
		snooze  time.Duration
	}{
		{"complete", app.DeleteRowsResult{Complete: true}, nil, 25, -1},
		{"waiting", app.DeleteRowsResult{Waiting: true}, nil, 25, 500 * time.Millisecond},
		{"budget", app.DeleteRowsResult{Deleted: 3}, nil, 25, 0},
		{"retry", app.DeleteRowsResult{}, failure, 24, -1},
		{"exhausted", app.DeleteRowsResult{}, failure, 25, -1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, nil))
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			uc := deleteRowsCall(func(got context.Context, project uuid.UUID, at time.Time, by uuid.UUID, enqueued time.Time) (app.DeleteRowsResult, error) {
				if got != ctx || project != args.ProjectID || at != args.At || by != args.By || !enqueued.Equal(created) {
					t.Error("lost arguments, context or enqueueing")
				}
				return tt.result, tt.err
			})
			worker := NewDeleteRowsWorker(uc, 45*time.Second, logger)
			job := &river.Job[DeleteRowsArgs]{JobRow: &rivertype.JobRow{ID: 73, Attempt: tt.attempt, MaxAttempts: 25, CreatedAt: created}, Args: args}
			if worker.Timeout(job) != 50*time.Second || NewDeleteRowsWorker(uc, time.Second, logger).Timeout(job) != 6*time.Second {
				t.Fatal("timeout")
			}
			err := worker.Work(ctx, job)
			var snooze *river.JobSnoozeError
			switch {
			case tt.err != nil:
				if err != failure {
					t.Fatal(err)
				}
			case tt.snooze >= 0:
				if !errors.As(err, &snooze) || snooze.Duration != tt.snooze {
					t.Fatalf("want snooze(%s), got %v", tt.snooze, err)
				}
			case err != nil:
				t.Fatal(err)
			}
			if tt.name == "exhausted" {
				var got map[string]any
				if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if got["msg"] != "notification row deletion exhausted" || got["level"] != "ERROR" || got["job_id"] != float64(73) || got["project_id"] != args.ProjectID.String() || got["error"] != failure.Error() {
					t.Fatal(got)
				}
			} else if buf.Len() != 0 {
				t.Fatalf("unexpected exhausted log %s", buf.String())
			}
			if declaration := DeleteRowsJob(uc, 45*time.Second, logger); declaration.Worker == nil || declaration.Periodic != nil {
				t.Fatal(declaration)
			}
		})
	}
}

type insertCall func(context.Context, river.JobArgs, *river.InsertOpts) error

func (f insertCall) InsertTx(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) error {
	return f(ctx, args, opts)
}

// The job carries the project and the deletion's moment and actor, on the default queue with River's default
// attempts, inserted in the caller's context (its transaction).
func TestRowDeletionsEnqueue(t *testing.T) {
	ctx := t.Context()
	at := time.Date(2026, 10, 28, 1, 2, 3, 123456000, time.UTC)
	project, by := uuid.NewV7(), uuid.NewV7()
	failure := errors.New("insert failed")
	calls := 0
	q := NewRowDeletions(insertCall(func(got context.Context, args river.JobArgs, opts *river.InsertOpts) error {
		calls++
		if got != ctx || args.Kind() != "notification.delete_rows" {
			t.Fatal("context/kind")
		}
		raw, err := json.Marshal(args)
		if err != nil {
			t.Fatal(err)
		}
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatal(err)
		}
		want := map[string]any{"project_id": project.String(), "at": at.Format(time.RFC3339Nano), "by": by.String()}
		if !reflect.DeepEqual(payload, want) {
			t.Fatalf("payload=%v", payload)
		}
		if opts == nil || !reflect.DeepEqual(*opts, river.InsertOpts{Queue: "default"}) {
			t.Fatalf("opts=%+v", opts)
		}
		return failure
	}))
	if err := q.Enqueue(ctx, project, at, by); !errors.Is(err, failure) || calls != 1 {
		t.Fatalf("Enqueue=%v calls=%d; want the inserter's failure once", err, calls)
	}
}
