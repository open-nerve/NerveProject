package app

import (
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	rowStarted, rowCancelled = uuid.NewV7(), uuid.NewV7()
	// rowAnn and rowBen are two assignees, ann's id first.
	rowAnn, rowBen = func() (uuid.UUID, uuid.UUID) { ids := ascendingIDs(2); return ids[0], ids[1] }()
	// rowFrozen is a frozen progress whose counts are not any live ones.
	rowFrozen = domain.Frozen{Progress: domain.Progress{Counts: domain.Counts{Total: 9, Started: 4, Completed: 5},
		Assignees: []domain.AssigneeDistribution{}, Labels: []domain.LabelDistribution{}}, FrozenAt: time.Date(2026, 11, 2, 3, 4, 5, 6000, time.UTC)}
)

// newRowWorld is the module world in Shanghai, at 01:00 on the 8th of November 2026 there, whose project has a
// backlog, a started, a completed and a cancelled state, with the cycles' and the modules' pages it answers, in id
// order: the cycles lockCycle, current, with work items assigned to ann and ben; Quiet, a draft without any; Done,
// completed and frozen, with work items assigned to ann; and Old, archived and frozen, with one. The modules
// lockModule, with work items, Idle, without any, and Shelved, archived, with one. The issue module answers each
// cycle's and module's counts by state, the counts of a state summed by group, and each cycle's assignees, each once.
func newRowWorld(t *testing.T) (w *lockWorld, quiet, done, old, idle, shelved uuid.UUID) {
	w = newModuleWorld()
	w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, Timezone: "Asia/Shanghai", CycleView: true, ModuleView: true}
	w.now = time.Date(2026, 11, 7, 17, 0, 0, 0, time.UTC)
	w.groups = map[uuid.UUID]domain.StateGroup{progressBacklog: domain.GroupBacklog, rowStarted: domain.GroupStarted,
		progressDone: domain.GroupCompleted, rowCancelled: domain.GroupCancelled}
	archived := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	ids := ascendingIDs(5)
	quiet, done, old, idle, shelved = ids[0], ids[1], ids[2], ids[3], ids[4]
	c := w.cycles[lockCycle]
	c.Dates = domain.Dates{Start: date(t, "2026-11-08"), End: date(t, "2026-11-14")}
	w.cycles[lockCycle] = c
	w.cycles[quiet] = domain.Cycle{ID: quiet, ProjectID: lockProject}
	w.cycles[done] = domain.Cycle{ID: done, ProjectID: lockProject, Dates: domain.Dates{Start: date(t, "2026-10-01"), End: date(t, "2026-10-14")}}
	w.cycles[old] = domain.Cycle{ID: old, ProjectID: lockProject, Dates: domain.Dates{Start: date(t, "2026-09-01"), End: date(t, "2026-09-14")},
		ArchivedAt: &archived}
	w.modules[idle] = domain.Module{ID: idle, ProjectID: lockProject}
	w.modules[shelved] = domain.Module{ID: shelved, ProjectID: lockProject, ArchivedAt: &archived}
	w.counts = []domain.StateCount{
		{TargetID: lockCycle, StateID: progressBacklog, Issues: 2}, {TargetID: lockCycle, StateID: rowStarted, Issues: 1},
		{TargetID: lockCycle, StateID: rowCancelled, Issues: 1},
		{TargetID: done, StateID: progressDone, Issues: 3},
		{TargetID: old, StateID: rowStarted, Issues: 1},
		{TargetID: lockModule, StateID: progressDone, Issues: 2}, {TargetID: lockModule, StateID: rowStarted, Issues: 1},
		{TargetID: shelved, StateID: progressBacklog, Issues: 4},
	}
	w.assignees = []domain.CycleAssignee{{CycleID: lockCycle, AssigneeID: rowAnn}, {CycleID: lockCycle, AssigneeID: rowBen},
		{CycleID: done, AssigneeID: rowAnn}, {CycleID: old, AssigneeID: rowBen}}
	w.snapshots = map[uuid.UUID]*domain.Frozen{done: &rowFrozen, old: &rowFrozen}
	return w, quiet, done, old, idle, shelved
}

// rowFrozenCounts are rowFrozen's counts and moment, as a list row answers them.
var rowFrozenCounts = domain.FrozenCounts{Counts: rowFrozen.Counts, FrozenAt: rowFrozen.FrozenAt}

// A cycle's list row has the counts of its work items by their state's group, their assignees and its frozen counts,
// and a cycle without work items has every count 0 and no assignee (M6 P4 spec R9). The page's are read in the list's
// snapshot, after the cycles, each once for the whole page, whose ids, in the list's order, each read is asked for:
// the project's states' groups, the counts, the assignees and the frozen counts, then the clock, last, as before.
// Every account's work items are counted (no viewer). The archived catalog's rows are read alike.
func TestListCyclesRows(t *testing.T) {
	w, quiet, done, old, _, _ := newRowWorld(t)
	got, err := w.list(lockProject, false)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Begin", "Facts", "Authorize cycle.read", "ListCycles", "StateGroups", "CycleCounts", "CycleAssignees",
		"CycleSnapshotCounts", "Now", "End"}; !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
	page := []uuid.UUID{lockCycle, quiet, done}
	slices.SortFunc(page, uuid.UUID.Compare)
	if !reflect.DeepEqual(w.pages, [][]uuid.UUID{page, page, page}) || !reflect.DeepEqual(w.viewers, []*uuid.UUID{nil, nil}) {
		t.Fatalf("pages %v, viewers %v; want %v for every account's, three times", w.pages, w.viewers, page)
	}
	want := map[uuid.UUID]domain.CycleListItem{
		lockCycle: {Live: domain.Counts{Total: 4, Backlog: 2, Started: 1, Cancelled: 1}, AssigneeIDs: []uuid.UUID{rowAnn, rowBen}},
		quiet:     {},
		done:      {Live: domain.Counts{Total: 3, Completed: 3}, AssigneeIDs: []uuid.UUID{rowAnn}, Snapshot: &rowFrozenCounts},
	}
	status := map[uuid.UUID]domain.Status{lockCycle: domain.StatusCurrent, quiet: domain.StatusDraft, done: domain.StatusCompleted}
	if len(got) != len(page) {
		t.Fatalf("got %d rows, want %d", len(got), len(page))
	}
	for i, row := range got {
		if row.ID != page[i] || row.Status != status[row.ID] {
			t.Errorf("row %d is %s, %s; want %s, %s", i, row.ID, row.Status, page[i], status[page[i]])
		}
		if wr := want[row.ID]; !reflect.DeepEqual(row.Live, wr.Live) || !reflect.DeepEqual(row.AssigneeIDs, wr.AssigneeIDs) ||
			!reflect.DeepEqual(row.Snapshot, wr.Snapshot) {
			t.Errorf("row %s: %+v, %v, %+v; want %+v, %v, %+v", row.ID, row.Live, row.AssigneeIDs, row.Snapshot, wr.Live, wr.AssigneeIDs, wr.Snapshot)
		}
	}

	w.calls, w.pages, w.viewers = nil, nil, nil
	got, err = w.list(lockProject, true)
	if err != nil {
		t.Fatal(err)
	}
	if want := (domain.CycleListItem{ReadCycle: domain.ReadCycle{Cycle: w.cycles[old], Status: domain.StatusCompleted},
		Live: domain.Counts{Total: 1, Started: 1}, AssigneeIDs: []uuid.UUID{rowBen}, Snapshot: &rowFrozenCounts}); !reflect.DeepEqual(got, []domain.CycleListItem{want}) {
		t.Fatalf("the archived catalog\n%+v\nwant\n%+v", got, want)
	}
	if !reflect.DeepEqual(w.pages, [][]uuid.UUID{{old}, {old}, {old}}) {
		t.Fatalf("the archived catalog's pages %v", w.pages)
	}
}

// A page costs the same reads whatever its rows: one of each for a page of one cycle as for a page of three, and none
// for an empty page, whose list is answered, empty, without them (M6 P4 spec A5: no read per row).
func TestListCyclesReadsOncePerPage(t *testing.T) {
	w, _, _, _, _, _ := newRowWorld(t)
	lone, only, empty := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	w.facts[lone], w.facts[empty] = w.facts[lockProject], w.facts[lockProject]
	w.cycles[only] = domain.Cycle{ID: only, ProjectID: lone}
	reads := []string{"Begin", "Facts", "Authorize cycle.read", "ListCycles", "StateGroups", "CycleCounts", "CycleAssignees",
		"CycleSnapshotCounts", "Now", "End"}
	for _, tt := range []struct {
		name    string
		project uuid.UUID
		rows    int
	}{{"one cycle", lone, 1}, {"three cycles", lockProject, 3}} {
		w.calls = nil
		if got, err := w.list(tt.project, false); err != nil || len(got) != tt.rows || !slices.Equal(w.calls, reads) {
			t.Errorf("%s: %d rows, %v, calls\n%q\nwant %d rows, calls\n%q", tt.name, len(got), err, w.calls, tt.rows, reads)
		}
	}
	w.calls = nil
	got, err := w.list(empty, false)
	if want := []string{"Begin", "Facts", "Authorize cycle.read", "ListCycles", "End"}; err != nil || got == nil || len(got) != 0 ||
		!slices.Equal(w.calls, want) {
		t.Fatalf("an empty page: %v, %v, calls %q; want [], %q", got, err, w.calls, want)
	}
}

// A caller who sees only his own work items (OwnOnly) has his own counted alone, the issue module asked for his
// counts and his assignees, and is not shown the frozen counts, of every account's work items, which are not even
// read (M6 P4 spec R5, R10).
func TestListCyclesRowsOfAnOwnOnlyCaller(t *testing.T) {
	w, _, done, _, _, _ := newRowWorld(t)
	w.grant = shared.Grant{ProjectRole: shared.RoleGuest, OwnOnly: true}
	caller := shared.Actor{UserID: uuid.NewV7()}
	got, err := w.listAs(caller, lockProject, false)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Begin", "Facts", "Authorize cycle.read", "ListCycles", "StateGroups", "CycleCounts", "CycleAssignees", "Now",
		"End"}; !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
	if len(w.viewers) != 2 || w.viewers[0] == nil || *w.viewers[0] != caller.UserID || w.viewers[1] == nil || *w.viewers[1] != caller.UserID {
		t.Fatalf("viewers %v, want the caller %s twice", w.viewers, caller.UserID)
	}
	for _, row := range got {
		if row.Snapshot != nil {
			t.Errorf("%s answers the frozen counts %+v to an OwnOnly caller", row.ID, row.Snapshot)
		}
		if row.ID == done && row.Live.Completed != 3 {
			t.Errorf("Done counts %+v, want the issue module's", row.Live)
		}
	}
}

// A module's list row has the counts of its work items by their state's group, and a module without work items has
// every count 0 (M6 P4 spec R9): read in the list's snapshot after the modules, the project's states' groups and the
// counts, each once for the whole page, whose ids, in the list's order, it is asked for, of every account's work
// items. The archived catalog's rows are read alike; an empty page reads neither. An OwnOnly caller's are his own.
func TestListModulesRows(t *testing.T) {
	w, _, _, _, idle, shelved := newRowWorld(t)
	got, err := w.listModules(lockProject, false)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Begin", "Facts", "Authorize module.read", "ListModules", "StateGroups", "ModuleCounts", "End"}; !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
	page := []uuid.UUID{lockModule, idle}
	slices.SortFunc(page, uuid.UUID.Compare)
	if !reflect.DeepEqual(w.pages, [][]uuid.UUID{page}) || !reflect.DeepEqual(w.viewers, []*uuid.UUID{nil}) {
		t.Fatalf("pages %v, viewers %v; want %v once, for every account's", w.pages, w.viewers, page)
	}
	want := map[uuid.UUID]domain.Counts{lockModule: {Total: 3, Started: 1, Completed: 2}, idle: {}}
	if len(got) != len(page) {
		t.Fatalf("got %d rows, want %d", len(got), len(page))
	}
	for i, row := range got {
		if row.ID != page[i] || !reflect.DeepEqual(row.Module, w.modules[page[i]]) || row.Live != want[row.ID] {
			t.Errorf("row %d: %+v; want %s as read, counting %+v", i, row, page[i], want[page[i]])
		}
	}

	w.calls, w.pages = nil, nil
	if got, err := w.listModules(lockProject, true); err != nil || len(got) != 1 || got[0].ID != shelved ||
		got[0].Live != (domain.Counts{Total: 4, Backlog: 4}) || !reflect.DeepEqual(w.pages, [][]uuid.UUID{{shelved}}) {
		t.Fatalf("the archived catalog: %+v, %v, pages %v", got, err, w.pages)
	}

	empty := uuid.NewV7()
	w.facts[empty] = w.facts[lockProject]
	w.calls = nil
	if got, err := w.listModules(empty, false); err != nil || got == nil || len(got) != 0 ||
		!slices.Equal(w.calls, []string{"Begin", "Facts", "Authorize module.read", "ListModules", "End"}) {
		t.Fatalf("an empty page: %v, %v, calls %q", got, err, w.calls)
	}

	w, _, _, _, _, _ = newRowWorld(t)
	w.grant = shared.Grant{ProjectRole: shared.RoleGuest, OwnOnly: true}
	caller := shared.Actor{UserID: uuid.NewV7()}
	if _, err := w.listModulesAs(caller, lockProject, false); err != nil || len(w.viewers) != 1 || w.viewers[0] == nil || *w.viewers[0] != caller.UserID {
		t.Fatalf("an OwnOnly caller: %v, viewers %v; want the caller %s", err, w.viewers, caller.UserID)
	}
}

// What a list row cannot be counted from is the list's failure, a defect rather than numbers (M6 P4 spec R1): work
// items in a state not the project's.
func TestListRowsPassTheirFailures(t *testing.T) {
	w, _, _, _, _, _ := newRowWorld(t)
	delete(w.groups, rowStarted)
	if got, err := w.list(lockProject, false); err == nil {
		t.Errorf("cycles: %+v, want a failure", got)
	}
	if got, err := w.listModules(lockProject, false); err == nil {
		t.Errorf("modules: %+v, want a failure", got)
	}
}
