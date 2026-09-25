package httpadapter_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
)

type fixedSource domain.Build

func (s fixedSource) Build() domain.Build { return domain.Build(s) }

// noAccounts authenticates nobody: every operation of instance is public, so
// it is never asked.
type noAccounts struct{}

func (noAccounts) Authenticate(context.Context, string) (context.Context, string, error) {
	return nil, "", errors.New("instance has no operation that needs a token")
}

func TestGetInstanceMatchesTheContract(t *testing.T) {
	contract := apitest.Load(t)
	logger := slog.New(slog.DiscardHandler)
	router := httpserver.NewRouter(logger)
	limit := ratelimit.New(time.Now).Bucket("test", ratelimit.Rate{PerMinute: 600, Burst: 100})
	api, err := httpserver.NewAPI(httpserver.APIConfig{
		Logger:           logger,
		Authenticator:    noAccounts{},
		PublicOperations: httpadapter.PublicOperations(),
		MaxBodyBytes:     1 << 20,
		RequestTimeout:   time.Second,
		IPv6PrefixLen:    64,
		Anonymous:        limit,
		Authenticated:    limit,
		AuthFailure:      limit,
	})
	if err != nil {
		t.Fatal(err)
	}
	getInfo := app.NewGetInfo(fixedSource{Version: "1.2.3", Commit: "4f2a9c1"})
	httpadapter.Register(router, api, httpadapter.UseCases{GetInfo: getInfo})
	req := httptest.NewRequest(http.MethodGet, "/api/v0/instance", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	res := rec.Result()

	contract.CheckResponse(t, req, res)
	body, _ := io.ReadAll(res.Body)
	want := `{"api_version":"v0","commit":"4f2a9c1","product":"Nerve","version":"1.2.3"}` + "\n"
	if res.StatusCode != http.StatusOK || string(body) != want {
		t.Errorf("GET /api/v0/instance = %d %s, want 200 %s", res.StatusCode, body, want)
	}
}
