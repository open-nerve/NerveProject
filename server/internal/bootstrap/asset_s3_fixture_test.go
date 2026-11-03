package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"uuid"

	assets3 "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/s3"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/platform/s3test"
	"github.com/open-nerve/NerveProject/server/migrations"
)

type assetS3World struct {
	issueWorld
	files   *assets3.Store
	program *app
	cfg     config.Config
	stop    func()
}

func configureAssetS3(t *testing.T, cfg *config.Config, b *s3test.Bucket) {
	t.Helper()
	cfg.Files.Driver = "s3"
	cfg.Files.S3 = config.S3FilesConfig{Endpoint: b.Endpoint, Region: b.Region, Bucket: b.Name, Prefix: "application/", PathStyle: true, AccessKeyID: b.AccessKeyID, SecretAccessKey: b.SecretAccessKey}
	cfg.Files.Local.Root = filepath.Join(t.TempDir(), "must-stay-absent")
}

func buildAssetS3App(t *testing.T, cfg config.Config, requests *requestQueries, alter func(*assetComponents)) (*app, *assets3.Store) {
	t.Helper()
	var store *assets3.Store
	factory := func(ctx context.Context, files config.FilesConfig, keys *identity.Keys, logger *slog.Logger) (assetComponents, error) {
		components, err := newAssets(ctx, files, keys, logger)
		if err == nil {
			var ok bool
			store, ok = components.sealing.Inspector.(*assets3.Store)
			if !ok {
				t.Fatal("S3 configuration selected another storage implementation")
			}
			alter(&components)
		}
		return components, err
	}
	a, err := newAppWithFactories(t.Context(), cfg, slog.New(slog.DiscardHandler), migrations.FS(), testWebUI, appFactories{pool: requests.pool, assets: factory})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.close)
	if _, err := os.Stat(cfg.Files.Local.Root); !os.IsNotExist(err) {
		t.Fatalf("S3 created local storage: %v", err)
	}
	return a, store
}

func newAssetS3World(t *testing.T, b *s3test.Bucket, configure func(*config.Config), alter func(*assetComponents)) assetS3World {
	t.Helper()
	db := pgtest.NewDatabase(t)
	cfg := testConfig(t, db, false)
	configureAssetS3(t, &cfg, b)
	configure(&cfg)
	requests := &requestQueries{}
	a, store := buildAssetS3App(t, cfg, requests, alter)
	base, stop := runBuiltAppWithStop(t, a)
	w := prepareIssueWorld(t, issueWorld{requests: requests, base: base, pool: openPool(t, db), contract: apitest.Load(t), tokens: map[string]string{}, ids: map[string]uuid.UUID{}})
	return assetS3World{issueWorld: w, files: store, program: a, cfg: cfg, stop: stop}
}

func putAssetS3HTTP(t *testing.T, w assetS3World, file preparedAssetAnswer, data []byte) {
	t.Helper()
	target, err := url.Parse(file.Upload.URL)
	if err != nil || target.Scheme+"://"+target.Host != w.files.Origin() {
		t.Fatal("upload destination is not configured S3 origin")
	}
	assetRequest(t, "", file.uploadRequest(), string(data), http.StatusOK)
}

func readAssetS3HTTP(t *testing.T, w assetS3World, actor string, id uuid.UUID) (*http.Response, []byte) {
	t.Helper()
	var content struct {
		URL string `json:"url"`
	}
	decodeAnswer(t, assetAPICall(t, w.issueWorld, actor, "GET", "/api/v0/assets/"+id.String()+"/content", "", 200), &content)
	target, err := url.Parse(content.URL)
	if err != nil || target.Scheme+"://"+target.Host != w.files.Origin() {
		t.Fatal("read destination is not configured S3 origin")
	}
	return send(t, newRequest(t, "GET", content.URL, "", nil))
}
