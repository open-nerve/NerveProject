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

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Every problem code the module's operations declare must be answered by a
// test here (M2 design 3.11), whichever module produces it (M3 design 9.4).
func TestMain(m *testing.M) { apitest.Main(m, "notification") }

var (
	aliceID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	bobID   = uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")
	acmeID  = uuid.MustParse("0199a2b4-0000-7000-8000-00000000000a")
	webID   = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000a1")
	fixID   = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000b1")
	noteID  = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000c1")
	sent    = time.Date(2026, 11, 18, 10, 0, 0, 123456000, time.UTC)
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

// The paths of the module's operations in acme, and of alice's note.
const (
	listPath   = "/api/v0/workspaces/acme/notifications"
	countPath  = "/api/v0/workspaces/acme/notifications/unread-count"
	markPath   = "/api/v0/workspaces/acme/notifications/mark-all-read"
	updatePath = "/api/v0/notifications/0199a2b4-0000-7000-8000-0000000000c1"
)

// The answers of the refusals every operation shares: a busy server
// (shared's server_busy, with Retry-After: 1) and a failure.
const (
	busy   = `{"status":503,"code":"server_busy","title":"Service Unavailable","detail":"The server is busy; retry shortly."}`
	failed = `{"status":500,"code":"internal_error","title":"Internal Server Error"}`
)

// actorOf is the caller of ctx, the zero id when there is none.
func actorOf(ctx context.Context) uuid.UUID {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return uuid.UUID{}
	}
	return actor.UserID
}

// listCall is what fakeList was asked.
type listCall struct {
	Caller uuid.UUID
	Slug   string
	Query  domain.ListQuery
	Limit  *int
	Cursor *string
}

// fakeList is listNotifications' use case: it records each call and
// answers answer, or err.
type fakeList struct {
	calls  []listCall
	answer app.NotificationPage
	err    error
}

func (f *fakeList) Execute(ctx context.Context, slug string, q domain.ListQuery, limit *int, cursor *string) (app.NotificationPage, error) {
	f.calls = append(f.calls, listCall{Caller: actorOf(ctx), Slug: slug, Query: q, Limit: limit, Cursor: cursor})
	return f.answer, f.err
}

// countCall is what fakeCount was asked.
type countCall struct {
	Caller uuid.UUID
	Slug   string
}

// fakeCount is getNotificationUnreadCount's use case: it records each call
// and answers answer, or err.
type fakeCount struct {
	calls  []countCall
	answer app.UnreadCounts
	err    error
}

func (f *fakeCount) Execute(ctx context.Context, slug string) (app.UnreadCounts, error) {
	f.calls = append(f.calls, countCall{Caller: actorOf(ctx), Slug: slug})
	return f.answer, f.err
}

// markCall is what fakeMark was asked.
type markCall struct {
	Caller uuid.UUID
	Slug   string
	Values domain.FilterValues
}

// fakeMark is markAllNotificationsRead's use case: it records each call and
// answers answer, or err.
type fakeMark struct {
	calls  []markCall
	answer int
	err    error
}

func (f *fakeMark) Execute(ctx context.Context, slug string, v domain.FilterValues) (int, error) {
	f.calls = append(f.calls, markCall{Caller: actorOf(ctx), Slug: slug, Values: v})
	return f.answer, f.err
}

// updateCall is what fakeUpdate was asked.
type updateCall struct {
	Caller uuid.UUID
	ID     uuid.UUID
	Patch  domain.Patch
}

// fakeUpdate is updateNotification's use case: it records each call and
// answers answer, or err.
type fakeUpdate struct {
	calls  []updateCall
	answer app.NotificationView
	err    error
}

func (f *fakeUpdate) Execute(ctx context.Context, id uuid.UUID, p domain.Patch) (app.NotificationView, error) {
	f.calls = append(f.calls, updateCall{Caller: actorOf(ctx), ID: id, Patch: p})
	return f.answer, f.err
}

// fakes are the four use cases, each a fake that answers nothing.
type fakes struct {
	list   *fakeList
	count  *fakeCount
	mark   *fakeMark
	update *fakeUpdate
}

// newFakes are fakes, each a use case that answers its zero value.
func newFakes() fakes {
	return fakes{list: &fakeList{}, count: &fakeCount{}, mark: &fakeMark{}, update: &fakeUpdate{}}
}

// useCases are the fakes as the module's use cases.
func (f fakes) useCases() httpadapter.UseCases {
	return httpadapter.UseCases{ListNotifications: f.list, GetUnreadCount: f.count, MarkAllRead: f.mark, UpdateNotification: f.update}
}

// asked is how many calls the four fakes got in all.
func (f fakes) asked() int {
	return len(f.list.calls) + len(f.count.calls) + len(f.mark.calls) + len(f.update.calls)
}
