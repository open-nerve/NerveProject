package main

import (
	"context"
	"strings"
	"testing"
)

// `nerve workspaces create` makes the account of the address the admin of
// a new workspace while creation is switched off, and prints one line; a
// refused one exits 1 with one line on stderr and nothing on stdout (M3
// design 3.11).
func TestWorkspacesCreateCommand(t *testing.T) {
	environ, pool := usersDatabase(t)
	environ = append(environ, "NERVE_WORKSPACE__CREATION_ENABLED=false")
	if code, _, stderr := executeWithInput(context.Background(), environ, "Tr0ub4dor&3\n", "users", "create", "--email", "nia@corp.com"); code != 0 {
		t.Fatalf("create the account = %d: %s", code, stderr)
	}

	code, stdout, stderr := execute(context.Background(), environ,
		"workspaces", "create", "--slug", "acme", "--name", "Acme", "--admin-email", "NIA@corp.com")

	if code != 0 || stdout != "created workspace acme with admin nia@corp.com\n" {
		t.Fatalf("nerve workspaces create = %d %q (stderr %q), want 0 and one line", code, stdout, stderr)
	}
	var role int
	if err := pool.QueryRow(context.Background(), `SELECT m.role FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
		JOIN users u ON u.id = m.member_id WHERE w.slug = 'acme' AND u.email = 'nia@corp.com'`).Scan(&role); err != nil || role != 20 {
		t.Errorf("nia's role in acme = %d (%v), want 20", role, err)
	}

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"a taken slug", []string{"workspaces", "create", "--slug", "acme", "--name", "Acme", "--admin-email", "nia@corp.com"},
			"nerve: A workspace with this slug exists.\n"},
		{"an unknown account", []string{"workspaces", "create", "--slug", "beta", "--name", "Beta", "--admin-email", "may@corp.com"},
			"nerve: No account has this e-mail address.\n"},
		{"a reserved slug", []string{"workspaces", "create", "--slug", "settings", "--name", "Beta", "--admin-email", "nia@corp.com"},
			"nerve: --slug is reserved\n"},
		// Refused by the API's rules, not lower-cased (M3 design 3.11).
		{"an upper-case slug", []string{"workspaces", "create", "--slug", "Beta", "--name", "Beta", "--admin-email", "nia@corp.com"},
			"nerve: --slug may hold only lower-case letters, digits, - and _\n"},
		{"no admin", []string{"workspaces", "create", "--slug", "beta", "--name", "Beta"}, "nerve: required flag(s) \"admin-email\" not set\n"},
		{"no slug and no name", []string{"workspaces", "create", "--admin-email", "nia@corp.com"},
			"nerve: required flag(s) \"name\", \"slug\" not set\n"},
		{"an unknown command", []string{"workspaces", "delete"}, "nerve: unknown command \"delete\" for \"nerve workspaces\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := execute(context.Background(), environ, tt.args...)
			if code != 1 || stdout != "" || !strings.HasSuffix(stderr, tt.want) {
				t.Errorf("nerve %s = %d, stdout %q, stderr %q; want 1 and %q", strings.Join(tt.args, " "), code, stdout, stderr, tt.want)
			}
		})
	}
	var workspaces int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM workspaces").Scan(&workspaces); err != nil || workspaces != 1 {
		t.Errorf("%d workspaces (%v), want acme alone", workspaces, err)
	}
}

func TestBareWorkspacesPrintsHelp(t *testing.T) {
	code, stdout, stderr := execute(context.Background(), nil, "workspaces")

	if code != 0 || !strings.Contains(stdout, "create") || stderr != "" {
		t.Errorf("nerve workspaces = %d, stdout %q, stderr %q; want 0 and the help", code, stdout, stderr)
	}
}
