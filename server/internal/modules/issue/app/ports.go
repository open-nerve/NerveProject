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
}

// LockedProject is a project as ProjectGate's lock answers it: its
// workspace, whether it is archived, and its default assignee, nil for
// none.
type LockedProject struct {
	WorkspaceID       uuid.UUID
	Archived          bool
	DefaultAssigneeID *uuid.UUID
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

// IssueCreator is the work items' store as createIssue writes and reads
// them, in the transaction ctx carries (M4 design 3.5).
type IssueCreator interface {
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
	// GetIssue reads the undeleted work item with its assignees and
	// labels; found is false when there is none.
	GetIssue(ctx context.Context, id uuid.UUID) (i domain.Issue, found bool, err error)
}
