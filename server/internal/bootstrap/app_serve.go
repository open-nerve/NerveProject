package bootstrap

import (
	"io/fs"
	"slices"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/webui"
)

// schedule gives the app its jobs runner: the purge of the deleted rows,
// module by module in the order of purgeModule's list, then the modules'
// jobs and the purge's, and the kinds the modules enqueue, each concatenated
// in its order here.
func (a *app) schedule(w *wiring) error {
	purge := newPurgeDeleted([]purgeModule{{"asset", forAssetPurge(w.assetPurger)}, {"notification", forNotificationPurge(w.notificationPorts.Purger)}, {"navigation", forNavigationPurge(w.navigationPorts.Purger)}, {"issue", forIssuePurge(w.issuePorts.Purger)}, {"planning", forPlanningPurge(w.planningPorts.Purger)}, {"project", forProjectPurge(w.projectPorts.Purger)}, {"workspace", forWorkspacePurge(w.workspacePorts.Purger)}, {"identity", forIdentityPurge(w.identityPorts.Purger)}}, w.businessClock, time.Now, w.cfg.Cleanup.PurgeAfter, w.cfg.Cleanup.PurgeBatch, w.cfg.Files.CleanupBudget)
	definitions := append(slices.Concat(w.ident.Jobs(), w.issues.Jobs(), w.plans.Jobs(), w.files.Jobs(), w.notes.Jobs(), w.views.Jobs()), purgeJob(purge, w.cfg.Cleanup.PurgeInterval))
	var err error
	a.jobs, err = jobs.New(w.pool, jobs.Config{ShutdownTimeout: w.cfg.Jobs.ShutdownTimeout, Logger: w.logger, DefaultWorkers: w.cfg.Jobs.DefaultWorkers, PeriodicWorkers: w.cfg.Jobs.PeriodicWorkers}, definitions, slices.Concat(identity.EnqueuedKinds(), issue.EnqueuedKinds(), planning.EnqueuedKinds(), asset.EnqueuedKinds(), notification.EnqueuedKinds(),
		navigation.EnqueuedKinds()))
	return err
}

// route builds the instance module, then gives the app its router: the
// health checks, the public operations, the API, each module's routes in
// the order here, and last the web UI.
func (a *app) route(w *wiring, webFiles fs.FS) error {
	inst, err := instance.New(instance.Deps{
		SignupEnabled:            w.cfg.Auth.SignupEnabled,
		WorkspaceCreationEnabled: w.cfg.Workspace.CreationEnabled,
		FileSizeLimit:            w.cfg.Files.SizeLimit,
		Clock:                    w.businessClock,
	})
	if err != nil {
		return err
	}

	a.router = httpserver.NewRouter(w.logger,
		httpserver.Check{Name: "database", Run: a.pool.Ping},
		httpserver.Check{Name: "migrations", Run: a.migrator.CheckUpToDate},
	)
	a.publicOperations = slices.Concat(w.ident.PublicOperations(), inst.PublicOperations(), w.ws.PublicOperations())
	api, err := httpserver.NewAPI(httpserver.APIConfig{
		Logger:           w.logger,
		Authenticator:    w.ident.Authenticator(),
		PublicOperations: a.publicOperations,
		MaxBodyBytes:     w.cfg.Server.MaxBodyBytes,
		RequestTimeout:   w.cfg.Server.RequestTimeout,
		TrustedProxies:   w.cfg.Server.TrustedProxies,
		IPv6PrefixLen:    w.cfg.RateLimit.IPv6PrefixLen,
		Anonymous:        bucket(w.limiter, "anonymous", w.cfg.RateLimit.Anonymous),
		Authenticated:    bucket(w.limiter, "authenticated", w.cfg.RateLimit.Authenticated),
		AuthFailure:      bucket(w.limiter, "auth_failure", w.cfg.RateLimit.AuthFailure),
	})
	if err != nil {
		return err
	}
	// Modules mount their generated routes on this root router, next to the
	// platform's /api/ fallback; an /api/v0/ sub-mux would shadow it.
	w.ident.Register(a.router, api)
	inst.Register(a.router, api)
	w.ws.Register(a.router, api)
	w.proj.Register(a.router, api)
	w.issues.Register(a.router, api)
	w.plans.Register(a.router, api)
	w.files.Register(a.router, api, bucket(w.limiter, "file_transfer", w.cfg.Files.TransferRate))
	w.notes.Register(a.router, api)
	w.views.Register(a.router, api)
	// The web UI takes every path no other pattern claims. It must be the
	// method-less "/": "GET /" and the method-less "/api/" would conflict.
	webHandler, err := webui.Handler(webFiles, webui.Options{StorageOrigin: w.assets.origin})
	if err != nil {
		return err
	}
	a.router.Handle("/", webHandler)
	return nil
}
