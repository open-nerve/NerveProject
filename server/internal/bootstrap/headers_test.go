package bootstrap

import (
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
