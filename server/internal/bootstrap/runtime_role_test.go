package bootstrap

import (
	"context"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The role nerve serves with when another role owns the tables and runs the
// migrations (README "部署"): a login role in the group role nerve_runtime,
// which deploy/runtime-grants.sql grants to. These tests run nerve on such a
// role, and hold the file to every relation and function of the schema.

// splitRoles is a database whose tables belong to an owner role that ran the
// migrations and then deploy/runtime-grants.sql, and a login role in
// nerve_runtime to serve with.
type splitRoles struct {
	owner      *pgxpool.Pool // the owner of the tables
	server     *pgxpool.Pool // the login role nerve serves with
	serverName string
	serverURL  string
}

func newSplitRoles(t *testing.T) splitRoles {
	t.Helper()
	ctx := context.Background()
	adminURL := pgtest.NewEmptyDatabase(t)
	admin := openPool(t, adminURL)
	u, err := url.Parse(adminURL)
	if err != nil {
		t.Fatal(err)
	}
	db := strings.TrimPrefix(u.Path, "/")
	// The roles belong to the cluster, which the tests of this binary share: named after the database, which
	// is new; nerve_runtime once.
	ownerName, serverName := db+"_owner", db+"_server"
	for _, sql := range []string{
		"CREATE ROLE " + pgx.Identifier{ownerName}.Sanitize() + " LOGIN PASSWORD 'owner'",
		"ALTER DATABASE " + pgx.Identifier{db}.Sanitize() + " OWNER TO " + pgx.Identifier{ownerName}.Sanitize(),
		"DO $$ BEGIN CREATE ROLE nerve_runtime NOLOGIN; EXCEPTION WHEN duplicate_object THEN NULL; END $$",
		"CREATE ROLE " + pgx.Identifier{serverName}.Sanitize() + " LOGIN PASSWORD 'server' IN ROLE nerve_runtime",
	} {
		if _, err := admin.Exec(ctx, sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
	as := func(name, password string) string {
		v := *u
		v.User = url.UserPassword(name, password)
		return v.String()
	}
	owner := openPool(t, as(ownerName, "owner"))
	m, err := postgres.NewMigrator(owner, migrations.FS())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Close() })
	if _, err := m.Up(ctx); err != nil {
		t.Fatalf("migrate as the owner: %v", err)
	}
	grants, err := os.ReadFile(grantsFile())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, string(grants)); err != nil {
		t.Fatalf("run deploy/runtime-grants.sql as the owner: %v", err)
	}
	serverURL := as(serverName, "server")
	return splitRoles{owner: owner, server: openPool(t, serverURL), serverName: serverName, serverURL: serverURL}
}

// grantsFile is deploy/runtime-grants.sql, found from this file, three
// directories below the repository root (tests run without -trimpath).
func grantsFile() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "deploy", "runtime-grants.sql")
}

// nerve serves on the role: ready, River's jobs run (the session cleanup
// completes as the app starts), and River's daily reindex goes through, as
// River runs it, index by index.
func TestTheRuntimeRoleServesWithTheGrantsFile(t *testing.T) {
	roles := newSplitRoles(t)
	ctx := context.Background()
	user, expired, live := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for _, insert := range []struct {
		sql  string
		args []any
	}{
		{"INSERT INTO users (id, email, password, display_name) VALUES ($1, 'runtime@example.com', 'x', 'runtime')", []any{user}},
		{`INSERT INTO auth_sessions (id, user_id, token_hash, expires_at) VALUES
			($1, $3, sha256('a'), now() - interval '1 minute'), ($2, $3, sha256('b'), now() + interval '1 hour')`, []any{expired, live, user}},
	} {
		if _, err := roles.owner.Exec(ctx, insert.sql, insert.args...); err != nil {
			t.Fatal(err)
		}
	}
	var logs lockedBuffer
	base := startAppLogging(t, testConfig(t, roles.serverURL, false), migrations.FS(), slog.New(slog.NewTextHandler(&logs, nil)))

	if status, p := getReadyz(t, base); status != 200 {
		t.Errorf("/readyz = %d %+v, want 200", status, p)
	}
	for deadline := time.Now().Add(15 * time.Second); cleanupRuns(t, roles.owner) == 0; time.Sleep(50 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("the session cleanup did not run as %s; logs:\n%s", roles.serverName, logs.String())
		}
	}
	if sessionExists(t, roles.owner, expired) || !sessionExists(t, roles.owner, live) {
		t.Errorf("after the cleanup: expired session left %v, live one left %v; want only the live one",
			sessionExists(t, roles.owner, expired), sessionExists(t, roles.owner, live))
	}
	if all := logs.String(); !strings.Contains(all, `msg="jobs started"`) || strings.Contains(all, "permission denied") {
		t.Errorf("logs want jobs started and no permission denied:\n%s", all)
	}
	for _, index := range river.ReindexerIndexNamesDefault() {
		if _, err := roles.server.Exec(ctx, "REINDEX INDEX CONCURRENTLY "+pgx.Identifier{index}.Sanitize()); err != nil {
			t.Errorf("REINDEX INDEX CONCURRENTLY %s as %s: %v", index, roles.serverName, err)
		}
	}
}

// Every table, view, materialized view, sequence and function the
// migrations leave in the schema public has its grant, and no more: a
// migration that adds one fails here until deploy/runtime-grants.sql grants
// it. The runtime role reads and writes every table but goose's record,
// which it only reads; it reads the views; it uses the sequences of the
// tables it writes; it may reindex River's jobs; it runs the functions,
// River's river_job_state_in_bitmask among them, which it may through
// PUBLIC's default EXECUTE. Types are left to PUBLIC's default USAGE.
func TestTheGrantsFileCoversEveryRelationAndFunction(t *testing.T) {
	roles := newSplitRoles(t)
	rows, err := roles.owner.Query(context.Background(), `
		SELECT c.relname, c.relkind::text,
		       CASE WHEN c.relkind = 'S' THEN ARRAY[has_sequence_privilege($1, c.oid, 'USAGE'),
		                       has_sequence_privilege($1, c.oid, 'SELECT'), has_sequence_privilege($1, c.oid, 'UPDATE')]
		            ELSE ARRAY[has_table_privilege($1, c.oid, 'SELECT'), has_table_privilege($1, c.oid, 'INSERT'),
		                       has_table_privilege($1, c.oid, 'UPDATE'), has_table_privilege($1, c.oid, 'DELETE'),
		                       has_table_privilege($1, c.oid, 'TRUNCATE'), has_table_privilege($1, c.oid, 'REFERENCES'),
		                       has_table_privilege($1, c.oid, 'TRIGGER'), has_table_privilege($1, c.oid, 'MAINTAIN')] END
		  FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		 WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p', 'v', 'm', 'S')
		UNION ALL
		SELECT p.oid::regprocedure::text, 'f', ARRAY[has_function_privilege($1, p.oid, 'EXECUTE')]
		  FROM pg_proc p JOIN pg_namespace n ON n.oid = p.pronamespace
		 WHERE n.nspname = 'public'
		 ORDER BY 1`, roles.serverName)
	if err != nil {
		t.Fatal(err)
	}
	privileges := map[string][]string{
		"S": {"USAGE", "SELECT", "UPDATE"},
		"f": {"EXECUTE"},
	}
	tablePrivileges := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "TRUNCATE", "REFERENCES", "TRIGGER", "MAINTAIN"}
	dml := []string{"SELECT", "INSERT", "UPDATE", "DELETE"}
	seen := map[string]int{}
	for rows.Next() {
		var name, kind string
		var has []bool
		if err := rows.Scan(&name, &kind, &has); err != nil {
			t.Fatal(err)
		}
		seen[kind]++
		names, ok := privileges[kind]
		if !ok {
			names = tablePrivileges
		}
		var got, want []string
		for i, ok := range has {
			if ok {
				got = append(got, names[i])
			}
		}
		switch {
		case name == "goose_db_version_id_seq":
		case kind == "S":
			want = []string{"USAGE"}
		case kind == "f":
			want = []string{"EXECUTE"}
		case kind == "v", kind == "m", name == "goose_db_version":
			want = []string{"SELECT"}
		case name == "river_job":
			want = append(slices.Clone(dml), "MAINTAIN")
		default:
			want = dml
		}
		if !slices.Equal(got, want) {
			t.Errorf("%s: %s has %v, want %v: deploy/runtime-grants.sql grants each of them", name, roles.serverName, got, want)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if seen["r"] == 0 || seen["S"] == 0 || seen["f"] == 0 {
		t.Fatalf("the schema public has %v tables (r), sequences (S) and functions (f), want some of each", seen)
	}
}
