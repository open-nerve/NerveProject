package httpadapter_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Every problem code the module's operations declare must be answered by a
// test here (M2 design 3.11), whichever module produces it (M3 design 9.4).
func TestMain(m *testing.M) { apitest.Main(m, "navigation") }

var (
	aliceID  = uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	bobID    = uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")
	acmeID   = uuid.MustParse("0199a2b4-0000-7000-8000-00000000000a")
	webID    = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000a1")
	boardID  = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000e1")
	sheetID  = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000e2")
	created  = time.Date(2026, 11, 20, 10, 0, 0, 123456000, time.UTC)
	changed  = created.Add(time.Hour)
	errGone  = errors.New("connection reset")
	viewPath = "/api/v0/views/0199a2b4-0000-7000-8000-0000000000e1"
)

// The paths of the module's collections: Web's views, and acme's own.
const (
	projectPath   = "/api/v0/projects/0199a2b4-0000-7000-8000-0000000000a1/views"
	workspacePath = "/api/v0/workspaces/acme/views"
)

// fakeAuth accepts the token "alice" as her account.
type fakeAuth struct{}

func (fakeAuth) Authenticate(ctx context.Context, token string) (context.Context, string, error) {
	if token == "alice" {
		return shared.WithActor(ctx, shared.Actor{UserID: aliceID, SessionID: uuid.NewV7()}), "session:alice", nil
	}
	return nil, "", shared.Unauthenticated()
}

// newServer serves the module's API over the use cases uc, as bootstrap
// mounts it.
func newServer(t *testing.T, uc httpadapter.UseCases) http.Handler {
	t.Helper()
	logger := slog.New(slog.DiscardHandler)
	router := httpserver.NewRouter(logger)
	limit := ratelimit.New(time.Now).Bucket("test", ratelimit.Rate{PerMinute: 600, Burst: 100})
	api, err := httpserver.NewAPI(httpserver.APIConfig{
		Logger: logger, Authenticator: fakeAuth{}, MaxBodyBytes: 4096, RequestTimeout: 5 * time.Second, IPv6PrefixLen: 64,
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

// request is alice's request, with body as JSON, if any.
func request(method, path, body string) *http.Request {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer alice")
	return req
}

// sameJSON tells whether got and want are the same JSON value: each key
// there or not alike, null apart from [] and from a key left out.
func sameJSON(t *testing.T, got, want string) bool {
	t.Helper()
	var g, w any
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("the wanted %s: %v", want, err)
	}
	return json.Unmarshal([]byte(got), &g) == nil && reflect.DeepEqual(g, w)
}

// call is what a use case was asked: the caller, the operation, and what
// it names and gives.
type call struct {
	Caller uuid.UUID
	Op     string
	ID     uuid.UUID
	Slug   string
	New    domain.NewView
	Patch  domain.Patch
}

// fakes are the module's use cases, each a fake that records its call and
// answers list, view, or err.
type fakes struct {
	calls []call
	list  []domain.View
	view  domain.View
	err   error
}

// record keeps c, asked by the caller of ctx (the zero id when none).
func (f *fakes) record(ctx context.Context, c call) {
	if actor, err := shared.RequireActor(ctx); err == nil {
		c.Caller = actor.UserID
	}
	f.calls = append(f.calls, c)
}

type (
	fakeListProject   struct{ *fakes }
	fakeListWorkspace struct{ *fakes }
	fakeGet           struct{ *fakes }
	fakeCreateProject struct{ *fakes }
	fakeCreateSpace   struct{ *fakes }
	fakeUpdate        struct{ *fakes }
	fakeDelete        struct{ *fakes }
)

func (f fakeListProject) Execute(ctx context.Context, id uuid.UUID) ([]domain.View, error) {
	f.record(ctx, call{Op: "listProjectViews", ID: id})
	return f.list, f.err
}

func (f fakeListWorkspace) Execute(ctx context.Context, slug string) ([]domain.View, error) {
	f.record(ctx, call{Op: "listWorkspaceViews", Slug: slug})
	return f.list, f.err
}

func (f fakeGet) Execute(ctx context.Context, id uuid.UUID) (domain.View, error) {
	f.record(ctx, call{Op: "getView", ID: id})
	return f.view, f.err
}

func (f fakeCreateProject) Execute(ctx context.Context, id uuid.UUID, n domain.NewView) (domain.View, error) {
	f.record(ctx, call{Op: "createProjectView", ID: id, New: n})
	return f.view, f.err
}

func (f fakeCreateSpace) Execute(ctx context.Context, slug string, n domain.NewView) (domain.View, error) {
	f.record(ctx, call{Op: "createWorkspaceView", Slug: slug, New: n})
	return f.view, f.err
}

func (f fakeUpdate) Execute(ctx context.Context, id uuid.UUID, p domain.Patch) (domain.View, error) {
	f.record(ctx, call{Op: "updateView", ID: id, Patch: p})
	return f.view, f.err
}

func (f fakeDelete) Execute(ctx context.Context, id uuid.UUID) error {
	f.record(ctx, call{Op: "deleteView", ID: id})
	return f.err
}

// useCases are f as the module's use cases.
func (f *fakes) useCases() httpadapter.UseCases {
	return httpadapter.UseCases{ListProjectViews: fakeListProject{f}, ListWorkspaceViews: fakeListWorkspace{f}, GetView: fakeGet{f},
		CreateProjectView: fakeCreateProject{f}, CreateWorkspaceView: fakeCreateSpace{f}, UpdateView: fakeUpdate{f}, DeleteView: fakeDelete{f}}
}
