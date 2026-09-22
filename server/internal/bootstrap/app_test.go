package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// sampleMigrations stands in for the production set, which is empty in M0.
var sampleMigrations = fstest.MapFS{
	"00001_probe_create_widgets.sql": {Data: []byte("-- +goose Up\nCREATE TABLE widgets (id bigint);\n-- +goose Down\nDROP TABLE widgets;\n")},
	"00002_probe_create_gadgets.sql": {Data: []byte("-- +goose Up\nCREATE TABLE gadgets (id bigint);\n-- +goose Down\nDROP TABLE gadgets;\n")},
}

// testWebUI stands in for the built frontend, so tests do not depend on make build.
const testIndexHTML = "<!doctype html><title>Nerve test</title>"

var testWebUI = fstest.MapFS{"index.html": {Data: []byte(testIndexHTML)}}

const unreachableDB = "postgres://nobody@127.0.0.1:1/nowhere"

var client = &http.Client{Timeout: 5 * time.Second}

func testConfig(t *testing.T, dbURL string, autoMigrate bool) config.Config {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return config.Config{
		Env:      config.EnvTest,
		Server:   config.ServerConfig{Addr: addr, ReadHeaderTimeout: time.Second, ShutdownTimeout: 5 * time.Second},
		Database: config.DatabaseConfig{URL: dbURL, MaxConns: 4, AutoMigrate: autoMigrate},
		Log:      config.LogConfig{Level: "error", Format: "text"},
	}
}

// startApp runs the app, serving testWebUI, until the test ends and returns
// its base URL once it answers /healthz. The cleanup checks that run shut
// down cleanly.
func startApp(t *testing.T, cfg config.Config, migrations fs.FS) string {
	t.Helper()
	a, err := newApp(context.Background(), cfg, slog.New(slog.DiscardHandler), migrations, testWebUI)
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.run(ctx) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("run() = %v, want nil after cancel", err)
		}
		a.close()
	})

	base := "http://" + cfg.Server.Addr
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		select {
		case err := <-done:
			t.Fatalf("run() returned early: %v", err)
		default:
		}
		if resp, err := client.Get(base + "/healthz"); err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return base
			}
		}
	}
	t.Fatal("server did not become healthy")
	return ""
}

func getReadyz(t *testing.T, base string) (int, httpserver.Problem) {
	t.Helper()
	resp, err := client.Get(base + "/readyz")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var p httpserver.Problem
	if resp.StatusCode != http.StatusOK {
		if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
			t.Fatalf("decode problem: %v", err)
		}
	}
	return resp.StatusCode, p
}

func TestReadyWhenAutoMigrateAppliedEverything(t *testing.T) {
	base := startApp(t, testConfig(t, pgtest.NewEmptyDatabase(t), true), sampleMigrations)

	if status, p := getReadyz(t, base); status != http.StatusOK {
		t.Errorf("GET /readyz = %d %+v, want 200", status, p)
	}
}

func TestNotReadyWithPendingMigrations(t *testing.T) {
	base := startApp(t, testConfig(t, pgtest.NewEmptyDatabase(t), false), sampleMigrations)

	status, p := getReadyz(t, base)
	if status != http.StatusServiceUnavailable || p.Code != httpserver.CodeNotReady || p.Detail != "migrations is not ready" {
		t.Errorf("GET /readyz = %d %+v, want 503 not_ready for migrations", status, p)
	}
}

func TestNotReadyWhenDatabaseIsUnavailable(t *testing.T) {
	base := startApp(t, testConfig(t, unreachableDB, false), sampleMigrations)

	status, p := getReadyz(t, base)
	if status != http.StatusServiceUnavailable || p.Code != httpserver.CodeNotReady || p.Detail != "database is not ready" {
		t.Errorf("GET /readyz = %d %+v, want 503 not_ready for database", status, p)
	}
}

func TestRunFailsWhenAutoMigrateFails(t *testing.T) {
	a, err := newApp(context.Background(), testConfig(t, unreachableDB, true), slog.New(slog.DiscardHandler), sampleMigrations, testWebUI)
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	defer a.close()
	// Bounded: should run succeed instead, it would serve until ctx is done.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.run(ctx); err == nil {
		t.Error("run() = nil, want the migration error")
	}
}

// database.url is masked as a whole in the configuration log, so newApp logs
// where the pool connects, as pgx parsed it, and never the password.
func TestNewAppLogsTheDatabaseTarget(t *testing.T) {
	var logs bytes.Buffer
	cfg := testConfig(t, "postgres://nobody:secret@127.0.0.1:1/nowhere?password=secret;more", false)
	a, err := newApp(context.Background(), cfg, slog.New(slog.NewTextHandler(&logs, nil)), sampleMigrations, testWebUI)
	if err != nil {
		t.Fatalf("newApp() error = %v", err)
	}
	defer a.close()

	out := logs.String()
	if strings.Contains(out, "secret") {
		t.Errorf("log output leaks the password: %s", out)
	}
	if !strings.Contains(out, `msg="database pool created" host=127.0.0.1 port=1 database=nowhere user=nobody`) {
		t.Errorf("log output lacks the database target: %s", out)
	}
}
