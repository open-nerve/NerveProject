package bootstrap

import (
	"reflect"
	"testing"

	planningdomain "github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

// The checks of the progress rows of the cycle and the module tables (M6
// P4 spec R10): the seeds put no work item in any cycle or module, so each
// progress counts none.

// noWorkItems is the progress of no work item over burndown.
func noWorkItems(burndown *planningdomain.Burndown) planningdomain.Progress {
	return planningdomain.Progress{Assignees: []planningdomain.AssigneeDistribution{}, Labels: []planningdomain.LabelDistribution{},
		Burndown: burndown}
}

// answersAnEmptyCycleProgress: the column's own cycle (cycleOf), completed,
// its live progress of no work item, every day of its burndown past and at
// nothing left, and not frozen.
func answersAnEmptyCycleProgress(t *testing.T, c caller, _ seeded, answer string) {
	var got cycleProgressAnswer
	decodeAnswer(t, answer, &got)
	for _, m := range matrixCycles {
		if m.name != cycleOf(c) {
			continue
		}
		// Each seeded cycle has 14 days.
		want := cycleProgressAnswer{Status: "completed", Live: noWorkItems(burndown(t, m.start, m.end, 0, make([]int64, 14)...))}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s answers %s; want %s's %s without work items, not frozen", c, answer, projectOf(c), m.name)
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
