package bootstrap

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

	"github.com/jackc/pgx/v5/pgxpool"

	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// tokenIsAccount accepts an account's id as its bearer token.
type tokenIsAccount struct{}

func (tokenIsAccount) Authenticate(ctx context.Context, token string) (context.Context, string, error) {
	id, err := uuid.Parse(token)
	if err != nil {
		return nil, "", shared.Unauthenticated()
	}
	return shared.WithActor(ctx, shared.Actor{UserID: id, SessionID: uuid.NewV7()}), "account:" + token, nil
}

// projectRoute is the project module as bootstrap wires it (project.New),
// on pool, with auth as its Authorizer and members as its WorkspaceMembers,
// mounted behind an API whose bearer token is an account's id and whose
// requests end after 3 seconds.
type projectRoute struct {
	router http.Handler
}

func newProjectRoute(t *testing.T, pool *pgxpool.Pool, auth shared.Authorizer, members projectapp.WorkspaceMembers) projectRoute {
	t.Helper()
	module := project.New(project.Deps{Pool: pool, Tx: postgres.NewTxManager(pool, 2*time.Second), Clock: clock.System{}, Authorizer: auth,
		Workspaces: projectWorkspaces{directory: workspace.Provide(pool).WorkspaceDirectory}, Members: members})
	logger := slog.New(slog.DiscardHandler)
	router := httpserver.NewRouter(logger)
	limit := ratelimit.New(time.Now).Bucket("test", ratelimit.Rate{PerMinute: 600, Burst: 100})
	api, err := httpserver.NewAPI(httpserver.APIConfig{Logger: logger, Authenticator: tokenIsAccount{}, MaxBodyBytes: 1 << 20,
		RequestTimeout: 3 * time.Second, IPv6PrefixLen: 64, Anonymous: limit, Authenticated: limit, AuthFailure: limit})
	if err != nil {
		t.Fatal(err)
	}
	module.Register(router, api)
	return projectRoute{router: router}
}

// send sends method path as caller, with body when it is not empty, and
// returns the request and its answer. It touches no test, so another
// goroutine may send it.
func (p projectRoute) send(method, path string, caller uuid.UUID, body string) (*http.Request, *httptest.ResponseRecorder) {
	var b io.Reader
	if body != "" {
		b = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, b)
	req.Header.Set("Authorization", "Bearer "+caller.String())
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	p.router.ServeHTTP(rec, req)
	return req, rec
}

// Each write on a project runs every statement on its transaction's
// connection (M3 design 3.6 convention 2, 6.7; P4a's L3): its locks, its
// reads, the facts its decision reads (ProjectFacts and the workspace
// roles), its writes and its answer. The project module and the Authorizer
// are wired as bootstrap wires them, on a pool of one connection: a
// statement sent through the pool rather than the transaction would wait
// for a second connection that never comes, and its request fail at the
// request's deadline. alice, acme's admin, creates Ops, changes Web,
// archives and unarchives it, changes her display settings in it and adds
// bob, whose ended membership she restores; carol joins it anew; alice
// deletes both projects.
func TestTheWritesOnAProjectRunOnTheirTransactionsConnection(t *testing.T) {
	r := newGrowthRace(t, true)
	carol := uuid.NewV7()
	ctx, now := context.Background(), time.Now()
	if err := identitypg.New(r.pool).CreateUser(ctx, identityapp.NewUser{ID: carol, Email: "carol@example.com", PasswordHash: "x",
		DisplayName: "carol", Now: now}); err != nil {
		t.Fatal(err)
	}
	var acme uuid.UUID
	if err := r.pool.QueryRow(ctx, "SELECT workspace_id FROM projects WHERE id = $1", r.web).Scan(&acme); err != nil {
		t.Fatal(err)
	}
	if err := workspacepg.New(r.pool).CreateMember(ctx, workspaceapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, MemberID: carol,
		Role: shared.RoleMember, CreatedBy: r.alice, Now: now}); err != nil {
		t.Fatal(err)
	}
	one, err := postgres.NewPool(ctx, config.DatabaseConfig{URL: r.pool.Config().ConnString(), MaxConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	// A statement on a context without the request's deadline would wait for the connection for ever, and so would
	// closing the pool: each has a deadline of its own.
	t.Cleanup(func() {
		closed := make(chan struct{})
		go func() { one.Close(); close(closed) }()
		select {
		case <-closed:
		case <-time.After(5 * time.Second):
			t.Error("the pool of one connection did not close within 5s: a transaction still holds its connection")
		}
	})
	route := newProjectRoute(t, one, authorizerOn(one), workspace.Provide(one).WorkspaceMembers)
	contract := apitest.Load(t)
	send := func(method, path string, caller uuid.UUID, body string, want int) string {
		t.Helper()
		type answer struct {
			req *http.Request
			rec *httptest.ResponseRecorder
		}
		done := make(chan answer, 1)
		go func() {
			req, rec := route.send(method, path, caller, body)
			done <- answer{req, rec}
		}()
		var a answer
		select {
		case a = <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("%s %s did not answer within 5s: a statement waits for the pool's one connection without the request's deadline", method,
				path)
		}
		req, rec := a.req, a.rec
		contract.CheckResponse(t, req, rec.Result())
		if rec.Code != want {
			t.Fatalf("%s %s = %d %s, want %d", method, path, rec.Code, rec.Body, want)
		}
		return rec.Body.String()
	}

	var created struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, send(http.MethodPost, "/api/v0/workspaces/acme/projects", r.alice, `{"name":"Ops","identifier":"OPS"}`, http.StatusCreated),
		&created)
	web := "/api/v0/projects/" + r.web.String()
	send(http.MethodPatch, web, r.alice, `{"name":"Site"}`, http.StatusOK)
	send(http.MethodPost, web+"/archive", r.alice, "", http.StatusOK)
	send(http.MethodPost, web+"/unarchive", r.alice, "", http.StatusOK)
	send(http.MethodPatch, "/api/v0/me/projects/"+r.web.String()+"/preferences", r.alice, `{"sort_order":1}`, http.StatusOK)
	send(http.MethodPost, web+"/members", r.alice, `{"members":[{"member_id":"`+r.bob.String()+`","role":15}]}`, http.StatusCreated)
	send(http.MethodPost, web+"/join", carol, "", http.StatusOK)
	send(http.MethodDelete, web, r.alice, "", http.StatusNoContent)
	send(http.MethodDelete, "/api/v0/projects/"+created.ID.String(), r.alice, "", http.StatusNoContent)
}
