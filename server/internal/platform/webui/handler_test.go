package webui

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

const indexHTML = "<!doctype html><title>Nerve</title>"

// built mirrors the layout of web/apps/web/build/client.
var built = fstest.MapFS{
	".gitkeep":                  {},
	"index.html":                {Data: []byte(indexHTML)},
	"site.webmanifest.json":     {Data: []byte(`{"name":"Nerve"}`)},
	"icons/icon-192x192.png":    {Data: []byte("\x89PNG\r\n\x1a\n")},
	"assets/entry.client-a1.js": {Data: []byte("export {};")},
	"assets/globals-b2.css":     {Data: []byte("body{}")},
	"assets/.hidden":            {Data: []byte("secret")},
}

func serve(h http.Handler, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

// wantResponse checks status, Content-Type, Cache-Control and body.
func wantResponse(t *testing.T, rec *httptest.ResponseRecorder, target string, status int, contentType, cacheControl, body string) {
	t.Helper()
	if rec.Code != status {
		t.Errorf("%s: status = %d, want %d", target, rec.Code, status)
	}
	if got := rec.Header().Get("Content-Type"); got != contentType {
		t.Errorf("%s: Content-Type = %q, want %q", target, got, contentType)
	}
	if got := rec.Header().Get("Cache-Control"); got != cacheControl {
		t.Errorf("%s: Cache-Control = %q, want %q", target, got, cacheControl)
	}
	if got := rec.Body.String(); got != body {
		t.Errorf("%s: body = %q, want %q", target, got, body)
	}
}

func TestServesIndexAtRoot(t *testing.T) {
	rec := serve(Handler(built), http.MethodGet, "/")

	wantResponse(t, rec, "/", http.StatusOK, "text/html; charset=utf-8", "no-cache", indexHTML)
}

func TestPagePathsFallBackToIndex(t *testing.T) {
	h := Handler(built)
	for _, target := range []string{
		"/sign-up",
		"/acme/projects/0199f1c2-7a1b-7c3d-8e4f-5a6b7c8d9e0f/issues",
		"/acme/settings/",
		"/icons",         // a directory, not a file
		"/.gitkeep",      // hidden files are never served
		"/favicon.ico",   // no such file outside assets/
		"/acme?tab=home", // the query does not matter
	} {
		wantResponse(t, serve(h, http.MethodGet, target), target, http.StatusOK, "text/html; charset=utf-8", "no-cache", indexHTML)
	}
}

func TestServesFilesWithCachePolicy(t *testing.T) {
	h := Handler(built)
	tests := []struct {
		target, contentType, cacheControl, body string
	}{
		// Vite content-hashes everything under assets/: a changed file gets a new name.
		{"/assets/entry.client-a1.js", "text/javascript; charset=utf-8", "public, max-age=31536000, immutable", "export {};"},
		{"/assets/globals-b2.css", "text/css; charset=utf-8", "public, max-age=31536000, immutable", "body{}"},
		// Files copied from public/ keep their names, so they are revalidated.
		{"/site.webmanifest.json", "application/json", "no-cache", `{"name":"Nerve"}`},
		{"/icons/icon-192x192.png", "image/png", "no-cache", "\x89PNG\r\n\x1a\n"},
	}
	for _, tt := range tests {
		wantResponse(t, serve(h, http.MethodGet, tt.target), tt.target, http.StatusOK, tt.contentType, tt.cacheControl, tt.body)
	}
}

// A missing hashed file (for example a chunk of an older build) must fail as
// a missing file: answering index.html would hand HTML to a script loader.
func TestMissingAssetsAre404(t *testing.T) {
	h := Handler(built)
	for _, target := range []string{"/assets/entry.client-old.js", "/assets/", "/assets", "/assets/.hidden"} {
		rec := serve(h, http.MethodGet, target)
		if rec.Code != http.StatusNotFound || strings.Contains(rec.Body.String(), indexHTML) {
			t.Errorf("GET %s = %d %q, want 404 without index.html", target, rec.Code, rec.Body)
		}
	}
}

func TestIndexHTMLRedirectsToRoot(t *testing.T) {
	rec := serve(Handler(built), http.MethodGet, "/index.html")

	if rec.Code != http.StatusMovedPermanently || rec.Header().Get("Location") != "./" {
		t.Errorf("GET /index.html = %d Location %q, want 301 to ./", rec.Code, rec.Header().Get("Location"))
	}
}

func TestHeadHasHeadersButNoBody(t *testing.T) {
	h := Handler(built)
	for _, target := range []string{"/", "/acme/projects", "/assets/entry.client-a1.js"} {
		rec := serve(h, http.MethodHead, target)
		if rec.Code != http.StatusOK || rec.Body.Len() != 0 || rec.Header().Get("Content-Length") == "" {
			t.Errorf("HEAD %s = %d, body %d bytes, Content-Length %q; want 200, no body, a length",
				target, rec.Code, rec.Body.Len(), rec.Header().Get("Content-Length"))
		}
	}
}

func TestOnlyGetAndHeadAreAllowed(t *testing.T) {
	for _, files := range []fs.FS{built, fstest.MapFS{}} {
		h := Handler(files)
		for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions} {
			rec := serve(h, method, "/assets/entry.client-a1.js")
			if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "GET, HEAD" {
				t.Errorf("%s = %d Allow %q, want 405 Allow \"GET, HEAD\"", method, rec.Code, rec.Header().Get("Allow"))
			}
		}
	}
}

// Without index.html (dist/ holds only .gitkeep) every page explains how to
// get the frontend.
func TestUnbuiltFrontendExplainsItself(t *testing.T) {
	h := Handler(fstest.MapFS{".gitkeep": {}})
	for _, target := range []string{"/", "/acme/projects", "/assets/entry.client-a1.js"} {
		wantResponse(t, serve(h, http.MethodGet, target), target, http.StatusNotFound, "text/plain; charset=utf-8", "no-cache", notBuiltMessage+"\n")
	}
}
