package bootstrap

import (
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
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
	base := startApp(t, cfg, migrations.FS())
	if origin == "" {
		origin = base
	}
	w := issueWorld{base: base, pool: openPool(t, db), contract: apitest.Load(t), tokens: map[string]string{}, ids: map[string]uuid.UUID{}}
	w.tokens["alice"] = registerAccount(t, w.contract, base, "alice@example.com").AccessToken
	w.ids["alice"] = accountID(t, w.contract, base, w.tokens["alice"])
	var workspace struct{ ID uuid.UUID }
	decodeAnswer(t, w.send(t, "alice", "POST", "/api/v0/workspaces", `{"name":"Images","slug":"acme"}`, 201), &workspace)
	w.acme = workspace.ID
	w.web = createdProject(t, w.contract, base, w.tokens["alice"], "acme", "Web", "WEB")
	return w, origin
}
