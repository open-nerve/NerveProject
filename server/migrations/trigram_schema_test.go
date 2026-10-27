package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

func TestMigrationsGoUpDownAndUpAgain(t *testing.T) {
	for _, preexisting := range []bool{false, true} {
		name := "absent"
		if preexisting {
			name = "preexisting"
		}
		t.Run(name, func(t *testing.T) { testMigrationsRoundTrip(t, preexisting) })
	}
}

// Every migration can go up, down and up again (M2 design 4.1), River's
// too: goose runs each of its halves as one statement (M2 design 3.15).
func testMigrationsRoundTrip(t *testing.T, preexisting bool) {
	ctx := context.Background()
	pool := newPool(t, pgtest.NewEmptyDatabase(t))
	var before uint32
	if preexisting {
		if _, err := pool.Exec(ctx, "CREATE EXTENSION pg_trgm"); err != nil {
			t.Fatal(err)
		}
		before = trigramOID(t, pool)
	}
	m, err := postgres.NewMigrator(pool, migrations.FS())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.Close() })

	up, err := m.Up(ctx)
	if err != nil || len(up) != 29 {
		t.Fatalf("Up() = %d migrations, %v; want 29", len(up), err)
	}
	wantSchema := []struct {
		query string
		names []string
	}{
		{tablesQuery, []string{"api_tokens", "auth_sessions", "comment_reactions", "draft_issues", "issue_activities", "issue_assignees", "issue_comments", "issue_description_versions", "issue_labels", "issue_links", "issue_mentions", "issue_reactions", "issue_relations",
			"issue_subscribers", "issues", "labels", "profiles", "project_members", "project_user_properties", "projects", "river_job",
			"river_leader", "river_notification", "river_queue", "states", "users", "workspace_member_invites", "workspace_members",
			"workspace_user_properties", "workspaces"}},
		{enumsQuery, []string{"river_job_state"}},
		{functionsQuery, []string{"river_job_state_in_bitmask"}},
	}
	checkSchema := func(stage string) {
		t.Helper()
		for _, want := range wantSchema {
			if got := names(t, pool, want.query); !slices.Equal(got, want.names) {
				t.Errorf("after %s, %s = %q, want %q", stage, want.query, got, want.names)
			}
		}
	}
	checkSchema("Up")
	oid := trigramOID(t, pool)
	if preexisting && oid != before {
		t.Fatalf("preexisting pg_trgm OID changed: %d -> %d", before, oid)
	}
	assertTrigramIndex(t, pool)
	// Application overloads are still catalogued even when they share an
	// extension function's name: classification follows pg_depend, not names.
	if _, err := pool.Exec(ctx, "CREATE FUNCTION public.similarity(integer) RETURNS integer LANGUAGE SQL AS 'SELECT $1'"); err != nil {
		t.Fatal(err)
	}
	if got := names(t, pool, functionsQuery); !slices.Equal(got, []string{"river_job_state_in_bitmask", "similarity"}) {
		t.Fatalf("application overload missing from function catalog: %v", got)
	}
	if _, err := pool.Exec(ctx, "DROP FUNCTION public.similarity(integer)"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Down(ctx); err != nil {
		t.Fatal(err)
	}
	if got := names(t, pool, "SELECT relname FROM pg_class WHERE relnamespace='public'::regnamespace AND relname='issues_name_trgm_idx'"); len(got) != 0 {
		t.Fatalf("00029 Down left index: %v", got)
	}
	if got := trigramOID(t, pool); got != oid {
		t.Fatalf("00029 Down changed extension OID: %d -> %d", oid, got)
	}
	for range len(up) - 1 {
		if _, err := m.Down(ctx); err != nil {
			t.Fatalf("Down() error = %v", err)
		}
	}
	for _, query := range []string{tablesQuery, enumsQuery, functionsQuery} {
		if got := names(t, pool, query); len(got) != 0 {
			t.Errorf("after every Down, %s = %q, want none", query, got)
		}
	}
	if again, err := m.Up(ctx); err != nil || len(again) != 29 {
		t.Errorf("Up() again = %d migrations, %v; want 29", len(again), err)
	}
	checkSchema("Up again")
	assertTrigramIndex(t, pool)
	if got := trigramOID(t, pool); got != oid {
		t.Fatalf("Down/Up changed extension OID: %d -> %d", oid, got)
	}
	t.Logf("preexisting=%v pg_trgm OID=%d unchanged across Down/Up", preexisting, oid)
}

func trigramOID(t *testing.T, pool *pgxpool.Pool) uint32 {
	t.Helper()
	var oid uint32
	if err := pool.QueryRow(context.Background(), "SELECT oid FROM pg_extension WHERE extname='pg_trgm'").Scan(&oid); err != nil {
		t.Fatal("pg_trgm extension:", err)
	}
	return oid
}

func assertTrigramIndex(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	var method, key, opclass, predicate string
	if err := pool.QueryRow(context.Background(), `
  SELECT am.amname, a.attname, opc.opcname, pg_get_expr(i.indpred,i.indrelid)
  FROM pg_index i JOIN pg_class c ON c.oid=i.indexrelid
  JOIN pg_am am ON am.oid=c.relam
  JOIN pg_attribute a ON a.attrelid=i.indrelid AND a.attnum=i.indkey[0]
  JOIN pg_opclass opc ON opc.oid=i.indclass[0]
  WHERE c.oid='public.issues_name_trgm_idx'::regclass AND i.indrelid='public.issues'::regclass AND i.indnkeyatts=1
 `).Scan(&method, &key, &opclass, &predicate); err != nil {
		t.Fatal(err)
	}
	if method != "gin" || key != "name" || opclass != "gin_trgm_ops" || predicate != "(deleted_at IS NULL)" {
		t.Fatalf("trigram index = %s(%s %s) WHERE %s", method, key, opclass, predicate)
	}
	t.Logf("issues_name_trgm_idx USING %s(%s %s) WHERE %s", method, key, opclass, predicate)
}
