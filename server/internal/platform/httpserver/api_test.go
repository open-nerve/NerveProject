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
	want := "httpserver: APIConfig lacks Logger, Authenticator, Anonymous, Authenticated, AuthFailure; has IPv6PrefixLen 0, outside 1-128"
	if err == nil || err.Error() != want {
		t.Errorf("NewAPI(without dependencies) error = %v, want %q", err, want)
	}
	cfg := testAPIConfig(&fakeAuth{}, slog.New(slog.DiscardHandler))
	cfg.AuthFailure = nil
	if _, err := NewAPI(cfg); err == nil || err.Error() != "httpserver: APIConfig lacks AuthFailure" {
		t.Errorf("NewAPI(without AuthFailure) error = %v, want it named alone", err)
	}
}

// A prefix length that cannot key an IPv6 client is refused at startup: a
// bootstrap that forgets ratelimit.ipv6_prefix_len does not start.
func TestNewAPIRequiresAnIPv6PrefixLenFrom1To128(t *testing.T) {
	for _, n := range []int{0, 129} {
		cfg := testAPIConfig(&fakeAuth{}, slog.New(slog.DiscardHandler))
		cfg.IPv6PrefixLen = n
		want := fmt.Sprintf("httpserver: APIConfig has IPv6PrefixLen %d, outside 1-128", n)
		if _, err := NewAPI(cfg); err == nil || err.Error() != want {
			t.Errorf("NewAPI(IPv6PrefixLen %d) error = %v, want %q", n, err, want)
		}
	}
	for _, n := range []int{1, 128} {
		cfg := testAPIConfig(&fakeAuth{}, slog.New(slog.DiscardHandler))
		cfg.IPv6PrefixLen = n
		if _, err := NewAPI(cfg); err != nil {
			t.Errorf("NewAPI(IPv6PrefixLen %d) error = %v, want none", n, err)
		}
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
