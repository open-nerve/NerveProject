package bootstrap

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

func TestAppPoolFactoryOrderAndOwnership(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	cfg := testConfig(t, pgtest.NewDatabase(t), false)
	called := 0
	sentinel := errors.New("factory failed")
	fail := func(context.Context, config.DatabaseConfig) (*pgxpool.Pool, error) { called++; return nil, sentinel }
	bad := cfg
	bad.Auth.JWT.PrivateKeyFile = t.TempDir() + "/missing"
	if _, err := newAppWithPoolFactory(t.Context(), bad, logger, migrations.FS(), testWebUI, fail); err == nil || called != 0 {
		t.Fatalf("key failure=%v factory calls=%d", err, called)
	}
	if a, err := newAppWithPoolFactory(t.Context(), cfg, logger, migrations.FS(), testWebUI, fail); !errors.Is(err, sentinel) || a != nil || called != 1 {
		if a != nil {
			a.close()
		}
		t.Fatalf("factory failure=%v app=%v calls=%d", err, a != nil, called)
	}
	for _, test := range []struct {
		name       string
		migrations fs.FS
		badJobs    bool
	}{
		{"migrator failure", fstest.MapFS{"00001_a.sql": {Data: []byte("-- +goose Up\nSELECT 1;")}, "00001_b.sql": {Data: []byte("-- +goose Up\nSELECT 1;")}}, false},
		{"later constructor failure", migrations.FS(), true},
		{"normal ownership", migrations.FS(), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := cfg
			if test.badJobs {
				c.Auth.SessionCleanupInterval = 0
			}
			var made *pgxpool.Pool
			factory := func(ctx context.Context, db config.DatabaseConfig) (*pgxpool.Pool, error) {
				var err error
				made, err = postgres.NewPool(ctx, db)
				return made, err
			}
			a, err := newAppWithPoolFactory(t.Context(), c, logger, test.migrations, testWebUI, factory)
			if made == nil {
				t.Fatal("pool factory was not called")
			}
			defer made.Close()
			if test.name == "normal ownership" {
				if err != nil {
					t.Fatal(err)
				}
				if a.pool != made {
					t.Fatal("app did not own factory pool")
				}
				a.close()
			} else if err == nil {
				a.close()
				t.Fatal("wanted constructor failure")
			}
			if conn, err := made.Acquire(t.Context()); err == nil {
				conn.Release()
				t.Fatal("pool remained open after failed construction or app close")
			}
		})
	}
}
