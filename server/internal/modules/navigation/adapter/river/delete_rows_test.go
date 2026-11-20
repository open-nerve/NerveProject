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

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
)

var _ app.RowDeletions = (*RowDeletions)(nil)

type deleteRowsCall func(context.Context, app.DeletionScope, uuid.UUID, time.Time, uuid.UUID) (app.DeleteRowsResult, error)

func (f deleteRowsCall) Execute(ctx context.Context, scope app.DeletionScope, id uuid.UUID, at time.Time, by uuid.UUID) (app.DeleteRowsResult, error) {
	return f(ctx, scope, id, at, by)
}

// A complete run ends the job; a run stopped at its budget snoozes it to continue at once; a failure retries it and
// the last failed attempt is logged with the job, its scope and its parent (P3b spec 2.7). The job's arguments reach
// the use case as they were enqueued. The job's timeout is its budget and five seconds.
func TestDeleteRowsWorkerMapsBudgetFailureAndTimeout(t *testing.T) {
	args := DeleteRowsArgs{Scope: app.WorkspaceScope, ID: uuid.NewV7(), At: time.Now().Add(-61 * 24 * time.Hour), By: uuid.NewV7()}
	failure := errors.New("database unavailable")
	for _, tt := range []struct {
		name    string
		result  app.DeleteRowsResult
		err     error
		attempt int
		snooze  bool
	}{
		{"complete", app.DeleteRowsResult{Deleted: 2, Complete: true}, nil, 25, false},
		{"budget", app.DeleteRowsResult{Deleted: 3}, nil, 25, true},
		{"retry", app.DeleteRowsResult{}, failure, 24, false},
		{"exhausted", app.DeleteRowsResult{}, failure, 25, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, nil))
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			uc := deleteRowsCall(func(got context.Context, scope app.DeletionScope, id uuid.UUID, at time.Time, by uuid.UUID) (app.DeleteRowsResult, error) {
				if got != ctx || scope != args.Scope || id != args.ID || at != args.At || by != args.By {
					t.Error("lost arguments or context")
				}
				return tt.result, tt.err
			})
			worker := NewDeleteRowsWorker(uc, 45*time.Second, logger)
			job := &river.Job[DeleteRowsArgs]{JobRow: &rivertype.JobRow{ID: 73, Attempt: tt.attempt, MaxAttempts: 25}, Args: args}
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
			case tt.snooze:
				if !errors.As(err, &snooze) || snooze.Duration != 0 {
					t.Fatalf("want snooze(0), got %v", err)
				}
			case err != nil:
				t.Fatal(err)
			}
			if tt.name == "exhausted" {
				var got map[string]any
				if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if got["msg"] != "navigation row deletion exhausted" || got["level"] != "ERROR" || got["job_id"] != float64(73) || got["scope"] != "workspace" ||
					got["id"] != args.ID.String() || got["error"] != failure.Error() {
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

// The job carries the scope, the parent and the deletion's moment and actor, on the default queue with River's
// default attempts, inserted in the caller's context (its transaction).
func TestRowDeletionsEnqueue(t *testing.T) {
	ctx := t.Context()
	at := time.Date(2026, 11, 20, 1, 2, 3, 123456000, time.UTC)
	id, by := uuid.NewV7(), uuid.NewV7()
	for _, scope := range []app.DeletionScope{app.ProjectScope, app.WorkspaceScope} {
		failure := errors.New("insert failed")
		calls := 0
		q := NewRowDeletions(insertCall(func(got context.Context, args river.JobArgs, opts *river.InsertOpts) error {
			calls++
			if got != ctx || args.Kind() != "navigation.delete_rows" {
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
			want := map[string]any{"scope": string(scope), "id": id.String(), "at": at.Format(time.RFC3339Nano), "by": by.String()}
			if !reflect.DeepEqual(payload, want) {
				t.Fatalf("payload=%v", payload)
			}
			if opts == nil || !reflect.DeepEqual(*opts, river.InsertOpts{Queue: "default"}) {
				t.Fatalf("opts=%+v", opts)
			}
			return failure
		}))
		if err := q.Enqueue(ctx, scope, id, at, by); !errors.Is(err, failure) || calls != 1 {
			t.Fatalf("%s: Enqueue=%v calls=%d; want the inserter's failure once", scope, err, calls)
		}
	}
}
