package bootstrap

import (
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	planningdomain "github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The checks of the progress rows of the cycle and the module tables (M6
// P4 spec R10): the seeds put no work item in any cycle or module, so each
// progress counts none.

// noWorkItems is the progress of no work item over burndown.
func noWorkItems(burndown *planningdomain.Burndown) planningdomain.Progress {
	return planningdomain.Progress{Assignees: []planningdomain.AssigneeDistribution{}, Labels: []planningdomain.LabelDistribution{},
		Burndown: burndown}
}

// freezeSeededCycles freezes the seeded cycles, which have all ended, as
// the freeze job does (M6 P4 spec 2.5): every app the matrix starts runs
// the job as it starts, so the seed holds them frozen already, and no run
// on a cell's copy changes what its cell reads. The prepare's app's own
// run may have frozen some of them first.
func freezeSeededCycles(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := freezeCyclesOn(pool, clock.System{}).Execute(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
	var open int
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM cycles WHERE deleted_at IS NULL AND progress_snapshot IS NULL").Scan(&open); err != nil || open != 0 {
		t.Fatalf("%d seeded cycles are not frozen, %v; want none", open, err)
	}
}

// seesOwnOnly: the columns of the cycle table who see only their own work
// items in their project (Grant.OwnOnly), and so are not answered a frozen
// progress (M6 P4 spec R5): its guest, in acme's public project, which does
// not show its guests every work item, and acme's admin who is a guest
// there.
func seesOwnOnly(c caller) bool {
	return c == callerProjectGuest || c == callerAdminGuest
}

// answersAnEmptyCycleProgress: the column's own cycle (cycleOf), completed,
// its live progress of no work item, every day of its burndown past and at
// nothing left, and frozen so by the seed (freezeSeededCycles), at a
// moment, unless the column sees only its own work items.
func answersAnEmptyCycleProgress(t *testing.T, c caller, _ seeded, answer string) {
	var got cycleProgressAnswer
	decodeAnswer(t, answer, &got)
	for _, m := range matrixCycles {
		if m.name != cycleOf(c) {
			continue
		}
		// Each seeded cycle has 14 days.
		want := cycleProgressAnswer{Status: "completed", Live: noWorkItems(burndown(t, m.start, m.end, 0, make([]int64, 14)...))}
		if !seesOwnOnly(c) {
			if got.Snapshot == nil || got.Snapshot.FrozenAt.IsZero() {
				t.Errorf("%s answers %s; want %s's %s frozen at a moment", c, answer, projectOf(c), m.name)
				return
			}
			want.Snapshot = &planningdomain.Frozen{Progress: want.Live, FrozenAt: got.Snapshot.FrozenAt}
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s answers %s; want %s's %s without work items, frozen so unless it sees only its own", c, answer, projectOf(c), m.name)
		}
		return
	}
	t.Fatalf("no cycle %s in matrixCycles", cycleOf(c))
}

// answersAnEmptyModuleProgress: the progress of no work item, without a
// burndown: no seeded module has dates.
func answersAnEmptyModuleProgress(t *testing.T, c caller, _ seeded, answer string) {
	var got planningdomain.Progress
	decodeAnswer(t, answer, &got)
	if want := noWorkItems(nil); !reflect.DeepEqual(got, want) {
		t.Errorf("%s answers %s; want %s's %s without work items, without a burndown", c, answer, projectOf(c), moduleOf(c))
	}
}
