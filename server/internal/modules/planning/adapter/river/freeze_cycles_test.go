package riveradapter

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/riverqueue/river"
)

type freezeCall struct {
	ctx context.Context
	err error
}

func (f *freezeCall) Execute(ctx context.Context) (int, error) { f.ctx = ctx; return 2, f.err }

// The worker runs the use case on its context and fails with its error, kept
// whole for River's retry; the job is planning.freeze_cycles on the periodic
// queue, at its interval, also when the server starts (M6 P4 spec 2.5).
func TestFreezeCyclesWorkerAndSchedule(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	boom := errors.New("lock failed")
	uc := &freezeCall{err: boom}
	if err := NewFreezeCyclesWorker(uc).Work(ctx, &river.Job[FreezeCyclesArgs]{}); err != boom || uc.ctx != ctx {
		t.Fatalf("worker err=%v context=%v", err, uc.ctx)
	}
	job := FreezeCyclesJob(uc, 2*time.Second)
	if job.Worker == nil || job.Periodic == nil || job.Periodic.Args.Kind() != "planning.freeze_cycles" || job.Periodic.Interval != 2*time.Second ||
		job.Periodic.ID != "planning.freeze_cycles" || !job.Periodic.RunOnStart || job.Periodic.InsertOpts == nil || job.Periodic.InsertOpts.Queue != "periodic" {
		t.Fatalf("job = %+v", job)
	}
}
