package app

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// TriageIntakeIssueDeps supplies lock orchestration, the update core an
// accept changes the work item's state through, the intake rows' store,
// the work items' store, the project's states and the clock.
type TriageIntakeIssueDeps struct {
	Locks  Locks
	Update *UpdateIssue
	Intake IntakeWriter
	Issues IssueTriager
	States StateCatalog
	Clock  Clock
}

// TriageIntakeIssue triages a work item of a project's intake (M7 P2a
// spec 2.5).
type TriageIntakeIssue struct{ d TriageIntakeIssueDeps }

// NewTriageIntakeIssue returns the use case.
func NewTriageIntakeIssue(d TriageIntakeIssueDeps) *TriageIntakeIssue {
	return &TriageIntakeIssue{d: d}
}

// Execute checks t's body (domain.CheckTriage), before any read; then
// Locks runs the write on the work item id, decided on intake.triage, its
// 404 issue.intake_not_found: its precheck, the work item read and the
// decision on it, the guest rule first, so that a work item not there,
// whose project the caller does not see or that the guest rule hides from
// him is that 404, then a role the rule does not allow 403, then a work
// item without an intake row that 404 again (hasIntakeRow), each the one
// value a random id answers; then the write gate, and in one transaction
// the project's lock, the work item read again, the decision and the
// intake row's check again (spec 2.5, pre-flight I9). Under them, in this
// order: the project archived or its intake turned off refuses it
// (refuseIntakeOff); then the clock, read once: the project's lock, taken
// before this runs, is the request's last, so the moment comes after it,
// and every row the request writes is at it; the transition from the
// row's effective status at it (domain.IntakeFacts.Transition), else 409
// issue.intake_transition_not_allowed; the 422s that need the moment or a
// read under the lock (checked); the intake row (IntakeWriter.TriageIntake);
// its activity, from the effective status to the new one, naming no
// duplicate's target (domain.TriageActivity, R7). An accept then moves the
// work item to the project's default state, read under the lock, through
// the update core at the same moment (updateLocked, R6): the state's
// activity after the triage's, its completion, issue.updated, and the
// work item read back as the caller sees it. Any other triage changes no
// work item, publishes no event and notifies no one; the work item is read
// back and answered as the caller sees it at the triage's moment
// (Locks.seenAt), as an accept's is.
func (u *TriageIntakeIssue) Execute(ctx context.Context, id uuid.UUID, t domain.Triage) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	if err := domain.CheckTriage(t); err != nil {
		return domain.Issue{}, err
	}
	w := write{issue: &id, action: domain.ActionTriageIntake, notFound: domain.ErrIntakeNotFound, check: hasIntakeRow}
	var triaged domain.Issue
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		if err := refuseIntakeOff(h.project); err != nil {
			return err
		}
		now := u.d.Clock.Now()
		from, err := h.issue.Intake.Transition(t.Status, now)
		if err != nil {
			return err
		}
		row, err := u.checked(ctx, actor.UserID, h, t, now)
		if err != nil {
			return err
		}
		if err := u.d.Intake.TriageIntake(ctx, row); err != nil {
			return err
		}
		s := stamp{workspace: h.issue.WorkspaceID, project: h.issue.ProjectID, issue: id, by: actor.UserID, at: now}
		if err := u.d.Issues.InsertActivities(ctx, s.activity([]domain.Activity{domain.TriageActivity(from, t.Status)})); err != nil {
			return err
		}
		if t.Status == domain.IntakeAccepted {
			triaged, err = u.accept(ctx, actor.UserID, h, now)
			return err
		}
		back, err := readBack(ctx, u.d.Issues, id)
		if err != nil {
			return err
		}
		triaged, err = u.d.Locks.seenAt(ctx, actor.UserID, h.grant, back, moment(now))
		return err
	})
	if err != nil {
		return domain.Issue{}, err
	}
	return triaged, nil
}

// hasIntakeRow is a triage's check of its work item i (write.check), as
// read before the lock and again under it: one without an undeleted intake
// row is 404 issue.intake_not_found, the value of one not there.
func hasIntakeRow(_ context.Context, i domain.Issue, _ shared.Grant) error {
	if i.Intake == nil {
		return domain.ErrIntakeNotFound
	}
	return nil
}

// checked is the triage t of the work item h holds, by the caller actor,
// as its intake row is written at now, once the 422s that need the moment
// or a read under the lock pass (spec 2.5): a snooze's time, at the
// microsecond, after now (domain.CheckSnoozedTill); a duplicate's target
// one it may duplicate (duplicable), else 422 not_allowed on
// duplicate_to_id. CheckTriage has given each its field and no other.
func (u *TriageIntakeIssue) checked(ctx context.Context, actor uuid.UUID, h held, t domain.Triage, now time.Time) (IntakeTriage, error) {
	row := IntakeTriage{IssueID: h.issue.ID, Status: t.Status, By: actor, At: now}
	switch t.Status {
	case domain.IntakeSnoozed:
		till, err := domain.CheckSnoozedTill(*t.SnoozedTill, now)
		if err != nil {
			return IntakeTriage{}, err
		}
		row.SnoozedTill = &till
	case domain.IntakeDuplicate:
		ok, err := u.duplicable(ctx, actor, h, *t.DuplicateToID)
		if err != nil {
			return IntakeTriage{}, err
		}
		if !ok {
			return IntakeTriage{}, shared.Invalid(domain.DuplicateNotAllowed())
		}
		row.DuplicateToID = t.DuplicateToID
	}
	return row, nil
}

// duplicable reports whether the work item h holds, which the caller actor
// triages, may duplicate the work item target (spec 2.5, design 3.5): one
// other than itself, which is not read; undeleted, archived or not
// (pre-flight m6); of its project; seen by him there, the guest rule with
// his grant on it, h's (Issue.HiddenFrom); not in the project's intake. Each
// it is not answers the same false, read under the lock.
func (u *TriageIntakeIssue) duplicable(ctx context.Context, actor uuid.UUID, h held, target uuid.UUID) (bool, error) {
	if target == h.issue.ID {
		return false, nil
	}
	i, found, err := u.d.Issues.GetIssue(ctx, target)
	if err != nil {
		return false, err
	}
	return found && i.ProjectID == h.issue.ProjectID && !i.HiddenFrom(actor, h.grant.OwnOnly) && !i.InTriage(), nil
}

// accept moves the work item h holds, accepted by the caller actor at the
// triage's moment now, to the project's default state, read under the
// lock (pickState), through the update core at the same moment (R6), and
// answers its answer. A project without a default state is an error, as
// it is for a creation.
func (u *TriageIntakeIssue) accept(ctx context.Context, actor uuid.UUID, h held, now time.Time) (domain.Issue, error) {
	states, err := u.d.States.States(ctx, h.issue.ProjectID)
	if err != nil {
		return domain.Issue{}, err
	}
	state, found := pickState(states, nil)
	if !found {
		return domain.Issue{}, fmt.Errorf("project %s has no default state", h.issue.ProjectID)
	}
	return u.d.Update.updateLocked(ctx, actor, h, domain.IssuePatch{StateID: &state.ID}, moment(now))
}
