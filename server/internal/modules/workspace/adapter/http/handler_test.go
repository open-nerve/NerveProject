package httpadapter_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Every problem code the module's operations declare must be answered by a
// test here (M2 design 3.11), whichever module produces it (M3 design 9.4).
func TestMain(m *testing.M) { apitest.Main(m, "workspace") }

var (
	aliceID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	bobID   = uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")
	created = time.Date(2026, 9, 29, 10, 0, 0, 123456000, time.UTC)
)

// fakeAuth accepts the tokens "alice" and "bob" as those accounts.
type fakeAuth struct{}

func (fakeAuth) Authenticate(ctx context.Context, token string) (context.Context, string, error) {
	switch token {
	case "alice":
		return shared.WithActor(ctx, shared.Actor{UserID: aliceID, SessionID: uuid.NewV7()}), "session:alice", nil
	case "bob":
		return shared.WithActor(ctx, shared.Actor{UserID: bobID, SessionID: uuid.NewV7()}), "session:bob", nil
	}
	return nil, "", shared.Unauthenticated()
}

// caller is the account the context's actor is, by name.
func caller(ctx context.Context) string {
	actor, err := shared.RequireActor(ctx)
	switch {
	case err != nil:
		return "nobody"
	case actor.UserID == aliceID:
		return "alice"
	case actor.UserID == bobID:
		return "bob"
	}
	return "someone else"
}

// fakes are the use cases behind a test server: each records who called it
// with what, and answers what it is given.
type fakes struct {
	list   *fakeList
	create *fakeCreate
	get    *fakeGet
	update *fakeUpdate
	del    *fakeDelete
	check  *fakeCheck
	prefs  *fakePrefs
}

type fakeList struct {
	calls []string
	lists map[string][]domain.Workspace
}

func (f *fakeList) Execute(ctx context.Context) ([]domain.Workspace, error) {
	f.calls = append(f.calls, caller(ctx))
	return f.lists[caller(ctx)], nil
}

type fakeCreate struct {
	callers []string
	got     []domain.NewWorkspace
	answer  domain.Workspace
	err     error
}

func (f *fakeCreate) Execute(ctx context.Context, w domain.NewWorkspace) (domain.Workspace, error) {
	f.callers = append(f.callers, caller(ctx))
	f.got = append(f.got, w)
	return f.answer, f.err
}

type fakeGet struct {
	calls      []string
	workspaces map[string]domain.Workspace // "caller slug" → workspace
}

func (f *fakeGet) Execute(ctx context.Context, slug string) (domain.Workspace, error) {
	key := caller(ctx) + " " + slug
	f.calls = append(f.calls, key)
	w, ok := f.workspaces[key]
	if !ok {
		return domain.Workspace{}, domain.ErrNotFound
	}
	return w, nil
}

type fakeUpdate struct {
	calls  []string // "caller slug"
	got    []domain.WorkspacePatch
	answer domain.Workspace
	err    error
}

func (f *fakeUpdate) Execute(ctx context.Context, slug string, p domain.WorkspacePatch) (domain.Workspace, error) {
	f.calls = append(f.calls, caller(ctx)+" "+slug)
	f.got = append(f.got, p)
	return f.answer, f.err
}

type fakeDelete struct {
	calls []string // "caller slug"
	err   error
}

func (f *fakeDelete) Execute(ctx context.Context, slug string) error {
	f.calls = append(f.calls, caller(ctx)+" "+slug)
	return f.err
}

// fakePrefs is both preference use cases: each call is recorded as
// "caller slug", and a PATCH's patch; it answers what it is given.
type fakePrefs struct {
	calls   []string
	patches []domain.PreferencesPatch
	answer  domain.Preferences
	err     error
}

type fakeGetPrefs struct{ *fakePrefs }

func (f fakeGetPrefs) Execute(ctx context.Context, slug string) (domain.Preferences, error) {
	f.calls = append(f.calls, "GET "+caller(ctx)+" "+slug)
	return f.answer, f.err
}

type fakeUpdatePrefs struct{ *fakePrefs }

func (f fakeUpdatePrefs) Execute(ctx context.Context, slug string, p domain.PreferencesPatch) (domain.Preferences, error) {
	f.calls = append(f.calls, "PATCH "+caller(ctx)+" "+slug)
	f.patches = append(f.patches, p)
	return f.answer, f.err
}

type fakeCheck struct {
	calls   []string
	reasons map[string]domain.SlugReason
}

func (f *fakeCheck) Execute(ctx context.Context, slug string) (domain.SlugReason, error) {
	f.calls = append(f.calls, caller(ctx)+" "+slug)
	return f.reasons[slug], nil
}

// newServer serves the module with f; a fake left nil is an idle one.
func newServer(t *testing.T, f fakes) http.Handler {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)
	router := httpserver.NewRouter(logger)
	limit := ratelimit.New(time.Now).Bucket("test", ratelimit.Rate{PerMinute: 600, Burst: 100})
	api, err := httpserver.NewAPI(httpserver.APIConfig{
		Logger:         logger,
		Authenticator:  fakeAuth{},
		MaxBodyBytes:   1024,
		RequestTimeout: 5 * time.Second,
		IPv6PrefixLen:  64,
		Anonymous:      limit,
		Authenticated:  limit,
		AuthFailure:    limit,
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.list == nil {
		f.list = &fakeList{}
	}
	if f.create == nil {
		f.create = &fakeCreate{}
	}
	if f.get == nil {
		f.get = &fakeGet{}
	}
	if f.update == nil {
		f.update = &fakeUpdate{}
	}
	if f.del == nil {
		f.del = &fakeDelete{}
	}
	if f.check == nil {
		f.check = &fakeCheck{}
	}
	if f.prefs == nil {
		f.prefs = &fakePrefs{}
	}
	httpadapter.Register(router, api, httpadapter.UseCases{
		ListWorkspaces: f.list, CreateWorkspace: f.create, GetWorkspace: f.get, UpdateWorkspace: f.update, DeleteWorkspace: f.del, CheckSlug: f.check,
		GetPreferences: fakeGetPrefs{f.prefs}, UpdatePreferences: fakeUpdatePrefs{f.prefs},
	})
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
