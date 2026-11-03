package pgtest

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestFailedClusterStartupRemovesItsContainer(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test: needs Docker")
	}
	for _, stage := range []string{"readiness", "template"} {
		t.Run(stage, func(t *testing.T) {
			name := "nerve-pgtest-startup-" + uuid.NewV7().String()
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				// This exact unique name was allocated by this test, including its RED run.
				out, err := exec.CommandContext(ctx, "docker", "ps", "-aq", "--filter", "name=^/"+name+"$").CombinedOutput()
				if err != nil {
					t.Errorf("inspect fixture cleanup: %s %v", out, err)
					return
				}
				if strings.TrimSpace(string(out)) != "" {
					out, err = exec.CommandContext(ctx, "docker", "rm", "-f", "-v", name).CombinedOutput()
					if err != nil {
						t.Errorf("remove own failed fixture: %s %v", out, err)
					}
				}
			})
			opts := []testcontainers.ContainerCustomizer{testcontainers.WithName(name)}
			if stage == "readiness" {
				opts = append(opts, testcontainers.WithWaitStrategy(wait.ForLog("nerve-intentionally-absent-readiness").WithStartupTimeout(100*time.Millisecond)))
			} else {
				script := filepath.Join(t.TempDir(), "conflict.sql")
				if err := os.WriteFile(script, []byte("CREATE DATABASE nerve_template;\n"), 0600); err != nil {
					t.Fatal(err)
				}
				opts = append(opts, tcpostgres.WithInitScripts(script))
			}
			c, err := startClusterWith(opts...)
			if err == nil || c != nil {
				t.Fatalf("expected actual %s startup failure: cluster=%v err=%v", stage, c, err)
			}
			if stage == "template" && !strings.Contains(err.Error(), "create template database") {
				t.Fatal("startup cause lost", err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			out, inspectErr := exec.CommandContext(ctx, "docker", "ps", "-aq", "--filter", "name=^/"+name+"$").CombinedOutput()
			if inspectErr != nil {
				t.Fatal(fmt.Sprint(string(out), inspectErr))
			}
			if strings.TrimSpace(string(out)) != "" {
				t.Fatalf("%s failure leaked its container %s", stage, strings.TrimSpace(string(out)))
			}
		})
	}
}
