package bootstrap

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// lockedBuffer collects the log lines that the app's goroutines write.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// cleanupRuns counts the cleanup jobs that River has completed.
func cleanupRuns(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(),
		"SELECT count(*) FROM river_job WHERE kind = 'identity.cleanup_expired_sessions' AND state = 'completed'").Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func sessionExists(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(context.Background(), "SELECT EXISTS (SELECT 1 FROM auth_sessions WHERE id = $1)", id).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	return exists
}

// The session cleanup is registered with River and runs when the app
// starts, then every auth.session_cleanup_interval; the jobs get
// jobs.shutdown_timeout (M2 design 3.15). Two settings of each, so that
// both reach the jobs whichever they are.
func TestTheSessionCleanupRunsAsConfigured(t *testing.T) {
	tests := []struct {
		interval, shutdown time.Duration
		runs               int // within 3 seconds after the first
	}{
		{time.Second, 2 * time.Second, 2},
		{time.Hour, 3 * time.Second, 1},
	}
	for _, tt := range tests {
		t.Run(tt.interval.String(), func(t *testing.T) {
			t.Parallel()
			url := pgtest.NewDatabase(t)
			pool := openPool(t, url)
			user, expired, live := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			for _, insert := range []struct {
				sql  string
				args []any
			}{
				{"INSERT INTO users (id, email, password, display_name) VALUES ($1, 'cleanup@example.com', 'x', 'cleanup')", []any{user}},
				{`INSERT INTO auth_sessions (id, user_id, token_hash, expires_at) VALUES
					($1, $3, sha256('a'), now() - interval '1 minute'), ($2, $3, sha256('b'), now() + interval '1 hour')`, []any{expired, live, user}},
			} {
				if _, err := pool.Exec(context.Background(), insert.sql, insert.args...); err != nil {
					t.Fatal(err)
				}
			}
			cfg := testConfig(t, url, false)
			cfg.Auth.SessionCleanupInterval, cfg.Jobs.ShutdownTimeout = tt.interval, tt.shutdown
			var logs lockedBuffer
			startAppLogging(t, cfg, migrations.FS(), slog.New(slog.NewTextHandler(&logs, nil)))

			for deadline := time.Now().Add(15 * time.Second); cleanupRuns(t, pool) == 0; time.Sleep(50 * time.Millisecond) {
				if time.Now().After(deadline) {
					t.Fatal("the cleanup did not run when the app started")
				}
			}
			n := 1
			for deadline := time.Now().Add(3 * time.Second); n < 2 && time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
				n = cleanupRuns(t, pool)
			}

			if n != tt.runs || sessionExists(t, pool, expired) || !sessionExists(t, pool, live) {
				t.Errorf("%d runs within 3s of the first (expired session left: %v, live one left: %v); want %d, only the live one left",
					n, sessionExists(t, pool, expired), sessionExists(t, pool, live), tt.runs)
			}
			if want := `msg="jobs started" shutdown_timeout=` + tt.shutdown.String(); !strings.Contains(logs.String(), want) {
				t.Errorf("logs lack %s:\n%s", want, logs.String())
			}
		})
	}
}

// close waits for the pool's connections for poolCloseTimeout at most: a
// handler that ignores its context may hold one (M2 design 3.15).
func TestCloseDoesNotWaitForAConnectionInUse(t *testing.T) {
	var logs bytes.Buffer
	a, err := newApp(context.Background(), testConfig(t, pgtest.NewDatabase(t), false), slog.New(slog.NewTextHandler(&logs, nil)), sampleMigrations, testWebUI)
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	a.poolCloseTimeout = 200 * time.Millisecond
	conn, err := a.pool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()

	closing := time.Now()
	closed := make(chan struct{})
	go func() {
		a.close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("close() still waits for the connection after 2s")
	}

	if took := time.Since(closing); took < 200*time.Millisecond {
		t.Errorf("close() returned after %s, want it to wait 200ms for the connection", took)
	}
	if !strings.Contains(logs.String(), `msg="database pool not closed: connections still in use" waited=200ms`) {
		t.Errorf("logs lack the warning:\n%s", logs.String())
	}
}

// run stops HTTP first, draining the requests in flight, and the jobs only
// then (M2 design 3.15): while a login waits for its account's row lock at
// shutdown, the jobs run on.
func TestRunStopsTheJobsAfterHTTP(t *testing.T) {
	var logs lockedBuffer
	url := pgtest.NewDatabase(t)
	cfg := testConfig(t, url, false)
	a := buildAppLogging(t, cfg, migrations.FS(), slog.New(slog.NewTextHandler(&logs, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- a.run(ctx) }()
	waitForLog := func(want string) {
		t.Helper()
		for deadline := time.Now().Add(10 * time.Second); !strings.Contains(logs.String(), want); time.Sleep(10 * time.Millisecond) {
			if time.Now().After(deadline) {
				t.Fatalf("logs lack %s:\n%s", want, logs.String())
			}
		}
	}
	waitForLog(`msg="jobs started"`)
	waitForLog(`msg="http server listening"`)
	addr, err := os.ReadFile(cfg.Server.AddrFile)
	if err != nil {
		t.Fatal(err)
	}
	base := "http://" + string(addr)
	registerAccount(t, apitest.Load(t), base, "drain@example.com")
	pool := openPool(t, url)
	holder, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Rollback(context.Background()) }()
	if _, err := holder.Exec(context.Background(), "SELECT 1 FROM users WHERE email = 'drain@example.com' FOR UPDATE"); err != nil {
		t.Fatal(err)
	}
	answered := make(chan int, 1)
	go func() {
		status, _ := login(t, base, "drain@example.com", "Tr0ub4dor&3")
		answered <- status
	}()
	pgtest.WaitForLockWait(t, pool, 5*time.Second) // the login is in flight

	cancel()
	waitForLog(`msg="http server shutting down"`)
	time.Sleep(time.Second) // the jobs would stop meanwhile if they did not wait for HTTP
	stoppedEarly := strings.Contains(logs.String(), `msg="jobs stopped"`)
	if err := holder.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	status := receiveWithin(t, answered, 10*time.Second, "login answer")
	if err := receiveWithin(t, done, 10*time.Second, "return from run"); err != nil {
		t.Fatalf("run() = %v", err)
	}

	out := logs.String()
	if stoppedEarly || status != http.StatusOK ||
		strings.Index(out, `msg="http server stopped"`) > strings.Index(out, `msg="jobs stopped"`) {
		t.Errorf("jobs stopped while the login was in flight: %v; the login %d; want the jobs stopped after HTTP, the login 200:\n%s",
			stoppedEarly, status, out)
	}
}

// receiveWithin fails the test unless ch yields within limit.
func receiveWithin[T any](t *testing.T, ch <-chan T, limit time.Duration, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(limit):
		t.Fatalf("no %s within %s", what, limit)
	}
	var zero T
	return zero
}
