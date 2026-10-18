package bootstrap

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	platformgate "github.com/open-nerve/NerveProject/server/internal/platform/gate"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The work-item side's write gate and lock timeout (M4 design 3.5, 11.11),
// and the one place that answers either's refusal as 503 server_busy: a
// project too busy for one more write now, which the client tries again
// after Retry-After.

// busyRetry is the Retry-After of a write refused as server_busy.
const busyRetry = time.Second

// writeGate is platform/gate's gate per project as issue's WriteGate.
type writeGate struct {
	gate *platformgate.Keyed[uuid.UUID]
}

// newWriteGate is the write gate of issue.write_gate.
func newWriteGate(c config.WriteGateConfig) writeGate {
	limits := platformgate.Limits{PerKey: c.PerProject, MaxQueue: c.MaxQueue, MaxWait: c.MaxWait}
	return writeGate{gate: platformgate.New[uuid.UUID](limits)}
}

// Enter enters the project's gate: a write the gate refuses, its queue
// full or its wait at max_wait, or whose request reaches its deadline
// while it waits, is server_busy; one whose request is cancelled keeps
// context.Canceled, which no one waits for the answer of.
func (g writeGate) Enter(ctx context.Context, projectID uuid.UUID) (func(), error) {
	leave, err := g.gate.Enter(ctx, projectID)
	if errors.Is(err, platformgate.ErrFull) || errors.Is(err, platformgate.ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
		return nil, shared.ServerBusy(busyRetry)
	}
	return leave, err
}

// busyTx are the transactions of the work-item side's writes:
// platform/postgres's with issue.write_lock_timeout, each of whose waits
// for a lock ends there; a transaction ended so (55P03) is server_busy.
type busyTx struct {
	tx shared.TxManager
}

// newIssueTx is the work-item side's transactions on pool.
func newIssueTx(pool *pgxpool.Pool, cfg config.Config) busyTx {
	return busyTx{tx: postgres.NewTxManager(pool, cfg.Database.CommitTimeout, postgres.LockTimeout(cfg.Issue.WriteLockTimeout))}
}

func (b busyTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	err := b.tx.WithinTx(ctx, fn)
	if postgres.LockNotAvailable(err) {
		return shared.ServerBusy(busyRetry)
	}
	return err
}
