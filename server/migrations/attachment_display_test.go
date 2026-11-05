package migrations_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

const attachmentDisplayDefault = `{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"attachment_count":true,"created_on":true,"updated_on":true}`

// A stored choice keeps every key and shows the new attachment count, as a
// new row does; Down removes exactly that key and the eleven-key shape holds.
func TestAttachmentDisplayMigration(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t, pgtest.NewEmptyDatabase(t))
	before, through := fstest.MapFS{}, fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.FS(), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() > "00038_z" {
			continue
		}
		data, err := fs.ReadFile(migrations.FS(), e.Name())
		if err != nil {
			t.Fatal(err)
		}
		through[e.Name()] = &fstest.MapFile{Data: data}
		if e.Name() <= "00037_z" {
			before[e.Name()] = through[e.Name()]
		}
	}
	m, err := postgres.NewMigrator(pool, before)
	if err != nil {
		t.Fatal(err)
	}
	if up, err := m.Up(ctx); err != nil || len(up) != 37 {
		t.Fatalf("old Up: %v, %v", up, err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	custom := `{"assignee":false,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":false,"created_on":true,"updated_on":true}`
	for _, sql := range []string{
		`INSERT INTO users(id,email,password,display_name) VALUES('0199a2b4-0000-7000-8000-000000000011','attachments@example.com','x','attachments')`,
		`INSERT INTO workspaces(id,name,slug) VALUES('0199a2b4-0000-7000-8000-000000000012','Attachments','attachments')`,
		`INSERT INTO projects(id,workspace_id,name,identifier) VALUES('0199a2b4-0000-7000-8000-000000000013','0199a2b4-0000-7000-8000-000000000012','Attachments','ATT')`,
		`INSERT INTO project_user_properties(id,workspace_id,project_id,user_id,display_properties)
		 VALUES('0199a2b4-0000-7000-8000-000000000014','0199a2b4-0000-7000-8000-000000000012','0199a2b4-0000-7000-8000-000000000013','0199a2b4-0000-7000-8000-000000000011','` + custom + `')`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	stored := func(want string) {
		t.Helper()
		var equal bool
		if err := pool.QueryRow(ctx, `SELECT display_properties = $1::jsonb FROM project_user_properties`, want).Scan(&equal); err != nil || !equal {
			t.Fatalf("stored row = %v, %v; want %s", equal, err, want)
		}
	}
	defaulted := func(want string) {
		t.Helper()
		var expr string
		if err := pool.QueryRow(ctx, `SELECT column_default FROM information_schema.columns WHERE table_name='project_user_properties' AND column_name='display_properties'`).Scan(&expr); err != nil {
			t.Fatal(err)
		}
		var equal bool
		if err := pool.QueryRow(ctx, "SELECT ("+expr+") = $1::jsonb", want).Scan(&equal); err != nil || !equal {
			t.Fatalf("default %s = %v, %v; want %s", expr, equal, err, want)
		}
	}
	refused := func(value string) {
		t.Helper()
		_, err := pool.Exec(ctx, "UPDATE project_user_properties SET display_properties=$1::jsonb", value)
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != "23514" || pg.ConstraintName != "project_user_properties_display_properties_check" {
			t.Errorf("%s = %v; want named 23514", value, err)
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
	shown := `{"assignee":false,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":false,"attachment_count":true,"created_on":true,"updated_on":true}`
	for pass := range 2 {
		up, err := m.Up(ctx)
		if err != nil || len(up) != 1 || up[0].Version != 38 || up[0].Source != "00038_project_project_user_properties_attachment_display.sql" {
			t.Fatalf("Up %d = %v, %v; want only 38", pass, up, err)
		}
		stored(shown)
		defaulted(attachmentDisplayDefault)
		if pass == 0 {
			down, err := m.Down(ctx)
			if err != nil || down == nil || down.Version != 38 {
				t.Fatalf("Down = %v, %v", down, err)
			}
			stored(custom)
			defaulted(displayPropertiesDefault)
			refused(shown)
		}
	}
	for _, value := range []string{
		custom,
		`{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"attachment_count":"yes","created_on":true,"updated_on":true}`,
		`{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"attachment_count":null,"created_on":true,"updated_on":true}`,
		`{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"attachment_count":true,"created_on":true,"updated_on":true,"future":true}`,
	} {
		refused(value)
	}
	if _, err := pool.Exec(ctx, "UPDATE project_user_properties SET display_properties=$1::jsonb", attachmentDisplayDefault); err != nil {
		t.Fatal(err)
	}
}
