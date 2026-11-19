package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteIntakeIssueDeps supplies lock orchestration, the shared deletion
// core a work item not accepted is deleted through, the intake rows' store
// an accepted one's row alone is deleted through, and the clock.
type DeleteIntakeIssueDeps struct {
	Locks  Locks
	Delete *IssueRemoval
	Intake IntakeWriter
	Clock  Clock
}

// DeleteIntakeIssue deletes a work item of a project's intake, or the
// intake row of one accepted: DELETE /api/v0/intake-issues/{issue_id} (M7
// P2b spec 2.5).
type DeleteIntakeIssue struct{ d DeleteIntakeIssueDeps }

// NewDeleteIntakeIssue returns the use case.
func NewDeleteIntakeIssue(d DeleteIntakeIssueDeps) *DeleteIntakeIssue {
	return &DeleteIntakeIssue{d: d}
}

// Execute runs, through Locks, the deletion of the intake work item id (M7
// P2b spec 2.5, R6, R11), whatever its status: its precheck, the work item
// read and the decision on intake.delete on it, the guest rule first, so
// that a work item not there, whose project the caller does not see or
// that the guest rule hides from him is 404 issue.intake_not_found; then a
// role the rule does not allow 403: the project's admins may, and the
// member or above who made the work item, its submitter (AllowCreator), so
// that one who did not is refused before his own intake row is looked at
// (re-86 m5); then a work item without an intake row that 404 again
// (hasIntakeRow), each the one value a random id answers; then the write
// gate, and in one transaction the project's lock, the work item read
// again, the decision and the intake row's check again. Under them, the
// project archived or its intake turned off refuses it (refuseIntakeOff),
// the toggle read under the lock alone. Then a work item not accepted,
// pending, snoozed, declined or a duplicate, is deleted through the
// deletion core (deleteLocked), as deleteIssue deletes one: its rows, its
// intake row among them, and issue.deleted, at the moment its clock reads
// after the last lock. An accepted one, archived since or not, stays: its
// intake row alone is deleted at that moment (IntakeWriter.DeleteIntake),
// no activity written and no event published.
func (u *DeleteIntakeIssue) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	w := write{issue: &id, action: domain.ActionDeleteIntake, notFound: domain.ErrIntakeNotFound, check: hasIntakeRow}
	return u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		if err := refuseIntakeOff(h.project); err != nil {
			return err
		}
		if h.issue.Intake.Status != domain.IntakeAccepted {
			return u.d.Delete.deleteLocked(ctx, actor.UserID, h, u.d.Clock)
		}
		return u.d.Intake.DeleteIntake(ctx, IntakeDeletion{IssueID: id, By: actor.UserID, At: u.d.Clock.Now()})
	})
}
