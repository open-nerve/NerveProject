package bootstrap

import (
	"errors"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Removing the extension migration, owner CREATE or runtime DML/EXECUTE
// must prevent the deployed roles from using search and drafts.
func TestSearchTrigramRuntimeRole(t *testing.T) {
	roles := newSplitRoles(t) // Up runs on the owner connection, never the admin.
	ctx := pgtest.Soon(t)
	var superuser, create bool
	if err := roles.owner.QueryRow(ctx, `SELECT rolsuper, has_database_privilege(current_user, current_database(), 'CREATE') FROM pg_roles WHERE rolname=current_user`).Scan(&superuser, &create); err != nil || superuser || !create {
		t.Fatalf("owner: superuser=%v database CREATE=%v error=%v", superuser, create, err)
	}
	var installed bool
	if err := roles.owner.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname='pg_trgm')`).Scan(&installed); err != nil || !installed {
		t.Fatalf("owner Up did not install pg_trgm: %v, %v", installed, err)
	}
	t.Logf("owner Up succeeded: superuser=%v database CREATE=%v pg_trgm=%v", superuser, create, installed)
	user, workspace, project, state, issue, draft := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for _, fixture := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO users(id,email,password,display_name) VALUES($1,'search@example.com','x','Search')`, []any{user}},
		{`INSERT INTO workspaces(id,name,slug) VALUES($1,'Search','search')`, []any{workspace}},
		{`INSERT INTO projects(id,workspace_id,name,identifier) VALUES($1,$2,'Search','SRCH')`, []any{project, workspace}},
		{`INSERT INTO states(id,workspace_id,project_id,name,color) VALUES($1,$2,$3,'Backlog','#60646C')`, []any{state, workspace, project}},
		{`INSERT INTO issues(id,workspace_id,project_id,state_id,name,sequence_id) VALUES($1,$2,$3,$4,'Work',1)`, []any{issue, workspace, project, state}},
	} {
		if _, err := roles.owner.Exec(ctx, fixture.sql, fixture.args...); err != nil {
			t.Fatal(err)
		}
	}
	var similarity float32
	if err := roles.server.QueryRow(ctx, `SELECT similarity(name,'Work') FROM issues WHERE id=$1`, issue).Scan(&similarity); err != nil || similarity != 1 {
		t.Fatalf("runtime similarity over issue = %v, %v; want 1", similarity, err)
	}
	rows, err := gen.New(roles.server).SearchIssues(ctx, gen.SearchIssuesParams{Query: "Work", Pattern: "%Work%", ProjectIds: []uuid.UUID{project}, ProjectOwn: []bool{false}, PageSize: 1})
	if err != nil || len(rows) != 1 || rows[0].ID != issue || rows[0].Similarity != 1 {
		t.Fatalf("runtime full static search = %+v, %v", rows, err)
	}
	if _, err := roles.server.Exec(ctx, `INSERT INTO draft_issues(id,workspace_id,created_by_id) VALUES($1,$2,$3)`, draft, workspace, user); err != nil {
		t.Fatal("runtime draft INSERT:", err)
	}
	if _, err := roles.server.Exec(ctx, `UPDATE draft_issues SET payload='{"name":"Search"}' WHERE id=$1`, draft); err != nil {
		t.Fatal("runtime draft UPDATE:", err)
	}
	var payload string
	if err := roles.server.QueryRow(ctx, `SELECT payload->>'name' FROM draft_issues WHERE id=$1`, draft).Scan(&payload); err != nil || payload != "Search" {
		t.Fatalf("runtime draft SELECT = %q, %v", payload, err)
	}
	if tag, err := roles.server.Exec(ctx, `DELETE FROM draft_issues WHERE id=$1`, draft); err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("runtime draft DELETE = %v, %v", tag, err)
	}
	for _, ddl := range []string{`CREATE TABLE public.search_forbidden(id int)`, `CREATE INDEX search_forbidden_idx ON issues(name)`} {
		_, err := roles.server.Exec(ctx, ddl)
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "42501" {
			t.Fatalf("runtime DDL %s = %v; want insufficient_privilege", ddl, err)
		}
		t.Logf("runtime DDL refused with SQLSTATE %s: %s", pe.Code, ddl)
	}
	t.Logf("runtime similarity=%v; issue SELECT and draft INSERT/UPDATE/SELECT/DELETE succeeded", similarity)
}
