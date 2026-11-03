package bootstrap

import (
	"log/slog"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/platform/s3test"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// Unlike the older issue world, every business row here comes from an API:
// registration, workspace creation, and project creation (including membership).
func newImageStorageWorld(t *testing.T, driver string) (issueWorld, string) {
	t.Helper()
	db := pgtest.NewDatabase(t)
	cfg := testConfig(t, db, false)
	origin := ""
	if driver == "s3" {
		bucket := s3test.New(t).NewBucket(t)
		configureAssetS3(t, &cfg, bucket)
		origin = bucket.Endpoint
	} else if driver != "local" {
		t.Fatal("unexpected image driver", driver)
	}
	w, _ := startImageAPIWorld(t, cfg, appFactories{pool: postgres.NewPool, assets: newAssets})
	if origin == "" {
		origin = w.base
	}
	return w, origin
}

// The existing composition factories expose observations/faults without any
// production test switch. stop ends the HTTP server and its actual job workers.
func startImageAPIWorld(t *testing.T, cfg config.Config, factories appFactories) (issueWorld, func()) {
	t.Helper()
	a, err := newAppWithFactories(t.Context(), cfg, slog.New(slog.DiscardHandler), migrations.FS(), testWebUI, factories)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.close)
	base, stop := runBuiltAppWithStop(t, a)
	w := issueWorld{base: base, pool: openPool(t, cfg.Database.URL), contract: apitest.Load(t), tokens: map[string]string{}, ids: map[string]uuid.UUID{}}
	w.tokens["alice"] = registerAccount(t, w.contract, base, "alice@example.com").AccessToken
	w.ids["alice"] = accountID(t, w.contract, base, w.tokens["alice"])
	var workspace struct{ ID uuid.UUID }
	decodeAnswer(t, w.send(t, "alice", "POST", "/api/v0/workspaces", `{"name":"Images","slug":"acme"}`, 201), &workspace)
	w.acme = workspace.ID
	w.web = createdProject(t, w.contract, base, w.tokens["alice"], "acme", "Web", "WEB")
	return w, stop
}
