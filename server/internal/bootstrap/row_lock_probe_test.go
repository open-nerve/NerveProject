package bootstrap

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

type rowProbeWait struct {
	locktype, xid string
	pid           uint32
}

func rowProbeXID(t *testing.T, tx pgx.Tx) string {
	t.Helper()
	var xid string
	if err := tx.QueryRow(pgtest.Soon(t), "SELECT pg_current_xact_id()::text").Scan(&xid); err != nil {
		t.Fatal(err)
	}
	return xid
}

// The control probe has no prior row lock. The candidate first joins the row's
// multixact with KEY SHARE, so an upgrade can wait on actual transaction IDs.
func probeRowMode(t *testing.T, pool *pgxpool.Pool, mode string, upgrade bool) rowProbeWait {
	t.Helper()
	tx, err := pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer requestRelease(t, tx)
	if upgrade {
		if _, err := tx.Exec(pgtest.Soon(t), "SELECT id FROM row_probe WHERE id=1 FOR KEY SHARE"); err != nil {
			t.Fatal(err)
		}
	}
	return rowProbeQuery(t, pool, tx, "SELECT id FROM row_probe WHERE id=1 "+mode, nil, func(rowProbeWait) bool { return true })
}

func rowProbeQuery(t *testing.T, observer *pgxpool.Pool, tx pgx.Tx, sql string, args []any, accept func(rowProbeWait) bool) rowProbeWait {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		tag, err := tx.Exec(ctx, sql, args...)
		if err == nil && tag.RowsAffected() != 1 {
			err = fmt.Errorf("probe returned %d rows, want one", tag.RowsAffected())
		}
		done <- err
	}()
	joined := false
	defer func() {
		if !joined {
			if err := tx.Conn().PgConn().CancelRequest(context.Background()); err != nil {
				t.Error(err)
			}
			<-done
		}
		cancel()
	}()
	var wait rowProbeWait
	defer func() {
		if !joined {
			t.Logf("probe SQL=%s final observed wait=%+v", sql, wait)
		}
	}()
	requestEventually(t, "row probe result or actual wait", func(ctx context.Context) bool {
		select {
		case err := <-done:
			joined = true
			if err != nil {
				t.Fatal(err)
			}
			wait = rowProbeWait{}
			return true
		default:
		}
		err := observer.QueryRow(ctx, `SELECT locktype,coalesce(transactionid::text,''),pid FROM pg_locks WHERE pid=$1 AND NOT granted`, tx.Conn().PgConn().PID()).Scan(&wait.locktype, &wait.xid, &wait.pid)
		if err == pgx.ErrNoRows {
			return false
		}
		if err != nil {
			t.Fatal(err)
		}
		return accept(wait)
	})
	if wait.locktype != "" {
		t.Logf("probe mode SQL=%s wait=%+v", sql, wait)
	}
	return wait
}

func TestRowLockProbeBypassesTupleQueue(t *testing.T) {
	pool := openPool(t, pgtest.NewDatabase(t))
	if _, err := pool.Exec(pgtest.Soon(t), "CREATE TABLE row_probe(id int PRIMARY KEY); INSERT INTO row_probe VALUES(1)"); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"FOR NO KEY UPDATE", "FOR SHARE"} {
		t.Run(mode, func(t *testing.T) {
			owner := holding(t, pool, "SELECT id FROM row_probe WHERE id=1 "+mode)
			xid := rowProbeXID(t, owner)
			worker, err := pool.Begin(pgtest.Soon(t))
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() {
				_, err := worker.Exec(ctx, "SELECT id FROM row_probe WHERE id=1 FOR NO KEY UPDATE")
				done <- err
			}()
			t.Cleanup(func() {
				if err := worker.Conn().PgConn().CancelRequest(context.Background()); err != nil {
					t.Error(err)
				}
				<-done
				cancel()
				requestRelease(t, worker)
				requestRelease(t, owner)
			})
			pgtest.WaitForLockWaitBehind(t, pool, owner.Conn().PgConn(), 2*time.Second)
			if mode == "FOR NO KEY UPDATE" {
				if got := probeRowMode(t, pool, "FOR SHARE", false); got.locktype != "tuple" {
					t.Fatalf("control=%+v want tuple queue", got)
				}
			}
			got := probeRowMode(t, pool, "FOR SHARE", true)
			if mode == "FOR SHARE" {
				if got.locktype != "" {
					t.Fatalf("compatible SHARE=%+v", got)
				}
			} else if got.locktype != "transactionid" || got.xid != xid {
				t.Fatalf("candidate=%+v want direct owner xid %s", got, xid)
			}
		})
	}
}

func TestRowLockProbeDoesNotAttributeQueuedTarget(t *testing.T) {
	pool := openPool(t, pgtest.NewDatabase(t))
	if _, err := pool.Exec(pgtest.Soon(t), "CREATE TABLE row_probe(id int PRIMARY KEY); INSERT INTO row_probe VALUES(1)"); err != nil {
		t.Fatal(err)
	}
	fixture := holding(t, pool, "SELECT id FROM row_probe WHERE id=1 FOR SHARE")
	fixtureXID := rowProbeXID(t, fixture)
	target, err := pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := target.Exec(ctx, "SELECT id FROM row_probe WHERE id=1 FOR NO KEY UPDATE")
		done <- err
	}()
	t.Cleanup(func() {
		if err := target.Conn().PgConn().CancelRequest(context.Background()); err != nil {
			t.Error(err)
		}
		<-done
		cancel()
		requestRelease(t, target)
		requestRelease(t, fixture)
	})
	pgtest.WaitForLockWaitBehind(t, pool, fixture.Conn().PgConn(), 2*time.Second)
	if got := probeRowMode(t, pool, "FOR SHARE", true); got.locktype != "" {
		t.Fatalf("SHARE=%+v want compatible with fixture", got)
	}
	if got := probeRowMode(t, pool, "FOR NO KEY UPDATE", true); got.locktype != "transactionid" || got.xid != fixtureXID {
		t.Fatalf("N=%+v want fixture xid %s", got, fixtureXID)
	}
}

func TestRowLockProbeReportsOtherOwnerAndOverstrongLock(t *testing.T) {
	for _, mode := range []string{"FOR NO KEY UPDATE", "FOR UPDATE"} {
		t.Run(mode, func(t *testing.T) {
			pool := openPool(t, pgtest.NewDatabase(t))
			if _, err := pool.Exec(pgtest.Soon(t), "CREATE TABLE row_probe(id int PRIMARY KEY); INSERT INTO row_probe VALUES(1)"); err != nil {
				t.Fatal(err)
			}
			owner := holding(t, pool, "SELECT id FROM row_probe WHERE id=1 "+mode)
			xid := rowProbeXID(t, owner)
			probeMode, upgrade := "FOR SHARE", true
			if mode == "FOR UPDATE" {
				probeMode, upgrade = "FOR KEY SHARE", false
			}
			if got := probeRowMode(t, pool, probeMode, upgrade); got.locktype != "transactionid" || got.xid != xid {
				t.Fatalf("other/strong owner=%+v want %s", got, xid)
			}
			requestRelease(t, owner)
			if got := probeRowMode(t, pool, "FOR UPDATE", true); got.locktype != "" {
				t.Fatalf("released=%+v want success", got)
			}
		})
	}
}
