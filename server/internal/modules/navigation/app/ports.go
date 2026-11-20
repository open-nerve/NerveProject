package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ProjectFacts are a project's facts as the module reads them (P3b spec
// 2.1): its workspace, and whether its views are turned on
// (issue_views_view), which a write of them requires and a read does not
// (M7 design 3.7). An archived project's views are read and written as
// ever (R2): the facts do not tell it.
type ProjectFacts struct {
	WorkspaceID    uuid.UUID
	IssueViewsView bool
}

// Projects is what the module asks of a project: its facts without a lock,
// in the transaction ctx carries, if any, for a read and for a write's
// check before its locks; and the same facts under the project's FOR
// SHARE, the project's lock of a write of its views, which the write takes
// after its workspace's FOR SHARE (spec 2.6). found is false for a project
// that is not there or is deleted, also one deleted while the lock waited.
// project.Provide's NavigationProjects is it, converted (bootstrap).
type Projects interface {
	Facts(ctx context.Context, id uuid.UUID) (f ProjectFacts, found bool, err error)
	Share(ctx context.Context, id uuid.UUID) (f ProjectFacts, found bool, err error)
}

// Workspaces finds a workspace by its slug and locks one, in the
// transaction ctx carries, if any: the workspace module's directory, which
// bootstrap converts. found is false for a workspace that is not there or
// is deleted, also one deleted while the lock waited.
type Workspaces interface {
	// WorkspaceBySlug reads the id of the undeleted workspace slug names,
	// without a lock.
	WorkspaceBySlug(ctx context.Context, slug string) (id uuid.UUID, found bool, err error)
	// ShareWorkspaceBySlug reads it locked FOR SHARE until the transaction
	// ends: the first lock of a write of a workspace's views (spec 2.6).
	ShareWorkspaceBySlug(ctx context.Context, slug string) (id uuid.UUID, found bool, err error)
	// ShareWorkspaceByID locks the undeleted workspace id FOR SHARE until
	// the transaction ends: the first lock of a write of a project's
	// views, before the project's.
	ShareWorkspaceByID(ctx context.Context, id uuid.UUID) (found bool, err error)
}

// ListSettings checks the words of a view's saved list settings without a
// directory or an actor (spec 2.4): the issue module's ListSettings, a
// project's list's words, for a project's view, and its
// WorkspaceListSettings for a workspace's, which bootstrap hands over as
// they are. A nil display is omitted. It answers each word it refuses at
// its field (filters.<key>, display_filters.<word>), none for none.
type ListSettings interface {
	Validate(filters map[string][]string, display *shared.ListDisplayFilters) []shared.FieldError
}

// Clock reads the business moment a write stamps, once, after its last
// lock: bootstrap hands over the business clock, UTC at the microsecond.
type Clock interface {
	Now() time.Time
}

// ViewReader reads the undeleted views without a lock, in the transaction
// ctx carries or on the pool (P3b spec 2.4, 2.5). The project's views and
// the workspace's own, those without a project, are each listed the newest
// first, then by id descending; ownedBy, when it is not nil, keeps only the
// views it owns, an OwnOnly grant's list (spec 2.2). View reads one view of
// either kind by its id, the first read of /views/{view_id}; found is false
// for one that is not there or is deleted.
type ViewReader interface {
	ProjectViews(ctx context.Context, project uuid.UUID, ownedBy *uuid.UUID) ([]domain.View, error)
	WorkspaceViews(ctx context.Context, workspace uuid.UUID, ownedBy *uuid.UUID) ([]domain.View, error)
	View(ctx context.Context, id uuid.UUID) (v domain.View, found bool, err error)
}

// ViewWriter writes the views in the transaction ctx carries, which holds
// the view's workspace FOR SHARE and, for a project's view, its project
// FOR SHARE (spec 2.6). LockView reads a view locked FOR UPDATE, found
// false for one that is not there or is deleted. InsertView stores a new
// view made by by at its CreatedAt; UpdateView replaces a view's content
// whole, last changed by by at its UpdatedAt; DeleteView soft-deletes a
// view, deleted and last changed by by at at. A change or deletion of a
// view deleted already changes nothing.
type ViewWriter interface {
	LockView(ctx context.Context, id uuid.UUID) (v domain.View, found bool, err error)
	InsertView(ctx context.Context, v domain.View, by uuid.UUID) error
	UpdateView(ctx context.Context, v domain.View, by uuid.UUID) error
	DeleteView(ctx context.Context, id uuid.UUID, at time.Time, by uuid.UUID) error
}

// DeletionScope is whose views a deletion of the views deletes (P3b spec
// 2.7): a deleted project's, or a deleted workspace's own, those without a
// project; the views of a workspace's projects are their projects'.
type DeletionScope string

// The scopes of a deletion of the views.
const (
	ProjectScope   DeletionScope = "project"
	WorkspaceScope DeletionScope = "workspace"
)

// RowDeletions inserts, in the publishing transaction ctx carries, the
// deletion of the views of the scope's parent id, a deleted project or a
// deleted workspace, at the deletion's moment at and by its actor by
// (spec 2.7): navigation.delete_rows.
type RowDeletions interface {
	Enqueue(ctx context.Context, scope DeletionScope, id uuid.UUID, at time.Time, by uuid.UUID) error
}

// RowBatch is one batch of a deletion of the views: at most Limit of the
// undeleted views of the deleted parent ParentID whose ids are greater than
// After, in id order, soft-deleted at At by By.
type RowBatch struct {
	ParentID, After, By uuid.UUID
	Limit               int
	At                  time.Time
}

// ViewDeletions soft-deletes, in the transaction ctx carries, a batch of the
// views of a deleted project, or of a deleted workspace's own, those without
// a project, each locked FOR NO KEY UPDATE in id order before it writes
// them, and answers their ids (spec 2.7). A view deleted before keeps its
// deletion.
type ViewDeletions interface {
	DeleteProjectViews(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteWorkspaceViews(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
}
