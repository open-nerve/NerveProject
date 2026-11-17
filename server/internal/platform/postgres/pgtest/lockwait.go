package pgtest

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The probes below count the backends of pool's database that wait for a
// lock. WaitForLockWait, WaitForLockWaitOn, WaitForLockWaitBehind and
// WaitForKeyWaitOn count any such backend, and so suit a test whose own
// statements are the only ones that can wait: no app runs on its database.
// In a test that runs the app, its River jobs wait for rows of the same
// database too, such as M7 P1b's notification.fanout, which takes a work
// item's row FOR SHARE and, inserting its rows, their parents' rows FOR KEY
// SHARE, as their foreign keys' checks do. There WaitForStatementLockWaitOn
// and WaitForStatementLockWaitBehind name the waiter by its statement (M7
// P1b spec R10).

// WaitForLockWait returns once a backend connected to pool's database waits
// for a lock of any kind, and fails the test when none has within limit: a
// wait for a row, for an advisory lock, or for the transaction that inserted
// the same key all count. Only that database counts: every test has its own
// (NewDatabase), so another test's lock waits never satisfy it. It suits
// only a test in which nothing else can wait on that database; for a wait
// on a table's rows, use WaitForLockWaitOn.
func WaitForLockWait(t testing.TB, pool *pgxpool.Pool, limit time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	waitFor(ctx, t, pool, limit, anyStatement, "a lock", lockWait)
}

// WaitForLockWaitOn returns once a backend connected to pool's database
// waits for a row lock of table, and fails the test when none has within
// limit, or at once when the database has no such table. A backend waiting
// for a row holds that row's tuple lock on the table while it awaits the
// holder's transaction, or, queued behind another waiter, awaits the tuple
// lock itself; so a wait for another table's rows, or for a lock of another
// kind, such as an advisory lock, does not count. Two waits for a row have
// no tuple lock, and the probe does not see them: it fails at its deadline
// and never returns early. PostgreSQL skips the tuple lock when a
// transaction upgrades a row lock it already shares with another, such as
// its FOR SHARE to FOR NO KEY UPDATE while another transaction holds FOR
// SHARE too; and when a FOR KEY SHARE, as a foreign key's check takes,
// follows the row's update chain to a newer version that a live transaction
// has locked or deleted. Use it where more than one statement could wait,
// or where the table waited on is what the test asserts, as with the
// parent-row locks of M3 design 3.6.
func WaitForLockWaitOn(t testing.TB, pool *pgxpool.Pool, table string, limit time.Duration) {
	t.Helper()
	waitForOn(t, pool, table, limit, anyStatement, "a row lock of "+table, rowLockWaitOn)
}

// WaitForStatementLockWaitOn is WaitForLockWaitOn for the backend that runs
// one of statements, the names of sqlc queries (StatementIn): another
// backend waiting for a row of table, such as a River job of the app or
// another request, does not count.
func WaitForStatementLockWaitOn(t testing.TB, pool *pgxpool.Pool, table string, limit time.Duration, statements ...string) {
	t.Helper()
	waitForOn(t, pool, table, limit, named(t, statements), "a row lock of "+table, rowLockWaitOn+" AND "+StatementIn(2), statements)
}

// WaitForLockWaitBehind returns once a backend connected to pool's
// database waits for a lock that holder's backend holds, and fails the
// test when none has within limit: pg_blocking_pids of the waiter names
// holder. A test that holds each row in a transaction of its own names the
// row waited for this way: a wait for another transaction's row does not
// count, nor does a wait that ended when its holder let go, though the
// waiter still waits on that table.
func WaitForLockWaitBehind(t testing.TB, pool *pgxpool.Pool, holder *pgconn.PgConn, limit time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	waitFor(ctx, t, pool, limit, anyStatement, fmt.Sprintf("a lock of backend %d", holder.PID()), lockWaitBehind, int64(holder.PID()))
}

// WaitForStatementLockWaitBehind is WaitForLockWaitBehind for the backend
// that runs one of statements (StatementIn): another backend waiting
// behind holder, such as a River job of the app, does not count.
func WaitForStatementLockWaitBehind(t testing.TB, pool *pgxpool.Pool, holder *pgconn.PgConn, limit time.Duration, statements ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	waitFor(ctx, t, pool, limit, named(t, statements), fmt.Sprintf("a lock of backend %d", holder.PID()), lockWaitBehind+" AND "+StatementIn(2),
		int64(holder.PID()), statements)
}

// WaitForKeyWaitOn returns once a backend connected to pool's database that
// has written table in its transaction waits for another transaction to
// end without holding a tuple lock, and fails the test when none has within
// limit, or at once when the database has no such table. That is the wait
// of a unique index's check: an INSERT whose key a live transaction has
// inserted too waits for that transaction's transactionid until it commits
// or rolls back, with no row to lock (M3 design 9.3, interleaving 18).
// WaitForLockWaitOn does not see it, having no tuple lock to look for. A
// wait for a row of the table holds its tuple lock and does not count; nor
// does a wait of a transaction that has not written the table, nor one for
// a lock of another kind, such as an advisory lock. By a transaction that
// has written table, it cannot tell a key's wait on table from one on
// another table, or from either of the two row waits PostgreSQL makes
// without a tuple lock (WaitForLockWaitOn): use it where the waiting
// transaction writes no other table with a unique key and neither of those
// two waits can occur: it upgrades no row lock it shares with another
// transaction, and none of its foreign keys' checks follows a row's update
// chain to a version that a live transaction has locked or deleted.
func WaitForKeyWaitOn(t testing.TB, pool *pgxpool.Pool, table string, limit time.Duration) {
	t.Helper()
	waitForOn(t, pool, table, limit, anyStatement, "a key of "+table, `
		SELECT count(*) FROM pg_stat_activity a
		WHERE a.datname = current_database() AND a.wait_event_type = 'Lock' AND a.wait_event = 'transactionid'
			AND EXISTS (SELECT 1 FROM pg_locks l WHERE l.pid = a.pid AND l.locktype = 'relation' AND l.relation = $1
				AND l.mode = 'RowExclusiveLock' AND l.granted)
			AND NOT EXISTS (SELECT 1 FROM pg_locks l WHERE l.pid = a.pid AND l.locktype = 'tuple')`)
}

// The counts of the waiting backends that the probes poll, each over
// pg_stat_activity a.
const (
	lockWait = `SELECT count(*) FROM pg_stat_activity a
		WHERE a.datname = current_database() AND a.wait_event_type = 'Lock'`
	// $1 is the OID of the table.
	rowLockWaitOn = `SELECT count(DISTINCT a.pid) FROM pg_stat_activity a JOIN pg_locks l ON l.pid = a.pid
		WHERE a.datname = current_database() AND a.wait_event_type = 'Lock'
			AND l.locktype = 'tuple' AND l.relation = $1`
	// $1 is the holder's backend.
	lockWaitBehind = `SELECT count(*) FROM pg_stat_activity a
		WHERE a.datname = current_database() AND a.wait_event_type = 'Lock' AND $1::int = ANY (pg_blocking_pids(a.pid))`
)

// anyStatement names the waiter of a probe that counts any backend.
const anyStatement = "statement"

// StatementIn is a condition on pg_stat_activity, true while its backend
// runs a sqlc query whose name is in $n, a text array: the query's text
// begins with the query's header, "-- name: <name> :<command>", so the name
// is its third word. A probe of a test that runs the app names its waiter
// so, as M7 P1b spec R10 has it.
func StatementIn(n int) string {
	return fmt.Sprintf("starts_with(query, '-- name: ') AND split_part(query, ' ', 3) = ANY ($%d::text[])", n)
}

// named names the waiter of a statement probe in its failure: its
// statements, one or another. A probe given none fails at once.
func named(t testing.TB, statements []string) string {
	t.Helper()
	if len(statements) == 0 {
		t.Fatal("pgtest: a statement probe names no statement")
	}
	return strings.Join(statements, " or ")
}

// waitForOn is waitFor for a probe of table: query counts with $1 the OID
// of table in pool's database, and with args after it. A database without
// the table fails the test at once, as a name that can never be waited on;
// the lookup, like each poll, ends limit after the start.
func waitForOn(t testing.TB, pool *pgxpool.Pool, table string, limit time.Duration, who, what, query string, args ...any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	var relation *uint32
	if err := pool.QueryRow(ctx, "SELECT to_regclass($1)::oid", table).Scan(&relation); err != nil {
		failPoll(ctx, t, limit, who, what, err)
	}
	if relation == nil {
		t.Fatalf("pgtest: no table %q", table)
	}
	waitFor(ctx, t, pool, limit, who, what, query, append([]any{*relation}, args...)...)
}

// waitFor polls query, a count of the waiting backends, until it is
// positive; who and what name the waiter and the wait in the failure. ctx
// ends limit after the start: every poll, its wait for a connection of the
// pool included, ends with it, so an exhausted pool fails the test at the
// deadline too.
func waitFor(ctx context.Context, t testing.TB, pool *pgxpool.Pool, limit time.Duration, who, what, query string, args ...any) {
	t.Helper()
	for {
		var waiting int
		if err := pool.QueryRow(ctx, query, args...).Scan(&waiting); err != nil {
			failPoll(ctx, t, limit, who, what, err)
		}
		if waiting > 0 {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("no %s waited for %s within %v", who, what, limit)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// failPoll fails the test for err, the error of a poll: past ctx's
// deadline, as no wait within limit.
func failPoll(ctx context.Context, t testing.TB, limit time.Duration, who, what string, err error) {
	t.Helper()
	if ctx.Err() != nil {
		t.Fatalf("no %s waited for %s within %v", who, what, limit)
	}
	t.Fatal(err)
}
