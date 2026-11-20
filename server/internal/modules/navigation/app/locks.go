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
// a refused caller takes no lock. Then, in the write's transaction, the
// workspace's row FOR SHARE and, for a project's view, the project's row
// FOR SHARE after it (M3 design 3.6 convention 2); what the write names read
// again under them; and the decision again, which is final. Then a project
// whose views are turned off is navigation.views_disabled (409); its views
// are read as ever. No write gate: a view is no write of the project's work
// items (spec 2.4). A change of the toggle, of a member or a role, an
// archiving and a deletion lock the project's or the workspace's row FOR NO
// KEY UPDATE, so they and a write of a view run one after the other, and
// what the write decides under its locks stays so until it commits.
type Locks struct {
	containers
	auth shared.Authorizer
	tx   shared.TxManager
}

// NewLocks returns the locks over the workspace module's directory, the
// project's facts and lock, the Authorizer and the writes' transactions.
func NewLocks(workspaces Workspaces, projects Projects, auth shared.Authorizer, tx shared.TxManager) Locks {
	return Locks{containers: containers{workspaces: workspaces, projects: projects}, auth: auth, tx: tx}
}

// write is a write of a view as Locks takes it: what it names, and what it
// does, whose action the kind of the views chooses (domain.ActionOf).
type write struct {
	named
	verb domain.Verb
}

// run takes w's check, then in one transaction its locks and decision, and
// runs fn in it with what they hold: w's whole write. A refusal of the
// check, of the locks or of the decision, and a project's views turned off,
// is its answer, and fn does not run.
func (l Locks) run(ctx context.Context, actor shared.Actor, w write, fn func(ctx context.Context, h held) error) error {
	pre, err := l.find(ctx, w.named)
	if err != nil {
		return err
	}
	if _, err := decide(ctx, l.auth, actor, w.verb, pre, w.notFound()); err != nil {
		return err
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
