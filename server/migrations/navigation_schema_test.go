package migrations_test

import (
	"slices"
	"strings"
	"testing"
)

// The ids the issue_views table's tests write besides newIssueDatabase's,
// as SQL literals: the owner of the views, the account that last wrote
// them, a second workspace, a view of web and a view of acme.
const (
	viewOwner     = "'0199a2b4-0000-7000-8000-0000000000b1'"
	viewWriter    = "'0199a2b4-0000-7000-8000-0000000000b2'"
	viewBeta      = "'0199a2b4-0000-7000-8000-0000000000b3'"
	webView       = "'0199a2b4-0000-7000-8000-0000000000b4'"
	workspaceView = "'0199a2b4-0000-7000-8000-0000000000b5'"
)

// viewOf is the insertion of the undeleted view id named name of project in
// workspace, a workspace view when project is NULL, owned by owner and
// written by the writer, with no filter and the default display settings.
func viewOf(id, workspace, project, owner, name string) string {
	return "INSERT INTO issue_views (id, workspace_id, project_id, name, filters, display_filters, display_properties, owned_by_id, " +
		"created_by_id, updated_by_id, created_at, updated_at) VALUES (" + id + ", " + workspace + ", " + project + ", '" + name + "', '{}', '" +
		displayFiltersDefault + "', '" + planningDisplayDefault + "', " + owner + ", " + viewWriter + ", " + viewWriter + ", now(), now())"
}

// viewKinds are the two kinds of view (P3b spec 2.3) by a view of each: the
// saved filter keys it takes, the ten of every list page and the project's
// cycle_id and module_id or the workspace's project_id, and the other
// kind's, which it refuses.
var viewKinds = []struct {
	kind, id       string
	keys, otherKey []string
}{
	{"project", webView, append(slices.Clone(listFilterKeys), "cycle_id", "module_id"), []string{"project_id"}},
	{"workspace", workspaceView, append(slices.Clone(listFilterKeys), "project_id"), []string{"cycle_id", "module_id"}},
}

// The issue_views table (M7 design 4.2, P3b spec 2.3): its columns, indexes
// and foreign keys; the filters taking the saved keys of the view's kind,
// twelve for a project's view and eleven for a workspace's, each an array of
// strings, and refusing the other kind's, sub_issue and archived; the
// display settings shaped as a list page's; the icon shaped as a project's;
// a non-empty name of up to 255 characters; and where the rows go: the writer's purge keeps them
// without him, the owner's, the project's and the workspace's take their
// own.
func TestIssueViewsSchema(t *testing.T) {
	pool := newIssueDatabase(t)
	if got, want := columnsOf(t, pool, "issue_views"), []string{
		"id uuid NO -", "workspace_id uuid NO -", "project_id uuid YES -", "name character varying NO -", "description text NO ''::text",
		"logo_props jsonb NO '{}'::jsonb", "filters jsonb NO -", "display_filters jsonb NO -", "display_properties jsonb NO -",
		"owned_by_id uuid NO -", "created_by_id uuid YES -", "updated_by_id uuid YES -", "created_at timestamp with time zone NO -",
		"updated_at timestamp with time zone NO -", "deleted_at timestamp with time zone YES -",
	}; !slices.Equal(got, want) {
		t.Fatalf("issue_views columns\n%q\nwant\n%q", got, want)
	}
	if got, want := indexesOf(t, pool, "issue_views"), []string{
		"CREATE INDEX issue_views_deleted_at_idx ON public.issue_views USING btree (deleted_at) WHERE (deleted_at IS NOT NULL)",
		"CREATE INDEX issue_views_owned_by_id_idx ON public.issue_views USING btree (owned_by_id)",
		"CREATE UNIQUE INDEX issue_views_pkey ON public.issue_views USING btree (id)",
		"CREATE INDEX issue_views_project_id_created_at_idx ON public.issue_views USING btree (project_id, created_at DESC, id DESC) WHERE ((deleted_at IS NULL) AND (project_id IS NOT NULL))",
		"CREATE INDEX issue_views_project_id_idx ON public.issue_views USING btree (project_id)",
		"CREATE INDEX issue_views_workspace_id_created_at_idx ON public.issue_views USING btree (workspace_id, created_at DESC, id DESC) WHERE ((deleted_at IS NULL) AND (project_id IS NULL))",
		"CREATE INDEX issue_views_workspace_id_idx ON public.issue_views USING btree (workspace_id)",
	}; !slices.Equal(got, want) {
		t.Errorf("issue_views indexes\n%q\nwant\n%q", got, want)
	}
	if got, want := foreignKeysOf(t, pool, "issue_views"), []string{
		"issue_views_created_by_id_fkey users n", "issue_views_owned_by_id_fkey users c", "issue_views_project_id_fkey projects c",
		"issue_views_updated_by_id_fkey users n", "issue_views_workspace_id_fkey workspaces c",
	}; !slices.Equal(got, want) {
		t.Errorf("issue_views foreign keys\n%q\nwant\n%q", got, want)
	}

	accepted(t, pool,
		"INSERT INTO users (id, email, password, display_name) VALUES ("+viewOwner+", 'carol@corp.com', 'x', 'carol'), ("+
			viewWriter+", 'dave@corp.com', 'x', 'dave')",
		viewOf(webView, issueAcme, issueWeb, viewOwner, "carol web"),
		viewOf(workspaceView, issueAcme, "NULL", viewOwner, "carol acme"))
	for _, v := range viewKinds {
		set := func(column, value string) string {
			return "UPDATE issue_views SET " + column + "='" + value + "' WHERE id=" + v.id
		}
		every := []string{}
		for _, key := range v.keys {
			every = append(every, `"`+key+`":["none","`+key+`"]`)
			accepted(t, pool, set("filters", `{"`+key+`":[]}`), set("filters", `{`+every[len(every)-1]+`}`))
		}
		accepted(t, pool, set("filters", "{"+strings.Join(every, ",")+"}"))
		refused := []string{`{"sub_issue":[]}`, `{"sub_issue":["true"]}`, `{"archived":[]}`, `{"label_ids":[]}`, `[]`, `null`, `"label_id"`,
			`{"label_id":"none"}`, `{"label_id":[1]}`, `{"label_id":[null]}`, `{"label_id":[["x"]]}`}
		for _, key := range v.otherKey {
			refused = append(refused, `{"`+key+`":[]}`, `{"label_id":[],"`+key+`":["none"]}`)
		}
		for _, value := range refused {
			refusedBy(t, pool, set("filters", value), "23514", "issue_views_filters_check")
		}
		accepted(t, pool, set("filters", `{}`))
	}
	// The kind is the project's absence: a project's view keeping its cycle
	// filter cannot become a workspace's, nor a workspace's its project
	// filter a project's.
	accepted(t, pool, `UPDATE issue_views SET filters='{"cycle_id":["none"]}' WHERE id=`+webView,
		`UPDATE issue_views SET filters='{"project_id":["none"]}' WHERE id=`+workspaceView)
	refusedBy(t, pool, "UPDATE issue_views SET project_id=NULL WHERE id="+webView, "23514", "issue_views_filters_check")
	refusedBy(t, pool, "UPDATE issue_views SET project_id="+issueWeb+" WHERE id="+workspaceView, "23514", "issue_views_filters_check")

	set := func(column, value string) string {
		return "UPDATE issue_views SET " + column + "='" + value + "' WHERE id=" + webView
	}
	spreadsheet := strings.Replace(displayFiltersDefault, `"layout":"list"`, `"layout":"spreadsheet"`, 1)
	accepted(t, pool, set("display_filters", spreadsheet), set("display_filters", displayFiltersDefault))
	for _, value := range []string{`[]`, `{}`, `null`,
		strings.Replace(displayFiltersDefault, `,"calendar":{"layout":"month","show_weekends":false}`, ``, 1),
		strings.Replace(displayFiltersDefault, `{"layout"`, `{"future":true,"layout"`, 1),
		strings.Replace(displayFiltersDefault, `"group_by":null`, `"group_by":1`, 1),
		strings.Replace(displayFiltersDefault, `"sub_issue":true`, `"sub_issue":"true"`, 1),
		strings.Replace(displayFiltersDefault, `{"layout":"month","show_weekends":false}`, `{}`, 1),
		strings.Replace(displayFiltersDefault, `"show_weekends":false`, `"show_weekends":false,"week":1`, 1),
	} {
		refusedBy(t, pool, set("display_filters", value), "23514", "issue_views_display_filters_check")
	}
	accepted(t, pool, set("display_properties", strings.Replace(planningDisplayDefault, `"cycle":true,"modules":true`, `"cycle":false,"modules":false`, 1)))
	for _, value := range []string{`[]`, `{}`, `null`, attachmentDisplayDefault,
		strings.Replace(planningDisplayDefault, `"modules":true`, `"modules":true,"future":true`, 1),
		strings.Replace(planningDisplayDefault, `"modules":true`, `"modules":"yes"`, 1),
		strings.Replace(planningDisplayDefault, `"cycle":true`, `"cycle":null`, 1),
	} {
		refusedBy(t, pool, set("display_properties", value), "23514", "issue_views_display_properties_check")
	}
	for _, value := range []string{`{}`, `{"emoji": {"value": "128640"}}`, `{"icon": {"name": "home", "color": "#6d7b8a"}}`,
		`{"in_use": "icon", "emoji": {"value": "128640", "url": "https://example.com/e.png"}, "icon": {"name": "home", "background_color": "#fff"}}`} {
		accepted(t, pool, set("logo_props", value))
	}
	for _, value := range []string{`{"unexpected": true}`, `{"in_use": "other"}`, `{"in_use": 17}`, `{"emoji": "x"}`, `{"emoji": {"value": 1}}`,
		`{"emoji": {"url": null}}`, `{"icon": {"shape": "round"}}`, `{"icon": {"color": null}}`, `{"icon": []}`, `[]`, `"x"`, `null`} {
		refusedBy(t, pool, set("logo_props", value), "23514", "issue_views_logo_props_check")
	}
	refusedBy(t, pool, set("name", ""), "23514", "issue_views_name_check")
	// The name holds 255 characters, however many bytes each takes, and
	// not 256.
	accepted(t, pool, set("name", strings.Repeat("视", 255)), set("name", "carol web"))
	refusedBy(t, pool, set("name", strings.Repeat("视", 256)), "22001", "")
	for _, column := range []string{"workspace_id", "name", "description", "logo_props", "filters", "display_filters", "display_properties",
		"owned_by_id", "created_at", "updated_at"} {
		nullRefused(t, pool, "UPDATE issue_views SET "+column+"=NULL WHERE id="+webView, column)
	}

	// Purging the writer's account keeps every view without him; the
	// owner's takes his views; a project's its views and not its
	// workspace's; a workspace's its own.
	accepted(t, pool, "INSERT INTO workspaces (id, name, slug) VALUES ("+viewBeta+", 'Beta', 'beta')",
		viewOf("gen_random_uuid()", issueAcme, issueWeb, issueBob, "bob web"),
		viewOf("gen_random_uuid()", issueAcme, issueOps, issueBob, "bob ops"),
		viewOf("gen_random_uuid()", issueAcme, "NULL", issueBob, "bob acme"),
		viewOf("gen_random_uuid()", viewBeta, "NULL", issueBob, "bob beta"))
	left := func(stmt string, want ...string) {
		t.Helper()
		accepted(t, pool, stmt)
		if got := names(t, pool, "SELECT name || ' ' || concat_ws(' ', created_by_id, updated_by_id) FROM issue_views ORDER BY name"); !slices.Equal(got, want) {
			t.Fatalf("after %s, the views and their writers = %q, want %q", stmt, got, want)
		}
	}
	left("DELETE FROM users WHERE id="+viewWriter, "bob acme ", "bob beta ", "bob ops ", "bob web ", "carol acme ", "carol web ")
	left("DELETE FROM users WHERE id="+viewOwner, "bob acme ", "bob beta ", "bob ops ", "bob web ")
	left("DELETE FROM projects WHERE id="+issueWeb, "bob acme ", "bob beta ", "bob ops ")
	left("DELETE FROM workspaces WHERE id="+viewBeta, "bob acme ", "bob ops ")
}

// navigationNames are the navigation module's constraint and index names
// with what each is (TestConstraintAndIndexNames).
var navigationNames = []string{
	"issue_views_created_by_id_fkey f n",
	"issue_views_deleted_at_idx iw",
	"issue_views_display_filters_check c",
	"issue_views_display_properties_check c",
	"issue_views_filters_check c",
	"issue_views_logo_props_check c",
	"issue_views_name_check c",
	"issue_views_owned_by_id_fkey f c",
	"issue_views_owned_by_id_idx i",
	"issue_views_pkey iu",
	"issue_views_pkey p",
	"issue_views_project_id_created_at_idx iw",
	"issue_views_project_id_fkey f c",
	"issue_views_project_id_idx i",
	"issue_views_updated_by_id_fkey f n",
	"issue_views_workspace_id_created_at_idx iw",
	"issue_views_workspace_id_fkey f c",
	"issue_views_workspace_id_idx i",
	"user_favorites_created_by_id_fkey f n",
	"user_favorites_deleted_at_idx iw",
	"user_favorites_entity_type_check c",
	"user_favorites_entity_type_entity_identifier_idx iw",
	"user_favorites_name_check c",
	"user_favorites_parent_id_fkey f c",
	"user_favorites_parent_id_idx i",
	"user_favorites_pkey iu",
	"user_favorites_pkey p",
	"user_favorites_project_id_check c",
	"user_favorites_project_id_fkey f c",
	"user_favorites_project_id_idx i",
	"user_favorites_sequence_check c",
	"user_favorites_shape_check c",
	"user_favorites_updated_by_id_fkey f n",
	"user_favorites_user_id_entity_type_entity_identifier_key iuw",
	"user_favorites_user_id_fkey f c",
	"user_favorites_user_id_idx i",
	"user_favorites_user_id_workspace_id_idx iw",
	"user_favorites_user_id_workspace_id_name_key iuw",
	"user_favorites_workspace_id_fkey f c",
	"user_favorites_workspace_id_idx i",
}
