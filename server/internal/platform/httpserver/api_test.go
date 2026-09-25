package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/bodyshape"
)

const (
	privateRoute = "POST /api/v0/things"
	publicRoute  = "POST /api/v0/open"
)

type callerKey struct{}

// fakeAuth accepts every token except "bad" (401) and "boom" (a fault).
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
	case "boom":
		return nil, "", errors.New("database is down")
	}
	return context.WithValue(ctx, callerKey{}, "caller-"+token), "session:" + token, nil
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

func newTestAPI(auth Authenticator, logger *slog.Logger) *API {
	return NewAPI(APIConfig{
		Logger:           logger,
		Authenticator:    auth,
		PublicOperations: []string{publicRoute},
		MaxBodyBytes:     64,
		RequestTimeout:   2 * time.Second,
		IPv6PrefixLen:    64,
	})
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

// The authenticator already sees the request meta and the deadline: both run
// before authentication (M2 design 3.6).
func TestMetaAndDeadlineRunBeforeAuthentication(t *testing.T) {
	auth := &fakeAuth{}
	router, got := mount(t, newTestAPI(auth, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

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
	router, got := mount(t, newTestAPI(&fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "", `{"nope":1}`))

	if rec.Code != http.StatusUnauthorized || got.called {
		t.Errorf("status = %d, handler called %v; want 401 before the body is checked", rec.Code, got.called)
	}
}

// The body check reads the body through the body limit: the limit runs
// before it, and the check's own read is what fails.
func TestBodyLimitRunsBeforeTheBodyCheck(t *testing.T) {
	router, got := mount(t, newTestAPI(&fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "tok", `{"name":"`+strings.Repeat("a", 100)+`"}`))

	if p := decodeProblem(t, rec); rec.Code != http.StatusRequestEntityTooLarge || p.Code != CodePayloadTooLarge || got.called {
		t.Errorf("response = %d %+v, handler called %v; want 413 payload_too_large", rec.Code, p, got.called)
	}
}

func TestBodyCheckAnswersEveryProblemAs400(t *testing.T) {
	router, got := mount(t, newTestAPI(&fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "tok", `{"extra":1}`))

	want := `{"status":400,"code":"bad_request","title":"Bad Request","detail":"The request body does not match the API description.",` +
		`"errors":[{"field":"extra","code":"not_allowed","message":"is not a property of this request"},{"field":"name","code":"required","message":"is required"}]}` + "\n"
	if rec.Code != http.StatusBadRequest || rec.Body.String() != want || got.called {
		t.Errorf("response = %d %s, handler called %v; want 400 %s", rec.Code, rec.Body, got.called, want)
	}
}

func TestBodyThatIsNotJSONIs400WithAGenericDetail(t *testing.T) {
	router, _ := mount(t, newTestAPI(&fakeAuth{}, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))

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
		{"valid token", "/api/v0/things", "Bearer tok", 204, "", 1},
		{"scheme in lower case", "/api/v0/things", "bearer tok", 204, "", 1},
		{"authenticator fault", "/api/v0/things", "Bearer boom", 500, "", 1},
		{"public without a token", "/api/v0/open", "", 204, "", 0},
		{"public ignores a bad token", "/api/v0/open", "Bearer bad", 204, "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &fakeAuth{}
			router, _ := mount(t, newTestAPI(auth, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))
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
	router, _ := mount(t, newTestAPI(&fakeAuth{}, logger), slog.New(slog.DiscardHandler))

	rec := serve(router, post("/api/v0/things", "bad", `{"name":"a"}`))

	entry := findLog(logs(), "authentication failed")
	if entry == nil || entry["level"] != "DEBUG" || entry["route"] != privateRoute || entry["error"] != "Authentication is required." {
		t.Errorf("log = %v, want the reason at debug level", entry)
	}
	if strings.Contains(rec.Body.String(), "Authentication is required.") {
		t.Errorf("body %s carries the authenticator's reason", rec.Body)
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
		router, _ := mount(t, newTestAPI(auth, slog.New(slog.DiscardHandler)), slog.New(slog.DiscardHandler))
		req := post("/api/v0/things", "tok", `{"name":"a"}`)
		req.RemoteAddr = tt.remote

		serve(router, req)

		if got := RequestMetaFrom(auth.ctx).ClientIP; got != netip.MustParseAddr(tt.want) {
			t.Errorf("RemoteAddr %s: ClientIP = %v, want %s", tt.remote, got, tt.want)
		}
	}
}

func TestRequestMetaOutsideTheMiddlewaresIsZero(t *testing.T) {
	if m := RequestMetaFrom(context.Background()); m.ClientIP.IsValid() || m.UserAgent != "" {
		t.Errorf("RequestMetaFrom() = %+v, want the zero value", m)
	}
}
