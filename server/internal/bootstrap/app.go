// Package bootstrap is nerve's only composition root: it builds every adapter
// from the configuration, wires them together and runs the commands.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"slices"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
	"github.com/open-nerve/NerveProject/server/internal/platform/webui"
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
	cfg      config.Config
	logger   *slog.Logger
	pool     *pgxpool.Pool
	migrator *postgres.Migrator
	jobs     *jobs.Runner
	router   *httpserver.Router
	// poolCloseTimeout is the package's, but for tests.
	poolCloseTimeout time.Duration
	// publicOperations are the routes that need no token: the union of what
	// the modules declare, checked against the contract by a test.
	publicOperations []string
}

// newApp wires the server described by cfg around the given migrations and
// built web frontend. close releases it.
func newApp(ctx context.Context, cfg config.Config, logger *slog.Logger, migrationFiles, webFiles fs.FS) (*app, error) {
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
	invitationMAC, err := keys.MAC(workspace.InvitationMACPurpose)
	if err != nil {
		return nil, err
	}
	pool, err := postgres.NewPool(ctx, cfg.Database)
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
	// Every rate-limit bucket lives on this limiter (M2 design 3.10). It reads
	// time.Now, not the Clock: the monotonic reading keeps a step of the wall
	// clock from filling or draining the buckets.
	limiter := ratelimit.New(time.Now)
	tx := postgres.NewTxManager(pool, cfg.Database.CommitTimeout)

	// The modules in two steps (M3 design 6.6): first the adapters each
	// offers the others, built from the pool alone; then the modules, each
	// with every port it needs, so no construction waits on another.
	identityPorts := identity.Provide(pool)
	workspacePorts := workspace.Provide(pool)
	authorizer := access.New(access.Deps{WorkspaceRoles: workspacePorts.WorkspaceRoles})
	ws := workspace.New(workspace.Deps{
		Pool: pool, Tx: tx, Clock: clock.System{}, Logger: logger, Authorizer: authorizer,
		Accounts:        workspaceAccounts{accounts: identityPorts.Accounts},
		Profiles:        workspaceProfiles{profiles: identityPorts.PublicProfiles},
		InvitationMAC:   invitationMAC,
		CallerLock:      identityPorts.CredentialLock,
		CreationEnabled: cfg.Workspace.CreationEnabled,
	})
	ident, err := identity.New(identity.Deps{
		Pool:            pool,
		Tx:              tx,
		Clock:           clock.System{},
		Logger:          logger,
		SignupPolicy:    signupSwitch(cfg.Auth.SignupEnabled),
		Keys:            keys,
		AccessTokenTTL:  cfg.Auth.AccessTokenTTL,
		SessionTTL:      cfg.Auth.SessionTTL,
		RefreshDeadline: cfg.Auth.RefreshDeadline,
		// The periodic job that deletes the expired sessions (M2 design 3.15).
		SessionCleanupInterval: cfg.Auth.SessionCleanupInterval,
		Password:               passwordHashing(cfg.Auth.Password),
		RateLimits: identity.RateLimits{
			Limiter:      limiter,
			LoginIP:      bucket(limiter, "login_ip", cfg.RateLimit.LoginIP),
			LoginIPEmail: bucket(limiter, "login_ip_email", cfg.RateLimit.LoginIPEmail),
			RegisterIP:   bucket(limiter, "register_ip", cfg.RateLimit.RegisterIP),
			PasswordUser: bucket(limiter, "password_user", cfg.RateLimit.PasswordUser),
		},
	})
	if err != nil {
		a.close()
		return nil, err
	}
	a.jobs, err = jobs.New(pool, jobs.Config{ShutdownTimeout: cfg.Jobs.ShutdownTimeout, Logger: logger}, ident.Jobs())
	if err != nil {
		a.close()
		return nil, err
	}
	inst, err := instance.New(instance.Deps{
		SignupEnabled:            cfg.Auth.SignupEnabled,
		WorkspaceCreationEnabled: cfg.Workspace.CreationEnabled,
		FileSizeLimit:            cfg.Files.SizeLimit,
		Clock:                    clock.System{},
	})
	if err != nil {
		a.close()
		return nil, err
	}

	a.router = httpserver.NewRouter(logger,
		httpserver.Check{Name: "database", Run: pool.Ping},
		httpserver.Check{Name: "migrations", Run: migrator.CheckUpToDate},
	)
	a.publicOperations = slices.Concat(ident.PublicOperations(), inst.PublicOperations(), ws.PublicOperations())
	api, err := httpserver.NewAPI(httpserver.APIConfig{
		Logger:           logger,
		Authenticator:    ident.Authenticator(),
		PublicOperations: a.publicOperations,
		MaxBodyBytes:     cfg.Server.MaxBodyBytes,
		RequestTimeout:   cfg.Server.RequestTimeout,
		TrustedProxies:   cfg.Server.TrustedProxies,
		IPv6PrefixLen:    cfg.RateLimit.IPv6PrefixLen,
		Anonymous:        bucket(limiter, "anonymous", cfg.RateLimit.Anonymous),
		Authenticated:    bucket(limiter, "authenticated", cfg.RateLimit.Authenticated),
		AuthFailure:      bucket(limiter, "auth_failure", cfg.RateLimit.AuthFailure),
	})
	if err != nil {
		a.close()
		return nil, err
	}
	// Modules mount their generated routes on this root router, next to the
	// platform's /api/ fallback; an /api/v0/ sub-mux would shadow it.
	ident.Register(a.router, api)
	inst.Register(a.router, api)
	ws.Register(a.router, api)
	// The web UI takes every path no other pattern claims. It must be the
	// method-less "/": "GET /" and the method-less "/api/" would conflict.
	a.router.Handle("/", webui.Handler(webFiles))
	return a, nil
}

// readSigningKey returns the content of auth.jwt.private_key_file, or nil
// when it is not set. Errors name the key, never the path (M0-P2 handoff 4).
func readSigningKey(path string) ([]byte, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			err = pathErr.Err
		}
		return nil, fmt.Errorf("auth.jwt.private_key_file: %w", err)
	}
	return data, nil
}

// passwordHashing is auth.password as identity takes it.
func passwordHashing(p config.PasswordConfig) identity.PasswordHashing {
	return identity.PasswordHashing{
		MemoryKiB:     p.Argon2MemoryKiB,
		Iterations:    p.Argon2Iterations,
		Parallelism:   p.Argon2Parallelism,
		MaxConcurrent: p.MaxConcurrentHashes,
		MaxWait:       p.MaxWait,
	}
}

// bucket is the rate-limit bucket name on limiter, sized by c (M2 design
// 3.10).
func bucket(limiter *ratelimit.Limiter, name string, c config.BucketConfig) *ratelimit.Bucket {
	return limiter.Bucket(name, ratelimit.Rate{PerMinute: c.PerMinute, Burst: c.Burst})
}

// signupSwitch is auth.signup_enabled as identity's SignupPolicy.
type signupSwitch bool

func (s signupSwitch) AllowSignup(context.Context) (bool, error) { return bool(s), nil }

// warnIfExposed warns once when a non-prod nerve listens beyond loopback:
// most likely a deployment without NERVE_ENV=prod (M2 design 6.1).
func warnIfExposed(ctx context.Context, logger *slog.Logger, cfg config.Config) {
	if cfg.Env == config.EnvProd || loopback(cfg.Server.Addr) {
		return
	}
	logger.WarnContext(ctx, "not running as prod but listening beyond this machine: sign-up is open by default, "+
		"and without auth.jwt.private_key_file the signing key is ephemeral; set NERVE_ENV=prod to deploy",
		slog.String("env", cfg.Env), slog.String("addr", cfg.Server.Addr))
}

func loopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.IsLoopback()
}

// run applies pending migrations when database.auto_migrate is on, then
// runs the jobs and serves HTTP on server.addr until ctx is done. It stops
// in the order of M2 design 3.15: HTTP first (see httpserver.Server.Serve),
// so no request is left to enqueue work, then the jobs; close then releases
// the migrator and the pool.
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

// close releases the database resources: the migrator, then the pool, whose
// wait for connections still in use is bounded. Call it after run has
// returned.
func (a *app) close() {
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
}
