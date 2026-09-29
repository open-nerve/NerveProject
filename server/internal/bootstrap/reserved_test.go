package bootstrap

import (
	"io/fs"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
)

// serverPaths are the top-level path segments the server answers itself,
// beside the web app's pages: the first segment of every route the
// composition root registers but the web UI's "/", and each top-level
// directory of dirs below which a's router answers a missing file with 404
// instead of the page. dirs only names the directories to probe: the router
// serves the web UI a was built with, testWebUI (buildApp).
func serverPaths(t *testing.T, a *app, dirs fs.FS) []string {
	t.Helper()
	paths := map[string]bool{}
	for _, pattern := range a.router.Patterns() {
		path := pattern
		if _, rest, ok := strings.Cut(pattern, " "); ok {
			path = rest
		}
		if segment, _, _ := strings.Cut(strings.TrimPrefix(path, "/"), "/"); segment != "" {
			paths[segment] = true
		}
	}
	entries, err := fs.ReadDir(dirs, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		rec := httptest.NewRecorder()
		a.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/"+e.Name()+"/nerve-reserved-probe.js", nil))
		if rec.Code == http.StatusNotFound {
			paths[e.Name()] = true
		}
	}
	return slices.Sorted(maps.Keys(paths))
}

// The reserved list's server section is exactly what the server answers
// itself (M3 design 3.10, 9.4): a route added without its name on the list,
// or a name the server does not answer, fails here. A workspace named api
// would have its pages under /api/, which the API answers.
func TestTheReservedServerSlugsAreTheServersTopLevelPaths(t *testing.T) {
	a := buildApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})
	// The probes below tell the page from a 404, so the router must serve
	// the page: a web UI that answers every path with 404 would count every
	// directory as the server's.
	rec := httptest.NewRecorder()
	a.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != testIndexHTML {
		t.Fatalf("GET / = %d %q, want 200 with the test web UI's page", rec.Code, rec.Body)
	}
	got := serverPaths(t, a, testWebUI)
	if want := slices.Sorted(slices.Values(workspace.ReservedSlugs().Server)); !slices.Equal(got, want) {
		t.Errorf("the server answers the top-level paths %q; the reserved list's server section is %q", got, want)
	}
	// The probe tells the web UI's page from its 404: a directory whose
	// missing files get the page is the app's, not the server's.
	pages := fstest.MapFS{"index.html": {Data: []byte(testIndexHTML)}, "icons/logo.svg": {Data: []byte("<svg/>")}}
	if got := serverPaths(t, a, pages); slices.Contains(got, "icons") {
		t.Errorf("serverPaths = %q; icons/ is served the page, want it left out", got)
	}
}
