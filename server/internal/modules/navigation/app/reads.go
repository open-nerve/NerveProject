package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Reads is the one way a read of the views takes its decision (P3b spec
// 2.2, 2.5): in one read-only snapshot transaction of its own, what it
// names read without a lock and the decision on it. No lock: a read waits
// for no write. A project's views turned off are read as ever (M7 design
// 3.7).
type Reads struct {
	containers
	auth shared.Authorizer
	tx   shared.TxManager
}

// NewReads returns the reads over the workspace module's directory, the
// project's facts, the views, the Authorizer and read-only snapshot
// transactions.
func NewReads(workspaces Workspaces, projects Projects, views ViewReader, auth shared.Authorizer, tx shared.TxManager) Reads {
	return Reads{containers: containers{workspaces: workspaces, projects: projects, views: views}, auth: auth, tx: tx}
}

// list lists the views of what n names (listProjectViews,
// listWorkspaceViews): the decision of its kind's read on the project, or
// the workspace, then its views, the newest first, only the caller's own
// for a grant that sees only his own (held.ownedBy). What n names that is
// not there, or that the caller does not see, is its 404; a role refused is
// the Authorizer's shared.Forbidden.
func (r Reads) list(ctx context.Context, n named) ([]domain.View, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	var out []domain.View
	err = r.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := r.find(ctx, n)
		if err != nil {
			return err
		}
		if h, err = decide(ctx, r.auth, actor, domain.Read, h, n.notFound()); err != nil {
			return err
		}
		if h.projectID != nil {
			out, err = r.views.ProjectViews(ctx, *h.projectID, h.ownedBy(actor))
		} else {
			out, err = r.views.WorkspaceViews(ctx, h.workspaceID, h.ownedBy(actor))
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ListProjectViews lists a project's views (listProjectViews).
type ListProjectViews struct{ r Reads }

// NewListProjectViews returns the use case over r.
func NewListProjectViews(r Reads) *ListProjectViews { return &ListProjectViews{r: r} }

// Execute lists the views of the project projectID (Reads.list).
func (u *ListProjectViews) Execute(ctx context.Context, projectID uuid.UUID) ([]domain.View, error) {
	return u.r.list(ctx, named{project: &projectID})
}

// ListWorkspaceViews lists a workspace's own views, those without a project
// (listWorkspaceViews).
type ListWorkspaceViews struct{ r Reads }

// NewListWorkspaceViews returns the use case over r.
func NewListWorkspaceViews(r Reads) *ListWorkspaceViews { return &ListWorkspaceViews{r: r} }

// Execute lists the own views of the workspace slug names (Reads.list).
func (u *ListWorkspaceViews) Execute(ctx context.Context, slug string) ([]domain.View, error) {
	return u.r.list(ctx, named{slug: slug})
}

// GetView reads a view of either kind by its id (getView).
type GetView struct{ r Reads }

// NewGetView returns the use case over r.
func NewGetView(r Reads) *GetView { return &GetView{r: r} }

// Execute reads the view id, in one read-only snapshot transaction, and
// decides its kind's read on it (spec 2.5): on its project, a project's
// view, by view.read, a workspace's on its workspace by workspace_view.read
// (containers.find, decide). Every view it cannot answer is
// navigation.view_not_found: one that is not there, one whose project or
// workspace the caller does not see, and another's while he sees only his
// own (View.HiddenFrom); a role refused on one he sees is the
// Authorizer's shared.Forbidden.
func (u *GetView) Execute(ctx context.Context, id uuid.UUID) (domain.View, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.View{}, err
	}
	n := named{view: &id}
	var out domain.View
	err = u.r.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.r.find(ctx, n)
		if err != nil {
			return err
		}
		if h, err = decide(ctx, u.r.auth, actor, domain.Read, h, n.notFound()); err != nil {
			return err
		}
		out = *h.view
		return nil
	})
	if err != nil {
		return domain.View{}, err
	}
	return out, nil
}
