package httpadapter_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Every problem code the module's operations declare must be answered by a
// test here (M2 design 3.11), whichever module produces it (M3 design 9.4).
func TestMain(m *testing.M) { apitest.Main(m, "project") }

var (
	aliceID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	bobID   = uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")
	created = time.Date(2026, 10, 1, 10, 0, 0, 123456000, time.UTC)
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
	list         *fakeList
	create       *fakeCreate
	get          *fakeGet
	check        *fakeCheck
	update       *fakeUpdate
	archive      *fakeOnProject
	unarchive    *fakeOnProject
	delete       *fakeDelete
	prefs        *fakePreferences
	members      *fakeMembers
	add          *fakeAdd
	join         *fakeOnProject
	updateMember *fakeUpdateMember
	removeMember *fakeDelete
	leave        *fakeDelete
	createState  *fakeCreateState
}

type fakeList struct {
	calls []string // "caller slug archived"
	lists map[string][]domain.Project
	err   error
}

func (f *fakeList) Execute(ctx context.Context, slug string, archived bool) ([]domain.Project, error) {
	f.calls = append(f.calls, fmt.Sprintf("%s %s %v", caller(ctx), slug, archived))
	return f.lists[caller(ctx)], f.err
}

type fakeCreate struct {
	calls  []string // "caller slug"
	got    []domain.NewProject
	answer domain.Project
	err    error
}

func (f *fakeCreate) Execute(ctx context.Context, slug string, in domain.NewProject) (domain.Project, error) {
	f.calls = append(f.calls, caller(ctx)+" "+slug)
	f.got = append(f.got, in)
	return f.answer, f.err
}

type fakeGet struct {
	calls    []string // "caller id"
	projects map[string]domain.Project
}

// Execute answers the project the caller's key names, project.not_found
// for any other.
func (f *fakeGet) Execute(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	key := caller(ctx) + " " + id.String()
	f.calls = append(f.calls, key)
	p, ok := f.projects[key]
	if !ok {
		return domain.Project{}, domain.ErrNotFound
	}
	return p, nil
}

type fakeCheck struct {
	calls     []string // "caller slug identifier"
	available map[string]bool
	err       error
}

func (f *fakeCheck) Execute(ctx context.Context, slug, identifier string) (bool, error) {
	f.calls = append(f.calls, caller(ctx)+" "+slug+" "+identifier)
	return f.available[identifier], f.err
}

type fakeUpdate struct {
	calls  []string // "caller id"
	got    []domain.ProjectPatch
	answer domain.Project
	err    error
}

func (f *fakeUpdate) Execute(ctx context.Context, id uuid.UUID, p domain.ProjectPatch) (domain.Project, error) {
	f.calls = append(f.calls, caller(ctx)+" "+id.String())
	f.got = append(f.got, p)
	return f.answer, f.err
}

// fakeOnProject is a use case on a project that takes nothing more.
type fakeOnProject struct {
	calls  []string // "caller id"
	answer domain.Project
	err    error
}

func (f *fakeOnProject) Execute(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	f.calls = append(f.calls, caller(ctx)+" "+id.String())
	return f.answer, f.err
}

type fakeDelete struct {
	calls []string // "caller id"
	err   error
}

func (f *fakeDelete) Execute(ctx context.Context, id uuid.UUID) error {
	f.calls = append(f.calls, caller(ctx)+" "+id.String())
	return f.err
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
	if f.check == nil {
		f.check = &fakeCheck{}
	}
	if f.update == nil {
		f.update = &fakeUpdate{}
	}
	if f.archive == nil {
		f.archive = &fakeOnProject{}
	}
	if f.unarchive == nil {
		f.unarchive = &fakeOnProject{}
	}
	if f.delete == nil {
		f.delete = &fakeDelete{}
	}
	if f.prefs == nil {
		f.prefs = &fakePreferences{}
	}
	if f.members == nil {
		f.members = &fakeMembers{}
	}
	if f.add == nil {
		f.add = &fakeAdd{}
	}
	if f.join == nil {
		f.join = &fakeOnProject{}
	}
	if f.updateMember == nil {
		f.updateMember = &fakeUpdateMember{}
	}
	if f.removeMember == nil {
		f.removeMember = &fakeDelete{}
	}
	if f.leave == nil {
		f.leave = &fakeDelete{}
	}
	if f.createState == nil {
		f.createState = &fakeCreateState{}
	}
	httpadapter.Register(router, api, httpadapter.UseCases{ListProjects: f.list, CreateProject: f.create, GetProject: f.get,
		CheckIdentifier: f.check, UpdateProject: f.update, ArchiveProject: f.archive, UnarchiveProject: f.unarchive, DeleteProject: f.delete,
		GetPreferences: f.prefs, UpdatePreferences: fakeUpdatePreferences{f.prefs}, ListMembers: f.members, AddMembers: f.add,
		JoinProject: f.join, UpdateMember: f.updateMember, RemoveMember: f.removeMember, LeaveProject: f.leave,
		CreateState: f.createState})
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
