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

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
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

// get serves the module with uc and answers a GET of path, checked against
// the contract.
func get(t *testing.T, uc httpadapter.UseCases, path string) (*http.Response, string) {
	t.Helper()
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
	httpadapter.Register(router, api, uc)
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	res := rec.Result()

	apitest.Load(t).CheckResponse(t, req, res)
	body, _ := io.ReadAll(res.Body)
	return res, string(body)
}

func TestGetInstanceMatchesTheContract(t *testing.T) {
	getInfo := app.NewGetInfo(fixedSource{Version: "1.2.3", Commit: "4f2a9c1"},
		domain.Settings{SignupEnabled: true, WorkspaceCreationEnabled: false, FileSizeLimit: 5242880})

	res, body := get(t, httpadapter.UseCases{GetInfo: getInfo}, "/api/v0/instance")

	want := `{"api_version":"v0","commit":"4f2a9c1","file_size_limit":5242880,"product":"Nerve","signup_enabled":true,` +
		`"version":"1.2.3","workspace_creation_enabled":false}` + "\n"
	if res.StatusCode != http.StatusOK || body != want {
		t.Errorf("GET /api/v0/instance = %d %s, want 200 %s", res.StatusCode, body, want)
	}
}

func TestListTimezones(t *testing.T) {
	zones, err := domain.LoadTimezones()
	if err != nil {
		t.Fatal(err)
	}
	list := app.NewListTimezones(zones, clocktest.At(time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)))

	res, body := get(t, httpadapter.UseCases{ListTimezones: list}, "/api/v0/timezones")

	first := `{"data":[{"gmt_offset":"GMT-11:00","label":"American Samoa","utc_offset":"UTC-11:00","value":"Pacific/Pago_Pago"},`
	marquesas := `{"gmt_offset":"GMT-09:30","label":"Marquesas Islands","utc_offset":"UTC-09:30","value":"Pacific/Marquesas"}`
	if res.StatusCode != http.StatusOK || !strings.HasPrefix(body, first) || !strings.Contains(body, marquesas) {
		t.Errorf("GET /api/v0/timezones = %d %s, want 200 starting %s and holding %s", res.StatusCode, body, first, marquesas)
	}
}
