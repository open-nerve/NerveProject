package riveradapter

import (
	"context"
	"log/slog"
	"time"
	"uuid"

	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

// DeleteRowsKind identifies the request-driven deletion of a deleted
// project's notifications.
const DeleteRowsKind = "notification.delete_rows"

// issueWait is how long a run that found the project's work items still
// being deleted snoozes the job: a fixed positive wait, not configured,
// apart from the budget's JobSnooze(0) (M7 P1b spec 2.8).
const issueWait = 500 * time.Millisecond

// DeleteRowsArgs keeps the project and the deletion's moment and actor across
// retries.
type DeleteRowsArgs struct {
	ProjectID uuid.UUID `json:"project_id"`
	At        time.Time `json:"at"`
	By        uuid.UUID `json:"by"`
}

func (DeleteRowsArgs) Kind() string { return DeleteRowsKind }

// InsertOpts leaves MaxAttempts unset to keep River's default of 25.
func (DeleteRowsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault}
}

// DeleteRowsUseCase keeps River out of the application layer.
type DeleteRowsUseCase interface {
	Execute(ctx context.Context, project uuid.UUID, at time.Time, by uuid.UUID, enqueued time.Time) (app.DeleteRowsResult, error)
}

// DeleteRowsWorker hands the use case the job's enqueueing, snoozes while
// the project's work items are being deleted and at the budget, and logs the
// last failed attempt.
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
	result, err := w.uc.Execute(ctx, a.ProjectID, a.At, a.By, job.CreatedAt)
	if err != nil {
		if job.Attempt >= job.MaxAttempts {
			w.logger.ErrorContext(ctx, "notification row deletion exhausted", slog.Int64("job_id", job.ID), slog.String("project_id", a.ProjectID.String()), slog.Any("error", err))
		}
		return err
	}
	switch {
	case result.Waiting:
		return river.JobSnooze(issueWait)
	case !result.Complete:
		return river.JobSnooze(0)
	}
	return nil
}

func (w *DeleteRowsWorker) Timeout(*river.Job[DeleteRowsArgs]) time.Duration {
	return w.budget + 5*time.Second
}

// DeleteRowsJob declares the request-driven worker.
func DeleteRowsJob(uc DeleteRowsUseCase, budget time.Duration, logger *slog.Logger) jobs.Job {
	return jobs.Job{Worker: jobs.Register(NewDeleteRowsWorker(uc, budget, logger))}
}
