package bootstrap

import (
	"reflect"
	"testing"
	"time"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func TestAssetDeletePrepared(t *testing.T) {
	w, store := newAssetCompleteWorld(t, func(*assetComponents) {})
	item := w.newWorkItem(t)
	for _, uploaded := range []bool{false, true} {
		name := "pending"
		if uploaded {
			name = "uploaded"
		}
		t.Run(name, func(t *testing.T) {
			prepared := prepareAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "delete.txt", "text/plain", 3)
			putAssetHTTP(t, w, prepared, []byte("old"), 204)
			if uploaded {
				assetAPICall(t, w, "bob", "POST", completeAssetPath(prepared.Asset.ID), `{}`, 200)
			}
			before, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
			if err != nil {
				t.Fatal(err)
			}
			path := "/api/v0/assets/" + prepared.Asset.ID.String()
			foreign := assetAPICall(t, w, "alice", "DELETE", path, "", 404)
			if problemCode(t, []byte(foreign)) != "asset.not_found" {
				t.Fatal(foreign)
			}
			assetAPICall(t, w, "bob", "DELETE", path, "", 204)
			deleted, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
			if err != nil || deleted.DeletedAt == nil || deleted.DeletedByID == nil || *deleted.DeletedByID != w.ids["bob"] || !deleted.UpdatedAt.Equal(*deleted.DeletedAt) || !reflect.DeepEqual(before.Metadata, deleted.Metadata) || deleted.Uploaded != uploaded {
				t.Fatalf("deletion forgot objects/audit: %+v %v", deleted, err)
			}
			assetAPICall(t, w, "bob", "DELETE", path, "", 404)
			again, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
			if err != nil || !reflect.DeepEqual(again, deleted) {
				t.Fatalf("repeat changed original tombstone: %+v %v", again, err)
			}
			assetAPICall(t, w, "bob", "GET", path, "", 404)
			assetAPICall(t, w, "bob", "GET", path+"/content", "", 404)
			assetAPICall(t, w, "bob", "POST", path+"/complete", `{}`, 404)
			if _, err := store.Inspect(t.Context(), before.Metadata.Staging.Key); err != nil {
				t.Fatal("delete synchronously removed staging", err)
			}
			if uploaded {
				if _, err := store.Inspect(t.Context(), before.Metadata.Final.Key); err != nil {
					t.Fatal("delete synchronously removed final", err)
				}
			}
		})
	}
	t.Run("bound fixture is refused without changing its row", func(t *testing.T) {
		prepared := uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "bound.txt", "text/plain", []byte("old"))
		if _, err := w.pool.Exec(t.Context(), "UPDATE file_assets SET bound_at=clock_timestamp() WHERE id=$1", prepared.Asset.ID); err != nil {
			t.Fatal(err)
		}
		before, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
		if err != nil {
			t.Fatal(err)
		}
		body := assetAPICall(t, w, "bob", "DELETE", "/api/v0/assets/"+prepared.Asset.ID.String(), "", 409)
		if problemCode(t, []byte(body)) != "asset.bound" {
			t.Fatal(body)
		}
		after, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
		if err != nil || !reflect.DeepEqual(before, after) {
			t.Fatalf("bound asset changed: %+v %v", after, err)
		}
	})
}

func TestAssetDeletePreparedBeforeLateComplete(t *testing.T) {
	attempts := make(chan *assetSealAttempt, 1)
	w, store := newAssetCompleteWorld(t, func(c *assetComponents) { c.sealing.Sealer = controlledAssetSeal(c.store, attempts) })
	item := w.newWorkItem(t)
	prepared := prepareAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "x.txt", "text/plain", 3)
	putAssetHTTP(t, w, prepared, []byte("old"), 204)
	req := newRequest(t, "POST", w.base+completeAssetPath(prepared.Asset.ID), w.tokens["bob"], []byte(`{}`))
	response := sendInBackground(req)
	attempt := awaitAssetSeal(t, attempts, response)
	defer attempt.resume(nil)
	assetAPICall(t, w, "bob", "DELETE", "/api/v0/assets/"+prepared.Asset.ID.String(), "", 204)
	before, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	attempt.resume(nil)
	got := receiveWithin(t, response, 5*time.Second, "late complete after delete")
	if got.err != nil || got.res.StatusCode != 404 || problemCode(t, got.body) != "asset.not_found" {
		t.Fatalf("deleted preparation resurrected: %+v", got)
	}
	w.contract.CheckResponse(t, req, got.res)
	after, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
	if err != nil || !reflect.DeepEqual(before, after) || len(after.Metadata.Candidates) != 1 || after.Metadata.Candidates[0].Key != attempt.key || after.Metadata.Candidates[0].Phase != domain.ObjectActive {
		t.Fatalf("late result changed tombstone or recovery: %+v %v", after, err)
	}
	if _, err := store.Inspect(t.Context(), attempt.key); err != nil {
		t.Fatal(err)
	}
}
