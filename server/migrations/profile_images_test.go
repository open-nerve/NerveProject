package migrations_test

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

var profileImageNames = []string{
	"users_avatar_asset_fk f a", "users_cover_image_asset_fk f a",
	"users_avatar_asset_id_idx i", "users_cover_image_asset_id_idx i",
	"workspaces_logo_asset_fk f a", "workspaces_logo_asset_id_idx i",
	"projects_cover_image_asset_fk f a", "projects_cover_image_asset_id_idx i",
	"projects_cover_builtin_id_check c", "projects_cover_choice_check c",
}

// Catches omitted owner columns, destructive upgrades and deferred/automatic
// reference removal. These are schema fixtures, not successful API bindings.
func TestProfileImageColumns(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t, pgtest.NewEmptyDatabase(t))
	old := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.FS(), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() > "00034_z" {
			continue
		}
		data, err := fs.ReadFile(migrations.FS(), entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		old[entry.Name()] = &fstest.MapFile{Data: data}
	}
	m, err := postgres.NewMigrator(pool, old)
	if err != nil {
		t.Fatal(err)
	}
	_, upErr := m.Up(ctx)
	closeErr := m.Close()
	if upErr != nil || closeErr != nil {
		t.Fatalf("old schema: %v / %v", upErr, closeErr)
	}
	for _, sql := range []string{
		"INSERT INTO users(id,email,password,display_name) VALUES(" + issueBob + ",'image@example.com','x','Image')",
		"INSERT INTO workspaces(id,name,slug) VALUES(" + issueAcme + ",'Images','images')",
		"INSERT INTO projects(id,workspace_id,name,identifier) VALUES(" + issueWeb + "," + issueAcme + ",'Images','IMG')",
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	snapshot := func(table string) string {
		t.Helper()
		var row string
		query := "SELECT (to_jsonb(r)-ARRAY['avatar_asset_id','cover_image_asset_id','logo_asset_id','cover_builtin_id'])::text FROM " + table + " r"
		if err := pool.QueryRow(ctx, query).Scan(&row); err != nil {
			t.Fatal(err)
		}
		return row
	}
	before := map[string]string{}
	for _, table := range []string{"users", "workspaces", "projects"} {
		before[table] = snapshot(table)
	}
	m, err = postgres.NewMigrator(pool, migrations.FS())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	if _, err = m.Up(ctx); err != nil {
		t.Fatal(err)
	}
	for table, want := range before {
		if got := snapshot(table); got != want {
			t.Errorf("upgrade changed %s: %s want %s", table, got, want)
		}
	}
	columns := names(t, pool, `SELECT table_name||'.'||column_name||' '||data_type||' '||is_nullable
 FROM information_schema.columns WHERE table_schema='public' AND
 ((table_name='users' AND column_name IN ('avatar_asset_id','cover_image_asset_id')) OR
 (table_name='workspaces' AND column_name='logo_asset_id') OR
 (table_name='projects' AND column_name IN ('cover_image_asset_id','cover_builtin_id'))) ORDER BY 1`)
	want := []string{"projects.cover_builtin_id text YES", "projects.cover_image_asset_id uuid YES", "users.avatar_asset_id uuid YES", "users.cover_image_asset_id uuid YES", "workspaces.logo_asset_id uuid YES"}
	if !slices.Equal(columns, want) {
		t.Fatalf("image columns=%v want=%v", columns, want)
	}
	for _, ref := range []struct{ table, column, constraint string }{
		{"users", "avatar_asset_id", "users_avatar_asset_fk"},
		{"users", "cover_image_asset_id", "users_cover_image_asset_fk"},
		{"workspaces", "logo_asset_id", "workspaces_logo_asset_fk"},
		{"projects", "cover_image_asset_id", "projects_cover_image_asset_fk"},
	} {
		t.Run(ref.table+"."+ref.column, func(t *testing.T) {
			var unset bool
			if err := pool.QueryRow(ctx, "SELECT "+ref.column+" IS NULL FROM "+ref.table).Scan(&unset); err != nil || !unset {
				t.Fatalf("legacy default null=%t: %v", unset, err)
			}
			var action string
			var deferred, deferrable, indexed bool
			err := pool.QueryRow(ctx, `SELECT c.confdeltype::text,c.condeferred,c.condeferrable,
 EXISTS(SELECT 1 FROM pg_index i WHERE i.indrelid=c.conrelid AND i.indisvalid AND i.indpred IS NULL AND i.indexprs IS NULL AND i.indkey[0]=c.conkey[1])
 FROM pg_constraint c JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=c.conkey[1]
 JOIN pg_attribute target ON target.attrelid=c.confrelid AND target.attnum=c.confkey[1]
 WHERE c.conrelid=$1::regclass AND c.conname=$2 AND c.confrelid='file_assets'::regclass
 AND cardinality(c.conkey)=1 AND cardinality(c.confkey)=1 AND a.attname=$3 AND target.attname='id'`, ref.table, ref.constraint, ref.column).Scan(&action, &deferred, &deferrable, &indexed)
			if err != nil || action != "a" || deferred || deferrable || !indexed {
				t.Fatalf("FK %s: %s %t %t %t %v", ref.constraint, action, deferred, deferrable, indexed, err)
			}
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			_, err = tx.Exec(ctx, "UPDATE "+ref.table+" SET "+ref.column+"=gen_random_uuid()")
			assertImageConstraint(t, err, "23503", ref.constraint)
			if err := tx.Rollback(ctx); err != nil {
				t.Fatal(err)
			}
			id := schemaImage(t, pool)
			if _, err := pool.Exec(ctx, "UPDATE "+ref.table+" SET "+ref.column+"=$1", id); err != nil {
				t.Fatal(err)
			}
			_, err = pool.Exec(ctx, "DELETE FROM file_assets WHERE id=$1", id)
			assertImageConstraint(t, err, "23503", ref.constraint)
			var kept bool
			if err := pool.QueryRow(ctx, "SELECT "+ref.column+"=$1 FROM "+ref.table, id).Scan(&kept); err != nil || !kept {
				t.Fatalf("reference lost: %t %v", kept, err)
			}
			if _, err := pool.Exec(ctx, "UPDATE "+ref.table+" SET "+ref.column+"=NULL"); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, "DELETE FROM file_assets WHERE id=$1", id); err != nil {
				t.Fatal(err)
			}
		})
	}
	var builtinNull bool
	if err := pool.QueryRow(ctx, "SELECT cover_builtin_id IS NULL FROM projects").Scan(&builtinNull); err != nil || !builtinNull {
		t.Fatalf("builtin default: %t %v", builtinNull, err)
	}
}

func TestProjectCoverChoice(t *testing.T) {
	pool := newIssueDatabase(t)
	ctx := context.Background()
	for _, id := range []string{"cover-01", "cover-15", "cover-29"} {
		if _, err := pool.Exec(ctx, "UPDATE projects SET cover_builtin_id=$1 WHERE id="+issueWeb, id); err != nil {
			t.Fatalf("valid %s: %v", id, err)
		}
	}
	for _, id := range []string{"", "cover-00", "cover-30", "cover-1", "cover-001", "COVER-01", "cover-01\n"} {
		_, err := pool.Exec(ctx, "UPDATE projects SET cover_builtin_id=$1 WHERE id="+issueWeb, id)
		assertImageConstraint(t, err, "23514", "projects_cover_builtin_id_check")
	}
	asset := schemaImage(t, pool)
	_, err := pool.Exec(ctx, "UPDATE projects SET cover_image_asset_id=$1 WHERE id="+issueWeb, asset)
	assertImageConstraint(t, err, "23514", "projects_cover_choice_check")
	if _, err := pool.Exec(ctx, "UPDATE projects SET cover_builtin_id=NULL,cover_image_asset_id=$1 WHERE id="+issueWeb, asset); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "UPDATE projects SET cover_image_asset_id=NULL WHERE id="+issueWeb); err != nil {
		t.Fatal(err)
	}
}

func schemaImage(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `INSERT INTO file_assets(id,user_id,entity_type,scope_kind,size,attributes,storage_metadata)
 VALUES(gen_random_uuid(),`+issueBob+`,'USER_AVATAR','user',1,'{"name":"avatar.png","content_type":"image/png"}',
 '{"schema_version":1,"origin":"copy","staging":null,"final":null,"candidates":[]}') RETURNING id::text`).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	return id
}
func assertImageConstraint(t *testing.T, err error, code, name string) {
	t.Helper()
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != code || pg.ConstraintName != name {
		t.Fatalf("want %s/%s, got %v", code, name, err)
	}
}
