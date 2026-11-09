package migrations_test

import (
	"context"
	"slices"
	"strings"
	"testing"
)

// planningPreferenceTables are the planning module's preference tables (M6/P3b
// spec 2.4): an account's display settings of one cycle's or one module's
// page, as project_user_properties' without the navigation and the sidebar
// order. column is the page's key, which the filters refuse; other is the
// other page's, which they take. first and second are pages of web, ops one
// of ops (newMembershipDatabase).
var planningPreferenceTables = []struct{ table, parent, column, other, first, second, ops string }{
	{"cycle_user_properties", "cycles", "cycle_id", "module_id", sprintOne, sprintTwo, opsSprint},
	{"module_user_properties", "modules", "module_id", "cycle_id", alphaModule, betaModule, opsModule},
}

// The two accounts whose display settings the preference tables' tests write,
// as SQL literals; bob (newIssueDatabase) writes the rows.
const (
	preferenceAnn = "'0199a2b4-0000-7000-8000-0000000000f1'"
	preferenceCal = "'0199a2b4-0000-7000-8000-0000000000f2'"
)

// listFilterKeys are the filter keys of every work-item list page: the
// project's twelve but cycle_id and module_id.
var listFilterKeys = strings.Fields("state_id state_group priority label_id assignee_id created_by_id mention_id subscriber_id start_date target_date")

// Each preference table: its columns, indexes and foreign keys; the
// project's defaults; the filters taking the eleven keys of the page, each an
// array of strings, and refusing the page's own; the project's display
// filters; exactly the fourteen display properties, each a boolean; one
// undeleted row per page and account, deleted rows holding no place; and
// where the rows go: the writer's purge keeps them without him, the page's,
// the account's, the project's and the workspace's deletion each take their
// own.
func TestPlanningPreferencesSchema(t *testing.T) {
	for _, p := range planningPreferenceTables {
		t.Run(p.table, func(t *testing.T) {
			pool := newMembershipDatabase(t)
			ctx := context.Background()
			accepted(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ("+preferenceAnn+", 'ann@corp.com', 'x', 'ann'), ("+
				preferenceCal+", 'cal@corp.com', 'x', 'cal')")
			// A jsonb column's default is read back by value on the first row.
			if got, want := names(t, pool, `SELECT column_name||' '||data_type||' '||is_nullable||' '||
 CASE WHEN data_type='jsonb' AND column_default IS NOT NULL THEN 'default' ELSE coalesce(column_default,'-') END
 FROM information_schema.columns WHERE table_schema='public' AND table_name='`+p.table+`' ORDER BY ordinal_position`), []string{
				"id uuid NO -", "workspace_id uuid NO -", "project_id uuid NO -", p.column + " uuid NO -", "user_id uuid NO -",
				"filters jsonb NO default", "display_filters jsonb NO default", "display_properties jsonb NO default",
				"created_by_id uuid YES -", "updated_by_id uuid YES -", "created_at timestamp with time zone NO now()",
				"updated_at timestamp with time zone NO now()", "deleted_at timestamp with time zone YES -",
			}; !slices.Equal(got, want) {
				t.Fatalf("%s columns\n%q\nwant\n%q", p.table, got, want)
			}
			named := strings.NewReplacer("TABLE", p.table, "PAGE", p.column, "PARENT", p.parent)
			pinned := func(what string, got, want []string) {
				t.Helper()
				for i := range want {
					want[i] = named.Replace(want[i])
				}
				slices.Sort(got)
				slices.Sort(want)
				if !slices.Equal(got, want) {
					t.Errorf("%s %s\n%q\nwant\n%q", p.table, what, got, want)
				}
			}
			pinned("indexes", indexesOf(t, pool, p.table), []string{
				"CREATE INDEX TABLE_PAGE_idx ON public.TABLE USING btree (PAGE)",
				"CREATE UNIQUE INDEX TABLE_PAGE_user_id_key ON public.TABLE USING btree (PAGE, user_id) WHERE (deleted_at IS NULL)",
				"CREATE INDEX TABLE_deleted_at_idx ON public.TABLE USING btree (deleted_at) WHERE (deleted_at IS NOT NULL)",
				"CREATE UNIQUE INDEX TABLE_pkey ON public.TABLE USING btree (id)",
				"CREATE INDEX TABLE_project_id_idx ON public.TABLE USING btree (project_id)",
				"CREATE INDEX TABLE_user_id_idx ON public.TABLE USING btree (user_id)",
				"CREATE INDEX TABLE_workspace_id_idx ON public.TABLE USING btree (workspace_id)",
			})
			pinned("foreign keys", foreignKeysOf(t, pool, p.table), []string{
				"TABLE_created_by_id_fkey users n", "TABLE_PAGE_fkey PARENT c", "TABLE_project_id_fkey projects c",
				"TABLE_updated_by_id_fkey users n", "TABLE_user_id_fkey users c", "TABLE_workspace_id_fkey workspaces c",
			})

			row := func(page, project, user string) string {
				return "INSERT INTO " + p.table + " (id, workspace_id, project_id, " + p.column + ", user_id, created_by_id, updated_by_id) VALUES (gen_random_uuid(), " +
					issueAcme + ", " + project + ", " + page + ", " + user + ", " + issueBob + ", " + issueBob + ")"
			}
			mine := " WHERE " + p.column + "=" + p.first + " AND user_id=" + preferenceAnn
			accepted(t, pool, row(p.first, issueWeb, preferenceAnn))
			var filters string
			var displayFilters, displayProperties bool
			if err := pool.QueryRow(ctx, "SELECT filters::text, display_filters=$1::jsonb, display_properties=$2::jsonb FROM "+p.table+mine,
				displayFiltersDefault, planningDisplayDefault).Scan(&filters, &displayFilters, &displayProperties); err != nil || filters != "{}" || !displayFilters || !displayProperties {
				t.Fatalf("defaults: filters %s, the project's display filters %t, the fourteen properties shown %t, %v", filters, displayFilters, displayProperties, err)
			}

			set := func(column, value string) string {
				return "UPDATE " + p.table + " SET " + column + "='" + value + "'" + mine
			}
			every := []string{}
			for _, key := range append(slices.Clone(listFilterKeys), p.other) {
				every = append(every, `"`+key+`":["none","`+key+`"]`)
				accepted(t, pool, set("filters", `{"`+key+`":[]}`), set("filters", `{`+every[len(every)-1]+`}`))
			}
			accepted(t, pool, set("filters", "{"+strings.Join(every, ",")+"}"))
			for _, value := range []string{`{"` + p.column + `":[]}`, `{"` + p.column + `":["none"]}`, `{"label_id":[],"` + p.column + `":[]}`,
				`{"` + p.column + `s":[]}`, `{"project_id":[]}`, `[]`, `null`, `"label_id"`, `{"label_id":"none"}`, `{"label_id":[1]}`,
				`{"label_id":[null]}`, `{"label_id":[["x"]]}`} {
				refusedBy(t, pool, set("filters", value), "23514", p.table+"_filters_check")
			}

			// Grouping by the page's own key is a shape the database takes (spec R8).
			own := strings.NewReplacer(`"group_by":null`, `"group_by":"`+p.column+`"`, `"order_by":"-created_at"`,
				`"order_by":"`+strings.TrimSuffix(p.column, "_id")+`_name"`).Replace(displayFiltersDefault)
			accepted(t, pool, set("display_filters", own), set("display_filters", displayFiltersDefault))
			for _, value := range []string{`[]`, `{}`, `null`,
				strings.Replace(displayFiltersDefault, `,"calendar":{"layout":"month","show_weekends":false}`, ``, 1),
				strings.Replace(displayFiltersDefault, `{"layout"`, `{"future":true,"layout"`, 1),
				strings.Replace(displayFiltersDefault, `"group_by":null`, `"group_by":1`, 1),
				strings.Replace(displayFiltersDefault, `"sub_issue":true`, `"sub_issue":"true"`, 1),
				strings.Replace(displayFiltersDefault, `{"layout":"month","show_weekends":false}`, `{}`, 1),
				strings.Replace(displayFiltersDefault, `"show_weekends":false`, `"show_weekends":false,"week":1`, 1),
			} {
				refusedBy(t, pool, set("display_filters", value), "23514", p.table+"_display_filters_check")
			}

			accepted(t, pool, set("display_properties", strings.Replace(planningDisplayDefault, `"cycle":true,"modules":true`, `"cycle":false,"modules":false`, 1)))
			for _, value := range []string{`[]`, `{}`, `null`, attachmentDisplayDefault,
				strings.Replace(planningDisplayDefault, `,"modules":true`, ``, 1),
				strings.Replace(planningDisplayDefault, `"modules":true`, `"modules":true,"future":true`, 1),
				strings.Replace(planningDisplayDefault, `"modules":true`, `"modules":"yes"`, 1),
				strings.Replace(planningDisplayDefault, `"cycle":true`, `"cycle":null`, 1),
			} {
				refusedBy(t, pool, set("display_properties", value), "23514", p.table+"_display_properties_check")
			}
			for _, column := range []string{"workspace_id", "project_id", p.column, "user_id", "filters", "display_filters", "display_properties", "created_at", "updated_at"} {
				nullRefused(t, pool, "UPDATE "+p.table+" SET "+column+"=NULL", column)
			}

			refusedBy(t, pool, row(p.first, issueWeb, preferenceAnn), "23505", p.table+"_"+p.column+"_user_id_key")
			accepted(t, pool, row(p.first, issueWeb, preferenceCal), row(p.second, issueWeb, preferenceAnn), row(p.ops, issueOps, preferenceAnn))
			// Deleted rows hold no place, however many share a moment.
			accepted(t, pool, "UPDATE "+p.table+" SET deleted_at='2026-11-09'"+mine, row(p.first, issueWeb, preferenceAnn),
				"UPDATE "+p.table+" SET deleted_at='2026-11-09'"+mine+" AND deleted_at IS NULL", row(p.first, issueWeb, preferenceAnn))

			count := func(where string) int {
				t.Helper()
				var n int
				if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+p.table+" WHERE "+where).Scan(&n); err != nil {
					t.Fatal(err)
				}
				return n
			}
			all := count("true")
			accepted(t, pool, "DELETE FROM users WHERE id="+issueBob)
			if n, written := count("true"), count("created_by_id IS NOT NULL OR updated_by_id IS NOT NULL"); n != all || written != 0 {
				t.Fatalf("after the writer's purge: %d rows, %d with a writer; want %d, none", n, written, all)
			}
			for _, step := range []struct{ stmt, rows string }{
				{"DELETE FROM " + p.parent + " WHERE id=" + p.second, p.column + "=" + p.second},
				{"DELETE FROM users WHERE id=" + preferenceCal, "user_id=" + preferenceCal},
				{"DELETE FROM projects WHERE id=" + issueOps, "project_id=" + issueOps},
				{"DELETE FROM workspaces WHERE id=" + issueAcme, "true"},
			} {
				gone := count(step.rows)
				if gone == 0 {
					t.Fatalf("no rows where %s before %s: the step proves nothing", step.rows, step.stmt)
				}
				accepted(t, pool, step.stmt)
				if n, left := count("true"), count(step.rows); n != all-gone || left != 0 {
					t.Fatalf("after %s: %d rows, %d where %s; want %d, none", step.stmt, n, left, step.rows, all-gone)
				}
				all -= gone
			}
		})
	}
}
