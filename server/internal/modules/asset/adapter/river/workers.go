package riveradapter

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

const CleanupKind = "asset.cleanup"

type CleanupArgs struct{}

func (CleanupArgs) Kind() string { return CleanupKind }

type CleanupUseCase interface {
	Execute(context.Context, app.SweepCursor) (app.SweepResult, error)
}
type DeleteScopeUseCase interface {
	Execute(context.Context, domain.ScopeKind, uuid.UUID, app.DeletionStamp, app.SweepCursor) (app.SweepResult, error)
}
type CleanupWorker struct {
	river.WorkerDefaults[CleanupArgs]
	uc     CleanupUseCase
	budget time.Duration
}

func NewCleanupWorker(uc CleanupUseCase, budget time.Duration) *CleanupWorker {
	return &CleanupWorker{uc: uc, budget: budget}
}
func (w *CleanupWorker) Work(ctx context.Context, job *river.Job[CleanupArgs]) error {
	previous, present, err := decodeOutput(job.Metadata, "rows")
	if err != nil {
		return err
	}
	if present && previous.Complete {
		return nil
	}
	result, err := w.uc.Execute(ctx, previous.Cursor)
	return finishSweep(ctx, result, err, "rows")
}
func (w *CleanupWorker) Timeout(*river.Job[CleanupArgs]) time.Duration {
	return w.budget + 5*time.Second
}

type DeleteScopeWorker struct {
	river.WorkerDefaults[DeleteScopeArgs]
	uc     DeleteScopeUseCase
	budget time.Duration
}

func NewDeleteScopeWorker(uc DeleteScopeUseCase, budget time.Duration) *DeleteScopeWorker {
	return &DeleteScopeWorker{uc: uc, budget: budget}
}
func (w *DeleteScopeWorker) Work(ctx context.Context, job *river.Job[DeleteScopeArgs]) error {
	previous, present, err := decodeOutput(job.Metadata, "scope")
	if err != nil {
		return err
	}
	if present && !previous.Cursor.Cutoff.Equal(job.Args.At) {
		return errors.New("asset scope output changed original time")
	}
	if present && previous.Complete {
		return nil
	}
	a := job.Args
	result, err := w.uc.Execute(ctx, a.Scope, a.ID, app.DeletionStamp{At: a.At, By: a.By}, previous.Cursor)
	return finishSweep(ctx, result, err, "scope")
}
func (w *DeleteScopeWorker) Timeout(*river.Job[DeleteScopeArgs]) time.Duration {
	return w.budget + 5*time.Second
}
