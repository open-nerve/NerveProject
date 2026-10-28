package bootstrap

import (
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// Omitting a whole module's Job must fail even though the remaining workers
// and schedules agree. The expected kinds are independent of Module.Jobs.
func TestAllPeriodicJobsRun(t *testing.T) {
	for _, tt := range []struct {
		name                          string
		cleanup, archive, purge, trim time.Duration
		fast                          string
	}{
		{"cleanup repeats", time.Second, time.Hour, time.Hour, time.Hour, "identity.cleanup_expired_sessions"},
		{"archive repeats", time.Hour, time.Second, time.Hour, time.Hour, "issue.auto_archive"},
		{"purge repeats", time.Hour, time.Hour, time.Second, time.Hour, "cleanup.purge_deleted"},
		{"trim repeats", time.Hour, time.Hour, time.Hour, time.Second, "issue.trim_description_versions"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dbURL := pgtest.NewDatabase(t)
			pool := openPool(t, dbURL)
			cfg := testConfig(t, dbURL, false)
			cfg.Auth.SessionCleanupInterval = tt.cleanup
			cfg.Issue.AutoArchiveInterval = tt.archive
			cfg.Cleanup.PurgeInterval = tt.purge
			cfg.Issue.TrimVersionsInterval = tt.trim
			var logs lockedBuffer
			base := startAppLogging(t, cfg, migrations.FS(), slog.New(slog.NewTextHandler(&logs, nil)))
			if status, p := getReadyz(t, base); status != 200 {
				t.Fatalf("/readyz = %d %+v", status, p)
			}
			waitForCompletedKinds(t, pool, map[string]int{"identity.cleanup_expired_sessions": 1, "issue.auto_archive": 1, "cleanup.purge_deleted": 1, "issue.trim_description_versions": 1}, &logs)
			waitForCompletedKinds(t, pool, map[string]int{tt.fast: 2}, &logs)
			for _, slow := range []string{"identity.cleanup_expired_sessions", "issue.auto_archive", "cleanup.purge_deleted", "issue.trim_description_versions"} {
				if slow != tt.fast {
					if n := completedKind(t, pool, slow); n != 1 {
						t.Errorf("hourly %s completed %d times, want only its startup run", slow, n)
					}
				}
			}
			var wrong int
			if err := pool.QueryRow(pgtest.Soon(t), `SELECT count(*) FROM river_job
				WHERE kind IN ('identity.cleanup_expired_sessions', 'issue.auto_archive', 'cleanup.purge_deleted','issue.trim_description_versions') AND queue <> 'periodic'`).Scan(&wrong); err != nil {
				t.Fatal(err)
			}
			if wrong != 0 {
				t.Errorf("%d periodic jobs used a queue other than periodic", wrong)
			}
		})
	}
}

func completedKind(t *testing.T, pool *pgxpool.Pool, kind string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM river_job WHERE kind=$1 AND state='completed'", kind).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func waitForCompletedKinds(t *testing.T, pool *pgxpool.Pool, want map[string]int, logs *lockedBuffer) {
	t.Helper()
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	deadline := time.NewTimer(15 * time.Second)
	defer deadline.Stop()
	for {
		got, ready := make(map[string]int), true
		for kind, n := range want {
			got[kind] = completedKind(t, pool, kind)
			ready = ready && got[kind] >= n
		}
		if ready {
			return
		}
		select {
		case <-tick.C:
		case <-deadline.C:
			t.Fatalf("completed jobs = %v, want at least %v; logs:\n%s", got, want, logs.String())
		}
	}
}
