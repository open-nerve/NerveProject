package pgtest

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
	waitFor(ctx, t, pool, limit, "a lock",
		"SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'")
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
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()
	what := "a row lock of " + table
	var relation *uint32
	if err := pool.QueryRow(ctx, "SELECT to_regclass($1)::oid", table).Scan(&relation); err != nil {
		failPoll(ctx, t, limit, what, err)
	}
	if relation == nil {
		t.Fatalf("pgtest: no table %q", table)
	}
	waitFor(ctx, t, pool, limit, what, `
		SELECT count(DISTINCT a.pid) FROM pg_stat_activity a JOIN pg_locks l ON l.pid = a.pid
		WHERE a.datname = current_database() AND a.wait_event_type = 'Lock'
			AND l.locktype = 'tuple' AND l.relation = $1`, *relation)
}

// waitFor polls query, a count of the waiting backends, until it is
// positive; what names the wait in the failure. ctx ends limit after the
// start: every poll, its wait for a connection of the pool included, ends
// with it, so an exhausted pool fails the test at the deadline too.
func waitFor(ctx context.Context, t testing.TB, pool *pgxpool.Pool, limit time.Duration, what, query string, args ...any) {
	t.Helper()
	for {
		var waiting int
		if err := pool.QueryRow(ctx, query, args...).Scan(&waiting); err != nil {
			failPoll(ctx, t, limit, what, err)
		}
		if waiting > 0 {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("no statement waited for %s within %v", what, limit)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// failPoll fails the test for err, the error of a poll: past ctx's
// deadline, as no wait within limit.
func failPoll(ctx context.Context, t testing.TB, limit time.Duration, what string, err error) {
	t.Helper()
	if ctx.Err() != nil {
		t.Fatalf("no statement waited for %s within %v", what, limit)
	}
	t.Fatal(err)
}
