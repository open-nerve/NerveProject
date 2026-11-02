package bootstrap

import (
	"context"
	"errors"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

type purgeArgs struct{}

func (purgeArgs) Kind() string { return "cleanup.purge_deleted" }

type purgeWorker struct {
	river.WorkerDefaults[purgeArgs]
	uc *purgeDeleted
}

func (w *purgeWorker) Work(ctx context.Context, job *river.Job[purgeArgs]) error {
	previous, present, err := decodePurgeOutput(job.Metadata)
	if err != nil {
		return err
	}
	if present {
		index, err := w.uc.cursorModule(previous.Cursor)
		if err != nil {
			return err
		}
		if previous.Complete {
			if index != len(w.uc.modules)-1 {
				return errors.New("completed purge has unfinished modules")
			}
			return nil
		}
	}
	result, workErr := w.uc.Execute(ctx, previous.Cursor)
	output, err := encodePurgeOutput(result)
	if err != nil {
		return errors.Join(workErr, err)
	}
	if err := river.RecordOutput(ctx, output); err != nil {
		return errors.Join(workErr, err)
	}
	if workErr != nil {
		return workErr
	}
	if !result.Complete {
		return river.JobSnooze(0)
	}
	return nil
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
