package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

type assetInspectFunc func(context.Context, domain.ObjectKey) (domain.ObjectInfo, error)

func (f assetInspectFunc) Inspect(ctx context.Context, key domain.ObjectKey) (domain.ObjectInfo, error) {
	return f(ctx, key)
}

func TestAssetStorageDoesNotHoldResources(t *testing.T) {
	for _, phase := range []string{"Inspect", "Open", "Seal"} {
		for _, canceled := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s canceled=%t", phase, canceled), func(t *testing.T) {
				var armed, paused atomic.Bool
				var opened, closed atomic.Int64
				g := newGate()
				defer requestOpen(g)
				pause := func(ctx context.Context, at string) error {
					if at == phase && armed.Load() && paused.CompareAndSwap(false, true) {
						return g.wait(ctx)
					}
					return nil
				}
				w, store := newAssetCompleteWorldWith(t, func(c *config.Config) {
					c.Issue.WriteGate = config.WriteGateConfig{PerProject: 1, MaxQueue: 0, MaxWait: time.Second}
				}, func(c *assetComponents) {
					base := c.store
					c.sealing.Inspector = assetInspectFunc(func(ctx context.Context, key domain.ObjectKey) (domain.ObjectInfo, error) {
						info, err := base.Inspect(ctx, key)
						if err == nil {
							err = pause(ctx, "Inspect")
						}
						return info, err
					})
					c.sealing.Reader = assetOpenFunc(func(ctx context.Context, key domain.ObjectKey, etag string) (io.ReadCloser, error) {
						r, err := base.Open(ctx, key, etag)
						if err != nil {
							return nil, err
						}
						opened.Add(1)
						return assetObservedReader{ReadCloser: r, read: func(b []byte) (int, error) {
							if err := pause(ctx, "Open"); err != nil {
								return 0, err
							}
							return r.Read(b)
						}, close: func() error { closed.Add(1); return r.Close() }}, nil
					})
					c.sealing.Sealer = assetSealFunc(func(ctx context.Context, src, dst domain.ObjectKey, etag string) (domain.ObjectInfo, error) {
						info, err := base.Seal(ctx, src, dst, etag)
						if err == nil {
							err = pause(ctx, "Seal")
						}
						return info, err
					})
				})
				waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
				item := w.newWorkItem(t)
				other := uploadedAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "other.txt", "text/plain", []byte("other"))
				file := prepareAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", item), "x.txt", "text/plain", 3)
				putAssetHTTP(t, w, file, []byte("old"), 204)
				armed.Store(true)
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				req := newRequest(t, "POST", w.base+completeAssetPath(file.Asset.ID), w.tokens["bob"], []byte(`{}`)).WithContext(ctx)
				response := sendInBackground(req)
				receiveWithin(t, g.held, 5*time.Second, "external byte operation")
				row, err := assetpg.New(w.pool).FindAsset(t.Context(), file.Asset.ID)
				if err != nil || row.Uploaded || len(row.Metadata.Candidates) != 1 {
					t.Fatalf("I/O without durable candidate: %+v %v", row, err)
				}
				requestEventually(t, "no application connection during external I/O", func(context.Context) bool {
					return w.requests.appPool.Stat().AcquiredConns() == 0
				})
				// The actual app pool can immediately take all these row locks.
				tx, err := w.requests.appPool.Begin(pgtest.Soon(t))
				if err != nil {
					t.Fatal(err)
				}
				defer requestRelease(t, tx)
				for _, lock := range []struct {
					table string
					id    uuid.UUID
				}{{"users", w.ids["bob"]}, {"workspaces", w.acme}, {"projects", w.web}, {"file_assets", file.Asset.ID}} {
					if _, err := tx.Exec(pgtest.Soon(t), "SELECT id FROM "+lock.table+" WHERE id=$1 FOR UPDATE NOWAIT", lock.id); err != nil {
						t.Fatalf("external I/O held %s: %v", lock.table, err)
					}
				}
				requestRelease(t, tx)
				assetAPICall(t, w, "bob", "GET", "/api/v0/assets/"+other.Asset.ID.String()+"/content", "", 200)
				w.send(t, "bob", "PATCH", "/api/v0/issues/"+item.String(), `{"name":"Unblocked while bytes wait"}`, 200)
				if canceled {
					cancel()
				} else {
					assetAPICall(t, w, "bob", "DELETE", "/api/v0/assets/"+file.Asset.ID.String(), "", 204)
					requestOpen(g)
				}
				got := receiveWithin(t, response, 5*time.Second, "complete after external pause")
				if canceled {
					if !errors.Is(got.err, context.Canceled) {
						t.Fatalf("cancel=%+v", got)
					}
				} else {
					if got.err != nil || got.res.StatusCode != 404 {
						t.Fatalf("late publication=%+v", got)
					}
					w.contract.CheckResponse(t, req, got.res)
				}
				requestEventually(t, "all readers and app connections released", func(context.Context) bool {
					return closed.Load() == opened.Load() && w.requests.appPool.Stat().AcquiredConns() == 0
				})
				after, err := assetpg.New(w.pool).FindAsset(t.Context(), file.Asset.ID)
				if err != nil || after.Uploaded || after.Metadata.Final != nil || len(after.Metadata.Candidates) != 1 || after.Metadata.Candidates[0].Key != row.Metadata.Candidates[0].Key {
					t.Fatalf("lost recovery: %+v %v", after, err)
				}
				if phase == "Seal" {
					if _, err := store.Inspect(t.Context(), after.Metadata.Candidates[0].Key); err != nil {
						t.Fatal(err)
					}
				}
			})
		}
	}
}
