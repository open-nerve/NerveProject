package app

import (
	"context"
	"errors"
	"uuid"

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
// issue.New builds one Locks for every write: a write holds no Authorizer
// of its own.
type Locks struct {
	projects ProjectGate
	auth     shared.Authorizer
}

// NewLocks returns the locks over project's gate and the Authorizer.
func NewLocks(projects ProjectGate, auth shared.Authorizer) Locks {
	return Locks{projects: projects, auth: auth}
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
	return decide(ctx, l.auth, actor, w, workspaceID)
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
	if err := decide(ctx, l.auth, actor, w, p.WorkspaceID); err != nil {
		return LockedProject{}, err
	}
	return p, nil
}

// decide asks the Authorizer for w's action on w's project of the
// workspace for actor. A project not visible to actor is w.notFound.
func decide(ctx context.Context, auth shared.Authorizer, actor shared.Actor, w write, workspaceID uuid.UUID) error {
	_, err := auth.Authorize(ctx, actor, w.action, shared.Target{WorkspaceID: workspaceID, ProjectID: w.project})
	if errors.Is(err, shared.ErrNotVisible) {
		return w.notFound
	}
	return err
}
