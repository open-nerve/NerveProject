package bootstrap

import (
	"io"
	"net/http"
	"testing"
	"testing/fstest"
)

// The web UI needs no database: the app runs against an unreachable one.
// startApp serves testWebUI.
func TestServesTheWebUIOnEveryOtherPath(t *testing.T) {
	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})
	for _, path := range []string{"/", "/acme/projects/0199f1c2-7a1b-7c3d-8e4f-5a6b7c8d9e0f/issues"} {
		res, err := client.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}

		if res.StatusCode != http.StatusOK || string(body) != testIndexHTML {
			t.Errorf("GET %s = %d %q, want 200 with index.html", path, res.StatusCode, body)
		}
	}
}

// Mounting the web UI on "/" keeps the probes: /healthz still answers JSON.
func TestProbesAreNotTheWebUI(t *testing.T) {
	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})

	res, err := client.Get(base + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()

	if res.StatusCode != http.StatusOK || res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("GET /healthz = %d %s, want 200 application/json", res.StatusCode, res.Header.Get("Content-Type"))
	}
}
