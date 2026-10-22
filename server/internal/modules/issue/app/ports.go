package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Clock reads the moment a write stamps on its rows, under its lock (M4
// design 3.5).
type Clock interface {
	Now() time.Time
}

// ProjectGate is a project's gate of every write on the work-item side (M4
// design 3.5, 6.5): project.Provide's, converted (bootstrap/ports.go).
type ProjectGate interface {
	// Find reads the workspace of the undeleted project, archived or not,
	// without a lock; found is false when there is none.
	Find(ctx context.Context, projectID uuid.UUID) (workspaceID uuid.UUID, found bool, err error)
	// Lock locks the undeleted project FOR NO KEY UPDATE, its workspace's
	// row not locked, in the transaction ctx carries, and answers it;
	// found is false when there is none, also when it was deleted while the
	// lock waited.
	Lock(ctx context.Context, projectID uuid.UUID) (p LockedProject, found bool, err error)
	// LockMany locks the undeleted projects among projectIDs, each named
	// once, in ascending id order, as Lock does, in that order, in one
	// statement, and answers them by id: a write across projects takes
	// them so (convention 7, M4 design 3.5). One that is not there, also
	// one deleted while the lock waited, is not among them.
	LockMany(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]LockedProject, error)
}

// LockedProject is a project as ProjectGate's lock answers it: its
// workspace, whether it is archived, its default assignee, nil for none,
// and its identifier, which names its work items in their activity (M4
// design 3.13).
type LockedProject struct {
	WorkspaceID       uuid.UUID
	Archived          bool
	DefaultAssigneeID *uuid.UUID
	Identifier        string
}

// WriteGate bounds the writes on one project's work items that hold a
// connection at once, in this process (M4 design 3.5, 11.11):
// bootstrap/gate.go's, over platform/gate.
type WriteGate interface {
	// Enter returns once a write on the projects' work items may begin its
	// transaction, with the function that leaves their gates, which the
	// write calls once its transaction has ended. A write across projects
	// names each once, in ascending id order (convention 7), and enters
	// their gates in that order, waiting for each holding no connection
	// and no lock, only its places in the gates before. A write that may
	// wait no longer is refused (503 server_busy); one whose request is
	// cancelled while it waits ends with context.Canceled. Either holds
	// nothing.
	Enter(ctx context.Context, projectIDs ...uuid.UUID) (leave func(), err error)
}

// Sequences numbers a project's work items (M4 design 3.7, 6.5):
// project.Provide's.
type Sequences interface {
	// Next takes the project's next work-item number, under its lock.
	Next(ctx context.Context, projectID uuid.UUID) (int, error)
}

// StateCatalog reads a project's states (M4 design 6.5), under its lock:
// project.Provide's.
type StateCatalog interface {
	// States reads the project's undeleted states, its triage state too.
	States(ctx context.Context, projectID uuid.UUID) ([]StateRef, error)
}

// StateRef is a state as StateCatalog reads it: its id, its name, its
// group, and whether it is the project's default.
type StateRef struct {
	ID      uuid.UUID
	Name    string
	Group   domain.StateGroup
	Default bool
}

// LabelCatalog reads a project's labels (M4 design 6.5), under its lock:
// project.Provide's.
type LabelCatalog interface {
	// Labels reads those of ids that are the project's undeleted labels.
	Labels(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) ([]LabelRef, error)
}

// LabelRef is a label as LabelCatalog reads it: its id and its name.
type LabelRef struct {
	ID   uuid.UUID
	Name string
}

// ProjectMembers reads a project's memberships (M4 design 6.5), under its
// lock: project.Provide's.
type ProjectMembers interface {
	// Roles reads the project roles of those of the accounts ids that are
	// its active members, by account.
	Roles(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error)
}

// PublicProfiles reads accounts' display names (M4 design 6.5): identity's
// PublicProfiles, converted (bootstrap/ports.go).
type PublicProfiles interface {
	// DisplayNames reads the display name of each of the accounts ids that
	// exists, by account.
	DisplayNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error)
}

// IssueReader reads work items, in the transaction ctx carries, or
// without one (M4 design 3.5, 5.2).
type IssueReader interface {
	// GetIssue reads the undeleted work item, archived or not, with its
	// assignees and labels; found is false when there is none.
	GetIssue(ctx context.Context, id uuid.UUID) (i domain.Issue, found bool, err error)
	// GetIssues reads the undeleted work items among ids, archived or
	// not, each as GetIssue reads it, in id order.
	GetIssues(ctx context.Context, ids []uuid.UUID) ([]domain.Issue, error)
	// SubIssueCounts counts the undeleted, unarchived direct sub-items of
	// each of the work items ids that has any (M4 design 3.9).
	SubIssueCounts(ctx context.Context, actor uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]SubIssueCount, error)
}

// SubIssueCount is a work item's number of undeleted, unarchived direct
// sub-items, as IssueReader counts them for an account: All of them, and
// Own, those the account created.
type SubIssueCount struct {
	All, Own int
}

// SubIssueLister reads the sub-items of a work item (M4 design 3.9).
type SubIssueLister interface {
	// SubIssues reads the undeleted, unarchived work items directly under
	// the work item parent, with their assignees and labels, those the
	// account by created when by is not nil, in the order of their
	// creation, (created_at, id), without a lock.
	SubIssues(ctx context.Context, parent uuid.UUID, by *uuid.UUID) ([]domain.Issue, error)
}

// SubIssueAdder is the work items' store as addSubIssues writes and reads
// them, in the transaction ctx carries (M4 design 3.5, 3.9).
type SubIssueAdder interface {
	IssueReader
	// Ancestry reads the work item and those above it, as IssueUpdater's.
	Ancestry(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error)
	// SetParents puts the undeleted work items r names under its parent.
	SetParents(ctx context.Context, r ParentRows) error
	// InsertActivities inserts the rows in their order.
	InsertActivities(ctx context.Context, rows []ActivityRow) error
}

// RelationCreator is the work items' store as createIssueRelations writes
// and reads them, in the transaction ctx carries (M4 design 3.5, 3.10).
type RelationCreator interface {
	IssueReader
	// Related reads which of the work items ids relate to the work item
	// id: an undeleted relation between them, of any type, either way, in
	// id order.
	Related(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error)
	// InsertRelations inserts the relations in their order.
	InsertRelations(ctx context.Context, rs []domain.Relation) error
	// MarkChanged writes the undeleted work items r names last changed.
	MarkChanged(ctx context.Context, r ChangedRows) error
	// InsertActivities inserts the rows in their order.
	InsertActivities(ctx context.Context, rows []ActivityRow) error
}

// IssueArchiver is the work items' store as archiveIssue and
// unarchiveIssue write and read them, in the transaction ctx carries (M4
// design 3.5).
type IssueArchiver interface {
	IssueReader
	// SetArchivedAt writes the work item's archived_at and its last writer
	// and moment.
	SetArchivedAt(ctx context.Context, a IssueArchival) error
	// InsertActivities inserts the rows in their order.
	InsertActivities(ctx context.Context, rows []ActivityRow) error
}

// WorkspaceDirectory finds a workspace by its slug (M4 design 3.3): the
// workspace module's directory (workspace.Provide), converted
// (bootstrap/ports.go).
type WorkspaceDirectory interface {
	// WorkspaceBySlug reads the id of the undeleted workspace slug names,
	// without a lock; found is false when there is none.
	WorkspaceBySlug(ctx context.Context, slug string) (id uuid.UUID, found bool, err error)
}

// ProjectIdentifiers reads the identifiers of a workspace's projects, the
// projects a work item's identifier may name (M4 design 3.7):
// project.Provide's, converted (bootstrap/ports.go).
type ProjectIdentifiers interface {
	// Identifiers reads the workspace's undeleted projects, archived or
	// not, each with its identifier, without a lock.
	Identifiers(ctx context.Context, workspaceID uuid.UUID) ([]ProjectIdentifier, error)
}

// ProjectIdentifier is a project as ProjectIdentifiers reads it: its id
// and its identifier, as stored.
type ProjectIdentifier struct {
	ID         uuid.UUID
	Identifier string
}

// IssueNumbers finds a work item by its number in its project (M4 design
// 3.7).
type IssueNumbers interface {
	// IssueByNumber reads the id of the project's undeleted work item,
	// archived or not, of the number, without a lock; found is false when
	// there is none.
	IssueByNumber(ctx context.Context, projectID uuid.UUID, number int) (id uuid.UUID, found bool, err error)
}

// IssueCreator is the work items' store as createIssue writes and reads
// them, in the transaction ctx carries (M4 design 3.5).
type IssueCreator interface {
	IssueReader
	// GreatestSortOrder reads the greatest sort order of the project's
	// undeleted work items in the state, archived ones too, nil for none.
	GreatestSortOrder(ctx context.Context, projectID, stateID uuid.UUID) (*float64, error)
	InsertIssue(ctx context.Context, r IssueRow) error
	InsertAssignees(ctx context.Context, r LinkRows) error
	InsertLabels(ctx context.Context, r LinkRows) error
	// InsertSubscribers inserts the rows, but of an account that
	// subscribes to the work item already.
	InsertSubscribers(ctx context.Context, r LinkRows) error
	// InsertActivities inserts the rows in their order.
	InsertActivities(ctx context.Context, rows []ActivityRow) error
}

// IssueUpdater is the work items' store as updateIssue writes and reads
// them, in the transaction ctx carries (M4 design 3.5).
type IssueUpdater interface {
	IssueReader
	// Ancestry reads the work item and those above it, its parent, its
	// parent's parent and so on, each once, in no order: a work item that
	// is in it may not be put under it (M4 design 3.9).
	Ancestry(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error)
	// UpdateIssue writes the work item's fields.
	UpdateIssue(ctx context.Context, f IssueFields) error
	InsertAssignees(ctx context.Context, r LinkRows) error
	// RemoveAssignees soft-deletes the work item's undeleted assignees'
	// rows of the accounts r names.
	RemoveAssignees(ctx context.Context, r UnlinkRows) error
	InsertLabels(ctx context.Context, r LinkRows) error
	// RemoveLabels soft-deletes the work item's undeleted labels' rows of
	// the labels r names.
	RemoveLabels(ctx context.Context, r UnlinkRows) error
	// InsertSubscribers inserts the rows, but of an account that
	// subscribes to the work item already.
	InsertSubscribers(ctx context.Context, r LinkRows) error
	// InsertActivities inserts the rows in their order.
	InsertActivities(ctx context.Context, rows []ActivityRow) error
}
