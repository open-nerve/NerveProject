package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The form of Locks.run for a write on either of two work items
// (deleteIssueRelation's, M4 design 3.10): it is the caller's to make
// when he may make it on either, decided on the first, then, when that
// refuses, on the second; both are read, and both their projects locked
// all at once in ascending id order (convention 7), whichever was decided
// on, since the write writes on both.

// both reads the two work items of a write on either of them, w.issue's,
// then w.instead's. One not there is w.notFound.
func (l Locks) both(ctx context.Context, w write) ([2]domain.Issue, error) {
	var items [2]domain.Issue
	for n, id := range []uuid.UUID{*w.issue, *w.instead} {
		i, found, err := l.issues.GetIssue(ctx, id)
		switch {
		case err != nil:
			return [2]domain.Issue{}, err
		case !found:
			return [2]domain.Issue{}, w.notFound
		}
		items[n] = i
	}
	return items, nil
}

// eitherOf is the precheck of a write on either of two work items, before
// any lock (M4 design 3.5, 3.10): both read (both), and the decision
// (decideEither). It answers the scope of the write: both their projects,
// in ascending id order, each once.
func (l Locks) eitherOf(ctx context.Context, actor shared.Actor, w write) (scope, error) {
	items, err := l.both(ctx, w)
	if err != nil {
		return scope{}, err
	}
	if _, err := l.decideEither(ctx, actor, w, items); err != nil {
		return scope{}, err
	}
	return scope{projects: ascending([]uuid.UUID{items[0].ProjectID, items[1].ProjectID})}, nil
}

// eitherUnder reads again, under the lock, the two work items of a write
// on either of them (both), and decides on them again (decideEither): it
// holds the first and its project, the second and its project
// (held.instead), and the grant that let the write through. A work item
// deleted while the lock waited is w.notFound, and so, before any
// decision, is one of a project deleted meanwhile, which locked does not
// hold.
func (l Locks) eitherUnder(ctx context.Context, actor shared.Actor, w write, locked map[uuid.UUID]LockedProject) (held, error) {
	items, err := l.both(ctx, w)
	if err != nil {
		return held{}, err
	}
	p, foundFirst := locked[items[0].ProjectID]
	q, foundSecond := locked[items[1].ProjectID]
	if !foundFirst || !foundSecond {
		return held{}, w.notFound
	}
	grant, err := l.decideEither(ctx, actor, w, items)
	if err != nil {
		return held{}, err
	}
	return held{project: p, issue: items[0], grant: grant, instead: end{issue: items[1], project: q, ok: true}}, nil
}

// decideEither decides w's action for actor on the first of items, then,
// when that refuses, on the second, each on its own project, the guest
// rule first (M4 design 3.4, 3.10), and answers the grant of the decision
// that let the write through. Both refusing is w.notFound when actor sees
// neither, else the refusal of the first he sees, his role's (forbidden).
// A failing decision is its error.
func (l Locks) decideEither(ctx context.Context, actor shared.Actor, w write, items [2]domain.Issue) (shared.Grant, error) {
	var refusal error
	for _, i := range items {
		grant, err := l.decide(ctx, actor, w.action, i.WorkspaceID, i.ProjectID, &i, domain.ErrIssueNotFound)
		var refused *shared.Error
		switch {
		case err == nil:
			return grant, nil
		case !errors.As(err, &refused):
			return shared.Grant{}, err
		case refusal == nil && !errors.Is(err, domain.ErrIssueNotFound):
			refusal = err
		}
	}
	if refusal == nil {
		refusal = w.notFound
	}
	return shared.Grant{}, refusal
}
