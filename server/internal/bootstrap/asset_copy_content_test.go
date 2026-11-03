package bootstrap

import (
	"bytes"
	"context"
	"io"
	"sync/atomic"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
)

// Persisted object records are recovery facts, not a substitute for the
// asset's declared size/type. Simulate a stale record at the database boundary.
func TestAssetDuplicateChecksDeclaredSize(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItem(t)
	source := uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "source.txt", "text/plain", []byte("old"))
	if _, err := w.pool.Exec(t.Context(), "UPDATE file_assets SET size=4 WHERE id=$1", source.Asset.ID); err != nil {
		t.Fatal(err)
	}
	body := assetAPICall(t, w, "bob", "POST", duplicateAssetPath(source.Asset.ID), `{"purpose":"ISSUE_ATTACHMENT","scope":`+assetScopeBody("issue", item)+`}`, 422)
	if problemCode(t, []byte(body)) != "asset.content_invalid" {
		t.Fatal(body)
	}
	rows := queryIDs(t, w.pool, "SELECT id FROM file_assets WHERE id<>$1", source.Asset.ID)
	if len(rows) != 1 {
		t.Fatalf("failed copy reservation missing: %v", rows)
	}
	target, err := assetpg.New(w.pool).FindAsset(t.Context(), rows[0])
	if err != nil || target.Uploaded || target.Metadata.Final != nil || len(target.Metadata.Candidates) != 1 {
		t.Fatalf("invalid content published or recovery lost: %+v %v", target, err)
	}
}

func TestAssetCompleteChecksDeclaredSize(t *testing.T) {
	w := newIssueWorld(t)
	item := w.newWorkItem(t)
	source := prepareAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "source.txt", "text/plain", 3)
	putAssetHTTP(t, w, source, []byte("old"), 204)
	if _, err := w.pool.Exec(t.Context(), "UPDATE file_assets SET size=4 WHERE id=$1", source.Asset.ID); err != nil {
		t.Fatal(err)
	}
	body := assetAPICall(t, w, "bob", "POST", completeAssetPath(source.Asset.ID), `{}`, 422)
	if problemCode(t, []byte(body)) != "asset.content_invalid" {
		t.Fatal(body)
	}
}

func TestAssetDuplicateVerifiesSourceProof(t *testing.T) {
	var armed atomic.Bool
	var seals atomic.Int64
	w, _ := newAssetCompleteWorld(t, func(c *assetComponents) {
		base := c.store
		c.sealing.Reader = assetOpenFunc(func(ctx context.Context, key domain.ObjectKey, etag string) (io.ReadCloser, error) {
			source, err := base.Open(ctx, key, etag)
			if err != nil || !armed.Load() {
				return source, err
			}
			data, err := io.ReadAll(source)
			closeErr := source.Close()
			if err != nil {
				return nil, err
			}
			if closeErr != nil {
				return nil, closeErr
			}
			data[0] ^= 1
			return io.NopCloser(bytes.NewReader(data)), nil
		})
		c.sealing.Sealer = assetSealFunc(func(ctx context.Context, src, dst domain.ObjectKey, etag string) (domain.ObjectInfo, error) {
			if armed.Load() {
				seals.Add(1)
			}
			return base.Seal(ctx, src, dst, etag)
		})
	})
	item := w.newWorkItem(t)
	source := uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "source.txt", "text/plain", []byte("old"))
	armed.Store(true)
	body := assetAPICall(t, w, "bob", "POST", duplicateAssetPath(source.Asset.ID), `{"purpose":"ISSUE_ATTACHMENT","scope":`+assetScopeBody("issue", item)+`}`, 409)
	if problemCode(t, []byte(body)) != "asset.object_changed" || seals.Load() != 0 {
		t.Fatalf("unproved source reached copy: %s seals=%d", body, seals.Load())
	}
}
