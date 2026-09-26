package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	argon2adapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/argon2"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// usersDatabase is a migrated database of its own, the environment that
// points nerve at it, and a pool on it for the assertions. The environment
// sets auth.password's iterations apart from its parallelism, so that the
// stored hash shows each reached the hasher.
func usersDatabase(t *testing.T) ([]string, *pgxpool.Pool) {
	t.Helper()
	url := pgtest.NewDatabase(t)
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=" + url, "NERVE_AUTH__PASSWORD__ARGON2_ITERATIONS=2"}, pool
}

// storedHash is the password hash of the account with email.
func storedHash(t *testing.T, pool *pgxpool.Pool, email string) string {
	t.Helper()
	var hash string
	if err := pool.QueryRow(context.Background(), "SELECT password FROM users WHERE email = $1", email).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	return hash
}

// hasPassword reports whether the account with email has password, hashed
// with auth.password's argon2 parameters: the test profile's memory and
// parallelism, and the iterations usersDatabase sets.
func hasPassword(t *testing.T, pool *pgxpool.Pool, email, password string) bool {
	t.Helper()
	hash := storedHash(t, pool, email)
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=64,t=2,p=1$") {
		t.Errorf("hash %s, want auth.password's parameters m=64,t=2,p=1", hash)
	}
	hasher := argon2adapter.New(argon2adapter.Params{MemoryKiB: 64, Iterations: 1, Parallelism: 1, MaxConcurrent: 1, MaxWait: time.Second},
		slog.New(slog.DiscardHandler))
	ok, _, err := hasher.Verify(context.Background(), password, hash)
	return err == nil && ok
}

// Without a terminal, the password is one line of standard input: only its
// line ending, \n or \r\n, is removed, and a last line without one counts
// too (M2 design 3.17). A lone \r is not a line ending: it stays.
func TestUsersReadThePasswordFromStandardInput(t *testing.T) {
	environ, pool := usersDatabase(t)
	tests := []struct {
		email, input, password string
	}{
		{"ida@corp.com", "Tr0ub4dor&3\n", "Tr0ub4dor&3"},
		{"jan@corp.com", "Pass word1!\r\nignored\n", "Pass word1!"},
		{"kim@corp.com", " Tr0ub4dor&3 ", " Tr0ub4dor&3 "},
		{"lou@corp.com", "Tr0ub4dor&3\r", "Tr0ub4dor&3\r"},
		{"max@corp.com", "Tr0ub4dor&3\r\r\n", "Tr0ub4dor&3\r"},
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
// input is no error for them. Each runs its own use case and prints its
// line.
func TestUsersCommandsWithoutAPassword(t *testing.T) {
	environ, _ := usersDatabase(t)
	if code, _, stderr := executeWithInput(context.Background(), environ, "Tr0ub4dor&3\n", "users", "create", "--email", "lee@corp.com"); code != 0 {
		t.Fatalf("create = %d: %s", code, stderr)
	}
	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"users", "deactivate", "--email", "lee@corp.com"}, "deactivated lee@corp.com: revoked 0 sessions\n"},
		{[]string{"users", "activate", "--email", "lee@corp.com"}, "activated lee@corp.com: 0 API tokens are usable again\n"},
		{[]string{"users", "set-email", "--email", "lee@corp.com", "--new-email", "lee@new.example"},
			"email changed from lee@corp.com to lee@new.example: revoked 0 sessions\n"},
	} {
		if code, stdout, stderr := execute(context.Background(), environ, tt.args...); code != 0 || stdout != tt.want {
			t.Errorf("nerve %s = %d %q (stderr %q), want 0 and %q", strings.Join(tt.args, " "), code, stdout, stderr, tt.want)
		}
	}
}

// A refused command exits 1 with one line on stderr and nothing on stdout.
func TestUsersCommandsFail(t *testing.T) {
	environ, _ := usersDatabase(t)
	if code, _, stderr := executeWithInput(context.Background(), environ, "Tr0ub4dor&3\n", "users", "create", "--email", "nia@corp.com"); code != 0 {
		t.Fatalf("create = %d: %s", code, stderr)
	}
	tests := []struct {
		name  string
		input string
		args  []string
		want  string
	}{
		{"no password", "", []string{"users", "create", "--email", "may@corp.com"}, "nerve: read the password from standard input: EOF\n"},
		{"an empty password", "\n", []string{"users", "create", "--email", "may@corp.com"}, "nerve: the password is required\n"},
		{"no address", "Tr0ub4dor&3\n", []string{"users", "create"}, "nerve: required flag(s) \"email\" not set\n"},
		{"no address to change", "", []string{"users", "set-email", "--new-email", "may@new.example"}, "nerve: required flag(s) \"email\" not set\n"},
		{"no new address", "", []string{"users", "set-email", "--email", "may@corp.com"}, "nerve: required flag(s) \"new-email\" not set\n"},
		{"the same address", "", []string{"users", "set-email", "--email", "nia@corp.com", "--new-email", "NIA@corp.com"},
			"nerve: The new e-mail address is the account's current one.\n"},
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

// Neither the output nor the logs, at every level, hold a password, a
// stored hash or the hash's derived key, in any spelling a log would give
// them (M2 design 8.4).
func TestUsersPrintAndLogNoSecret(t *testing.T) {
	environ, pool := usersDatabase(t)
	environ = append(environ, "NERVE_LOG__LEVEL=debug")
	var printed strings.Builder
	secrets := map[string][]byte{}
	for _, step := range []struct {
		password string
		args     []string
		logged   string
	}{
		{"Tr0ub4dor&3", []string{"users", "create", "--email", "oli@corp.com"}, `msg="account created"`},
		{"N3w-Passw0rd!", []string{"users", "reset-password", "--email", "oli@corp.com"}, `msg="password reset"`},
	} {
		code, stdout, stderr := executeWithInput(context.Background(), environ, step.password+"\n", step.args...)
		if code != 0 || !strings.Contains(stderr, step.logged) {
			t.Fatalf("nerve %s = %d (stderr %q), want 0 and the log %s", strings.Join(step.args, " "), code, stderr, step.logged)
		}
		printed.WriteString(stdout + stderr)
		hash := storedHash(t, pool, "oli@corp.com")
		secrets["password "+step.password] = []byte(step.password)
		secrets["hash "+hash] = []byte(hash)
		secrets["derived key of "+hash] = derivedKey(t, hash)
	}
	for name, secret := range secrets {
		assertNoSecret(t, printed.String(), name, secret)
	}
}

// derivedKey is the key a PHC string holds, $argon2id$v=19$<params>$<salt>$<key>,
// as the raw bytes it encodes.
func derivedKey(t *testing.T, hash string) []byte {
	t.Helper()
	parts := strings.Split(hash, "$")
	key, err := base64.RawStdEncoding.DecodeString(parts[len(parts)-1])
	if len(parts) != 6 || err != nil || len(key) == 0 {
		t.Fatalf("hash %s holds no key (%v)", hash, err)
	}
	return key
}

// assertNoSecret fails when text holds secret: as is, escaped as slog's
// text handler writes bytes, in hex of either case, or in either base64
// alphabet. The unpadded spellings also find the padded ones.
func assertNoSecret(t *testing.T, text, name string, secret []byte) {
	t.Helper()
	for _, spelling := range []string{
		string(secret),
		strings.Trim(strconv.Quote(string(secret)), `"`),
		hex.EncodeToString(secret),
		strings.ToUpper(hex.EncodeToString(secret)),
		base64.RawStdEncoding.EncodeToString(secret),
		base64.RawURLEncoding.EncodeToString(secret),
	} {
		if strings.Contains(text, spelling) {
			t.Errorf("the output or the logs hold the %s as %q:\n%s", name, spelling, text)
		}
	}
}

// A file that is not a terminal, such as a pipe from a script, is read as
// one line, without a prompt.
func TestReadPasswordFromAPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	if _, err := w.WriteString("Tr0ub4dor&3\n"); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	var prompt strings.Builder

	got, err := readPassword(r, &prompt)

	if err != nil || got != "Tr0ub4dor&3" || prompt.Len() != 0 {
		t.Errorf("readPassword(pipe) = %q, %v, prompting %q; want the line and no prompt", got, err, prompt.String())
	}
}

func TestBareUsersPrintsHelp(t *testing.T) {
	code, stdout, stderr := execute(context.Background(), nil, "users")

	if code != 0 || !strings.Contains(stdout, "reset-password") || stderr != "" {
		t.Errorf("nerve users = %d, stdout %q, stderr %q; want 0 and the help", code, stdout, stderr)
	}
}
