package bootstrap

import (
	"reflect"
	"slices"
	"testing"
	"uuid"

	planningdomain "github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

// The checks of the count rows of the cycle and the module tables
// (permission_matrix_counts_test.go): the numbers of countedItems each
// column reads, written out, and how a list row or a progress answers them.

// countedNumbers are the numbers of countedItems a column reads: the
// counts; the distributions by assignee and by label, each row of none
// apart, nil for no such row; the burndown's scope and what is left on
// each of its 14 days; and a list row's assignees.
type countedNumbers struct {
	counts              planningdomain.Counts
	assignees           map[caller]planningdomain.Tally
	labels              map[string]planningdomain.Tally
	noAssignee, noLabel *planningdomain.Tally
	scope               int64
	left                []int64
	assigneeIDs         []caller
}

// tallyOf is a distribution's row of none: completed and pending work
// items.
func tallyOf(completed, pending int64) *planningdomain.Tally {
	t := tally(completed, pending)
	return &t
}

// countedBy are the numbers of countedItems by whose of them a column sees
// (whose): every one's, ""; or its account's own, for a column that sees
// only his own work items (Grant.OwnOnly): the guest's, PG's account, and
// the workspace admin's who is a guest. Each written out, not computed: a
// computation would be the code under test again.
var countedBy = map[caller]countedNumbers{
	"": {counts: planningdomain.Counts{Total: 7, Backlog: 1, Unstarted: 1, Started: 2, Completed: 2, Cancelled: 1},
		assignees: map[caller]planningdomain.Tally{callerProjectMember: tally(1, 2), callerRemoved: tally(0, 2)}, noAssignee: tallyOf(1, 1),
		labels: map[string]planningdomain.Tally{"Bug": tally(2, 1), "UI": tally(1, 1)}, noLabel: tallyOf(0, 2),
		scope: 6, left: []int64{6, 6, 5, 5, 5, 5, 5, 5, 5, 4, 4, 4, 4, 4},
		assigneeIDs: []caller{callerProjectMember, callerRemoved, callerAdminGuest}},
	callerGuest: {counts: planningdomain.Counts{Total: 2, Backlog: 1, Completed: 1},
		assignees: map[caller]planningdomain.Tally{callerProjectMember: tally(1, 1)},
		labels:    map[string]planningdomain.Tally{"Bug": tally(1, 1), "UI": tally(1, 0)},
		scope:     2, left: []int64{2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, assigneeIDs: []caller{callerProjectMember}},
	callerAdminGuest: {counts: planningdomain.Counts{Total: 2, Started: 1, Cancelled: 1}, noAssignee: tallyOf(0, 1), noLabel: tallyOf(0, 1),
		scope: 1, left: []int64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}, assigneeIDs: []caller{callerAdminGuest}},
}

// nobodyOwnOnly is a table's columns who see only their own work items
// once its project shows its guests every one (guestsSeeAll): none.
func nobodyOwnOnly(caller) bool { return false }

// countedFor are the numbers of countedItems the column c reads, its
// account's own when ownOnly(c), every one's else.
func countedFor(t *testing.T, c caller, ownOnly func(caller) bool) countedNumbers {
	t.Helper()
	whose := caller("")
	if ownOnly(c) {
		whose = accountOf(c)
	}
	n, ok := countedBy[whose]
	if !ok {
		t.Fatalf("no numbers of countedItems for %s, who sees %q's", c, whose)
	}
	return n
}

// progress is the progress of n in the project key, over the burndown b.
func (n countedNumbers) progress(s seeded, key string, b *planningdomain.Burndown) planningdomain.Progress {
	assignees, labels := map[uuid.UUID]planningdomain.Tally{}, map[uuid.UUID]planningdomain.Tally{}
	for who, row := range n.assignees {
		assignees[s.account(who)] = row
	}
	for name, row := range n.labels {
		labels[s.label(key, name)] = row
	}
	return planningdomain.Progress{Counts: n.counts, Assignees: byAssignee(assignees, n.noAssignee), Labels: byLabel(labels, n.noLabel),
		Burndown: b}
}

// assigneesOf are n's list row's assignees, in id order.
func (n countedNumbers) assigneesOf(s seeded) []uuid.UUID {
	ids := []uuid.UUID{}
	for _, who := range n.assigneeIDs {
		ids = append(ids, s.account(who))
	}
	slices.SortFunc(ids, uuid.UUID.Compare)
	return ids
}

// listsTheCountedCycles: the column's project's seeded cycles, or its own
// alone, archived, in the archive, each as seeded; its own cycle's numbers
// those of countedItems it sees (countedFor), the other's none; each
// frozen, at a moment, at its live counts, unless ownOnly(c).
func listsTheCountedCycles(archive bool, ownOnly func(caller) bool) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		rows, names := listIn[cycleRowAnswer](t, answer), seededCycles(projectOf(c))
		if archive {
			names = []string{cycleOf(c)}
		}
		if len(rows) != len(names) {
			t.Fatalf("%s lists %s; want %s's %v", c, answer, projectOf(c), names)
		}
		for i, name := range names {
			r := rows[i]
			isTheSeededCycle(t, c, s, name, r.cycleAnswer, answer)
			var n countedNumbers
			if name == cycleOf(c) {
				n = countedFor(t, c, ownOnly)
			}
			frozen := r.Snapshot == nil
			if !ownOnly(c) {
				frozen = r.Snapshot != nil && !r.Snapshot.FrozenAt.IsZero() && r.Snapshot.Counts == n.counts
			}
			if (r.ArchivedAt != nil) != archive || r.Counts != n.counts || !slices.Equal(r.AssigneeIDs, n.assigneesOf(s)) || !frozen {
				t.Errorf("%s lists %s; want %s's %s archived=%t, of %+v assigned to %v, frozen so unless it sees only its own", c, answer,
					projectOf(c), name, archive, n.counts, n.assigneeIDs)
			}
		}
	}
}

// readsTheCountedCycle: the column's own cycle, completed, its live
// progress that of countedItems it sees (countedFor), over its 14 days,
// all past; frozen so, at a moment, unless ownOnly(c), who sees fewer.
func readsTheCountedCycle(ownOnly func(caller) bool) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		var got cycleProgressAnswer
		decodeAnswer(t, answer, &got)
		n := countedFor(t, c, ownOnly)
		start, end := matrixCycleDays(t, cycleOf(c))
		want := cycleProgressAnswer{Status: "completed", Live: n.progress(s, projectOf(c), burndown(t, start, end, n.scope, n.left...))}
		if !ownOnly(c) {
			if got.Snapshot == nil || got.Snapshot.FrozenAt.IsZero() {
				t.Errorf("%s answers %s; want %s's %s frozen at a moment", c, answer, projectOf(c), cycleOf(c))
				return
			}
			want.Snapshot = &planningdomain.Frozen{Progress: want.Live, FrozenAt: got.Snapshot.FrozenAt}
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s answers %s; want %s's %s of countedItems it sees, %+v", c, answer, projectOf(c), cycleOf(c), want)
		}
	}
}

// listsTheCountedModules: the column's project's seeded modules, or its
// own alone, archived, in the archive, each as seeded; its own module's
// counts those of countedItems it sees (countedFor), the other's none.
func listsTheCountedModules(archive bool, ownOnly func(caller) bool) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		rows := listIn[moduleRowAnswer](t, answer)
		var names []string
		for _, m := range matrixModules {
			if m.project == "" || m.project == projectOf(c) {
				names = append(names, m.name)
			}
		}
		if archive {
			names = []string{moduleOf(c)}
		}
		if len(rows) != len(names) {
			t.Fatalf("%s lists %s; want %s's %v", c, answer, projectOf(c), names)
		}
		for i, name := range names {
			r := rows[i]
			seeded := r.moduleAnswer
			seeded.ArchivedAt = nil
			isTheSeededModule(t, c, s, name, seeded, answer)
			var n countedNumbers
			if name == moduleOf(c) {
				n = countedFor(t, c, ownOnly)
			}
			if (r.ArchivedAt != nil) != archive || r.Counts != n.counts {
				t.Errorf("%s lists %s; want %s's %s archived=%t, of %+v", c, answer, projectOf(c), name, archive, n.counts)
			}
		}
	}
}

// readsTheCountedModule: the progress of countedItems the column sees in
// its own module (countedFor), without a burndown: it has no days.
func readsTheCountedModule(ownOnly func(caller) bool) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		var got planningdomain.Progress
		decodeAnswer(t, answer, &got)
		if want := countedFor(t, c, ownOnly).progress(s, projectOf(c), nil); !reflect.DeepEqual(got, want) {
			t.Errorf("%s answers %s; want %s's %s of countedItems it sees, %+v", c, answer, projectOf(c), moduleOf(c), want)
		}
	}
}
