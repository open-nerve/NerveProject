package riveradapter

import (
	"context"
	"log/slog"
	"time"
	"uuid"

	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

// DeleteRowsKind identifies request-driven soft deletion batches.
const DeleteRowsKind = "issue.delete_rows"

// DeleteRowsArgs preserves the request scope and audit data across retries.
type DeleteRowsArgs struct {
	Scope string    `json:"scope"`
	ID    uuid.UUID `json:"id"`
	At    time.Time `json:"at"`
	By    uuid.UUID `json:"by"`
}

func (DeleteRowsArgs) Kind() string { return DeleteRowsKind }

// InsertOpts leaves MaxAttempts unset to retain River's default of 25.
func (DeleteRowsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault}
}

// DeleteRowsUseCase keeps River out of the application layer.
type DeleteRowsUseCase interface {
	Execute(context.Context, string, uuid.UUID, time.Time, uuid.UUID) (app.DeleteRowsResult, error)
}

// DeleteRowsWorker snoozes budget boundaries and reports exhausted real failures.
type DeleteRowsWorker struct {
	river.WorkerDefaults[DeleteRowsArgs]
	uc     DeleteRowsUseCase
	budget time.Duration
	logger *slog.Logger
}

func NewDeleteRowsWorker(uc DeleteRowsUseCase, budget time.Duration, logger *slog.Logger) *DeleteRowsWorker {
	return &DeleteRowsWorker{uc: uc, budget: budget, logger: logger}
}
func (w *DeleteRowsWorker) Work(ctx context.Context, job *river.Job[DeleteRowsArgs]) error {
	a := job.Args
	result, err := w.uc.Execute(ctx, a.Scope, a.ID, a.At, a.By)
	if err != nil {
		if job.Attempt >= job.MaxAttempts {
			w.logger.ErrorContext(ctx, "issue row deletion exhausted", slog.Int64("job_id", job.ID), slog.String("scope", a.Scope), slog.String("id", a.ID.String()), slog.Any("error", err))
		}
		return err
	}
	if !result.Complete {
		return river.JobSnooze(0)
	}
	return nil
}
func (w *DeleteRowsWorker) Timeout(*river.Job[DeleteRowsArgs]) time.Duration {
	return w.budget + 5*time.Second
}
func DeleteRowsJob(uc DeleteRowsUseCase, budget time.Duration, logger *slog.Logger) jobs.Job {
	return jobs.Job{Worker: jobs.Register(NewDeleteRowsWorker(uc, budget, logger))}
}
