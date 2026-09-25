package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/bodyshape"
)

const (
	privateRoute = "POST /api/v0/things"
	publicRoute  = "POST /api/v0/open"
)

type callerKey struct{}

// fakeAuth accepts every token except "bad" (401), "expired" (401 with
// ExpiredCredential) and "boom" (a fault).
type fakeAuth struct {
	calls int
	ctx   context.Context // what Authenticate was given
	token string
}

func (f *fakeAuth) Authenticate(ctx context.Context, token string) (context.Context, string, error) {
	f.calls++
	f.ctx, f.token = ctx, token
	switch token {
	case "bad":
		return nil, "", problemErr{status: http.StatusUnauthorized, code: "unauthorized", detail: "Authentication is required."}
	case "expired":
		return nil, "", fmt.Errorf("check token: %w", expiredErr{problemErr{status: http.StatusUnauthorized, code: "unauthorized", detail: "expired"}})
	case "boom":
		return nil, "", errors.New("database is down")
	}
	return context.WithValue(ctx, callerKey{}, "caller-"+token), "session:" + token, nil
}

// expiredErr is the 401 of an access token whose exp has passed.
type expiredErr struct{ problemErr }

func (expiredErr) ExpiredCredential() bool { return true }

// fakeLimiter gives each key burst units and never refills them. It records
// the keys it took a unit of and the keys it got one back for.
type fakeLimiter struct {
	mu      sync.Mutex
	burst   int
	retry   time.Duration
	used    map[string]int
	taken   []string
	refunds []string
}

func newFakeLimiter(burst int) *fakeLimiter {
	return &fakeLimiter{burst: burst, retry: 1500 * time.Millisecond, used: map[string]int{}}
}

func (f *fakeLimiter) take(key string) (time.Duration, bool) {
	if f.used[key] == f.burst {
		return f.retry, false
	}
	f.used[key]++
	f.taken = append(f.taken, key)
	return 0, true
}

func (f *fakeLimiter) Allow(key string) (time.Duration, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.take(key)
}

func (f *fakeLimiter) Reserve(key string) (func(), time.Duration, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if retry, ok := f.take(key); !ok {
		return nil, retry, false
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.used[key]--
			f.refunds = append(f.refunds, key)
		})
	}, 0, true
}

// left is what key has of its burst.
func (f *fakeLimiter) left(key string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.burst - f.used[key]
}

// thingBody is a table as bodyshapegen writes it: {"name": string}, closed.
func thingBody() *bodyshape.Table {
	return &bodyshape.Table{
		Nodes: []bodyshape.Node{
			{Types: bodyshape.Object, Extra: bodyshape.Closed, Items: bodyshape.Open, Props: map[string]int{"name": 1}, Required: []string{"name"}},
			{Types: bodyshape.String, Extra: bodyshape.Open, Items: bodyshape.Open},
		},
		Roots: map[string]int{privateRoute: 0, publicRoute: 0},
	}
}

type reached struct {
	called bool
	ctx    context.Context
	body   string
}

// mount registers h on a router behind api's middlewares, applied the way
// the generated code does: for each middleware in the list, h = m(h).
func mount(t *testing.T, api *API, logger *slog.Logger) (*Router, *reached) {
	t.Helper()
	got := &reached{}
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.called, got.ctx = true, r.Context()
		data, err := io.ReadAll(r.Body)
		if err != nil {
			api.Errors.BodyError(w, r, err)
			return
		}
		got.body = string(data)
		w.WriteHeader(http.StatusNoContent)
	})
	for _, m := range api.Middlewares(thingBody()) {
		h = m(h)
	}
	router := NewRouter(logger)
	router.Handle(privateRoute, h)
	router.Handle(publicRoute, h)
	return router, got
}

// testAPIConfig trusts the proxies of fd00::/8 and has limiters that never
// run out in these tests.
func testAPIConfig(auth Authenticator, logger *slog.Logger) APIConfig {
	return APIConfig{
		Logger:           logger,
		Authenticator:    auth,
		PublicOperations: []string{publicRoute},
		MaxBodyBytes:     64,
		RequestTimeout:   2 * time.Second,
		TrustedProxies:   []netip.Prefix{netip.MustParsePrefix("fd00::/8")},
		IPv6PrefixLen:    64,
		Anonymous:        newFakeLimiter(100),
		Authenticated:    newFakeLimiter(100),
		AuthFailure:      newFakeLimiter(100),
	}
}

func buildAPI(t *testing.T, cfg APIConfig) *API {
	t.Helper()
	api, err := NewAPI(cfg)
	if err != nil {
		t.Fatalf("NewAPI() error = %v", err)
	}
	return api
}

func newTestAPI(t *testing.T, auth Authenticator, logger *slog.Logger) *API {
	t.Helper()
	return buildAPI(t, testAPIConfig(auth, logger))
}

func post(path, token, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	r.RemoteAddr = "203.0.113.7:5555"
	r.Header.Set("User-Agent", "agent/1.0")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return r
}

func decodeProblem(t *testing.T, rec *httptest.ResponseRecorder) Problem {
	t.Helper()
	var p Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil {
		t.Fatalf("body %q: %v", rec.Body, err)
	}
	return p
}

func TestNewAPIRequiresItsDependencies(t *testing.T) {
	_, err := NewAPI(APIConfig{PublicOperations: []string{publicRoute}, MaxBodyBytes: 64})
	want := "httpserver: APIConfig lacks Logger, Authenticator, Anonymous, Authenticated, AuthFailure"
	if err == nil || err.Error() != want {
		t.Errorf("NewAPI(without dependencies) error = %v, want %q", err, want)
	}
	cfg := testAPIConfig(&fakeAuth{}, slog.New(slog.DiscardHandler))
	cfg.AuthFailure = nil
	if _, err := NewAPI(cfg); err == nil || err.Error() != "httpserver: APIConfig lacks AuthFailure" {
		t.Errorf("NewAPI(without AuthFailure) error = %v, want it named alone", err)
	}
}

// The authenticator already sees the request meta and the deadline: both run
// before authentication (M2 design 3.6).
func TestMetaAndDeadlineRunBeforeAuthentication(t *testing.T) {
	auth := &fakeAuth{}
	router, got := mount(t, newTestAPI(t, auth, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "tok", `{"name":"a"}`))

	if rec.Code != http.StatusNoContent || !got.called {
		t.Fatalf("status = %d, handler called %v; want 204", rec.Code, got.called)
	}
	meta := RequestMetaFrom(auth.ctx)
	if meta.ClientIP != netip.MustParseAddr("203.0.113.7") || meta.UserAgent != "agent/1.0" {
		t.Errorf("meta seen by the authenticator = %+v", meta)
	}
	deadline, ok := auth.ctx.Deadline()
	if !ok || time.Until(deadline) > 2*time.Second {
		t.Errorf("authenticator's deadline = %v, %v; want within the 2s request timeout", deadline, ok)
	}
	if got.ctx.Value(callerKey{}) != "caller-tok" || got.body != `{"name":"a"}` {
		t.Errorf("handler saw caller %v and body %q; want the authenticator's context and the body unchanged", got.ctx.Value(callerKey{}), got.body)
	}
}

// Without a token the body is never looked at: authentication runs before
// the body structure check.
func TestAuthenticationRunsBeforeTheBodyCheck(t *testing.T) {
	router, got := mount(t, newTestAPI(t, &fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "", `{"nope":1}`))

	if rec.Code != http.StatusUnauthorized || got.called {
		t.Errorf("status = %d, handler called %v; want 401 before the body is checked", rec.Code, got.called)
	}
}

// The body check reads the body through the body limit: the limit runs
// before it, and the check's own read is what fails.
func TestBodyLimitRunsBeforeTheBodyCheck(t *testing.T) {
	router, got := mount(t, newTestAPI(t, &fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "tok", `{"name":"`+strings.Repeat("a", 100)+`"}`))

	if p := decodeProblem(t, rec); rec.Code != http.StatusRequestEntityTooLarge || p.Code != CodePayloadTooLarge || got.called {
		t.Errorf("response = %d %+v, handler called %v; want 413 payload_too_large", rec.Code, p, got.called)
	}
}

func TestBodyCheckAnswersEveryProblemAs400(t *testing.T) {
	router, got := mount(t, newTestAPI(t, &fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "tok", `{"extra":1}`))

	want := `{"status":400,"code":"bad_request","title":"Bad Request","detail":"The request body does not match the API description.",` +
		`"errors":[{"field":"extra","code":"not_allowed","message":"is not a property of this request"},{"field":"name","code":"required","message":"is required"}]}` + "\n"
	if rec.Code != http.StatusBadRequest || rec.Body.String() != want || got.called {
		t.Errorf("response = %d %s, handler called %v; want 400 %s", rec.Code, rec.Body, got.called, want)
	}
}

func TestBodyThatIsNotJSONIs400WithAGenericDetail(t *testing.T) {
	router, _ := mount(t, newTestAPI(t, &fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "tok", `{"name":`))

	if p := decodeProblem(t, rec); rec.Code != http.StatusBadRequest || p.Detail != "The request body could not be decoded." || len(p.Errors) != 0 {
		t.Errorf("response = %d %+v, want 400 with the generic detail", rec.Code, p)
	}
}

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

			if rec.Code != tt.status || rec.Header().Get("WWW-Authenticate") != tt.challenge || auth.calls != tt.authCalls {
				t.Errorf("response = %d, WWW-Authenticate %q, authenticator called %d times; want %d, %q, %d",
					rec.Code, rec.Header().Get("WWW-Authenticate"), auth.calls, tt.status, tt.challenge, tt.authCalls)
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

		if p := decodeProblem(t, rec); rec.Code != http.StatusTooManyRequests || p.Code != CodeRateLimited || rec.Header().Get("Retry-After") != "2" {
			t.Errorf("token %s: response = %d %+v, Retry-After %q; want 429 rate_limited, Retry-After 2", token, rec.Code, p, rec.Header().Get("Retry-After"))
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
			if rec.Code != http.StatusTooManyRequests || rec.Body.String() != want || rec.Header().Get("Retry-After") != "2" || got.called {
				t.Errorf("response = %d %s, Retry-After %q, handler called %v; want 429 %s, Retry-After 2",
					rec.Code, rec.Body, rec.Header().Get("Retry-After"), got.called, want)
			}
			entry := findLog(logs(), "rate limited")
			if entry == nil || entry["level"] != "INFO" || entry["bucket"] != tt.bucket || entry["ip"] != "203.0.113.7" || entry["request_id"] == nil {
				t.Errorf("log = %v, want bucket %s, the client and the request id at info level", entry, tt.bucket)
			}
		})
	}
}

func TestRequestMetaClientIP(t *testing.T) {
	tests := []struct{ remote, want string }{
		{"203.0.113.7:5555", "203.0.113.7"},
		{"[2001:db8::1]:443", "2001:db8::1"},
		{"[::ffff:192.0.2.1]:80", "192.0.2.1"},
		{"[fe80::1%en0]:80", "fe80::1"},
	}
	for _, tt := range tests {
		auth := &fakeAuth{}
		router, _ := mount(t, newTestAPI(t, auth, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))
		req := post("/api/v0/things", "tok", `{"name":"a"}`)
		req.RemoteAddr = tt.remote

		serve(router, req)

		if got := RequestMetaFrom(auth.ctx).ClientIP; got != netip.MustParseAddr(tt.want) {
			t.Errorf("RemoteAddr %s: ClientIP = %v, want %s", tt.remote, got, tt.want)
		}
	}
}

func TestRequestMetaOutsideTheMiddlewaresIsZero(t *testing.T) {
	if m := RequestMetaFrom(context.Background()); m.ClientIP.IsValid() || m.IPKey != "" || m.UserAgent != "" {
		t.Errorf("RequestMetaFrom() = %+v, want the zero value", m)
	}
}
