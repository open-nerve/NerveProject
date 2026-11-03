package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"
	"time"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

type assetObservedReader struct {
	io.ReadCloser
	read  func([]byte) (int, error)
	close func() error
}

func (r assetObservedReader) Read(b []byte) (int, error) { return r.read(b) }
func (r assetObservedReader) Close() error               { return r.close() }

// These faults occur at the actual object reader boundary, after a successful
// Inspect. A lost read or failed Close must not be treated as complete content.
func TestAssetCompleteStorageFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		target byte
		mode   string
	}{
		{"source read failure", 'u', "read"},
		{"destination read failure", 'o', "read"},
		{"source close failure", 'u', "close"},
		{"external timeout", 'u', "timeout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var opened, closed atomic.Int64
			w, _ := newAssetCompleteWorldWith(t, func(c *config.Config) { c.Files.OperationTimeout = time.Second }, func(c *assetComponents) {
				base := c.store
				c.sealing.Reader = assetOpenFunc(func(ctx context.Context, key domain.ObjectKey, etag string) (io.ReadCloser, error) {
					if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > time.Second {
						t.Error("reader does not use the configured one-second operation budget")
					}
					if etag == "" {
						t.Error("reader received no expected version")
					}
					source, err := base.Open(ctx, key, etag)
					if err != nil {
						return nil, err
					}
					opened.Add(1)
					return assetObservedReader{ReadCloser: source, read: func(b []byte) (int, error) {
						if key.Kind() == tc.target {
							if tc.mode == "read" {
								return 0, errors.New("private storage path and query must not escape")
							}
							if tc.mode == "timeout" {
								<-ctx.Done()
								return 0, ctx.Err()
							}
						}
						return source.Read(b)
					}, close: func() error {
						closed.Add(1)
						err := source.Close()
						if key.Kind() == tc.target && tc.mode == "close" {
							return errors.New("private close failure")
						}
						return err
					}}, nil
				})
			})
			item := w.newWorkItem(t)
			prepared := prepareAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", fmt.Sprintf(`{"kind":"issue","id":%q}`, item), "x.txt", "text/plain", 3)
			putAssetHTTP(t, w, prepared, []byte("old"), 204)
			body := assetAPICall(t, w, "bob", "POST", completeAssetPath(prepared.Asset.ID), `{}`, 503)
			if problemCode(t, []byte(body)) != "asset.storage_unavailable" || opened.Load() == 0 || closed.Load() != opened.Load() {
				t.Fatalf("failed stream leaked or became missing: %s opened=%d closed=%d", body, opened.Load(), closed.Load())
			}
			row, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
			if err != nil || row.Uploaded || row.Metadata.Final != nil || len(row.Metadata.Candidates) != 1 {
				t.Fatalf("failed stream lost candidate: %+v %v", row, err)
			}
		})
	}
}

func TestAssetCompleteOneDeadlineAndCancellation(t *testing.T) {
	t.Run("all byte operations use one deadline", func(t *testing.T) {
		var first time.Time
		var calls atomic.Int64
		w, _ := newAssetCompleteWorld(t, func(c *assetComponents) {
			base := c.store
			c.sealing.Reader = assetOpenFunc(func(ctx context.Context, key domain.ObjectKey, etag string) (io.ReadCloser, error) {
				deadline, ok := ctx.Deadline()
				if !ok {
					t.Error("unbounded byte operation")
				}
				if calls.Add(1) == 1 {
					first = deadline
				} else if !first.Equal(deadline) {
					t.Error("deadline reset between byte operations")
				}
				return base.Open(ctx, key, etag)
			})
		})
		item := w.newWorkItem(t)
		prepared := prepareAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", fmt.Sprintf(`{"kind":"issue","id":%q}`, item), "x.txt", "text/plain", 3)
		putAssetHTTP(t, w, prepared, []byte("old"), 204)
		assetAPICall(t, w, "bob", "POST", completeAssetPath(prepared.Asset.ID), `{}`, 200)
		if calls.Load() != 2 {
			t.Fatalf("full source/target streams: %d", calls.Load())
		}
	})
	t.Run("client cancellation closes the active reader", func(t *testing.T) {
		reading, closed := make(chan struct{}), make(chan struct{})
		w, _ := newAssetCompleteWorld(t, func(c *assetComponents) {
			base := c.store
			c.sealing.Reader = assetOpenFunc(func(ctx context.Context, key domain.ObjectKey, etag string) (io.ReadCloser, error) {
				source, err := base.Open(ctx, key, etag)
				if err != nil {
					return nil, err
				}
				return assetObservedReader{ReadCloser: source, read: func([]byte) (int, error) { close(reading); <-ctx.Done(); return 0, ctx.Err() }, close: func() error { err := source.Close(); close(closed); return err }}, nil
			})
		})
		item := w.newWorkItem(t)
		prepared := prepareAssetHTTP(t, w, "bob", "ISSUE_ATTACHMENT", fmt.Sprintf(`{"kind":"issue","id":%q}`, item), "x.txt", "text/plain", 3)
		putAssetHTTP(t, w, prepared, []byte("old"), 204)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		req := newRequest(t, "POST", w.base+completeAssetPath(prepared.Asset.ID), w.tokens["bob"], []byte(`{}`)).WithContext(ctx)
		result := sendInBackground(req)
		receiveWithin(t, reading, 5*time.Second, "active reader")
		cancel()
		receiveWithin(t, closed, 5*time.Second, "closed reader after cancellation")
		got := receiveWithin(t, result, 5*time.Second, "canceled HTTP request")
		if !errors.Is(got.err, context.Canceled) {
			t.Fatalf("cancel did not end request: %+v", got)
		}
		row, err := assetpg.New(w.pool).FindAsset(t.Context(), prepared.Asset.ID)
		if err != nil || row.Uploaded || row.Metadata.Final != nil || len(row.Metadata.Candidates) != 1 {
			t.Fatalf("canceled attempt lost recovery: %+v %v", row, err)
		}
	})
}
