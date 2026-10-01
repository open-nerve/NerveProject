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

// The schema's tables (but goose's own), enum types and functions.
const (
	tablesQuery    = "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name <> 'goose_db_version' ORDER BY 1"
	enumsQuery     = "SELECT typname FROM pg_type WHERE typnamespace = 'public'::regnamespace AND typtype = 'e' ORDER BY 1"
	functionsQuery = "SELECT proname FROM pg_proc WHERE pronamespace = 'public'::regnamespace ORDER BY 1"
)

func names(t *testing.T, pool *pgxpool.Pool, query string) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		found = append(found, n)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return found
}

// Every migration can go up, down and up again (M2 design 4.1), River's
// too: goose runs each of its halves as one statement (M2 design 3.15).
func TestMigrationsGoUpDownAndUpAgain(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t, pgtest.NewEmptyDatabase(t))
	m, err := postgres.NewMigrator(pool, migrations.FS())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Close() })

	up, err := m.Up(ctx)
	if err != nil || len(up) != 13 {
		t.Fatalf("Up() = %d migrations, %v; want 13", len(up), err)
	}
	for _, want := range []struct {
		query string
		names []string
	}{
		{tablesQuery, []string{"api_tokens", "auth_sessions", "profiles", "project_members", "project_user_properties", "projects", "river_job",
			"river_leader", "river_notification", "river_queue", "states", "users", "workspace_member_invites", "workspace_members",
			"workspace_user_properties", "workspaces"}},
		{enumsQuery, []string{"river_job_state"}},
		{functionsQuery, []string{"river_job_state_in_bitmask"}},
	} {
		if got := names(t, pool, want.query); !slices.Equal(got, want.names) {
			t.Errorf("after Up, %s = %q, want %q", want.query, got, want.names)
		}
	}
	for range up {
		if _, err := m.Down(ctx); err != nil {
			t.Fatalf("Down() error = %v", err)
		}
	}
	for _, query := range []string{tablesQuery, enumsQuery, functionsQuery} {
		if got := names(t, pool, query); len(got) != 0 {
			t.Errorf("after every Down, %s = %q, want none", query, got)
		}
	}
	if again, err := m.Up(ctx); err != nil || len(again) != 13 {
		t.Errorf("Up() again = %d migrations, %v; want 13", len(again), err)
	}
}

// Constraint and index names are the stable names of M2 design 3.13 and M3
// design 4 (an unconditional index <table>_<column>_idx under each CASCADE
// foreign key to workspaces): errors are mapped by them, and later
// migrations drop them by name. Each index is pinned with what it is too,
// unique and partial or not, so a partial unique key cannot turn into a
// plain or a total one unseen. The tables are every table of the schema but
// River's and goose's own, read from the catalog: a new table's constraints
// and indexes fail here until they are in want.
func TestConstraintAndIndexNames(t *testing.T) {
	pool := newPool(t, pgtest.NewDatabase(t))
	rows, err := pool.Query(context.Background(), `
		WITH tables AS (
			SELECT oid FROM pg_class
			WHERE relnamespace = 'public'::regnamespace AND relkind IN ('r', 'p')
				AND relname NOT LIKE 'river\_%' AND relname <> 'goose_db_version'
		)
		SELECT conname || ' ' || contype::text || CASE WHEN contype = 'f' THEN ' ' || confdeltype::text ELSE '' END FROM pg_constraint
		WHERE conrelid IN (SELECT oid FROM tables)
			AND contype <> 'n' -- PG 18 lists NOT NULL as constraints too
		UNION ALL
		SELECT c.relname || ' i' || CASE WHEN i.indisunique THEN 'u' ELSE '' END || CASE WHEN i.indpred IS NOT NULL THEN 'w' ELSE '' END
		FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid
		WHERE i.indrelid IN (SELECT oid FROM tables)
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
	// DELETE CASCADE, n: ON DELETE SET NULL), c check. An index is i, then
	// u when it is unique and w when it is partial (has a WHERE). The project
	// module's tables are in projectNames (project_schema_test.go).
	want := slices.Concat([]string{
		"api_tokens_created_by_id_fkey f n",
		"api_tokens_label_check c",
		"api_tokens_pkey iu",
		"api_tokens_pkey p",
		"api_tokens_token_hash_check c",
		"api_tokens_token_hash_key iu",
		"api_tokens_token_hash_key u",
		"api_tokens_updated_by_id_fkey f n",
		"api_tokens_user_id_created_at_idx iw",
		"api_tokens_user_id_fkey f c",
		"auth_sessions_expires_at_idx i",
		"auth_sessions_generation_check c",
		"auth_sessions_pkey iu",
		"auth_sessions_pkey p",
		"auth_sessions_revoke_reason_check c",
		"auth_sessions_revoked_consistent_check c",
		"auth_sessions_token_hash_check c",
		"auth_sessions_user_id_fkey f c",
		"auth_sessions_user_id_idx i",
		"profiles_language_check c",
		"profiles_onboarding_step_check c",
		"profiles_pkey iu",
		"profiles_pkey p",
		"profiles_start_of_the_week_check c",
		"profiles_theme_check c",
		"profiles_user_id_fkey f c",
		"profiles_user_id_key iu",
		"profiles_user_id_key u",
		"users_display_name_check c",
		"users_email_check c",
		"users_email_key iu",
		"users_email_key u",
		"users_pkey iu",
		"users_pkey p",
		"workspace_member_invites_created_by_id_fkey f n",
		"workspace_member_invites_email_check c",
		"workspace_member_invites_email_idx iw",
		"workspace_member_invites_pkey iu",
		"workspace_member_invites_pkey p",
		"workspace_member_invites_responded_check c",
		"workspace_member_invites_role_check c",
		"workspace_member_invites_updated_by_id_fkey f n",
		"workspace_member_invites_workspace_id_email_key iuw",
		"workspace_member_invites_workspace_id_fkey f c",
		"workspace_member_invites_workspace_id_idx i",
		"workspace_members_created_by_id_fkey f n",
		"workspace_members_member_id_fkey f c",
		"workspace_members_member_id_idx iw",
		"workspace_members_pkey iu",
		"workspace_members_pkey p",
		"workspace_members_role_check c",
		"workspace_members_updated_by_id_fkey f n",
		"workspace_members_workspace_id_fkey f c",
		"workspace_members_workspace_id_idx i",
		"workspace_members_workspace_id_member_id_key iuw",
		"workspace_user_properties_created_by_id_fkey f n",
		"workspace_user_properties_navigation_control_preference_check c",
		"workspace_user_properties_navigation_project_limit_check c",
		"workspace_user_properties_pkey iu",
		"workspace_user_properties_pkey p",
		"workspace_user_properties_updated_by_id_fkey f n",
		"workspace_user_properties_user_id_fkey f c",
		"workspace_user_properties_workspace_id_fkey f c",
		"workspace_user_properties_workspace_id_idx i",
		"workspace_user_properties_workspace_id_user_id_key iuw",
		"workspaces_created_by_id_fkey f n",
		"workspaces_name_check c",
		"workspaces_organization_size_check c",
		"workspaces_pkey iu",
		"workspaces_pkey p",
		"workspaces_slug_check c",
		"workspaces_slug_key iuw",
		"workspaces_updated_by_id_fkey f n",
	}, projectNames)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("constraints and indexes =\n%q\nwant\n%q", got, want)
	}
}

// The CHECKs accept what the domain writes and reject what bypasses it
// (M2 design 4.2, 4.3, 4.5; M3 design 4.2, 4.3, 4.4, 4.5).
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
		"INSERT INTO workspaces (id, name, slug, organization_size) VALUES ('0199a2b4-0000-7000-8000-000000000005', 'Acme', 'acme_1-2', '500+')",
		"INSERT INTO workspace_members (id, workspace_id, member_id, role) VALUES ('0199a2b4-0000-7000-8000-000000000006', " +
			"'0199a2b4-0000-7000-8000-000000000005', " + user + ", 20)",
		"INSERT INTO workspace_user_properties (id, workspace_id, user_id, navigation_project_limit, navigation_control_preference) " +
			"VALUES ('0199a2b4-0000-7000-8000-000000000007', '0199a2b4-0000-7000-8000-000000000005', " + user + ", 0, 'TABBED')",
		// An accepted invitation and a declined one, each with its response's
		// time, and a pending one.
		"INSERT INTO workspace_member_invites (id, workspace_id, email, role, accepted, responded_at) VALUES " +
			"('0199a2b4-0000-7000-8000-000000000008', '0199a2b4-0000-7000-8000-000000000005', 'élodie@exämple.com', 20, true, now()), " +
			"('0199a2b4-0000-7000-8000-000000000009', '0199a2b4-0000-7000-8000-000000000005', 'bob@corp.com', 5, false, now())",
		"INSERT INTO workspace_member_invites (id, workspace_id, email) VALUES " +
			"('0199a2b4-0000-7000-8000-00000000000a', '0199a2b4-0000-7000-8000-000000000005', 'carol@corp.com')",
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
		{"empty workspace name", "UPDATE workspaces SET name = ''", "workspaces_name_check"},
		{"upper-case slug", "UPDATE workspaces SET slug = 'Acme'", "workspaces_slug_check"},
		{"slug with a dot", "UPDATE workspaces SET slug = 'acme.io'", "workspaces_slug_check"},
		{"empty slug", "UPDATE workspaces SET slug = ''", "workspaces_slug_check"},
		{"unknown organization size", "UPDATE workspaces SET organization_size = '1000+'", "workspaces_organization_size_check"},
		{"role 10", "UPDATE workspace_members SET role = 10", "workspace_members_role_check"},
		{"role 0", "UPDATE workspace_members SET role = 0", "workspace_members_role_check"},
		{"negative project limit", "UPDATE workspace_user_properties SET navigation_project_limit = -1", "workspace_user_properties_navigation_project_limit_check"},
		{"unknown navigation control", "UPDATE workspace_user_properties SET navigation_control_preference = 'SIDEBAR'",
			"workspace_user_properties_navigation_control_preference_check"},
		{"lower-case navigation control", "UPDATE workspace_user_properties SET navigation_control_preference = 'tabbed'",
			"workspace_user_properties_navigation_control_preference_check"},
		{"an invitation's upper-case ASCII e-mail", "UPDATE workspace_member_invites SET email = 'Carol@corp.com' WHERE email = 'carol@corp.com'",
			"workspace_member_invites_email_check"},
		{"an invitation's upper-case non-ASCII e-mail", "UPDATE workspace_member_invites SET email = 'Élodie@exämple.com' WHERE role = 20",
			"workspace_member_invites_email_check"},
		{"an invitation's e-mail with a trailing tab", `UPDATE workspace_member_invites SET email = E'carol@corp.com\t' WHERE email = 'carol@corp.com'`,
			"workspace_member_invites_email_check"},
		{"an invitation's e-mail with an inner U+3000", `UPDATE workspace_member_invites SET email = U&'carol\3000@corp.com' WHERE email = 'carol@corp.com'`,
			"workspace_member_invites_email_check"},
		{"an invitation's empty e-mail", "UPDATE workspace_member_invites SET email = '' WHERE email = 'carol@corp.com'",
			"workspace_member_invites_email_check"},
		{"an invitation's role 10", "UPDATE workspace_member_invites SET role = 10 WHERE email = 'carol@corp.com'", "workspace_member_invites_role_check"},
		{"an invitation's role 0", "UPDATE workspace_member_invites SET role = 0 WHERE email = 'carol@corp.com'", "workspace_member_invites_role_check"},
		{"accepted without a response", "UPDATE workspace_member_invites SET accepted = true WHERE email = 'carol@corp.com'",
			"workspace_member_invites_responded_check"},
		{"an acceptance's time removed", "UPDATE workspace_member_invites SET responded_at = NULL WHERE accepted",
			"workspace_member_invites_responded_check"},
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

// A partial unique key holds among undeleted rows only: a second undeleted
// row with the key is refused, and soft-deleting the first frees the key
// (M3 design 3.10, 4.2, 4.3, 4.4, 4.5). A declined invitation is an
// undeleted row: it holds its address until it is deleted (M3 design 3.8).
func TestUniqueKeysHoldAmongUndeletedRowsOnly(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t, pgtest.NewDatabase(t))
	const (
		user      = "'0199a2b4-0000-7000-8000-000000000001'"
		other     = "'0199a2b4-0000-7000-8000-000000000002'"
		workspace = "'0199a2b4-0000-7000-8000-000000000005'"
		beta      = "'0199a2b4-0000-7000-8000-000000000006'"
	)
	for _, stmt := range []string{
		"INSERT INTO users (id, email, password, display_name) VALUES (" + user + ", 'alice@corp.com', 'x', 'alice')",
		"INSERT INTO users (id, email, password, display_name) VALUES (" + other + ", 'bob@corp.com', 'x', 'bob')",
		"INSERT INTO workspaces (id, name, slug) VALUES (" + workspace + ", 'Acme', 'acme')",
		"INSERT INTO workspaces (id, name, slug) VALUES (" + beta + ", 'Beta', 'beta')",
		"INSERT INTO workspace_members (id, workspace_id, member_id) VALUES (gen_random_uuid(), " + workspace + ", " + user + ")",
		"INSERT INTO workspace_user_properties (id, workspace_id, user_id) VALUES (gen_random_uuid(), " + workspace + ", " + user + ")",
		// Another account in the workspace and the account in another
		// workspace hold keys of their own: a key short of a column refuses one.
		"INSERT INTO workspace_user_properties (id, workspace_id, user_id) VALUES (gen_random_uuid(), " + workspace + ", " + other + ")",
		"INSERT INTO workspace_user_properties (id, workspace_id, user_id) VALUES (gen_random_uuid(), " + beta + ", " + user + ")",
		// carol's invitation to the workspace, declined; another address in
		// the workspace and carol in another workspace hold keys of their own.
		"INSERT INTO workspace_member_invites (id, workspace_id, email, responded_at) VALUES (gen_random_uuid(), " + workspace + ", 'carol@corp.com', now())",
		"INSERT INTO workspace_member_invites (id, workspace_id, email) VALUES (gen_random_uuid(), " + workspace + ", 'dave@corp.com')",
		"INSERT INTO workspace_member_invites (id, workspace_id, email) VALUES (gen_random_uuid(), " + beta + ", 'carol@corp.com')",
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	tests := []struct{ name, insert, softDelete, index string }{
		{
			"an account's membership of a workspace",
			"INSERT INTO workspace_members (id, workspace_id, member_id) VALUES (gen_random_uuid(), " + workspace + ", " + user + ")",
			"UPDATE workspace_members SET deleted_at = now()",
			"workspace_members_workspace_id_member_id_key",
		},
		{
			"an account's preferences in a workspace",
			"INSERT INTO workspace_user_properties (id, workspace_id, user_id) VALUES (gen_random_uuid(), " + workspace + ", " + user + ")",
			"UPDATE workspace_user_properties SET deleted_at = now()",
			"workspace_user_properties_workspace_id_user_id_key",
		},
		{
			"a declined invitation's address in a workspace",
			"INSERT INTO workspace_member_invites (id, workspace_id, email) VALUES (gen_random_uuid(), " + workspace + ", 'carol@corp.com')",
			"UPDATE workspace_member_invites SET deleted_at = now() WHERE workspace_id = " + workspace + " AND email = 'carol@corp.com'",
			"workspace_member_invites_workspace_id_email_key",
		},
		{
			"a workspace's slug",
			"INSERT INTO workspaces (id, name, slug) VALUES (gen_random_uuid(), 'Acme 2', 'acme')",
			"UPDATE workspaces SET deleted_at = now()",
			"workspaces_slug_key",
		},
	}
	// Any case can run first: each soft-deletes rows of its own table only,
	// and no key reads another table (a membership or a preferences row of a
	// deleted workspace still holds its key).
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.Exec(ctx, tt.insert)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "23505" || pgErr.ConstraintName != tt.index {
				t.Errorf("%s = %v, want unique_violation (23505) of %s", tt.insert, err, tt.index)
			}
			if _, err := pool.Exec(ctx, tt.softDelete); err != nil {
				t.Fatalf("%s: %v", tt.softDelete, err)
			}
			if _, err := pool.Exec(ctx, tt.insert); err != nil {
				t.Errorf("after %s, %s = %v; want the key free", tt.softDelete, tt.insert, err)
			}
		})
	}
}
