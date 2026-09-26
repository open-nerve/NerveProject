package httpserver

import (
	"log/slog"
	"net/http"
	"slices"
	"testing"
)

// A request takes a unit of anonymous by its IP key when it carries no
// credential, else of authenticated by its credential (M2 design 3.10). A
// request authentication turns away takes neither.
func TestRateLimitPicksTheBucketAndKey(t *testing.T) {
	tests := []struct {
		name, route, token string
		status             int
		anonymous          []string
		authenticated      []string
	}{
		{"public operation", "/api/v0/open", "", 204, []string{"203.0.113.7"}, nil},
		{"public operation with a token", "/api/v0/open", "tok", 204, []string{"203.0.113.7"}, nil},
		{"authenticated", "/api/v0/things", "tok", 204, nil, []string{"session:tok"}},
		{"no token", "/api/v0/things", "", 401, nil, nil},
		{"failed credential", "/api/v0/things", "bad", 401, nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testAPIConfig(&fakeAuth{}, slog.New(slog.DiscardHandler))
			anonymous, authenticated := newFakeLimiter(5), newFakeLimiter(5)
			cfg.Anonymous, cfg.Authenticated = anonymous, authenticated
			router, _ := mount(t, buildAPI(t, cfg), slog.New(slog.DiscardHandler))

			rec := serve(router, post(tt.route, tt.token, `{"name":"a"}`))

			if rec.Code != tt.status || !slices.Equal(anonymous.taken, tt.anonymous) || !slices.Equal(authenticated.taken, tt.authenticated) {
				t.Errorf("response %d, anonymous took %q, authenticated took %q; want %d, %q, %q",
					rec.Code, anonymous.taken, authenticated.taken, tt.status, tt.anonymous, tt.authenticated)
			}
		})
	}
}

// Over its rate, a request gets 429 with Retry-After in whole seconds,
// rounded up, before its body is looked at; the log names the bucket.
func TestRateLimitedRequestIs429(t *testing.T) {
	tests := []struct {
		name, route, token, bucket string
	}{
		{"anonymous", "/api/v0/open", "", "anonymous"},
		{"authenticated", "/api/v0/things", "tok", "authenticated"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, logs := captureLogs(t)
			cfg := testAPIConfig(&fakeAuth{}, logger)
			cfg.Anonymous, cfg.Authenticated = newFakeLimiter(0), newFakeLimiter(0)
			router, got := mount(t, buildAPI(t, cfg), slog.New(slog.DiscardHandler))

			rec := serve(router, post(tt.route, tt.token, `{"nope":1}`))

			want := `{"status":429,"code":"rate_limited","title":"Too Many Requests","detail":"Too many requests; retry later."}` + "\n"
			retry := rec.Result().Header.Get("Retry-After")
			if rec.Code != http.StatusTooManyRequests || rec.Body.String() != want || retry != "2" || got.called {
				t.Errorf("response = %d %s, Retry-After %q, handler called %v; want 429 %s, Retry-After 2",
					rec.Code, rec.Body, retry, got.called, want)
			}
			entry := findLog(logs(), "rate limited")
			if entry == nil || entry["level"] != "INFO" || entry["bucket"] != tt.bucket || entry["ip"] != "203.0.113.7" || entry["request_id"] == nil {
				t.Errorf("log = %v, want bucket %s, the client and the request id at info level", entry, tt.bucket)
			}
		})
	}
}
