package riveradapter

import (
	"context"
	"time"

	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

// FreezeCyclesKind is the kind of the job that freezes the progress of the
// cycles that have ended, and the id of its schedule (M6 P4 spec 2.5).
const FreezeCyclesKind = "planning.freeze_cycles"

// FreezeCyclesUseCase is the use case the job runs: app.FreezeCycles.
type FreezeCyclesUseCase interface {
	Execute(ctx context.Context) (int, error)
}

// FreezeCyclesArgs are the job's arguments: none.
type FreezeCyclesArgs struct{}

// Kind is FreezeCyclesKind.
func (FreezeCyclesArgs) Kind() string { return FreezeCyclesKind }

// FreezeCyclesWorker runs the freeze.
type FreezeCyclesWorker struct {
	river.WorkerDefaults[FreezeCyclesArgs]
	uc FreezeCyclesUseCase
}

// NewFreezeCyclesWorker returns the worker of uc.
func NewFreezeCyclesWorker(uc FreezeCyclesUseCase) *FreezeCyclesWorker {
	return &FreezeCyclesWorker{uc: uc}
}

// Work freezes the cycles that have ended. A failure makes River retry the
// job; the projects committed before it stay.
func (w *FreezeCyclesWorker) Work(ctx context.Context, _ *river.Job[FreezeCyclesArgs]) error {
	_, err := w.uc.Execute(ctx)
	return err
}

// FreezeCyclesJob schedules the freeze on the periodic queue when the server
// starts, then every interval (planning.freeze_interval).
func FreezeCyclesJob(uc FreezeCyclesUseCase, interval time.Duration) jobs.Job {
	return jobs.Job{
		Worker: jobs.Register(NewFreezeCyclesWorker(uc)),
		Periodic: &jobs.PeriodicSpec{Args: FreezeCyclesArgs{}, Interval: interval, ID: FreezeCyclesKind, RunOnStart: true,
			InsertOpts: &river.InsertOpts{Queue: "periodic"}},
	}
}
