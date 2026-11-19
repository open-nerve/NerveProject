package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IssueRemovalDeps are the ports the deletion core writes through while
// its caller holds the work item's write lock. They carry no clock: each
// caller gives deleteLocked its own.
type IssueRemovalDeps struct {
	Events    shared.EventPublisher
	Rows      IssueRowsDeleter
	Relations *RelationDeletions
}

// IssueRemoval is the deletion core (M7 P2b spec 2.2, R2, R6): it deletes
// a work item within its caller's transaction, under his write's lock
// (deleteLocked), and owns no authorization, gate or transaction boundary.
// Its rows' stamp is IssueDeletion.
type IssueRemoval struct{ d IssueRemovalDeps }

// NewIssueRemoval returns the shared deletion core.
func NewIssueRemoval(d IssueRemovalDeps) *IssueRemoval { return &IssueRemoval{d: d} }

// deleteLocked is the deletion core: the deletion, by the caller actorID,
// of the work item that h, his write's lock and decision, holds, in its
// transaction (M4 design 3.5, 3.6, 3.9). It owns no guard of its caller's
// write (write.check): deleteIssue and deleteIntakeIssue each decide their
// own action before it. Under the lock, in this order: an archived project
// refuses it (409 project.archived); its subtree, the work item and each
// one under it, however deep, archived ones too, read under the lock,
// which every write of a parent holds; the moment, read once from clock;
// the subtree's work items and each row under them soft-deleted, their
// intake rows among them, table by table in the global order
// (issueRowSteps); issue.deleted for each of them. All of it is the
// caller's at that moment. An archived work item is deleted as any other
// (3.8). A failing statement ends it with its failure, and none after it
// runs.
func (u *IssueRemoval) deleteLocked(ctx context.Context, actorID uuid.UUID, h held, clock Clock) error {
	id := h.issue.ID
	if h.project.Archived {
		return domain.ErrProjectArchived
	}
	subtree, err := u.d.Rows.Subtree(ctx, id)
	if err != nil {
		return err
	}
	d := IssueDeletion{IssueIDs: subtree, By: actorID, At: clock.Now()}
	for _, step := range issueRowSteps(u.d.Rows, func(ctx context.Context, d IssueDeletion) error {
		return u.d.Relations.ForIssues(ctx, h.issue.ProjectID, d)
	}) {
		if err := step(ctx, d); err != nil {
			return err
		}
	}
	for _, issueID := range subtree {
		s := stamp{workspace: h.issue.WorkspaceID, project: h.issue.ProjectID, issue: issueID, by: d.By, at: d.At}
		if err := u.d.Events.Publish(ctx, s.event(domain.EventIssueDeleted, nil)); err != nil {
			return err
		}
	}
	return nil
}
