package bootstrap

import (
	"context"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func TestAttachmentPublicationLegacyRetry(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItem(t)
	p := uploadedAssetHTTP(t, w, "bob", "ISSUE_DESCRIPTION", attachmentScope("issue", item), "legacy.png", "image/png", assetTestPNG(t))
	// Explicit pre-upgrade fixture: P3 completed ordinary attachments without
	// binding. Preserve actual uploaded bytes and all other metadata/audit.
	if _, err := w.pool.Exec(t.Context(), "UPDATE file_assets SET entity_type='ISSUE_ATTACHMENT' WHERE id=$1", p.Asset.ID); err != nil {
		t.Fatal(err)
	}
	before, err := assetpg.New(w.pool).FindAsset(t.Context(), p.Asset.ID)
	if err != nil || before.BoundAt != nil {
		t.Fatalf("invalid legacy fixture: %+v %v", before, err)
	}
	path := "/api/v0/assets/" + p.Asset.ID.String()
	assetAPICall(t, w, "alice", "GET", path, "", 404)
	assetAPICall(t, w, "alice", "POST", path+"/complete", `{}`, 404)
	assetAPICall(t, w, "bob", "POST", path+"/complete", `{}`, 200)
	after, err := assetpg.New(w.pool).FindAsset(t.Context(), p.Asset.ID)
	if err != nil || after.BoundAt == nil || !after.CreatedAt.Equal(before.CreatedAt) || !reflect.DeepEqual(after.Metadata, before.Metadata) || attachmentActivityCount(t, w, item) != 1 {
		t.Fatalf("legacy retry lost origin or first binding: %+v %v", after, err)
	}
	assetAPICall(t, w, "bob", "POST", path+"/complete", `{}`, 200)
	again, err := assetpg.New(w.pool).FindAsset(t.Context(), p.Asset.ID)
	if err != nil || !reflect.DeepEqual(again, after) || attachmentActivityCount(t, w, item) != 1 {
		t.Fatal("legacy retry was not idempotent", err)
	}
	assetAPICall(t, w, "alice", "GET", path, "", 200)
}

// T3 exercises the actual T2 port under the owner's project lock; the public
// content save caller is connected in T5 and proved separately there.
func bindDescriptionForT3(t *testing.T, w issueWorld, item, file uuid.UUID) {
	t.Helper()
	binder := asset.NewContentBindings(asset.Provide(w.pool), clockAt(time.Now().UTC()))
	q := asset.ContentBindRequest{ActorID: w.ids["bob"], Host: asset.ContentHost{WorkspaceID: w.acme, ProjectID: w.web, IssueID: item}, ImageIDs: []uuid.UUID{file}}
	if err := postgres.NewTxManager(w.pool, time.Second).WithinTx(t.Context(), func(ctx context.Context) error {
		tx, _ := postgres.Tx(ctx)
		if _, err := tx.Exec(ctx, "SELECT id FROM projects WHERE id=$1 FOR NO KEY UPDATE", w.web); err != nil {
			return err
		}
		at, err := binder.LockContent(ctx, q)
		if err != nil {
			return err
		}
		return binder.BindContent(ctx, q, at)
	}); err != nil {
		t.Fatal(err)
	}
}

func TestBoundAssetAccessDescriptionCannotDelete(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItem(t)
	p := uploadedAssetHTTP(t, w, "bob", "ISSUE_DESCRIPTION", attachmentScope("issue", item), "private.png", "image/png", assetTestPNG(t))
	bindDescriptionForT3(t, w, item, p.Asset.ID)
	before, err := assetpg.New(w.pool).FindAsset(t.Context(), p.Asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, who := range []string{"bob", "alice"} {
		assetAPICall(t, w, who, "GET", "/api/v0/assets/"+p.Asset.ID.String(), "", 200)
		_, data := readAssetHTTP(t, w, who, p.Asset.ID)
		if len(data) == 0 {
			t.Fatal("bound image lost actual content")
		}
		body := assetAPICall(t, w, who, "DELETE", "/api/v0/assets/"+p.Asset.ID.String(), "", 409)
		if problemCode(t, []byte(body)) != "asset.bound" {
			t.Fatal(body)
		}
	}
	after, err := assetpg.New(w.pool).FindAsset(t.Context(), p.Asset.ID)
	if err != nil || !reflect.DeepEqual(after, before) || attachmentActivityCount(t, w, item) != 0 {
		t.Fatal("description changed or emitted ordinary attachment action", err)
	}
}
