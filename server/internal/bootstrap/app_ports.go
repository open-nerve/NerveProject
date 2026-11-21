package bootstrap

import (
	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/outbound"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// provide is the first of the two steps (M3 design 6.6): every module's
// Provide, from the pool alone, then what is built from their ports and
// the pool before any module's New: the image bindings and reader, the
// files' purger and access's three rules.
func (w *wiring) provide() {
	w.identityPorts = identity.Provide(w.pool)
	w.assetPorts = asset.Provide(w.pool)
	protected := protectedAssetReferences()
	w.workspacePorts = workspace.Provide(w.pool, protected)
	w.projectPorts = project.Provide(w.pool, protected)
	w.issuePorts = issue.Provide(w.pool, protected)
	w.planningPorts = planning.Provide(w.pool, planningProjects{projects: w.projectPorts.PlanningProjects})
	w.notificationPorts = notification.Provide(w.pool)
	w.navigationPorts = navigation.Provide(w.pool)
	w.imageBindings = asset.NewImageBindings(w.assetPorts, w.businessClock)
	w.imageReader = asset.NewImageReader(w.assetPorts, w.assets.store, w.assets.runtimeNow)
	w.assetPurger = asset.NewPurger(w.assetPorts, assetImageReferences{w.identityPorts.ImageReferences, w.workspacePorts.ImageReferences, w.projectPorts.ImageReferences}, w.tx)
	w.authorizer = access.New(access.Deps{
		WorkspaceRoles: w.workspacePorts.WorkspaceRoles,
		ProjectAccess:  accessProjects{projects: w.projectPorts.ProjectAccess},
	})
	// The recheck of a notification's recipients, Authorize's rule for many
	// accounts at once (M7 P1b spec 2.7).
	w.decider = access.NewDecider(access.DeciderDeps{
		WorkspaceRoles: w.workspacePorts.WorkspaceRolesMany,
		ProjectAccess:  accessProjectsMany{projects: w.projectPorts.ProjectAccessMany},
	})
	// The projects a notification's reader reads now, Authorize's rule of
	// issue.read for each (M7 P1c spec 2.5).
	w.readable = access.NewReadableProjects(access.ReadableDeps{
		WorkspaceRoles: w.workspacePorts.WorkspaceRoles,
		Projects:       accessMemberProjects{projects: w.projectPorts.MemberProjectAccess},
	})
}

// dispatchEvents builds the job inserter, then on it the dispatcher every
// module publishes to, with every module's subscribers (eventSubscribers).
func (w *wiring) dispatchEvents() error {
	client, err := jobs.NewInserter(w.pool)
	if err != nil {
		return err
	}
	w.inserter = jobInserter{client: client}
	w.events, err = newDispatcher(eventTypes(), eventSubscribers(w.issuePorts, w.assetPorts, w.notificationPorts, w.inserter))
	return err
}

// share builds what more than one module takes: the projects' write gate,
// the job transaction, the read-only snapshot transaction and the
// link-preview crawler on the outbound client.
func (w *wiring) share(outboundClient *outbound.Client) {
	w.projectWrites = newWriteGate(w.cfg.Issue.WriteGate)
	w.issueTx = newIssueJobTx(w.pool, w.cfg)
	// Each read-only snapshot transaction is its own (issue, planning and file reads).
	w.readTx = postgres.NewTxManager(w.pool, w.cfg.Database.CommitTimeout, postgres.ReadOnlySnapshot())
	// One link-preview crawler for the links of the work items and of the modules (M6 P2 spec R12).
	w.previews = newLinkPreviews(outboundClient, w.logger)
}
