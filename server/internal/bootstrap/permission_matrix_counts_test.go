package bootstrap

import (
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The count rows of the cycle and the module tables (M6 P4 spec R1, R5,
// R10, A5, A6): what a list row or a progress counts is what its caller
// sees, not only whether he may read it. The seeds put no work item in any
// cycle or module, and listsTheCycles holds each project's list to them,
// so each cell's prepare puts countedItems, in its own copy, in its
// column's own cycle or module (counted): each row a GET, with write for
// its copy, as listIssues' archived rows.

// countedItem is a work item of countedItems: its name, its maker, its
// state, the day of its column's own cycle it was completed on (1, the
// first; 0, none), its assignees and labels; and whether it is archived,
// deleted, or has left its target (its membership deleted), any of which
// keeps it out of every number.
type countedItem struct {
	name                    string
	by                      caller
	state                   string
	doneOn                  int
	assignees               []caller
	labels                  []string
	archived, deleted, left bool
}

// countedItems are the work items a count cell's prepare puts in its
// column's own cycle or module (counted): the guest's (PG's account), the
// workspace admin's who is a guest, PM's and acme's admin's, in each group
// of states. PA and Feature are only on work items out of the numbers, and
// on the cancelled one, which the distributions leave out too, so that a
// number that took one in shows it; the workspace admin who is a guest is
// assigned the cancelled one alone, which a list row's assignees count.
var countedItems = []countedItem{
	{name: "Guest's backlog", by: callerGuest, state: "Backlog", assignees: []caller{callerProjectMember}, labels: []string{"Bug"}},
	{name: "Guest's done", by: callerGuest, state: "Done", doneOn: 3, assignees: []caller{callerProjectMember}, labels: []string{"Bug", "UI"}},
	{name: "Guest's archived", by: callerGuest, state: "Done", doneOn: 2, assignees: []caller{callerProjectAdmin}, labels: []string{"Feature"},
		archived: true},
	{name: "Admin guest's started", by: callerAdminGuest, state: "In Progress"},
	{name: "Admin guest's cancelled", by: callerAdminGuest, state: "Cancelled", assignees: []caller{callerAdminGuest}, labels: []string{"Feature"}},
	{name: "Admin guest's deleted", by: callerAdminGuest, state: "In Progress", assignees: []caller{callerProjectAdmin},
		labels: []string{"Feature"}, deleted: true},
	{name: "Member's review", by: callerProjectMember, state: "Review", assignees: []caller{callerProjectMember, callerRemoved},
		labels: []string{"UI"}},
	{name: "Member's left", by: callerProjectMember, state: "Todo", assignees: []caller{callerProjectAdmin}, labels: []string{"Feature"},
		left: true},
	{name: "Admin's todo", by: callerAdmin, state: "Todo", assignees: []caller{callerRemoved}},
	{name: "Admin's done", by: callerAdmin, state: "Done", doneOn: 10, labels: []string{"Bug"}},
}

// countTarget is what a count cell's prepare puts countedItems in: its
// column's own cycle (cycleOf) or module (moduleOf), by the table of its
// memberships and their column naming it; a cycle has ended and is frozen.
type countTarget struct {
	memberships, column string
	of                  func(s seeded, c caller) uuid.UUID
	frozen              bool
}

var (
	itsCycle = countTarget{memberships: "cycle_issues", column: "cycle_id", frozen: true,
		of: func(s seeded, c caller) uuid.UUID { return s.cycle(projectOf(c), cycleOf(c)) }}
	itsModule = countTarget{memberships: "module_issues", column: "module_id",
		of: func(s seeded, c caller) uuid.UUID { return s.module(projectOf(c), moduleOf(c)) }}
)

// matrixCycleDays are the first and the last day of the cycle name of
// matrixCycles.
func matrixCycleDays(t *testing.T, name string) (start, end string) {
	t.Helper()
	for _, m := range matrixCycles {
		if m.name == name {
			return m.start, m.end
		}
	}
	t.Fatalf("no cycle %s in matrixCycles", name)
	return "", ""
}

// counted is a count cell's prepare: in its copy at url, through SQL, each
// of countedItems in its column's project, numbered past the seeded ones
// and made by its maker; completed, when it was, at noon (UTC, the
// project's time zone) of its day of its column's own cycle (cycleOf; a
// module, which has no days, takes the same moments); with its assignees
// and labels; in target. A cycle, frozen without work items by the seed
// (freezeSeededCycles), is frozen again by the freeze job's use case, with
// them as they are when it ended (M6 P4 spec R5, R6): the deleted
// workspace's, which the job skips, stays unfrozen, and its cells are
// refused.
func counted(target countTarget) func(t *testing.T, c caller, s seeded, url string) {
	return func(t *testing.T, c caller, s seeded, url string) {
		t.Helper()
		pool := openPool(t, url)
		defer pool.Close()
		ctx, key := pgtest.Soon(t), projectOf(c)
		slug, _, _ := strings.Cut(key, "/")
		workspace, project, into := s.workspace(slug), s.project(key), target.of(s, c)
		exec := func(sql string, args ...any) {
			t.Helper()
			if _, err := pool.Exec(ctx, sql, args...); err != nil {
				t.Fatalf("preparing %s's counts: %v", c, err)
			}
		}
		first, _ := matrixCycleDays(t, cycleOf(c))
		noon := seedDay(t, first).Time().Add(12 * time.Hour)
		for n, item := range countedItems {
			id := uuid.NewV7()
			var done *time.Time
			if item.doneOn > 0 {
				at := noon.AddDate(0, 0, item.doneOn-1)
				done = &at
			}
			exec(`INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id, created_by_id, updated_by_id, completed_at,
				archived_at, deleted_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $7, $8, CASE WHEN $9::boolean THEN now() END,
				CASE WHEN $10::boolean THEN now() END)`, id, workspace, project, s.state(key, item.state), item.name, 100+n,
				s.account(item.by), done, item.archived, item.deleted)
			var assignees, labels []uuid.UUID
			for _, a := range item.assignees {
				assignees = append(assignees, s.account(a))
			}
			for _, l := range item.labels {
				labels = append(labels, s.label(key, l))
			}
			exec(`INSERT INTO issue_assignees (id, workspace_id, project_id, issue_id, assignee_id)
				SELECT gen_random_uuid(), $2, $3, $1, a FROM unnest($4::uuid[]) a`, id, workspace, project, assignees)
			exec(`INSERT INTO issue_labels (id, workspace_id, project_id, issue_id, label_id)
				SELECT gen_random_uuid(), $2, $3, $1, l FROM unnest($4::uuid[]) l`, id, workspace, project, labels)
			exec(`INSERT INTO `+target.memberships+` (id, workspace_id, project_id, `+target.column+`, issue_id, deleted_at)
				VALUES (gen_random_uuid(), $1, $2, $3, $4, CASE WHEN $5::boolean THEN now() END)`, workspace, project, into, id, item.left)
		}
		if target.frozen {
			exec("UPDATE cycles SET progress_snapshot = NULL WHERE id = $1", into)
			if _, err := freezeCyclesOn(pool, clock.System{}).Execute(ctx); err != nil {
				t.Fatalf("freezing %s's %s again: %v", c, cycleOf(c), err)
			}
		}
	}
}

// preparing is the prepare of each of steps, in order.
func preparing(steps ...func(t *testing.T, c caller, s seeded, url string)) func(t *testing.T, c caller, s seeded, url string) {
	return func(t *testing.T, c caller, s seeded, url string) {
		t.Helper()
		for _, step := range steps {
			step(t, c, s, url)
		}
	}
}

// guestsSeeAll makes the column's project show its guests every work item
// (guest_view_all_features) in the cell's copy, as its admin may: its
// guest, and the workspace admin who is one, no longer see only their own
// (Grant.OwnOnly). acme's private project shows them already.
func guestsSeeAll(t *testing.T, c caller, s seeded, url string) {
	t.Helper()
	pool := openPool(t, url)
	defer pool.Close()
	tag, err := pool.Exec(pgtest.Soon(t), "UPDATE projects SET guest_view_all_features = true WHERE id = $1", s.project(projectOf(c)))
	if err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("showing %s's guests every work item: %v, %d rows", projectOf(c), err, tag.RowsAffected())
	}
}

func countMatrixRows() []matrixRow {
	cycles, modules := counted(itsCycle), counted(itsModule)
	return []matrixRow{
		// As listCycles (M6 P4 spec R10): each row's numbers those of the
		// work items its caller sees, of countedItems in his column's own
		// cycle (counted), of none in the other; its frozen counts for none
		// who sees only his own (R5). The archive, the column's cycle
		// archived (cycleArchived), and a project that shows its guests
		// every work item (guestsSeeAll), whose guest and the workspace
		// admin who is one count them all.
		{op: "listCycles", variant: "counts", write: true, columns: cycleColumns, prepare: cycles,
			request: toProject(http.MethodGet, "/cycles", ""), cells: onTheCycles(cellOK, cellOK, cellOK, cellOK),
			check: listsTheCountedCycles(false, seesOwnOnly)},
		{op: "listCycles", variant: "counts in the archive", write: true, columns: cycleColumns, prepare: preparing(cycles, cycleArchived),
			request: toProject(http.MethodGet, "/cycles?archived=true", ""), cells: onTheCycles(cellOK, cellOK, cellOK, cellOK),
			check: listsTheCountedCycles(true, seesOwnOnly)},
		{op: "listCycles", variant: "counts, guests see all", write: true, columns: cycleColumns, prepare: preparing(guestsSeeAll, cycles),
			request: toProject(http.MethodGet, "/cycles", ""), cells: onTheCycles(cellOK, cellOK, cellOK, cellOK),
			check: listsTheCountedCycles(false, nobodyOwnOnly)},
		{op: "getCycleProgress", variant: "counts", write: true, columns: cycleColumns, prepare: cycles,
			request: toCycle(http.MethodGet, "/progress", ""), cells: onACycle(cellOK, cellOK, cellOK, cellOK, cellOK),
			check: readsTheCountedCycle(seesOwnOnly)},
		{op: "getCycleProgress", variant: "counts, guests see all", write: true, columns: cycleColumns, prepare: preparing(guestsSeeAll, cycles),
			request: toCycle(http.MethodGet, "/progress", ""), cells: onACycle(cellOK, cellOK, cellOK, cellOK, cellOK),
			check: readsTheCountedCycle(nobodyOwnOnly)},
		// As listCycles and getCycleProgress, of the column's own module: a
		// module is never frozen, and the seeded ones have no days.
		{op: "listModules", variant: "counts", write: true, columns: cycleColumns, prepare: modules,
			request: toProject(http.MethodGet, "/modules", ""), cells: onTheCycles(cellOK, cellOK, cellOK, cellOK),
			check: listsTheCountedModules(false, seesOwnOnly)},
		{op: "listModules", variant: "counts in the archive", write: true, columns: cycleColumns, prepare: preparing(modules, moduleArchived),
			request: toProject(http.MethodGet, "/modules?archived=true", ""), cells: onTheCycles(cellOK, cellOK, cellOK, cellOK),
			check: listsTheCountedModules(true, seesOwnOnly)},
		{op: "listModules", variant: "counts, guests see all", write: true, columns: cycleColumns, prepare: preparing(guestsSeeAll, modules),
			request: toProject(http.MethodGet, "/modules", ""), cells: onTheCycles(cellOK, cellOK, cellOK, cellOK),
			check: listsTheCountedModules(false, nobodyOwnOnly)},
		{op: "getModuleProgress", variant: "counts", write: true, columns: cycleColumns, prepare: modules,
			request: toModule(http.MethodGet, "/progress", ""), cells: moduleRows.inCycleColumns(cellOK, cellOK, cellOK, cellOK, cellOK),
			check: readsTheCountedModule(seesOwnOnly)},
		{op: "getModuleProgress", variant: "counts, guests see all", write: true, columns: cycleColumns, prepare: preparing(guestsSeeAll, modules),
			request: toModule(http.MethodGet, "/progress", ""), cells: moduleRows.inCycleColumns(cellOK, cellOK, cellOK, cellOK, cellOK),
			check: readsTheCountedModule(nobodyOwnOnly)},
	}
}
