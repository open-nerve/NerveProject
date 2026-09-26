package main

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/buildinfo"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func execute(ctx context.Context, environ []string, args ...string) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = run(ctx, args, environ, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestVersion(t *testing.T) {
	code, stdout, _ := execute(context.Background(), nil, "version")

	want := "nerve " + buildinfo.Get().Version + " commit="
	if code != 0 || !strings.HasPrefix(stdout, want) {
		t.Errorf("nerve version = %d %q, want 0 and a line starting with %q", code, stdout, want)
	}
}

func TestUnknownCommandFails(t *testing.T) {
	code, _, stderr := execute(context.Background(), nil, "bogus")

	if code != 1 || !strings.Contains(stderr, `nerve: unknown command "bogus"`) {
		t.Errorf("nerve bogus = %d %q, want 1 and an unknown command error", code, stderr)
	}
}

func TestUnknownMigrateSubcommandFails(t *testing.T) {
	code, _, stderr := execute(context.Background(), nil, "migrate", "upp")

	if code != 1 || !strings.Contains(stderr, `nerve: unknown command "upp" for "nerve migrate"`) {
		t.Errorf("nerve migrate upp = %d %q, want 1 and an unknown command error", code, stderr)
	}
}

func TestBareMigratePrintsHelp(t *testing.T) {
	code, stdout, stderr := execute(context.Background(), nil, "migrate")

	if code != 0 || !strings.Contains(stdout, "Available Commands:") || stderr != "" {
		t.Errorf("nerve migrate = %d, stdout %q, stderr %q; want 0 and the help", code, stdout, stderr)
	}
}

func TestInvalidConfigurationIsReported(t *testing.T) {
	code, _, stderr := execute(context.Background(), []string{"NERVE_ENV=test"}, "migrate", "status")

	if code != 1 || !strings.Contains(stderr, "database.url: is required") {
		t.Errorf("nerve migrate status = %d %q, want 1 and the invalid key", code, stderr)
	}
}

func TestMigrateStatus(t *testing.T) {
	environ := []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=" + pgtest.NewDatabase(t)}

	code, stdout, stderr := execute(context.Background(), environ, "migrate", "status")

	// The production set is empty in M0 and grows from M2 on: accept both outputs.
	if code != 0 || (stdout != "no migrations\n" && !strings.HasPrefix(stdout, "VERSION ")) {
		t.Errorf("nerve migrate status = %d %q (stderr %q), want 0 and \"no migrations\" or the status table", code, stdout, stderr)
	}
}

// nerve serve runs the API and the jobs until it is cancelled, then stops in
// the order of M2 design 3.15: HTTP, the jobs, the pool.
func TestServeUntilCancelled(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	url := pgtest.NewDatabase(t)
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	environ := []string{
		"NERVE_ENV=test",
		"NERVE_DATABASE__URL=" + url,
		"NERVE_SERVER__ADDR=" + addr,
		"NERVE_LOG__LEVEL=info",
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type result struct {
		code   int
		stderr string
	}
	done := make(chan result, 1)
	go func() {
		code, _, stderr := execute(ctx, environ, "serve")
		done <- result{code, stderr}
	}()

	client := &http.Client{Timeout: time.Second}
	ready, worked := false, false
	for deadline := time.Now().Add(15 * time.Second); (!ready || !worked) && time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if resp, err := client.Get("http://" + addr + "/readyz"); err == nil {
			_ = resp.Body.Close()
			ready = resp.StatusCode == http.StatusOK
		}
		// The jobs run: the session cleanup ran when nerve started.
		_ = pool.QueryRow(context.Background(), "SELECT EXISTS (SELECT 1 FROM river_job WHERE state = 'completed')").Scan(&worked)
	}
	cancel()
	r := <-done

	if !ready || !worked {
		t.Fatalf("nerve serve never became ready (%v) or never worked a job (%v); stderr:\n%s", ready, worked, r.stderr)
	}
	if r.code != 0 {
		t.Errorf("nerve serve exit code = %d, want 0; stderr:\n%s", r.code, r.stderr)
	}
	for _, want := range []string{"configuration loaded", "config.database.url=xxxxx", "database pool created"} {
		if !strings.Contains(r.stderr, want) {
			t.Errorf("stderr lacks %q:\n%s", want, r.stderr)
		}
	}
	last := -1
	for _, step := range []string{`msg="http server stopped"`, `msg="jobs stopped"`, `msg="database pool closed"`} {
		at := strings.Index(r.stderr, step)
		if at <= last {
			t.Errorf("stderr has %s at %d, want it after the step before (at %d):\n%s", step, at, last, r.stderr)
		}
		last = at
	}
}
