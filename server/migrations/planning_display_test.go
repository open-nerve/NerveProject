package migrations_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

const planningDisplayDefault = `{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"attachment_count":true,"created_on":true,"updated_on":true,"cycle":true,"modules":true}`

// The two rows of TestPlanningDisplayMigration: one stored before 00045,
// one that saves the new keys after it.
const (
	planningDisplayOld   = "0199a2b4-0000-7000-8000-000000000025"
	planningDisplaySaved = "0199a2b4-0000-7000-8000-000000000026"
)

// Cycles and modules join the project's issue display properties and
// filters (M6/P3b spec R4): a stored choice keeps its keys and shows both,
// as a new row does. Down strips exactly the new keys, from the rows that
// saved them too, and the old shapes hold again; Up works once more.
func TestPlanningDisplayMigration(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t, pgtest.NewEmptyDatabase(t))
	before, through := fstest.MapFS{}, fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.FS(), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() > "00045_z" {
			continue
		}
		data, err := fs.ReadFile(migrations.FS(), e.Name())
		if err != nil {
			t.Fatal(err)
		}
		through[e.Name()] = &fstest.MapFile{Data: data}
		if e.Name() <= "00044_z" {
			before[e.Name()] = through[e.Name()]
		}
	}
	m, err := postgres.NewMigrator(pool, before)
	if err != nil {
		t.Fatal(err)
	}
	if up, err := m.Up(ctx); err != nil || len(up) != 44 {
		t.Fatalf("old Up: %v, %v", up, err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	custom := `{"assignee":false,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":false,"attachment_count":false,"created_on":true,"updated_on":true}`
	oldFilters := `{"label_id":["none"],"priority":[]}`
	for _, sql := range []string{
		`INSERT INTO users(id,email,password,display_name) VALUES('0199a2b4-0000-7000-8000-000000000021','planning-a@example.com','x','planning a'),('0199a2b4-0000-7000-8000-000000000022','planning-b@example.com','x','planning b')`,
		`INSERT INTO workspaces(id,name,slug) VALUES('0199a2b4-0000-7000-8000-000000000023','Planning','planning')`,
		`INSERT INTO projects(id,workspace_id,name,identifier) VALUES('0199a2b4-0000-7000-8000-000000000024','0199a2b4-0000-7000-8000-000000000023','Planning','PLN')`,
		`INSERT INTO project_user_properties(id,workspace_id,project_id,user_id,filters,display_properties)
		 VALUES('` + planningDisplayOld + `','0199a2b4-0000-7000-8000-000000000023','0199a2b4-0000-7000-8000-000000000024','0199a2b4-0000-7000-8000-000000000021','` + oldFilters + `','` + custom + `')`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	m, err = postgres.NewMigrator(pool, through)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	shown := `{"assignee":false,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":false,"attachment_count":false,"created_on":true,"updated_on":true,"cycle":true,"modules":true}`
	hidden := `{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"attachment_count":true,"created_on":true,"updated_on":true,"cycle":false,"modules":false}`
	savedFilters := `{"cycle_id":["none"],"module_id":["0199a2b4-0000-7000-8000-0000000000ff","none"],"label_id":[]}`
	for pass := range 2 {
		up, err := m.Up(ctx)
		if err != nil || len(up) != 1 || up[0].Version != 45 || up[0].Source != "00045_project_project_user_properties_planning_display.sql" {
			t.Fatalf("Up %d = %v, %v; want only 45", pass, up, err)
		}
		storedPlanningDisplay(t, pool, planningDisplayOld, "display_properties", shown)
		storedPlanningDisplay(t, pool, planningDisplayOld, "filters", oldFilters)
		planningDisplayDefaulted(t, pool, planningDisplayDefault)
		if pass == 1 {
			// Down lost the choice to hide them: they show again, as on any old row.
			storedPlanningDisplay(t, pool, planningDisplaySaved, "display_properties", planningDisplayDefault)
			storedPlanningDisplay(t, pool, planningDisplaySaved, "filters", `{"label_id":[]}`)
			break
		}
		// A new row takes the new default; it then saves both new filter
		// keys and hides both new properties.
		if _, err := pool.Exec(ctx, `INSERT INTO project_user_properties(id,workspace_id,project_id,user_id)
		 VALUES('`+planningDisplaySaved+`','0199a2b4-0000-7000-8000-000000000023','0199a2b4-0000-7000-8000-000000000024','0199a2b4-0000-7000-8000-000000000022')`); err != nil {
			t.Fatal(err)
		}
		storedPlanningDisplay(t, pool, planningDisplaySaved, "display_properties", planningDisplayDefault)
		if _, err := pool.Exec(ctx, "UPDATE project_user_properties SET filters=$2::jsonb,display_properties=$3::jsonb WHERE id=$1", planningDisplaySaved, savedFilters, hidden); err != nil {
			t.Fatal(err)
		}
		refusedPlanningDisplay(t, pool, "display_properties", custom)
		refusedPlanningDisplay(t, pool, "display_properties", `{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"attachment_count":true,"created_on":true,"updated_on":true,"cycle":true}`)
		refusedPlanningDisplay(t, pool, "display_properties", `{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"attachment_count":true,"created_on":true,"updated_on":true,"cycle":"yes","modules":true}`)
		refusedPlanningDisplay(t, pool, "display_properties", `{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"attachment_count":true,"created_on":true,"updated_on":true,"cycle":true,"modules":null}`)
		refusedPlanningDisplay(t, pool, "display_properties", `{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"attachment_count":true,"created_on":true,"updated_on":true,"cycle":true,"modules":true,"future":true}`)
		for _, value := range []string{`{"cycle_ids":[]}`, `{"module_ids":[]}`, `{"project_id":[]}`, `{"cycle_id":"none"}`, `{"cycle_id":[1]}`, `{"module_id":null}`, `{"module_id":[["x"]]}`} {
			refusedPlanningDisplay(t, pool, "filters", value)
		}
		down, err := m.Down(ctx)
		if err != nil || down == nil || down.Version != 45 {
			t.Fatalf("Down = %v, %v", down, err)
		}
		storedPlanningDisplay(t, pool, planningDisplayOld, "display_properties", custom)
		storedPlanningDisplay(t, pool, planningDisplayOld, "filters", oldFilters)
		storedPlanningDisplay(t, pool, planningDisplaySaved, "display_properties", attachmentDisplayDefault)
		storedPlanningDisplay(t, pool, planningDisplaySaved, "filters", `{"label_id":[]}`)
		planningDisplayDefaulted(t, pool, attachmentDisplayDefault)
		refusedPlanningDisplay(t, pool, "display_properties", shown)
		for _, value := range []string{`{"cycle_id":[]}`, `{"module_id":["none"]}`} {
			refusedPlanningDisplay(t, pool, "filters", value)
		}
	}
}

// storedPlanningDisplay: row's column holds want, as jsonb.
func storedPlanningDisplay(t *testing.T, pool *pgxpool.Pool, row, column, want string) {
	t.Helper()
	var got string
	var equal bool
	if err := pool.QueryRow(context.Background(), "SELECT "+column+"::text, "+column+" = $2::jsonb FROM project_user_properties WHERE id=$1", row, want).Scan(&got, &equal); err != nil || !equal {
		t.Fatalf("row %s's %s = %s, %v; want %s", row, column, got, err, want)
	}
}

// planningDisplayDefaulted: display_properties' default is want, as jsonb.
func planningDisplayDefaulted(t *testing.T, pool *pgxpool.Pool, want string) {
	t.Helper()
	var expr string
	if err := pool.QueryRow(context.Background(), `SELECT column_default FROM information_schema.columns WHERE table_name='project_user_properties' AND column_name='display_properties'`).Scan(&expr); err != nil {
		t.Fatal(err)
	}
	var equal bool
	if err := pool.QueryRow(context.Background(), "SELECT ("+expr+") = $1::jsonb", want).Scan(&equal); err != nil || !equal {
		t.Fatalf("default %s = %v, %v; want %s", expr, equal, err, want)
	}
}

// refusedPlanningDisplay: storing value in column fails the column's
// named CHECK.
func refusedPlanningDisplay(t *testing.T, pool *pgxpool.Pool, column, value string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), "UPDATE project_user_properties SET "+column+"=$1::jsonb", value)
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "23514" || pg.ConstraintName != "project_user_properties_"+column+"_check" {
		t.Errorf("%s %s = %v; want named 23514", column, value, err)
	}
}
