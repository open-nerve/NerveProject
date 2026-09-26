package migrations_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

func newPool(t *testing.T, url string) *pgxpool.Pool {
	t.Helper()
	pool, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: url, MaxConns: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func tables(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		"SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name <> 'goose_db_version' ORDER BY 1")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		names = append(names, n)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return names
}

// Every migration can go up, down and up again (M2 design 4.1).
func TestMigrationsGoUpDownAndUpAgain(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t, pgtest.NewEmptyDatabase(t))
	m, err := postgres.NewMigrator(pool, migrations.FS())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Close() })

	up, err := m.Up(ctx)
	if err != nil || len(up) != 4 {
		t.Fatalf("Up() = %d migrations, %v; want 4", len(up), err)
	}
	if got, want := tables(t, pool), []string{"api_tokens", "auth_sessions", "profiles", "users"}; !slices.Equal(got, want) {
		t.Errorf("tables after Up = %q, want %q", got, want)
	}
	for range up {
		if _, err := m.Down(ctx); err != nil {
			t.Fatalf("Down() error = %v", err)
		}
	}
	if got := tables(t, pool); len(got) != 0 {
		t.Errorf("tables after every Down = %q, want none", got)
	}
	if again, err := m.Up(ctx); err != nil || len(again) != 4 {
		t.Errorf("Up() again = %d migrations, %v; want 4", len(again), err)
	}
}

// Constraint and index names are the stable names of M2 design 3.13: errors
// are mapped by them, and later migrations drop them by name.
func TestConstraintAndIndexNames(t *testing.T) {
	pool := newPool(t, pgtest.NewDatabase(t))
	rows, err := pool.Query(context.Background(), `
		SELECT conname || ' ' || contype::text || CASE WHEN contype = 'f' THEN ' ' || confdeltype::text ELSE '' END FROM pg_constraint
		WHERE conrelid IN ('users'::regclass, 'profiles'::regclass, 'auth_sessions'::regclass, 'api_tokens'::regclass)
			AND contype <> 'n' -- PG 18 lists NOT NULL as constraints too
		UNION ALL
		SELECT indexname || ' i' FROM pg_indexes WHERE tablename IN ('users', 'profiles', 'auth_sessions', 'api_tokens')
		ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		got = append(got, s)
	}
	// contype: p primary key, u unique, f foreign key (confdeltype c: ON
	// DELETE CASCADE, n: ON DELETE SET NULL), c check.
	want := []string{
		"api_tokens_created_by_id_fkey f n",
		"api_tokens_label_check c",
		"api_tokens_pkey i",
		"api_tokens_pkey p",
		"api_tokens_token_hash_check c",
		"api_tokens_token_hash_key i",
		"api_tokens_token_hash_key u",
		"api_tokens_updated_by_id_fkey f n",
		"api_tokens_user_id_created_at_idx i",
		"api_tokens_user_id_fkey f c",
		"auth_sessions_expires_at_idx i",
		"auth_sessions_generation_check c",
		"auth_sessions_pkey i",
		"auth_sessions_pkey p",
		"auth_sessions_revoke_reason_check c",
		"auth_sessions_revoked_consistent_check c",
		"auth_sessions_token_hash_check c",
		"auth_sessions_user_id_fkey f c",
		"auth_sessions_user_id_idx i",
		"profiles_language_check c",
		"profiles_onboarding_step_check c",
		"profiles_pkey i",
		"profiles_pkey p",
		"profiles_start_of_the_week_check c",
		"profiles_theme_check c",
		"profiles_user_id_fkey f c",
		"profiles_user_id_key i",
		"profiles_user_id_key u",
		"users_display_name_check c",
		"users_email_check c",
		"users_email_key i",
		"users_email_key u",
		"users_pkey i",
		"users_pkey p",
	}
	if !slices.Equal(got, want) {
		t.Errorf("constraints and indexes =\n%q\nwant\n%q", got, want)
	}
}

// The CHECKs accept what the domain writes and reject what bypasses it
// (M2 design 4.2, 4.3, 4.5).
func TestChecksRejectCounterexamples(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t, pgtest.NewDatabase(t))
	const user = "'0199a2b4-0000-7000-8000-000000000001'"
	for _, stmt := range []string{
		"INSERT INTO users (id, email, password, display_name) VALUES (" + user + ", 'élodie@exämple.com', 'x', 'élodie')",
		"INSERT INTO profiles (id, user_id) VALUES ('0199a2b4-0000-7000-8000-000000000002', " + user + ")",
		"INSERT INTO auth_sessions (id, user_id, token_hash, expires_at) VALUES ('0199a2b4-0000-7000-8000-000000000003', " + user + ", sha256('x'), now())",
		`UPDATE profiles SET onboarding_step = onboarding_step || '{"profile_complete": true}'`,
		"UPDATE profiles SET onboarding_step = onboarding_step || '{}'",
		"UPDATE auth_sessions SET revoked_at = now(), revoke_reason = 'logout'",
		"INSERT INTO api_tokens (id, user_id, token_hash, label) VALUES ('0199a2b4-0000-7000-8000-000000000004', " + user + ", sha256('t'), 'x')",
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	insertUser := func(email, displayName string) string {
		return "INSERT INTO users (id, email, password, display_name) VALUES (gen_random_uuid(), " + email + ", 'x', " + displayName + ")"
	}
	tests := []struct{ name, stmt, constraint string }{
		{"upper-case ASCII e-mail", insertUser("'Bob@corp.com'", "'b'"), "users_email_check"},
		{"upper-case non-ASCII e-mail", insertUser("'Élodie@corp.com'", "'e'"), "users_email_check"},
		{"leading space", insertUser("' carol@corp.com'", "'c'"), "users_email_check"},
		{"trailing tab", insertUser(`E'carol@corp.com\t'`, "'c'"), "users_email_check"},
		{"trailing newline", insertUser(`E'carol@corp.com\n'`, "'c'"), "users_email_check"},
		{"inner space", insertUser("'car ol@corp.com'", "'c'"), "users_email_check"},
		{"leading U+3000", insertUser(`U&'\3000carol@corp.com'`, "'c'"), "users_email_check"},
		{"trailing U+2028", insertUser(`U&'carol@corp.com\2028'`, "'c'"), "users_email_check"},
		{"empty display name", insertUser("'dave@corp.com'", "''"), "users_display_name_check"},
		{"onboarding_step an array", `UPDATE profiles SET onboarding_step = '["profile_complete"]'`, "profiles_onboarding_step_check"},
		{"onboarding_step a scalar", "UPDATE profiles SET onboarding_step = '5'", "profiles_onboarding_step_check"},
		{"onboarding_step JSON null", "UPDATE profiles SET onboarding_step = 'null'", "profiles_onboarding_step_check"},
		{"onboarding_step missing a key", `UPDATE profiles SET onboarding_step = onboarding_step - 'workspace_join'`, "profiles_onboarding_step_check"},
		{"onboarding_step extra key", `UPDATE profiles SET onboarding_step = onboarding_step || '{"extra": true}'`, "profiles_onboarding_step_check"},
		{"onboarding_step string value", `UPDATE profiles SET onboarding_step = onboarding_step || '{"workspace_join": "yes"}'`, "profiles_onboarding_step_check"},
		{"onboarding_step null value", `UPDATE profiles SET onboarding_step = onboarding_step || '{"workspace_join": null}'`, "profiles_onboarding_step_check"},
		{"onboarding_step number value", `UPDATE profiles SET onboarding_step = onboarding_step || '{"workspace_join": 1}'`, "profiles_onboarding_step_check"},
		{"unknown theme", "UPDATE profiles SET theme = 'custom'", "profiles_theme_check"},
		{"unknown language", "UPDATE profiles SET language = 'fr'", "profiles_language_check"},
		{"start_of_the_week 7", "UPDATE profiles SET start_of_the_week = 7", "profiles_start_of_the_week_check"},
		{"token hash not 32 bytes", "UPDATE auth_sessions SET token_hash = '\\x00'", "auth_sessions_token_hash_check"},
		{"negative generation", "UPDATE auth_sessions SET generation = -1", "auth_sessions_generation_check"},
		{"unknown revoke reason", "UPDATE auth_sessions SET revoke_reason = 'expired'", "auth_sessions_revoke_reason_check"},
		{"revoked without a reason", "UPDATE auth_sessions SET revoke_reason = NULL", "auth_sessions_revoked_consistent_check"},
		{"a reason without revoked_at", "UPDATE auth_sessions SET revoked_at = NULL", "auth_sessions_revoked_consistent_check"},
		{"token hash of a PAT not 32 bytes", "UPDATE api_tokens SET token_hash = '\\x00'", "api_tokens_token_hash_check"},
		{"empty label", "UPDATE api_tokens SET label = ''", "api_tokens_label_check"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, tt.stmt)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23514" || pgErr.ConstraintName != tt.constraint {
				t.Errorf("%s = %v, want check_violation (23514) of %s", tt.stmt, err, tt.constraint)
			}
		})
	}
}
