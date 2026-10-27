// Package issue is the work items module (M4 design 3.3, 6.2): it owns the
// work items' tables and brings creating, reading, by id or by identifier,
// changing, archiving, unarchiving and deleting a work item, listing the
// work items under one and putting work items under it, and listing a
// work item's relations, relating it to others and deleting a relation.
// For the project module's deletions (its IssueCascade port) it
// soft-deletes a deleted project's rows of them, tells whether work items
// are in a state, and takes deleted labels off them.
package issue

import (
	"context"
	"log/slog"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/content"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/crawler"
	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/outbound"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// deleteBatch is the rows a statement of a project's deletion soft-deletes
// at most (M4 design 3.6).
const deleteBatch = 1000

// Cascade is what the project module's deletions ask of the work items:
// project's IssueCascade port (M4 design 3.3). Each method runs in the
// caller's transaction, which ctx carries, and returns a failure as
// itself, so the caller's whole write rolls back; one that deletes writes
// by and now into the rows it deletes.
type Cascade interface {
	// DeleteProjectIssues soft-deletes the project's work-item-side rows.
	DeleteProjectIssues(ctx context.Context, projectID, by uuid.UUID, now time.Time) error
	// DeleteWorkspaceIssues soft-deletes the work-item-side rows of each
	// of the projects, and the workspace drafts without a project.
	DeleteWorkspaceIssues(ctx context.Context, workspaceID uuid.UUID, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error
	// StateInUse reports whether an undeleted work item, archived or not,
	// is in the state (M4 design 3.6).
	StateInUse(ctx context.Context, stateID uuid.UUID) (bool, error)
	// DetachLabel soft-deletes the rows that put the labels on work items.
	DetachLabel(ctx context.Context, labelIDs []uuid.UUID, by uuid.UUID, now time.Time) error
}

// ListCalendar and ListDisplayFilters are the pure saved-settings boundary values.
type ListCalendar = app.ListCalendar
type ListDisplayFilters = app.ListDisplayFilters

// ListSettings validates saved list words without a directory or actor.
type ListSettings interface {
	Validate(map[string][]string, *ListDisplayFilters, map[string]bool) []shared.FieldError
}

// Provided are the adapters bootstrap builds before wiring consuming modules.
type Provided struct {
	IssueCascade Cascade
	ListSettings ListSettings
	Purger       *app.Purger
}

// Provide builds issue's adapters for the other modules.
func Provide(pool *pgxpool.Pool) Provided {
	store := postgresadapter.New(pool)
	return Provided{Purger: app.NewPurger([]app.PurgeStep{
		{Table: "comment_reactions", Delete: store.PurgeCommentReactions},
		{Table: "issue_reactions", Delete: store.PurgeIssueReactions},
		{Table: "issue_mentions", Delete: store.PurgeIssueMentions},
		{Table: "issue_description_versions", Delete: store.PurgeIssueDescriptionVersions},
		{Table: "issue_activities", Delete: store.PurgeIssueActivities},
		{Table: "issue_comments", Delete: store.PurgeIssueComments},
		{Table: "issue_links", Delete: store.PurgeIssueLinks},
		{Table: "issue_relations", Delete: store.PurgeIssueRelations},
		{Table: "issue_assignees", Delete: store.PurgeIssueAssignees},
		{Table: "issue_labels", Delete: store.PurgeIssueLabels},
		{Table: "issue_subscribers", Delete: store.PurgeIssueSubscribers},
		{Table: "draft_issues", Delete: store.PurgeDraftIssues},
		{Table: "issues", Delete: store.PurgeIssues},
	}), IssueCascade: app.NewCascade(store, store, deleteBatch), ListSettings: app.ListSettings{}}
}

// ProjectPorts are what the module asks of a project (M4 design 6.5): its
// gate, its numbers, its states', labels' and members' catalogs, for its
// writes; a workspace's projects' identifiers, for a read by identifier;
// ordinary and historical memberships, and complete list state and label
// catalogs.
// project.Provide's IssueProjects is them, converted (bootstrap/ports.go).
type ProjectPorts interface {
	app.RetentionProjects
	app.ProjectGate
	app.DraftProjects
	app.Sequences
	app.StateCatalog
	app.LabelCatalog
	app.ProjectMembers
	app.ProjectIdentifiers
	app.VisibleProjects
	app.ListStateCatalog
	app.ListLabelCatalog
}

// LockedProject, StateRef and its StateGroup, LabelRef and
// ProjectIdentifier are the values ProjectPorts hand over: bootstrap
// converts project's into them (M4 design 6.5).
type (
	VisibleProject    = app.VisibleProject
	MemberVisibility  = app.MemberVisibility
	LockedProject     = app.LockedProject
	StateRef          = app.StateRef
	StateGroup        = domain.StateGroup
	LabelRef          = app.LabelRef
	ProjectIdentifier = app.ProjectIdentifier
)

// Deps are what bootstrap gives the module (M4 design 6.6, step 6).
type Deps struct {
	Outbound *outbound.Client
	Logger   *slog.Logger
	Events   shared.EventPublisher
	Pool     *pgxpool.Pool
	// Tx are the transactions of the module's writes, each of which waits
	// for a lock at most issue.write_lock_timeout (bootstrap/gate.go).
	Tx shared.TxManager
	// JobTx retains PostgreSQL errors for River retries, with the same lock timeout.
	JobTx                shared.TxManager
	ArchivePolicies      app.ArchivePolicies
	AutoArchiveInterval  time.Duration
	TrimVersionsInterval time.Duration
	DeleteBatch          int
	DeleteBudget         time.Duration
	RuntimeNow           func() time.Time
	// ReadTx supplies independent, read-only repeatable-read transactions for
	// collaboration lists. The first target read, authorization, directories
	// and list queries must all use its context; never call such a list from
	// an existing write transaction, which WithinTx would join unchanged.
	ReadTx shared.TxManager
	// WriteGate is the write gate of each project (bootstrap/gate.go).
	WriteGate  app.WriteGate
	Clock      app.Clock
	Authorizer shared.Authorizer
	// Projects are project's IssueProjects, converted (bootstrap/ports.go).
	Projects ProjectPorts
	// Workspaces is workspace's WorkspaceDirectory, converted
	// (bootstrap/ports.go).
	Workspaces interface {
		app.WorkspaceDirectory
		app.WorkspaceSharer
	}
	Members app.WorkspaceMemberDirectory
	// Profiles is identity's PublicProfiles, converted (bootstrap/ports.go).
	Profiles app.PublicProfiles
}

// Module is the wired issue module.
type Module struct {
	deleteRows           *app.DeleteRows
	deleteBudget         time.Duration
	logger               *slog.Logger
	uc                   httpadapter.UseCases
	trimVersions         *app.TrimVersions
	trimVersionsInterval time.Duration
	autoArchive          *app.AutoArchive
	crawlLink            *app.CrawlLink
	autoArchiveInterval  time.Duration
}

// New wires the module's use cases and its HTTP side from d; nothing is
// registered or injected after it (M4 design 6.6).
func New(d Deps) *Module {
	store := postgresadapter.New(d.Pool)
	locks := app.NewLocks(d.Projects, store, d.Authorizer, d.WriteGate, d.Tx)
	draftLocks := app.NewDraftLocks(locks, d.Workspaces, d.Projects, store)
	sanitizer := content.NewSanitizer()
	listing := app.NewListing(app.ListingDeps{Rows: store, States: d.Projects, Labels: d.Projects, Assignees: store, Profiles: d.Profiles, Issues: store})
	archive := app.ArchiveIssueDeps{Events: d.Events, Locks: locks, Issues: store, States: d.Projects, Clock: d.Clock}
	creation := app.NewIssueCreation(app.IssueCreationDeps{Events: d.Events,
		Issues: store, Sequences: d.Projects, States: d.Projects, Labels: d.Projects, Members: d.Projects,
		Profiles: d.Profiles, Clock: d.Clock, Sanitizer: sanitizer,
	})
	return &Module{
		deleteRows: app.NewDeleteRows(app.DeleteRowsDeps{Rows: store, Relations: app.NewRelationDeletions(store, d.Projects), Remaining: store, Tx: d.JobTx, RuntimeNow: d.RuntimeNow, Batch: d.DeleteBatch, Budget: d.DeleteBudget}), deleteBudget: d.DeleteBudget, logger: d.Logger,
		trimVersions: app.NewTrimVersions(app.TrimVersionsDeps{Projects: d.Projects, Gate: d.Projects, Rows: store, Tx: d.JobTx}), trimVersionsInterval: d.TrimVersionsInterval,
		crawlLink: app.NewCrawlLink(app.CrawlLinkDeps{Crawler: crawler.New(d.Outbound, d.Logger), Links: store}), autoArchive: app.NewAutoArchive(app.AutoArchiveDeps{Policies: d.ArchivePolicies, Projects: d.Projects, States: d.Projects, Issues: store, Tx: d.JobTx, Clock: d.Clock, Events: d.Events}), autoArchiveInterval: d.AutoArchiveInterval,
		uc: httpadapter.UseCases{
			SearchIssues:            app.NewSearchIssues(app.SearchIssuesDeps{ReadTx: d.ReadTx, Authorizer: d.Authorizer, Workspaces: d.Workspaces, Projects: d.Projects, Identifiers: d.Projects, Issues: store, Locks: locks, Rows: store}),
			PublishDraft:            app.NewPublishDraft(app.PublishDraftDeps{Locks: draftLocks, Decoder: httpadapter.IssueCreatePayloadDecoder{}, Creation: creation, Drafts: store}),
			ListDrafts:              app.NewListDrafts(app.ListDraftsDeps{ReadTx: d.ReadTx, Locks: draftLocks, Workspaces: d.Workspaces, Projects: d.Projects, Drafts: store}),
			GetDraft:                app.NewGetDraft(app.GetDraftDeps{ReadTx: d.ReadTx, Locks: draftLocks}),
			CreateDraft:             app.NewCreateDraft(app.CreateDraftDeps{Locks: draftLocks, Workspaces: d.Workspaces, Drafts: store, Clock: d.Clock}),
			UpdateDraft:             app.NewUpdateDraft(app.UpdateDraftDeps{Locks: draftLocks, Drafts: store, Clock: d.Clock}),
			DeleteDraft:             app.NewDeleteDraft(app.DeleteDraftDeps{Locks: draftLocks, Drafts: store, Clock: d.Clock}),
			UpdateIssueLink:         app.NewUpdateIssueLink(app.UpdateIssueLinkDeps{Locks: locks, Links: store, Clock: d.Clock, Events: d.Events}),
			DeleteIssueLink:         app.NewDeleteIssueLink(app.DeleteIssueLinkDeps{Locks: locks, Links: store, Clock: d.Clock, Events: d.Events}),
			CreateIssueLink:         app.NewCreateIssueLink(app.CreateIssueLinkDeps{Locks: locks, Links: store, Clock: d.Clock, Events: d.Events}),
			ListIssueLinks:          app.NewListIssueLinks(app.ListIssueLinksDeps{Locks: locks, Links: store, ReadTx: d.ReadTx}),
			ListMemberIssues:        app.NewListMemberIssues(app.ListMemberIssuesDeps{ReadTx: d.ReadTx, Authorizer: d.Authorizer, Workspaces: d.Workspaces, Members: d.Members, Projects: d.Projects, Listing: listing}),
			ListIssues:              app.NewListIssues(app.ListIssuesDeps{ReadTx: d.ReadTx, Locks: locks, Listing: listing}),
			ListIssueActivities:     app.NewListIssueActivities(app.ListIssueActivitiesDeps{Locks: locks, Activities: store, Projects: d.Projects, ReadTx: d.ReadTx}),
			ListDescriptionVersions: app.NewListDescriptionVersions(app.ListDescriptionVersionsDeps{Locks: locks, Versions: store, ReadTx: d.ReadTx}),
			GetDescriptionVersion:   app.NewGetDescriptionVersion(app.GetDescriptionVersionDeps{Locks: locks, Versions: store}),
			AddCommentReaction:      app.NewAddCommentReaction(app.AddCommentReactionDeps{Events: d.Events, Locks: locks, Comments: store, Reactions: store, Clock: d.Clock}),
			RemoveCommentReaction:   app.NewRemoveCommentReaction(app.RemoveCommentReactionDeps{Events: d.Events, Locks: locks, Comments: store, Reactions: store, Clock: d.Clock}),
			AddIssueReaction:        app.NewAddIssueReaction(app.AddIssueReactionDeps{Events: d.Events, Locks: locks, Reactions: store, Clock: d.Clock}),
			RemoveIssueReaction:     app.NewRemoveIssueReaction(app.RemoveIssueReactionDeps{Events: d.Events, Locks: locks, Reactions: store, Clock: d.Clock}),
			ListIssueReactions:      app.NewListIssueReactions(app.ListIssueReactionsDeps{Locks: locks, Reactions: store, ReadTx: d.ReadTx}),
			UpdateIssueComment:      app.NewUpdateIssueComment(app.UpdateIssueCommentDeps{Events: d.Events, Locks: locks, Comments: store, Members: d.Projects, Sanitizer: sanitizer, Clock: d.Clock}),
			DeleteIssueComment:      app.NewDeleteIssueComment(app.DeleteIssueCommentDeps{Events: d.Events, Locks: locks, Comments: store, Clock: d.Clock}),
			CreateIssueComment:      app.NewCreateIssueComment(app.CreateIssueCommentDeps{Events: d.Events, Locks: locks, Comments: store, Members: d.Projects, Sanitizer: sanitizer, Clock: d.Clock}),
			ListIssueComments:       app.NewListIssueComments(app.ListIssueCommentsDeps{Locks: locks, Comments: store, ReadTx: d.ReadTx}),
			CreateIssue:             app.NewCreateIssue(app.CreateIssueDeps{Locks: locks, Creation: creation}),
			GetIssue:                app.NewGetIssue(locks),
			GetIssueByIdentifier: app.NewGetIssueByIdentifier(app.GetIssueByIdentifierDeps{
				Locks: locks, Workspaces: d.Workspaces, Projects: d.Projects, Issues: store,
			}),
			UpdateIssue: app.NewUpdateIssue(app.UpdateIssueDeps{Events: d.Events,
				Locks: locks, Issues: store, States: d.Projects, Labels: d.Projects, Members: d.Projects, Profiles: d.Profiles, Clock: d.Clock, Sanitizer: sanitizer,
			}),
			DeleteIssue:        app.NewDeleteIssue(app.DeleteIssueDeps{Events: d.Events, Locks: locks, Rows: store, Relations: app.NewRelationDeletions(store, d.Projects), Clock: d.Clock}),
			ArchiveIssue:       app.NewArchiveIssue(archive),
			UnarchiveIssue:     app.NewUnarchiveIssue(archive),
			ListSubIssues:      app.NewListSubIssues(locks, store),
			AddSubIssues:       app.NewAddSubIssues(app.AddSubIssuesDeps{Events: d.Events, Locks: locks, Issues: store, Clock: d.Clock}),
			ListIssueRelations: app.NewListIssueRelations(app.ListIssueRelationsDeps{Locks: locks, Relations: store, Projects: d.Projects, ReadTx: d.ReadTx}),
			CreateIssueRelations: app.NewCreateIssueRelations(app.CreateIssueRelationsDeps{Events: d.Events,
				Locks: locks, Relations: store, Clock: d.Clock,
			}),
			DeleteIssueRelation: app.NewDeleteIssueRelation(app.DeleteIssueRelationDeps{Events: d.Events,
				Locks: locks, Relations: store, Clock: d.Clock,
			}),
		}}
}

// Register mounts the module's API on router behind api's middlewares.
func (m *Module) Register(router *httpserver.Router, api *httpserver.API) {
	httpadapter.Register(router, api, m.uc)
}

// Actions lists the module's actions: bootstrap's test holds the union of
// every module's actions equal to access's rule table (M3 design 3.4).
func Actions() []shared.Action {
	return domain.Actions()
}
