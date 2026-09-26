package httpadapter_test

import (
	"context"
	"errors"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type fakeLogin struct {
	calls  int
	got    app.LoginInput
	tokens app.Tokens
	err    error
}

func (f *fakeLogin) Execute(_ context.Context, in app.LoginInput) (app.Tokens, error) {
	f.calls++
	f.got = in
	return f.tokens, f.err
}

// fakeRefresh records what it was given and how long its context had left.
type fakeRefresh struct {
	token  string
	ip     netip.Addr
	left   time.Duration
	tokens app.Tokens
	err    error
}

func (f *fakeRefresh) Execute(ctx context.Context, token string, ip netip.Addr) (app.Tokens, error) {
	f.token, f.ip, f.left = token, ip, timeLeft(ctx)
	return f.tokens, f.err
}

type fakeLogout struct {
	token string
	left  time.Duration
	err   error
}

func (f *fakeLogout) Execute(ctx context.Context, token string) error {
	f.token, f.left = token, timeLeft(ctx)
	return f.err
}

// timeLeft is what remains of ctx's deadline; zero without one.
func timeLeft(ctx context.Context) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return 0
	}
	return time.Until(deadline)
}

var sampleTokens = app.Tokens{
	AccessToken: "access", AccessExpiresIn: 15 * time.Minute, RefreshToken: "nrv_rt_x", RefreshExpiresAt: created.Add(720 * time.Hour),
}

const sampleTokensJSON = `{"access_token":"access","access_token_expires_in":900,"refresh_token":"nrv_rt_x",` +
	`"refresh_token_expires_at":"2026-10-25T10:00:00.123456Z","token_type":"Bearer"}` + "\n"

func TestLoginAnswers200WithTheTokens(t *testing.T) {
	login := &fakeLogin{tokens: sampleTokens}
	req := postJSON("/api/v0/auth/login", `{"email":" Alice@Corp.com","password":"Tr0ub4dor&3"}`)
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, fakes{login: login}), req)

	if res.StatusCode != http.StatusOK || body != sampleTokensJSON {
		t.Errorf("POST /auth/login = %d %s, want 200 %s", res.StatusCode, body, sampleTokensJSON)
	}
	// The use case normalizes the address.
	want := app.LoginInput{Email: " Alice@Corp.com", Password: "Tr0ub4dor&3", UserAgent: "agent/1", IP: netip.MustParseAddr("203.0.113.7")}
	if login.got != want {
		t.Errorf("use case got %+v, want %+v", login.got, want)
	}
}

func TestLoginProblems(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		status     int
		code       string
		challenge  string
		retryAfter string
	}{
		{"unknown address or wrong password", domain.ErrInvalidCredentials, 401, "identity.invalid_credentials", "Bearer", ""},
		{"deactivated", domain.ErrAccountDeactivated, 403, "identity.account_deactivated", "", ""},
		{"hashing saturated", shared.ServerBusy(time.Second), 503, "server_busy", "", "1"},
		{"a fault", errors.New("database is down"), 500, "internal_error", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, body := do(t, newServer(t, fakes{login: &fakeLogin{err: tt.err}}), postJSON("/api/v0/auth/login", `{"email":"a@b.co","password":"x"}`))

			if res.StatusCode != tt.status || !strings.Contains(body, `"code":"`+tt.code+`"`) ||
				res.Header.Get("WWW-Authenticate") != tt.challenge || res.Header.Get("Retry-After") != tt.retryAfter {
				t.Errorf("response = %d %s, WWW-Authenticate %q, Retry-After %q; want %d %s, %q, %q", res.StatusCode, body,
					res.Header.Get("WWW-Authenticate"), res.Header.Get("Retry-After"), tt.status, tt.code, tt.challenge, tt.retryAfter)
			}
		})
	}
}

// A3's API version: a login without a password is the platform's 400.
func TestLoginWithoutAPassword(t *testing.T) {
	login := &fakeLogin{}

	res, body := do(t, newServer(t, fakes{login: login}), postJSON("/api/v0/auth/login", `{"email":"a@b.co"}`))

	want := `"errors":[{"field":"password","code":"required","message":"is required"}]`
	if res.StatusCode != http.StatusBadRequest || !strings.Contains(body, want) || login.calls != 0 {
		t.Errorf("response = %d %s after %d logins, want 400 with %s and none", res.StatusCode, body, login.calls, want)
	}
}

// Refresh and logout run within auth.refresh_deadline, shorter than the
// request timeout (M2 design 3.5).
func TestRefreshTokens(t *testing.T) {
	refresh := &fakeRefresh{tokens: sampleTokens}
	req := postJSON("/api/v0/auth/refresh", `{"refresh_token":"nrv_rt_old"}`)
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, fakes{refresh: refresh}), req)

	if res.StatusCode != http.StatusOK || body != sampleTokensJSON {
		t.Errorf("POST /auth/refresh = %d %s, want 200 %s", res.StatusCode, body, sampleTokensJSON)
	}
	if refresh.token != "nrv_rt_old" || refresh.ip != netip.MustParseAddr("203.0.113.7") || refresh.left <= 0 || refresh.left > refreshDeadline {
		t.Errorf("use case got %q from %v with %v left; want the token, the client, at most %v", refresh.token, refresh.ip, refresh.left, refreshDeadline)
	}
}

func TestRefreshTokensInvalid(t *testing.T) {
	res, body := do(t, newServer(t, fakes{refresh: &fakeRefresh{err: domain.ErrRefreshTokenInvalid}}),
		postJSON("/api/v0/auth/refresh", `{"refresh_token":"nrv_rt_old"}`))

	if res.StatusCode != http.StatusUnauthorized || !strings.Contains(body, `"code":"identity.refresh_token_invalid"`) || res.Header.Get("WWW-Authenticate") != "Bearer" {
		t.Errorf("response = %d %s, WWW-Authenticate %q; want 401 identity.refresh_token_invalid with Bearer", res.StatusCode, body, res.Header.Get("WWW-Authenticate"))
	}
}

func TestLogout(t *testing.T) {
	logout := &fakeLogout{}
	req := postJSON("/api/v0/auth/logout", `{"refresh_token":"nrv_rt_current"}`)
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, fakes{logout: logout}), req)

	if res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("POST /auth/logout = %d %q, want 204 and no body", res.StatusCode, body)
	}
	if logout.token != "nrv_rt_current" || logout.left <= 0 || logout.left > refreshDeadline {
		t.Errorf("use case got %q with %v left, want the token and at most %v", logout.token, logout.left, refreshDeadline)
	}
}

func TestLogoutFault(t *testing.T) {
	res, body := do(t, newServer(t, fakes{logout: &fakeLogout{err: errors.New("database is down")}}),
		postJSON("/api/v0/auth/logout", `{"refresh_token":"nrv_rt_current"}`))

	if res.StatusCode != http.StatusInternalServerError || !strings.Contains(body, `"code":"internal_error"`) {
		t.Errorf("response = %d %s, want 500 internal_error", res.StatusCode, body)
	}
}
