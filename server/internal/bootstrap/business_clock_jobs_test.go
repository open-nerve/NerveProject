package bootstrap

import (
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// Offset business cleanup must expire the first session, while River still
// schedules and finalizes its jobs on real time, including the next periodic
// run. A runner accidentally driven by business time cannot satisfy both.
func TestBusinessClockWorkerKeepsRiverRealTime(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := openPool(t, url)
	user, expired, live := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	if _, err := pool.Exec(pgtest.Soon(t), "INSERT INTO users (id,email,password,display_name) VALUES ($1,'clock@example.com','x','clock')", user); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(pgtest.Soon(t), `INSERT INTO auth_sessions(id,user_id,token_hash,expires_at) VALUES
 ($1,$3,sha256('a'),now()+interval '1 day'),($2,$3,sha256('b'),now()+interval '4 days')`, expired, live, user); err != nil {
		t.Fatal(err)
	}
	cfg := testConfig(t, url, false)
	cfg.Clock.Offset = 48 * time.Hour
	cfg.Auth.SessionCleanupInterval = time.Second
	realStart := time.Now()
	base := startApp(t, cfg, migrations.FS())
	for cleanupRuns(t, pool) < 2 {
		if time.Since(realStart) > 15*time.Second {
			t.Fatal("River did not complete two periodic runs in real time")
		}
		// Poll completion, bounded by real elapsed time rather than shifted time.
		select {
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		case <-time.After(25 * time.Millisecond):
		}
	}
	if sessionExists(t, pool, expired) || !sessionExists(t, pool, live) {
		t.Errorf("cleanup used wrong cutoff: expired present=%v, live present=%v", sessionExists(t, pool, expired), sessionExists(t, pool, live))
	}
	var scheduled, finished time.Time
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT scheduled_at,finalized_at FROM river_job WHERE kind='identity.cleanup_expired_sessions' AND state='completed' ORDER BY finalized_at DESC LIMIT 1`).Scan(&scheduled, &finished); err != nil {
		t.Fatal(err)
	}
	realEnd := time.Now()
	for _, at := range []time.Time{scheduled, finished} {
		if at.Before(realStart) || at.After(realEnd) {
			t.Errorf("River timestamp %s outside real interval [%s,%s]", at, realStart, realEnd)
		}
	}
	// Requests remain responsive while the shifted business worker runs.
	resp, err := client.Get(base + "/api/v0/instance")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("instance = %d", resp.StatusCode)
	}
}
