package migrations_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestAssetSchemaConstraints(t *testing.T) {
	pool := newIssueDatabase(t)
	ctx := context.Background()
	columns := names(t, pool, `SELECT column_name FROM information_schema.columns WHERE table_schema='public' AND table_name='file_assets' ORDER BY column_name`)
	want := []string{"attributes", "bound_at", "comment_id", "created_at", "created_by_id", "deleted_at", "deleted_by_id", "draft_issue_id", "entity_type", "id", "is_deleted", "is_uploaded", "issue_id", "project_id", "scope_kind", "size", "storage_metadata", "storage_revision", "updated_at", "updated_by_id", "upload_expires_at", "user_id", "workspace_id"}
	if !slices.Equal(columns, want) {
		t.Fatalf("asset columns=%v want=%v", columns, want)
	}
	parents := map[string]string{"workspace": "workspaces", "project": "projects", "issue": "issues", "comment": "issue_comments", "draft_issue": "draft_issues", "user": "users"}
	for column, parent := range parents {
		var action string
		var deferred, deferrable, indexed bool
		err := pool.QueryRow(ctx, `SELECT c.confdeltype::text,c.condeferred,c.condeferrable,
 EXISTS(SELECT 1 FROM pg_index i WHERE i.indrelid=c.conrelid AND i.indisvalid AND i.indpred IS NULL AND i.indexprs IS NULL AND i.indkey[0]=c.conkey[1])
 FROM pg_constraint c JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=c.conkey[1]
 WHERE c.conrelid='file_assets'::regclass AND c.conname=$1 AND c.confrelid=$2::regclass AND cardinality(c.conkey)=1 AND a.attname=$3`, "file_assets_"+column+"_fk", parent, column+"_id").Scan(&action, &deferred, &deferrable, &indexed)
		if err != nil || action != "a" || deferred || deferrable || !indexed {
			t.Errorf("parent %s action=%s deferred=%t deferrable=%t indexed=%t: %v", column, action, deferred, deferrable, indexed, err)
		}
	}
	for _, column := range []string{"created_by_id", "updated_by_id", "deleted_by_id"} {
		got := names(t, pool, `SELECT confdeltype::text FROM pg_constraint WHERE conrelid='file_assets'::regclass AND conname='file_assets_`+column+`_fkey'`)
		if !slices.Equal(got, []string{"n"}) {
			t.Errorf("audit %s=%v", column, got)
		}
	}
	metadata := `{"schema_version":1,"origin":"copy","staging":null,"final":null,"candidates":[]}`
	insert := `INSERT INTO file_assets(id,user_id,entity_type,scope_kind,size,attributes,storage_metadata) VALUES(gen_random_uuid(),` + issueBob + `,'USER_AVATAR','user',1,'{"name":"头像.png","content_type":"image/png"}',$1) RETURNING id::text`
	var id string
	if err := pool.QueryRow(ctx, insert, metadata).Scan(&id); err != nil {
		t.Fatal(err)
	}
	bad := []string{`null`, `{}`, `[]`, `{"schema_version":null,"origin":"copy","staging":null,"final":null,"candidates":[]}`}
	var original map[string]json.RawMessage
	if err := json.Unmarshal([]byte(metadata), &original); err != nil {
		t.Fatal(err)
	}
	for key := range original {
		for _, remove := range []bool{false, true} {
			copy := make(map[string]json.RawMessage, len(original))
			for name, value := range original {
				copy[name] = value
			}
			if remove {
				delete(copy, key)
			} else {
				copy[key] = json.RawMessage(`null`)
				if key == "staging" || key == "final" {
					copy[key] = json.RawMessage(`false`)
				}
			}
			raw, err := json.Marshal(copy)
			if err != nil {
				t.Fatal(err)
			}
			bad = append(bad, string(raw))
		}
	}
	bad = append(bad, strings.Replace(metadata, `"schema_version":1`, `"schema_version":2`, 1), strings.Replace(metadata, `"candidates":[]`, `"candidates":[{},{},{},{}]`, 1), strings.Replace(metadata, `"candidates":[]`, `"candidates":{}`, 1), strings.Replace(metadata, `"copy"`, `"unknown"`, 1), strings.Replace(metadata, `"candidates":[]`, `"candidates":[],"unknown":true`, 1), strings.Replace(metadata, `"candidates":[]`, `"candidates":[{"padding":"`+strings.Repeat("x", 16384)+`"}]`, 1))
	for i, raw := range bad {
		_, err := pool.Exec(ctx, `UPDATE file_assets SET storage_metadata=$1 WHERE id=$2`, raw, id)
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "23514" || pe.ConstraintName != "file_assets_storage_metadata_check" {
			t.Errorf("invalid metadata %d accepted or wrong error: %v", i, err)
		}
	}
	for _, update := range []string{"size=0", "size=104857601", "storage_revision=-1", "is_deleted=true", "deleted_at=now()", "workspace_id=" + issueAcme, "scope_kind='issue'", "entity_type='PAGE_DESCRIPTION'", "attributes='null'", "attributes='{}'"} {
		_, err := pool.Exec(ctx, "UPDATE file_assets SET "+update+" WHERE id=$1", id)
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "23514" {
			t.Errorf("accepted invalid %s: %v", update, err)
		}
	}
}

// Exact catalog names and actions, independently enumerated from the migration.
var assetNames = []string{
	"file_assets_pkey p", "file_assets_pkey iu",
	"file_assets_user_fk f a", "file_assets_workspace_fk f a", "file_assets_project_fk f a",
	"file_assets_issue_fk f a", "file_assets_comment_fk f a", "file_assets_draft_issue_fk f a",
	"file_assets_created_by_id_fkey f n", "file_assets_updated_by_id_fkey f n", "file_assets_deleted_by_id_fkey f n",
	"file_assets_size_check c", "file_assets_storage_revision_check c", "file_assets_deletion_check c",
	"file_assets_attributes_check c", "file_assets_scope_check c", "file_assets_storage_metadata_check c",
	"file_assets_user_id_idx i", "file_assets_workspace_id_idx i", "file_assets_project_id_idx i",
	"file_assets_issue_id_idx i", "file_assets_comment_id_idx i", "file_assets_draft_issue_id_idx i",
	"file_assets_deleted_at_idx iw", "file_assets_created_at_id_idx iw",
}
