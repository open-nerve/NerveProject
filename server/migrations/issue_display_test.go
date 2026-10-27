package migrations_test

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

const displayFiltersDefault = `{"layout":"list","group_by":null,"sub_group_by":null,"order_by":"-created_at","sub_issue":true,"show_empty_groups":true,"calendar":{"layout":"month","show_weekends":false}}`
const displayPropertiesDefault = `{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"created_on":true,"updated_on":true}`

// The real old schema and row must survive adding, dropping and adding the
// columns. Shape errors must remain CHECK failures, not JSON operator errors.
func TestIssueDisplayMigration(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t, pgtest.NewEmptyDatabase(t))
	old, throughDisplay := fstest.MapFS{}, fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.FS(), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() > "00026_z" {
			continue
		}
		data, err := fs.ReadFile(migrations.FS(), e.Name())
		if err != nil {
			t.Fatal(err)
		}
		throughDisplay[e.Name()] = &fstest.MapFile{Data: data}
		if e.Name() <= "00025_z" {
			old[e.Name()] = throughDisplay[e.Name()]
		}
	}
	m, err := postgres.NewMigrator(pool, old)
	if err != nil {
		t.Fatal(err)
	}
	oldMigrator, oldClosed := m, false
	t.Cleanup(func() {
		if !oldClosed {
			if err := oldMigrator.Close(); err != nil {
				t.Error(err)
			}
		}
	})
	up, err := m.Up(ctx)
	if err != nil || len(up) != 25 {
		t.Fatalf("old Up: %v, %v", up, err)
	}
	for i, migration := range up {
		if migration.Version != int64(i+1) {
			t.Fatalf("old version %d: %+v", i, migration)
		}
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	oldClosed = true
	for _, sql := range []string{
		`INSERT INTO users(id,email,password,display_name) VALUES('0199a2b4-0000-7000-8000-000000000001','display@example.com','x','display')`,
		`INSERT INTO workspaces(id,name,slug) VALUES('0199a2b4-0000-7000-8000-000000000002','Display','display')`,
		`INSERT INTO projects(id,workspace_id,name,identifier) VALUES('0199a2b4-0000-7000-8000-000000000003','0199a2b4-0000-7000-8000-000000000002','Display','DSP')`,
		`INSERT INTO project_user_properties(id,workspace_id,project_id,user_id,preferences,sort_order)
		 VALUES('0199a2b4-0000-7000-8000-000000000004','0199a2b4-0000-7000-8000-000000000002','0199a2b4-0000-7000-8000-000000000003','0199a2b4-0000-7000-8000-000000000001','{"navigation":{"default_tab":"views","hide_in_more_menu":["cycles"]}}',7.5)`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	legacy := func() string {
		t.Helper()
		var row string
		if err := pool.QueryRow(ctx, `SELECT (to_jsonb(p)-array['filters','display_filters','display_properties'])::text FROM project_user_properties p`).Scan(&row); err != nil {
			t.Fatal(err)
		}
		return row
	}
	before := legacy()
	m, err = postgres.NewMigrator(pool, throughDisplay)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	defaults := map[string]string{"filters": "{}", "display_filters": displayFiltersDefault, "display_properties": displayPropertiesDefault}
	for pass := range 2 {
		up, err := m.Up(ctx)
		if err != nil || len(up) != 1 || up[0].Version != 26 || up[0].Source != "00026_project_project_user_properties_issue_display.sql" {
			t.Fatalf("Up %d = %v, %v; want only 26", pass, up, err)
		}
		if got := legacy(); got != before {
			t.Fatalf("legacy row changed: %s, want %s", got, before)
		}
		for col, want := range defaults {
			var equal bool
			if err := pool.QueryRow(ctx, "SELECT "+col+" = $1::jsonb FROM project_user_properties", want).Scan(&equal); err != nil || !equal {
				t.Fatalf("%s default: %v, %v", col, equal, err)
			}
		}
		if pass == 0 {
			down, err := m.Down(ctx)
			if err != nil || down == nil || down.Version != 26 {
				t.Fatalf("Down = %v, %v", down, err)
			}
			if got := legacy(); got != before {
				t.Fatalf("Down lost legacy row: %s", got)
			}
			var n int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_name='project_user_properties' AND column_name=ANY($1)`, []string{"filters", "display_filters", "display_properties"}).Scan(&n); err != nil || n != 0 {
				t.Fatalf("Down columns: %d, %v", n, err)
			}
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM pg_constraint WHERE conrelid='project_user_properties'::regclass AND conname=ANY($1)`, []string{"project_user_properties_filters_check", "project_user_properties_display_filters_check", "project_user_properties_display_properties_check"}).Scan(&n); err != nil || n != 0 {
				t.Fatalf("Down constraints: %d, %v", n, err)
			}
		}
	}
	for col, good := range defaults {
		t.Run(col, func(t *testing.T) {
			bad := []string{`[]`, `"text"`, `4`, `true`, `null`}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal([]byte(good), &fields); err != nil {
				t.Fatal(err)
			}
			if col == "filters" {
				bad = append(bad, `{"archived":[]}`, `{"project_id":[]}`, `{"sub_issue":[]}`)
				for _, key := range strings.Fields("state_id state_group priority label_id assignee_id created_by_id mention_id subscriber_id start_date target_date") {
					for _, value := range []string{`"x"`, `{}`, `null`, `1`, `false`, `[1]`, `[false]`, `[null]`, `[{}]`, `[["x"]]`, `["x",["y"]]`} {
						bad = append(bad, `{"`+key+`":`+value+`}`)
					}
					for _, value := range []string{`[]`, `["future"]`, `["a","b"]`} {
						if _, err := pool.Exec(ctx, "UPDATE project_user_properties SET filters=$1::jsonb", `{"`+key+`":`+value+`}`); err != nil {
							t.Fatal(err)
						}
					}
				}
			} else {
				for key, original := range fields {
					delete(fields, key)
					bad = append(bad, displayJSON(t, fields))
					for _, value := range []string{`[]`, `{}`, `1`, `"x"`, `false`, `null`} {
						// A string/null group, string text, and boolean flag are valid shapes.
						if (key == "group_by" || key == "sub_group_by") && (value == `"x"` || value == `null`) {
							continue
						}
						if (key == "layout" || key == "order_by") && value == `"x"` {
							continue
						}
						if string(original) == "true" && value == "false" {
							continue
						}
						fields[key] = json.RawMessage(value)
						bad = append(bad, displayJSON(t, fields))
					}
					fields[key] = original
				}
				fields["future"] = json.RawMessage(`true`)
				bad = append(bad, displayJSON(t, fields))
				delete(fields, "future")
				if col == "display_filters" {
					for _, value := range []string{`[]`, `{}`, `1`, `true`, `null`} {
						fields["calendar"] = json.RawMessage(`{"layout":` + value + `,"show_weekends":false}`)
						bad = append(bad, displayJSON(t, fields))
					}
					for _, value := range []string{`[]`, `{}`, `1`, `"false"`, `null`} {
						fields["calendar"] = json.RawMessage(`{"layout":"month","show_weekends":` + value + `}`)
						bad = append(bad, displayJSON(t, fields))
					}
					for _, calendar := range []string{`{}`, `{"layout":"month"}`, `{"show_weekends":false}`, `{"layout":"month","show_weekends":false,"x":true}`, `{"layout":null,"show_weekends":false}`, `{"layout":"month","show_weekends":null}`, `{"layout":[],"show_weekends":false}`, `{"layout":"month","show_weekends":1}`} {
						fields["calendar"] = json.RawMessage(calendar)
						bad = append(bad, displayJSON(t, fields))
					}
				}
			}
			for _, value := range bad {
				_, err := pool.Exec(ctx, "UPDATE project_user_properties SET "+col+"=$1::jsonb", value)
				var pg *pgconn.PgError
				if !errors.As(err, &pg) || pg.Code != "23514" || pg.ConstraintName != "project_user_properties_"+col+"_check" {
					t.Errorf("%s = %v; want named 23514", value, err)
				}
			}
			_, err := pool.Exec(ctx, "UPDATE project_user_properties SET "+col+"=NULL")
			var pg *pgconn.PgError
			if !errors.As(err, &pg) || pg.Code != "23502" || pg.ColumnName != col {
				t.Errorf("SQL NULL = %v; want 23502 on %s", err, col)
			}
			if _, err := pool.Exec(ctx, "UPDATE project_user_properties SET "+col+"=$1::jsonb", good); err != nil {
				t.Fatal(err)
			}
		})
	}
	// Valid shape intentionally permits words which the pure adapter refuses.
	if _, err := pool.Exec(ctx, `UPDATE project_user_properties SET display_filters=jsonb_set(jsonb_set(jsonb_set(jsonb_set(display_filters,'{group_by}','"cycle_id"'),'{layout}','"future"'),'{order_by}','"link_count"'),'{calendar,layout}','"day"'),filters='{"label_id":["bad-id"],"start_date":["bad-date"]}'`); err != nil {
		t.Fatal(err)
	}
}

func displayJSON(t *testing.T, value any) string {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
