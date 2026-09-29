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
	waitFor(t, pool, limit, "a lock",
		"SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'")
}

// WaitForLockWaitOn returns once a backend connected to pool's database
// waits for a row lock of table, and fails the test when none has within
// limit. A backend waiting for a row holds that row's tuple lock on the
// table while it awaits the holder's transaction, or, queued behind another
// waiter, awaits the tuple lock itself; so a wait for another table's rows,
// or for a lock of another kind, such as an advisory lock, does not count.
// One wait for a row has no tuple lock: PostgreSQL skips it when a
// transaction upgrades a row lock it already shares with another, such as
// its FOR SHARE to FOR NO KEY UPDATE while another transaction holds FOR
// SHARE too. The probe does not see that wait: it fails at its deadline and
// never returns early. Use it where more than one statement could wait, or
// where the table waited on is what the test asserts, as with the
// parent-row locks of M3 design 3.6.
func WaitForLockWaitOn(t testing.TB, pool *pgxpool.Pool, table string, limit time.Duration) {
	t.Helper()
	waitFor(t, pool, limit, "a row lock of "+table, `
		SELECT count(DISTINCT a.pid) FROM pg_stat_activity a JOIN pg_locks l ON l.pid = a.pid
		WHERE a.datname = current_database() AND a.wait_event_type = 'Lock'
			AND l.locktype = 'tuple' AND l.relation = to_regclass($1)`, table)
}

// waitFor polls query, a count of the waiting backends, until it is
// positive; what names the wait in the failure.
func waitFor(t testing.TB, pool *pgxpool.Pool, limit time.Duration, what, query string, args ...any) {
	t.Helper()
	for deadline := time.Now().Add(limit); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		var waiting int
		if err := pool.QueryRow(context.Background(), query, args...).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting > 0 {
			return
		}
	}
	t.Fatalf("no statement waited for %s within %v", what, limit)
}
