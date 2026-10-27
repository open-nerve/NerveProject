package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// acquireFixtureRow owns acquisition, not the returned transaction's lifetime.
// The caller must release a successful acquisition; every failure rolls back.
func acquireFixtureRow(ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) (pgx.Tx, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	tag, err := tx.Exec(ctx, sql, args...)
	if err == nil && tag.RowsAffected() != 1 {
		err = fmt.Errorf("fixture acquisition: expected one row, got %d", tag.RowsAffected())
	}
	if err != nil {
		cleanup, stop := context.WithTimeout(context.Background(), 2*time.Second)
		defer stop()
		if rollback := tx.Rollback(cleanup); rollback != nil && !errors.Is(rollback, pgx.ErrTxClosed) {
			err = errors.Join(err, rollback)
		}
		return nil, err
	}
	return tx, nil
}

type fixtureAcquisition struct {
	tx  pgx.Tx
	err error
}

// The prior owner is explicit and controlled, not an attribution of the
// historical checkpoint's unknown peer. The acquired fixture then drives the
// original HTTP membership-removal recheck and exact unchanged-row assertions.
func TestRowFixtureAcquisitionWaitsBeforeMemberRecheck(t *testing.T) {
	w := newPrivateWorld(t)
	var write rowWrite
	for _, candidate := range rowWrites {
		if candidate.op == "removeProjectMember" {
			write = candidate
		}
	}
	if write.op == "" {
		t.Fatal("removeProjectMember missing")
	}
	table, id := write.row(t, w)
	others := rowsBut(t, w.pool, []uuid.UUID{id})
	first, err := w.pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer requestRelease(t, first)
	if _, err := first.Exec(pgtest.Soon(t), "SELECT 1 FROM projects WHERE id=$1 FOR NO KEY UPDATE", w.web); err != nil {
		t.Fatal(err)
	}
	probe := &requestProbe{}
	pool := requestPool(t, w.pool)
	acquired := make(chan fixtureAcquisition, 1)
	collected := false
	t.Cleanup(func() {
		requestRelease(t, first)
		if !collected {
			got := receiveWithin(t, acquired, 5*time.Second, "fixture acquisition cleanup")
			if got.tx != nil {
				requestRelease(t, got.tx)
			}
		}
	})
	go func() {
		tx, err := acquireFixtureRow(probe.context(t.Context()), pool, "SELECT 1 FROM projects WHERE id=$1 FOR NO KEY UPDATE", w.web)
		acquired <- fixtureAcquisition{tx, err}
	}()
	requestEventually(t, "fixture waits for controlled prior project owner", func(ctx context.Context) bool {
		select {
		case got := <-acquired:
			collected = true
			if got.tx != nil {
				requestRelease(t, got.tx)
			}
			t.Fatalf("fixture acquisition returned before prior owner release: %v", got.err)
		default:
		}
		sample, waiting := readRequestLock(t, ctx, w.pool, probe)
		return waiting && slices.Contains(sample.blockers, int32(first.Conn().PgConn().PID()))
	})
	requestBlocked(t, w.pool, probe, first.Conn().PgConn().PID(), "projects")
	requestRelease(t, first)
	got := receiveWithin(t, acquired, 5*time.Second, "fixture acquisition after prior owner")
	collected = true
	if got.err != nil {
		t.Fatal(got.err)
	}
	other := got.tx
	defer requestRelease(t, other)
	if tag, err := other.Exec(pgtest.Soon(t), "UPDATE project_members SET deleted_at=now() WHERE id=$1", id); err != nil || tag.RowsAffected() != 1 {
		t.Fatal(tag, err)
	}
	changed := rowJSON(t, other, table, id)
	req, answered := write.sent(t, w)
	// Issue background workers use LockIssueProject; the only pending M3
	// LockProject request here is this real removeProjectMember operation.
	requestEventually(t, "removeProjectMember waits for the acquired fixture", func(ctx context.Context) bool {
		var waiting bool
		if err := w.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity
   WHERE datname=current_database() AND wait_event_type='Lock'
   AND query LIKE '-- name: LockProject :one%' AND $1::int=ANY(pg_blocking_pids(pid)))`, int64(other.Conn().PgConn().PID())).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		return waiting
	})
	if err := other.Commit(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
	answer := receiveWithin(t, answered, 10*time.Second, "removeProjectMember after fixture commits")
	if answer.err != nil {
		t.Fatal(answer.err)
	}
	w.contract.CheckResponse(t, req, answer.res)
	if answer.res.StatusCode != http.StatusNotFound || problemCode(t, answer.body) != write.notFound {
		t.Fatalf("answer=%d %s want 404 %s", answer.res.StatusCode, answer.body, write.notFound)
	}
	if !reflect.DeepEqual(rowJSON(t, w.pool, table, id), changed) || !maps.Equal(rowsBut(t, w.pool, []uuid.UUID{id}), others) {
		t.Fatal("rejected member removal changed rows")
	}
}

func TestRowFixtureAcquisitionDeadlineAndCancel(t *testing.T) {
	url := pgtest.NewDatabase(t)
	observer := openPool(t, url)
	if _, err := observer.Exec(pgtest.Soon(t), "CREATE TABLE fixture_rows(id int PRIMARY KEY); INSERT INTO fixture_rows VALUES(1)"); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"cancel", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			first, err := observer.Begin(pgtest.Soon(t))
			if err != nil {
				t.Fatal(err)
			}
			defer requestRelease(t, first)
			if _, err := first.Exec(pgtest.Soon(t), "SELECT id FROM fixture_rows WHERE id=1 FOR NO KEY UPDATE"); err != nil {
				t.Fatal(err)
			}
			one := poolOfOne(t, url)
			pool := requestPool(t, one)
			probe := &requestProbe{}
			var ctx context.Context
			var cancel context.CancelFunc
			if mode == "deadline" {
				ctx, cancel = context.WithTimeout(t.Context(), 500*time.Millisecond)
			} else {
				ctx, cancel = context.WithCancel(t.Context())
			}
			defer cancel()
			done := make(chan fixtureAcquisition, 1)
			go func() {
				tx, err := acquireFixtureRow(probe.context(ctx), pool, "SELECT id FROM fixture_rows WHERE id=1 FOR NO KEY UPDATE")
				done <- fixtureAcquisition{tx, err}
			}()
			collected := false
			t.Cleanup(func() {
				cancel()
				if !collected {
					got := receiveWithin(t, done, 5*time.Second, "cancelled fixture cleanup")
					if got.tx != nil {
						requestRelease(t, got.tx)
					}
				}
			})
			requestEventually(t, "fixture acquisition is blocked before context ends", func(ctx context.Context) bool {
				select {
				case got := <-done:
					collected = true
					if got.tx != nil {
						requestRelease(t, got.tx)
					}
					t.Fatalf("fixture did not wait: %v", got.err)
				default:
				}
				sample, waiting := readRequestLock(t, ctx, observer, probe)
				return waiting && slices.Contains(sample.blockers, int32(first.Conn().PgConn().PID()))
			})
			if mode == "cancel" {
				cancel()
			}
			got := receiveWithin(t, done, 2*time.Second, "bounded fixture acquisition")
			collected = true
			want := context.Canceled
			if mode == "deadline" {
				want = context.DeadlineExceeded
			}
			if got.tx != nil || !errors.Is(got.err, want) {
				if got.tx != nil {
					requestRelease(t, got.tx)
				}
				t.Fatalf("result=%v/%v want %v", got.tx, got.err, want)
			}
			var n int
			if err := pool.QueryRow(pgtest.Soon(t), "SELECT 1").Scan(&n); err != nil || n != 1 {
				t.Fatalf("failed acquisition retained single pool connection: %d/%v", n, err)
			}
			if !heldBy(t, observer, "SELECT id FROM fixture_rows WHERE id=1 FOR UPDATE") {
				t.Fatal("cancellation released the unrelated prior owner")
			}
		})
	}
}

func TestRowFixtureAcquisitionSingleRowAndLifetime(t *testing.T) {
	url := pgtest.NewDatabase(t)
	observer := openPool(t, url)
	pool := poolOfOne(t, url)
	if _, err := observer.Exec(pgtest.Soon(t), "CREATE TABLE fixture_rows(id int PRIMARY KEY); INSERT INTO fixture_rows VALUES(1),(2)"); err != nil {
		t.Fatal(err)
	}
	for _, where := range []string{"id=99", "id IN(1,2)"} {
		t.Run(where, func(t *testing.T) {
			tx, err := acquireFixtureRow(t.Context(), pool, "SELECT id FROM fixture_rows WHERE "+where+" FOR UPDATE")
			if tx != nil || err == nil || !strings.Contains(err.Error(), "expected one row") {
				if tx != nil {
					requestRelease(t, tx)
				}
				t.Fatalf("row count accepted: %v/%v", tx, err)
			}
			for _, id := range []int{1, 2} {
				if heldBy(t, observer, "SELECT id FROM fixture_rows WHERE id=$1 FOR UPDATE", id) {
					t.Fatal("failed count left a row lock")
				}
			}
			var n int
			if err := pool.QueryRow(pgtest.Soon(t), "SELECT 1").Scan(&n); err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Run("wrapper lifetime", func(t *testing.T) {
		tx := holding(t, pool, "SELECT id FROM fixture_rows WHERE id=1 FOR NO KEY UPDATE")
		if !heldBy(t, observer, "SELECT id FROM fixture_rows WHERE id=1 FOR SHARE") {
			t.Fatal("acquisition context cancellation released transaction")
		}
		if _, err := tx.Exec(pgtest.Soon(t), "UPDATE fixture_rows SET id=id WHERE id=1"); err != nil {
			t.Fatal(err)
		}
	})
	if heldBy(t, observer, "SELECT id FROM fixture_rows WHERE id=1 FOR UPDATE") {
		t.Fatal("wrapper cleanup retained row lock")
	}
}
