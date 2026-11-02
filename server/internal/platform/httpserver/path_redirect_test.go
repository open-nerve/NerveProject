package httpserver

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPathRedirectPolicyTracksNormalizationTraversal(t *testing.T) {
	const query = "?payload=capability-payload&tag=capability-tag"
	for _, tc := range []struct{ prefix, path string }{
		{"/static", "/static//local/key"},
		{"/static", "/static/./local/key"},
		{"/static", "/static/local/../../settings"},
		{"/static", "/elsewhere/../static/local/../../settings"},
		{"/static", "/elsewhere/../%73tatic/local/../../settings"},
		{"/static", "//static/local/key"},
		{"/static", "/elsewhere/..//static/local/../../settings"},
		{"/static", "/elsewhere/%2e%2e//static/local/../../settings"},
		{"/static", "/elsewhere/../static%2flocal/../../settings"},
		{"/assets/storage", "/assets//storage/local/../../../settings"},
		{"/assets/storage", "/assets/./storage/local/../../../settings"},
		{"/assets/storage", "/%61ssets/./storage/local/../../../settings"},
		{"/assets/storage", "/elsewhere/../assets/storage/local/../../../settings"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			router := NewRouter(slog.New(slog.DiscardHandler))
			router.RejectPathRedirects(tc.prefix)
			router.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
				t.Error("noncanonical protected path reached an endpoint")
				w.WriteHeader(http.StatusNoContent)
			})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest("GET", tc.path+query, nil))
			if response.Code != http.StatusNotFound || response.Header().Get("Location") != "" || response.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("path policy: status=%d headers=%v body=%s", response.Code, response.Header(), response.Body)
			}
			if strings.Contains(response.Body.String(), "capability") {
				t.Fatalf("response exposes query: %s", response.Body)
			}
		})
	}
}

func TestPathRedirectPolicyLeavesOtherPathsAlone(t *testing.T) {
	for _, tc := range []struct {
		path, location string
		status         int
	}{
		{"/static/local/key?payload=x&tag=y", "", 204},
		{"/static/", "", 204},
		{"/%73tatic/local/key", "", 204},
		{"/static/%2e%2e/settings", "", 204}, // ServeMux does not clean escaped dots.
		{"/static-more//projects", "/static-more/projects", 307},
		{"/other/../settings?ordinary=yes", "/settings?ordinary=yes", 307},
		{"/assets/storage-more//chunk.js", "/assets/storage-more/chunk.js", 307},
	} {
		t.Run(tc.path, func(t *testing.T) {
			router := NewRouter(slog.New(slog.DiscardHandler))
			router.RejectPathRedirects("/static")
			router.RejectPathRedirects("/assets/storage")
			router.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest("GET", tc.path, nil))
			if response.Code != tc.status || response.Header().Get("Location") != tc.location {
				t.Fatalf("ordinary routing: status=%d location=%q", response.Code, response.Header().Get("Location"))
			}
		})
	}
}
