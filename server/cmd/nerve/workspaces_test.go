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
	for _, email := range []string{"nia@corp.com", "lee@corp.com"} {
		if code, _, stderr := executeWithInput(context.Background(), environ, "Tr0ub4dor&3\n", "users", "create", "--email", email); code != 0 {
			t.Fatalf("create the account of %s = %d: %s", email, code, stderr)
		}
	}
	if code, _, stderr := execute(context.Background(), environ, "users", "deactivate", "--email", "lee@corp.com"); code != 0 {
		t.Fatalf("deactivate lee = %d: %s", code, stderr)
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
		{"a deactivated account", []string{"workspaces", "create", "--slug", "beta", "--name", "Beta", "--admin-email", "lee@corp.com"},
			"nerve: The account is deactivated.\n"},
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

// `nerve workspaces reactivate-member` makes an ended membership active
// again and prints one line; a refused one exits 1 with one line on stderr
// and nothing on stdout (M3 design 3.11). That a refusal changes nothing is
// bootstrap's TestWorkspacesReactivateMemberErrors.
func TestWorkspacesReactivateMemberCommand(t *testing.T) {
	environ, pool := usersDatabase(t)
	for _, email := range []string{"nia@corp.com", "lee@corp.com"} {
		if code, _, stderr := executeWithInput(context.Background(), environ, "Tr0ub4dor&3\n", "users", "create", "--email", email); code != 0 {
			t.Fatalf("create the account of %s = %d: %s", email, code, stderr)
		}
	}
	if code, _, stderr := execute(context.Background(), environ,
		"workspaces", "create", "--slug", "acme", "--name", "Acme", "--admin-email", "nia@corp.com"); code != 0 {
		t.Fatalf("create acme = %d: %s", code, stderr)
	}
	// lee's ended membership: a fixture, which the store test and the
	// bootstrap test make as a removal does.
	if _, err := pool.Exec(context.Background(), `INSERT INTO workspace_members (id, workspace_id, member_id, role, is_active)
		SELECT gen_random_uuid(), w.id, u.id, 15, false FROM workspaces w, users u WHERE w.slug = 'acme' AND u.email = 'lee@corp.com'`); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := execute(context.Background(), environ, "workspaces", "reactivate-member", "--slug", "acme", "--email", "LEE@corp.com")

	if want := "reactivated lee@corp.com in acme as member; project memberships still ended: 0, each restored when the member joins or is " +
		"added to its project\n"; code != 0 || stdout != want {
		t.Fatalf("nerve workspaces reactivate-member = %d %q (stderr %q), want 0 and %q", code, stdout, stderr, want)
	}
	var active bool
	if err := pool.QueryRow(context.Background(), `SELECT m.is_active FROM workspace_members m JOIN users u ON u.id = m.member_id
		WHERE u.email = 'lee@corp.com'`).Scan(&active); err != nil || !active {
		t.Errorf("lee's membership active = %v (%v), want true", active, err)
	}

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"an unknown account", []string{"workspaces", "reactivate-member", "--slug", "acme", "--email", "may@corp.com"},
			"nerve: No account has this e-mail address.\n"},
		{"an unknown workspace", []string{"workspaces", "reactivate-member", "--slug", "beta", "--email", "lee@corp.com"},
			"nerve: No workspace has this slug.\n"},
		{"no email", []string{"workspaces", "reactivate-member", "--slug", "acme"}, "nerve: required flag(s) \"email\" not set\n"},
		{"no slug and no email", []string{"workspaces", "reactivate-member"}, "nerve: required flag(s) \"email\", \"slug\" not set\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := execute(context.Background(), environ, tt.args...)
			if code != 1 || stdout != "" || !strings.HasSuffix(stderr, tt.want) {
				t.Errorf("nerve %s = %d, stdout %q, stderr %q; want 1 and %q", strings.Join(tt.args, " "), code, stdout, stderr, tt.want)
			}
		})
	}
}

func TestBareWorkspacesPrintsHelp(t *testing.T) {
	code, stdout, stderr := execute(context.Background(), nil, "workspaces")

	if code != 0 || !strings.Contains(stdout, "create") || !strings.Contains(stdout, "reactivate-member") || stderr != "" {
		t.Errorf("nerve workspaces = %d, stdout %q, stderr %q; want 0 and the help", code, stdout, stderr)
	}
}
