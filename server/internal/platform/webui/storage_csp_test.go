package webui

import (
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
)

func TestStorageCSP(t *testing.T) {
	files := fstest.MapFS{"index.html": {Data: []byte(scriptedIndex)}, "assets/app.js": {Data: []byte("export {};")}}
	for _, origin := range []string{"", "https://files.example.com", "http://127.0.0.1:8333", "https://my.bucket.s3.example.com:9443"} {
		t.Run(origin, func(t *testing.T) {
			h, err := Handler(files, Options{StorageOrigin: origin})
			if err != nil {
				t.Fatal(err)
			}
			want := scriptedPolicy
			if origin != "" {
				want = strings.Replace(want, "img-src 'self' data: blob:", "img-src 'self' data: blob: "+origin, 1)
				want = strings.Replace(want, "connect-src 'self'", "connect-src 'self' "+origin, 1)
			}
			for _, method := range []string{http.MethodGet, http.MethodHead} {
				got := serve(h, method, "/acme/projects").Header().Get("Content-Security-Policy")
				if got != want {
					t.Errorf("page CSP = %q, want %q", got, want)
				}
			}
			for _, target := range []string{"/assets/app.js", "/assets/missing.js"} {
				if got := serve(h, http.MethodGet, target).Header().Get("Content-Security-Policy"); got != "" {
					t.Errorf("asset/error CSP = %q", got)
				}
			}
		})
	}
	for _, origin := range []string{"http://[::1]:8333", "https://[2001:db8::1]", "*", "https:", "//files.example.com", "https://*.example.com", "https://example.com/", "https://example.com/path", "https://user:secret@example.com", "https://example.com?", "https://example.com#", "https://example.com; connect-src *", "https://example.com https://other.example.com", "https://example.com\n", "https://example.com:0", "https://example.com:65536", "https://example.com:", "https://例子.example", "http://[fe80::1%25lo0]"} {
		t.Run("invalid_"+origin, func(t *testing.T) {
			for _, fs := range []fstest.MapFS{files, {}} {
				h, err := Handler(fs, Options{StorageOrigin: origin})
				if err == nil || h != nil {
					t.Errorf("invalid origin accepted: %q", origin)
				}
			}
		})
	}
}
