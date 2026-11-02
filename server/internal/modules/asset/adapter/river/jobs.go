package riveradapter

import (
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

func CleanupJob(uc CleanupUseCase, interval, budget time.Duration) jobs.Job {
	return jobs.Job{Worker: jobs.Register(NewCleanupWorker(uc, budget)), Periodic: &jobs.PeriodicSpec{Args: CleanupArgs{}, ID: CleanupKind, Interval: interval, RunOnStart: true, InsertOpts: &river.InsertOpts{Queue: "periodic", UniqueOpts: river.UniqueOpts{ByState: []rivertype.JobState{rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRunning, rivertype.JobStateRetryable, rivertype.JobStateScheduled}}}}}
}
func DeleteScopeJob(uc DeleteScopeUseCase, budget time.Duration) jobs.Job {
	return jobs.Job{Worker: jobs.Register(NewDeleteScopeWorker(uc, budget))}
}
