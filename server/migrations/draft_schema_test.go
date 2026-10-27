package migrations_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// Extra columns, nullable authors, non-object payloads and missing live ordering
// indexes each break the persistence contract independently of the API.
func TestDraftSchema(t *testing.T) {
	pool := newIssueDatabase(t)
	ctx := context.Background()
	got := names(t, pool, `SELECT column_name||' '||data_type||' '||is_nullable FROM information_schema.columns WHERE table_schema='public' AND table_name='draft_issues' ORDER BY ordinal_position`)
	want := []string{"id uuid NO", "workspace_id uuid NO", "project_id uuid YES", "payload jsonb NO", "created_by_id uuid NO", "updated_by_id uuid YES", "created_at timestamp with time zone NO", "updated_at timestamp with time zone NO", "deleted_at timestamp with time zone YES"}
	if !slices.Equal(got, want) {
		t.Fatalf("draft columns %v; want %v", got, want)
	}
	indexes := names(t, pool, `SELECT indexdef FROM pg_indexes WHERE schemaname='public' AND tablename='draft_issues' ORDER BY indexname`)
	wantIndexes := []string{
		"CREATE INDEX draft_issues_author_created_at_idx ON public.draft_issues USING btree (workspace_id, created_by_id, created_at DESC, id DESC) WHERE (deleted_at IS NULL)",
		"CREATE UNIQUE INDEX draft_issues_pkey ON public.draft_issues USING btree (id)",
		"CREATE INDEX draft_issues_project_id_idx ON public.draft_issues USING btree (project_id)",
		"CREATE INDEX draft_issues_workspace_id_idx ON public.draft_issues USING btree (workspace_id)",
	}
	if !slices.Equal(indexes, wantIndexes) {
		t.Errorf("draft indexes %v; want %v", indexes, wantIndexes)
	}
	insert := "INSERT INTO draft_issues(id,workspace_id,created_by_id) VALUES(gen_random_uuid()," + issueAcme + "," + issueBob + ")"
	var payload string
	var defaults bool
	if err := pool.QueryRow(ctx, insert+" RETURNING payload::text,project_id IS NULL AND updated_by_id IS NULL AND deleted_at IS NULL AND created_at=now() AND updated_at=now()").Scan(&payload, &defaults); err != nil || payload != "{}" || !defaults {
		t.Fatalf("defaults: %s, %t, %v", payload, defaults, err)
	}
	for _, value := range []string{`[]`, `null`, `"text"`, `1`, `true`} {
		_, err := pool.Exec(ctx, "UPDATE draft_issues SET payload=$1", value)
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "23514" || pe.ConstraintName != "draft_issues_payload_check" {
			t.Errorf("payload %s accepted: %v", value, err)
		}
	}
	for _, column := range []string{"workspace_id", "payload", "created_by_id", "created_at", "updated_at"} {
		_, err := pool.Exec(ctx, "UPDATE draft_issues SET "+column+"=NULL")
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "23502" || pe.ColumnName != column {
			t.Errorf("nullable %s: %v", column, err)
		}
	}
}
