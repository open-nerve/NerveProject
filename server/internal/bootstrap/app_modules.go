package bootstrap

import (
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
)

// modules is the second of the two steps (M3 design 6.6): each module's New,
// with every port it needs, after every Provide (provide). Three of the
// modules take another's use cases, which orders them: workspace takes
// project's cascade, identity workspace's signup invitations and
// deactivator, and asset the AssetAccess of issue, project and workspace.
// The first failure stops the step; nothing after it is built.
func (w *wiring) modules() error {
	w.issues = w.issueModule()
	w.plans = w.planningModule()
	w.notes = w.notificationModule()
	w.views = w.navigationModule()
	// project before workspace: workspace's writes take its cascade.
	w.proj = w.projectModule()
	// workspace before identity: identity's signup and deactivation take
	// workspace's invitations and deactivator.
	w.ws = w.workspaceModule()
	var err error
	if w.ident, err = w.identityModule(); err != nil {
		return err
	}
	// asset after issue, project and workspace: it takes their AssetAccess.
	w.files, err = w.assetModule()
	return err
}

func (w *wiring) issueModule() *issue.Module {
	attachmentReads := asset.NewAttachmentReads(w.assetPorts)
	issueAttachmentReads := issueAttachments{reader: attachmentReads, counter: attachmentReads, catalog: attachmentReads}
	return issue.New(issue.Deps{
		AttachmentReader: issueAttachmentReads, AttachmentCounter: issueAttachmentReads, AttachmentCatalog: issueAttachmentReads,
		ContentBindings:    issueContentBindings{bindings: asset.NewContentBindings(w.assetPorts, w.businessClock)},
		DraftAssetTransfer: issueDraftAssetTransfer{assets: asset.NewDraftAssetTransfer(w.assetPorts, w.businessClock, w.events)},
		DraftAssets:        issueDraftAssets{assets: w.assetPorts.DraftAssets},
		LinkCrawler:        w.previews, Logger: w.logger,
		Events: w.events, ArchiveEvents: w.events,
		Pool: w.pool, Tx: busyTx{tx: w.issueTx}, JobTx: w.issueTx, ArchivePolicies: w.projectPorts.ArchivePolicies, AutoArchiveInterval: w.cfg.Issue.AutoArchiveInterval, TrimVersionsInterval: w.cfg.Issue.TrimVersionsInterval,
		DeleteBatch: w.cfg.Issue.DeleteBatch, DeleteBudget: w.cfg.Issue.DeleteBudget, RuntimeNow: time.Now,
		ReadTx:    w.readTx,
		WriteGate: w.projectWrites, Clock: w.businessClock, Authorizer: w.authorizer,
		Projects:   issueProjects{projects: w.projectPorts.IssueProjects},
		Planning:   issuePlanning{targets: w.planningPorts.IssueTargets, directory: w.planningPorts.IssueDirectory},
		Workspaces: issueWorkspaces{directory: w.workspacePorts.WorkspaceDirectory},
		Members:    issueMembers{members: w.workspacePorts.WorkspaceMemberDirectory},
		Profiles:   issueProfiles{profiles: w.identityPorts.PublicProfiles},
	})
}

func (w *wiring) planningModule() *planning.Module {
	return planning.New(planning.Deps{
		Pool: w.pool, Tx: busyTx{tx: w.issueTx}, ReadTx: w.readTx, JobTx: w.issueTx, RuntimeNow: time.Now, Logger: w.logger,
		DeleteBatch: w.cfg.Planning.DeleteBatch, DeleteBudget: w.cfg.Planning.DeleteBudget, FreezeInterval: w.cfg.Planning.FreezeInterval,
		WriteGate: w.projectWrites, Clock: w.businessClock, Authorizer: w.authorizer, Events: w.events,
		Projects:      planningProjects{projects: w.projectPorts.PlanningProjects},
		IssueStats:    planningIssues{stats: w.issuePorts.PlanningStats},
		IssueTransfer: planningTransfer{transfer: w.issuePorts.PlanningTransfer},
		ListSettings:  w.issuePorts.ListSettings,
		LinkCrawls:    w.inserter, LinkCrawler: w.previews,
	})
}

func (w *wiring) notificationModule() *notification.Module {
	// A change of a notification waits for its row at most
	// issue.write_lock_timeout, 503 past it (busyTx); mark-all-read's
	// batches never wait for a row (SKIP LOCKED) and run in tx (M7 P1c spec
	// 2.7, R5).
	return notification.New(notification.Deps{
		Pool: w.pool, Issues: notificationIssues{issues: w.issuePorts.NotificationIssues}, JobTx: w.issueTx, RuntimeNow: time.Now, Logger: w.logger,
		Projects: notificationProjects{names: w.projectPorts.NotificationNames}, Readers: notificationReaders{decider: w.decider},
		DeleteBatch: w.cfg.Notification.DeleteBatch, DeleteBudget: w.cfg.Notification.DeleteBudget,
		Workspaces: notificationWorkspaces{directory: w.workspacePorts.WorkspaceDirectory}, Authorizer: w.authorizer,
		Readable:  notificationReadable{readable: w.readable},
		IssueRefs: notificationIssueRefs{issues: w.issuePorts.IssueRefs, projects: w.projectPorts.NotificationIdentifiers},
		Clock:     w.businessClock, ReadTx: w.readTx, Tx: busyTx{tx: w.issueTx}, MarkTx: w.tx,
	})
}

func (w *wiring) navigationModule() *navigation.Module {
	// A view's write enters no write gate and answers no server_busy (M7 P3b spec 2.4): tx.
	return navigation.New(navigation.Deps{Pool: w.pool, Tx: w.tx, ReadTx: w.readTx, JobTx: w.issueTx, RuntimeNow: time.Now, Logger: w.logger,
		DeleteBatch: w.cfg.Navigation.DeleteBatch, DeleteBudget: w.cfg.Navigation.DeleteBudget, Clock: w.businessClock, Authorizer: w.authorizer,
		Projects: navigationProjects{projects: w.projectPorts.NavigationProjects}, Workspaces: navigationWorkspaces{directory: w.workspacePorts.WorkspaceDirectory},
		ListSettings: w.issuePorts.ListSettings, WorkspaceListSettings: w.issuePorts.WorkspaceListSettings})
}

func (w *wiring) projectModule() *project.Module {
	return project.New(project.Deps{
		Caller: w.identityPorts.CredentialLock, ImageGate: projectImageGate{gate: w.projectWrites},
		ImageBindings: projectImageBindings{bindings: w.imageBindings}, Images: projectImageReader{images: w.imageReader},
		Events: w.events,
		Pool:   w.pool, Tx: w.tx, Clock: w.businessClock, Authorizer: w.authorizer,
		Workspaces:   projectWorkspaces{directory: w.workspacePorts.WorkspaceDirectory},
		Members:      w.workspacePorts.WorkspaceMembers,
		Issues:       w.issuePorts.IssueCascade,
		ListSettings: w.issuePorts.ListSettings,
	})
}

// workspaceModule takes project's cascade (w.proj).
func (w *wiring) workspaceModule() *workspace.Module {
	return workspace.New(workspace.Deps{
		ImageBindings: workspaceImageBindings{bindings: w.imageBindings}, Images: workspaceImageReader{images: w.imageReader}, MemberAvatars: memberAvatarReader{images: w.imageReader},
		Events: w.events,
		Pool:   w.pool, Tx: w.tx, Clock: w.businessClock, Logger: w.logger, Authorizer: w.authorizer,
		Accounts:        workspaceAccounts{accounts: w.identityPorts.Accounts},
		Profiles:        workspaceProfiles{profiles: w.identityPorts.PublicProfiles},
		InvitationMAC:   w.invitations,
		CallerLock:      w.identityPorts.CredentialLock,
		Projects:        w.proj.Cascade(),
		CreationEnabled: w.cfg.Workspace.CreationEnabled,
	})
}

// identityModule takes workspace's signup invitations and deactivator (w.ws).
func (w *wiring) identityModule() (*identity.Module, error) {
	return identity.New(identity.Deps{
		ImageBindings:   profileImageBindings{bindings: w.imageBindings},
		Images:          profileImageReader{images: w.imageReader},
		Pool:            w.pool,
		Tx:              w.tx,
		Clock:           w.businessClock,
		Logger:          w.logger,
		SignupPolicy:    signupPolicy{enabled: w.cfg.Auth.SignupEnabled, invitations: w.ws.SignupInvitations()},
		Memberships:     w.ws.Deactivator(),
		Keys:            w.keys,
		AccessTokenTTL:  w.cfg.Auth.AccessTokenTTL,
		SessionTTL:      w.cfg.Auth.SessionTTL,
		RefreshDeadline: w.cfg.Auth.RefreshDeadline,
		// The periodic job that deletes the expired sessions (M2 design 3.15).
		SessionCleanupInterval: w.cfg.Auth.SessionCleanupInterval,
		Password:               passwordHashing(w.cfg.Auth.Password),
		RateLimits: identity.RateLimits{
			Limiter:      w.limiter,
			LoginIP:      bucket(w.limiter, "login_ip", w.cfg.RateLimit.LoginIP),
			LoginIPEmail: bucket(w.limiter, "login_ip_email", w.cfg.RateLimit.LoginIPEmail),
			RegisterIP:   bucket(w.limiter, "register_ip", w.cfg.RateLimit.RegisterIP),
			PasswordUser: bucket(w.limiter, "password_user", w.cfg.RateLimit.PasswordUser),
		},
	})
}

// assetModule takes the AssetAccess of workspace, project and issue (w.ws,
// w.proj, w.issues).
func (w *wiring) assetModule() (*asset.Module, error) {
	return asset.New(asset.Deps{Events: w.events, Files: w.cfg.Files, RuntimeNow: w.assets.runtimeNow, Transfer: w.assets.transfer, Sealing: w.assets.sealing, Ports: w.assetPorts, Objects: w.assets.store, Inventory: w.assets.store, Temporary: w.assets.store, JobTx: w.tx,
		PurgeAfter: w.cfg.Cleanup.PurgeAfter,
		ReadTx:     w.readTx, UploadSigner: w.assets.store, ReadSigner: w.assets.store,
		Parents: assetParentReader{workspaces: w.workspacePorts.AssetParents, projects: w.projectPorts.AssetParents, issues: w.issuePorts.AssetParents},
		Access:  assetapp.AccessDeps{Hosts: assetHostAccess{workspaces: w.ws.AssetAccess(), projects: w.proj.AssetAccess(), issues: w.issues.AssetAccess()}, Workspaces: assetWorkspaceLocks{parents: w.workspacePorts.AssetParents}, Projects: assetProjectLocks{projects: w.projectPorts.IssueProjects}, Drafts: assetDraftLocks{parents: w.issuePorts.AssetParents}, Caller: w.identityPorts.CredentialLock, Gate: forAssetWrites(w.projectWrites), Tx: busyTx{tx: w.issueTx}, CredentialClock: w.businessClock, Clock: w.businessClock},
	})
}
