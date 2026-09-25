package archtest

import (
	"slices"
	"testing"
)

// The spike's layout (M2 design 3.14): identity creates users; a later
// module, asset, creates assets; identity's own later migration alters
// users to reference assets; River's migration belongs to no entry.
func sqlcBase() ([]sqlcEntry, []migrationFile, []string) {
	entries := []sqlcEntry{
		{
			schema:  []string{"migrations/sql/00001_identity_users.sql", "migrations/sql/00021_identity_users_avatar_asset.sql"},
			queries: "internal/modules/identity/adapter/postgres/queries",
			out:     "internal/modules/identity/adapter/postgres/gen",
		},
		{
			schema:  []string{"migrations/sql/00020_asset_assets.sql"},
			queries: "internal/modules/asset/adapter/postgres/queries",
			out:     "internal/modules/asset/adapter/postgres/gen",
		},
	}
	migrations := []migrationFile{
		{"00001_identity_users.sql", "-- +goose Up\nCREATE TABLE users (id uuid PRIMARY KEY);\n-- +goose Down\nDROP TABLE users;\n"},
		{"00005_river_main_v2_to_v7.sql", "-- +goose Up\nCREATE TABLE river_job (id bigint);\nALTER TABLE river_job ADD COLUMN x int;\n"},
		{"00020_asset_assets.sql", "-- +goose Up\nCREATE TABLE IF NOT EXISTS assets (id uuid PRIMARY KEY);\n"},
		{"00021_identity_users_avatar_asset.sql", "-- +goose Up\n-- ALTER TABLE assets would be wrong; a comment is not SQL\n" +
			"ALTER TABLE users ADD COLUMN avatar_asset_id uuid REFERENCES assets ON DELETE SET NULL;\n" +
			"-- +goose Down\nALTER TABLE ONLY public.users DROP COLUMN avatar_asset_id;\n"},
	}
	return entries, migrations, []string{"identity", "asset"}
}

func TestSQLCScopeOfTheBaseLayoutPasses(t *testing.T) {
	if got := sqlcScopeViolations(sqlcBase()); len(got) != 0 {
		t.Errorf("violations = %q, want none", got)
	}
}

func TestSQLCScopeReportsViolations(t *testing.T) {
	tests := []struct {
		name   string
		change func(entries []sqlcEntry, migrations []migrationFile, modules []string) ([]sqlcEntry, []migrationFile, []string)
		want   string
	}{
		{"ALTER TABLE in a file of another module", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			m[3].name = "00021_asset_users_avatar.sql"
			e[0].schema = e[0].schema[:1]
			e[1].schema = append(e[1].schema, "migrations/sql/00021_asset_users_avatar.sql")
			return e, m, mods
		}, "migration 00021_asset_users_avatar.sql alters users, which module identity creates: the migration belongs to identity"},
		{"ALTER TABLE of an unknown table", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			m[0].sql = "CREATE TABLE people (id uuid);"
			return e, m, mods
		}, "migration 00021_identity_users_avatar_asset.sql alters users, which no migration creates"},
		{"a migration of another module", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			e[1].schema = append(e[1].schema, "migrations/sql/00001_identity_users.sql")
			return e, m, mods
		}, "sqlc entry asset lists migrations/sql/00001_identity_users.sql, a migration of identity"},
		{"River's migration", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			e[0].schema = append(e[0].schema, "migrations/sql/00005_river_main_v2_to_v7.sql")
			return e, m, mods
		}, "sqlc entry identity lists migrations/sql/00005_river_main_v2_to_v7.sql, a migration of river"},
		{"an own migration missing", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			e[0].schema = e[0].schema[:1]
			return e, m, mods
		}, "sqlc entry identity does not list its migration migrations/sql/00021_identity_users_avatar_asset.sql"},
		{"not a migration", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			e[1].schema = append(e[1].schema, "migrations/sql/schema.sql")
			return e, m, mods
		}, "sqlc entry asset lists migrations/sql/schema.sql, which is not a migration"},
		{"a module with queries but no entry", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			return e, m, append(mods, "workspace")
		}, "module workspace has adapter/postgres/queries but no sqlc entry"},
		{"queries outside a module's adapter", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			e[1].queries = "queries/asset"
			return e, m, mods[:1]
		}, "sqlc entry queries/asset: queries must be internal/modules/<module>/adapter/postgres/queries"},
		{"out outside the module's adapter", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			e[1].out = "internal/db"
			return e, m, mods
		}, "sqlc entry asset: out is internal/db, want internal/modules/asset/adapter/postgres/gen"},
		{"a misnamed migration", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			m = append(m, migrationFile{"00030-assets.sql", ""})
			return e, m, mods
		}, "migration 00030-assets.sql is not named <version>_<module>_<content>.sql"},
		{"a table created twice", func(e []sqlcEntry, m []migrationFile, mods []string) ([]sqlcEntry, []migrationFile, []string) {
			m[2].sql += "CREATE TABLE users (id uuid);"
			return e, m, mods
		}, "tables: users is created by both identity and asset"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sqlcScopeViolations(tt.change(sqlcBase()))
			if !slices.Equal(got, []string{tt.want}) {
				t.Errorf("violations = %q, want %q", got, tt.want)
			}
		})
	}
}
