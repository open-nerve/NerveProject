package bootstrap

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A real purge must not satisfy either ordinary write's request-lock probe.
// Hold its first table before the full barrier, so a background waiter is
// present before the HTTP request is sent, independent of startup timing.
func TestIssueLockProbeWithConcurrentPurge(t *testing.T) {
	w := newIssueWorldWith(t, func(cfg *config.Config) { cfg.Cleanup.PurgeInterval = time.Second })
	for _, iw := range issueWrites {
		if iw.op != "updateIssueComment" {
			continue
		}
		iw.checkProjectLocksWith(t, w, func(t *testing.T, pool *pgxpool.Pool, tables, mode string) pgx.Tx {
			first, query := "issues", "PurgeIssues"
			if tables == issueTables {
				first, query = "comment_reactions", "PurgeCommentReactions"
			}
			held := lockedTables(t, pool, first, mode)
			waitForPurgeBehind(t, pool, held.Conn().PgConn().PID(), query)
			if tables != first {
				if _, err := held.Exec(pgtest.Soon(t), "LOCK TABLE "+tables+" IN "+mode+" MODE"); err != nil {
					t.Fatal(err)
				}
			}
			return held
		})
		return
	}
	t.Fatal("updateIssueComment missing from directory")
}

func waitForPurgeBehind(t *testing.T, pool *pgxpool.Pool, holder uint32, queryName string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
	defer cancel()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		rows, err := pool.Query(ctx, `SELECT pid,query FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND $1::int=ANY(pg_blocking_pids(pid))`, int64(holder))
		if err != nil {
			t.Fatal(err)
		}
		var found bool
		for rows.Next() {
			var pid uint32
			var query string
			if err := rows.Scan(&pid, &query); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			if strings.Contains(query, "-- name: "+queryName+" ") {
				t.Logf("real purge PID=%d holder=%d SQL=%s", pid, holder, strings.Join(strings.Fields(query), " "))
				found = true
			}
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			t.Fatal(err)
		}
		if found {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("no real %s behind holder %d: %v", queryName, holder, ctx.Err())
		case <-tick.C:
		}
	}
}

// A finished observed query/transaction cannot lend its identity to the next
// user of the same connection, even when that user really waits for a lock.
func TestRequestLockProbeClearsIdentityBeforeConnectionReuse(t *testing.T) {
	source := poolOfOne(t, pgtest.NewDatabase(t))
	observer, err := pgxpool.New(t.Context(), source.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(observer.Close)
	for _, kind := range []string{"precheck query", "write transaction"} {
		t.Run(kind, func(t *testing.T) {
			pool := requestPool(t, source)
			probe := &requestProbe{TxManager: postgres.NewTxManager(pool, 2*time.Second)}
			held := lockedTables(t, observer, "issues", "ACCESS EXCLUSIVE")
			collect := requestBackground(t, func() error {
				if kind == "precheck query" {
					var n int
					return pool.QueryRow(probe.context(t.Context()), "SELECT count(*) FROM issues").Scan(&n)
				}
				return probe.WithinTx(t.Context(), func(ctx context.Context) error {
					tx, _ := postgres.Tx(ctx)
					var n int
					return tx.QueryRow(ctx, "SELECT count(*) FROM issues").Scan(&n)
				})
			}, func() { requestRelease(t, held) })
			pid := requestTableBlocked(t, observer, probe, held.Conn().PgConn().PID(), "issues")
			requestRelease(t, held)
			collect()
			if probe.pid.Load() != 0 {
				t.Fatal("finished request retained PID", probe.pid.Load())
			}
			reused, err := pool.Acquire(pgtest.Soon(t))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(reused.Release)
			if got := reused.Conn().PgConn().PID(); got != pid {
				t.Fatalf("connection not reused: got %d want %d", got, pid)
			}
			held = lockedTables(t, observer, "issues", "ACCESS EXCLUSIVE")
			collect = requestBackground(t, func() error {
				var n int
				return reused.QueryRow(t.Context(), "SELECT count(*) FROM issues").Scan(&n)
			}, func() { requestRelease(t, held) })
			requestEventually(t, "unmarked reused connection lock wait", func(ctx context.Context) bool {
				var waits bool
				if err := observer.QueryRow(ctx, `SELECT coalesce(wait_event_type='Lock',false) FROM pg_stat_activity WHERE pid=$1`, pid).Scan(&waits); err != nil {
					t.Fatal(err)
				}
				return waits
			})
			if probe.pid.Load() != 0 {
				t.Fatal("unmarked waiter inherited request identity", probe.pid.Load())
			}
			t.Logf("ended %s PID=%d reused by actual unmarked lock waiter; request PID remains zero", kind, pid)
			requestRelease(t, held)
			collect()
		})
	}
}
