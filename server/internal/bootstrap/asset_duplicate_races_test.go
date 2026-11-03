package bootstrap

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func TestAssetDuplicateRechecksSourceAndTarget(t *testing.T) {
	for _, deletedSide := range []string{"source", "target"} {
		t.Run(deletedSide, func(t *testing.T) {
			var armed atomic.Bool
			attempts := make(chan *assetSealAttempt, 1)
			w, store := newAssetCompleteWorld(t, func(c *assetComponents) {
				base := c.store
				controlled := controlledAssetSeal(base, attempts)
				c.sealing.Sealer = assetSealFunc(func(ctx context.Context, src, dst domain.ObjectKey, etag string) (domain.ObjectInfo, error) {
					if armed.Load() {
						return controlled(ctx, src, dst, etag)
					}
					return base.Seal(ctx, src, dst, etag)
				})
			})
			item := w.newWorkItem(t)
			source := uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "x.txt", "text/plain", []byte("old"))
			armed.Store(true)
			req := newRequest(t, "POST", w.base+duplicateAssetPath(source.Asset.ID), w.tokens["bob"], []byte(fmt.Sprintf(`{"purpose":"ISSUE_ATTACHMENT","scope":%s}`, assetScopeBody("issue", item))))
			response := sendInBackground(req)
			attempt := awaitAssetSeal(t, attempts, response)
			defer attempt.resume(nil)
			targetID := attempt.key.AssetID()
			target, err := assetpg.New(w.pool).FindAsset(t.Context(), targetID)
			if err != nil || target.Uploaded || len(target.Metadata.Candidates) != 1 || target.Metadata.Candidates[0].Key != attempt.key {
				t.Fatalf("copy did I/O before durable target: %+v %v", target, err)
			}
			victim := source.Asset.ID
			if deletedSide == "target" {
				victim = targetID
			}
			assetAPICall(t, w, "bob", "DELETE", "/api/v0/assets/"+victim.String(), "", 204)
			attempt.resume(nil)
			got := receiveWithin(t, response, 5*time.Second, "copy after a side was deleted")
			if got.err != nil || got.res.StatusCode != 404 || problemCode(t, got.body) != "asset.not_found" {
				t.Fatalf("copy published without both live sides: %+v", got)
			}
			w.contract.CheckResponse(t, req, got.res)
			after, err := assetpg.New(w.pool).FindAsset(t.Context(), targetID)
			if err != nil || after.Uploaded || after.Metadata.Final != nil || len(after.Metadata.Candidates) != 1 || after.Metadata.Candidates[0].Key != attempt.key || (after.DeletedAt != nil) != (deletedSide == "target") {
				t.Fatalf("failed copy lost recovery/tombstone: %+v %v", after, err)
			}
			if _, err := store.Inspect(t.Context(), attempt.key); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAssetDuplicateInputAndPendingSource(t *testing.T) {
	w := newIssueWorld(t)
	source := prepareAssetHTTP(t, w, "bob", "USER_AVATAR", `{"kind":"user"}`, "source.png", "image/png", 3)
	path := duplicateAssetPath(source.Asset.ID)
	for _, tc := range []struct {
		body   string
		status int
		code   string
	}{
		{`{}`, 400, "bad_request"},
		{`{"purpose":"USER_AVATAR","scope":{"kind":"user"},"size":8}`, 400, "bad_request"},
		{`{"purpose":"USER_AVATAR","scope":{"kind":"user"},"name":null}`, 400, "bad_request"},
		{`{"purpose":"unknown","scope":{"kind":"user"}}`, 422, "validation_failed"},
		{`{"purpose":"USER_AVATAR","scope":{"kind":"user"},"name":""}`, 422, "validation_failed"},
		{`{"purpose":"USER_AVATAR","scope":{"kind":"workspace"}}`, 422, "validation_failed"},
		{`{"purpose":"USER_AVATAR","scope":{"kind":"user","id":"00000000-0000-0000-0000-000000000000"}}`, 422, "validation_failed"},
		{`{"purpose":"USER_AVATAR","scope":{"kind":"user"}}`, 409, "asset.not_uploaded"},
	} {
		body := assetAPICall(t, w, "bob", "POST", path, tc.body, tc.status)
		if problemCode(t, []byte(body)) != tc.code {
			t.Fatal(body)
		}
	}
	if rows := queryIDs(t, w.pool, "SELECT id FROM file_assets"); len(rows) != 1 || rows[0] != source.Asset.ID {
		t.Fatalf("invalid/pending copy created records: %v", rows)
	}
}
