package httpadapter_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Every problem code the module's operations declare must be answered by a
// test here (M2 design 3.11).
func TestMain(m *testing.M) { apitest.Main(m, "identity") }

var (
	userID    = uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	sessionID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")
	created   = time.Date(2026, 9, 25, 10, 0, 0, 123456000, time.UTC)
)

type fakeRegister struct {
	got    app.RegisterInput
	tokens app.Tokens
	err    error
}

func (f *fakeRegister) Execute(_ context.Context, in app.RegisterInput) (app.Tokens, error) {
	f.got = in
	return f.tokens, f.err
}

type fakeGetMe struct{}

func (fakeGetMe) Execute(ctx context.Context) (domain.User, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.User{}, err
	}
	return domain.User{ID: actor.UserID, Email: "alice@corp.com", DisplayName: "alice", Timezone: "UTC", CreatedAt: created}, nil
}

// fakeAuth accepts the token "valid" as the account userID.
type fakeAuth struct{}

func (fakeAuth) Authenticate(ctx context.Context, token string) (context.Context, string, error) {
	if token != "valid" {
		return nil, "", shared.Unauthenticated()
	}
	return shared.WithActor(ctx, shared.Actor{UserID: userID, SessionID: sessionID}), "session:" + sessionID.String(), nil
}

func newServer(t *testing.T, register *fakeRegister) http.Handler {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)
	router := httpserver.NewRouter(logger)
	limit := ratelimit.New(time.Now).Bucket("test", ratelimit.Rate{PerMinute: 600, Burst: 100})
	api, err := httpserver.NewAPI(httpserver.APIConfig{
		Logger:           logger,
		Authenticator:    fakeAuth{},
		PublicOperations: httpadapter.PublicOperations(),
		MaxBodyBytes:     1024,
		RequestTimeout:   5 * time.Second,
		IPv6PrefixLen:    64,
		Anonymous:        limit,
		Authenticated:    limit,
		AuthFailure:      limit,
	})
	if err != nil {
		t.Fatal(err)
	}
	httpadapter.Register(router, api, httpadapter.UseCases{Register: register, GetMe: fakeGetMe{}})
	return router
}

func do(t *testing.T, h http.Handler, req *http.Request) (*http.Response, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	res := rec.Result()
	apitest.Load(t).CheckResponse(t, req, res)
	body, _ := io.ReadAll(res.Body)
	return res, string(body)
}

func registerRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/v0/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "agent/1")
	req.RemoteAddr = "203.0.113.7:5555"
	return req
}

func TestRegisterAnswers201WithTheTokens(t *testing.T) {
	register := &fakeRegister{tokens: app.Tokens{
		AccessToken: "access", AccessExpiresIn: 15 * time.Minute, RefreshToken: "nrv_rt_x", RefreshExpiresAt: created.Add(720 * time.Hour),
	}}
	req := registerRequest(`{"email":"Alice@Corp.com","password":"Tr0ub4dor&3"}`)
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, register), req)

	want := `{"access_token":"access","access_token_expires_in":900,"refresh_token":"nrv_rt_x",` +
		`"refresh_token_expires_at":"2026-10-25T10:00:00.123456Z","token_type":"Bearer"}` + "\n"
	if res.StatusCode != http.StatusCreated || body != want {
		t.Errorf("POST /auth/register = %d %s, want 201 %s", res.StatusCode, body, want)
	}
	wantIn := app.RegisterInput{Email: "Alice@Corp.com", Password: "Tr0ub4dor&3", UserAgent: "agent/1", IP: netip.MustParseAddr("203.0.113.7")}
	if register.got != wantIn {
		t.Errorf("use case got %+v, want %+v", register.got, wantIn)
	}
}

// The handler exit: every error the use case returns becomes its problem.
func TestRegisterProblems(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		status     int
		code       string
		retryAfter string
	}{
		{"sign-up off", domain.ErrSignupDisabled, 403, "identity.signup_disabled", ""},
		{"invalid values", shared.Invalid(shared.FieldError{Field: "password", Code: shared.FieldCommonPassword, Message: "is too common"}), 422, "validation_failed", ""},
		{"address taken", domain.ErrEmailTaken, 409, "identity.email_taken", ""},
		{"hashing saturated", shared.ServerBusy(time.Second), 503, "server_busy", "1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, body := do(t, newServer(t, &fakeRegister{err: tt.err}), registerRequest(`{"email":"a@b.co","password":"x"}`))

			if res.StatusCode != tt.status || !strings.Contains(body, `"code":"`+tt.code+`"`) || res.Header.Get("Retry-After") != tt.retryAfter {
				t.Errorf("response = %d %s Retry-After %q, want %d %s", res.StatusCode, body, res.Header.Get("Retry-After"), tt.status, tt.code)
			}
		})
	}
}

// The body decoding exit (M0-P3 handoff 2): the structure check answers
// with fields, anything else with a generic detail, never a Go type name.
func TestRegisterBodyProblems(t *testing.T) {
	tests := []struct {
		name, body string
		status     int
		want       string
	}{
		{"unknown and missing fields", `{"email":"a@b.co","extra":1}`, 400,
			`"errors":[{"field":"extra","code":"not_allowed","message":"is not a property of this request"},{"field":"password","code":"required","message":"is required"}]`},
		{"null for a string", `{"email":null,"password":"x"}`, 400, `"errors":[{"field":"email","code":"invalid_format"`},
		{"not JSON", `{"email":`, 400, `"detail":"The request body could not be decoded."`},
		{"empty", ``, 400, `"detail":"The request body could not be decoded."`},
		{"too large", `{"email":"` + strings.Repeat("a", 2000) + `"}`, 413, `"code":"payload_too_large"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			register := &fakeRegister{}
			res, body := do(t, newServer(t, register), registerRequest(tt.body))

			if res.StatusCode != tt.status || !strings.Contains(body, tt.want) || strings.Contains(body, "Go struct") {
				t.Errorf("response = %d %s, want %d with %s", res.StatusCode, body, tt.status, tt.want)
			}
			if register.got != (app.RegisterInput{}) {
				t.Errorf("the use case ran with %+v", register.got)
			}
		})
	}
}

func TestGetMe(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v0/me", nil)
	req.Header.Set("Authorization", "Bearer valid")
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, &fakeRegister{}), req)

	want := `{"avatar_url":null,"cover_image_url":null,"created_at":"2026-09-25T10:00:00.123456Z","display_name":"alice",` +
		`"email":"alice@corp.com","first_name":"","id":"` + userID.String() + `","last_name":"","user_timezone":"UTC"}` + "\n"
	if res.StatusCode != http.StatusOK || body != want {
		t.Errorf("GET /me = %d %s, want 200 %s", res.StatusCode, body, want)
	}
}

func TestGetMeWithoutAValidToken(t *testing.T) {
	for _, header := range []string{"", "Bearer forged"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v0/me", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}

		res, body := do(t, newServer(t, &fakeRegister{}), req)

		if res.StatusCode != http.StatusUnauthorized || !strings.Contains(body, `"code":"unauthorized"`) {
			t.Errorf("GET /me with %q = %d %s, want 401 unauthorized", header, res.StatusCode, body)
		}
	}
}
