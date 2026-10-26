// Package riveradapter works the issue module's background jobs on River
// (M4 design 3.5): each worker only runs a use case.
package riveradapter

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

// AutoArchiveKind is the kind of the job that archives inactive terminal work items, and
// the id of its schedule.
const AutoArchiveKind = "issue.auto_archive"

// AutoArchiveUseCase is the use case the job runs: app.AutoArchive.
type AutoArchiveUseCase interface {
	Execute(ctx context.Context) (int, error)
}

// AutoArchiveArgs are the job's arguments: none.
type AutoArchiveArgs struct{}

// Kind is AutoArchiveKind.
func (AutoArchiveArgs) Kind() string { return AutoArchiveKind }

// AutoArchiveWorker runs automatic archival.
type AutoArchiveWorker struct {
	river.WorkerDefaults[AutoArchiveArgs]
	uc AutoArchiveUseCase
}

// NewAutoArchiveWorker returns the worker of uc.
func NewAutoArchiveWorker(uc AutoArchiveUseCase) *AutoArchiveWorker {
	return &AutoArchiveWorker{uc: uc}
}

// Work archives inactive terminal work items. A failure makes River retry the job.
func (w *AutoArchiveWorker) Work(ctx context.Context, _ *river.Job[AutoArchiveArgs]) error {
	_, err := w.uc.Execute(ctx)
	return err
}

// AutoArchiveJob schedules archival when the server starts, then every
// interval (issue.auto_archive_interval).
func AutoArchiveJob(uc AutoArchiveUseCase, interval time.Duration) jobs.Job {
	return jobs.Job{
		Worker: jobs.Register(NewAutoArchiveWorker(uc)),
		Periodic: &jobs.PeriodicSpec{Args: AutoArchiveArgs{}, Interval: interval, ID: AutoArchiveKind, RunOnStart: true,
			InsertOpts: &river.InsertOpts{Queue: "periodic"}},
	}
}
