package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Reading the run result twice used to strand cleanup after a startup failure.
// A child test lets the real Fatal/cleanup path finish without failing its parent.
func TestAppFixtureStartupFailureClosesResources(t *testing.T) {
	const child = "NERVE_TEST_STARTUP_FAILURE_CHILD"
	if os.Getenv(child) == "1" {
		cfg := testConfig(t, unreachableDB, false)
		cfg.Server.Addr = "127.0.0.1:-1"
		startAppLogging(t, cfg, sampleMigrations, slog.New(slog.NewTextHandler(os.Stdout, nil)))
		t.Fatal("invalid listen address unexpectedly started")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestAppFixtureStartupFailureClosesResources$", "-test.timeout=5s", "-test.v")
	cmd.Env = append(os.Environ(), child+"=1")
	output, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if ctx.Err() != nil || !errors.As(err, &exit) || exit.ExitCode() != 1 {
		t.Fatalf("startup failure did not finish as a normal failed test: %v\n%s", err, output)
	}
	for _, required := range []string{"run() returned early:", "database pool closed"} {
		if !strings.Contains(string(output), required) {
			t.Fatalf("startup failure lost its diagnostic or resource cleanup (%s):\n%s", required, output)
		}
	}
}
