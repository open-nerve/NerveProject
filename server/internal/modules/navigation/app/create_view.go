package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// create creates the view n of the kind of the views of what named names
// (spec 2.4, 2.5, DP-22): its body checked first, before any read
// (Words.checkNewView); then its locks and decision (Locks.run); then, at
// the moment read after its last lock, the view of the workspace and the
// project as the locks read them, owned, created and last changed by the
// caller.
func (d WriteDeps) create(ctx context.Context, where named, n domain.NewView) (domain.View, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.View{}, err
	}
	content, err := d.Words.checkNewView(where.kind(), n)
	if err != nil {
		return domain.View{}, err
	}
	var out domain.View
	err = d.Locks.run(ctx, actor, write{named: where, verb: domain.Create}, func(ctx context.Context, h held) error {
		now := d.Clock.Now()
		v := domain.View{ID: uuid.NewV7(), WorkspaceID: h.workspaceID, ProjectID: h.projectID, Content: content, OwnedByID: actor.UserID,
			CreatedAt: now, UpdatedAt: now}
		if err := d.Views.InsertView(ctx, v, actor.UserID); err != nil {
			return err
		}
		out = v
		return nil
	})
	if err != nil {
		return domain.View{}, err
	}
	return out, nil
}

// CreateProjectView creates a project's view (createProjectView).
type CreateProjectView struct{ d WriteDeps }

// NewCreateProjectView returns the use case over d.
func NewCreateProjectView(d WriteDeps) *CreateProjectView { return &CreateProjectView{d: d} }

// Execute creates the view n of the project projectID
// (WriteDeps.create), refused 409 navigation.views_disabled while the
// project's views are turned off.
func (u *CreateProjectView) Execute(ctx context.Context, projectID uuid.UUID, n domain.NewView) (domain.View, error) {
	return u.d.create(ctx, named{project: &projectID}, n)
}

// CreateWorkspaceView creates a workspace's own view, one without a project
// (createWorkspaceView).
type CreateWorkspaceView struct{ d WriteDeps }

// NewCreateWorkspaceView returns the use case over d.
func NewCreateWorkspaceView(d WriteDeps) *CreateWorkspaceView { return &CreateWorkspaceView{d: d} }

// Execute creates the view n of the workspace slug names
// (WriteDeps.create).
func (u *CreateWorkspaceView) Execute(ctx context.Context, slug string, n domain.NewView) (domain.View, error) {
	return u.d.create(ctx, named{slug: slug}, n)
}
