package migrations_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// projectNames are the constraints and indexes of the project module's
// tables (M3 design 4.6–4.9), as TestConstraintAndIndexNames reads them:
// the part of its want that the migrations 00010–00013 add.
var projectNames = []string{
	"project_members_created_by_id_fkey f n",
	"project_members_member_id_fkey f c",
	"project_members_member_id_idx iw",
	"project_members_pkey iu",
	"project_members_pkey p",
	"project_members_project_id_fkey f c",
	"project_members_project_id_idx i",
	"project_members_project_id_member_id_key iuw",
	"project_members_role_check c",
	"project_members_updated_by_id_fkey f n",
	"project_members_workspace_id_fkey f c",
	"project_members_workspace_id_idx i",
	"project_user_properties_created_by_id_fkey f n",
	"project_user_properties_pkey iu",
	"project_user_properties_pkey p",
	"project_user_properties_preferences_check c",
	"project_user_properties_project_id_fkey f c",
	"project_user_properties_project_id_idx i",
	"project_user_properties_project_id_user_id_key iuw",
	"project_user_properties_updated_by_id_fkey f n",
	"project_user_properties_user_id_fkey f c",
	"project_user_properties_workspace_id_fkey f c",
	"project_user_properties_workspace_id_idx i",
	"projects_archive_in_check c",
	"projects_created_by_id_fkey f n",
	"projects_default_assignee_id_fkey f n",
	"projects_identifier_check c",
	"projects_last_issue_sequence_check c",
	"projects_logo_props_check c",
	"projects_name_check c",
	"projects_network_check c",
	"projects_pkey iu",
	"projects_pkey p",
	"projects_project_lead_id_fkey f n",
	"projects_updated_by_id_fkey f n",
	"projects_workspace_id_fkey f c",
	"projects_workspace_id_identifier_key iuw",
	"projects_workspace_id_idx i",
	"projects_workspace_id_name_key iuw",
	"states_created_by_id_fkey f n",
	"states_group_check c",
	"states_name_check c",
	"states_pkey iu",
	"states_pkey p",
	"states_project_id_default_key iuw",
	"states_project_id_fkey f c",
	"states_project_id_idx i",
	"states_project_id_name_key iuw",
	"states_project_id_triage_key iuw",
	"states_updated_by_id_fkey f n",
	"states_workspace_id_fkey f c",
	"states_workspace_id_idx i",
}

// The project tables' CHECKs accept what the domain writes and reject what
// bypasses it (M3 design 3.17, 3.19, 4.6–4.9). projects_logo_props_check
// takes five valid values, the four of 4.6 and the web app's create body,
// and refuses twenty counterexamples: the ten of 4.6, Codex S5's two among
// them; one for each conjunct those ten leave untried (the emoji's keys and
// url; the icon an object, its name and background color); and a second
// wrong type for each of the five texts, so that none of their checks
// weakened to "not that one type" passes.
func TestProjectChecksRejectCounterexamples(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t, pgtest.NewDatabase(t))
	const (
		user      = "'0199a2b4-0000-7000-8000-000000000001'"
		workspace = "'0199a2b4-0000-7000-8000-000000000002'"
		project   = "'0199a2b4-0000-7000-8000-000000000003'"
	)
	for _, stmt := range []string{
		"INSERT INTO users (id, email, password, display_name) VALUES (" + user + ", 'alice@corp.com', 'x', 'alice')",
		"INSERT INTO workspaces (id, name, slug) VALUES (" + workspace + ", 'Acme', 'acme')",
		"INSERT INTO projects (id, workspace_id, name, identifier) VALUES (" + project + ", " + workspace + ", 'Web', 'WEB')",
		"INSERT INTO project_members (id, workspace_id, project_id, member_id) VALUES (gen_random_uuid(), " + workspace + ", " + project + ", " + user + ")",
		"INSERT INTO project_user_properties (id, workspace_id, project_id, user_id) VALUES (gen_random_uuid(), " + workspace + ", " + project +
			", " + user + ")",
		"INSERT INTO states (id, workspace_id, project_id, name, color) VALUES (gen_random_uuid(), " + workspace + ", " + project + ", 'Backlog', '#60646C')",
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	// What the domain writes, and the edges of each rule, all accepted: the
	// four logo_props values of 4.6 and the web app's create body, an emoji
	// in use (core/components/projects/create/utils.ts:14-19); a name with a
	// backslash (the CHECK's \| is the bar, not a backslash); the upper-case
	// letters of the identifier's set; every state group.
	for _, stmt := range []string{
		"UPDATE projects SET logo_props = '{}'",
		`UPDATE projects SET logo_props = '{"emoji": {"value": "128640"}}'`,
		`UPDATE projects SET logo_props = '{"icon": {"name": "home", "color": "#6d7b8a"}}'`,
		`UPDATE projects SET logo_props = '{"in_use": "icon", "emoji": {"value": "128640", "url": "https://example.com/e.png"}, ` +
			`"icon": {"name": "home", "color": "#6d7b8a", "background_color": "#ffffff"}}'`,
		`UPDATE projects SET logo_props = '{"in_use": "emoji", "emoji": {"value": "128640"}}'`,
		`UPDATE projects SET name = E'研发 Web_2 [beta] \\ /'`,
		"UPDATE projects SET identifier = 'ÇŞĞİÖÜ0129'",
		"UPDATE projects SET identifier = 'A'",
		"UPDATE projects SET network = 0",
		"UPDATE projects SET archive_in = 12",
		"UPDATE project_members SET role = 20",
		"UPDATE project_members SET role = 15",
		`UPDATE project_user_properties SET preferences = '{"navigation": {"default_tab": "cycles", "hide_in_more_menu": ["intake"]}}'`,
		`UPDATE states SET "group" = 'unstarted'`,
		`UPDATE states SET "group" = 'started'`,
		`UPDATE states SET "group" = 'completed'`,
		`UPDATE states SET "group" = 'triage'`,
		`UPDATE states SET "group" = 'cancelled'`,
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Errorf("%s: %v, want it accepted", stmt, err)
		}
	}
	prefs := func(json string) string { return "UPDATE project_user_properties SET preferences = '" + json + "'" }
	tests := []struct{ name, stmt, constraint string }{
		{"empty project name", "UPDATE projects SET name = ''", "projects_name_check"},
		{"lower-case identifier", "UPDATE projects SET identifier = 'web'", "projects_identifier_check"},
		{"empty identifier", "UPDATE projects SET identifier = ''", "projects_identifier_check"},
		{"identifier of 11 characters", "UPDATE projects SET identifier = 'ABCDEFGHIJK'", "projects_identifier_check"},
		{"identifier with -", "UPDATE projects SET identifier = 'WEB-2'", "projects_identifier_check"},
		{"identifier with a space", "UPDATE projects SET identifier = 'WEB 2'", "projects_identifier_check"},
		{"identifier with another letter", "UPDATE projects SET identifier = 'ÉQUIPE'", "projects_identifier_check"},
		{"identifier with a trailing newline", "UPDATE projects SET identifier = E'WEB\\n'", "projects_identifier_check"},
		{"network 1", "UPDATE projects SET network = 1", "projects_network_check"},
		{"network -1", "UPDATE projects SET network = -1", "projects_network_check"},
		{"archive_in 13", "UPDATE projects SET archive_in = 13", "projects_archive_in_check"},
		{"archive_in -1", "UPDATE projects SET archive_in = -1", "projects_archive_in_check"},
		{"negative last_issue_sequence", "UPDATE projects SET last_issue_sequence = -1", "projects_last_issue_sequence_check"},
		{"logo_props with an unknown key", `UPDATE projects SET logo_props = '{"unexpected": true}'`, "projects_logo_props_check"},
		{"logo_props' in_use a number, emoji an array", `UPDATE projects SET logo_props = '{"in_use": 17, "emoji": []}'`, "projects_logo_props_check"},
		{"logo_props' in_use other", `UPDATE projects SET logo_props = '{"in_use": "other"}'`, "projects_logo_props_check"},
		{"logo_props' emoji value a number", `UPDATE projects SET logo_props = '{"emoji": {"value": 1}}'`, "projects_logo_props_check"},
		{"logo_props' emoji a string", `UPDATE projects SET logo_props = '{"emoji": "x"}'`, "projects_logo_props_check"},
		{"logo_props' icon with an unknown key", `UPDATE projects SET logo_props = '{"icon": {"shape": "round"}}'`, "projects_logo_props_check"},
		{"logo_props' icon color null", `UPDATE projects SET logo_props = '{"icon": {"color": null}}'`, "projects_logo_props_check"},
		{"logo_props' emoji with an unknown key", `UPDATE projects SET logo_props = '{"emoji": {"shape": "x"}}'`, "projects_logo_props_check"},
		{"logo_props' emoji url a number", `UPDATE projects SET logo_props = '{"emoji": {"url": 1}}'`, "projects_logo_props_check"},
		{"logo_props' icon a string", `UPDATE projects SET logo_props = '{"icon": "x"}'`, "projects_logo_props_check"},
		{"logo_props' icon name a number", `UPDATE projects SET logo_props = '{"icon": {"name": 1}}'`, "projects_logo_props_check"},
		{"logo_props' icon background_color a number", `UPDATE projects SET logo_props = '{"icon": {"background_color": 1}}'`,
			"projects_logo_props_check"},
		// Each text's second wrong type.
		{"logo_props' emoji value a boolean", `UPDATE projects SET logo_props = '{"emoji": {"value": true}}'`, "projects_logo_props_check"},
		{"logo_props' emoji url null", `UPDATE projects SET logo_props = '{"emoji": {"url": null}}'`, "projects_logo_props_check"},
		{"logo_props' icon name an array", `UPDATE projects SET logo_props = '{"icon": {"name": []}}'`, "projects_logo_props_check"},
		{"logo_props' icon color a number", `UPDATE projects SET logo_props = '{"icon": {"color": 1}}'`, "projects_logo_props_check"},
		{"logo_props' icon background_color an object", `UPDATE projects SET logo_props = '{"icon": {"background_color": {}}}'`,
			"projects_logo_props_check"},
		{"logo_props an array", `UPDATE projects SET logo_props = '[]'`, "projects_logo_props_check"},
		{"logo_props a string", `UPDATE projects SET logo_props = '"x"'`, "projects_logo_props_check"},
		{"logo_props JSON null", `UPDATE projects SET logo_props = 'null'`, "projects_logo_props_check"},
		{"project role 10", "UPDATE project_members SET role = 10", "project_members_role_check"},
		{"project role 0", "UPDATE project_members SET role = 0", "project_members_role_check"},
		{"preferences an array", prefs(`[]`), "project_user_properties_preferences_check"},
		{"preferences a scalar", prefs(`5`), "project_user_properties_preferences_check"},
		{"preferences JSON null", prefs(`null`), "project_user_properties_preferences_check"},
		{"preferences without navigation", prefs(`{}`), "project_user_properties_preferences_check"},
		{"preferences with pages", prefs(`{"navigation": {"default_tab": "work_items", "hide_in_more_menu": []}, "pages": {}}`),
			"project_user_properties_preferences_check"},
		{"navigation an array", prefs(`{"navigation": []}`), "project_user_properties_preferences_check"},
		{"navigation without default_tab", prefs(`{"navigation": {"hide_in_more_menu": []}}`), "project_user_properties_preferences_check"},
		{"navigation without hide_in_more_menu", prefs(`{"navigation": {"default_tab": "work_items"}}`), "project_user_properties_preferences_check"},
		{"navigation with another key", prefs(`{"navigation": {"default_tab": "work_items", "hide_in_more_menu": [], "x": 1}}`),
			"project_user_properties_preferences_check"},
		{"default_tab a number", prefs(`{"navigation": {"default_tab": 1, "hide_in_more_menu": []}}`), "project_user_properties_preferences_check"},
		{"hide_in_more_menu a string", prefs(`{"navigation": {"default_tab": "work_items", "hide_in_more_menu": "cycles"}}`),
			"project_user_properties_preferences_check"},
		{"empty state name", "UPDATE states SET name = ''", "states_name_check"},
		{"unknown group", `UPDATE states SET "group" = 'other'`, "states_group_check"},
		{"upper-case group", `UPDATE states SET "group" = 'Backlog'`, "states_group_check"},
	}
	// Each of Plane's forbidden characters in a name (M3 design 3.19).
	for _, c := range "&+,:;$^}{*=?@#|'<>.()%!-" {
		name := strings.ReplaceAll("Web"+string(c)+"2", "'", "''")
		tests = append(tests, struct{ name, stmt, constraint string }{"name with " + string(c), "UPDATE projects SET name = '" + name + "'",
			"projects_name_check"})
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

// The project tables' partial unique keys hold among undeleted rows only:
// a second undeleted row with the key is refused, and soft-deleting the
// first frees the key (M3 design 3.17, 3.19, 4.6–4.9). Another workspace,
// project or account holds keys of its own: a key short of a column
// refuses one of the seeds. A project's or a state's name that differs
// from another in case only is another name, as in Plane (3.17, 3.19): a
// key that folds case refuses one of the seeds too.
func TestProjectUniqueKeysHoldAmongUndeletedRowsOnly(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t, pgtest.NewDatabase(t))
	const (
		alice = "'0199a2b4-0000-7000-8000-000000000001'"
		bob   = "'0199a2b4-0000-7000-8000-000000000002'"
		acme  = "'0199a2b4-0000-7000-8000-000000000003'"
		beta  = "'0199a2b4-0000-7000-8000-000000000004'"
		web   = "'0199a2b4-0000-7000-8000-000000000005'"
		ops   = "'0199a2b4-0000-7000-8000-000000000006'"
		other = "'0199a2b4-0000-7000-8000-000000000007'"
	)
	project := func(id, workspace, name, identifier string) string {
		return "INSERT INTO projects (id, workspace_id, name, identifier) VALUES (" + id + ", " + workspace + ", '" + name + "', '" + identifier + "')"
	}
	member := func(table, column, project, user string) string {
		return "INSERT INTO " + table + " (id, workspace_id, project_id, " + column + ") VALUES (gen_random_uuid(), " + acme + ", " + project + ", " + user + ")"
	}
	state := func(project, name, group string, isDefault bool) string {
		return fmt.Sprintf(`INSERT INTO states (id, workspace_id, project_id, name, color, "group", "default") VALUES (gen_random_uuid(), %s, %s, '%s', '#60646C', '%s', %t)`,
			acme, project, name, group, isDefault)
	}
	for _, stmt := range []string{
		"INSERT INTO users (id, email, password, display_name) VALUES (" + alice + ", 'alice@corp.com', 'x', 'alice'), (" + bob + ", 'bob@corp.com', 'x', 'bob')",
		"INSERT INTO workspaces (id, name, slug) VALUES (" + acme + ", 'Acme', 'acme'), (" + beta + ", 'Beta', 'beta')",
		project(web, acme, "Web", "WEB"), project(ops, acme, "Ops", "OPS"),
		// Another workspace holds the same name and identifier.
		project(other, beta, "Web", "WEB"),
		// web's name in lower case is another name.
		project("gen_random_uuid()", acme, "web", "WEBL"),
		// alice in web; bob in web and alice in ops hold keys of their own.
		member("project_members", "member_id", web, alice), member("project_members", "member_id", web, bob),
		member("project_members", "member_id", ops, alice),
		member("project_user_properties", "user_id", web, alice), member("project_user_properties", "user_id", web, bob),
		member("project_user_properties", "user_id", ops, alice),
		// web's default, triage, Todo and todo, another name; ops has its own.
		state(web, "Backlog", "backlog", true), state(web, "Triage", "triage", false), state(web, "Todo", "unstarted", false),
		state(web, "todo", "unstarted", false),
		state(ops, "Backlog", "backlog", true), state(ops, "Triage", "triage", false), state(ops, "Todo", "unstarted", false),
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	tests := []struct{ name, insert, softDelete, index string }{
		{"a project's identifier in a workspace", project("gen_random_uuid()", acme, "Web 2", "WEB"),
			"UPDATE projects SET deleted_at = now() WHERE identifier = 'WEB' AND workspace_id = " + acme, "projects_workspace_id_identifier_key"},
		{"a project's name in a workspace", project("gen_random_uuid()", acme, "Ops", "OPS2"),
			"UPDATE projects SET deleted_at = now() WHERE name = 'Ops' AND workspace_id = " + acme, "projects_workspace_id_name_key"},
		{"an account's membership of a project", member("project_members", "member_id", web, alice),
			"UPDATE project_members SET deleted_at = now()", "project_members_project_id_member_id_key"},
		{"an account's display settings in a project", member("project_user_properties", "user_id", web, alice),
			"UPDATE project_user_properties SET deleted_at = now()", "project_user_properties_project_id_user_id_key"},
		{"a state's name in a project", state(web, "Todo", "started", false),
			"UPDATE states SET deleted_at = now() WHERE name = 'Todo' AND project_id = " + web, "states_project_id_name_key"},
		{"a project's default state", state(web, "Other", "backlog", true),
			`UPDATE states SET deleted_at = now() WHERE "default" AND project_id = ` + web, "states_project_id_default_key"},
		{"a project's triage state", state(web, "Intake", "triage", false),
			`UPDATE states SET deleted_at = now() WHERE "group" = 'triage' AND project_id = ` + web, "states_project_id_triage_key"},
	}
	// Any case can run first: each soft-deletes rows of its own table only,
	// and none of those rows is another case's key.
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
