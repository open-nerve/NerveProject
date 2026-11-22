package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// RecentVisitLocks is the one way a recording of a recent visit takes its
// locks and its decision (P3d spec 2.7, 2.8), beside the views' Locks and
// FavoriteLocks, whose workspaces' and projects' ports it shares. Before any
// lock and outside the recording's transaction, its check (recording): its
// target's place read without a lock, which names the workspace it decides
// on and the project it locks; recent_visit.write's decision on that
// workspace; and what the caller sees of the target. Then, in the
// recording's transaction (run), the workspace's row FOR SHARE and the
// decision again; the project's row FOR SHARE (M3 design 3.6 convention
// 2); the target's place and what he sees of it read again under them,
// which the recording decides on; then its last lock, the
// transaction-level advisory lock of his visits in the workspace, under
// which it reads the moment and writes. No write gate (spec 2.4). A work
// item's or a project's deletion, a change of a member or a role and an
// archiving lock the project's or the workspace's row FOR NO KEY UPDATE, so
// they and a recording run one after the other, and what the recording
// decides under its locks stays so until it commits. The advisory lock's
// holder waits for no row (spec 2.8).
type RecentVisitLocks struct {
	workspaces Workspaces
	projects   Projects
	targets    RecentTargets
	locker     RecentVisitLocker
	auth       shared.Authorizer
	tx         shared.TxManager
}

// NewRecentVisitLocks returns the locks over the workspace module's
// directory, the project's lock, the reads of the targets, the advisory
// lock, the Authorizer and the recordings' transactions.
func NewRecentVisitLocks(workspaces Workspaces, projects Projects, targets RecentTargets, locker RecentVisitLocker, auth shared.Authorizer,
	tx shared.TxManager) RecentVisitLocks {
	return RecentVisitLocks{workspaces: workspaces, projects: projects, targets: targets, locker: locker, auth: auth, tx: tx}
}

// recentWrite is a recording as RecentVisitLocks takes it, once checked:
// its target, and where its check found it, the workspace it decides on
// and the project it locks.
type recentWrite struct {
	target recentTarget
	place  recentPlace
}

// recording is the recording of the caller's visit of t, checked before
// any lock (spec 2.7): t's place, read without a lock; recent_visit.write's
// decision on its workspace; what he sees of t there (decide). A target
// not there, deleted, of a workspace he is not a member of or that he does
// not see is t.notFound, the one value, and takes no lock.
func (l RecentVisitLocks) recording(ctx context.Context, actor shared.Actor, t recentTarget) (recentWrite, error) {
	p, found, err := l.targets.place(ctx, t)
	switch {
	case err != nil:
		return recentWrite{}, err
	case !found:
		return recentWrite{}, t.notFound
	}
	if err := allow(ctx, l.auth, actor, domain.ActionRecentVisitWrite, p.workspaceID, t.notFound); err != nil {
		return recentWrite{}, err
	}
	if err := l.see(ctx, actor, t, p.workspaceID); err != nil {
		return recentWrite{}, err
	}
	return recentWrite{target: t, place: p}, nil
}

// run takes, in one transaction, w's locks and the decision again, and runs
// fn under them: the workspace's row FOR SHARE and recent_visit.write's
// decision on it; its project's row FOR SHARE; its target's place, which
// must be where the check found it, and what the caller sees of it, read
// again (see); and last the advisory lock of his visits in the workspace.
// A workspace or a project deleted while its lock waited, a decision
// refused, a target deleted, moved or no longer seen under them is the
// target's 404; fn does not run and the advisory lock is not taken.
func (l RecentVisitLocks) run(ctx context.Context, actor shared.Actor, w recentWrite, fn func(ctx context.Context) error) error {
	return l.tx.WithinTx(ctx, func(ctx context.Context) error {
		found, err := l.workspaces.ShareWorkspaceByID(ctx, w.place.workspaceID)
		switch {
		case err != nil:
			return err
		case !found:
			return w.target.notFound
		}
		if err := allow(ctx, l.auth, actor, domain.ActionRecentVisitWrite, w.place.workspaceID, w.target.notFound); err != nil {
			return err
		}
		if _, found, err := l.projects.Share(ctx, w.place.projectID); err != nil || !found {
			return orNotFound(err, w.target.notFound)
		}
		p, found, err := l.targets.place(ctx, w.target)
		if err != nil || !found || p != w.place {
			return orNotFound(err, w.target.notFound)
		}
		if err := l.see(ctx, actor, w.target, w.place.workspaceID); err != nil {
			return err
		}
		if err := l.locker.LockRecentVisits(ctx, actor.UserID, w.place.workspaceID); err != nil {
			return err
		}
		return fn(ctx)
	})
}

// see reads what actor sees of t in the workspace (RecentTargets.sight) and
// answers t.notFound unless he sees it (recentSight.sees): the decision of
// a recording's check and of its run, the one predicate.
func (l RecentVisitLocks) see(ctx context.Context, actor shared.Actor, t recentTarget, workspaceID uuid.UUID) error {
	var issues []uuid.UUID
	if t.kind == domain.RecentVisitIssue {
		issues = []uuid.UUID{t.id}
	}
	s, err := l.targets.sight(ctx, actor, workspaceID, issues, nil)
	switch {
	case err != nil:
		return err
	case !s.sees(t.kind, t.id):
		return t.notFound
	}
	return nil
}

// orNotFound is err, or notFound when err is nil: a read under the locks
// that failed, or that did not find what it read.
func orNotFound(err, notFound error) error {
	if err != nil {
		return err
	}
	return notFound
}
