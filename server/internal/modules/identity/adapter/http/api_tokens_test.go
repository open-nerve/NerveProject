package httpadapter_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var tokenID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000003")

type fakeListTokens struct {
	calls  int
	limit  *int
	cursor *string
	page   app.APITokenPage
	err    error
}

func (f *fakeListTokens) Execute(_ context.Context, limit *int, cursor *string) (app.APITokenPage, error) {
	f.calls++
	f.limit, f.cursor = limit, cursor
	return f.page, f.err
}

type fakeCreateToken struct {
	calls   int
	spec    domain.APITokenSpec
	created app.CreatedAPIToken
	err     error
}

func (f *fakeCreateToken) Execute(_ context.Context, spec domain.APITokenSpec) (app.CreatedAPIToken, error) {
	f.calls++
	f.spec = spec
	return f.created, f.err
}

type fakeRevokeToken struct {
	calls int
	id    uuid.UUID
	err   error
}

func (f *fakeRevokeToken) Execute(_ context.Context, id uuid.UUID) error {
	f.calls++
	f.id = id
	return f.err
}

// withToken is req with the bearer token fakeAuth accepts.
func withToken(req *http.Request) *http.Request {
	req.Header.Set("Authorization", "Bearer valid")
	return req
}

func TestCreateAPITokenAnswers201WithTheToken(t *testing.T) {
	expires := created.Add(7 * 24 * time.Hour)
	create := &fakeCreateToken{created: app.CreatedAPIToken{
		APIToken: domain.APIToken{ID: tokenID, Label: "deploy", Description: "ci", ExpiredAt: &expires, CreatedAt: created},
		Token:    "nrv_pat_x",
	}}
	req := withToken(postJSON("/api/v0/me/api-tokens", `{"label":"deploy","description":"ci","expired_at":"2026-10-02T10:00:00.123456Z"}`))
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, fakes{createToken: create}), req)

	want := `{"created_at":"2026-09-25T10:00:00.123456Z","description":"ci","expired_at":"2026-10-02T10:00:00.123456Z",` +
		`"id":"` + tokenID.String() + `","label":"deploy","last_used":null,"token":"nrv_pat_x"}` + "\n"
	if res.StatusCode != http.StatusCreated || body != want {
		t.Errorf("POST /me/api-tokens = %d %s, want 201 %s", res.StatusCode, body, want)
	}
	if s := create.spec; s.Label == nil || *s.Label != "deploy" || s.Description != "ci" || s.ExpiredAt == nil || !s.ExpiredAt.Equal(expires) {
		t.Errorf("use case got %+v, want the label, description and expiry sent", s)
	}
}

// Absent fields and a null expiry reach the use case as absent: a
// generated label, no description, no expiry.
func TestCreateAPITokenWithoutFields(t *testing.T) {
	for _, body := range []string{`{}`, `{"expired_at":null}`} {
		create := &fakeCreateToken{created: app.CreatedAPIToken{APIToken: domain.APIToken{ID: tokenID, Label: "l", CreatedAt: created}, Token: "nrv_pat_x"}}

		res, out := do(t, newServer(t, fakes{createToken: create}), withToken(postJSON("/api/v0/me/api-tokens", body)))

		if res.StatusCode != http.StatusCreated || !strings.Contains(out, `"expired_at":null`) {
			t.Errorf("POST %s = %d %s, want 201 with a null expiry", body, res.StatusCode, out)
		}
		if s := create.spec; s.Label != nil || s.Description != "" || s.ExpiredAt != nil {
			t.Errorf("POST %s: use case got %+v, want nothing set", body, s)
		}
	}
}

func TestCreateAPITokenInvalid(t *testing.T) {
	create := &fakeCreateToken{err: shared.Invalid(shared.FieldError{Field: "expired_at", Code: shared.FieldMustBeFuture, Message: "must be in the future"})}

	res, body := do(t, newServer(t, fakes{createToken: create}), withToken(postJSON("/api/v0/me/api-tokens", `{"expired_at":"2020-01-01T00:00:00Z"}`)))

	if res.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(body, `"field":"expired_at","code":"must_be_future"`) {
		t.Errorf("response = %d %s, want 422 on expired_at", res.StatusCode, body)
	}
}

func TestListAPITokens(t *testing.T) {
	used := created.Add(time.Hour)
	list := &fakeListTokens{page: app.APITokenPage{
		Tokens:     []domain.APIToken{{ID: tokenID, Label: "deploy", LastUsed: &used, CreatedAt: created}},
		NextCursor: "next",
	}}
	req := withToken(httptest.NewRequest(http.MethodGet, "/api/v0/me/api-tokens?limit=2&cursor=abc", nil))
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, fakes{listTokens: list}), req)

	want := `{"data":[{"created_at":"2026-09-25T10:00:00.123456Z","description":"","expired_at":null,"id":"` + tokenID.String() +
		`","label":"deploy","last_used":"2026-09-25T11:00:00.123456Z"}],"next_cursor":"next"}` + "\n"
	if res.StatusCode != http.StatusOK || body != want {
		t.Errorf("GET /me/api-tokens = %d %s, want 200 %s", res.StatusCode, body, want)
	}
	if list.limit == nil || *list.limit != 2 || list.cursor == nil || *list.cursor != "abc" {
		t.Errorf("use case got limit %v cursor %v, want 2 and abc", list.limit, list.cursor)
	}
}

// Without parameters the use case gets neither; the last page's cursor is
// null.
func TestListAPITokensFirstAndLastPage(t *testing.T) {
	list := &fakeListTokens{page: app.APITokenPage{Tokens: []domain.APIToken{}}}

	res, body := do(t, newServer(t, fakes{listTokens: list}), withToken(httptest.NewRequest(http.MethodGet, "/api/v0/me/api-tokens", nil)))

	if want := `{"data":[],"next_cursor":null}` + "\n"; res.StatusCode != http.StatusOK || body != want {
		t.Errorf("GET /me/api-tokens = %d %s, want 200 %s", res.StatusCode, body, want)
	}
	if list.limit != nil || list.cursor != nil {
		t.Errorf("use case got limit %v cursor %v, want neither", list.limit, list.cursor)
	}
}

// The handler exit: limit out of range is 422, a foreign cursor 400.
func TestListAPITokensProblems(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"limit out of range", shared.Invalid(shared.FieldError{Field: "limit", Code: shared.FieldOutOfRange, Message: "must be between 1 and 100"}),
			422, `"code":"validation_failed"`},
		{"foreign cursor", shared.InvalidCursor(), 400, `"errors":[{"field":"cursor","code":"invalid_format"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, body := do(t, newServer(t, fakes{listTokens: &fakeListTokens{err: tt.err}}),
				withToken(httptest.NewRequest(http.MethodGet, "/api/v0/me/api-tokens?limit=0", nil)))

			if res.StatusCode != tt.status || !strings.Contains(body, tt.want) {
				t.Errorf("response = %d %s, want %d with %s", res.StatusCode, body, tt.status, tt.want)
			}
		})
	}
}

func TestRevokeAPIToken(t *testing.T) {
	revoke := &fakeRevokeToken{}
	req := withToken(httptest.NewRequest(http.MethodDelete, "/api/v0/api-tokens/"+tokenID.String(), nil))
	apitest.Load(t).CheckRequest(t, req)

	res, body := do(t, newServer(t, fakes{revokeToken: revoke}), req)

	if res.StatusCode != http.StatusNoContent || body != "" || revoke.id != tokenID {
		t.Errorf("DELETE /api-tokens/{id} = %d %q, use case got %v; want 204 for %v", res.StatusCode, body, revoke.id, tokenID)
	}
}

func TestRevokeAnAPITokenThatIsNotTheCallers(t *testing.T) {
	revoke := &fakeRevokeToken{err: domain.ErrAPITokenNotFound}

	res, body := do(t, newServer(t, fakes{revokeToken: revoke}), withToken(httptest.NewRequest(http.MethodDelete, "/api/v0/api-tokens/"+tokenID.String(), nil)))

	if res.StatusCode != http.StatusNotFound || !strings.Contains(body, `"code":"identity.api_token_not_found"`) {
		t.Errorf("response = %d %s, want 404 identity.api_token_not_found", res.StatusCode, body)
	}
}

// The parameter binding exit (M0-P3 handoff 2): a parameter that does not
// bind is 400 with the parameter as the field, before authentication (M2
// design 3.6) and without Go's words; the use case never runs.
func TestParametersThatDoNotBind(t *testing.T) {
	tests := []struct {
		name, method, target, field string
	}{
		{"limit not an integer", http.MethodGet, "/api/v0/me/api-tokens?limit=abc", "limit"},
		{"token_id not a uuid", http.MethodDelete, "/api/v0/api-tokens/not-a-uuid", "token_id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, revoke := &fakeListTokens{}, &fakeRevokeToken{}

			res, body := do(t, newServer(t, fakes{listTokens: list, revokeToken: revoke}), httptest.NewRequest(tt.method, tt.target, nil))

			want := `{"status":400,"code":"bad_request","title":"Bad Request","detail":"The request parameters do not match the API description.",` +
				`"errors":[{"field":"` + tt.field + `","code":"invalid_format","message":"has the wrong type or format"}]}` + "\n"
			if res.StatusCode != http.StatusBadRequest || body != want {
				t.Errorf("response = %d %s, want 400 %s", res.StatusCode, body, want)
			}
			if list.calls+revoke.calls != 0 {
				t.Errorf("the use case ran %d times, want never", list.calls+revoke.calls)
			}
		})
	}
}
