package bootstrap

import (
	"context"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

// Purger physically removes the module's old tombstones in bounded batches.
type Purger interface {
	Purge(context.Context, time.Time, int) (int, error)
}
type purgeClock interface{ Now() time.Time }
type purgeDeleted struct {
	modules []Purger
	clock   purgeClock
	after   time.Duration
	batch   int
}

func newPurgeDeleted(modules []Purger, clock purgeClock, after time.Duration, batch int) *purgeDeleted {
	return &purgeDeleted{modules: modules, clock: clock, after: after, batch: batch}
}

// Execute uses one cutoff for the ordered modules and retains every committed
// batch, including progress returned together with an error.
func (p *purgeDeleted) Execute(ctx context.Context) (int, error) {
	cutoff := p.clock.Now().Add(-p.after)
	total := 0
	for _, module := range p.modules {
		count, err := module.Purge(ctx, cutoff, p.batch)
		total += count
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

type purgeArgs struct{}

func (purgeArgs) Kind() string { return "cleanup.purge_deleted" }

type purgeWorker struct {
	river.WorkerDefaults[purgeArgs]
	uc *purgeDeleted
}

func (w *purgeWorker) Work(ctx context.Context, _ *river.Job[purgeArgs]) error {
	_, err := w.uc.Execute(ctx)
	return err
}

// purgeJob excludes completed jobs from uniqueness so the next period can run,
// while an old running or retryable job prevents a new overlapping cleanup.
func purgeJob(uc *purgeDeleted, interval time.Duration) jobs.Job {
	return jobs.Job{
		Worker: jobs.Register(&purgeWorker{uc: uc}),
		Periodic: &jobs.PeriodicSpec{
			Args: purgeArgs{}, ID: "cleanup.purge_deleted", Interval: interval, RunOnStart: true,
			InsertOpts: &river.InsertOpts{Queue: "periodic", UniqueOpts: river.UniqueOpts{ByState: []rivertype.JobState{
				rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRunning,
				rivertype.JobStateRetryable, rivertype.JobStateScheduled,
			}}},
		},
	}
}
