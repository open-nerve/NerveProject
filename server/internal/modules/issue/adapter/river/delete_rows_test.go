package riveradapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"
	"uuid"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

type deleteRowsCall func(context.Context, string, uuid.UUID, time.Time, uuid.UUID) (app.DeleteRowsResult, error)

func (f deleteRowsCall) Execute(ctx context.Context, s string, id uuid.UUID, at time.Time, by uuid.UUID) (app.DeleteRowsResult, error) {
	return f(ctx, s, id, at, by)
}
func TestDeleteRowsWorkerMapsBudgetAndTimeout(t *testing.T) {
	args := DeleteRowsArgs{Scope: "project", ID: uuid.NewV7(), At: time.Now().Add(-61 * 24 * time.Hour), By: uuid.NewV7()}
	failure := errors.New("database unavailable")
	for _, tt := range []struct {
		name     string
		complete bool
		err      error
		attempt  int
	}{{"complete", true, nil, 25}, {"snooze", false, nil, 25}, {"retry", false, failure, 24}, {"exhausted", false, failure, 25}} {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, nil))
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			uc := deleteRowsCall(func(got context.Context, s string, id uuid.UUID, at time.Time, by uuid.UUID) (app.DeleteRowsResult, error) {
				if got != ctx || s != args.Scope || id != args.ID || at != args.At || by != args.By {
					t.Error("lost arguments/context")
				}
				return app.DeleteRowsResult{Complete: tt.complete}, tt.err
			})
			worker := NewDeleteRowsWorker(uc, 45*time.Second, logger)
			job := &river.Job[DeleteRowsArgs]{JobRow: &rivertype.JobRow{ID: 73, Attempt: tt.attempt, MaxAttempts: 25}, Args: args}
			if worker.Timeout(job) != 50*time.Second || NewDeleteRowsWorker(uc, time.Second, logger).Timeout(job) != 6*time.Second {
				t.Fatal("timeout")
			}
			err := worker.Work(ctx, job)
			var snooze *river.JobSnoozeError
			if tt.err != nil {
				if err != failure || !errors.Is(err, failure) {
					t.Fatal(err)
				}
			} else if !tt.complete {
				if !errors.As(err, &snooze) || snooze.Duration != 0 {
					t.Fatalf("want snooze(0), got %v", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if tt.name == "exhausted" {
				var got map[string]any
				if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				if got["msg"] != "issue row deletion exhausted" || got["level"] != "ERROR" || got["job_id"] != float64(73) || got["scope"] != args.Scope || got["id"] != args.ID.String() || got["error"] != failure.Error() {
					t.Fatal(got)
				}
			} else if buf.Len() != 0 {
				t.Fatalf("unexpected exhausted log %s", buf.String())
			}
			declaration := DeleteRowsJob(uc, 45*time.Second, logger)
			if declaration.Worker == nil || declaration.Periodic != nil {
				t.Fatal(declaration)
			}
		})
	}
}
