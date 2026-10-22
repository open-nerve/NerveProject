package app

import (
	"context"
	"errors"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The form of Locks.run for a write across projects (convention 7, M4
// design 3.5): the work items it reaches besides its own, each read and
// decided on without a lock first, so that one the caller may not reach
// takes no lock and no place in a gate, its project's above all; then the
// projects of the others, with the write's, locked all at once in
// ascending id order, and each work item read and decided on again under
// them.

// scope is what a write's precheck lets through to its lock: the projects
// it locks, in ascending id order, each once, the write's among them;
// and, across projects, which of the work items it reaches the precheck
// reached, in write.reach's order.
type scope struct {
	projects []uuid.UUID
	reached  []bool
}

// end is a work item a write across projects reaches, as its lock holds
// it: the work item and its project, read under the lock, and the caller's
// grant on the project, as decided under it; ok is false for a work item
// the write does not reach, and the rest is zero then.
type end struct {
	issue   domain.Issue
	project LockedProject
	grant   shared.Grant
	ok      bool
}

// lockProjects locks s's projects in the transaction ctx carries and
// answers those it locked, by id: a write on one project its row FOR NO
// KEY UPDATE (ProjectGate.Lock); a write across projects each of s's, all
// at once in ascending id order (ProjectGate.LockMany), even when they are
// one. A project deleted while the lock waited is not among them.
func (l Locks) lockProjects(ctx context.Context, w write, s scope) (map[uuid.UUID]LockedProject, error) {
	if w.reach != nil {
		return l.projects.LockMany(ctx, s.projects)
	}
	p, found, err := l.projects.Lock(ctx, w.project)
	if err != nil || !found {
		return nil, err
	}
	return map[uuid.UUID]LockedProject{w.project: p}, nil
}

// reachAll is the precheck of the work items ids that a write from the
// work item from reaches, before any lock (M4 design 3.5): each read and
// decided on without one (reach). It answers the scope of the write:
// from's project and those of the work items it reached, in ascending id
// order, each once, and which it reached; one it did not reach takes no
// lock. A failing read or decision is its error.
func (l Locks) reachAll(ctx context.Context, actor shared.Actor, from domain.Issue, ids []uuid.UUID) (scope, error) {
	s := scope{projects: []uuid.UUID{from.ProjectID}, reached: make([]bool, len(ids))}
	for n, id := range ids {
		e, err := l.reach(ctx, actor, from, id)
		if err != nil {
			return scope{}, err
		}
		if e.ok {
			s.reached[n] = true
			s.projects = append(s.projects, e.issue.ProjectID)
		}
	}
	slices.SortFunc(s.projects, uuid.UUID.Compare)
	s.projects = slices.Compact(s.projects)
	return s, nil
}

// reachUnder reads again, under the lock, each work item of ids that the
// precheck reached (s.reached), and decides on it again (reach): its ends,
// in ids' order. One the precheck did not reach is not reached, and
// neither is one whose project is not among locked, deleted while the lock
// waited. A failing read or decision is its error.
func (l Locks) reachUnder(ctx context.Context, actor shared.Actor, from domain.Issue, ids []uuid.UUID, s scope,
	locked map[uuid.UUID]LockedProject) ([]end, error) {
	ends := make([]end, len(ids))
	for n, id := range ids {
		if !s.reached[n] {
			continue
		}
		e, err := l.reach(ctx, actor, from, id)
		if err != nil {
			return nil, err
		}
		if p, found := locked[e.issue.ProjectID]; e.ok && found {
			e.project = p
			ends[n] = e
		}
	}
	return ends, nil
}

// reach reads the work item id and decides whether a write from the work
// item from reaches it (M4 design 3.4, 3.10): the work item is there,
// undeleted, it is not from, it is of from's workspace, and actor sees it,
// on issue.read on its project, the guest rule first. ok is false for one
// it does not reach: a refusal of the decision is no error here. A failing
// read or decision is its error.
func (l Locks) reach(ctx context.Context, actor shared.Actor, from domain.Issue, id uuid.UUID) (end, error) {
	i, found, err := l.issues.GetIssue(ctx, id)
	switch {
	case err != nil:
		return end{}, err
	case !found || i.ID == from.ID || i.WorkspaceID != from.WorkspaceID:
		return end{}, nil
	}
	grant, err := l.decide(ctx, actor, domain.ActionRead, i.WorkspaceID, i.ProjectID, &i, domain.ErrIssueNotFound)
	var refused *shared.Error
	switch {
	case errors.As(err, &refused):
		return end{}, nil
	case err != nil:
		return end{}, err
	}
	return end{issue: i, grant: grant, ok: true}, nil
}
