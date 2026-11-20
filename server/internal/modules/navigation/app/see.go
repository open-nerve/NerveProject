package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// named is what a call on the views names (P3b spec 2.4): the views of
// the project project, or, when it is nil, of the workspace slug names;
// or, when view is not nil, the view view, /views/{view_id}, whose kind
// its row tells.
type named struct {
	project *uuid.UUID
	slug    string
	view    *uuid.UUID
}

// kind is the kind of the views of the project or the workspace n names.
func (n named) kind() domain.Kind {
	if n.project != nil {
		return domain.ProjectView
	}
	return domain.WorkspaceView
}

// notFound is the 404 of what n names (spec 2.2, 2.5): project.not_found,
// workspace.not_found, or for a view navigation.view_not_found, whatever
// it is that the caller cannot have.
func (n named) notFound() error {
	switch {
	case n.view != nil:
		return domain.ErrViewNotFound
	case n.project != nil:
		return domain.ErrProjectNotFound
	}
	return domain.ErrWorkspaceNotFound
}

// held is what a call on the views holds once decided: the workspace; for
// a project's views the project and its facts, projectID nil for a
// workspace's; for a call on a view, the view; and the caller's grant.
type held struct {
	workspaceID uuid.UUID
	projectID   *uuid.UUID
	project     ProjectFacts
	view        *domain.View
	grant       shared.Grant
}

// kind is the kind of the views of h's project, or of its workspace.
func (h held) kind() domain.Kind {
	if h.projectID != nil {
		return domain.ProjectView
	}
	return domain.WorkspaceView
}

// ownedBy is whose views a list of h's views lists (spec 2.2, DP-15): only
// actor's, for a grant that sees only his own (Grant.OwnOnly), which the
// store filters on; nil, every one, else.
func (h held) ownedBy(actor shared.Actor) *uuid.UUID {
	if !h.grant.OwnOnly {
		return nil
	}
	id := actor.UserID
	return &id
}

// containers find what a call on the views names: a project, a workspace,
// or a view, and through it its project or its workspace.
type containers struct {
	workspaces Workspaces
	projects   Projects
	views      ViewReader
}

// find reads what n names without a lock, in the transaction ctx carries,
// if any: the project's facts, the workspace slug names, or the view and
// with it its workspace and project. One that is not there is n's 404.
// A view's project's facts are left for its write's locks, which alone
// read its toggle.
func (c containers) find(ctx context.Context, n named) (held, error) {
	switch {
	case n.view != nil:
		v, found, err := c.views.View(ctx, *n.view)
		switch {
		case err != nil:
			return held{}, err
		case !found:
			return held{}, n.notFound()
		}
		return held{workspaceID: v.WorkspaceID, projectID: v.ProjectID, view: &v}, nil
	case n.project != nil:
		f, found, err := c.projects.Facts(ctx, *n.project)
		switch {
		case err != nil:
			return held{}, err
		case !found:
			return held{}, n.notFound()
		}
		id := *n.project
		return held{workspaceID: f.WorkspaceID, projectID: &id, project: f}, nil
	}
	id, found, err := c.workspaces.WorkspaceBySlug(ctx, n.slug)
	switch {
	case err != nil:
		return held{}, err
	case !found:
		return held{}, n.notFound()
	}
	return held{workspaceID: id}, nil
}

// target is what Authorize decides an action on the views of the workspace
// and, when projectID is not nil, of its project, on.
func target(workspaceID uuid.UUID, projectID *uuid.UUID) shared.Target {
	t := shared.Target{WorkspaceID: workspaceID}
	if projectID != nil {
		t.ProjectID = *projectID
	}
	return t
}

// decide asks auth for verb's action on h's views for actor, the action of
// their kind (domain.ActionOf), on h's project or workspace, and answers h
// with the grant: a project or a workspace actor does not see is notFound;
// a role the rule refuses is the Authorizer's shared.Forbidden. On the view
// h holds, a /views/{view_id} call's, one the guest rule hides from actor
// (View.HiddenFrom) is notFound too, the one value, also when the rule
// refuses his role, before any 403 (spec 2.2, 2.5); then a change of it
// by anyone but its owner, or a deletion by anyone but its owner or its
// admin, is shared.Forbidden (permits).
func decide(ctx context.Context, auth shared.Authorizer, actor shared.Actor, verb domain.Verb, h held, notFound error) (held, error) {
	grant, err := auth.Authorize(ctx, actor, domain.ActionOf(h.kind(), verb), target(h.workspaceID, h.projectID))
	switch {
	case errors.Is(err, shared.ErrNotVisible), h.view != nil && h.view.HiddenFrom(actor.UserID, grant):
		return held{}, notFound
	case err != nil:
		return held{}, err
	case h.view != nil && !permits(*h.view, verb, actor.UserID, grant):
		return held{}, shared.Forbidden()
	}
	h.grant = grant
	return h, nil
}

// permits tells whether actor, whose decision on v gave g, may do verb
// with v by its owner rules (spec 2.2): change it only as its owner
// (View.UpdatableBy), delete it as its owner or its admin
// (View.DeletableBy); read it as the decision lets him.
func permits(v domain.View, verb domain.Verb, actor uuid.UUID, g shared.Grant) bool {
	switch verb {
	case domain.Update:
		return v.UpdatableBy(actor)
	case domain.Delete:
		return v.DeletableBy(actor, g)
	}
	return true
}
