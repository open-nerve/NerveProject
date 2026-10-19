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
// who would be refused takes no lock at all, nor the project's place in
// the write gate. Then the write gate, waited for holding no connection
// and no lock (11.11). Then, in the write's transaction, the project's row
// FOR NO KEY UPDATE, its workspace's row not locked: the write's first
// lock, which every write on the project's work items, states, labels and
// memberships also takes, so they run one after another; the work item
// read again under it; and the decision again under it, which is final: a
// removal or a demotion that committed while the write waited is seen. A
// read of a work item takes the same decision on its project, without a
// lock (see). Every decision on a work item applies the guest rule first
// (decide). issue.New builds one Locks for every use case: none holds an
// Authorizer, a write gate or a transaction of its own.
type Locks struct {
	projects ProjectGate
	issues   IssueReader
	auth     shared.Authorizer
	gate     WriteGate
	tx       shared.TxManager
}

// NewLocks returns the locks over project's gate, the work items' reads,
// the Authorizer, the write gate and the transactions.
func NewLocks(projects ProjectGate, issues IssueReader, auth shared.Authorizer, gate WriteGate, tx shared.TxManager) Locks {
	return Locks{projects: projects, issues: issues, auth: auth, gate: gate, tx: tx}
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

// run takes w's precheck, before any lock and outside any transaction;
// then the write gate of w's project; then in one transaction w's lock and
// decision, and runs fn in it with what they hold (M4 design 3.5, 11.11):
// w's whole write. It leaves the gate once the transaction has ended, on
// every path, a panic's too. A refusal of the precheck, the gate or the
// decision is its answer, and fn does not run.
func (l Locks) run(ctx context.Context, actor shared.Actor, w write, fn func(ctx context.Context, h held) error) error {
	w, err := l.precheck(ctx, actor, w)
	if err != nil {
		return err
	}
	leave, err := l.gate.Enter(ctx, w.project)
	if err != nil {
		return err
	}
	defer leave()
	return l.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := l.lockAndDecide(ctx, actor, w)
		if err != nil {
			return err
		}
		return fn(ctx, h)
	})
}

// precheck decides w's action for actor before any lock and outside any
// transaction (M4 design 3.5), and answers w on its project. For a write
// that names a work item: the work item read without a lock and the
// decision on it (see); a work item that is not there, whose project actor
// does not see, or that the guest rule hides from him, is
// issue.not_found. For one on a project: the project read without a lock
// and the decision on it; a project that is not there, or not visible to
// actor, is w.notFound. A role the rule does not allow is the Authorizer's
// shared.Forbidden. What it lets through, lockAndDecide decides again.
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
	return w, l.decide(ctx, actor, w.action, workspaceID, w.project, nil, w.notFound)
}

// lockAndDecide takes w's lock and decides w's action under it, in the
// transaction ctx carries: the project's row FOR NO KEY UPDATE; the work
// item w names read again; then the decision on the locked project, and on
// the work item as read again. It answers what they hold. A project
// deleted while the lock waited is w.notFound, and so is a work item
// deleted meanwhile, or no longer in the project, a project no longer
// visible to actor, and a work item the guest rule now hides from him; a
// role the rule no longer allows is the Authorizer's shared.Forbidden.
func (l Locks) lockAndDecide(ctx context.Context, actor shared.Actor, w write) (held, error) {
	p, found, err := l.projects.Lock(ctx, w.project)
	switch {
	case err != nil:
		return held{}, err
	case !found:
		return held{}, w.notFound
	}
	h := held{project: p}
	var on *domain.Issue
	if w.issue != nil {
		i, found, err := l.issues.GetIssue(ctx, *w.issue)
		switch {
		case err != nil:
			return held{}, err
		case !found || i.ProjectID != w.project:
			return held{}, w.notFound
		}
		h.issue, on = i, &i
	}
	if err := l.decide(ctx, actor, w.action, p.WorkspaceID, w.project, on, w.notFound); err != nil {
		return held{}, err
	}
	return h, nil
}

// see reads the undeleted work item id, archived or not, without a lock,
// and decides action on it for actor (M4 design 3.4, 3.5): a work item that
// is not there, whose project actor does not see, or that the guest rule
// hides from him, is issue.not_found; a role the rule does not allow is
// the Authorizer's shared.Forbidden. It answers the work item as read.
func (l Locks) see(ctx context.Context, actor shared.Actor, id uuid.UUID, action shared.Action) (domain.Issue, error) {
	i, found, err := l.issues.GetIssue(ctx, id)
	switch {
	case err != nil:
		return domain.Issue{}, err
	case !found:
		return domain.Issue{}, domain.ErrIssueNotFound
	}
	if err := l.decide(ctx, actor, action, i.WorkspaceID, i.ProjectID, &i, domain.ErrIssueNotFound); err != nil {
		return domain.Issue{}, err
	}
	return i, nil
}

// decide asks the Authorizer for action on the project projectID of the
// workspace workspaceID for actor, and, for an action on a work item, i, on
// it: the decision knows its creator (shared.Target.CreatorID), whom a rule
// may allow (AllowCreator). A project not visible to actor is notFound, and
// so is a work item the guest rule hides from him (M4 design 3.4, 6.7):
// the project's guest who sees only his own work items (Grant.OwnOnly), a
// workspace admin among them, does not see one another created, before
// what the rule says of his role, so another's work item is as unknown to
// him as one that is not there (8.2). i is nil for an action on the
// project.
func (l Locks) decide(ctx context.Context, actor shared.Actor, action shared.Action, workspaceID, projectID uuid.UUID, i *domain.Issue,
	notFound error) error {
	target := shared.Target{WorkspaceID: workspaceID, ProjectID: projectID}
	if i != nil && i.CreatedByID != nil {
		target.CreatorID = *i.CreatedByID
	}
	grant, err := l.auth.Authorize(ctx, actor, action, target)
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return notFound
	case i != nil && i.HiddenFrom(actor.UserID, grant.OwnOnly):
		return notFound
	}
	return err
}
