package riveradapter

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

// TrimVersionsKind is the kind of the job that trims older live description versions, and
// the id of its schedule.
const TrimVersionsKind = "issue.trim_description_versions"

// TrimVersionsUseCase is the use case the job runs: app.TrimVersions.
type TrimVersionsUseCase interface {
	Execute(ctx context.Context) (int, error)
}

// TrimVersionsArgs are the job's arguments: none.
type TrimVersionsArgs struct{}

// Kind is TrimVersionsKind.
func (TrimVersionsArgs) Kind() string { return TrimVersionsKind }

// TrimVersionsWorker runs description retention.
type TrimVersionsWorker struct {
	river.WorkerDefaults[TrimVersionsArgs]
	uc TrimVersionsUseCase
}

// NewTrimVersionsWorker returns the worker of uc.
func NewTrimVersionsWorker(uc TrimVersionsUseCase) *TrimVersionsWorker {
	return &TrimVersionsWorker{uc: uc}
}

// Work trims older live description versions. A failure makes River retry the job.
func (w *TrimVersionsWorker) Work(ctx context.Context, _ *river.Job[TrimVersionsArgs]) error {
	_, err := w.uc.Execute(ctx)
	return err
}

// TrimVersionsJob schedules description retention when the server starts, then every
// interval (issue.trim_versions_interval).
func TrimVersionsJob(uc TrimVersionsUseCase, interval time.Duration) jobs.Job {
	return jobs.Job{
		Worker: jobs.Register(NewTrimVersionsWorker(uc)),
		Periodic: &jobs.PeriodicSpec{Args: TrimVersionsArgs{}, Interval: interval, ID: TrimVersionsKind, RunOnStart: true,
			InsertOpts: &river.InsertOpts{Queue: "periodic"}},
	}
}
