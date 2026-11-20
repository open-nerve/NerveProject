package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// named is what a call on the views of a project or of a workspace names
// (P3b spec 2.4): the project project, or, when it is nil, the workspace
// slug names.
type named struct {
	project *uuid.UUID
	slug    string
}

// kind is the kind of the views n's call is on.
func (n named) kind() domain.Kind {
	if n.project != nil {
		return domain.ProjectView
	}
	return domain.WorkspaceView
}

// notFound is the 404 of what n names (spec 2.2): project.not_found, or
// workspace.not_found.
func (n named) notFound() error {
	if n.project != nil {
		return domain.ErrProjectNotFound
	}
	return domain.ErrWorkspaceNotFound
}

// held is what a call on the views of a project or of a workspace holds once
// decided: the workspace; for a project's views the project and its facts,
// projectID nil for a workspace's; and the caller's grant.
type held struct {
	workspaceID uuid.UUID
	projectID   *uuid.UUID
	project     ProjectFacts
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

// containers find and lock the project or the workspace a call on the views
// names.
type containers struct {
	workspaces Workspaces
	projects   Projects
}

// find reads what n names without a lock, in the transaction ctx carries,
// if any: the project's facts, or the workspace slug names. One that is not
// there is n's 404.
func (c containers) find(ctx context.Context, n named) (held, error) {
	if n.project != nil {
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

// share takes, in the transaction ctx carries, the locks of a write on what
// n names, whose check found pre, and reads it again under them (spec 2.6,
// M3 design 3.6 convention 2): for a project's views, the workspace's row
// FOR SHARE, then the project's row FOR SHARE and its facts; for a
// workspace's, the workspace's row FOR SHARE as its slug names it now. A
// project or a workspace deleted while its lock waited is n's 404.
func (c containers) share(ctx context.Context, n named, pre held) (held, error) {
	if n.project == nil {
		id, found, err := c.workspaces.ShareWorkspaceBySlug(ctx, n.slug)
		switch {
		case err != nil:
			return held{}, err
		case !found:
			return held{}, n.notFound()
		}
		return held{workspaceID: id}, nil
	}
	found, err := c.workspaces.ShareWorkspaceByID(ctx, pre.workspaceID)
	switch {
	case err != nil:
		return held{}, err
	case !found:
		return held{}, n.notFound()
	}
	f, found, err := c.projects.Share(ctx, *pre.projectID)
	switch {
	case err != nil:
		return held{}, err
	case !found:
		return held{}, n.notFound()
	}
	return held{workspaceID: pre.workspaceID, projectID: pre.projectID, project: f}, nil
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
// a role the rule refuses is the Authorizer's shared.Forbidden.
func decide(ctx context.Context, auth shared.Authorizer, actor shared.Actor, verb domain.Verb, h held, notFound error) (held, error) {
	grant, err := auth.Authorize(ctx, actor, domain.ActionOf(h.kind(), verb), target(h.workspaceID, h.projectID))
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return held{}, notFound
	case err != nil:
		return held{}, err
	}
	h.grant = grant
	return h, nil
}

// seeView reads the undeleted view id without a lock, in the transaction
// ctx carries, then decides verb's action on it for actor, the action its
// kind chooses (spec 2.5, domain.ActionOf): a project's view's on its
// project, a workspace's on its workspace. A view that is not there, one
// whose project or workspace actor does not see, and one the guest rule
// hides from him (View.HiddenFrom), also when the rule refuses his role,
// are each navigation.view_not_found, the one value; a role refused on a
// view he sees is the Authorizer's shared.Forbidden. It answers the view as
// read, and the grant.
func seeView(ctx context.Context, views ViewReader, auth shared.Authorizer, actor shared.Actor, id uuid.UUID, verb domain.Verb) (domain.View, shared.Grant, error) {
	v, found, err := views.View(ctx, id)
	switch {
	case err != nil:
		return domain.View{}, shared.Grant{}, err
	case !found:
		return domain.View{}, shared.Grant{}, domain.ErrViewNotFound
	}
	grant, err := auth.Authorize(ctx, actor, domain.ActionOf(v.Kind(), verb), target(v.WorkspaceID, v.ProjectID))
	switch {
	case errors.Is(err, shared.ErrNotVisible), v.HiddenFrom(actor.UserID, grant):
		return domain.View{}, shared.Grant{}, domain.ErrViewNotFound
	case err != nil:
		return domain.View{}, shared.Grant{}, err
	}
	return v, grant, nil
}
