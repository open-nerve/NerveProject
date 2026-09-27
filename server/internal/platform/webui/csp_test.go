package webui

import (
	"net/http"
	"testing"
	"testing/fstest"
)

// A page like the one React Router builds: inline scripts, one of them a
// module, a script file, and a script repeated. The hashes were computed
// apart from this package (Python's hashlib and base64).
const scriptedIndex = `<!DOCTYPE html><html><head>` +
	`<script>document.documentElement.dataset.theme = "dark";</script>` +
	`<link rel="modulepreload" href="/assets/entry.client-a1.js"/>` +
	`<script src="/assets/legacy-c3.js"></script>` +
	`</head><body>` +
	`<SCRIPT>window.__ctx = {"a":1};</SCRIPT >` +
	"<script type=\"module\" async=\"\">import \"/assets/manifest-1.js\";\n  window.x = 1;</script>" +
	`<script>window.__ctx = {"a":1};</script>` +
	`</body></html>`

const scriptedPolicy = "default-src 'self'; " +
	"script-src 'self' 'sha256-RH/36EFn2TnZtn39Gf6aq1rnKqEXnEQbfAXoYkwhRP0=' " +
	"'sha256-CACaKUMP3bUaeK7JgU+grVR6sA9VUsIgZNwdQnbXMCA=' 'sha256-p61WDS2Nz6pOwhZMG1Gv1r9QDC4titM1f6gnicWbCZU='; " +
	"style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; " +
	"object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'"

func TestPolicyAllowsEachInlineScriptByItsHash(t *testing.T) {
	if got := contentSecurityPolicy([]byte(scriptedIndex)); got != scriptedPolicy {
		t.Errorf("policy =\n%s\nwant\n%s", got, scriptedPolicy)
	}
}

func TestPolicyOfAPageWithoutInlineScriptsAllowsOnlyNervesScripts(t *testing.T) {
	want := "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; " +
		"font-src 'self' data:; connect-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; " +
		"frame-ancestors 'none'"
	if got := contentSecurityPolicy([]byte(indexHTML)); got != want {
		t.Errorf("policy =\n%s\nwant\n%s", got, want)
	}
}

// The policy goes on the pages, made from the index.html being served, and on
// nothing else: files, missing assets and the answers of an unbuilt frontend
// have none.
func TestOnlyPagesCarryThePolicy(t *testing.T) {
	scripted := fstest.MapFS{
		"index.html":                {Data: []byte(scriptedIndex)},
		"site.webmanifest.json":     {Data: []byte(`{"name":"Nerve"}`)},
		"assets/entry.client-a1.js": {Data: []byte("export {};")},
	}
	h := Handler(scripted)
	for _, tt := range []struct{ method, target, want string }{
		{http.MethodGet, "/", scriptedPolicy},
		{http.MethodGet, "/sign-up?next_path=%2F", scriptedPolicy},
		{http.MethodHead, "/onboarding", scriptedPolicy},
		{http.MethodGet, "/assets/entry.client-a1.js", ""},
		{http.MethodGet, "/site.webmanifest.json", ""},
		{http.MethodGet, "/assets/entry.client-old.js", ""},
	} {
		rec := serve(h, tt.method, tt.target)
		if got := rec.Result().Header.Get("Content-Security-Policy"); got != tt.want {
			t.Errorf("%s %s (%d): Content-Security-Policy = %q, want %q", tt.method, tt.target, rec.Code, got, tt.want)
		}
	}

	unbuilt := serve(Handler(fstest.MapFS{".gitkeep": {}}), http.MethodGet, "/")
	if got := unbuilt.Result().Header.Get("Content-Security-Policy"); got != "" {
		t.Errorf("unbuilt GET / (%d): Content-Security-Policy = %q, want none", unbuilt.Code, got)
	}
}
