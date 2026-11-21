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
// workspaces' and projects' ports it shares. Before any lock and outside
// the write's transaction, its check: a creation's workspace (creation), or
// a change's or a deletion's favorite (own), decided on, and its target
// read without a lock, which names the project and the view to lock: those
// of a target of the write's workspace; a target that is not there or is
// another workspace's never is, and takes no lock. Then, in the write's
// transaction (run), the workspace's row FOR SHARE and favorite.write's
// decision again; the target's project's row FOR SHARE (M3 design 3.6
// convention 2); a view's row FOR SHARE; then the favorites' rows in their
// one total order, a folder's before an item's (spec 2.8, re-91 I2): the
// parent a write names FOR SHARE, then the favorite it changes or deletes
// FOR UPDATE, and, a folder's deletion's, the favorites the folder holds
// FOR UPDATE in id order, or, a creation's of a target, the caller's
// undeleted favorite of it FOR SHARE, each in its own statement; and its
// favorite, its target, the target's project and the projects the caller
// reads, read again under them, which the write decides on. No write
// gate: a favorite is no write of a project's work items (spec 2.4). A
// cycle's and a module's writes, an archiving, a deletion, a change of a
// member or a role lock the project's or the workspace's row FOR NO KEY
// UPDATE, a view's change or deletion its row FOR UPDATE, and a folder's
// deletion its row FOR UPDATE, so they and a write of a favorite run one
// after the other, and what the write decides under its locks stays so
// until it commits.
type FavoriteLocks struct {
	workspaces Workspaces
	projects   Projects
	favorites  FavoriteReader
	locker     FavoriteLocker
	targets    Targets
	auth       shared.Authorizer
	tx         shared.TxManager
}

// NewFavoriteLocks returns the locks over the workspace module's directory,
// the project's lock, the module's own reads of a favorite and locks of a
// view and of the favorites, the reads of the targets, the Authorizer and
// the writes' transactions.
func NewFavoriteLocks(workspaces Workspaces, projects Projects, favorites FavoriteReader, locker FavoriteLocker, targets Targets,
	auth shared.Authorizer, tx shared.TxManager) FavoriteLocks {
	return FavoriteLocks{workspaces: workspaces, projects: projects, favorites: favorites, locker: locker, targets: targets, auth: auth, tx: tx}
}

// favoriteWrite is a write of a favorite as FavoriteLocks takes it, once
// checked: its workspace; notFound, the 404 of a workspace deleted or no
// longer seen; its target, nil for a folder, and the project and the view
// of it to lock (aim); the parent it names, nil for none; and, a change's
// or a deletion's, the favorite it writes, nil for a creation, and whether
// that is a folder's deletion, which takes the favorites the folder holds.
type favoriteWrite struct {
	workspaceID   uuid.UUID
	notFound      error
	target        *entity
	project, view *uuid.UUID
	parentID      *uuid.UUID
	row           *uuid.UUID
	children      bool
}

// favoriteHeld is what a write of a favorite holds once its locks are
// taken: what the caller sees of its target as read under them; whether
// its parent, if it names one, is one of his folders there; a change's or
// a deletion's favorite as read under its lock; and, a creation's of a
// target, his undeleted favorite of it as read under its lock, nil when he
// has none.
type favoriteHeld struct {
	sight    sight
	parent   bool
	row      domain.Favorite
	existing *domain.Favorite
}

// creation is the creation of the caller's favorite n in the workspace slug
// names, checked before any lock (spec 2.7): the workspace and
// favorite.write's decision on it (decideWorkspace); then n's target, if
// any, read without a lock, which names its locks (aim). It decides
// nothing on the target: its decision is under the locks.
func (l FavoriteLocks) creation(ctx context.Context, actor shared.Actor, slug string, n domain.NewFavorite) (favoriteWrite, error) {
	workspaceID, err := decideWorkspace(ctx, l.workspaces, l.auth, actor, domain.ActionFavoriteWrite, slug)
	if err != nil {
		return favoriteWrite{}, err
	}
	w := favoriteWrite{workspaceID: workspaceID, notFound: domain.ErrWorkspaceNotFound, parentID: n.ParentID}
	if n.Kind == domain.FavoriteFolder {
		return w, nil
	}
	w.target = &entity{kind: n.Kind, id: *n.EntityID}
	pre, err := l.targets.read(ctx, []entity{*w.target})
	if err != nil {
		return favoriteWrite{}, err
	}
	w.aim(pre)
	return w, nil
}

// own is the change or the deletion of the caller's favorite id, checked
// before any lock (spec 2.7): his undeleted favorite read without a lock;
// favorite.write's decision on its workspace; and what he sees of its
// target (Targets.sight, sight.shows), which names its locks (aim). One
// not there, deleted or another's, one of a workspace he does not see and
// one whose target he does not see are each navigation.favorite_not_found,
// the one value; a role refused there is the Authorizer's
// shared.Forbidden. It answers the favorite as it read it.
func (l FavoriteLocks) own(ctx context.Context, actor shared.Actor, id uuid.UUID) (favoriteWrite, domain.Favorite, error) {
	f, found, err := l.favorites.Favorite(ctx, actor.UserID, id)
	switch {
	case err != nil:
		return favoriteWrite{}, domain.Favorite{}, err
	case !found:
		return favoriteWrite{}, domain.Favorite{}, domain.ErrFavoriteNotFound
	}
	if err := allow(ctx, l.auth, actor, domain.ActionFavoriteWrite, f.WorkspaceID, domain.ErrFavoriteNotFound); err != nil {
		return favoriteWrite{}, domain.Favorite{}, err
	}
	w := favoriteWrite{workspaceID: f.WorkspaceID, notFound: domain.ErrFavoriteNotFound, row: &f.ID}
	t, ofTarget := entityOf(f)
	if !ofTarget {
		return w, f, nil
	}
	w.target = &t
	pre, err := l.targets.sight(ctx, actor, f.WorkspaceID, []entity{t})
	if err != nil {
		return favoriteWrite{}, domain.Favorite{}, err
	}
	if _, shown := pre.shows(f); !shown {
		return favoriteWrite{}, domain.Favorite{}, domain.ErrFavoriteNotFound
	}
	w.aim(pre)
	return w, f, nil
}

// aim names the locks of w's target as pre, read without a lock, found it:
// its project and, a view's, the view, when it is of w's workspace; none
// for a target not there, deleted or another workspace's, all of which
// stay so.
func (w *favoriteWrite) aim(pre sight) {
	if a, found := pre.find(*w.target); found && a.workspaceID == w.workspaceID {
		w.project = a.projectID
		if w.target.kind == domain.FavoriteView {
			w.view = &w.target.id
		}
	}
}

// run takes, in one transaction, w's locks and the decision on its
// workspace again, reads its favorite, a creation's of a target the
// caller's undeleted favorite of it, and its target under them, and runs
// fn with what they hold: w's whole write, whose decision on its target and
// its parent, a creation's, or on its parent and its change, a change's,
// is fn's, before it reads the moment. A workspace deleted while its lock
// waited, or no longer seen, is w.notFound, as are a change's or a
// deletion's favorite deleted meanwhile and one whose target he no longer
// sees under them; a role refused there, the Authorizer's
// shared.Forbidden; fn does not run.
func (l FavoriteLocks) run(ctx context.Context, actor shared.Actor, w favoriteWrite, fn func(ctx context.Context, h favoriteHeld) error) error {
	var read []entity
	if w.target != nil {
		read = []entity{*w.target}
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
		if w.project != nil {
			if _, _, err := l.projects.Share(ctx, *w.project); err != nil {
				return err
			}
		}
		if w.view != nil {
			if err := l.locker.ShareView(ctx, *w.view); err != nil {
				return err
			}
		}
		h := favoriteHeld{parent: true}
		if w.parentID != nil {
			if h.parent, err = l.locker.ShareFolder(ctx, actor.UserID, w.workspaceID, *w.parentID); err != nil {
				return err
			}
		}
		if w.row != nil {
			if h.row, err = l.lockRow(ctx, actor, w); err != nil {
				return err
			}
		} else if w.target != nil {
			if h.existing, err = l.shareExisting(ctx, actor, *w.target); err != nil {
				return err
			}
		}
		if h.sight, err = l.targets.sight(ctx, actor, w.workspaceID, read); err != nil {
			return err
		}
		if w.row != nil {
			if _, shown := h.sight.shows(h.row); !shown {
				return w.notFound
			}
		}
		return fn(ctx, h)
	})
}

// lockRow locks FOR UPDATE, after its parent's FOR SHARE, the favorite a
// change or a deletion writes, and reads it again; then, a folder's
// deletion's, the favorites the folder holds FOR UPDATE in id order. One
// deleted meanwhile is w.notFound.
func (l FavoriteLocks) lockRow(ctx context.Context, actor shared.Actor, w favoriteWrite) (domain.Favorite, error) {
	f, found, err := l.locker.LockFavorite(ctx, actor.UserID, *w.row)
	switch {
	case err != nil:
		return domain.Favorite{}, err
	case !found:
		return domain.Favorite{}, w.notFound
	}
	if w.children {
		if err := l.locker.LockFolderChildren(ctx, f.ID); err != nil {
			return domain.Favorite{}, err
		}
	}
	return f, nil
}

// shareExisting locks FOR SHARE, after its parent's FOR SHARE, the caller's
// undeleted favorite of the target a creation names, and reads it: nil
// when he has none, as when the one he had was deleted while the lock
// waited (spec 2.5, 2.8). Its deletion waits until the creation ends, so
// the creation that finds it answers it as it is. A creation refused
// afterwards (422) takes it too: it then only reads and rolls back, so it
// waits on nothing while it holds the row.
func (l FavoriteLocks) shareExisting(ctx context.Context, actor shared.Actor, target entity) (*domain.Favorite, error) {
	f, found, err := l.locker.ShareFavoriteOfTarget(ctx, actor.UserID, target.kind, target.id)
	if err != nil || !found {
		return nil, err
	}
	return &f, nil
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
