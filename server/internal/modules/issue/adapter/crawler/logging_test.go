package crawler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/crawler"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/outbound"
)

func TestCrawlerLogsRedacted(t *testing.T) {
	for _, tc := range []struct{ name, stage, category, reason string }{
		{"page_connection", "page", "unreachable", ""}, {"page_timeout", "page", "timeout", ""},
		{"page_too_large", "page", "too_large", ""}, {"page_status", "page", "http_status", ""},
		{"page_type", "page", "non_html", ""}, {"page_parse", "page", "non_html", ""}, {"page_userinfo", "page", "blocked", ""},
		{"redirect_blocked", "page", "blocked", ""}, {"redirect_malformed", "page", "blocked", ""},
		{"redirect_limit", "page", "unreachable", "redirect_limit"},
		{"icon_blocked", "icon", "blocked", ""}, {"icon_too_large", "icon", "too_large", ""},
		{"icon_mime", "icon", "invalid_icon_type", ""}, {"icon_status", "icon", "http_status", ""},
		{"icon_timeout", "icon", "timeout", ""}, {"icon_connection", "icon", "unreachable", ""},
		{"icon_redirect_limit", "icon", "unreachable", "redirect_limit"},
		{"icon_redirect_blocked", "icon", "blocked", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const query = "QUERY_SENTINEL"
			const body = "BODY_SENTINEL"
			const location = "LOCATION_SENTINEL"
			const href = "HREF_SENTINEL"
			danger, listener, hits := dangerousFixture(t)
			blocked := strings.Replace(danger.URL, "http://", "http://USER_SENTINEL:PASS_SENTINEL@", 1)
			// A closed local listener produces a genuine connection failure without DNS or external traffic.
			closed := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
			closed.Close()
			var requests atomic.Int32
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if tc.stage == "icon" && r.URL.Path == "/page" {
					icon := "/icon?" + href
					if tc.name == "icon_blocked" {
						icon = blocked + "?" + href
					}
					if tc.name == "icon_connection" {
						icon = closed.URL + "?" + href
					}
					response(w, 200, "text/html", `<title>kept</title><link rel="icon" href="`+icon+`">`+body)
					return
				}
				switch tc.name {
				case "page_timeout", "icon_timeout":
					<-r.Context().Done()
				case "page_too_large":
					response(w, 200, "text/html", body+strings.Repeat("x", 1<<20))
				case "icon_too_large":
					response(w, 200, "image/png", body+strings.Repeat("x", 64<<10))
				case "page_status", "icon_status":
					response(w, 503, "text/html", body)
				case "page_parse":
					response(w, 200, "text/html", strings.Repeat("<div>", 513)+body)
				case "page_type":
					response(w, 200, "text/plain", body)
				case "icon_mime":
					response(w, 200, "image/svg+xml", body)
				case "redirect_blocked", "icon_redirect_blocked":
					w.Header().Set("Location", blocked+"?"+location)
					response(w, 302, "text/html", body)
				case "redirect_malformed":
					w.Header().Set("Location", "/%zz?"+location)
					response(w, 302, "text/html", body)
				case "redirect_limit", "icon_redirect_limit":
					w.Header().Set("Location", "/loop?"+location)
					response(w, 302, "text/html", body)
				default:
					t.Errorf("unexpected real request for %s", tc.name)
					w.WriteHeader(500)
				}
			}))
			defer s.Close()
			target := s.URL + "/page?" + query
			if tc.name == "page_userinfo" {
				target = blocked + "?" + query
			}
			if tc.name == "page_connection" {
				target = closed.URL + "?" + query
			}
			client := localClient(t)
			var rawError string
			if tc.name == "redirect_malformed" {
				_, err := client.Get(t.Context(), target, outbound.Limits{ConnectTimeout: time.Second, Timeout: time.Second, MaxBytes: 1024})
				if err == nil || !strings.Contains(err.Error(), location) {
					t.Fatalf("real wrapped error must contain URL sentinel: %v", err)
				}
				rawError = err.Error()
			}
			var logs bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.Level(math.MinInt)}))
			ctx := t.Context()
			if strings.HasSuffix(tc.name, "timeout") {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 200*time.Millisecond)
				defer cancel()
			}
			got := crawler.New(client, logger).Crawl(ctx, target)
			if tc.stage == "page" {
				public := "failed"
				if tc.category == "blocked" {
					public = "blocked"
				}
				wantMetadata(t, got, domain.LinkMetadata{"error": value(public), "url": value(target)})
			} else {
				wantMetadata(t, got, success(target, value("kept")))
			}
			raw := logs.Bytes()
			secrets := []string{query, "USER_SENTINEL", "PASS_SENTINEL", body, location, href, target, blocked}
			if rawError != "" {
				secrets = append(secrets, rawError)
			}
			for _, secret := range secrets {
				encoded, _ := json.Marshal(secret)
				if bytes.Contains(raw, []byte(secret)) || bytes.Contains(raw, encoded[1:len(encoded)-1]) {
					t.Errorf("raw JSON logs leaked %q: %s", secret, raw)
				}
			}
			lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
			if len(raw) == 0 || len(lines) != 1 {
				t.Fatalf("want one nonempty controlled diagnostic, got %q", raw)
			}
			var entry map[string]any
			if err := json.Unmarshal(lines[0], &entry); err != nil {
				t.Fatal(err)
			}
			if entry["stage"] != tc.stage || entry["category"] != tc.category {
				t.Errorf("wrong diagnostic stage/category: %v", entry)
			}
			if tc.reason == "" {
				if _, ok := entry["reason"]; ok {
					t.Errorf("unexpected reason: %v", entry)
				}
			} else if entry["reason"] != tc.reason {
				t.Errorf("missing controlled reason: %v", entry)
			}
			for key := range entry {
				switch key {
				case "time", "level", "msg", "stage", "category", "reason":
				default:
					t.Errorf("uncontrolled diagnostic field: %s", key)
				}
			}
			if listener.accepted.Load() != 1 || hits.Load() != 1 {
				t.Errorf("blocked destination got real socket/request: sockets=%d requests=%d (control=1)", listener.accepted.Load(), hits.Load())
			}
			wantRequests := int32(1)
			switch tc.name {
			case "page_connection", "page_userinfo":
				wantRequests = 0
			case "redirect_limit":
				wantRequests = 6
			case "redirect_malformed":
				wantRequests = 2 // One independent error-chain probe, one crawl.
			case "icon_redirect_limit":
				wantRequests = 7
			default:
				if tc.stage == "icon" && tc.name != "icon_blocked" && tc.name != "icon_connection" {
					wantRequests = 2
				}
			}
			if requests.Load() != wantRequests {
				t.Errorf("request count=%d want=%d", requests.Load(), wantRequests)
			}
		})
	}
}
