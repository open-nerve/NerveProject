package bootstrap

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// This explicit owner demonstrates the attribution mechanism, not the identity
// of the historical failure's peer. The actual unassigned PATCH waits at D's
// table fixture while another transaction alone holds project N.
func TestDraftRowLockProbeIgnoresUnrelatedProjectOwner(t *testing.T) {
	w := newIssueWorld(t)
	d := createDraftHTTP(t, w, "alice", `{"payload":{"name":"Unassigned"}}`)
	fixture := lockedTables(t, w.pool, "draft_issues", "SHARE")
	defer requestRelease(t, fixture)
	tx := requestTx(w.pool)
	var e exchange
	collect := requestBackground(t, draftWrite(t, w, tx, &e, "PATCH", draftPath(d.ID), `{"payload":{"name":"Changed"}}`), func() { requestRelease(t, fixture) })
	requestTableBlocked(t, w.pool, tx, fixture.Conn().PgConn().PID(), "draft_issues")
	owner := holding(t, w.pool, "SELECT 1 FROM projects WHERE id=$1 FOR NO KEY UPDATE", w.web)
	defer requestRelease(t, owner)
	ownerXID := rowProbeXID(t, owner)
	probes := poolOfOne(t, w.pool.Config().ConnString())
	conn, err := probes.Acquire(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	probePID := conn.Conn().PgConn().PID()
	conn.Release()
	done := make(chan struct{})
	go func() {
		defer close(done)
		checkDraftHeldLocks(t, w, probes, tx, draftCRUDWrites[1], d.ID, false, "draft")
	}()
	joined := false
	t.Cleanup(func() {
		requestRelease(t, owner)
		if !joined {
			receiveWithin(t, done, 5*time.Second, "row observer cleanup")
		}
	})
	requestEventually(t, "row observer actually waits on the unrelated owner", func(ctx context.Context) bool {
		select {
		case <-done:
			joined = true
			t.Fatal("wide row observer returned before unrelated owner was released")
		default:
		}
		var xid string
		var blockers []uint32
		err := w.pool.QueryRow(ctx, `SELECT coalesce(l.transactionid::text,''),pg_blocking_pids(a.pid)
   FROM pg_stat_activity a JOIN pg_locks l ON l.pid=a.pid AND NOT l.granted
   WHERE a.pid=$1 AND a.wait_event_type='Lock' AND l.locktype='transactionid'`, probePID).Scan(&xid, &blockers)
		if errors.Is(err, pgx.ErrNoRows) {
			return false
		}
		if err != nil {
			t.Fatal(err)
		}
		return xid == ownerXID && slices.Contains(blockers, owner.Conn().PgConn().PID())
	})
	requestRelease(t, owner)
	receiveWithin(t, done, 5*time.Second, "row observer after unrelated owner release")
	joined = true
	requestTableBlocked(t, w.pool, tx, fixture.Conn().PgConn().PID(), "draft_issues")
	requestRelease(t, fixture)
	collect()
	w.answered(t, e, 200)
}
