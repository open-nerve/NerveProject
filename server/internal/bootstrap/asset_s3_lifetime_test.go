package bootstrap

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	assets3 "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/s3"
	"github.com/open-nerve/NerveProject/server/internal/platform/s3test"
)

func assetS3Composition(t *testing.T, b *s3test.Bucket) {
	t.Helper()
	cfg := testConfig(t, unreachableDB, false)
	configureAssetS3(t, &cfg, b)
	// The actual composition must read these bounded, test-generated files.
	// A nil local signing key proves this driver never requests local MACs.
	cfg.Files.S3.AccessKeyIDFile = writeFile(t, b.AccessKeyID+"\n")
	cfg.Files.S3.SecretAccessKeyFile = writeFile(t, b.SecretAccessKey+"\r\n")
	cfg.Files.S3.AccessKeyID, cfg.Files.S3.SecretAccessKey = "", ""
	components, err := newAssets(t.Context(), cfg.Files, nil, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = components.close() })
	if components.transfer != nil || components.origin != b.Endpoint || components.runtimeNow == nil {
		t.Fatal("S3 composition retained local handler or lost exact origin/runtime")
	}
	if _, err := os.Stat(cfg.Files.Local.Root); !os.IsNotExist(err) {
		t.Fatalf("S3 accessed local root: %v", err)
	}
	if err := components.close(); err != nil {
		t.Fatal(err)
	}
	if _, err := components.store.List(t.Context(), "", 1); !errors.Is(err, fs.ErrClosed) {
		t.Fatalf("app-owned store not closed: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := newAssets(ctx, cfg.Files, nil, slog.New(slog.DiscardHandler)); !errors.Is(err, context.Canceled) {
		t.Fatalf("factory lost caller context: %v", err)
	}
	cfg.Files.S3.SecretAccessKeyFile = filepath.Join(t.TempDir(), "missing-secret")
	if _, err := newAssets(t.Context(), cfg.Files, nil, slog.New(slog.DiscardHandler)); err == nil {
		t.Fatal("active S3 secret read failed silently")
	}
	if _, err := os.Stat(cfg.Files.Local.Root); !os.IsNotExist(err) {
		t.Fatal("invalid S3 fell back to local")
	}
}

// Restart the complete production composition and let its actual RunOnStart
// River cleanup execute. The injected runtime clock models elapsed leases;
// it does not change the provider clock or claim two wall-clock hours elapsed.
func restartAssetS3Cleanup(t *testing.T, w assetS3World, businessShift time.Duration) *assets3.Store {
	t.Helper()
	waitForCompletedKinds(t, w.pool, map[string]int{"asset.cleanup": 1, "cleanup.purge_deleted": 1}, &lockedBuffer{})
	var before int64
	if err := w.pool.QueryRow(t.Context(), "SELECT max(id) FROM river_job WHERE kind='asset.cleanup'").Scan(&before); err != nil {
		t.Fatal(err)
	}
	w.stop()
	w.program.close()
	cfg := w.cfg
	cfg.Clock.Offset += businessShift
	cfg.Server.AddrFile = filepath.Join(t.TempDir(), "restarted.addr")
	a, store := buildAssetS3App(t, cfg, &requestQueries{}, func(c *assetComponents) { c.runtimeNow = func() time.Time { return time.Now().Add(2 * time.Hour) } })
	runBuiltApp(t, a)
	requestEventuallyWithin(t, "production S3 cleanup after restart", 15*time.Second, func(ctx context.Context) bool {
		var completed, failures int
		if err := w.pool.QueryRow(ctx, "SELECT count(*) FILTER (WHERE state='completed'),coalesce(sum(coalesce(cardinality(errors),0)),0) FROM river_job WHERE kind='asset.cleanup' AND id>$1", before).Scan(&completed, &failures); err != nil {
			t.Fatal(err)
		}
		if failures != 0 {
			t.Fatalf("restarted S3 cleanup failed %d times", failures)
		}
		return completed == 1
	})
	return store
}
