package bootstrap

import (
	"net/http"
	"testing"
	"testing/fstest"
)

// Every response of the wired app carries the security headers of the fixed
// chain (M2 design 8.3): pages, probes, API answers and API problems.
func TestSecurityHeadersOnEveryResponse(t *testing.T) {
	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})
	want := map[string]string{"X-Content-Type-Options": "nosniff", "Referrer-Policy": "same-origin", "X-Frame-Options": "DENY"}
	for _, path := range []string{"/", "/settings/profile/general", "/healthz", "/api/v0/instance", "/api/v0/nope", "/api/v0/me"} {
		res, err := client.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()

		for header, value := range want {
			if got := res.Header.Get(header); got != value {
				t.Errorf("GET %s (%d): %s = %q, want %q", path, res.StatusCode, header, got, value)
			}
		}
	}
}

// No cache may store an API response (M2 design 8.3): an answer, the /api/
// fallback's 404 and a 401 alike. The web UI's files keep their own
// caching: index.html is revalidated, a hashed asset kept for good.
func TestOnlyAPIResponsesAreNotStored(t *testing.T) {
	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})
	for _, tt := range []struct{ path, want string }{
		{"/api/v0/instance", "no-store"},
		{"/api/v0/nope", "no-store"},
		{"/api/v0/me", "no-store"},
		{"/", "no-cache"},
		{"/" + testAsset, "public, max-age=31536000, immutable"},
		{"/healthz", ""},
	} {
		res, err := client.Get(base + tt.path)
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()

		if got := res.Header.Get("Cache-Control"); got != tt.want {
			t.Errorf("GET %s (%d): Cache-Control = %q, want %q", tt.path, res.StatusCode, got, tt.want)
		}
	}
}

// The answer that holds a new personal access token is not stored either.
func TestACreatedTokenIsNotStored(t *testing.T) {
	contract, base, token := accountApp(t, "cache@example.com")
	req := newRequest(t, http.MethodPost, base+"/api/v0/me/api-tokens", token, []byte(`{"label":"cache"}`))

	res, body := send(t, req)

	contract.CheckResponse(t, req, res)
	if res.StatusCode != http.StatusCreated || res.Header.Get("Cache-Control") != "no-store" {
		t.Errorf("POST /me/api-tokens = %d %s, Cache-Control %q; want 201 and no-store", res.StatusCode, body, res.Header.Get("Cache-Control"))
	}
}
