package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// SubmitIntakeIssueDeps supplies lock orchestration, the shared creation
// core, the intake rows' store and the read of the work item written.
type SubmitIntakeIssueDeps struct {
	Locks    Locks
	Creation *IssueCreation
	Intake   IntakeWriter
	Issues   IssueReader
}

// SubmitIntakeIssue submits a work item to a project's intake (M7 P2a spec
// 2.4).
type SubmitIntakeIssue struct{ d SubmitIntakeIssueDeps }

// NewSubmitIntakeIssue returns the use case.
func NewSubmitIntakeIssue(d SubmitIntakeIssueDeps) *SubmitIntakeIssue {
	return &SubmitIntakeIssue{d: d}
}

// Execute checks in (domain.CheckNewIntakeIssue), before any read; then
// Locks runs the write on the project, decided on intake.submit, as a
// creation's (CreateIssue): its precheck, the write gate, then in one
// transaction the project's lock and the decision again. Under them, in
// this order: the project archived or its intake turned off refuses it
// (refuseIntakeOff); the creation core writes the work item in the
// project's triage state (createLocked, inTriage), every creation's rows
// and its event; then its intake row, pending, the caller's at the
// creation's one moment; then the work item is read back. All of it is the
// caller's, at that moment.
//
// The core publishes issue.created, its last write, before the intake row
// is inserted. That is safe (M7 P2a spec 2.4): no subscriber of the event
// within the transaction reads an intake row, and what they enqueue runs
// after the commit, which the work item and its intake row reach together.
// The answer is the work item read back after its intake row, as the
// caller sees it at the creation's moment (Locks.seenAt), as every other
// answer of a work item in full is (M7 P2a spec 2.8): its intake row, and
// whether he subscribes to it. The core's own answer, read before the
// intake row is, is not it.
func (u *SubmitIntakeIssue) Execute(ctx context.Context, projectID uuid.UUID, in domain.NewIntakeIssue) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	issue, err := domain.CheckNewIntakeIssue(in)
	if err != nil {
		return domain.Issue{}, err
	}
	w := write{project: projectID, action: domain.ActionSubmitIntake, notFound: domain.ErrProjectNotFound}
	var submitted domain.Issue
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		if err := refuseIntakeOff(h.project); err != nil {
			return err
		}
		created, at, err := u.d.Creation.createLocked(ctx, actor.UserID, projectID, h, issue, inTriage)
		if err != nil {
			return err
		}
		row := IntakeRow{ID: uuid.NewV7(), WorkspaceID: h.project.WorkspaceID, ProjectID: projectID, IssueID: created.ID, By: actor.UserID, At: at}
		if err := u.d.Intake.InsertIntake(ctx, row); err != nil {
			return err
		}
		back, err := readBack(ctx, u.d.Issues, created.ID)
		if err != nil {
			return err
		}
		submitted, err = u.d.Locks.seenAt(ctx, actor.UserID, h.grant, back, moment(at))
		return err
	})
	if err != nil {
		return domain.Issue{}, err
	}
	return submitted, nil
}

// refuseIntakeOff refuses a write on the intake of the project p, as its
// write's lock answers it (M7 P2a spec 2.4, 2.5): an archived project
// first (409 project.archived), as planning's toggles come after it, then
// its intake turned off (409 issue.intake_disabled). The toggle is read
// under the lock alone, as the cycles' and modules' toggles are for a work
// item's write (readMembership): the project's update that turns it off
// takes the same lock.
func refuseIntakeOff(p LockedProject) error {
	switch {
	case p.Archived:
		return domain.ErrProjectArchived
	case !p.IntakeView:
		return domain.ErrIntakeDisabled
	}
	return nil
}
