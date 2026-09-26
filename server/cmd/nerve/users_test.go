package main

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	argon2adapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/argon2"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// usersDatabase is a migrated database of its own, the environment that
// points nerve at it, and a pool on it for the assertions.
func usersDatabase(t *testing.T) ([]string, *pgxpool.Pool) {
	t.Helper()
	url := pgtest.NewDatabase(t)
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=" + url}, pool
}

// hasPassword reports whether the account with email has password, hashed
// with the test profile's argon2 parameters: the command hashes with
// auth.password.
func hasPassword(t *testing.T, pool *pgxpool.Pool, email, password string) bool {
	t.Helper()
	var hash string
	if err := pool.QueryRow(context.Background(), "SELECT password FROM users WHERE email = $1", email).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=64,t=1,p=1$") {
		t.Errorf("hash %s, want the test profile's parameters m=64,t=1,p=1", hash)
	}
	hasher := argon2adapter.New(argon2adapter.Params{MemoryKiB: 64, Iterations: 1, Parallelism: 1, MaxConcurrent: 1, MaxWait: time.Second},
		slog.New(slog.DiscardHandler))
	ok, _, err := hasher.Verify(context.Background(), password, hash)
	return err == nil && ok
}

// Without a terminal, the password is one line of standard input: only its
// line ending is removed, and a last line without one counts too
// (M2 design 3.17).
func TestUsersReadThePasswordFromStandardInput(t *testing.T) {
	environ, pool := usersDatabase(t)
	tests := []struct {
		email, input, password string
	}{
		{"ida@corp.com", "Tr0ub4dor&3\n", "Tr0ub4dor&3"},
		{"jan@corp.com", "Pass word1!\r\nignored\n", "Pass word1!"},
		{"kim@corp.com", " Tr0ub4dor&3 ", " Tr0ub4dor&3 "},
	}
	for _, tt := range tests {
		code, stdout, stderr := executeWithInput(context.Background(), environ, tt.input, "users", "create", "--email", tt.email)
		if code != 0 || stdout != "created user "+tt.email+"\n" || !hasPassword(t, pool, tt.email, tt.password) {
			t.Errorf("create %s from %q = %d %q (stderr %q); want the account with password %q", tt.email, tt.input, code, stdout, stderr, tt.password)
		}
	}

	code, stdout, _ := executeWithInput(context.Background(), environ, "N3w-Passw0rd!\n", "users", "reset-password", "--email", "ida@corp.com")
	if code != 0 || stdout != "password reset for ida@corp.com: revoked 0 sessions, 0 API tokens\n" || !hasPassword(t, pool, "ida@corp.com", "N3w-Passw0rd!") {
		t.Errorf("reset-password = %d %q, want ida's new password", code, stdout)
	}
}

// The commands that set no password do not read standard input: empty
// input is no error for them.
func TestUsersCommandsWithoutAPassword(t *testing.T) {
	environ, _ := usersDatabase(t)
	if code, _, stderr := executeWithInput(context.Background(), environ, "Tr0ub4dor&3\n", "users", "create", "--email", "lee@corp.com"); code != 0 {
		t.Fatalf("create = %d: %s", code, stderr)
	}
	for _, args := range [][]string{
		{"users", "deactivate", "--email", "lee@corp.com"},
		{"users", "activate", "--email", "lee@corp.com"},
		{"users", "set-email", "--email", "lee@corp.com", "--new-email", "lee@new.example"},
	} {
		if code, stdout, stderr := execute(context.Background(), environ, args...); code != 0 || stdout == "" {
			t.Errorf("nerve %s = %d %q (stderr %q), want 0 and its line", strings.Join(args, " "), code, stdout, stderr)
		}
	}
}

// A refused command exits 1 with one line on stderr and nothing on stdout.
func TestUsersCommandsFail(t *testing.T) {
	environ, _ := usersDatabase(t)
	tests := []struct {
		name  string
		input string
		args  []string
		want  string
	}{
		{"no password", "", []string{"users", "create", "--email", "may@corp.com"}, "nerve: read the password from standard input: EOF\n"},
		{"an empty password", "\n", []string{"users", "create", "--email", "may@corp.com"}, "nerve: the password is required\n"},
		{"no address", "Tr0ub4dor&3\n", []string{"users", "create"}, "nerve: required flag(s) \"email\" not set\n"},
		{"no new address", "", []string{"users", "set-email", "--email", "may@corp.com"}, "nerve: required flag(s) \"new-email\" not set\n"},
		{"an unknown account", "", []string{"users", "activate", "--email", "may@corp.com"}, "nerve: No account has this e-mail address.\n"},
		{"an unknown command", "", []string{"users", "delete"}, "nerve: unknown command \"delete\" for \"nerve users\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := executeWithInput(context.Background(), environ, tt.input, tt.args...)
			if code != 1 || stdout != "" || !strings.HasSuffix(stderr, tt.want) {
				t.Errorf("nerve %s = %d, stdout %q, stderr %q; want 1 and %q", strings.Join(tt.args, " "), code, stdout, stderr, tt.want)
			}
		})
	}
}

func TestBareUsersPrintsHelp(t *testing.T) {
	code, stdout, stderr := execute(context.Background(), nil, "users")

	if code != 0 || !strings.Contains(stdout, "reset-password") || stderr != "" {
		t.Errorf("nerve users = %d, stdout %q, stderr %q; want 0 and the help", code, stdout, stderr)
	}
}
