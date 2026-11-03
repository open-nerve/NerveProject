package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

func TestAssetLifetimeFollowsApp(t *testing.T) {
	assetLifetimeFollowsApp(t, func(*config.Config) {})
}

func assetLifetimeFollowsApp(t *testing.T, configure func(*config.Config)) {
	t.Helper()
	dbURL := pgtest.NewDatabase(t)
	for _, failure := range []string{"", "module", "webui"} {
		failLater := failure != ""
		name := "normal shutdown and repeated close"
		if failLater {
			name = "later " + failure + " construction failure"
		}
		t.Run(name, func(t *testing.T) {
			cfg := testConfig(t, dbURL, false)
			configure(&cfg)
			if failure == "module" {
				cfg.Auth.SessionCleanupInterval = 0
			}
			var logs lockedBuffer
			logger := slog.New(slog.NewTextHandler(&logs, nil))
			var pool *pgxpool.Pool
			var base string
			closed := 0
			factories := appFactories{
				pool: func(ctx context.Context, db config.DatabaseConfig) (*pgxpool.Pool, error) {
					var err error
					pool, err = postgres.NewPool(ctx, db)
					return pool, err
				},
				assets: func(ctx context.Context, files config.FilesConfig, keys *identity.Keys, log *slog.Logger) (assetComponents, error) {
					components, err := newAssets(ctx, files, keys, log)
					if err != nil {
						return components, err
					}
					if failure == "webui" {
						components.origin = "*"
					}
					closeStorage := components.close
					components.close = func() error {
						closed++
						if err := pool.Ping(t.Context()); err != nil {
							t.Errorf("database closed before storage: %v", err)
						}
						if !failLater {
							out := logs.String()
							httpAt, jobsAt := strings.Index(out, `msg="http server stopped"`), strings.Index(out, `msg="jobs stopped"`)
							if httpAt < 0 || jobsAt < httpAt {
								t.Errorf("storage closed before HTTP then jobs stopped:\n%s", out)
							}
							if resp, err := client.Get(base + "/healthz"); err == nil {
								_ = resp.Body.Close()
								t.Error("HTTP still accepts requests while storage closes")
							}
						}
						return closeStorage()
					}
					return components, nil
				},
			}
			a, err := newAppWithFactories(t.Context(), cfg, logger, migrations.FS(), testWebUI, factories)
			if pool == nil {
				t.Fatal("pool was not constructed")
			}
			t.Cleanup(pool.Close)
			if failLater {
				if err == nil || a != nil {
					if a != nil {
						a.close()
					}
					t.Fatal("expected later constructor failure")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(a.close)
				var stop func()
				base, stop = runBuiltAppWithStop(t, a)
				stop()
				a.close()
				a.close()
			}
			if closed != 1 {
				t.Errorf("storage closed %d times; want exactly once", closed)
			}
			if err := pool.Ping(t.Context()); err == nil {
				t.Error("database remains open after app cleanup")
			}
		})
	}
}

func TestAssetLifetimeIsolationAndRestart(t *testing.T) {
	cfg := testConfig(t, unreachableDB, false)
	cfg.Auth.JWT.PrivateKeyFile = writeFile(t, testKeyPEM)
	first := buildApp(t, cfg, migrations.FS())
	server := httptest.NewServer(first.router)
	t.Cleanup(server.Close)
	store := assetClientStore(t, cfg)
	u, o := assetKey(t, 'u'), assetKey(t, 'o')
	grant := domain.UploadGrant{Key: u, Size: 3, ContentType: "text/plain", ExpiresAt: time.Now().Add(5 * time.Minute)}
	put, err := store.SignUpload(t.Context(), grant)
	if err != nil {
		t.Fatal(err)
	}
	assetRequest(t, server.URL, put, "one", http.StatusNoContent)
	info, err := store.Inspect(t.Context(), u)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Seal(t.Context(), u, o, info.ETag); err != nil {
		t.Fatal(err)
	}
	get, err := store.SignRead(t.Context(), domain.ReadGrant{Key: o, Method: "GET", Name: "a.txt", ContentType: "text/plain", ExpiresAt: time.Now().Add(5 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if got := assetRequest(t, server.URL, get, "", http.StatusOK); got != "one" {
		t.Fatalf("first world returned %q", got)
	}
	server.Close()
	first.close()
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	// Advancing business time must not expire real-time transfer grants.
	cfg.Clock.Offset = 48 * time.Hour
	restarted := httptest.NewServer(buildApp(t, cfg, migrations.FS()).router)
	t.Cleanup(restarted.Close)
	if got := assetRequest(t, restarted.URL, get, "", http.StatusOK); got != "one" {
		t.Fatalf("restarted world returned %q", got)
	}

	other := cfg
	other.Files.Local.Root = t.TempDir()
	otherServer := httptest.NewServer(buildApp(t, other, migrations.FS()).router)
	t.Cleanup(otherServer.Close)
	assetRequest(t, otherServer.URL, get, "", http.StatusNotFound)
	assetRequest(t, otherServer.URL, put, "two", http.StatusNoContent)
	otherStore := assetClientStore(t, other)
	otherInfo, err := otherStore.Inspect(t.Context(), u)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := otherStore.Seal(t.Context(), u, o, otherInfo.ETag); err != nil {
		t.Fatal(err)
	}
	if got := assetRequest(t, otherServer.URL, get, "", http.StatusOK); got != "two" {
		t.Fatalf("second world returned %q", got)
	}
	if got := assetRequest(t, restarted.URL, get, "", http.StatusOK); got != "one" {
		t.Fatalf("second world changed first world's bytes: %q", got)
	}

	// The same root with a newly generated test key must reject the old signature.
	cfg.Auth.JWT.PrivateKeyFile = ""
	newKey := httptest.NewServer(buildApp(t, cfg, migrations.FS()).router)
	t.Cleanup(newKey.Close)
	assetRequest(t, newKey.URL, get, "", http.StatusForbidden)
}
