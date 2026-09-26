package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// runUsers runs cmd on the database at url and returns its line, its logs
// and its error.
func runUsers(t *testing.T, url string, cmd UserCommand) (out, logs string, err error) {
	t.Helper()
	cfg := testConfig(t, url, false)
	cfg.Log.Level = "info"
	var stdout, stderr bytes.Buffer
	err = Users(context.Background(), cfg, &stderr, &stdout, cmd)
	return stdout.String(), stderr.String(), err
}

// accountState is what the commands change of an account: its address, its
// state, its sessions' reasons and its tokens.
type accountState struct {
	email, sessions, tokens string
	active                  bool
}

func stateOf(t *testing.T, pool *pgxpool.Pool, id string) accountState {
	t.Helper()
	var s accountState
	err := pool.QueryRow(context.Background(), `SELECT u.email, u.is_active,
		coalesce((SELECT string_agg(coalesce(revoke_reason, 'live'), ',' ORDER BY coalesce(revoke_reason, 'live')) FROM auth_sessions WHERE user_id = u.id), ''),
		coalesce((SELECT string_agg(CASE WHEN deleted_at IS NULL THEN 'kept' ELSE 'revoked' END, ',' ORDER BY deleted_at) FROM api_tokens WHERE user_id = u.id), '')
		FROM users u WHERE u.id = $1`, id).Scan(&s.email, &s.active, &s.sessions, &s.tokens)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// The five commands run on the minimal composition, one after another on
// one account, each printing its line (M2 design 3.17); a second account
// shows that nothing reaches beyond the one named. The composition has no
// jobs client: nothing is enqueued.
func TestUsersCommands(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := openPool(t, url)
	bob := createdAccount(t, url, pool, "bob@corp.com")

	out, logs, err := runUsers(t, url, CreateUser(" Carol@Corp.COM ", "Tr0ub4dor&3"))
	if err != nil || out != "created user carol@corp.com\n" || !strings.Contains(logs, `msg="account created"`) {
		t.Fatalf("create = %q, %v, logs %s", out, err, logs)
	}
	var carol string
	if err := pool.QueryRow(context.Background(), "SELECT id FROM users WHERE email = 'carol@corp.com'").Scan(&carol); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		`INSERT INTO auth_sessions (id, user_id, token_hash, expires_at) VALUES (uuidv7(), $1, sha256(uuidv7()::text::bytea), now() + interval '1 hour'),
			(uuidv7(), $1, sha256(uuidv7()::text::bytea), now() + interval '1 hour')`,
		`INSERT INTO api_tokens (id, user_id, token_hash, label) VALUES (uuidv7(), $1, sha256(uuidv7()::text::bytea), 'c'),
			(uuidv7(), $1, sha256(uuidv7()::text::bytea), 'd'), (uuidv7(), $1, sha256(uuidv7()::text::bytea), 'e')`,
	} {
		for _, id := range []string{carol, bob} {
			if _, err := pool.Exec(context.Background(), sql, id); err != nil {
				t.Fatal(err)
			}
		}
	}
	// Each step after the reset signs carol in once more first: a live
	// session for it to revoke, or not.
	steps := []struct {
		name string
		cmd  UserCommand
		out  string
		want accountState
	}{
		{"reset-password", ResetPassword("CAROL@corp.com", "N3w-Passw0rd!"),
			"password reset for carol@corp.com: revoked 2 sessions, 3 API tokens\n",
			accountState{"carol@corp.com", "password_reset,password_reset", "revoked,revoked,revoked", true}},
		{"set-email", SetEmail("carol@corp.com", "Carol@New.Example"),
			"email changed from carol@corp.com to carol@new.example: revoked 1 sessions\n",
			accountState{"carol@new.example", "email_changed,password_reset,password_reset", "revoked,revoked,revoked", true}},
		{"deactivate", DeactivateUser("carol@new.example"),
			"deactivated carol@new.example: revoked 1 sessions\n",
			accountState{"carol@new.example", "deactivated,email_changed,password_reset,password_reset", "revoked,revoked,revoked", false}},
		{"activate", ActivateUser("carol@new.example"),
			"activated carol@new.example: 0 API tokens are usable again\n",
			accountState{"carol@new.example", "deactivated,email_changed,live,password_reset,password_reset", "revoked,revoked,revoked", true}},
	}
	for i, step := range steps {
		if i > 0 {
			if _, err := pool.Exec(context.Background(), `INSERT INTO auth_sessions (id, user_id, token_hash, expires_at)
				VALUES (uuidv7(), $1, sha256(uuidv7()::text::bytea), now() + interval '1 hour')`, carol); err != nil {
				t.Fatal(err)
			}
		}
		out, _, err := runUsers(t, url, step.cmd)
		if got := stateOf(t, pool, carol); err != nil || out != step.out || got != step.want {
			t.Errorf("%s = %q, %v leaving %+v; want %q leaving %+v", step.name, out, err, got, step.out, step.want)
		}
	}
	if got, want := stateOf(t, pool, bob), (accountState{"bob@corp.com", "live,live", "kept,kept,kept", true}); got != want {
		t.Errorf("bob = %+v, want %+v: untouched", got, want)
	}
	var jobs int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM river_job").Scan(&jobs); err != nil || jobs != 0 {
		t.Errorf("river_job holds %d rows (%v), want none: the commands have no jobs client", jobs, err)
	}
}

// createdAccount creates email through the command and returns its id.
func createdAccount(t *testing.T, url string, pool *pgxpool.Pool, email string) string {
	t.Helper()
	if _, _, err := runUsers(t, url, CreateUser(email, "Tr0ub4dor&3")); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := pool.QueryRow(context.Background(), "SELECT id FROM users WHERE email = $1", email).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// A deactivated account's tokens are counted as usable again when it is
// activated: unrevoked and unexpired ones only.
func TestActivateCountsTheUsableTokens(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := openPool(t, url)
	id := createdAccount(t, url, pool, "dave@corp.com")
	if _, err := pool.Exec(context.Background(), `INSERT INTO api_tokens (id, user_id, token_hash, label, expired_at, deleted_at) VALUES
		(uuidv7(), $1, sha256('a'), 'a', NULL, NULL), (uuidv7(), $1, sha256('b'), 'b', now() + interval '1 day', NULL),
		(uuidv7(), $1, sha256('c'), 'c', now() - interval '1 day', NULL), (uuidv7(), $1, sha256('d'), 'd', NULL, now())`, id); err != nil {
		t.Fatal(err)
	}
	if _, _, err := runUsers(t, url, DeactivateUser("dave@corp.com")); err != nil {
		t.Fatal(err)
	}

	out, _, err := runUsers(t, url, ActivateUser("dave@corp.com"))

	if err != nil || out != "activated dave@corp.com: 2 API tokens are usable again\n" {
		t.Errorf("activate = %q, %v; want 2 usable tokens", out, err)
	}
}

// A refused command prints no line and says why in one line: the invalid
// fields by the names the command line knows, or the error's detail.
func TestUsersCommandErrors(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := openPool(t, url)
	createdAccount(t, url, pool, "erin@corp.com")
	createdAccount(t, url, pool, "frank@corp.com")
	tests := []struct {
		name string
		cmd  UserCommand
		want string
	}{
		{"a taken address", CreateUser("erin@corp.com", "Tr0ub4dor&3"), "An account with this e-mail address already exists."},
		{"a weak password and a bad address", CreateUser("nobody", "short"),
			"--email is not a valid e-mail address; the password must be 8-128 characters with an upper-case letter, a lower-case letter, a digit and one of !@#$%^&*()-_+=[]{}|;:'\",.<>?/"},
		{"an unknown account", ResetPassword("nobody@corp.com", "N3w-Passw0rd!"), "No account has this e-mail address."},
		{"a common password", ResetPassword("erin@corp.com", "Password1!"), "the password is too common"},
		{"another account's address", SetEmail("erin@corp.com", "frank@corp.com"), "An account with this e-mail address already exists."},
		{"the same address", SetEmail("erin@corp.com", "ERIN@corp.com"), "The new e-mail address is the account's current one."},
		{"a bad new address", SetEmail("erin@corp.com", "erin"), "--new-email is not a valid e-mail address"},
		{"deactivating nobody", DeactivateUser("nobody@corp.com"), "No account has this e-mail address."},
		{"activating nobody", ActivateUser("nobody@corp.com"), "No account has this e-mail address."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := runUsers(t, url, tt.cmd)
			if err == nil || err.Error() != tt.want || out != "" {
				t.Errorf("= %q, %v; want no line and %q", out, err, tt.want)
			}
		})
	}
	if got := stateOf(t, pool, createdAccount(t, url, pool, "gina@corp.com")); got.email != "gina@corp.com" {
		t.Fatalf("gina = %+v", got)
	}
	var emails string
	if err := pool.QueryRow(context.Background(), "SELECT string_agg(email, ',' ORDER BY email) FROM users").Scan(&emails); err != nil ||
		emails != "erin@corp.com,frank@corp.com,gina@corp.com" {
		t.Errorf("accounts = %s (%v), want the three created and nothing changed", emails, err)
	}
}

func TestCommandErrorKeepsOtherErrors(t *testing.T) {
	boom := errors.New("connection refused")
	if err := commandError(boom); !errors.Is(err, boom) {
		t.Errorf("commandError(%v) = %v, want it unchanged", boom, err)
	}
	if err := commandError(domain.ErrAccountNotFound); !errors.Is(err, domain.ErrAccountNotFound) {
		t.Errorf("commandError(%v) = %v, want it unchanged", domain.ErrAccountNotFound, err)
	}
	unknown := shared.Invalid(shared.FieldError{Field: "label", Message: "is too long"})
	if err := commandError(unknown); err == nil || err.Error() != "label is too long" {
		t.Errorf("commandError(%v) = %v, want the field's own name", unknown, err)
	}
}
