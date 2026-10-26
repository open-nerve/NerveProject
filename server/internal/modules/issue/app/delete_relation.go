package app

import (
	"context"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteIssueRelationDeps are what deleteIssueRelation reads and writes
// through.
type DeleteIssueRelationDeps struct {
	Events    shared.EventPublisher
	Locks     Locks
	Relations RelationDeleter
	Clock     Clock
}

// DeleteIssueRelation deletes a relation: DELETE
// /api/v0/issue-relations/{relation_id} (M4 design 3.5, 3.10, 5.1).
type DeleteIssueRelation struct {
	d DeleteIssueRelationDeps
}

// NewDeleteIssueRelation returns the use case.
func NewDeleteIssueRelation(d DeleteIssueRelationDeps) *DeleteIssueRelation {
	return &DeleteIssueRelation{d: d}
}

// Execute reads the relation id without a lock: one not there, or
// deleted, is issue.relation_not_found. Then Locks runs the write on
// either of its work items (M4 design 3.5, 3.10, convention 7): its
// precheck, before any lock, both work items read, and issue.update
// decided on the one stored as issue_id, then, when that refuses, on the
// other, the guest rule first: neither seen is issue.relation_not_found,
// else a role that may change neither forbidden; then the write gates of
// both their projects, all at once in ascending id order, which refuse a
// write that may wait no longer (503 server_busy); then in one
// transaction those projects FOR NO KEY UPDATE, in ascending id order,
// both work items read and decided on again, and under them, in this
// order: the relation read again, deleted meanwhile
// issue.relation_not_found; either work item of an archived project
// refuses it (409 project.archived), then either archived work item (409
// issue.archived); then the clock and the write (delete). All of it is
// the caller's at that moment.
func (u *DeleteIssueRelation) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	r, found, err := u.d.Relations.GetRelation(ctx, id)
	switch {
	case err != nil:
		return err
	case !found:
		return domain.ErrRelationNotFound
	}
	w := write{issue: &r.IssueID, instead: &r.RelatedID, action: domain.ActionUpdate, notFound: domain.ErrRelationNotFound}
	return u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		_, found, err := u.d.Relations.GetRelation(ctx, id)
		switch {
		case err != nil:
			return err
		case !found:
			return domain.ErrRelationNotFound
		case h.project.Archived || h.instead.project.Archived:
			return domain.ErrProjectArchived
		case h.issue.ArchivedAt != nil || h.instead.issue.ArchivedAt != nil:
			return domain.ErrIssueArchived
		}
		return u.delete(ctx, h, r, actor.UserID)
	})
}

// delete deletes the relation r between the work items h holds, the
// caller by's at the clock's moment (M4 design 3.7, 3.10, 3.13): the
// relation soft-deleted; both work items last changed by him, in id
// order; a row of each one's activity, of its project, the one stored as
// issue_id first, the type as it reads it, the other by its identifier
// (rows 22 and 23 of 3.13).
func (u *DeleteIssueRelation) delete(ctx context.Context, h held, r domain.Relation, by uuid.UUID) error {
	at := u.d.Clock.Now()
	if err := u.d.Relations.DeleteRelation(ctx, RelationDeletion{ID: r.ID, By: by, At: at}); err != nil {
		return err
	}
	first, second := h.own(), h.instead
	changed := ascending([]uuid.UUID{first.issue.ID, second.issue.ID})
	if err := u.d.Relations.MarkChanged(ctx, ChangedRows{IDs: changed, By: by, At: at}); err != nil {
		return err
	}
	rows := slices.Concat(
		first.stamp(by, at).activity([]domain.Activity{domain.RelationRemoved(r.TypeFrom(first.issue.ID), *refOf(second.project, second.issue))}),
		second.stamp(by, at).activity([]domain.Activity{domain.RelationRemoved(r.TypeFrom(second.issue.ID), *refOf(first.project, first.issue))}))
	return u.d.Relations.InsertActivities(ctx, rows)
}
