package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ProjectLocker takes a project's row FOR NO KEY UPDATE, the first lock of
// every write on its work items, cycles and modules, in the transaction ctx
// carries, and reads its facts under it; found is false for a project that
// is not there or is deleted, also one deleted while the lock waited.
type ProjectLocker interface {
	Lock(ctx context.Context, id uuid.UUID) (f ProjectFacts, found bool, err error)
}

// FreezeStore is the cycles' store as the freeze job uses it (M6 P4 spec
// 2.5): without a lock, the projects with an undeleted cycle not frozen
// that ends on utcToday or before it; under a project's lock, its undeleted
// cycles not frozen, archived ones too, that ended before today, the
// project's day; and a cycle's freeze (CycleFreezer).
type FreezeStore interface {
	FreezeDueProjects(ctx context.Context, utcToday shared.Date) ([]uuid.UUID, error)
	CyclesToFreeze(ctx context.Context, project uuid.UUID, today shared.Date) ([]domain.Cycle, error)
	CycleFreezer
}

// CycleFreezer stores f as the frozen progress of the undeleted cycle id
// when it has none, in the transaction ctx carries, and answers whether it
// did: the first freeze stays (M6 P4 spec R5).
type CycleFreezer interface {
	FreezeCycle(ctx context.Context, id uuid.UUID, f domain.Frozen) (bool, error)
}

// FreezeCyclesDeps are FreezeCycles' ports. Tx are the job's own
// transactions: each lock wait bounded, no write gate, and a lock failure
// kept as PostgreSQL's error for River to retry (as the automatic archival's).
type FreezeCyclesDeps struct {
	Projects ProjectLocker
	Groups   StateGroupReader
	Cycles   FreezeStore
	Stats    IssueStats
	Tx       shared.TxManager
	Clock    Clock
}

// FreezeCycles is the periodic job planning.freeze_cycles (M6 P4 spec 2.5,
// R6), as the automatic archival runs (issue/app/auto_archive.go): its
// candidates without a lock, then the authoritative decision of each under
// its project's lock, on the clock read after the lock, one committed
// project at a time. A freeze is a record of a cycle's end, not anyone's
// write: an archived project, a project whose cycles are off and an
// archived cycle are frozen as any other, and the cycle's last change stays
// its own.
type FreezeCycles struct{ d FreezeCyclesDeps }

// NewFreezeCycles returns the job's use case over d.
func NewFreezeCycles(d FreezeCyclesDeps) *FreezeCycles { return &FreezeCycles{d: d} }

// Execute freezes the progress of the cycles that have ended, and answers
// how many it froze, committed. The candidates are the projects with a
// cycle that ends on the UTC day of the clock's reading or before it: in
// any time zone, today is at most the day after that one, so they hold
// every project with a cycle that has ended. That reading only bounds them;
// no freeze is taken at it. Then each candidate in a transaction of its own
// (freezeProject). A failure ends the run with its error chain whole, a lock
// timeout PostgreSQL's, for River to retry, and the number frozen before it:
// the projects before it stay committed, and the next run goes on with it.
func (u *FreezeCycles) Execute(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	projects, err := u.d.Cycles.FreezeDueProjects(ctx, shared.DateOf(u.d.Clock.Now().UTC()))
	if err != nil {
		return 0, err
	}
	total := 0
	for _, id := range projects {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		n := 0
		err := u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
			var err error
			n, err = u.freezeProject(ctx, id)
			return err
		})
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

// freezeProject freezes, in the transaction ctx carries, the cycles of the
// project id that have ended, and answers how many it froze. First the
// project's lock, which every write on its cycles and work items takes
// first: a project deleted is passed over. Then the clock, read once,
// after the lock; the cycles not frozen that ended before the project's day
// at that reading, its states' groups, and each one's freeze, taken at it.
func (u *FreezeCycles) freezeProject(ctx context.Context, id uuid.UUID) (int, error) {
	p, found, err := u.d.Projects.Lock(ctx, id)
	if err != nil || !found {
		return 0, err
	}
	now := u.d.Clock.Now()
	today, err := domain.Today(now, p.Timezone)
	if err != nil {
		return 0, err
	}
	cycles, err := u.d.Cycles.CyclesToFreeze(ctx, id, today)
	if err != nil || len(cycles) == 0 {
		return 0, err
	}
	groups, err := u.d.Groups.StateGroups(ctx, id)
	if err != nil {
		return 0, err
	}
	at := freezing{project: id, timezone: p.Timezone, groups: groups, now: now, today: today}
	n := 0
	for _, c := range cycles {
		wrote, err := at.freeze(ctx, u.d.Stats, u.d.Cycles, c)
		if err != nil {
			return 0, err
		}
		if wrote {
			n++
		}
	}
	return n, nil
}
