package bootstrap

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"testing/fstest"

	"github.com/open-nerve/NerveProject/server/internal/platform/buildinfo"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
)

// The API needs no database: the app runs against an unreachable one.
func TestServesTheInstanceAPI(t *testing.T) {
	contract := apitest.Load(t)
	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})
	req, err := http.NewRequest(http.MethodGet, base+"/api/v0/instance", nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()

	contract.CheckResponse(t, req, res)
	var got struct {
		Product    string `json:"product"`
		Version    string `json:"version"`
		APIVersion string `json:"api_version"`
	}
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if res.StatusCode != http.StatusOK || got.Product != "Nerve" || got.Version != buildinfo.Get().Version || got.APIVersion != "v0" {
		t.Errorf("GET /api/v0/instance = %d %+v, want 200 Nerve %s v0", res.StatusCode, got, buildinfo.Get().Version)
	}
	if res.Header.Get(httpserver.HeaderRequestID) == "" {
		t.Error("response has no X-Request-Id: the platform middleware did not run")
	}
}

// Mounting a module keeps the platform's /api/ fallback: an unknown path and
// a known path with the wrong method both answer 404 problem+json.
func TestUnknownAPIRequestsStillAnswerProblem404(t *testing.T) {
	contract := apitest.Load(t)
	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})
	for _, tt := range []struct{ method, path string }{
		{http.MethodGet, "/api/v0/nope"},
		{http.MethodPost, "/api/v0/instance"},
	} {
		req, err := http.NewRequest(tt.method, base+tt.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if err != nil {
			t.Fatal(err)
		}

		if res.StatusCode != http.StatusNotFound || res.Header.Get("Content-Type") != httpserver.ContentTypeProblem {
			t.Errorf("%s %s = %d %s, want 404 problem+json", tt.method, tt.path, res.StatusCode, res.Header.Get("Content-Type"))
		}
		contract.CheckSchema(t, "Problem", body)
		var p httpserver.Problem
		if err := json.Unmarshal(body, &p); err != nil || p.Detail != "no API endpoint for "+tt.method+" "+tt.path {
			t.Errorf("%s %s detail = %q (%v), want no API endpoint for %s %s", tt.method, tt.path, p.Detail, err, tt.method, tt.path)
		}
	}
}
