package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Locks is the one way a write of a view takes its locks and its decision
// (P3b spec 2.5, 2.6). Before any lock and outside the write's
// transaction, the check: what the write names, read without a lock, and
// the decision on it, which answers every 404 and 403 the locks would, so
// a refused caller takes no lock; then the write's own check of its body
// against it, if any. Then, in the write's transaction, the workspace's
// row FOR SHARE and, for a project's view, the project's row FOR SHARE
// after it (M3 design 3.6 convention 2), and for a change or a deletion
// the view's row FOR UPDATE last; what the write names read again under
// them; and the decision again, which is final. Then a project whose
// views are turned off is navigation.views_disabled (409); its views are
// read as ever. No write gate: a view is no write of the project's work
// items (spec 2.4). A change of the toggle, of a member or a role, an
// archiving and a deletion lock the project's or the workspace's row FOR
// NO KEY UPDATE, so they and a write of a view run one after the other,
// and what the write decides under its locks stays so until it commits.
type Locks struct {
	containers
	locker ViewLocker
	auth   shared.Authorizer
	tx     shared.TxManager
}

// NewLocks returns the locks over the workspace module's directory, the
// project's facts and lock, the views' reads and lock, the Authorizer and
// the writes' transactions.
func NewLocks(workspaces Workspaces, projects Projects, views ViewReader, locker ViewLocker, auth shared.Authorizer, tx shared.TxManager) Locks {
	return Locks{containers: containers{workspaces: workspaces, projects: projects, views: views}, locker: locker, auth: auth, tx: tx}
}

// write is a write of a view as Locks takes it: what it names; what it
// does, whose action the kind of the views chooses (domain.ActionOf); and
// check, when not nil, the check of its body against what its check holds
// once decided, before any lock: an update's, whose words are its view's
// kind's (spec 2.5).
type write struct {
	named
	verb  domain.Verb
	check func(h held) error
}

// run takes w's check, then in one transaction its locks and decision, and
// runs fn in it with what they hold: w's whole write. A refusal of the
// check, of the locks or of the decision, and a project's views turned
// off, is its answer, and fn does not run.
func (l Locks) run(ctx context.Context, actor shared.Actor, w write, fn func(ctx context.Context, h held) error) error {
	pre, err := l.find(ctx, w.named)
	if err != nil {
		return err
	}
	if pre, err = decide(ctx, l.auth, actor, w.verb, pre, w.notFound()); err != nil {
		return err
	}
	if w.check != nil {
		if err := w.check(pre); err != nil {
			return err
		}
	}
	return l.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := l.share(ctx, w.named, pre)
		if err != nil {
			return err
		}
		if h, err = decide(ctx, l.auth, actor, w.verb, h, w.notFound()); err != nil {
			return err
		}
		if h.projectID != nil && !h.project.IssueViewsView {
			return domain.ErrViewsDisabled
		}
		return fn(ctx, h)
	})
}

// share takes, in the transaction ctx carries, the locks of a write on what
// n names, whose check found pre, and reads it again under them (spec 2.6,
// M3 design 3.6 convention 2): for a project's views, or a project's view,
// the workspace's row FOR SHARE, then the project's row FOR SHARE and its
// facts; for a workspace's, the workspace's row FOR SHARE as its slug names
// it now, or for a workspace's view as the view names it; and for a view,
// last, its row FOR UPDATE, read again. A view never moves, so its check's
// read names its workspace and its project. A project, a workspace or a
// view deleted while its lock waited is n's 404.
func (l Locks) share(ctx context.Context, n named, pre held) (held, error) {
	if n.project == nil && n.view == nil {
		id, found, err := l.workspaces.ShareWorkspaceBySlug(ctx, n.slug)
		switch {
		case err != nil:
			return held{}, err
		case !found:
			return held{}, n.notFound()
		}
		return held{workspaceID: id}, nil
	}
	found, err := l.workspaces.ShareWorkspaceByID(ctx, pre.workspaceID)
	switch {
	case err != nil:
		return held{}, err
	case !found:
		return held{}, n.notFound()
	}
	h := held{workspaceID: pre.workspaceID, projectID: pre.projectID}
	if h.projectID != nil {
		f, found, err := l.projects.Share(ctx, *h.projectID)
		switch {
		case err != nil:
			return held{}, err
		case !found:
			return held{}, n.notFound()
		}
		h.project = f
	}
	if n.view == nil {
		return h, nil
	}
	v, found, err := l.locker.LockView(ctx, *n.view)
	switch {
	case err != nil:
		return held{}, err
	case !found:
		return held{}, n.notFound()
	}
	h.view = &v
	return h, nil
}
