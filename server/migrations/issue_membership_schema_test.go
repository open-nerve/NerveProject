package migrations_test

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The ids the membership tables' tests write besides newIssueDatabase's, as
// SQL literals: two cycles and two modules of web, a cycle and a module of
// ops, and another work item of web.
const (
	sprintOne   = "'0199a2b4-0000-7000-8000-0000000000e1'"
	sprintTwo   = "'0199a2b4-0000-7000-8000-0000000000e2'"
	opsSprint   = "'0199a2b4-0000-7000-8000-0000000000e3'"
	alphaModule = "'0199a2b4-0000-7000-8000-0000000000e4'"
	betaModule  = "'0199a2b4-0000-7000-8000-0000000000e5'"
	opsModule   = "'0199a2b4-0000-7000-8000-0000000000e6'"
	webOther    = "'0199a2b4-0000-7000-8000-0000000000e7'"
)

// newMembershipDatabase is newIssueDatabase with the cycles, the modules and
// web's other work item, numbered 2.
func newMembershipDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := newIssueDatabase(t)
	accepted(t, pool,
		"INSERT INTO cycles (id, workspace_id, project_id, name) VALUES ("+sprintOne+", "+issueAcme+", "+issueWeb+", 'Sprint 1'), ("+
			sprintTwo+", "+issueAcme+", "+issueWeb+", 'Sprint 2'), ("+opsSprint+", "+issueAcme+", "+issueOps+", 'Ops')",
		"INSERT INTO modules (id, workspace_id, project_id, name) VALUES ("+alphaModule+", "+issueAcme+", "+issueWeb+", 'Alpha'), ("+
			betaModule+", "+issueAcme+", "+issueWeb+", 'Beta'), ("+opsModule+", "+issueAcme+", "+issueOps+", 'Ops')",
		"INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id) VALUES ("+webOther+", "+issueAcme+", "+issueWeb+", "+
			webState+", 'Other', 2)")
	return pool
}

// membership is the insertion of the work item issue of project into the
// target of table's column, written by bob.
func membership(table, column, project, target, issue string) string {
	return "INSERT INTO " + table + " (id, workspace_id, project_id, " + column + ", issue_id, created_by_id, updated_by_id) VALUES (gen_random_uuid(), " +
		issueAcme + ", " + project + ", " + target + ", " + issue + ", " + issueBob + ", " + issueBob + ")"
}

// foreignKeysOf are table's foreign keys, each with the table it references
// and its ON DELETE (c CASCADE, n SET NULL).
func foreignKeysOf(t *testing.T, pool *pgxpool.Pool, table string) []string {
	t.Helper()
	return names(t, pool, `SELECT conname||' '||confrelid::regclass::text||' '||confdeltype::text FROM pg_constraint
 WHERE conrelid='public.`+table+`'::regclass AND contype='f' ORDER BY conname`)
}

// The cycle_issues table (M6 P3a spec 2.2): its columns, indexes and foreign
// keys; a cycle holding many work items, a work item in at most one cycle
// among the undeleted rows (v0 design 5.3), however many deleted rows it
// has, so moving it deletes the old row and inserts a new one (spec R4);
// where its rows go with the rows above them (membershipsGo).
func TestCycleIssuesSchema(t *testing.T) {
	pool := newMembershipDatabase(t)
	if got, want := columnsOf(t, pool, "cycle_issues"), []string{
		"id uuid NO -", "workspace_id uuid NO -", "project_id uuid NO -", "cycle_id uuid NO -", "issue_id uuid NO -",
		"created_by_id uuid YES -", "updated_by_id uuid YES -", "created_at timestamp with time zone NO now()",
		"updated_at timestamp with time zone NO now()", "deleted_at timestamp with time zone YES -",
	}; !slices.Equal(got, want) {
		t.Fatalf("cycle_issues columns\n%q\nwant\n%q", got, want)
	}
	if got, want := indexesOf(t, pool, "cycle_issues"), []string{
		"CREATE INDEX cycle_issues_cycle_id_idx ON public.cycle_issues USING btree (cycle_id)",
		"CREATE INDEX cycle_issues_deleted_at_idx ON public.cycle_issues USING btree (deleted_at) WHERE (deleted_at IS NOT NULL)",
		"CREATE INDEX cycle_issues_issue_id_idx ON public.cycle_issues USING btree (issue_id)",
		"CREATE UNIQUE INDEX cycle_issues_issue_id_key ON public.cycle_issues USING btree (issue_id) WHERE (deleted_at IS NULL)",
		"CREATE UNIQUE INDEX cycle_issues_pkey ON public.cycle_issues USING btree (id)",
		"CREATE INDEX cycle_issues_project_id_idx ON public.cycle_issues USING btree (project_id)",
		"CREATE INDEX cycle_issues_workspace_id_idx ON public.cycle_issues USING btree (workspace_id)",
	}; !slices.Equal(got, want) {
		t.Errorf("cycle_issues indexes\n%q\nwant\n%q", got, want)
	}
	if got, want := foreignKeysOf(t, pool, "cycle_issues"), []string{
		"cycle_issues_created_by_id_fkey users n", "cycle_issues_cycle_id_fkey cycles c", "cycle_issues_issue_id_fkey issues c",
		"cycle_issues_project_id_fkey projects c", "cycle_issues_updated_by_id_fkey users n", "cycle_issues_workspace_id_fkey workspaces c",
	}; !slices.Equal(got, want) {
		t.Errorf("cycle_issues foreign keys\n%q\nwant\n%q", got, want)
	}

	in := func(cycle, issue string) string {
		return membership("cycle_issues", "cycle_id", issueWeb, cycle, issue)
	}
	accepted(t, pool, in(sprintOne, webIssue), in(sprintOne, webOther), membership("cycle_issues", "cycle_id", issueOps, opsSprint, opsIssue))
	refusedBy(t, pool, in(sprintTwo, webIssue), "23505", "cycle_issues_issue_id_key")
	refusedBy(t, pool, in(sprintOne, webIssue), "23505", "cycle_issues_issue_id_key")
	// Deleted rows hold no place, however many share a moment.
	accepted(t, pool, "UPDATE cycle_issues SET deleted_at='2026-11-08' WHERE issue_id="+webIssue, in(sprintTwo, webIssue),
		"UPDATE cycle_issues SET deleted_at='2026-11-08' WHERE issue_id="+webIssue+" AND deleted_at IS NULL", in(sprintOne, webIssue))
	for _, column := range []string{"workspace_id", "project_id", "cycle_id", "issue_id", "created_at", "updated_at"} {
		nullRefused(t, pool, "UPDATE cycle_issues SET "+column+"=NULL", column)
	}
	membershipsGo(t, pool, "cycle_issues", "DELETE FROM cycles WHERE id="+sprintTwo, "cycle_id="+sprintTwo)
}

// The module_issues table (M6 P3a spec 2.2): its columns, indexes and
// foreign keys; a work item in many modules and a module holding many work
// items, a pair once among the undeleted rows, however many deleted rows it
// has; where its rows go with the rows above them (membershipsGo).
func TestModuleIssuesSchema(t *testing.T) {
	pool := newMembershipDatabase(t)
	if got, want := columnsOf(t, pool, "module_issues"), []string{
		"id uuid NO -", "workspace_id uuid NO -", "project_id uuid NO -", "module_id uuid NO -", "issue_id uuid NO -",
		"created_by_id uuid YES -", "updated_by_id uuid YES -", "created_at timestamp with time zone NO now()",
		"updated_at timestamp with time zone NO now()", "deleted_at timestamp with time zone YES -",
	}; !slices.Equal(got, want) {
		t.Fatalf("module_issues columns\n%q\nwant\n%q", got, want)
	}
	if got, want := indexesOf(t, pool, "module_issues"), []string{
		"CREATE INDEX module_issues_deleted_at_idx ON public.module_issues USING btree (deleted_at) WHERE (deleted_at IS NOT NULL)",
		"CREATE INDEX module_issues_issue_id_idx ON public.module_issues USING btree (issue_id)",
		"CREATE UNIQUE INDEX module_issues_issue_id_module_id_key ON public.module_issues USING btree (issue_id, module_id) WHERE (deleted_at IS NULL)",
		"CREATE INDEX module_issues_module_id_idx ON public.module_issues USING btree (module_id)",
		"CREATE UNIQUE INDEX module_issues_pkey ON public.module_issues USING btree (id)",
		"CREATE INDEX module_issues_project_id_idx ON public.module_issues USING btree (project_id)",
		"CREATE INDEX module_issues_workspace_id_idx ON public.module_issues USING btree (workspace_id)",
	}; !slices.Equal(got, want) {
		t.Errorf("module_issues indexes\n%q\nwant\n%q", got, want)
	}
	if got, want := foreignKeysOf(t, pool, "module_issues"), []string{
		"module_issues_created_by_id_fkey users n", "module_issues_issue_id_fkey issues c", "module_issues_module_id_fkey modules c",
		"module_issues_project_id_fkey projects c", "module_issues_updated_by_id_fkey users n", "module_issues_workspace_id_fkey workspaces c",
	}; !slices.Equal(got, want) {
		t.Errorf("module_issues foreign keys\n%q\nwant\n%q", got, want)
	}

	in := func(module, issue string) string {
		return membership("module_issues", "module_id", issueWeb, module, issue)
	}
	accepted(t, pool, in(alphaModule, webIssue), in(betaModule, webIssue), in(alphaModule, webOther),
		membership("module_issues", "module_id", issueOps, opsModule, opsIssue))
	refusedBy(t, pool, in(alphaModule, webIssue), "23505", "module_issues_issue_id_module_id_key")
	// Deleted rows hold no place, however many share a moment.
	pair := " WHERE issue_id=" + webIssue + " AND module_id=" + alphaModule
	accepted(t, pool, "UPDATE module_issues SET deleted_at='2026-11-08'"+pair, in(alphaModule, webIssue),
		"UPDATE module_issues SET deleted_at='2026-11-08'"+pair+" AND deleted_at IS NULL", in(alphaModule, webIssue))
	for _, column := range []string{"workspace_id", "project_id", "module_id", "issue_id", "created_at", "updated_at"} {
		nullRefused(t, pool, "UPDATE module_issues SET "+column+"=NULL", column)
	}
	membershipsGo(t, pool, "module_issues", "DELETE FROM modules WHERE id="+betaModule, "module_id="+betaModule)
}

// membershipsGo checks where table's rows go, each step taking its own rows
// only: purging their author's account keeps them without him; purging the
// target (deleteTarget, whose rows are targetRows) takes its rows, web's
// other work item its rows, web its rows, and the workspace the rest.
func membershipsGo(t *testing.T, pool *pgxpool.Pool, table, deleteTarget, targetRows string) {
	t.Helper()
	count := func(where string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+table+" WHERE "+where).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	steps := []struct{ stmt, rows string }{
		{deleteTarget, targetRows},
		{"DELETE FROM issues WHERE id=" + webOther, "issue_id=" + webOther},
		{"DELETE FROM projects WHERE id=" + issueWeb, "project_id=" + issueWeb},
		{"DELETE FROM workspaces WHERE id=" + issueAcme, "true"},
	}
	all := count("true")
	accepted(t, pool, "DELETE FROM users WHERE id="+issueBob)
	if n, authored := count("true"), count("created_by_id IS NOT NULL OR updated_by_id IS NOT NULL"); n != all || authored != 0 {
		t.Fatalf("%s after the author's purge: %d rows, %d with an author; want %d, none", table, n, authored, all)
	}
	for _, step := range steps {
		gone := count(step.rows)
		if gone == 0 {
			t.Fatalf("%s has no rows where %s before %s: the step proves nothing", table, step.rows, step.stmt)
		}
		accepted(t, pool, step.stmt)
		if n, left := count("true"), count(step.rows); n != all-gone || left != 0 {
			t.Fatalf("%s after %s: %d rows, %d where %s; want %d, none", table, step.stmt, n, left, step.rows, all-gone)
		}
		all -= gone
	}
}
