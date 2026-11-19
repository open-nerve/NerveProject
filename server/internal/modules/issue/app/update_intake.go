package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateIntakeIssueDeps supplies lock orchestration, the shared update
// core and the clock an edit's moment is read from.
type UpdateIntakeIssueDeps struct {
	Locks  Locks
	Update *IssueUpdate
	Clock  Clock
}

// UpdateIntakeIssue changes the content of a work item of a project's
// intake: PATCH /api/v0/intake-issues/{issue_id} (M7 P2b spec 2.4).
type UpdateIntakeIssue struct{ d UpdateIntakeIssueDeps }

// NewUpdateIntakeIssue returns the use case.
func NewUpdateIntakeIssue(d UpdateIntakeIssueDeps) *UpdateIntakeIssue {
	return &UpdateIntakeIssue{d: d}
}

// Execute checks edit (domain.CheckIntakeIssueUpdate) into the patch of
// its fields, before any read; then Locks runs the write on the work item
// id, decided on intake.update, its 404 issue.intake_not_found (spec 2.4,
// DP-8): its precheck, the work item read and the decision on it, the
// guest rule first, so that a work item not there, whose project the
// caller does not see or that the guest rule hides from him is that 404,
// then a role the rule does not allow 403; then the edit's check
// (intakeEditor): a work item without an intake row 404 again, each the
// one value a random id answers, then its relative rules, 403; then the
// write gate, and in one transaction the project's lock, the work item
// read again, the decision and the check again, on the grant as it is
// then. Under them, in this order: the project archived or its intake
// turned off refuses it (refuseIntakeOff), the toggle read under the lock
// alone; a work item accepted, declined or a duplicate is 409
// issue.intake_closed, by its stored status, no clock read
// (domain.IntakeFacts.CheckOpen); then the update core with the patch
// (updateLocked): its 422s, its writes, activity, issue.updated, mentions,
// description version and images, at the moment its clock reads after the
// last lock, and its answer, the work item read back as the caller sees
// it, its intake row with it. updateIssue's guard of a work item in the
// intake (refuseTriage) is its own write's, not the core's: this write
// reaches the core on such a work item, while updateIssue refuses it.
func (u *UpdateIntakeIssue) Execute(ctx context.Context, id uuid.UUID, edit domain.IntakeIssueUpdate) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	p, err := domain.CheckIntakeIssueUpdate(edit)
	if err != nil {
		return domain.Issue{}, err
	}
	w := write{issue: &id, action: domain.ActionUpdateIntake, notFound: domain.ErrIntakeNotFound, check: intakeEditor(actor.UserID, edit.ContentOnly())}
	var updated domain.Issue
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		if err := refuseIntakeOff(h.project); err != nil {
			return err
		}
		if err := h.issue.Intake.CheckOpen(); err != nil {
			return err
		}
		updated, err = u.d.Update.updateLocked(ctx, actor.UserID, h, p, u.d.Clock)
		return err
	})
	if err != nil {
		return domain.Issue{}, err
	}
	return updated, nil
}

// intakeEditor is the check (write.check) of a content edit by actorID of
// its work item i, whose fields are the content alone or not
// (domain.IntakeIssueUpdate.ContentOnly), as Locks reads it before the
// lock and again under it, each time on the grant its decision answered
// (spec 2.4, R4, R11, DP-5, DP-6): a work item without an intake row is
// 404 issue.intake_not_found (hasIntakeRow); then the submitter rule
// (refuseNonSubmitter); then a caller who is not the project's admin and
// is its guest is 403 forbidden for fields but its name, its description
// and skip_activity, given even as they are.
func intakeEditor(actorID uuid.UUID, contentOnly bool) func(context.Context, domain.Issue, shared.Grant) error {
	return func(ctx context.Context, i domain.Issue, g shared.Grant) error {
		if err := hasIntakeRow(ctx, i, g); err != nil {
			return err
		}
		if err := refuseNonSubmitter(actorID, i, g); err != nil {
			return err
		}
		if !g.ProjectAdmin && g.ProjectRole == shared.RoleGuest && !contentOnly {
			return shared.Forbidden()
		}
		return nil
	}
}

// refuseNonSubmitter is the submitter rule of a write by actorID on the
// work item i in the intake, on the grant g its decision answered (spec
// 2.4, R4): the project's admin (Grant.ProjectAdmin: its admin, or the
// workspace's admin who is its member, its guest too) passes; any other
// caller is 403 forbidden for a work item he did not submit, which he did
// not create (Issue.CreatedBy), so for one of no known creator too. A
// content edit's (intakeEditor), and an asset's write's on a work item in
// the intake (AssetAccess.read, P2c spec 2.2).
func refuseNonSubmitter(actorID uuid.UUID, i domain.Issue, g shared.Grant) error {
	if !g.ProjectAdmin && !i.CreatedBy(actorID) {
		return shared.Forbidden()
	}
	return nil
}
