package bootstrap

import (
	"fmt"
	"reflect"
	"testing"
)

func TestAssetScopeDeletionRollback(t *testing.T) {
	for _, scope := range []string{"issue", "comment", "draft", "project", "workspace"} {
		t.Run(scope, func(t *testing.T) {
			w := newIssueWorld(t)
			waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
			item := w.newWorkItem(t)
			purpose, fileScope := "ISSUE_ATTACHMENT", assetScopeBody("issue", item)
			table, path, actor := "issues", "/api/v0/issues/"+item.String(), "alice"
			switch scope {
			case "comment":
				comment := w.newComment(t, "bob", item)
				purpose, fileScope = "COMMENT_ATTACHMENT", assetScopeBody("comment", comment.ID)
				table, path = "issue_comments", commentPath(comment.ID)
			case "draft":
				d := createDraftHTTP(t, w, "bob", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Rollback"}}`, w.web))
				purpose, fileScope = "DRAFT_ISSUE_ATTACHMENT", assetScopeBody("draft", d.ID)
				table, path, actor = "draft_issues", draftPath(d.ID), "bob"
			case "project":
				table, path = "projects", "/api/v0/projects/"+w.web.String()
			case "workspace":
				table, path = "workspaces", "/api/v0/workspaces/acme"
			}
			file := uploadedAssetHTTP(t, w, "bob", purpose, fileScope, "rollback.txt", "text/plain", []byte("rollback"))
			// The sequence proves the actual COMMIT reached the failure after
			// the real synchronous mark / transactional enqueue. Unlike rows,
			// sequence advancement survives the intentionally aborted tx.
			statement := fmt.Sprintf(`CREATE SEQUENCE asset_rollback_observed;
CREATE FUNCTION reject_asset_scope_commit() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_TABLE_NAME IN ('projects','workspaces') THEN
  IF NOT EXISTS(SELECT 1 FROM river_job WHERE kind='asset.delete_scope' AND args->>'id'=NEW.id::text) THEN
   RAISE EXCEPTION 'scope job absent before commit';
  END IF;
 ELSE
  IF NOT EXISTS(SELECT 1 FROM file_assets WHERE id='%s' AND deleted_at IS NOT NULL) THEN
   RAISE EXCEPTION 'asset mark absent before commit';
  END IF;
 END IF;
 PERFORM nextval('asset_rollback_observed');
 RAISE EXCEPTION 'intentional scope commit fault';
END $$;
CREATE CONSTRAINT TRIGGER reject_asset_scope_commit AFTER UPDATE ON %s
DEFERRABLE INITIALLY DEFERRED FOR EACH ROW
WHEN (OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL)
EXECUTE FUNCTION reject_asset_scope_commit();`, file.Asset.ID, table)
			if _, err := w.pool.Exec(t.Context(), statement); err != nil {
				t.Fatal(err)
			}
			before := assetScopeRollbackState(t, w)
			body := w.send(t, actor, "DELETE", path, "", 500)
			if problemCode(t, []byte(body)) != "internal_error" {
				t.Fatal(body)
			}
			var observed bool
			if err := w.pool.QueryRow(t.Context(), "SELECT is_called FROM asset_rollback_observed").Scan(&observed); err != nil || !observed {
				t.Fatalf("did not reach commit after asset event: %t %v", observed, err)
			}
			if after := assetScopeRollbackState(t, w); !reflect.DeepEqual(before, after) {
				t.Fatalf("parent / asset / job escaped rollback: before=%v after=%v", before, after)
			}
			_, data := readAssetHTTP(t, w, "bob", file.Asset.ID)
			if string(data) != "rollback" {
				t.Fatalf("rollback lost original bytes: %s", data)
			}
		})
	}
}

func assetScopeRollbackState(t *testing.T, w issueWorld) map[string]string {
	t.Helper()
	rows := map[string]string{}
	for _, table := range []string{"issues", "issue_comments", "draft_issues", "projects", "workspaces", "project_members", "workspace_members", "file_assets", "river_job"} {
		where := ""
		if table == "river_job" {
			where = " WHERE kind IN ('asset.delete_scope','issue.delete_scope')"
		}
		var value string
		if err := w.pool.QueryRow(t.Context(), "SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id),'[]'::jsonb)::text FROM "+table+" r"+where).Scan(&value); err != nil {
			t.Fatal(err)
		}
		rows[table] = value
	}
	return rows
}
