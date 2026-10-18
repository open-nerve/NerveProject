package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Locks is the one way a write on the work-item side takes its lock and
// its decision (M4 design 3.5, decision point 1: B). Before any lock and
// outside the write's transaction, the precheck: the project, or the work
// item the write names and through it its project, read without a lock,
// and the decision on it, which answers every refusal there, so a caller
// who would be refused takes no lock at all. Then, in the write's
// transaction, the project's row FOR NO KEY UPDATE, its workspace's row
// not locked: the write's first lock, which every write on the project's
// work items, states, labels and memberships also takes, so they run one
// after another; the work item read again under it; and the decision again
// under it, which is final: a removal or a demotion that committed while
// the write waited is seen. A read of a work item takes the same decision
// on its project, without a lock (see). issue.New builds one Locks for
// every use case: none holds an Authorizer or a transaction of its own.
type Locks struct {
	projects ProjectGate
	issues   IssueReader
	auth     shared.Authorizer
	tx       shared.TxManager
}

// NewLocks returns the locks over project's gate, the work items' reads,
// the Authorizer and the transactions.
func NewLocks(projects ProjectGate, issues IssueReader, auth shared.Authorizer, tx shared.TxManager) Locks {
	return Locks{projects: projects, issues: issues, auth: auth, tx: tx}
}

// write is a write on the work-item side, as Locks takes its lock: the
// project it is on, or the work item it names, issue, whose project it is
// then on; the action it decides; and the 404 of what the caller named.
type write struct {
	project  uuid.UUID
	issue    *uuid.UUID
	action   shared.Action
	notFound error
}

// held is what a write holds under its lock: its project, locked, and the
// work item it names, read under the lock; zero for a write on a project.
type held struct {
	project LockedProject
	issue   domain.Issue
}

// run takes w's precheck, before any lock and outside any transaction,
// then in one transaction w's lock and decision, and runs fn in it with
// what they hold (M4 design 3.5): w's whole write. A refusal of either is
// its answer, and fn does not run.
func (l Locks) run(ctx context.Context, actor shared.Actor, w write, fn func(ctx context.Context, h held) error) error {
	w, err := l.precheck(ctx, actor, w)
	if err != nil {
		return err
	}
	return l.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := l.lockAndDecide(ctx, actor, w)
		if err != nil {
			return err
		}
		return fn(ctx, h)
	})
}

// precheck decides w's action for actor before any lock and outside any
// transaction (M4 design 3.5), and answers w on its project: for a write
// that names a work item, the work item read without a lock and the
// decision on its project (see); for one on a project, the project read
// without a lock, one that is not there w.notFound, and the decision on
// it. A project not visible to actor is w.notFound; a role the rule does
// not allow is the Authorizer's shared.Forbidden. What it lets through,
// lockAndDecide decides again.
func (l Locks) precheck(ctx context.Context, actor shared.Actor, w write) (write, error) {
	if w.issue != nil {
		i, err := l.see(ctx, actor, *w.issue, w.action)
		if err != nil {
			return write{}, err
		}
		w.project = i.ProjectID
		return w, nil
	}
	workspaceID, found, err := l.projects.Find(ctx, w.project)
	switch {
	case err != nil:
		return write{}, err
	case !found:
		return write{}, w.notFound
	}
	return w, l.decide(ctx, actor, w.action, shared.Target{WorkspaceID: workspaceID, ProjectID: w.project}, w.notFound)
}

// lockAndDecide takes w's lock and decides w's action under it, in the
// transaction ctx carries: the project's row FOR NO KEY UPDATE; the work
// item w names read again; then the decision on the locked project. It
// answers what they hold. A project deleted while the lock waited is
// w.notFound, and so is a work item deleted meanwhile, or no longer in the
// project, and a project no longer visible to actor; a role the rule no
// longer allows is the Authorizer's shared.Forbidden.
func (l Locks) lockAndDecide(ctx context.Context, actor shared.Actor, w write) (held, error) {
	p, found, err := l.projects.Lock(ctx, w.project)
	switch {
	case err != nil:
		return held{}, err
	case !found:
		return held{}, w.notFound
	}
	h := held{project: p}
	if w.issue != nil {
		i, found, err := l.issues.GetIssue(ctx, *w.issue)
		switch {
		case err != nil:
			return held{}, err
		case !found || i.ProjectID != w.project:
			return held{}, w.notFound
		}
		h.issue = i
	}
	if err := l.decide(ctx, actor, w.action, shared.Target{WorkspaceID: p.WorkspaceID, ProjectID: w.project}, w.notFound); err != nil {
		return held{}, err
	}
	return h, nil
}

// see reads the undeleted work item id, archived or not, without a lock,
// and decides action on its project for actor (M4 design 3.4, 3.5): a work
// item that is not there, or whose project actor does not see, is
// issue.not_found; a role the rule does not allow is the Authorizer's
// shared.Forbidden. It answers the work item as read.
func (l Locks) see(ctx context.Context, actor shared.Actor, id uuid.UUID, action shared.Action) (domain.Issue, error) {
	i, found, err := l.issues.GetIssue(ctx, id)
	switch {
	case err != nil:
		return domain.Issue{}, err
	case !found:
		return domain.Issue{}, domain.ErrIssueNotFound
	}
	if err := l.decide(ctx, actor, action, shared.Target{WorkspaceID: i.WorkspaceID, ProjectID: i.ProjectID}, domain.ErrIssueNotFound); err != nil {
		return domain.Issue{}, err
	}
	return i, nil
}

// decide asks the Authorizer for action on the target for actor. A
// project not visible to actor is notFound.
func (l Locks) decide(ctx context.Context, actor shared.Actor, action shared.Action, target shared.Target, notFound error) error {
	_, err := l.auth.Authorize(ctx, actor, action, target)
	if errors.Is(err, shared.ErrNotVisible) {
		return notFound
	}
	return err
}
