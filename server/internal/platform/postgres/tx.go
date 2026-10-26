package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier runs statements: the transaction WithinTx opened, or the pool.
// sqlc's generated DBTX interface has the same methods, so repositories pass
// DB(ctx, pool) to their generated queries.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txKey struct{}

// Tx returns the original transaction carried by ctx, without falling back
// to a pool. Adapters that require an existing transaction use it at the
// composition root.
func Tx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}

// DB returns the transaction that TxManager.WithinTx put in ctx, or pool
// when ctx carries none.
func DB(ctx context.Context, pool *pgxpool.Pool) Querier {
	if tx, ok := Tx(ctx); ok {
		return tx
	}
	return pool
}

// TxManager runs functions in transactions on one pool. It satisfies
// shared.TxManager by structure; the platform does not import shared.
//
// The statements of a transaction run under the caller's context and are
// cancelled with it, e.g. when the request deadline passes. COMMIT and
// ROLLBACK are not: they run under context.WithoutCancel, bounded by their own
// commitTimeout (database.commit_timeout). A transaction whose statements
// have all finished therefore commits even if the request is cancelled
// meanwhile, and a failed one is rolled back and its connection returned to
// the pool in a known state (M2 design 3.6). A COMMIT that gets no answer
// within commitTimeout has an unknown outcome and returns an error. A ROLLBACK
// that fails is an infrastructure fault: WithinTx returns an error that wraps
// the failure and keeps only the text of fn's error, so a domain error behind
// it is answered as a logged 500, not as itself.
type TxManager struct {
	pool             *pgxpool.Pool
	commitTimeout    time.Duration
	begin            pgx.TxOptions
	lockTimeout      time.Duration
	readOnlySnapshot bool
}

// Option sets how a TxManager begins its transactions.
type Option func(m *TxManager)

// LockTimeout makes each transaction the TxManager begins wait at most d,
// rounded up to whole milliseconds, for each lock it waits for: its BEGIN
// sets lock_timeout for the transaction alone (SET LOCAL), in the same
// round trip, before any statement of fn. A statement that waits longer
// fails with SQLSTATE 55P03 (LockNotAvailable), and the transaction rolls
// back (M4 design 3.5, 11.11). d must be positive: PostgreSQL reads a
// lock_timeout of 0 as no limit.
func LockTimeout(d time.Duration) Option {
	return func(m *TxManager) {
		m.lockTimeout = d
	}
}

// ReadOnlySnapshot begins an independent transaction as REPEATABLE READ,
// READ ONLY. All reads, including authorization and directory lookups, must
// use its context. WithinTx joins an existing transaction without changing
// its isolation or access mode; callers must not nest snapshot lists in a
// write transaction.
func ReadOnlySnapshot() Option {
	return func(m *TxManager) { m.readOnlySnapshot = true }
}

// NewTxManager returns a TxManager on pool, with opts.
func NewTxManager(pool *pgxpool.Pool, commitTimeout time.Duration, opts ...Option) *TxManager {
	m := &TxManager{pool: pool, commitTimeout: commitTimeout}
	for _, opt := range opts {
		opt(m)
	}
	if m.readOnlySnapshot || m.lockTimeout > 0 {
		query := "BEGIN"
		if m.readOnlySnapshot {
			query += " ISOLATION LEVEL REPEATABLE READ READ ONLY"
		}
		if m.lockTimeout > 0 {
			query += fmt.Sprintf("; SET LOCAL lock_timeout = %d", (m.lockTimeout + time.Millisecond - 1).Milliseconds())
		}
		m.begin.BeginQuery = query
	}
	return m
}

// LockNotAvailable reports whether err, or an error it wraps, is
// lock_not_available (SQLSTATE 55P03): e.g. a statement's that waited for
// a lock longer than its transaction's lock timeout (LockTimeout), or that
// asked with NOWAIT for a lock another transaction held.
func LockNotAvailable(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "55P03"
}

// WithinTx runs fn in a transaction and commits it when fn returns nil. The
// context fn receives carries the transaction; a nested WithinTx on it runs
// fn in the same transaction. fn's error, or a panic, rolls it back.
func (m *TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := Tx(ctx); ok {
		return fn(ctx)
	}
	tx, err := m.pool.BeginTx(ctx, m.begin)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	returned := false
	defer func() {
		if !returned { // fn panicked: roll back, then let the panic go on
			_ = m.end(ctx, tx.Rollback)
		}
	}()
	err = fn(context.WithValue(ctx, txKey{}, tx))
	returned = true
	if err != nil {
		if rbErr := m.end(ctx, tx.Rollback); rbErr != nil {
			// fn's error keeps only its text: a domain error must not
			// hide the infrastructure fault behind its own answer.
			return fmt.Errorf("roll back transaction after %v: %w", err, rbErr)
		}
		return err
	}
	if err := m.end(ctx, tx.Commit); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// end runs COMMIT or ROLLBACK detached from ctx's cancellation, within the
// commit timeout.
func (m *TxManager) end(ctx context.Context, finish func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), m.commitTimeout)
	defer cancel()
	return finish(ctx)
}
