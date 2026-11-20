package domain

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Kind is which of the two a view is (P3b spec 2.3): a project's, or,
// without a project, a workspace's. It chooses the view's actions, its
// words and which admin deletes it.
type Kind int

// The two kinds of views.
const (
	ProjectView Kind = iota + 1
	WorkspaceView
)

// View is a saved view as the module stores and answers it (spec 2.3,
// 2.4): its workspace and, for a project's view, its project; the content
// its owner writes; its owner, the account that created it; and when it
// was created and last changed.
type View struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	// ProjectID is the view's project, nil for a workspace's view.
	ProjectID *uuid.UUID
	Content
	OwnedByID            uuid.UUID
	CreatedAt, UpdatedAt time.Time
}

// Kind is the view's kind: a project's when it has a project.
func (v View) Kind() Kind {
	if v.ProjectID != nil {
		return ProjectView
	}
	return WorkspaceView
}

// OwnedBy tells whether actor owns the view.
func (v View) OwnedBy(actor uuid.UUID) bool { return v.OwnedByID == actor }

// HiddenFrom tells whether the guest rule hides the view from actor, whose
// decision on it gave g (spec 2.2, DP-15): a grant that sees only its own
// (Grant.OwnOnly, the project's guest while the project does not show its
// guests everything, the workspace's admin among them) does not see the
// views of others, which every operation naming one answers as one that is
// not there. A workspace view's grant is never OwnOnly.
func (v View) HiddenFrom(actor uuid.UUID, g shared.Grant) bool {
	return g.OwnOnly && !v.OwnedBy(actor)
}

// UpdatableBy tells whether actor may change the view: only its owner
// (spec 2.2, D29), whoever administers it.
func (v View) UpdatableBy(actor uuid.UUID) bool { return v.OwnedBy(actor) }

// DeletableBy tells whether actor, whose decision on the view gave g, may
// delete it (spec 2.2): its owner; or, for a project's view, the project's
// admin (Grant.ProjectAdmin: the project's admin, or the workspace's admin
// who is the project's member or guest); for a workspace's view, the
// workspace's admin.
func (v View) DeletableBy(actor uuid.UUID, g shared.Grant) bool {
	if v.OwnedBy(actor) {
		return true
	}
	if v.Kind() == ProjectView {
		return g.ProjectAdmin
	}
	return g.WorkspaceRole == shared.RoleAdmin
}
