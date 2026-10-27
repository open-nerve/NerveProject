package riveradapter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/riverqueue/river"

	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
)

func TestTrimVersionsWorkerPreservesContextAndFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	boom := errors.New("lock failed")
	uc := &archiveCall{err: boom}
	if err := riveradapter.NewTrimVersionsWorker(uc).Work(ctx, &river.Job[riveradapter.TrimVersionsArgs]{}); err != boom || uc.ctx != ctx {
		t.Fatalf("worker err=%v context=%v", err, uc.ctx)
	}
	job := riveradapter.TrimVersionsJob(uc, 2*time.Second)
	if job.Worker == nil || job.Periodic.Args.Kind() != "issue.trim_description_versions" || job.Periodic.Interval != 2*time.Second || job.Periodic.ID != "issue.trim_description_versions" || !job.Periodic.RunOnStart || job.Periodic.InsertOpts.Queue != "periodic" {
		t.Fatalf("job = %+v", job)
	}
}
