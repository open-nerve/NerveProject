package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// FavoriteLocks is the one way a write of a favorite takes its locks and
// its decision (P3c spec 2.7, 2.8), beside the views' Locks, whose
// workspaces' and projects' ports it shares. Before any lock and outside the
// write's transaction, its target read without a lock, which names the
// project and the view to lock: those of a target of the write's workspace;
// a target that is not there or is another workspace's never is, and takes
// no lock. Then, in the write's transaction, the workspace's row FOR SHARE
// and favorite.write's decision again; the target's project's row FOR SHARE
// (M3 design 3.6 convention 2); a view's row FOR SHARE; the parent's FOR
// SHARE, a folder row before any item row (spec 2.8); and its target, the
// target's project and the projects the caller reads, read again under
// them, which the write decides on. No write gate: a favorite is no write
// of a project's work items (spec 2.4). A cycle's and a module's writes,
// an archiving, a deletion, a change of a member or a role lock the
// project's or the workspace's row FOR NO KEY UPDATE, a view's change or
// deletion its row FOR UPDATE, and a folder's deletion its row FOR UPDATE,
// so they and a write of a favorite run one after the other, and what the
// write decides under its locks stays so until it commits.
type FavoriteLocks struct {
	workspaces Workspaces
	projects   Projects
	locker     FavoriteLocker
	targets    Targets
	auth       shared.Authorizer
	tx         shared.TxManager
}

// NewFavoriteLocks returns the locks over the workspace module's directory,
// the project's lock, the module's own locks of a view and of a folder,
// the reads of the targets, the Authorizer and the writes' transactions.
func NewFavoriteLocks(workspaces Workspaces, projects Projects, locker FavoriteLocker, targets Targets, auth shared.Authorizer,
	tx shared.TxManager) FavoriteLocks {
	return FavoriteLocks{workspaces: workspaces, projects: projects, locker: locker, targets: targets, auth: auth, tx: tx}
}

// favoriteWrite is a write of a favorite as FavoriteLocks takes it: its
// workspace, decided on before; notFound, the 404 of a workspace deleted or
// no longer seen; its target, nil for a folder; and the parent it names,
// nil for none.
type favoriteWrite struct {
	workspaceID uuid.UUID
	notFound    error
	target      *entity
	parentID    *uuid.UUID
}

// favoriteHeld is what a write of a favorite holds once its locks are
// taken: what the caller sees of its target as read under them, and
// whether its parent, if it names one, is one of his folders there.
type favoriteHeld struct {
	sight  sight
	parent bool
}

// workspace is the workspace slug names, decided on for a creation of a
// favorite before any lock (decideWorkspace).
func (l FavoriteLocks) workspace(ctx context.Context, actor shared.Actor, slug string) (uuid.UUID, error) {
	return decideWorkspace(ctx, l.workspaces, l.auth, actor, domain.ActionFavoriteWrite, slug)
}

// run reads w's target without a lock, then, in one transaction, takes w's
// locks and the decision on the workspace again, reads its target under
// them, and runs fn with what they hold: w's whole write, whose decision on
// its target and its parent is fn's, before it reads the moment. A
// workspace deleted while its lock waited, or no longer seen, is
// w.notFound; a role refused there, the Authorizer's shared.Forbidden; fn
// does not run.
func (l FavoriteLocks) run(ctx context.Context, actor shared.Actor, w favoriteWrite, fn func(ctx context.Context, h favoriteHeld) error) error {
	var read []entity
	var project, view *uuid.UUID
	if w.target != nil {
		read = []entity{*w.target}
		pre, err := l.targets.read(ctx, read)
		if err != nil {
			return err
		}
		if a, found := pre.find(*w.target); found && a.workspaceID == w.workspaceID {
			project = a.projectID
			if w.target.kind == domain.FavoriteView {
				view = &w.target.id
			}
		}
	}
	return l.tx.WithinTx(ctx, func(ctx context.Context) error {
		found, err := l.workspaces.ShareWorkspaceByID(ctx, w.workspaceID)
		switch {
		case err != nil:
			return err
		case !found:
			return w.notFound
		}
		if err := allow(ctx, l.auth, actor, domain.ActionFavoriteWrite, w.workspaceID, w.notFound); err != nil {
			return err
		}
		if project != nil {
			if _, _, err := l.projects.Share(ctx, *project); err != nil {
				return err
			}
		}
		if view != nil {
			if err := l.locker.ShareView(ctx, *view); err != nil {
				return err
			}
		}
		h := favoriteHeld{parent: true}
		if w.parentID != nil {
			if h.parent, err = l.locker.ShareFolder(ctx, actor.UserID, w.workspaceID, *w.parentID); err != nil {
				return err
			}
		}
		if h.sight, err = l.targets.sight(ctx, actor, w.workspaceID, read); err != nil {
			return err
		}
		return fn(ctx, h)
	})
}

// decideWorkspace is the undeleted workspace slug names, once auth lets
// actor do action there (spec 2.2, 2.7): a slug of none, and a workspace
// he is not an active member of, is workspace.not_found, the one value; a
// role the rule refuses is the Authorizer's shared.Forbidden.
func decideWorkspace(ctx context.Context, workspaces Workspaces, auth shared.Authorizer, actor shared.Actor, action shared.Action,
	slug string) (uuid.UUID, error) {
	id, found, err := workspaces.WorkspaceBySlug(ctx, slug)
	switch {
	case err != nil:
		return uuid.UUID{}, err
	case !found:
		return uuid.UUID{}, domain.ErrWorkspaceNotFound
	}
	if err := allow(ctx, auth, actor, action, id, domain.ErrWorkspaceNotFound); err != nil {
		return uuid.UUID{}, err
	}
	return id, nil
}

// allow asks auth whether actor may do action, a workspace-level one, in
// the workspace: one he does not see is notFound; a role the rule refuses,
// the Authorizer's shared.Forbidden.
func allow(ctx context.Context, auth shared.Authorizer, actor shared.Actor, action shared.Action, workspaceID uuid.UUID, notFound error) error {
	_, err := auth.Authorize(ctx, actor, action, shared.Target{WorkspaceID: workspaceID})
	if errors.Is(err, shared.ErrNotVisible) {
		return notFound
	}
	return err
}
