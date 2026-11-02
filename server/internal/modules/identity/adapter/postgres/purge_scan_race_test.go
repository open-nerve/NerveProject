package postgresadapter_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A candidate deleted by a concurrent transaction is still scanned. Returning
// only DELETE's rows would incorrectly exhaust this table before the next ID.
func TestPurgeCursorScannedCandidateDeletedConcurrently(t *testing.T) {
	for _, mode := range []string{"removed", "retimed"} {
		t.Run(mode, func(t *testing.T) { testPurgeCandidateChange(t, mode == "removed") })
	}
}
func testPurgeCandidateChange(t *testing.T, removed bool) {
	s, pool := newStore(t)
	u := newUser("purge-scan@corp.com")
	mustCreate(t, s, u)
	cutoff := now.Add(-60 * 24 * time.Hour)
	old := now.Add(-61 * 24 * time.Hour)
	low := purgeToken(t, s, pool, u.ID, "low", uuid.MustParse("0199a2b4-0000-7000-8000-000000000010"), &old)
	high := purgeToken(t, s, pool, u.ID, "high", uuid.MustParse("0199a2b4-0000-7000-8000-000000000020"), &old)
	cfg := pool.Config()
	cfg.MaxConns = 1
	request, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer request.Close()
	var pid uint32
	if err := request.QueryRow(pgtest.Soon(t), "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		t.Fatal(err)
	}
	holder, err := pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	if removed {
		_, err = holder.Exec(pgtest.Soon(t), "DELETE FROM api_tokens WHERE id=$1", low)
	} else {
		_, err = holder.Exec(pgtest.Soon(t), "UPDATE api_tokens SET deleted_at=$2 WHERE id=$1", low, cutoff)
	}
	if err != nil {
		_ = holder.Rollback(context.Background())
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	type result struct {
		page app.PurgePage
		err  error
	}
	done := make(chan result, 1)
	finished := false
	defer func() {
		cancel()
		_ = holder.Rollback(context.Background())
		if !finished {
			<-done
		}
	}()
	go func() {
		page, err := postgresadapter.New(request).PurgeAPITokens(ctx, app.PurgeBatch{Cutoff: cutoff, Limit: 1})
		done <- result{page, err}
	}()
	pgtest.WaitForLockWaitBehind(t, pool, holder.Conn().PgConn(), 3*time.Second)
	var same bool
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT wait_event_type='Lock' AND $2::int=ANY(pg_blocking_pids(pid)) AND position('DELETE FROM api_tokens' in query)>0 FROM pg_stat_activity WHERE pid=$1", pid, holder.Conn().PgConn().PID()).Scan(&same); err != nil || !same {
		t.Fatalf("unidentified purge waiter: %t %v", same, err)
	}
	if err := holder.Commit(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
	got := <-done
	finished = true
	if got.err != nil || got.page.Scanned != 1 || got.page.Deleted != 0 || got.page.After != low {
		t.Fatalf("lost scanned candidate: %+v %v", got.page, got.err)
	}
	next, err := s.PurgeAPITokens(t.Context(), app.PurgeBatch{Cutoff: cutoff, After: got.page.After, Limit: 1})
	if err != nil || next.Scanned != 1 || next.Deleted != 1 || next.After != high {
		t.Fatalf("later candidate skipped: %+v %v", next, err)
	}
	var retained bool
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT EXISTS(SELECT 1 FROM api_tokens WHERE id=$1)", low).Scan(&retained); err != nil || retained == removed {
		t.Fatalf("changed candidate retained=%t removed=%t err=%v", retained, removed, err)
	}

}
