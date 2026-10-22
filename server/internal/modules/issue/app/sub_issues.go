package app

import (
	"context"
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListSubIssues lists the work items directly under a work item (M4
// design 3.9, 5.1).
type ListSubIssues struct {
	locks  Locks
	issues SubIssueLister
}

// NewListSubIssues returns the use case.
func NewListSubIssues(locks Locks, issues SubIssueLister) *ListSubIssues {
	return &ListSubIssues{locks: locks, issues: issues}
}

// Execute reads the work item id and decides issue.read on it for the
// caller, as getIssue does, without a lock or a transaction (Locks.see),
// with its refusals; then its undeleted, unarchived direct sub-items that
// he sees, those he created when the guest rule hides others' from him
// (Grant.OwnOnly), in the order of their creation, each as he sees it
// (Locks.seen).
func (u *ListSubIssues) Execute(ctx context.Context, id uuid.UUID) ([]domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	i, grant, err := u.locks.see(ctx, actor, id, domain.ActionRead)
	if err != nil {
		return nil, err
	}
	var by *uuid.UUID
	if grant.OwnOnly {
		by = &actor.UserID
	}
	subItems, err := u.issues.SubIssues(ctx, i.ID, by)
	if err != nil {
		return nil, err
	}
	return u.locks.seen(ctx, actor.UserID, grants{i.ProjectID: grant}, subItems...)
}

// AddSubIssuesDeps are what addSubIssues reads and writes through.
type AddSubIssuesDeps struct {
	Locks  Locks
	Issues SubIssueAdder
	Clock  Clock
}

// AddSubIssues puts work items under a work item (M4 design 3.5, 3.9,
// 5.1).
type AddSubIssues struct {
	d AddSubIssuesDeps
}

// NewAddSubIssues returns the use case.
func NewAddSubIssues(d AddSubIssuesDeps) *AddSubIssues {
	return &AddSubIssues{d: d}
}

// Execute checks ids (domain.CheckSubIssueIDs), before any read; then
// Locks runs the write on the work item id, the parent (M4 design 3.5,
// 11.11): its precheck, the decision on issue.update on it, the guest rule
// first, before any lock; then the project's write gate, which refuses a
// write that may wait no longer (503 server_busy); then in one
// transaction, the project FOR NO KEY UPDATE, the parent read again and
// the decision again, and under them, in this order: an archived project
// refuses it (409 project.archived), then an archived parent (409
// issue.archived); the work items ids name, each in the caller's reach
// and not the parent or above it (check), every problem in one 422; then,
// when any is not under the parent yet, the clock; those work items under
// the parent, in id order, each changed last by the caller; for each of
// them a row of its activity, from its parent before, by its identifier,
// to this one (row 5 of 3.13). All of it is the caller's at that moment.
// The answer is those work items as stored, read back in the transaction,
// in id order, as the caller sees them (Locks.seen); none, and nothing
// written, when each is under the parent already.
func (u *AddSubIssues) Execute(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	if err := domain.CheckSubIssueIDs(ids); err != nil {
		return nil, err
	}
	w := write{issue: &id, action: domain.ActionUpdate, notFound: domain.ErrIssueNotFound}
	var added []domain.Issue
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		switch {
		case h.project.Archived:
			return domain.ErrProjectArchived
		case h.issue.ArchivedAt != nil:
			return domain.ErrIssueArchived
		}
		moved, err := u.check(ctx, h, actor.UserID, ids)
		if err != nil || len(moved) == 0 {
			return err
		}
		s := stamp{workspace: h.issue.WorkspaceID, project: h.issue.ProjectID, issue: id, by: actor.UserID, at: u.d.Clock.Now()}
		movedIDs := make([]uuid.UUID, len(moved))
		for n, m := range moved {
			movedIDs[n] = m.item.ID
		}
		if err := u.d.Issues.SetParents(ctx, ParentRows{IDs: movedIDs, ParentID: id, By: s.by, At: s.at}); err != nil {
			return err
		}
		var rows []ActivityRow
		for _, m := range moved {
			under := s
			under.issue = m.item.ID
			rows = append(rows, under.activity([]domain.Activity{domain.ParentActivity(m.parent)})...)
		}
		if err := u.d.Issues.InsertActivities(ctx, rows); err != nil {
			return err
		}
		stored, err := u.d.Issues.GetIssues(ctx, movedIDs)
		switch {
		case err != nil:
			return err
		case len(stored) != len(movedIDs):
			return fmt.Errorf("only %d of %d work items are there after their write", len(stored), len(movedIDs))
		}
		added, err = u.d.Locks.seen(ctx, actor.UserID, grants{h.issue.ProjectID: h.grant}, stored...)
		return err
	})
	if err != nil {
		return nil, err
	}
	return added, nil
}

// move is a work item an addition of sub-items puts under the work item:
// it as read under the lock, and its parent's change as its activity names
// it.
type move struct {
	item   domain.Issue
	parent domain.Changed[*domain.Ref]
}

// check reads the work items ids, which the caller actor's addition puts
// under the work item h holds, under h (M4 design 3.4, 3.9): each an
// undeleted work item in his reach (inReach) that is neither the work item
// nor one above it, in its ancestry (domain.Loops). Every problem is
// reported at once, in one 422, each at its place in ids. It answers those
// not under the work item yet, in id order: the ones it moves (moves).
func (u *AddSubIssues) check(ctx context.Context, h held, actor uuid.UUID, ids []uuid.UUID) ([]move, error) {
	found, err := u.d.Issues.GetIssues(ctx, slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare))
	if err != nil {
		return nil, err
	}
	ancestry, err := u.d.Issues.Ancestry(ctx, h.issue.ID)
	if err != nil {
		return nil, err
	}
	var problems []shared.FieldError
	var moving []domain.Issue
	for at, id := range ids {
		n := slices.IndexFunc(found, func(i domain.Issue) bool { return i.ID == id })
		if n < 0 || !inReach(h, actor, h.issue.ProjectID, found[n]) || domain.Loops(id, ancestry) {
			problems = append(problems, domain.SubIssueNotAllowed(at))
			continue
		}
		if found[n].ParentID == nil || *found[n].ParentID != h.issue.ID {
			moving = append(moving, found[n])
		}
	}
	if len(problems) > 0 {
		return nil, shared.Invalid(problems...)
	}
	slices.SortFunc(moving, func(a, b domain.Issue) int { return a.ID.Compare(b.ID) })
	return u.moves(ctx, h, moving)
}

// moves are the work items moving under the work item h holds, each with
// its parent's change: from its parent before (parentsBefore) to the work
// item, by its identifier (refOf).
func (u *AddSubIssues) moves(ctx context.Context, h held, moving []domain.Issue) ([]move, error) {
	before, err := parentsBefore(ctx, u.d.Issues, h, moving)
	if err != nil {
		return nil, err
	}
	out := make([]move, len(moving))
	for n, i := range moving {
		out[n] = move{item: i, parent: domain.Changed[*domain.Ref]{Old: before[n], New: refOf(h.project, h.issue)}}
	}
	return out, nil
}
