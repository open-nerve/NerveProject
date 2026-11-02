package bootstrap

import (
	"net/http"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

func TestRawStoragePreservesLegalWorkspacePages(t *testing.T) {
	contract := apitest.Load(t)
	base := startApp(t, testConfig(t, pgtest.NewDatabase(t), false), migrations.FS())
	token := registerAccount(t, contract, base, "storage-owner@example.com").AccessToken
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", token, `{"name":"Storage team","slug":"storage"}`)
	if status != http.StatusCreated {
		t.Fatalf("create storage workspace: %d %s", status, body)
	}
	t.Logf("workspace slug storage accepted: HTTP %d", status)
	status, body = call(t, contract, http.MethodGet, base+"/api/v0/workspaces/storage", token, "")
	if status != http.StatusOK {
		t.Fatalf("read existing storage workspace: %d %s", status, body)
	}
	t.Logf("workspace API remains accessible: HTTP %d", status)
	for _, path := range []string{"/storage", "/storage/projects", "/storage/settings", "/static-more/projects"} {
		res, page := send(t, newRequest(t, http.MethodGet, base+path, "", nil))
		t.Logf("page %s: HTTP %d content-type=%s", path, res.StatusCode, res.Header.Get("Content-Type"))
		if res.StatusCode != http.StatusOK || !strings.Contains(string(page), testIndexHTML) {
			t.Errorf("workspace page %s is intercepted: status=%d body=%s", path, res.StatusCode, page)
		}
	}
}

func TestRawStorageLeavesBuildAssetsAndOldURLs(t *testing.T) {
	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})
	for _, tc := range []struct {
		path   string
		status int
		cache  string
		body   string
	}{
		{"/" + testAsset, 200, "public, max-age=31536000, immutable", "export {};\n"},
		{"/assets/missing.js", 404, "", "404 page not found\n"},
		{"/storage/local/old", 200, "no-cache", testIndexHTML},
		{"/static/missing", 404, "no-store", ""},
	} {
		res, body := send(t, newRequest(t, "GET", base+tc.path, "", nil))
		if res.StatusCode != tc.status || res.Header.Get("Cache-Control") != tc.cache || (tc.body != "" && string(body) != tc.body) {
			t.Errorf("%s: status=%d cache=%q body=%q", tc.path, res.StatusCode, res.Header.Get("Cache-Control"), body)
		}
	}
}
