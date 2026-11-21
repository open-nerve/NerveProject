// Package bootstrap is nerve's only composition root: it builds every adapter
// from the configuration, wires them together and runs the commands.
package bootstrap

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/buildinfo"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The platform and the shared kernel meet here by structure: neither imports
// the other (M2 design 3.3, 3.11).
var (
	_ shared.TxManager        = (*postgres.TxManager)(nil)
	_ httpserver.ProblemError = (*shared.Error)(nil)
)

// poolCloseTimeout bounds the wait for the pool's connections at shutdown:
// a handler that ignores its context may still hold one (M2 design 3.15).
const poolCloseTimeout = 5 * time.Second

// app is a fully wired nerve server.
type app struct {
	cfg        config.Config
	logger     *slog.Logger
	pool       *pgxpool.Pool
	migrator   *postgres.Migrator
	jobs       *jobs.Runner
	router     *httpserver.Router
	assetClose func() error
	closeOnce  sync.Once
	// poolCloseTimeout is the package's, but for tests.
	poolCloseTimeout time.Duration
	// publicOperations are the routes that need no token: the union of what
	// the modules declare, checked against the contract by a test.
	publicOperations []string
}

// newApp wires the server described by cfg around the given migrations and
// built web frontend. close releases it.
func newApp(ctx context.Context, cfg config.Config, logger *slog.Logger, migrationFiles, webFiles fs.FS) (*app, error) {
	return newAppWithPoolFactory(ctx, cfg, logger, migrationFiles, webFiles, postgres.NewPool)
}

// newAppWithPoolFactory preserves one composition and the app's pool ownership.
func newAppWithPoolFactory(ctx context.Context, cfg config.Config, logger *slog.Logger, migrationFiles, webFiles fs.FS, factory func(context.Context, config.DatabaseConfig) (*pgxpool.Pool, error)) (*app, error) {
	return newAppWithFactories(ctx, cfg, logger, migrationFiles, webFiles, appFactories{pool: factory, assets: newAssets})
}

// Each factory transfers one explicit resource to the app. This keeps the same
// composition and cleanup path for normal startup and later constructor errors.
type appFactories struct {
	pool   func(context.Context, config.DatabaseConfig) (*pgxpool.Pool, error)
	assets func(context.Context, config.FilesConfig, *identity.Keys, *slog.Logger) (assetComponents, error)
}

// wiring carries the values newAppWithFactories's steps hand one another.
// newAppWithFactories fills the first group; app_ports.go's provide,
// dispatchEvents and share the next three; app_modules.go's modules the
// last; app_serve.go's schedule and route give the modules' jobs and routes
// to the app.
type wiring struct {
	cfg           config.Config
	logger        *slog.Logger
	pool          *pgxpool.Pool
	keys          *identity.Keys
	invitations   identity.MAC
	assets        assetComponents
	limiter       *ratelimit.Limiter
	businessClock clock.Offset
	tx            *postgres.TxManager

	// provide: the adapters each module offers the others, and what is
	// built from them alone.
	identityPorts     identity.Provided
	assetPorts        asset.Provided
	workspacePorts    workspace.Provided
	projectPorts      project.Provided
	issuePorts        issue.Provided
	planningPorts     planning.Provided
	notificationPorts notification.Provided
	navigationPorts   navigation.Provided
	imageBindings     asset.ImageBindings
	imageReader       asset.ImageReader
	assetPurger       *assetapp.Purger
	authorizer        shared.Authorizer
	decider           *access.Decider
	readable          *access.ReadableProjects

	// dispatchEvents: the job inserter and the dispatcher on it.
	inserter jobInserter
	events   *dispatcher

	// share: what more than one module takes.
	projectWrites writeGate
	issueTx       shared.TxManager
	readTx        *postgres.TxManager
	previews      linkPreviews

	// modules: the modules, in the order modules builds them.
	issues *issue.Module
	plans  *planning.Module
	notes  *notification.Module
	views  *navigation.Module
	proj   *project.Module
	ws     *workspace.Module
	ident  *identity.Module
	files  *asset.Module
}

// newAppWithFactories builds the app step by step, keeping six orders:
// (1) every module's Provide before any module's New (provide before
// modules); (2) project before workspace, (3) workspace before identity and
// (4) asset after issue, project and workspace (modules); (5) the order of
// the purge's modules, of the jobs and their kinds (schedule) and of the
// routes (route); (6) the cleanup of each failure: none before the pool,
// the pool's close when the migrator fails, and once the app exists its
// close, here, as soon as a step fails, so nothing after it is built.
func newAppWithFactories(ctx context.Context, cfg config.Config, logger *slog.Logger, migrationFiles, webFiles fs.FS, factories appFactories) (*app, error) {
	signingKey, err := readSigningKey(cfg.Auth.JWT.PrivateKeyFile)
	if err != nil {
		return nil, err
	}
	// The signing key first, before any module, and the MACs derived from
	// it for the other modules (M3 design 6.6 step 1).
	keys, err := identity.LoadKeys(signingKey, logger)
	if err != nil {
		return nil, err
	}
	invitations, err := invitationMAC(keys)
	if err != nil {
		return nil, err
	}
	pool, err := factories.pool(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}
	// The configuration log masks database.url as a whole; say where the pool
	// connects from pgx's own parse of it, which holds no password.
	target := pool.Config().ConnConfig
	logger.InfoContext(ctx, "database pool created",
		slog.String("host", target.Host), slog.Int("port", int(target.Port)),
		slog.String("database", target.Database), slog.String("user", target.User))
	migrator, err := postgres.NewMigrator(pool, migrationFiles)
	if err != nil {
		pool.Close()
		return nil, err
	}
	a := &app{cfg: cfg, logger: logger, pool: pool, migrator: migrator, poolCloseTimeout: poolCloseTimeout}
	assets, err := factories.assets(ctx, cfg.Files, keys, logger)
	a.assetClose = assets.close
	if err != nil {
		a.close()
		return nil, err
	}
	outboundClient, err := newOutbound(cfg, buildinfo.Get().Version)
	if err != nil {
		a.close()
		return nil, err
	}
	// Every rate-limit bucket lives on this limiter (M2 design 3.10). It reads
	// time.Now, not the Clock: the monotonic reading keeps a step of the wall
	// clock from filling or draining the buckets.
	limiter := ratelimit.New(time.Now)
	businessClock := newClock(cfg)
	tx := postgres.NewTxManager(pool, cfg.Database.CommitTimeout)

	// The modules in two steps (M3 design 6.6): first the adapters each
	// offers the others, built from the pool alone; then the modules, each
	// with every port it needs, so no construction waits on another.
	w := &wiring{cfg: cfg, logger: logger, pool: pool, keys: keys, invitations: invitations, assets: assets, limiter: limiter, businessClock: businessClock, tx: tx}
	w.provide()
	if err = w.dispatchEvents(); err != nil {
		a.close()
		return nil, err
	}
	w.share(outboundClient)
	if err = w.modules(); err != nil {
		a.close()
		return nil, err
	}
	if err = a.schedule(w); err != nil {
		a.close()
		return nil, err
	}
	if err = a.route(w, webFiles); err != nil {
		a.close()
		return nil, err
	}
	return a, nil
}

// run applies pending migrations when database.auto_migrate is on, then
// runs the jobs and serves HTTP on server.addr until ctx is done. It stops
// in the order of M2 design 3.15: HTTP first (see httpserver.Server.Serve),
// so no request is left to enqueue work, then the jobs; close then releases
// storage, the migrator and the pool.
func (a *app) run(ctx context.Context) error {
	warnIfExposed(ctx, a.logger, a.cfg)
	if a.cfg.Database.AutoMigrate {
		applied, err := a.migrator.Up(ctx)
		// Log what was applied even when a later migration failed.
		for _, m := range applied {
			a.logger.InfoContext(ctx, "migration applied", slog.Int64("version", m.Version), slog.String("source", m.Source))
		}
		if err != nil {
			return err
		}
	}
	a.jobs.Start(ctx)
	served := httpserver.NewServer(a.cfg.Server, a.router, a.logger).ListenAndServe(ctx)
	return errors.Join(served, a.jobs.Stop(ctx))
}

// close releases storage, the migrator, then the pool, whose
// wait for connections still in use is bounded. Call it after run has
// returned.
func (a *app) close() {
	a.closeOnce.Do(func() {
		if a.assetClose != nil {
			if err := a.assetClose(); err != nil {
				a.logger.Warn("close file storage", slog.Any("error", err))
			}
		}
		if err := a.migrator.Close(); err != nil {
			a.logger.Warn("close migrator", slog.Any("error", err))
		}
		closed := make(chan struct{})
		go func() {
			a.pool.Close()
			close(closed)
		}()
		select {
		case <-closed:
			a.logger.Info("database pool closed")
		case <-time.After(a.poolCloseTimeout):
			a.logger.Warn("database pool not closed: connections still in use", slog.Duration("waited", a.poolCloseTimeout))
		}
	})
}
