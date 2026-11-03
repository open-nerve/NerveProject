package bootstrap

import (
	"testing"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
)

func attachmentActivityCount(t *testing.T, w issueWorld, item uuid.UUID) int {
	t.Helper()
	var count int
	if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field='attachment'", item).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestAttachmentEvents(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItemBy(t, "bob", w.web, `{"name":"Attachment activities"}`)
	p := prepareAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", attachmentScope("issue", item), "never-in-activity.txt", "text/plain", 3)
	putAssetHTTP(t, w, p, []byte("abc"), 204)
	assetAPICall(t, w, "bob", "POST", completeAssetPath(p.Asset.ID), `{}`, 200)
	if n := attachmentActivityCount(t, w, item); n != 1 {
		t.Fatalf("publication activities=%d want=1", n)
	}
	row, err := assetpg.New(w.pool).FindAsset(t.Context(), p.Asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	var exact int
	if err := w.pool.QueryRow(t.Context(), `SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field='attachment' AND verb='created'
 AND new_identifier::text=$2 AND old_identifier IS NULL AND old_value IS NULL AND new_value IS NULL
 AND actor_id=$3 AND workspace_id=$4 AND project_id=$5 AND created_at=$6 AND updated_at=$6`, item, row.ID.String(), w.ids["bob"], w.acme, w.web, row.BoundAt).Scan(&exact); err != nil || exact != 1 {
		t.Fatalf("attachment activity payload/stamp=%d %v", exact, err)
	}
	var same bool
	if err := w.pool.QueryRow(t.Context(), "SELECT updated_at=$1 AND updated_by_id=$2 FROM issues WHERE id=$3", row.BoundAt, w.ids["bob"], item).Scan(&same); err != nil || !same {
		t.Fatalf("issue change stamp differs: %v %v", same, err)
	}
	assetAPICall(t, w, "bob", "POST", completeAssetPath(p.Asset.ID), `{}`, 200)
	if n := attachmentActivityCount(t, w, item); n != 1 {
		t.Fatal("duplicate complete emitted another activity")
	}
	path := "/api/v0/assets/" + p.Asset.ID.String()
	assetAPICall(t, w, "alice", "DELETE", path, "", 204)
	assetAPICall(t, w, "alice", "DELETE", path, "", 204)
	if n := attachmentActivityCount(t, w, item); n != 2 {
		t.Fatalf("delete activities=%d want=2 total", n)
	}
	row, err = assetpg.New(w.pool).FindAsset(t.Context(), p.Asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.pool.QueryRow(t.Context(), `SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field='attachment' AND verb='deleted'
 AND old_identifier::text=$2 AND new_identifier IS NULL AND old_value IS NULL AND new_value IS NULL
 AND actor_id=$3 AND created_at=$4 AND updated_at=$4`, item, row.ID.String(), w.ids["alice"], row.DeletedAt).Scan(&exact); err != nil || exact != 1 {
		t.Fatalf("delete activity payload/stamp=%d %v", exact, err)
	}
}
