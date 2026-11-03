package bootstrap

import (
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
)

func attachmentIssueStamp(t *testing.T, w issueWorld, item uuid.UUID) (time.Time, *uuid.UUID) {
	t.Helper()
	var at time.Time
	var by *uuid.UUID
	if err := w.pool.QueryRow(t.Context(), "SELECT updated_at,updated_by_id FROM issues WHERE id=$1", item).Scan(&at, &by); err != nil {
		t.Fatal(err)
	}
	return at, by
}

// Omitting synchronous event delivery, joining a fresh transaction, or
// swallowing subscriber/COMMIT errors would leave one side of this write.
func TestAttachmentEventsRollback(t *testing.T) {
	for _, operation := range []string{"complete", "duplicate", "delete"} {
		for _, fault := range []string{"subscriber", "commit"} {
			t.Run(operation+"/"+fault, func(t *testing.T) {
				w, files := newAssetCompleteWorld(t, func(*assetComponents) {})
				item := w.newWorkItem(t)
				p := prepareAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", attachmentScope("issue", item), "private-source.txt", "text/plain", 3)
				putAssetHTTP(t, w, p, []byte("abc"), 204)
				if operation != "complete" {
					assetAPICall(t, w, "bob", "POST", completeAssetPath(p.Asset.ID), `{}`, 200)
				}
				before, err := assetpg.New(w.pool).FindAsset(t.Context(), p.Asset.ID)
				if err != nil {
					t.Fatal(err)
				}
				oldAt, oldBy := attachmentIssueStamp(t, w, item)
				oldCount := attachmentActivityCount(t, w, item)
				trigger := "CREATE TRIGGER reject_attachment BEFORE INSERT ON issue_activities FOR EACH ROW WHEN (NEW.field='attachment') EXECUTE FUNCTION reject_attachment()"
				if fault == "commit" {
					trigger = "CREATE CONSTRAINT TRIGGER reject_attachment AFTER INSERT ON issue_activities DEFERRABLE INITIALLY DEFERRED FOR EACH ROW WHEN (NEW.field='attachment') EXECUTE FUNCTION reject_attachment()"
				}
				if _, err := w.pool.Exec(t.Context(), `CREATE FUNCTION reject_attachment() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'attachment transaction fault'; END $$;`+trigger); err != nil {
					t.Fatal(err)
				}
				path, method, body := completeAssetPath(p.Asset.ID), "POST", `{}`
				if operation == "delete" {
					path, method, body = "/api/v0/assets/"+p.Asset.ID.String(), "DELETE", ""
				}
				if operation == "duplicate" {
					path, body = "/api/v0/assets/"+p.Asset.ID.String()+"/duplicate", `{"scope":`+attachmentScope("issue", item)+`,"purpose":"ISSUE_ATTACHMENT"}`
				}
				raw := assetAPICall(t, w, "bob", method, path, body, 500)
				if strings.Contains(raw, "private-source") || strings.Contains(raw, "attachment transaction fault") || strings.Contains(raw, p.Asset.ID.String()) {
					t.Fatalf("private error detail escaped: %s", raw)
				}
				after, err := assetpg.New(w.pool).FindAsset(t.Context(), p.Asset.ID)
				if err != nil {
					t.Fatal(err)
				}
				if operation == "complete" {
					if after.Uploaded || after.BoundAt != nil || after.Metadata.Final != nil || len(after.Metadata.Candidates) != 1 {
						t.Fatalf("failed publication persisted: %+v", after)
					}
					if _, err := files.Inspect(t.Context(), after.Metadata.Candidates[0].Key); err != nil {
						t.Fatal("sealed recovery object disappeared", err)
					}
				} else if !reflect.DeepEqual(before, after) {
					t.Fatalf("failed operation changed source: %+v", after)
				}
				if operation == "duplicate" {
					ids := queryIDs(t, w.pool, "SELECT id FROM file_assets WHERE issue_id=$1 AND id<>$2", item, p.Asset.ID)
					if len(ids) != 1 {
						t.Fatalf("duplicate recovery rows=%d", len(ids))
					}
					copy, err := assetpg.New(w.pool).FindAsset(t.Context(), ids[0])
					if err != nil || copy.Uploaded || copy.BoundAt != nil || copy.Metadata.Final != nil || len(copy.Metadata.Candidates) != 1 {
						t.Fatalf("failed copy lost recovery: %+v %v", copy, err)
					}
					if _, err := files.Inspect(t.Context(), copy.Metadata.Candidates[0].Key); err != nil {
						t.Fatal(err)
					}
				}
				at, by := attachmentIssueStamp(t, w, item)
				if !at.Equal(oldAt) || !reflect.DeepEqual(by, oldBy) || attachmentActivityCount(t, w, item) != oldCount {
					t.Fatal("failed operation left issue stamp/activity")
				}
				if _, err := w.pool.Exec(t.Context(), "DROP TRIGGER reject_attachment ON issue_activities"); err != nil {
					t.Fatal(err)
				}
				status := map[string]int{"complete": 200, "delete": 204, "duplicate": 201}[operation]
				assetAPICall(t, w, "bob", method, path, body, status)
				if attachmentActivityCount(t, w, item) != oldCount+1 {
					t.Fatal("retry did not produce exactly one committed activity")
				}
			})
		}
	}
}

func TestAttachmentEventsParentDeletion(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItem(t)
	p := uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", attachmentScope("issue", item), "x.txt", "text/plain", []byte("abc"))
	w.send(t, "alice", "DELETE", "/api/v0/issues/"+item.String(), "", 204)
	row, err := assetpg.New(w.pool).FindAsset(t.Context(), p.Asset.ID)
	if err != nil || row.DeletedAt == nil || row.DeletedByID == nil || *row.DeletedByID != w.ids["alice"] {
		t.Fatalf("parent deletion lost audit: %+v %v", row, err)
	}
	if attachmentActivityCount(t, w, item) != 1 {
		t.Fatal("parent deletion emitted per-attachment user actions")
	}
}
