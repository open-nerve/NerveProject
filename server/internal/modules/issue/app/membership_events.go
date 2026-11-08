package app

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// MembershipRemovals are the writes a cycle's or a module's deletion asks
// of the work items' memberships (M6 P3a spec 2.5, R6), in the deleting
// transaction, which ctx carries and which each requires, under the lock of
// the deleted target's project the deletion holds (spec 6).
type MembershipRemovals interface {
	// RemoveCycleMemberships soft-deletes the cycle's undeleted memberships,
	// locked in id order, each deleted and last changed by by at at, and
	// answers the work items they were of, in id order; a membership deleted
	// before keeps its deletion.
	RemoveCycleMemberships(ctx context.Context, cycle uuid.UUID, at time.Time, by uuid.UUID) ([]uuid.UUID, error)
	// RemoveModuleMemberships is RemoveCycleMemberships of the module's.
	RemoveModuleMemberships(ctx context.Context, module uuid.UUID, at time.Time, by uuid.UUID) ([]uuid.UUID, error)
	InsertActivities(ctx context.Context, rows []ActivityRow) error
	MarkChanged(ctx context.Context, r ChangedRows) error
}

// NewMembershipRemovals consumes planning's deletion of a cycle and of a
// module (M6 P3a spec 2.5, R6) in the deleting transaction: the target's
// undeleted memberships removed at the deletion's moment by its actor; one
// row of activity for each work item that was in it, its leaving the target
// named as the event names it (domain.CycleLeft, domain.ModuleLeft), by the
// actor at that moment; then those work items last changed by the deletion
// (MarkChanged), as an attachment's change marks its work item. It publishes
// no issue.updated (precedent: the attachments' and the relations'
// changes). A target no work item is in writes nothing more.
func NewMembershipRemovals(rows MembershipRemovals) shared.EventSubscriber {
	return shared.EventSubscriber{Name: "issue_membership_removals", Types: []string{"planning.cycle_deleted", "planning.module_deleted"},
		Handle: func(ctx context.Context, e shared.Event) error {
			target, err := deletedTarget(e)
			if err != nil {
				return err
			}
			remove, activity := rows.RemoveCycleMemberships, domain.CycleLeft(target)
			if e.Entity.Kind == "module" {
				remove, activity = rows.RemoveModuleMemberships, domain.ModuleLeft(target)
			}
			at, by := e.OccurredAt, *e.ActorID
			issues, err := remove(ctx, target.ID, at, by)
			if err != nil || len(issues) == 0 {
				return err
			}
			activities := make([]ActivityRow, len(issues))
			for i, issue := range issues {
				activities[i] = ActivityRow{ID: uuid.NewV7(), WorkspaceID: e.WorkspaceID, ProjectID: e.ProjectID, IssueID: issue,
					Activity: activity, ActorID: &by, At: at}
			}
			if err := rows.InsertActivities(ctx, activities); err != nil {
				return err
			}
			return rows.MarkChanged(ctx, ChangedRows{IDs: issues, By: by, At: at})
		}}
}

// deletedTarget is the cycle or the module e tells the deletion of, by its
// id and its name before the deletion, read strictly before any write
// (precedents deletedProjects, attachmentIssue): the entity of its type's
// kind, the event's, its workspace's, its project's, its actor's ids and
// its moment set, and its one change the name, old and not empty, nothing
// else. Any other shape is an error, which fails the deletion.
func deletedTarget(e shared.Event) (domain.Named, error) {
	invalid := errors.New("invalid planning deletion event")
	kind := map[string]string{"planning.cycle_deleted": "cycle", "planning.module_deleted": "module"}[e.Type]
	if kind == "" || e.Entity.Kind != kind || e.ID == uuid.Nil() || e.Entity.ID == uuid.Nil() || e.WorkspaceID == uuid.Nil() ||
		e.ProjectID == uuid.Nil() || e.ActorID == nil || *e.ActorID == uuid.Nil() || e.OccurredAt.IsZero() || len(e.Changes) != 1 {
		return domain.Named{}, invalid
	}
	c := e.Changes[0]
	if c.Field != "name" || c.Old == nil || *c.Old == "" || c.New != nil || c.OldID != nil || c.NewID != nil {
		return domain.Named{}, invalid
	}
	return domain.Named{ID: e.Entity.ID, Name: *c.Old}, nil
}
