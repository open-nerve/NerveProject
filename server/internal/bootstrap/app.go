// Package bootstrap is nerve's only composition root: it builds every adapter
// from the configuration, wires them together and runs the commands.
package bootstrap

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/buildinfo"
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
	identityPorts := identity.Provide(pool)
	assetPorts := asset.Provide(pool)
	protected := protectedAssetReferences()
	workspacePorts := workspace.Provide(pool, protected)
	projectPorts := project.Provide(pool, protected)
	issuePorts := issue.Provide(pool, protected)
	planningPorts := planning.Provide(pool, planningProjects{projects: projectPorts.PlanningProjects})
	imageBindings := asset.NewImageBindings(assetPorts, businessClock)
	imageReader := asset.NewImageReader(assetPorts, assets.store, assets.runtimeNow)
	assetPurger := asset.NewPurger(assetPorts, assetImageReferences{identityPorts.ImageReferences, workspacePorts.ImageReferences, projectPorts.ImageReferences}, tx)
	authorizer := access.New(access.Deps{
		WorkspaceRoles: workspacePorts.WorkspaceRoles,
		ProjectAccess:  accessProjects{projects: projectPorts.ProjectAccess},
	})
	client, err := jobs.NewInserter(pool)
	if err != nil {
		a.close()
		return nil, err
	}
	inserter := jobInserter{client: client}
	events, err := newDispatcher(eventTypes(), slices.Concat(issue.Subscribers(issuePorts, inserter), planning.Subscribers(inserter), asset.Subscribers(assetPorts, inserter)))
	if err != nil {
		a.close()
		return nil, err
	}
	projectWrites := newWriteGate(cfg.Issue.WriteGate)
	issueTx := newIssueJobTx(pool, cfg)
	// Each read-only snapshot transaction is its own (issue, planning and file reads).
	readTx := postgres.NewTxManager(pool, cfg.Database.CommitTimeout, postgres.ReadOnlySnapshot())
	// One link-preview crawler for the links of the work items and of the modules (M6 P2 spec R12).
	previews := newLinkPreviews(outboundClient, logger)
	attachmentReads := asset.NewAttachmentReads(assetPorts)
	issueAttachmentReads := issueAttachments{reader: attachmentReads, counter: attachmentReads, catalog: attachmentReads}
	issues := issue.New(issue.Deps{
		AttachmentReader: issueAttachmentReads, AttachmentCounter: issueAttachmentReads, AttachmentCatalog: issueAttachmentReads,
		ContentBindings:    issueContentBindings{bindings: asset.NewContentBindings(assetPorts, businessClock)},
		DraftAssetTransfer: issueDraftAssetTransfer{assets: asset.NewDraftAssetTransfer(assetPorts, businessClock, events)},
		DraftAssets:        issueDraftAssets{assets: assetPorts.DraftAssets},
		LinkCrawler:        previews, Logger: logger,
		Events: events,
		Pool:   pool, Tx: busyTx{tx: issueTx}, JobTx: issueTx, ArchivePolicies: projectPorts.ArchivePolicies, AutoArchiveInterval: cfg.Issue.AutoArchiveInterval, TrimVersionsInterval: cfg.Issue.TrimVersionsInterval,
		DeleteBatch: cfg.Issue.DeleteBatch, DeleteBudget: cfg.Issue.DeleteBudget, RuntimeNow: time.Now,
		ReadTx:    readTx,
		WriteGate: projectWrites, Clock: businessClock, Authorizer: authorizer,
		Projects:   issueProjects{projects: projectPorts.IssueProjects},
		Planning:   issuePlanning{targets: planningPorts.IssueTargets},
		Workspaces: issueWorkspaces{directory: workspacePorts.WorkspaceDirectory},
		Members:    issueMembers{members: workspacePorts.WorkspaceMemberDirectory},
		Profiles:   issueProfiles{profiles: identityPorts.PublicProfiles},
	})
	plans := planning.New(planning.Deps{
		Pool: pool, Tx: busyTx{tx: issueTx}, ReadTx: readTx, JobTx: issueTx, RuntimeNow: time.Now, Logger: logger,
		DeleteBatch: cfg.Planning.DeleteBatch, DeleteBudget: cfg.Planning.DeleteBudget,
		WriteGate: projectWrites, Clock: businessClock, Authorizer: authorizer, Events: events,
		Projects:   planningProjects{projects: projectPorts.PlanningProjects},
		LinkCrawls: inserter, LinkCrawler: previews,
	})
	// project before workspace: workspace's writes take its cascade.
	proj := project.New(project.Deps{
		Caller: identityPorts.CredentialLock, ImageGate: projectImageGate{gate: projectWrites},
		ImageBindings: projectImageBindings{bindings: imageBindings}, Images: projectImageReader{images: imageReader},
		Events: events,
		Pool:   pool, Tx: tx, Clock: businessClock, Authorizer: authorizer,
		Workspaces:   projectWorkspaces{directory: workspacePorts.WorkspaceDirectory},
		Members:      workspacePorts.WorkspaceMembers,
		Issues:       issuePorts.IssueCascade,
		ListSettings: projectListSettings{settings: issuePorts.ListSettings},
	})
	ws := workspace.New(workspace.Deps{
		ImageBindings: workspaceImageBindings{bindings: imageBindings}, Images: workspaceImageReader{images: imageReader}, MemberAvatars: memberAvatarReader{images: imageReader},
		Events: events,
		Pool:   pool, Tx: tx, Clock: businessClock, Logger: logger, Authorizer: authorizer,
		Accounts:        workspaceAccounts{accounts: identityPorts.Accounts},
		Profiles:        workspaceProfiles{profiles: identityPorts.PublicProfiles},
		InvitationMAC:   invitations,
		CallerLock:      identityPorts.CredentialLock,
		Projects:        proj.Cascade(),
		CreationEnabled: cfg.Workspace.CreationEnabled,
	})
	ident, err := identity.New(identity.Deps{
		ImageBindings:   profileImageBindings{bindings: imageBindings},
		Images:          profileImageReader{images: imageReader},
		Pool:            pool,
		Tx:              tx,
		Clock:           businessClock,
		Logger:          logger,
		SignupPolicy:    signupPolicy{enabled: cfg.Auth.SignupEnabled, invitations: ws.SignupInvitations()},
		Memberships:     ws.Deactivator(),
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
	files, err := asset.New(asset.Deps{Events: events, Files: cfg.Files, RuntimeNow: assets.runtimeNow, Transfer: assets.transfer, Sealing: assets.sealing, Ports: assetPorts, Objects: assets.store, Inventory: assets.store, Temporary: assets.store, JobTx: tx,
		PurgeAfter: cfg.Cleanup.PurgeAfter,
		ReadTx:     readTx, UploadSigner: assets.store, ReadSigner: assets.store,
		Parents: assetParentReader{workspaces: workspacePorts.AssetParents, projects: projectPorts.AssetParents, issues: issuePorts.AssetParents},
		Access:  assetapp.AccessDeps{Hosts: assetHostAccess{workspaces: ws.AssetAccess(), projects: proj.AssetAccess(), issues: issues.AssetAccess()}, Workspaces: assetWorkspaceLocks{parents: workspacePorts.AssetParents}, Projects: assetProjectLocks{projects: projectPorts.IssueProjects}, Drafts: assetDraftLocks{parents: issuePorts.AssetParents}, Caller: identityPorts.CredentialLock, Gate: forAssetWrites(projectWrites), Tx: busyTx{tx: issueTx}, CredentialClock: businessClock, Clock: businessClock},
	})
	if err != nil {
		a.close()
		return nil, err
	}
	purge := newPurgeDeleted([]purgeModule{{"asset", forAssetPurge(assetPurger)}, {"issue", forIssuePurge(issuePorts.Purger)}, {"planning", forPlanningPurge(planningPorts.Purger)}, {"project", forProjectPurge(projectPorts.Purger)}, {"workspace", forWorkspacePurge(workspacePorts.Purger)}, {"identity", forIdentityPurge(identityPorts.Purger)}}, businessClock, time.Now, cfg.Cleanup.PurgeAfter, cfg.Cleanup.PurgeBatch, cfg.Files.CleanupBudget)
	definitions := append(slices.Concat(ident.Jobs(), issues.Jobs(), plans.Jobs(), files.Jobs()), purgeJob(purge, cfg.Cleanup.PurgeInterval))
	a.jobs, err = jobs.New(pool, jobs.Config{ShutdownTimeout: cfg.Jobs.ShutdownTimeout, Logger: logger, DefaultWorkers: cfg.Jobs.DefaultWorkers, PeriodicWorkers: cfg.Jobs.PeriodicWorkers}, definitions, slices.Concat(identity.EnqueuedKinds(), issue.EnqueuedKinds(), planning.EnqueuedKinds(), asset.EnqueuedKinds()))
	if err != nil {
		a.close()
		return nil, err
	}
	inst, err := instance.New(instance.Deps{
		SignupEnabled:            cfg.Auth.SignupEnabled,
		WorkspaceCreationEnabled: cfg.Workspace.CreationEnabled,
		FileSizeLimit:            cfg.Files.SizeLimit,
		Clock:                    businessClock,
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
	proj.Register(a.router, api)
	issues.Register(a.router, api)
	plans.Register(a.router, api)
	files.Register(a.router, api, bucket(limiter, "file_transfer", cfg.Files.TransferRate))
	// The web UI takes every path no other pattern claims. It must be the
	// method-less "/": "GET /" and the method-less "/api/" would conflict.
	webHandler, err := webui.Handler(webFiles, webui.Options{StorageOrigin: assets.origin})
	if err != nil {
		a.close()
		return nil, err
	}
	a.router.Handle("/", webHandler)
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
