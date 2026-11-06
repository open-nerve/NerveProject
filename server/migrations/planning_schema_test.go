package migrations_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// The planning module's cycles table (M6 P1 spec 2.2): its columns, its indexes, the date CHECK (both dates or
// neither, the start not after the end), the non-empty name, and where each foreign key goes, the owner's SET NULL
// keeping the cycle when its owner's account is purged.
func TestCyclesSchema(t *testing.T) {
	pool := newIssueDatabase(t)
	ctx := context.Background()
	got := names(t, pool, `SELECT column_name||' '||data_type||' '||is_nullable||' '||coalesce(column_default,'-') FROM information_schema.columns
 WHERE table_schema='public' AND table_name='cycles' ORDER BY ordinal_position`)
	want := []string{
		"id uuid NO -", "workspace_id uuid NO -", "project_id uuid NO -", "name character varying NO -",
		"description text NO ''::text", "start_date date YES -", "end_date date YES -", "owned_by_id uuid YES -",
		"sort_order double precision NO 65535", "archived_at timestamp with time zone YES -",
		"created_by_id uuid YES -", "updated_by_id uuid YES -", "created_at timestamp with time zone NO now()",
		"updated_at timestamp with time zone NO now()", "deleted_at timestamp with time zone YES -",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("cycles columns\n%q\nwant\n%q", got, want)
	}
	indexes := names(t, pool, `SELECT indexdef FROM pg_indexes WHERE schemaname='public' AND tablename='cycles' ORDER BY indexname`)
	wantIndexes := []string{
		"CREATE INDEX cycles_deleted_at_idx ON public.cycles USING btree (deleted_at) WHERE (deleted_at IS NOT NULL)",
		"CREATE UNIQUE INDEX cycles_pkey ON public.cycles USING btree (id)",
		"CREATE INDEX cycles_project_id_idx ON public.cycles USING btree (project_id)",
		"CREATE INDEX cycles_project_list_idx ON public.cycles USING btree (project_id, sort_order, created_at, id) WHERE (deleted_at IS NULL)",
		"CREATE INDEX cycles_workspace_id_idx ON public.cycles USING btree (workspace_id)",
	}
	if !slices.Equal(indexes, wantIndexes) {
		t.Errorf("cycles indexes\n%q\nwant\n%q", indexes, wantIndexes)
	}

	const owner = "'0199a2b4-0000-7000-8000-0000000000c1'"
	const cycle = "'0199a2b4-0000-7000-8000-0000000000c2'"
	for _, stmt := range []string{
		"INSERT INTO users (id, email, password, display_name) VALUES (" + owner + ", 'owner@corp.com', 'x', 'owner')",
		"INSERT INTO cycles (id, workspace_id, project_id, name, owned_by_id, created_by_id) VALUES (" + cycle + ", " + issueAcme + ", " +
			issueWeb + ", 'Sprint', " + owner + ", " + owner + ")",
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	// Accepted: no dates, one day, a range.
	for _, dates := range []string{"start_date=NULL,end_date=NULL", "start_date='2026-11-07',end_date='2026-11-07'",
		"start_date='2026-11-07',end_date='2026-11-20'"} {
		if _, err := pool.Exec(ctx, "UPDATE cycles SET "+dates+" WHERE id="+cycle); err != nil {
			t.Errorf("dates %s refused: %v", dates, err)
		}
	}
	for _, tt := range []struct{ update, constraint string }{
		{"start_date='2026-11-07',end_date=NULL", "cycles_dates_check"},
		{"start_date=NULL,end_date='2026-11-07'", "cycles_dates_check"},
		{"start_date='2026-11-08',end_date='2026-11-07'", "cycles_dates_check"},
		{"name=''", "cycles_name_check"},
	} {
		_, err := pool.Exec(ctx, "UPDATE cycles SET "+tt.update+" WHERE id="+cycle)
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "23514" || pe.ConstraintName != tt.constraint {
			t.Errorf("%s accepted or wrong error: %v; want %s", tt.update, err, tt.constraint)
		}
	}
	for _, column := range []string{"workspace_id", "project_id", "name", "description", "sort_order", "created_at", "updated_at"} {
		_, err := pool.Exec(ctx, "UPDATE cycles SET "+column+"=NULL WHERE id="+cycle)
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "23502" || pe.ColumnName != column {
			t.Errorf("nullable %s: %v", column, err)
		}
	}

	// Purging the owner's account keeps the cycle, without its owner and creator.
	if _, err := pool.Exec(ctx, "DELETE FROM users WHERE id="+owner); err != nil {
		t.Fatal(err)
	}
	var kept bool
	if err := pool.QueryRow(ctx, "SELECT owned_by_id IS NULL AND created_by_id IS NULL FROM cycles WHERE id="+cycle).Scan(&kept); err != nil || !kept {
		t.Fatalf("cycle after its owner's purge: kept=%t, %v", kept, err)
	}
	// Purging the project takes its cycles; the other project's stay.
	if _, err := pool.Exec(ctx, "INSERT INTO cycles (id, workspace_id, project_id, name) VALUES (gen_random_uuid(), "+issueAcme+", "+issueOps+", 'Ops')"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM projects WHERE id="+issueWeb); err != nil {
		t.Fatal(err)
	}
	left := names(t, pool, "SELECT name FROM cycles ORDER BY name")
	if !slices.Equal(left, []string{"Ops"}) {
		t.Fatalf("cycles after web's purge = %q, want [Ops]", left)
	}
}

// planningNames are the planning module's constraint and index names with what each is (TestConstraintAndIndexNames).
var planningNames = []string{
	"cycles_created_by_id_fkey f n",
	"cycles_dates_check c",
	"cycles_deleted_at_idx iw",
	"cycles_name_check c",
	"cycles_owned_by_id_fkey f n",
	"cycles_pkey iu",
	"cycles_pkey p",
	"cycles_project_id_fkey f c",
	"cycles_project_id_idx i",
	"cycles_project_list_idx iw",
	"cycles_updated_by_id_fkey f n",
	"cycles_workspace_id_fkey f c",
	"cycles_workspace_id_idx i",
}
