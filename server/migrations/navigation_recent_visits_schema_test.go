package migrations_test

import (
	"slices"
	"testing"
)

// The ids the user_recent_visits table's tests write besides
// newIssueDatabase's, as SQL literals: the visitor, another account, the
// account that last wrote the rows, a second workspace with its project, and
// the visitor's visit of the project web.
const (
	visitOwner   = "'0199a2b4-0000-7000-8000-000000000101'"
	visitOther   = "'0199a2b4-0000-7000-8000-000000000102'"
	visitWriter  = "'0199a2b4-0000-7000-8000-000000000103'"
	visitBeta    = "'0199a2b4-0000-7000-8000-000000000104'"
	visitBetaApp = "'0199a2b4-0000-7000-8000-000000000105'"
	webVisit     = "'0199a2b4-0000-7000-8000-000000000106'"
)

// visitRow is the insertion of the undeleted visit id of user in workspace,
// of kind, of the target target in the project project, written by the
// writer; target is an SQL literal.
func visitRow(id, user, workspace, project, kind, target string) string {
	return "INSERT INTO user_recent_visits (id, workspace_id, project_id, user_id, entity_type, entity_identifier, visited_at, created_by_id, " +
		"updated_by_id, created_at, updated_at) VALUES (" + id + ", " + workspace + ", " + project + ", " + user + ", '" + kind + "', " + target +
		", now(), " + visitWriter + ", " + visitWriter + ", now(), now())"
}

// The user_recent_visits table (M7 design 3.9, 4.2, P3d spec 2.3): its
// columns, with no default on the instants; its indexes and foreign keys by
// name; the two kinds and no other; a project's visit of its own project.
// Among the undeleted rows an account visits a target of a workspace once,
// and the conflict target of the recording (spec 2.3) finds that key: a
// second recording changes the one row; once it is soft deleted the target
// is visited anew. Another account, another workspace have their own. Each
// CHECK's counterexample breaks that CHECK alone: PostgreSQL tests a table's
// CHECKs in the order of their names and reports the first refusing one.
func TestUserRecentVisitsSchema(t *testing.T) {
	pool := newIssueDatabase(t)
	if got, want := columnsOf(t, pool, "user_recent_visits"), []string{
		"id uuid NO -", "workspace_id uuid NO -", "project_id uuid NO -", "user_id uuid NO -", "entity_type text NO -",
		"entity_identifier uuid NO -", "visited_at timestamp with time zone NO -", "created_by_id uuid YES -", "updated_by_id uuid YES -",
		"created_at timestamp with time zone NO -", "updated_at timestamp with time zone NO -", "deleted_at timestamp with time zone YES -",
	}; !slices.Equal(got, want) {
		t.Fatalf("user_recent_visits columns\n%q\nwant\n%q", got, want)
	}
	if got, want := indexesOf(t, pool, "user_recent_visits"), []string{
		"CREATE INDEX user_recent_visits_deleted_at_idx ON public.user_recent_visits USING btree (deleted_at) WHERE (deleted_at IS NOT NULL)",
		"CREATE UNIQUE INDEX user_recent_visits_pkey ON public.user_recent_visits USING btree (id)",
		"CREATE INDEX user_recent_visits_project_id_idx ON public.user_recent_visits USING btree (project_id)",
		"CREATE INDEX user_recent_visits_user_id_idx ON public.user_recent_visits USING btree (user_id)",
		"CREATE UNIQUE INDEX user_recent_visits_user_id_workspace_id_entity_key ON public.user_recent_visits USING btree (user_id, workspace_id, entity_type, entity_identifier) WHERE (deleted_at IS NULL)",
		"CREATE INDEX user_recent_visits_user_id_workspace_id_visited_at_idx ON public.user_recent_visits USING btree (user_id, workspace_id, visited_at DESC, id DESC) WHERE (deleted_at IS NULL)",
		"CREATE INDEX user_recent_visits_workspace_id_entity_identifier_idx ON public.user_recent_visits USING btree (workspace_id, entity_identifier) WHERE (deleted_at IS NULL)",
		"CREATE INDEX user_recent_visits_workspace_id_idx ON public.user_recent_visits USING btree (workspace_id)",
	}; !slices.Equal(got, want) {
		t.Errorf("user_recent_visits indexes\n%q\nwant\n%q", got, want)
	}
	if got, want := foreignKeysOf(t, pool, "user_recent_visits"), []string{
		"user_recent_visits_created_by_id_fkey users n", "user_recent_visits_project_id_fkey projects c", "user_recent_visits_updated_by_id_fkey users n",
		"user_recent_visits_user_id_fkey users c", "user_recent_visits_workspace_id_fkey workspaces c",
	}; !slices.Equal(got, want) {
		t.Errorf("user_recent_visits foreign keys\n%q\nwant\n%q", got, want)
	}

	accepted(t, pool,
		"INSERT INTO users (id, email, password, display_name) VALUES ("+visitOwner+", 'carol@corp.com', 'x', 'carol'), ("+
			visitOther+", 'dave@corp.com', 'x', 'dave'), ("+visitWriter+", 'erin@corp.com', 'x', 'erin')",
		"INSERT INTO workspaces (id, name, slug) VALUES ("+visitBeta+", 'Beta', 'beta')",
		"INSERT INTO projects (id, workspace_id, name, identifier) VALUES ("+visitBetaApp+", "+visitBeta+", 'App', 'APP')",
		// The two kinds: a project's visit of its own project, a work item's
		// in its project.
		visitRow(webVisit, visitOwner, issueAcme, issueWeb, "project", issueWeb),
		visitRow("gen_random_uuid()", visitOwner, issueAcme, issueWeb, "issue", webIssue))

	for _, kind := range []string{"cycle", "Issue", "projects", ""} {
		refusedBy(t, pool, visitRow("gen_random_uuid()", visitOther, issueAcme, issueWeb, kind, "gen_random_uuid()"), "23514",
			"user_recent_visits_entity_type_check")
	}
	// A project's visit is of its own project.
	for _, stmt := range []string{
		visitRow("gen_random_uuid()", visitOther, issueAcme, issueWeb, "project", issueOps),
		visitRow("gen_random_uuid()", visitOther, issueAcme, issueWeb, "project", webIssue),
		"UPDATE user_recent_visits SET project_id=" + issueOps + " WHERE id=" + webVisit,
		"UPDATE user_recent_visits SET entity_identifier=" + issueOps + " WHERE id=" + webVisit,
	} {
		refusedBy(t, pool, stmt, "23514", "user_recent_visits_project_id_check")
	}
	for _, column := range []string{"workspace_id", "project_id", "user_id", "entity_type", "entity_identifier", "visited_at", "created_at", "updated_at"} {
		nullRefused(t, pool, "UPDATE user_recent_visits SET "+column+"=NULL WHERE id="+webVisit, column)
	}
	nullRefused(t, pool, "INSERT INTO user_recent_visits (id, workspace_id, project_id, user_id, entity_type, entity_identifier, visited_at) VALUES "+
		"(gen_random_uuid(), "+issueAcme+", "+issueOps+", "+visitOther+", 'project', "+issueOps+", now())", "created_at")

	// An account visits a target of a workspace once among his undeleted
	// visits. The recording's conflict target (spec 2.3) finds that key: it
	// changes the one row, its id kept; once the visit is soft deleted the
	// target is visited anew, beside the deleted row.
	again := visitRow("gen_random_uuid()", visitOwner, issueAcme, issueWeb, "project", issueWeb)
	refusedBy(t, pool, again, "23505", "user_recent_visits_user_id_workspace_id_entity_key")
	record := again + " ON CONFLICT (user_id, workspace_id, entity_type, entity_identifier) WHERE deleted_at IS NULL " +
		"DO UPDATE SET visited_at = excluded.visited_at + interval '1 hour', updated_at = excluded.updated_at, updated_by_id = " + visitOther
	inserted(t, pool, record, 1)
	// Each of carol's visits of web in acme: whether it is the first row,
	// revisited after its creation, its last writer, and whether deleted.
	webVisits := func(want ...string) {
		t.Helper()
		if got := names(t, pool, `SELECT concat_ws(' ', v.id = `+webVisit+`, v.visited_at > v.created_at, u.display_name, v.deleted_at IS NOT NULL)
			FROM user_recent_visits AS v LEFT JOIN users AS u ON u.id = v.updated_by_id
			WHERE v.user_id = `+visitOwner+` AND v.workspace_id = `+issueAcme+` AND v.entity_type = 'project' AND v.entity_identifier = `+issueWeb+`
			ORDER BY v.deleted_at NULLS FIRST`); !slices.Equal(got, want) {
			t.Fatalf("carol's visits of web = %q, want %q", got, want)
		}
	}
	webVisits("t t dave f")
	// Another account's visit of the target, and the same target id under
	// another workspace (no foreign key ties a target to it), are other
	// keys.
	accepted(t, pool, visitRow("gen_random_uuid()", visitOther, issueAcme, issueWeb, "project", issueWeb),
		visitRow("gen_random_uuid()", visitOwner, visitBeta, issueWeb, "project", issueWeb),
		"UPDATE user_recent_visits SET deleted_at=now() WHERE id="+webVisit)
	inserted(t, pool, again, 1)
	webVisits("f f erin f", "t t dave t")
	refusedBy(t, pool, again, "23505", "user_recent_visits_user_id_workspace_id_entity_key")

	// Where the rows go: the writer's purge keeps every visit without him; a
	// project's takes its visits, of it and of its work items, and not its
	// workspace's others; a workspace's its own; an account's his.
	accepted(t, pool, "DELETE FROM user_recent_visits",
		visitRow(webVisit, visitOwner, issueAcme, issueWeb, "project", issueWeb),
		visitRow("gen_random_uuid()", visitOwner, issueAcme, issueWeb, "issue", webIssue),
		visitRow("gen_random_uuid()", visitOwner, issueAcme, issueOps, "issue", opsIssue),
		visitRow("gen_random_uuid()", visitOwner, visitBeta, visitBetaApp, "project", visitBetaApp),
		visitRow("gen_random_uuid()", visitOther, issueAcme, issueWeb, "issue", webIssue),
		visitRow("gen_random_uuid()", visitOther, issueAcme, issueOps, "project", issueOps))
	left := func(stmt string, want ...string) {
		t.Helper()
		accepted(t, pool, stmt)
		if got := names(t, pool, `SELECT concat_ws(' ', u.display_name, v.entity_type, w.slug, p.identifier,
				CASE WHEN v.created_by_id IS NULL AND v.updated_by_id IS NULL THEN 'unwritten' END) COLLATE "C"
			FROM user_recent_visits AS v JOIN users AS u ON u.id = v.user_id JOIN workspaces AS w ON w.id = v.workspace_id
			JOIN projects AS p ON p.id = v.project_id ORDER BY 1`); !slices.Equal(got, want) {
			t.Fatalf("after %s, the visits = %q, want %q", stmt, got, want)
		}
	}
	left("DELETE FROM users WHERE id="+visitWriter,
		"carol issue acme OPS unwritten", "carol issue acme WEB unwritten", "carol project acme WEB unwritten", "carol project beta APP unwritten",
		"dave issue acme WEB unwritten", "dave project acme OPS unwritten")
	left("DELETE FROM projects WHERE id="+issueWeb,
		"carol issue acme OPS unwritten", "carol project beta APP unwritten", "dave project acme OPS unwritten")
	left("DELETE FROM workspaces WHERE id="+visitBeta, "carol issue acme OPS unwritten", "dave project acme OPS unwritten")
	left("DELETE FROM users WHERE id="+visitOwner, "dave project acme OPS unwritten")
}
