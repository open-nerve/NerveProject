package bootstrap

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

type deactivationIdentity struct {
	pid              uint32
	backend, started time.Time
	xid              string
}

func deactivationIdentityOf(t *testing.T, pool *pgxpool.Pool, pid uint32) deactivationIdentity {
	t.Helper()
	id := deactivationIdentity{pid: pid}
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT backend_start,xact_start,coalesce(backend_xid::text,'') FROM pg_stat_activity WHERE pid=$1`, pid).Scan(&id.backend, &id.started, &id.xid); err != nil {
		t.Fatal(err)
	}
	return id
}

func deactivationWaiters(t *testing.T, pool *pgxpool.Pool, holder uint32) []uint32 {
	t.Helper()
	rows, err := pool.Query(pgtest.Soon(t), `SELECT pid FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND $1::int=ANY(pg_blocking_pids(pid))`, holder)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uint32])
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

func identifyDeactivation(t *testing.T, pool *pgxpool.Pool, holder pgx.Tx) deactivationIdentity {
	t.Helper()
	pgtest.WaitForLockWaitBehind(t, pool, holder.Conn().PgConn(), 5*time.Second)
	ids := deactivationWaiters(t, pool, holder.Conn().PgConn().PID())
	if len(ids) != 1 {
		t.Fatalf("exclusive users holder has waiters %v, want exactly the sent deactivation", ids)
	}
	return deactivationIdentityOf(t, pool, ids[0])
}

func (id *deactivationIdentity) waiting(t *testing.T, pool *pgxpool.Pool, holder pgx.Tx) {
	t.Helper()
	requestEventuallyWithin(t, "same deactivation waiting behind current fixture", 5*time.Second, func(ctx context.Context) bool {
		var waits bool
		var blockers []uint32
		if err := pool.QueryRow(ctx, `SELECT coalesce(wait_event_type='Lock',false),pg_blocking_pids(pid) FROM pg_stat_activity WHERE pid=$1`, id.pid).Scan(&waits, &blockers); err != nil {
			t.Fatal(err)
		}
		return waits && (slices.Contains(blockers, holder.Conn().PgConn().PID()) || slices.Contains(requestBlockerChain(t, pool, id.pid), int32(holder.Conn().PgConn().PID())))
	})
	got := deactivationIdentityOf(t, pool, id.pid)
	if got.pid != id.pid || got.backend != id.backend || got.started != id.started || (id.xid != "" && got.xid != id.xid) {
		t.Fatalf("deactivation identity changed: %+v want %+v", got, id)
	}
	id.xid = got.xid
}

// A single-row probe joins with K then upgrades. Only an actual transactionid
// conflict with this target or the known fixture counts. A tuple queue or an
// unrelated worker is diagnostic evidence, never an inferred target lock.
// Fixture S can hide weaker target locks; stage-by-stage waits still prove
// order. This reports their combined strongest state, not all row owners.
func deactivationRowLock(t *testing.T, observer, probes *pgxpool.Pool, owners map[string]string, from string, args ...any) string {
	t.Helper()
	tx, err := probes.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer requestRelease(t, tx)
	modes := []string{"FOR KEY SHARE", "FOR SHARE", "FOR NO KEY UPDATE", "FOR UPDATE"}
	for i, mode := range modes {
		var last rowProbeWait
		got := rowProbeQuery(t, observer, tx, "SELECT 1 FROM "+from+" "+mode, args, func(wait rowProbeWait) bool {
			last = wait
			return wait.locktype == "transactionid" && owners[wait.xid] != ""
		})
		if got.locktype != "" {
			t.Logf("row=%s conflict owner=%s xid=%s mode=%s", from, owners[got.xid], got.xid, mode)
			return modes[len(modes)-1-i]
		}
		if last.locktype != "" {
			t.Logf("row=%s transient other wait=%+v ended before mode=%s succeeded", from, last, mode)
		}
	}
	return "no lock"
}
