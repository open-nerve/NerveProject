package bootstrap

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

func TestRawStorageRoutesAreExact(t *testing.T) {
	a := buildApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})
	// Compare the whole router: an alias outside /static is not an exemption.
	assertStorageDriverRoutes(t, a, "local")
}

func TestRawStorageUnsignedAndUnknownPaths(t *testing.T) {
	base := startApp(t, testConfig(t, unreachableDB, false), fstest.MapFS{})
	noRedirect := *client
	noRedirect.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	for _, method := range []string{"PUT", "GET", "HEAD"} {
		t.Run(method+" without signature", func(t *testing.T) {
			req := newRequest(t, method, base+"/static/local/"+string(assetKey(t, 'o')), "irrelevant-bearer", nil)
			res, body := send(t, req)
			if res.StatusCode != http.StatusForbidden || res.Header.Get("Content-Type") != httpserver.ContentTypeProblem || res.Header.Get("Cache-Control") != "no-store" || res.Header.Get("WWW-Authenticate") != "" {
				t.Fatalf("unsigned %s status=%d headers=%v body=%s", method, res.StatusCode, res.Header, body)
			}
			if method == "HEAD" {
				if len(body) != 0 {
					t.Fatal("HEAD has a body")
				}
			} else if problemCode(t, body) != "asset.invalid_signature" {
				t.Fatalf("wrong refusal: %s", body)
			}
		})
	}
	for _, path := range []string{
		"/static", "/static/", "/static/missing", "/static/local", "/static/local/a/b",
		"/static//missing", "/static/local/../../settings", "//static/local/../../settings",
		"/elsewhere/../static/missing", "/%73tatic/local/../../settings",
	} {
		t.Run(path, func(t *testing.T) {
			req := newRequest(t, "GET", base+path+"?payload=secret&tag=secret", "", nil)
			res, err := noRedirect.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = res.Body.Close() }()
			body, err := io.ReadAll(res.Body)
			if err != nil || res.StatusCode != http.StatusNotFound || res.Header.Get("Content-Type") != httpserver.ContentTypeProblem || res.Header.Get("Cache-Control") != "no-store" || res.Header.Get("Location") != "" {
				t.Fatalf("unknown storage path status=%d headers=%v body=%q err=%v", res.StatusCode, res.Header, body, err)
			}
			if problemCode(t, body) != httpserver.CodeNotFound || strings.Contains(string(body), "secret") || strings.Contains(string(body), path) {
				t.Fatalf("unknown path leaked input or wrong code: %s", body)
			}
		})
	}
	for _, path := range []string{"/elsewhere/../settings", "/static-more/../settings"} {
		res, err := noRedirect.Get(base + path)
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		if res.StatusCode != http.StatusTemporaryRedirect || res.Header.Get("Location") != "/settings" {
			t.Fatalf("unrelated path normalization changed: %s status=%d location=%q", path, res.StatusCode, res.Header.Get("Location"))
		}
	}
}
