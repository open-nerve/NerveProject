package httpadapter_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Every problem code the module's operations declare must be answered by a
// test here (M2 design 3.11), whichever module produces it (M3 design 9.4).
func TestMain(m *testing.M) { apitest.Main(m, "issue") }

var (
	aliceID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	acmeID  = uuid.MustParse("0199a2b4-0000-7000-8000-00000000000a")
	webID   = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000a1")
	created = time.Date(2026, 10, 17, 10, 0, 0, 123456000, time.UTC)
	errGone = errors.New("connection reset")
)

// fakeAuth accepts the token "alice" as her account.
type fakeAuth struct{}

func (fakeAuth) Authenticate(ctx context.Context, token string) (context.Context, string, error) {
	if token == "alice" {
		return shared.WithActor(ctx, shared.Actor{UserID: aliceID, SessionID: uuid.NewV7()}), "session:alice", nil
	}
	return nil, "", shared.Unauthenticated()
}

// fakeCreate is createIssue: each call is recorded as "caller project",
// with what it got; it answers answer, or err.
type fakeCreate struct {
	calls  []string
	got    []domain.NewIssue
	answer domain.Issue
	err    error
}

func (f *fakeCreate) Execute(ctx context.Context, projectID uuid.UUID, in domain.NewIssue) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+projectID.String())
	f.got = append(f.got, in)
	return f.answer, f.err
}

// fakeGet is getIssue: each call is recorded as "caller work item"; it
// answers answer, or err.
type fakeGet struct {
	calls  []string
	answer domain.Issue
	err    error
}

func (f *fakeGet) Execute(ctx context.Context, id uuid.UUID) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	f.calls = append(f.calls, actor.UserID.String()+" "+id.String())
	return f.answer, f.err
}

// newServer serves the module's API over the use cases uc, as bootstrap
// mounts it.
func newServer(t *testing.T, uc httpadapter.UseCases) http.Handler {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)
	router := httpserver.NewRouter(logger)
	limit := ratelimit.New(time.Now).Bucket("test", ratelimit.Rate{PerMinute: 600, Burst: 100})
	api, err := httpserver.NewAPI(httpserver.APIConfig{
		Logger: logger, Authenticator: fakeAuth{}, MaxBodyBytes: 1024, RequestTimeout: 5 * time.Second, IPv6PrefixLen: 64,
		Anonymous: limit, Authenticated: limit, AuthFailure: limit,
	})
	if err != nil {
		t.Fatal(err)
	}
	httpadapter.Register(router, api, uc)
	return router
}

// do serves req and checks the answer against the contract.
func do(t *testing.T, h http.Handler, req *http.Request) (*http.Response, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	res := rec.Result()
	apitest.Load(t).CheckResponse(t, req, res)
	body, _ := io.ReadAll(res.Body)
	return res, string(body)
}

// request is a request with token as its bearer and body as JSON, if any.
func request(method, path, token, body string) *http.Request {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}
