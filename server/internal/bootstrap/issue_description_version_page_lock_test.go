package bootstrap

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// lockVersionPage owns only the pagination fixture's protection acquisition.
// Acquisition waits, with a deadline, for any writer that got project N first.
// Its transaction stays open for the caller's reads and must be released there.
func lockVersionPage(ctx context.Context, pool *pgxpool.Pool, id uuid.UUID) (pgx.Tx, error) {
	return acquireFixtureRow(ctx, pool, "SELECT 1 FROM projects WHERE id=$1 FOR SHARE", id)
}

// NOWAIT would fail with 55P03 when a legitimate project writer gets N first.
// The first owner here is an explicit test transaction, not a scheduled job.
// After protection succeeds, a real manually invoked trim Work must wait on it.
func TestDescriptionVersionPageProtectionWaitsForProjectWriter(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItemIn(t, w.web, `{"name":"Writer before page protection"}`)
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
	type result struct {
		tx  pgx.Tx
		err error
	}
	acquired := make(chan result, 1)
	go func() {
		tx, err := lockVersionPage(probe.context(t.Context()), pool, w.web)
		acquired <- result{tx, err}
	}()
	collected := false
	t.Cleanup(func() {
		requestRelease(t, first)
		if !collected {
			got := receiveWithin(t, acquired, 5*time.Second, "page protection cleanup")
			if got.tx != nil {
				requestRelease(t, got.tx)
			}
		}
	})
	requestEventually(t, "page protection waiting behind project writer", func(ctx context.Context) bool {
		select {
		case got := <-acquired:
			collected = true
			if got.tx != nil {
				requestRelease(t, got.tx)
			}
			t.Fatalf("page protection returned before writer release: %v", got.err)
		default:
		}
		pid, _ := probe.snapshot()
		if pid == 0 {
			return false
		}
		var waiting bool
		if err := w.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE pid=$1 AND wait_event_type='Lock')", pid).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		return waiting
	})
	requestBlocked(t, w.pool, probe, first.Conn().PgConn().PID(), "projects")
	requestRelease(t, first)
	got := receiveWithin(t, acquired, 5*time.Second, "page protection acquisition")
	collected = true
	if got.err != nil {
		t.Fatal(got.err)
	}
	holder := got.tx
	defer requestRelease(t, holder)
	ids := seedVersionPage(t, w, item, savedVersion(t, w, item))
	trim, collect := observedTrim(t, w.pool, func() { requestRelease(t, holder) })
	requestBlocked(t, w.pool, trim, holder.Conn().PgConn().PID(), "projects")
	if rows := trimVersionIDs(t, w.pool, item); len(rows) != len(ids) {
		t.Fatalf("protected versions=%d want %d", len(rows), len(ids))
	}
	requestRelease(t, holder)
	collect()
	awaitTrimVersions(t, w.pool, map[uuid.UUID][]uuid.UUID{item: ids[len(ids)-20:]})
}
