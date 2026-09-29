package bootstrap

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// runWorkspaces runs cmd on the database at url with workspace creation
// switched off, and returns its line, its logs and its error.
func runWorkspaces(t *testing.T, url string, cmd WorkspaceCommand) (out, logs string, err error) {
	t.Helper()
	cfg := testConfig(t, url, false)
	cfg.Log.Level = "info"
	cfg.Workspace.CreationEnabled = false
	var stdout, stderr bytes.Buffer
	err = Workspaces(context.Background(), cfg, &stderr, &stdout, cmd)
	return stdout.String(), stderr.String(), err
}

// workspaceRows are the workspaces and their members, one line each, in a
// fixed order.
func workspaceRows(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var rows string
	err := pool.QueryRow(context.Background(), `SELECT coalesce(string_agg(w.slug || ' ' || w.name || ' ' || w.timezone || ' by ' || c.email
			|| ': ' || u.email || ' ' || m.role || ' ' || m.is_active, E'\n' ORDER BY w.slug, u.email), '')
		FROM workspaces w JOIN users c ON c.id = w.created_by_id
		JOIN workspace_members m ON m.workspace_id = w.id JOIN users u ON u.id = m.member_id`).Scan(&rows)
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// `nerve workspaces create` runs on the minimal composition while creation
// is switched off (M3 design 3.11, 6.6): the workspace, with the account of
// the address, normalized, as its only member and admin; one line; the log
// says once that the command line asked. Nothing is enqueued.
func TestWorkspacesCreate(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := openPool(t, url)
	createdAccount(t, url, pool, "alice@corp.com")
	createdAccount(t, url, pool, "bob@corp.com")

	out, logs, err := runWorkspaces(t, url, CreateWorkspace("acme", "Acme", " Alice@Corp.COM "))

	if err != nil || out != "created workspace acme with admin alice@corp.com\n" {
		t.Fatalf("create = %q, %v", out, err)
	}
	if strings.Count(logs, `msg="workspace created"`) != 1 || !strings.Contains(logs, "by=cli") {
		t.Errorf("logs = %s, want the creation logged once, by cli", logs)
	}
	if got, want := workspaceRows(t, pool), "acme Acme UTC by alice@corp.com: alice@corp.com 20 true"; got != want {
		t.Errorf("rows = %q, want %q", got, want)
	}
	var jobs int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM river_job").Scan(&jobs); err != nil || jobs != 0 {
		t.Errorf("river_job holds %d rows (%v), want none: creating a workspace enqueues nothing", jobs, err)
	}
}

// A refused command prints no line, says why in one line, and leaves the
// database as it was (M3 design 3.11).
func TestWorkspacesCreateErrors(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := openPool(t, url)
	createdAccount(t, url, pool, "alice@corp.com")
	createdAccount(t, url, pool, "dave@corp.com")
	if _, _, err := runUsers(t, url, DeactivateUser("dave@corp.com")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runWorkspaces(t, url, CreateWorkspace("acme", "Acme", "alice@corp.com")); err != nil {
		t.Fatal(err)
	}
	before := workspaceRows(t, pool)
	tests := []struct {
		name string
		cmd  WorkspaceCommand
		want string
	}{
		{"a taken slug", CreateWorkspace("acme", "Acme again", "alice@corp.com"), "A workspace with this slug exists."},
		{"an unknown account", CreateWorkspace("beta", "Beta", "nobody@corp.com"), "No account has this e-mail address."},
		{"a deactivated account", CreateWorkspace("beta", "Beta", "dave@corp.com"), "The account is deactivated."},
		{"a reserved slug", CreateWorkspace("api", "API", "alice@corp.com"), "--slug is reserved"},
		{"a bad name and slug", CreateWorkspace("Beta!", " ", "alice@corp.com"),
			"--name must contain a letter or a digit; --slug may hold only lower-case letters, digits, - and _"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := runWorkspaces(t, url, tt.cmd)
			if err == nil || err.Error() != tt.want || out != "" {
				t.Errorf("= %q, %v; want no line and %q", out, err, tt.want)
			}
		})
	}
	if got := workspaceRows(t, pool); got != before {
		t.Errorf("rows = %q, want them unchanged: %q", got, before)
	}
}
