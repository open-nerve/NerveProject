package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// NewForgetDeletedIssueVisits deletes, within a work item's deletion's
// transaction, every account's recent visits of the work item (P3d spec
// 2.6, R10): one event, one call, for each work item of a deleted subtree,
// an intake's work item among them. Each in the event's workspace, at its
// moment and by its actor, none for nerve's own deletion, through visits,
// which locks them in id order and keeps their visits. The type is the
// string the issue module publishes; it is not imported, as the favorites'
// cascades import neither of theirs. The event is validated whole before
// any write (isIssueDeletion); a malformed one is refused, and its
// publisher's write with it.
func NewForgetDeletedIssueVisits(visits RecentVisitCascades) shared.EventSubscriber {
	return shared.EventSubscriber{Name: "navigation_forget_deleted_issue_visits", Types: []string{"issue.deleted"}, Handle: func(ctx context.Context, e shared.Event) error {
		if !isIssueDeletion(e) {
			return errors.New("invalid work item deletion event")
		}
		return visits.DeleteIssueRecentVisits(ctx, e.WorkspaceID, e.Entity.ID, e.OccurredAt, e.ActorID)
	}}
}

// isIssueDeletion tells whether e is a work item's deletion as the issue
// module writes it (its delete core, spec 2.6): its type, a work item with
// an id, a moment, its workspace and its project, an actor that is none or
// an account, and no change.
func isIssueDeletion(e shared.Event) bool {
	return e.Type == "issue.deleted" && e.Entity.Kind == "issue" && e.Entity.ID != uuid.Nil() && !e.OccurredAt.IsZero() &&
		e.WorkspaceID != uuid.Nil() && e.ProjectID != uuid.Nil() && (e.ActorID == nil || *e.ActorID != uuid.Nil()) && len(e.Changes) == 0
}
