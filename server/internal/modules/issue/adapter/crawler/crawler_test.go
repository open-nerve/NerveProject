package crawler_test

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func TestCrawlerLimitsAndFailures(t *testing.T) {
	t.Run("body_boundaries", func(t *testing.T) {
		for _, tc := range []struct {
			name               string
			pageSize, iconSize int
			pageOK, iconOK     bool
		}{
			{"page_exact", 1 << 20, 4, true, true}, {"page_over", 1<<20 + 1, 4, false, false},
			{"icon_exact", 64 << 10, 64 << 10, true, true}, {"icon_over", 64 << 10, 64<<10 + 1, true, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				const head = "<title>kept</title>"
				var icons atomic.Int32
				body := strings.Repeat("i", tc.iconSize)
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/page" {
						response(w, 200, "text/html", head+strings.Repeat(" ", tc.pageSize-len(head)))
						return
					}
					icons.Add(1)
					response(w, 200, "image/png", body)
				}))
				defer s.Close()
				target := s.URL + "/page"
				got := newCrawler(t).Crawl(t.Context(), target)
				if !tc.pageOK {
					wantMetadata(t, got, domain.LinkMetadata{"error": value("failed"), "url": value(target)})
					if icons.Load() != 0 {
						t.Error("icon requested after page failure")
					}
					return
				}
				want := success(target, value("kept"))
				if tc.iconOK {
					want["favicon"] = value("data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte(body)))
					want["favicon_url"] = value(s.URL + "/favicon.ico")
				}
				wantMetadata(t, got, want)
				if icons.Load() != 1 {
					t.Errorf("icon attempts=%d", icons.Load())
				}
			})
		}
	})
	t.Run("icon_mime", func(t *testing.T) {
		for _, tc := range []struct{ mime, normalized string }{
			{"image/png", "image/png"}, {"image/x-icon", "image/x-icon"}, {"image/vnd.microsoft.icon", "image/vnd.microsoft.icon"},
			{"image/gif", "image/gif"}, {"image/jpeg", "image/jpeg"}, {"image/webp", "image/webp"},
			{"IMAGE/PNG; charset=binary", "image/png"}, {"image/svg+xml", ""}, {"text/plain", ""}, {"image/png; broken", ""}, {"", ""},
		} {
			t.Run(tc.mime, func(t *testing.T) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/" {
						response(w, 200, "text/html", "<title>kept</title>")
						return
					}
					response(w, 200, tc.mime, "icon")
				}))
				defer s.Close()
				got := newCrawler(t).Crawl(t.Context(), s.URL+"/")
				want := success(s.URL+"/", value("kept"))
				if tc.normalized != "" {
					want["favicon"] = value("data:" + tc.normalized + ";base64,aWNvbg==")
					want["favicon_url"] = value(s.URL + "/favicon.ico")
				}
				wantMetadata(t, got, want)
			})
		}
	})
	t.Run("page_failures", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			status int
			mime   string
		}{{"status", 404, "text/html"}, {"not_html", 200, "application/json"}, {"bad_mime", 200, "text/html; broken"}} {
			t.Run(tc.name, func(t *testing.T) {
				var requests atomic.Int32
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					requests.Add(1)
					response(w, tc.status, tc.mime, "<title>discard</title>")
				}))
				defer s.Close()
				got := newCrawler(t).Crawl(t.Context(), s.URL)
				wantMetadata(t, got, domain.LinkMetadata{"error": value("failed"), "url": value(s.URL)})
				if requests.Load() != 1 {
					t.Errorf("page failure retried: %d", requests.Load())
				}
			})
		}
	})
	t.Run("blocked_before_socket", func(t *testing.T) {
		danger, listener, hits := dangerousFixture(t)
		blocked := strings.Replace(danger.URL, "http://", "http://user:pass@", 1)
		for _, stage := range []string{"page", "icon", "icon_redirect"} {
			t.Run(stage, func(t *testing.T) {
				s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/redirect" {
						w.Header().Set("Location", blocked)
						w.WriteHeader(302)
						return
					}
					href := blocked
					if stage == "icon_redirect" {
						href = "/redirect"
					}
					response(w, 200, "text/html", `<title>kept</title><link rel="icon" href="`+href+`">`)
				}))
				defer s.Close()
				target := s.URL
				if stage == "page" {
					target = blocked
				}
				got := newCrawler(t).Crawl(t.Context(), target)
				want := success(target, value("kept"))
				if stage == "page" {
					want = domain.LinkMetadata{"error": value("blocked"), "url": value(target)}
				}
				wantMetadata(t, got, want)
				if listener.accepted.Load() != 1 || hits.Load() != 1 {
					t.Errorf("dangerous target reached: sockets=%d requests=%d (positive control=1)", listener.accepted.Load(), hits.Load())
				}
			})
		}
	})
	t.Run("one_icon_attempt", func(t *testing.T) {
		var attempts atomic.Int32
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				response(w, 200, "text/html", `<title>kept</title><link rel="icon" href="/first"><link rel="icon" href="/second"><link rel="apple-touch-icon" href="/third">`)
				return
			}
			attempts.Add(1)
			if r.URL.Path != "/first" {
				t.Errorf("unexpected fallback: %s", r.URL.Path)
			}
			response(w, 503, "image/png", "failed")
		}))
		defer s.Close()
		wantMetadata(t, newCrawler(t).Crawl(t.Context(), s.URL+"/"), success(s.URL+"/", value("kept")))
		if attempts.Load() != 1 {
			t.Errorf("icon attempts=%d", attempts.Load())
		}
	})
	t.Run("cancelled_parent", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		wantMetadata(t, newCrawler(t).Crawl(ctx, "http://127.0.0.1/"), domain.LinkMetadata{"error": value("failed"), "url": value("http://127.0.0.1/")})
	})
	t.Run("shared_five_seconds", testSharedBudget)
	t.Run("short_parent_deadline", testParentDeadline)
}
