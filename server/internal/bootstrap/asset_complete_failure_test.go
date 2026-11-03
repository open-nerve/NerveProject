package bootstrap

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type assetOpenFunc func(context.Context, domain.ObjectKey, string) (io.ReadCloser, error)

func (f assetOpenFunc) Open(ctx context.Context, key domain.ObjectKey, etag string) (io.ReadCloser, error) {
	return f(ctx, key, etag)
}

func TestAssetSealFailureRecovery(t *testing.T) {
	t.Run("sealed object survives final commit rollback and retry", func(t *testing.T) {
		w, store := newAssetCompleteWorld(t, func(*assetComponents) {})
		item := w.newWorkItem(t)
		data := []byte("sealed before final commit")
		prepared := prepareAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", fmt.Sprintf(`{"kind":"issue","id":%q}`, item), "x.txt", "text/plain", len(data))
		putAssetHTTP(t, w, prepared, data, 204)
		_, err := w.pool.Exec(t.Context(), `CREATE FUNCTION reject_asset_publication() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'asset publication rejected'; END $$;
CREATE CONSTRAINT TRIGGER reject_asset_publication AFTER UPDATE ON file_assets DEFERRABLE INITIALLY DEFERRED FOR EACH ROW WHEN (NEW.is_uploaded) EXECUTE FUNCTION reject_asset_publication();`)
		if err != nil {
			t.Fatal(err)
		}
		assetAPICall(t, w, "bob", "POST", completeAssetPath(prepared.Asset.ID), `{}`, 500)
		row, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
		if err != nil || row.Uploaded || row.Metadata.Final != nil || len(row.Metadata.Candidates) != 1 {
			t.Fatalf("commit failure lost candidate: %+v %v", row, err)
		}
		old := row.Metadata.Candidates[0].Key
		if _, err := store.Inspect(t.Context(), old); err != nil {
			t.Fatalf("sealed object absent after rollback: %v", err)
		}
		if _, err := w.pool.Exec(t.Context(), `DROP TRIGGER reject_asset_publication ON file_assets`); err != nil {
			t.Fatal(err)
		}
		assetAPICall(t, w, "bob", "POST", completeAssetPath(row.ID), `{}`, 200)
		after, err := assetpg.New(w.pool).FindAsset(t.Context(), row.ID)
		if err != nil || !after.Uploaded || after.Metadata.Final == nil || after.Metadata.Final.Key == old || len(after.Metadata.Candidates) != 1 || after.Metadata.Candidates[0].Key != old {
			t.Fatalf("retry reused or forgot failed object: %+v %v", after, err)
		}
		_, got := readAssetHTTP(t, w, "bob", row.ID)
		if !bytes.Equal(got, data) {
			t.Fatal("retry published different bytes")
		}
	})
	t.Run("late success cannot publish after lease", func(t *testing.T) {
		var offset atomic.Int64
		attempts := make(chan *assetSealAttempt, 1)
		w, store := newAssetCompleteWorld(t, func(c *assetComponents) {
			c.runtimeNow = func() time.Time { return time.Now().Add(time.Duration(offset.Load())) }
			c.sealing.Sealer = controlledAssetSeal(c.store, attempts)
		})
		item := w.newWorkItem(t)
		prepared := prepareAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", fmt.Sprintf(`{"kind":"issue","id":%q}`, item), "x.txt", "text/plain", 3)
		putAssetHTTP(t, w, prepared, []byte("old"), 204)
		req := newRequest(t, "POST", w.base+completeAssetPath(prepared.Asset.ID), w.tokens["bob"], []byte(`{}`))
		response := sendInBackground(req)
		attempt := awaitAssetSeal(t, attempts, response)
		defer attempt.resume(nil)
		offset.Store(int64(31 * time.Second))
		attempt.resume(nil)
		got := receiveWithin(t, response, 5*time.Second, "late complete")
		if got.err != nil || got.res.StatusCode != 409 || problemCode(t, got.body) != "asset.scope_changed" {
			t.Fatalf("late candidate published: %+v", got)
		}
		w.contract.CheckResponse(t, req, got.res)
		row, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
		if err != nil || row.Uploaded || row.Metadata.Final != nil || len(row.Metadata.Candidates) != 1 || row.Metadata.Candidates[0].Key != attempt.key {
			t.Fatalf("late recovery lost: %+v %v", row, err)
		}
		if _, err := store.Inspect(t.Context(), attempt.key); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("destination with same size and different bytes is not published", func(t *testing.T) {
		var targetReads atomic.Int64
		w, store := newAssetCompleteWorld(t, func(c *assetComponents) {
			base := c.store
			c.sealing.Reader = assetOpenFunc(func(ctx context.Context, key domain.ObjectKey, etag string) (io.ReadCloser, error) {
				reader, err := base.Open(ctx, key, etag)
				if err != nil || key.Kind() != 'o' {
					return reader, err
				}
				data, err := io.ReadAll(reader)
				closeErr := reader.Close()
				if err != nil {
					return nil, err
				}
				if closeErr != nil {
					return nil, closeErr
				}
				data[0] ^= 1
				targetReads.Add(1)
				return io.NopCloser(bytes.NewReader(data)), nil
			})
		})
		item := w.newWorkItem(t)
		prepared := prepareAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", fmt.Sprintf(`{"kind":"issue","id":%q}`, item), "x.txt", "text/plain", 3)
		putAssetHTTP(t, w, prepared, []byte("old"), 204)
		body := assetAPICall(t, w, "bob", "POST", completeAssetPath(prepared.Asset.ID), `{}`, 409)
		if problemCode(t, []byte(body)) != "asset.object_changed" || targetReads.Load() != 1 {
			t.Fatalf("destination was not fully revalidated: %s reads=%d", body, targetReads.Load())
		}
		row, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
		if err != nil || row.Uploaded || row.Metadata.Final != nil || len(row.Metadata.Candidates) != 1 {
			t.Fatalf("mismatched destination lost recovery: %+v %v", row, err)
		}
		if _, err := store.Inspect(t.Context(), row.Metadata.Candidates[0].Key); err != nil {
			t.Fatal(err)
		}
	})
}
