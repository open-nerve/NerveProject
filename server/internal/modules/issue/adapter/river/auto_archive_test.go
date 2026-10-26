package riveradapter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/riverqueue/river"

	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
)

type archiveCall struct {
	ctx context.Context
	err error
}

func (a *archiveCall) Execute(ctx context.Context) (int, error) { a.ctx = ctx; return 3, a.err }
func TestAutoArchiveWorkerPreservesContextAndFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	boom := errors.New("lock failed")
	uc := &archiveCall{err: boom}
	if err := riveradapter.NewAutoArchiveWorker(uc).Work(ctx, &river.Job[riveradapter.AutoArchiveArgs]{}); err != boom || uc.ctx != ctx {
		t.Fatalf("worker err=%v context=%v", err, uc.ctx)
	}
	job := riveradapter.AutoArchiveJob(uc, 2*time.Second)
	if job.Worker == nil || job.Periodic.Args.Kind() != "issue.auto_archive" || job.Periodic.Interval != 2*time.Second || job.Periodic.ID != "issue.auto_archive" || !job.Periodic.RunOnStart || job.Periodic.InsertOpts.Queue != "periodic" {
		t.Fatalf("job = %+v", job)
	}
}
