package migrations_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// The ids the module tables' tests write besides newIssueDatabase's, as SQL
// literals: the module's lead and a member of it, and the module of web.
const (
	moduleLead   = "'0199a2b4-0000-7000-8000-0000000000d1'"
	moduleMember = "'0199a2b4-0000-7000-8000-0000000000d2'"
	webModule    = "'0199a2b4-0000-7000-8000-0000000000d3'"
)

// newModuleDatabase is newIssueDatabase with the lead and the member, and
// web's module Alpha, led by the lead and created by him.
func newModuleDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := newIssueDatabase(t)
	accepted(t, pool,
		"INSERT INTO users (id, email, password, display_name) VALUES ("+moduleLead+", 'lead@corp.com', 'x', 'lead'), ("+moduleMember+", 'member@corp.com', 'x', 'member')",
		"INSERT INTO modules (id, workspace_id, project_id, name, lead_id, created_by_id) VALUES ("+webModule+", "+issueAcme+", "+issueWeb+", 'Alpha', "+moduleLead+", "+moduleLead+")")
	return pool
}

// accepted runs each statement and fails the test on the first refused.
func accepted(t *testing.T, pool *pgxpool.Pool, stmts ...string) {
	t.Helper()
	for _, stmt := range stmts {
		if _, err := pool.Exec(context.Background(), stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
}

// refusedBy reports a statement that the constraint did not refuse with the
// code.
func refusedBy(t *testing.T, pool *pgxpool.Pool, stmt, code, constraint string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), stmt)
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != code || pe.ConstraintName != constraint {
		t.Errorf("%s = %v; want %s of %s", stmt, err, code, constraint)
	}
}

// nullRefused reports a statement that did not fail on the NOT NULL of
// column.
func nullRefused(t *testing.T, pool *pgxpool.Pool, stmt, column string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), stmt)
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "23502" || pe.ColumnName != column {
		t.Errorf("%s = %v; want the NOT NULL of %s", stmt, err, column)
	}
}

// columnsOf are table's columns, each with its type, nullability and default.
func columnsOf(t *testing.T, pool *pgxpool.Pool, table string) []string {
	t.Helper()
	return names(t, pool, `SELECT column_name||' '||data_type||' '||is_nullable||' '||coalesce(column_default,'-') FROM information_schema.columns
 WHERE table_schema='public' AND table_name='`+table+`' ORDER BY ordinal_position`)
}

// indexesOf are table's index definitions, by name.
func indexesOf(t *testing.T, pool *pgxpool.Pool, table string) []string {
	t.Helper()
	return names(t, pool, `SELECT indexdef FROM pg_indexes WHERE schemaname='public' AND tablename='`+table+`' ORDER BY indexname`)
}

// The modules table (M6 P2 spec 2.2): its columns and indexes; the six
// statuses and no other, 'planned' by default; the dates ordered only when
// both are there; the non-empty name, unique in its project among undeleted
// modules (archived ones too) whatever its case; the lead's SET NULL keeping
// the module when the lead's account is purged, and the project's purge
// taking its modules only.
func TestModulesSchema(t *testing.T) {
	pool := newModuleDatabase(t)
	if got, want := columnsOf(t, pool, "modules"), []string{
		"id uuid NO -", "workspace_id uuid NO -", "project_id uuid NO -", "name character varying NO -",
		"description text NO ''::text", "start_date date YES -", "target_date date YES -",
		"status character varying NO 'planned'::character varying", "lead_id uuid YES -",
		"sort_order double precision NO 65535", "archived_at timestamp with time zone YES -",
		"created_by_id uuid YES -", "updated_by_id uuid YES -", "created_at timestamp with time zone NO now()",
		"updated_at timestamp with time zone NO now()", "deleted_at timestamp with time zone YES -",
	}; !slices.Equal(got, want) {
		t.Fatalf("modules columns\n%q\nwant\n%q", got, want)
	}
	if got, want := indexesOf(t, pool, "modules"), []string{
		"CREATE INDEX modules_deleted_at_idx ON public.modules USING btree (deleted_at) WHERE (deleted_at IS NOT NULL)",
		"CREATE UNIQUE INDEX modules_pkey ON public.modules USING btree (id)",
		"CREATE INDEX modules_project_id_idx ON public.modules USING btree (project_id)",
		"CREATE UNIQUE INDEX modules_project_id_name_key ON public.modules USING btree (project_id, lower((name)::text)) WHERE (deleted_at IS NULL)",
		"CREATE INDEX modules_project_list_idx ON public.modules USING btree (project_id, sort_order, created_at, id) WHERE (deleted_at IS NULL)",
		"CREATE INDEX modules_workspace_id_idx ON public.modules USING btree (workspace_id)",
	}; !slices.Equal(got, want) {
		t.Errorf("modules indexes\n%q\nwant\n%q", got, want)
	}

	update := func(set string) string { return "UPDATE modules SET " + set + " WHERE id=" + webModule }
	// Accepted: every status; no dates, either date alone, one day, a range.
	for _, set := range []string{"status='backlog'", "status='planned'", "status='in-progress'", "status='paused'", "status='completed'",
		"status='cancelled'", "start_date=NULL,target_date=NULL", "start_date='2026-11-07',target_date=NULL",
		"start_date=NULL,target_date='2026-11-07'", "start_date='2026-11-07',target_date='2026-11-07'",
		"start_date='2026-11-07',target_date='2026-11-20'"} {
		accepted(t, pool, update(set))
	}
	for _, tt := range []struct{ set, constraint string }{
		{"start_date='2026-11-08',target_date='2026-11-07'", "modules_dates_check"},
		{"status='done'", "modules_status_check"},
		{"status='Planned'", "modules_status_check"},
		{"status='in_progress'", "modules_status_check"},
		{"status=''", "modules_status_check"},
		{"name=''", "modules_name_check"},
	} {
		refusedBy(t, pool, update(tt.set), "23514", tt.constraint)
	}
	for _, column := range []string{"workspace_id", "project_id", "name", "description", "status", "sort_order", "created_at", "updated_at"} {
		nullRefused(t, pool, update(column+"=NULL"), column)
	}

	// The name is unique in its project among undeleted modules, archived
	// ones too, whatever its case; another project may take it.
	named := func(project, name string) string {
		return "INSERT INTO modules (id, workspace_id, project_id, name) VALUES (gen_random_uuid(), " + issueAcme + ", " + project + ", '" + name + "')"
	}
	refusedBy(t, pool, named(issueWeb, "ALPHA"), "23505", "modules_project_id_name_key")
	accepted(t, pool, update("archived_at=now()"))
	refusedBy(t, pool, named(issueWeb, "alpha"), "23505", "modules_project_id_name_key")
	accepted(t, pool, named(issueOps, "Alpha"), update("deleted_at=now()"), named(issueWeb, "alpha"), "UPDATE modules SET deleted_at=now() WHERE name='alpha'",
		named(issueWeb, "Alpha"))

	// Purging the lead's account keeps the module, without its lead and
	// creator.
	accepted(t, pool, "DELETE FROM users WHERE id="+moduleLead)
	var kept bool
	if err := pool.QueryRow(context.Background(), "SELECT lead_id IS NULL AND created_by_id IS NULL FROM modules WHERE id="+webModule).Scan(&kept); err != nil || !kept {
		t.Fatalf("module after its lead's purge: kept=%t, %v", kept, err)
	}
	// Purging the project takes its modules; the other project's stay.
	accepted(t, pool, "DELETE FROM projects WHERE id="+issueWeb)
	if left := names(t, pool, "SELECT project_id::text||' '||name FROM modules ORDER BY name"); !slices.Equal(left, []string{strings.Trim(issueOps, "'") + " Alpha"}) {
		t.Fatalf("modules after web's purge = %q, want ops' Alpha only", left)
	}
}

// The module_members table (M6 P2 spec 2.2): its columns and indexes; a
// member once in a module among its undeleted rows, Plane's unique key with
// deleted_at dropped; the rows going with their module, and with their
// member's account.
func TestModuleMembersSchema(t *testing.T) {
	pool := newModuleDatabase(t)
	if got, want := columnsOf(t, pool, "module_members"), []string{
		"id uuid NO -", "workspace_id uuid NO -", "project_id uuid NO -", "module_id uuid NO -", "member_id uuid NO -",
		"created_by_id uuid YES -", "updated_by_id uuid YES -", "created_at timestamp with time zone NO now()",
		"updated_at timestamp with time zone NO now()", "deleted_at timestamp with time zone YES -",
	}; !slices.Equal(got, want) {
		t.Fatalf("module_members columns\n%q\nwant\n%q", got, want)
	}
	if got, want := indexesOf(t, pool, "module_members"), []string{
		"CREATE INDEX module_members_deleted_at_idx ON public.module_members USING btree (deleted_at) WHERE (deleted_at IS NOT NULL)",
		"CREATE INDEX module_members_member_id_idx ON public.module_members USING btree (member_id)",
		"CREATE INDEX module_members_module_id_idx ON public.module_members USING btree (module_id)",
		"CREATE UNIQUE INDEX module_members_module_id_member_id_key ON public.module_members USING btree (module_id, member_id) WHERE (deleted_at IS NULL)",
		"CREATE UNIQUE INDEX module_members_pkey ON public.module_members USING btree (id)",
		"CREATE INDEX module_members_project_id_idx ON public.module_members USING btree (project_id)",
		"CREATE INDEX module_members_workspace_id_idx ON public.module_members USING btree (workspace_id)",
	}; !slices.Equal(got, want) {
		t.Errorf("module_members indexes\n%q\nwant\n%q", got, want)
	}

	member := func(user string) string {
		return "INSERT INTO module_members (id, workspace_id, project_id, module_id, member_id) VALUES (gen_random_uuid(), " + issueAcme + ", " +
			issueWeb + ", " + webModule + ", " + user + ")"
	}
	accepted(t, pool, member(moduleMember), member(moduleLead))
	refusedBy(t, pool, member(moduleMember), "23505", "module_members_module_id_member_id_key")
	// Deleted rows hold no place, however many share a moment.
	accepted(t, pool, "UPDATE module_members SET deleted_at='2026-11-07' WHERE member_id="+moduleMember, member(moduleMember),
		"UPDATE module_members SET deleted_at='2026-11-07' WHERE member_id="+moduleMember+" AND deleted_at IS NULL", member(moduleMember))
	for _, column := range []string{"workspace_id", "project_id", "module_id", "member_id", "created_at", "updated_at"} {
		nullRefused(t, pool, "UPDATE module_members SET "+column+"=NULL", column)
	}

	count := func() int {
		var n int
		if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM module_members").Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if n := count(); n != 4 {
		t.Fatalf("module_members = %d, want 4", n)
	}
	// Purging the member's account takes his rows; the lead's stays.
	accepted(t, pool, "DELETE FROM users WHERE id="+moduleMember)
	if n := count(); n != 1 {
		t.Fatalf("module_members after the member's purge = %d, want the lead's 1", n)
	}
	// Purging the module takes its members.
	accepted(t, pool, "DELETE FROM modules WHERE id="+webModule)
	if n := count(); n != 0 {
		t.Fatalf("module_members after the module's purge = %d, want 0", n)
	}
}

// The module_links table (M6 P2 spec 2.2): its columns and indexes; the
// address required and at most 2048 long; the metadata an object of
// issue_links' keys, each a string or null; an address once in a module
// among its undeleted links, another module free to take it; the links
// going with their module.
func TestModuleLinksSchema(t *testing.T) {
	pool := newModuleDatabase(t)
	if got, want := columnsOf(t, pool, "module_links"), []string{
		"id uuid NO -", "workspace_id uuid NO -", "project_id uuid NO -", "module_id uuid NO -", "title character varying YES -",
		"url text NO -", "metadata jsonb NO '{}'::jsonb", "created_by_id uuid YES -", "updated_by_id uuid YES -",
		"created_at timestamp with time zone NO now()", "updated_at timestamp with time zone NO now()", "deleted_at timestamp with time zone YES -",
	}; !slices.Equal(got, want) {
		t.Fatalf("module_links columns\n%q\nwant\n%q", got, want)
	}
	if got, want := indexesOf(t, pool, "module_links"), []string{
		"CREATE INDEX module_links_deleted_at_idx ON public.module_links USING btree (deleted_at) WHERE (deleted_at IS NOT NULL)",
		"CREATE INDEX module_links_module_id_idx ON public.module_links USING btree (module_id)",
		"CREATE INDEX module_links_module_id_url_key ON public.module_links USING hash (((((module_id)::text || ':'::text) || url))) WHERE (deleted_at IS NULL)",
		"CREATE UNIQUE INDEX module_links_pkey ON public.module_links USING btree (id)",
		"CREATE INDEX module_links_project_id_idx ON public.module_links USING btree (project_id)",
		"CREATE INDEX module_links_workspace_id_idx ON public.module_links USING btree (workspace_id)",
	}; !slices.Equal(got, want) {
		t.Errorf("module_links indexes\n%q\nwant\n%q", got, want)
	}

	const otherModule = "'0199a2b4-0000-7000-8000-0000000000d4'"
	link := func(module, url string) string {
		return "INSERT INTO module_links (id, workspace_id, project_id, module_id, url) VALUES (gen_random_uuid(), " + issueAcme + ", " +
			issueWeb + ", " + module + ", '" + url + "')"
	}
	accepted(t, pool, "INSERT INTO modules (id, workspace_id, project_id, name) VALUES ("+otherModule+", "+issueAcme+", "+issueWeb+", 'Beta')",
		link(webModule, "https://nerve.dev/a"), link(otherModule, "https://nerve.dev/a"), link(webModule, "https://"+strings.Repeat("a", 2040)))
	refusedBy(t, pool, link(webModule, "https://nerve.dev/a"), "23P01", "module_links_module_id_url_key")
	accepted(t, pool, "UPDATE module_links SET deleted_at=now() WHERE module_id="+webModule+" AND url='https://nerve.dev/a'",
		link(webModule, "https://nerve.dev/a"))
	refusedBy(t, pool, link(webModule, ""), "23514", "module_links_url_check")
	refusedBy(t, pool, link(webModule, "https://"+strings.Repeat("a", 2041)), "23514", "module_links_url_check")

	setMetadata := func(value string) string {
		return "UPDATE module_links SET metadata='" + value + "' WHERE module_id=" + webModule + " AND url='https://nerve.dev/a' AND deleted_at IS NULL"
	}
	accepted(t, pool, setMetadata(`{}`), setMetadata(`{"title":"Nerve","favicon":null,"url":"https://nerve.dev","favicon_url":"https://nerve.dev/f.ico","error":null}`),
		setMetadata(`{"error":"timeout"}`))
	for _, value := range []string{`[]`, `"title"`, `null`, `{"other":"x"}`, `{"title":1}`, `{"title":{}}`, `{"error":["x"]}`} {
		refusedBy(t, pool, setMetadata(value), "23514", "module_links_metadata_check")
	}
	for _, column := range []string{"workspace_id", "project_id", "module_id", "url", "metadata", "created_at", "updated_at"} {
		nullRefused(t, pool, "UPDATE module_links SET "+column+"=NULL", column)
	}

	// Purging a module takes its links; the other module's stay.
	accepted(t, pool, "DELETE FROM modules WHERE id="+webModule)
	if left := names(t, pool, "SELECT module_id::text||' '||url FROM module_links ORDER BY url"); !slices.Equal(left, []string{strings.Trim(otherModule, "'") + " https://nerve.dev/a"}) {
		t.Fatalf("module_links after Alpha's purge = %q, want Beta's link only", left)
	}
}
