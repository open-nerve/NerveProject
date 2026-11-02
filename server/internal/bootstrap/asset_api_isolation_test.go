package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

type assetAPIDeadlineTrace struct{ deadlines chan time.Time }

func (trace assetAPIDeadlineTrace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	if httpserver.RequestID(ctx) == "m5-json-deadline-probe" {
		deadline, _ := ctx.Deadline()
		select {
		case trace.deadlines <- deadline:
		default:
		}
	}
	return ctx
}
func (assetAPIDeadlineTrace) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func TestStorageDoesNotWidenJSONAPI(t *testing.T) {
	cfg := defaultAssetConfig(t, pgtest.NewDatabase(t))
	if cfg.Server.MaxBodyBytes != 1<<20 || cfg.Server.RequestTimeout != 15*time.Second || cfg.Server.ReadTimeout != 30*time.Second {
		t.Fatalf("default JSON/network budgets changed: %+v", cfg.Server)
	}
	trace := assetAPIDeadlineTrace{deadlines: make(chan time.Time, 1)}
	factory := func(ctx context.Context, db config.DatabaseConfig) (*pgxpool.Pool, error) {
		source, err := postgres.NewPool(ctx, db)
		if err != nil {
			return nil, err
		}
		defer source.Close()
		pc := source.Config()
		pc.ConnConfig.Tracer = trace
		return pgxpool.NewWithConfig(ctx, pc)
	}
	a, err := newAppWithPoolFactory(t.Context(), cfg, slog.New(slog.DiscardHandler), migrations.FS(), testWebUI, factory)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.close)
	base := runBuiltApp(t, a)
	contract := apitest.Load(t)
	for _, tc := range []struct {
		name, method, path, body, code string
		status                         int
	}{
		{"body over 1 MiB", "POST", "/api/v0/auth/login", `{"email":"nobody@example.com","password":"` + strings.Repeat("p", 1<<20) + `"}`, "payload_too_large", 413},
		{"invalid body shape", "POST", "/api/v0/auth/login", `{"email":true,"password":12}`, "bad_request", 400},
		{"authenticated operation", "GET", "/api/v0/me", "", "unauthorized", 401},
		{"public instance", "GET", "/api/v0/instance", "", "", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := newRequest(t, tc.method, base+tc.path, "", []byte(tc.body))
			res, body := send(t, req)
			contract.CheckResponse(t, req, res)
			if res.StatusCode != tc.status || res.Header.Get("Cache-Control") != "no-store" {
				t.Fatalf("JSON status=%d cache=%q body=%s", res.StatusCode, res.Header.Get("Cache-Control"), body)
			}
			if tc.code != "" && problemCode(t, body) != tc.code {
				t.Fatalf("wrong JSON refusal: %s", body)
			}
			assertAssetSecurityHeaders(t, res)
		})
	}

	// Observe the actual request context at a real SQL boundary. This proves the
	// configured 15s reaches the ordinary API without a timing-sensitive sleep.
	req := newRequest(t, "POST", base+"/api/v0/auth/login", "", []byte(`{"email":"nobody@example.com","password":"wrong"}`))
	req.Header.Set(httpserver.HeaderRequestID, "m5-json-deadline-probe")
	before := time.Now()
	res, body := send(t, req)
	after := time.Now()
	contract.CheckResponse(t, req, res)
	if res.StatusCode != 401 || problemCode(t, body) != "identity.invalid_credentials" {
		t.Fatalf("deadline probe did not reach the real login query: %d %s", res.StatusCode, body)
	}
	select {
	case deadline := <-trace.deadlines:
		if deadline.Before(before.Add(15*time.Second)) || deadline.After(after.Add(15*time.Second)) {
			t.Fatalf("SQL context deadline %s is not the actual request's 15s deadline (%s..%s)", deadline, before, after)
		}
	default:
		t.Fatal("no observed query from the deadline probe")
	}
	for _, path := range []string{"/healthz", "/readyz"} {
		res, body := send(t, newRequest(t, "GET", base+path, "", nil))
		if res.StatusCode != 200 || string(body) != "{\"status\":\"ok\"}\n" {
			t.Fatalf("%s changed: %d %s", path, res.StatusCode, body)
		}
		assertAssetSecurityHeaders(t, res)
	}
}

func assertAssetSecurityHeaders(t *testing.T, res *http.Response) {
	t.Helper()
	for key, want := range map[string]string{"X-Content-Type-Options": "nosniff", "X-Frame-Options": "DENY", "Referrer-Policy": "same-origin"} {
		if res.Header.Get(key) != want {
			t.Errorf("security header %s=%q, want %q", key, res.Header.Get(key), want)
		}
	}
	if res.Header.Get(httpserver.HeaderRequestID) == "" {
		t.Error("request ID missing")
	}
}
