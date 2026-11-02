package httpserver

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
)

func TestTransferMiddlewarePreservesClientScope(t *testing.T) {
	for _, tc := range []struct{ name, remote, forwarded, secondRemote, secondForwarded, ip, key string }{
		{"untrusted forwarding", "203.0.113.7:1234", "198.51.100.2", "203.0.113.7:4321", "198.51.100.3", "203.0.113.7", "203.0.113.7"},
		{"trusted forwarding", "[fd00::1]:1234", "198.51.100.2, fd00::2", "[fd00::2]:4321", "198.51.100.2", "198.51.100.2", "198.51.100.2"},
		{"mapped IPv4", "[::ffff:192.0.2.1]:1234", "", "192.0.2.1:4321", "", "192.0.2.1", "192.0.2.1"},
		{"IPv6 prefix", "[2001:db8::1]:1234", "", "[2001:db8::2]:4321", "", "2001:db8::1", "2001:db8::/64"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger := slog.New(slog.DiscardHandler)
			auth := &fakeAuth{}
			cfg := testAPIConfig(auth, logger)
			cfg.Anonymous = newFakeLimiter(0)
			cfg.Authenticated = newFakeLimiter(0)
			cfg.AuthFailure = newFakeLimiter(0)
			api := buildAPI(t, cfg)
			body := strings.Repeat("x", 100) // Larger than this API's 64-byte JSON limit.
			h := api.TransferMiddleware(newFakeLimiter(1))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				meta := RequestMetaFrom(r.Context())
				if meta.ClientIP.String() != tc.ip || meta.IPKey != tc.key || meta.UserAgent != "agent/1.0" {
					t.Errorf("metadata = %+v", meta)
				}
				if _, ok := r.Context().Deadline(); ok {
					t.Error("transfer acquired the JSON deadline")
				}
				got, err := io.ReadAll(r.Body)
				if err != nil || string(got) != body {
					t.Errorf("raw body = %q, %v", got, err)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			first := post("/storage/local/key", "bad", body)
			first.RemoteAddr = tc.remote
			first.Header.Set("X-Forwarded-For", tc.forwarded)
			if response := serve(h, first); response.Code != 204 {
				t.Fatalf("first transfer = %d %s", response.Code, response.Body)
			}
			second := post("/storage/local/other", "bad", body)
			second.RemoteAddr = tc.secondRemote
			second.Header.Set("X-Forwarded-For", tc.secondForwarded)
			response := serve(h, second)
			if response.Code != 429 || decodeProblem(t, response).Code != CodeRateLimited || response.Header().Get("Retry-After") != "2" || response.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("shared bucket = %d %s %v", response.Code, response.Body, response.Header())
			}
			if auth.calls != 0 {
				t.Fatalf("raw route authenticated %d times", auth.calls)
			}
		})
	}
}
