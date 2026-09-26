package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAuthenticationDeniesByDefault(t *testing.T) {
	tests := []struct {
		name          string
		route, header string
		status        int
		challenge     string
		authCalls     int
	}{
		{"no token", "/api/v0/things", "", 401, "Bearer", 0},
		{"another scheme", "/api/v0/things", "Basic dXNlcjpwYXNz", 401, "Bearer", 0},
		{"empty bearer", "/api/v0/things", "Bearer ", 401, "Bearer", 0},
		{"invalid token", "/api/v0/things", "Bearer bad", 401, `Bearer error="invalid_token"`, 1},
		{"expired token", "/api/v0/things", "Bearer expired", 401, `Bearer error="invalid_token"`, 1},
		{"valid token", "/api/v0/things", "Bearer tok", 204, "", 1},
		{"scheme in lower case", "/api/v0/things", "bearer tok", 204, "", 1},
		{"authenticator fault", "/api/v0/things", "Bearer boom", 500, "", 1},
		{"public without a token", "/api/v0/open", "", 204, "", 0},
		{"public ignores a bad token", "/api/v0/open", "Bearer bad", 204, "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &fakeAuth{}
			router, _ := mount(t, newTestAPI(t, auth, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))
			req := post(tt.route, "", `{"name":"a"}`)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			rec := serve(router, req)

			challenge := rec.Result().Header.Get("WWW-Authenticate")
			if rec.Code != tt.status || challenge != tt.challenge || auth.calls != tt.authCalls {
				t.Errorf("response = %d, WWW-Authenticate %q, authenticator called %d times; want %d, %q, %d",
					rec.Code, challenge, auth.calls, tt.status, tt.challenge, tt.authCalls)
			}
			if tt.status == 401 {
				if p := decodeProblem(t, rec); p.Code != CodeUnauthorized {
					t.Errorf("problem code = %q, want unauthorized", p.Code)
				}
			}
		})
	}
}

// Why a credential failed goes to the debug log, never into the response.
func TestAuthenticationFailureIsLoggedAtDebugLevel(t *testing.T) {
	logger, logs := captureLogs(t)
	router, _ := mount(t, newTestAPI(t, &fakeAuth{}, logger), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "bad", `{"name":"a"}`))

	entry := findLog(logs(), "authentication failed")
	if entry == nil || entry["level"] != "DEBUG" || entry["route"] != privateRoute || entry["error"] != "Authentication is required." {
		t.Errorf("log = %v, want the reason at debug level", entry)
	}
	if strings.Contains(rec.Body.String(), "Authentication is required.") {
		t.Errorf("body %s carries the authenticator's reason", rec.Body)
	}
}

// The failure gate reserves a unit before the authenticator runs; only a
// credential that fails keeps it (M2 design 3.6).
func TestFailureGateKeepsTheUnitOfAFailedCredentialOnly(t *testing.T) {
	tests := []struct {
		name, route, token string
		status             int
		left               int // of the client's 3 units afterwards
		refunded           bool
	}{
		{"failed credential", "/api/v0/things", "bad", 401, 2, false},
		{"valid credential", "/api/v0/things", "tok", 204, 3, true},
		{"expired access token", "/api/v0/things", "expired", 401, 3, true},
		{"authenticator fault", "/api/v0/things", "boom", 500, 3, true},
		{"no token", "/api/v0/things", "", 401, 3, false},
		{"public operation", "/api/v0/open", "bad", 204, 3, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testAPIConfig(&fakeAuth{}, slog.New(slog.DiscardHandler))
			gate := newFakeLimiter(3)
			cfg.AuthFailure = gate
			router, _ := mount(t, buildAPI(t, cfg), slog.New(slog.DiscardHandler))

			rec := serve(router, post(tt.route, tt.token, `{"name":"a"}`))

			if rec.Code != tt.status || gate.left("203.0.113.7") != tt.left || (len(gate.refunds) == 1) != tt.refunded {
				t.Errorf("response %d, units left %d, refunds %q; want %d, %d, refunded %v",
					rec.Code, gate.left("203.0.113.7"), gate.refunds, tt.status, tt.left, tt.refunded)
			}
		})
	}
}

// An empty gate answers 429 without running the authenticator, for every
// token from that client, valid ones too; other clients are not affected.
func TestFailureGateTurnsAwayWithoutAuthenticating(t *testing.T) {
	logger, logs := captureLogs(t)
	auth := &fakeAuth{}
	cfg := testAPIConfig(auth, logger)
	cfg.AuthFailure = newFakeLimiter(2)
	router, got := mount(t, buildAPI(t, cfg), slog.New(slog.DiscardHandler))
	for range 2 {
		serve(router, post("/api/v0/things", "bad", `{"name":"a"}`))
	}

	for _, token := range []string{"bad", "tok"} {
		rec := serve(router, post("/api/v0/things", token, `{"name":"a"}`))

		retry := rec.Result().Header.Get("Retry-After")
		if p := decodeProblem(t, rec); rec.Code != http.StatusTooManyRequests || p.Code != CodeRateLimited || retry != "2" {
			t.Errorf("token %s: response = %d %+v, Retry-After %q; want 429 rate_limited, Retry-After 2", token, rec.Code, p, retry)
		}
	}
	if auth.calls != 2 || got.called {
		t.Errorf("authenticator called %d times, handler called %v; want 2 and false", auth.calls, got.called)
	}
	entry := findLog(logs(), "rate limited")
	if entry == nil || entry["level"] != "INFO" || entry["bucket"] != "auth_failure" || entry["ip"] != "203.0.113.7" {
		t.Errorf("log = %v, want the auth_failure bucket and the client at info level", entry)
	}

	other := post("/api/v0/things", "tok", `{"name":"a"}`)
	other.RemoteAddr = "198.51.100.1:5555"
	if rec := serve(router, other); rec.Code != http.StatusNoContent {
		t.Errorf("another client: status = %d, want 204", rec.Code)
	}
}

// Reserving before authenticating holds concurrent failures to the burst:
// 50 invalid tokens at once, 3 units, 3 authentications.
func TestFailureGateHoldsUnderConcurrency(t *testing.T) {
	auth := &slowFailingAuth{}
	cfg := testAPIConfig(auth, slog.New(slog.DiscardHandler))
	cfg.AuthFailure = newFakeLimiter(3)
	router, _ := mount(t, buildAPI(t, cfg), slog.New(slog.DiscardHandler))

	var wg sync.WaitGroup
	var mu sync.Mutex
	codes := map[int]int{}
	for range 50 {
		wg.Go(func() {
			rec := serve(router, post("/api/v0/things", "bad", `{"name":"a"}`))
			mu.Lock()
			defer mu.Unlock()
			codes[rec.Code]++
		})
	}
	wg.Wait()

	if auth.calls() != 3 || codes[http.StatusUnauthorized] != 3 || codes[http.StatusTooManyRequests] != 47 {
		t.Errorf("authenticator called %d times, responses %v; want 3, three 401 and 47 429", auth.calls(), codes)
	}
}

// slowFailingAuth rejects every token, slowly enough that concurrent
// requests are inside it together.
type slowFailingAuth struct {
	mu sync.Mutex
	n  int
}

func (s *slowFailingAuth) Authenticate(context.Context, string) (context.Context, string, error) {
	s.mu.Lock()
	s.n++
	s.mu.Unlock()
	time.Sleep(20 * time.Millisecond)
	return nil, "", problemErr{status: http.StatusUnauthorized, code: "unauthorized", detail: "no such session"}
}

func (s *slowFailingAuth) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.n
}

// The gate counts a client by its IP key: two addresses of one IPv6 /64 are
// one client.
func TestFailureGateCountsByTheIPKey(t *testing.T) {
	cfg := testAPIConfig(&fakeAuth{}, slog.New(slog.DiscardHandler))
	gate := newFakeLimiter(1)
	cfg.AuthFailure = gate
	router, _ := mount(t, buildAPI(t, cfg), slog.New(slog.DiscardHandler))
	first := post("/api/v0/things", "bad", `{"name":"a"}`)
	first.RemoteAddr = "[2001:db8:1:2::7]:443"
	second := post("/api/v0/things", "bad", `{"name":"a"}`)
	second.RemoteAddr = "[2001:db8:1:2::8]:443"

	serve(router, first)
	rec := serve(router, second)

	if rec.Code != http.StatusTooManyRequests || len(gate.taken) != 1 || gate.taken[0] != "2001:db8:1:2::/64" {
		t.Errorf("second address: status %d, units taken for %q; want 429 and one unit of 2001:db8:1:2::/64", rec.Code, gate.taken)
	}
}
