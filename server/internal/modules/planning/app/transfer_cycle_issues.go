package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// TransferStore is the cycles' store as a transfer uses it under its
// project's lock (M6 P4 spec 2.6): its target read again (CycleReader);
// whether the cycle it transfers from is frozen, by its frozen counts
// (SnapshotCountsReader); and the cycle's freeze (CycleFreezer).
type TransferStore interface {
	CycleReader
	SnapshotCountsReader
	CycleFreezer
}

// TransferCycleIssuesDeps are TransferCycleIssues' ports.
type TransferCycleIssuesDeps struct {
	Locks  Locks
	Cycles TransferStore
	Groups StateGroupReader
	Stats  IssueStats
	Issues IssueTransfer
	Clock  Clock
}

// TransferCycleIssues answers transferCycleIssues (M6 P4 spec 2.3, 2.6,
// R7, R8).
type TransferCycleIssues struct{ d TransferCycleIssuesDeps }

// NewTransferCycleIssues returns the use case over d.
func NewTransferCycleIssues(d TransferCycleIssuesDeps) *TransferCycleIssues {
	return &TransferCycleIssues{d: d}
}

// Execute takes the lock path on the cycle id (Locks, cycle.write); then
// under the lock, in this order (spec 2.3): an archived cycle is refused
// (planning.archived); the cycle target, read again, must be another
// undeleted cycle of the project (validation_failed at target_cycle_id),
// not archived (planning.archived); the clock, read last after the locks;
// on that moment's day in the project's time zone the cycle must be
// completed (planning.cycle_not_completed) and the target not
// (planning.cycle_completed). Then the cycle's progress is frozen, unless
// it is already, with the numbers it has before the move, at that moment
// (freezing, as the job); and its work items in the states of the
// backlog, unstarted and started groups move to the target, by the caller
// at that moment (IssueTransfer). The answer is how many moved: none
// writes nothing more than the freeze.
func (u *TransferCycleIssues) Execute(ctx context.Context, id, target uuid.UUID) (int, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return 0, err
	}
	moved := 0
	err = u.d.Locks.run(ctx, actor, write{named: named{cycle: &id}, action: domain.ActionCycleWrite}, func(ctx context.Context, h held) error {
		if h.cycle.ArchivedAt != nil {
			return domain.ErrArchived
		}
		to, err := u.target(ctx, h, target)
		if err != nil {
			return err
		}
		now := u.d.Clock.Now()
		today, err := domain.Today(now, h.project.Timezone)
		switch {
		case err != nil:
			return err
		case domain.StatusOf(h.cycle.Dates, today) != domain.StatusCompleted:
			return domain.ErrTransferFromOpenCycle
		case domain.StatusOf(to.Dates, today) == domain.StatusCompleted:
			return domain.ErrTransferToCompletedCycle
		}
		groups, err := u.d.Groups.StateGroups(ctx, h.projectID)
		if err != nil {
			return err
		}
		at := freezing{project: h.projectID, timezone: h.project.Timezone, groups: groups, now: now, today: today}
		if err := u.freeze(ctx, at, h.cycle); err != nil {
			return err
		}
		moved, err = u.d.Issues.TransferCycleIssues(ctx, Transfer{ProjectID: h.projectID, WorkspaceID: h.project.WorkspaceID,
			From: NamedCycle{ID: h.cycle.ID, Name: h.cycle.Name}, To: NamedCycle{ID: to.ID, Name: to.Name},
			StateIDs: domain.PendingStates(groups), By: actor.UserID, At: now})
		return err
	})
	return moved, err
}

// target reads, under the lock h holds, the cycle id a transfer from h's
// cycle moves to: one that is not there or is deleted, another project's,
// or h's cycle itself is refused at target_cycle_id, as no 404, which would
// tell of a cycle the caller may not see; an archived one is
// planning.archived.
func (u *TransferCycleIssues) target(ctx context.Context, h held, id uuid.UUID) (domain.Cycle, error) {
	c, found, err := u.d.Cycles.GetCycle(ctx, id)
	switch {
	case err != nil:
		return domain.Cycle{}, err
	case !found || c.ProjectID != h.projectID || c.ID == h.cycle.ID:
		return domain.Cycle{}, shared.Invalid(domain.TransferTargetNotAllowed())
	case c.ArchivedAt != nil:
		return domain.Cycle{}, domain.ErrArchived
	}
	return c, nil
}

// freeze freezes the cycle c, at's, unless it is frozen already, by the job
// or a transfer before: then its work items are not read again for a freeze
// the store would not write (spec R5).
func (u *TransferCycleIssues) freeze(ctx context.Context, at freezing, c domain.Cycle) error {
	frozen, err := u.d.Cycles.CycleSnapshotCounts(ctx, []uuid.UUID{c.ID})
	if err != nil || len(frozen) > 0 {
		return err
	}
	_, err = at.freeze(ctx, u.d.Stats, u.d.Cycles, c)
	return err
}
