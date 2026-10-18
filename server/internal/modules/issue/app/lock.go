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
// outside the write's transaction, the precheck: the project read without
// a lock, and the decision on it, which answers every refusal there, so a
// caller who would be refused takes no lock at all. Then, in the write's
// transaction, the project's row FOR NO KEY UPDATE, its workspace's row
// not locked: the write's first lock, which every write on the project's
// work items, states, labels and memberships also takes, so they run one
// after another; and the decision again under it, which is final: a
// removal or a demotion that committed while the write waited is seen.
// A read of a work item takes the same decision on its project, without a
// lock (see). issue.New builds one Locks for every use case: none holds an
// Authorizer of its own.
type Locks struct {
	projects ProjectGate
	issues   IssueReader
	auth     shared.Authorizer
}

// NewLocks returns the locks over project's gate, the work items' reads
// and the Authorizer.
func NewLocks(projects ProjectGate, issues IssueReader, auth shared.Authorizer) Locks {
	return Locks{projects: projects, issues: issues, auth: auth}
}

// write is a write on the work-item side of a project, as Locks takes its
// lock: the project, the action it decides, and the 404 of what the caller
// named.
type write struct {
	project  uuid.UUID
	action   shared.Action
	notFound error
}

// precheck reads w's project without a lock and decides w's action on it
// for actor, before any lock and outside any transaction (M4 design 3.5):
// a project that is not there or not visible to actor is w.notFound; a
// role the rule does not allow is the Authorizer's shared.Forbidden. What
// it lets through, lockAndDecide decides again.
func (l Locks) precheck(ctx context.Context, actor shared.Actor, w write) error {
	workspaceID, found, err := l.projects.Find(ctx, w.project)
	switch {
	case err != nil:
		return err
	case !found:
		return w.notFound
	}
	return l.decide(ctx, actor, w.action, shared.Target{WorkspaceID: workspaceID, ProjectID: w.project}, w.notFound)
}

// lockAndDecide takes w's lock and decides w's action under it, in the
// transaction ctx carries: the project's row FOR NO KEY UPDATE, then the
// decision on the locked project, which it answers. A project deleted
// while the lock waited is w.notFound, and so is one no longer visible to
// actor; a role the rule no longer allows is the Authorizer's
// shared.Forbidden.
func (l Locks) lockAndDecide(ctx context.Context, actor shared.Actor, w write) (LockedProject, error) {
	p, found, err := l.projects.Lock(ctx, w.project)
	switch {
	case err != nil:
		return LockedProject{}, err
	case !found:
		return LockedProject{}, w.notFound
	}
	if err := l.decide(ctx, actor, w.action, shared.Target{WorkspaceID: p.WorkspaceID, ProjectID: w.project}, w.notFound); err != nil {
		return LockedProject{}, err
	}
	return p, nil
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
