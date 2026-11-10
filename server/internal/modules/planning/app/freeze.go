package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// freezing is what a cycle's freeze is computed on, by the freeze job and
// by a transfer alike, under its project's row lock (M6 P4 spec R5, 2.5,
// 2.6): the project, its time zone and its states' groups, read under the
// lock, and the clock's one reading after the lock, the write's own, with
// the project's day at it.
type freezing struct {
	project  uuid.UUID
	timezone string
	groups   map[uuid.UUID]domain.StateGroup
	now      time.Time
	today    shared.Date
}

// freeze stores the frozen progress of the project's cycle c unless it has
// one, and answers whether it did: the progress of every work item of c,
// whoever reads it later (no viewer: an OwnOnly reader is never answered
// it, R5), from c's start to its end, on f's day, taken at f's reading.
func (f freezing) freeze(ctx context.Context, stats IssueStats, cycles CycleFreezer, c domain.Cycle) (bool, error) {
	items, err := stats.CycleItems(ctx, f.project, c.ID, nil)
	if err != nil {
		return false, err
	}
	p, err := domain.ProgressOf(domain.ProgressFacts{Items: items, Groups: f.groups, Start: c.Dates.Start, End: c.Dates.End,
		Timezone: f.timezone, Today: f.today})
	if err != nil {
		return false, err
	}
	return cycles.FreezeCycle(ctx, c.ID, domain.Frozen{Progress: p, FrozenAt: f.now})
}
