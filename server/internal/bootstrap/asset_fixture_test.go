package bootstrap

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/configs"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/local"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// Keep network and file budgets exactly as shipped; only isolate the address,
// filesystem and database, and retain the existing cheap business test setup.
func defaultAssetConfig(t *testing.T, dbURL string) config.Config {
	t.Helper()
	defaults, err := config.Load(config.Sources{Embedded: configs.FS(), Environ: []string{"NERVE_ENV=test", "NERVE_DATABASE__URL=" + dbURL}})
	if err != nil {
		t.Fatal(err)
	}
	cfg := testConfig(t, dbURL, false)
	defaults.Server.Addr, defaults.Server.AddrFile = cfg.Server.Addr, cfg.Server.AddrFile
	defaults.Files.Local.Root = cfg.Files.Local.Root
	cfg.Server, cfg.Files = defaults.Server, defaults.Files
	return cfg
}

func testFilesConfig(t *testing.T) config.FilesConfig {
	t.Helper()
	return config.FilesConfig{
		UploadTTL: 15 * time.Minute, OperationTimeout: 10 * time.Second, CleanupInterval: time.Hour, CleanupBatch: 100, CleanupBudget: 10 * time.Second,
		SizeLimit: 5 << 20, Driver: "local", Local: config.LocalFilesConfig{Root: t.TempDir()},
		MinTransferBytesPerSecond: 128 << 10, TransferGrace: 15 * time.Second,
		TransferIdleTimeout: 30 * time.Second, MaxTransferDuration: 15 * time.Minute,
		MaxConcurrentTransfers: 16, TransferRate: roomy,
	}
}

// Raw transport and storage-layer tests use this second handle with their
// own PEM/config and real local ports; business stories use the asset API.
func assetClientStore(t *testing.T, cfg config.Config) *local.Store {
	t.Helper()
	pem, err := readSigningKey(cfg.Auth.JWT.PrivateKeyFile)
	if err != nil {
		t.Fatal(err)
	}
	keys, err := identity.LoadKeys(pem, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	up, read, err := assetMACs(keys)
	if err != nil {
		t.Fatal(err)
	}
	store, err := local.New(local.Config{Root: cfg.Files.Local.Root, SizeLimit: cfg.Files.SizeLimit}, local.Deps{UploadMAC: up, ReadMAC: read, RuntimeNow: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error(err)
		}
	})
	return store
}

func assetKey(t *testing.T, kind byte) domain.ObjectKey {
	t.Helper()
	key, err := domain.NewObjectKey(kind, uuid.New(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func assetRequest(t *testing.T, base string, signed domain.SignedRequest, body string, status int) string {
	t.Helper()
	req, err := http.NewRequest(signed.Method, base+signed.URL, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	for key, value := range signed.Headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != status {
		t.Fatalf("%s transfer status=%d body=%q err=%v; want %d", signed.Method, resp.StatusCode, data, err, status)
	}
	return string(data)
}
