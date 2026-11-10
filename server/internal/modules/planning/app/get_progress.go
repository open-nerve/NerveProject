package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// SnapshotReader reads the frozen progress of the undeleted cycle id, whole,
// in the transaction ctx carries; nil while it is not frozen (M6 P4 spec
// R5).
type SnapshotReader interface {
	CycleSnapshot(ctx context.Context, id uuid.UUID) (*domain.Frozen, error)
}

// GetCycleProgressDeps are GetCycleProgress's ports.
type GetCycleProgressDeps struct {
	Reads     Reads
	Groups    StateGroupReader
	Stats     IssueStats
	Snapshots SnapshotReader
	Clock     Clock
}

// GetCycleProgress answers getCycleProgress (M6 P4 spec 2.3).
type GetCycleProgress struct{ d GetCycleProgressDeps }

// NewGetCycleProgress returns the use case over d.
func NewGetCycleProgress(d GetCycleProgressDeps) *GetCycleProgress { return &GetCycleProgress{d: d} }

// Execute reads the progress of the cycle id for the caller (Reads,
// cycle.read), all of it in the read's one snapshot: its status and its live
// progress, of the work items the caller sees, from its start to its end, on
// the project's day at the clock's reading, last; and its frozen progress,
// unless the caller sees only his own work items (OwnOnly), whom it is not
// read for (M6 P4 spec R1, R5).
func (u *GetCycleProgress) Execute(ctx context.Context, id uuid.UUID) (domain.CycleProgress, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.CycleProgress{}, err
	}
	var out domain.CycleProgress
	err = u.d.Reads.run(ctx, actor, domain.ActionCycleRead, named{cycle: &id}, func(ctx context.Context, h held) error {
		groups, err := u.d.Groups.StateGroups(ctx, h.projectID)
		if err != nil {
			return err
		}
		viewer := h.viewer(actor)
		items, err := u.d.Stats.CycleItems(ctx, h.projectID, h.cycle.ID, viewer)
		if err != nil {
			return err
		}
		var snapshot *domain.Frozen
		if viewer == nil {
			if snapshot, err = u.d.Snapshots.CycleSnapshot(ctx, h.cycle.ID); err != nil {
				return err
			}
		}
		today, err := domain.Today(u.d.Clock.Now(), h.project.Timezone)
		if err != nil {
			return err
		}
		live, err := domain.ProgressOf(domain.ProgressFacts{Items: items, Groups: groups, Start: h.cycle.Dates.Start, End: h.cycle.Dates.End,
			Timezone: h.project.Timezone, Today: today})
		if err != nil {
			return err
		}
		out = domain.CycleProgress{Status: domain.StatusOf(h.cycle.Dates, today), Live: live, Snapshot: snapshot}
		return nil
	})
	return out, err
}

// GetModuleProgressDeps are GetModuleProgress's ports.
type GetModuleProgressDeps struct {
	Reads  Reads
	Groups StateGroupReader
	Stats  IssueStats
	Clock  Clock
}

// GetModuleProgress answers getModuleProgress (M6 P4 spec 2.3).
type GetModuleProgress struct{ d GetModuleProgressDeps }

// NewGetModuleProgress returns the use case over d.
func NewGetModuleProgress(d GetModuleProgressDeps) *GetModuleProgress {
	return &GetModuleProgress{d: d}
}

// Execute reads the progress of the module id for the caller (Reads,
// module.read), in the read's one snapshot: of the work items the caller
// sees, from its start to its target day, on the project's day at the
// clock's reading, last. A module is never frozen (M6 P4 spec 1).
func (u *GetModuleProgress) Execute(ctx context.Context, id uuid.UUID) (domain.Progress, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Progress{}, err
	}
	var out domain.Progress
	err = u.d.Reads.run(ctx, actor, domain.ActionModuleRead, named{module: &id}, func(ctx context.Context, h held) error {
		groups, err := u.d.Groups.StateGroups(ctx, h.projectID)
		if err != nil {
			return err
		}
		items, err := u.d.Stats.ModuleItems(ctx, h.projectID, h.module.ID, h.viewer(actor))
		if err != nil {
			return err
		}
		today, err := domain.Today(u.d.Clock.Now(), h.project.Timezone)
		if err != nil {
			return err
		}
		out, err = domain.ProgressOf(domain.ProgressFacts{Items: items, Groups: groups, Start: h.module.StartDate, End: h.module.TargetDate,
			Timezone: h.project.Timezone, Today: today})
		return err
	})
	return out, err
}
