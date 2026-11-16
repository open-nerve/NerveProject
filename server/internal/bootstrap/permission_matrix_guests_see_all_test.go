package bootstrap

import (
	"maps"
	"net/http"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
)

// The rows of each write of the cycles, the modules and the modules' links
// on a project that shows its guests every work item (M6 P8 spec 2.7), as
// transferCycleIssues has its own (transferMatrixRows). In the seeded
// projects the guest is refused twice: by his role, which the rule of no
// planning write holds (*.write, *.delete), and as one who sees only his
// own work items (Grant.OwnOnly), which no planning write takes (decideOn);
// the workspace admin who is a guest of the project, by the second alone.
// So a write that decided its *.read action, which the guest's role holds,
// would answer each of their seeded cells as before. Here each cell's project shows its guests every work
// item first (guestsSeeAll): the guest is refused by his role alone, and
// the workspace admin who is a guest writes as an admin of the project.
// Each row is its operation's plain row (the first of its rows) on such a
// project: its prepare, if it has one (cycleArchived, moduleCompleted,
// moduleArchived), after guestsSeeAll, its request, its check, and its
// cells but the workspace admin's who is a guest, now its admins'. Each row
// has the whole cycle table: the matrix holds a row to a whole table
// (matrixViolations), not to the two columns it is about.
//
// No row here for:
//   - updateCyclePreferences and updateModulePreferences
//     (noGuestsSeeAllRow): they decide the *.preferences actions, which the
//     guest holds, OwnOnly or not; their rows answer him 200 already;
//   - transferCycleIssues: its row of a project that shows its guests every
//     work item is beside its others (transferMatrixRows);
//   - a module's members: they have no operation of their own, and are
//     written as createModule's and updateModule's member_ids, whose rows
//     are here;
//   - a work item's cycle and modules (M6 P3a): createIssue's and
//     updateIssue's cycle_id and module_ids, which decide the issue
//     module's actions, no *.write of the planning module's.

// guestsSeeAllVariant names a row of a project that shows its guests every
// work item, as transferMatrixRows names its own.
const guestsSeeAllVariant = "guests see all"

// noGuestsSeeAllRow are the planning module's writes that have no row of a
// project that shows its guests every work item, each for its reason.
var noGuestsSeeAllRow = map[string]string{
	"updateCyclePreferences":  "cycle.preferences, which the guest holds, OwnOnly or not: its rows answer him 200",
	"updateModulePreferences": "module.preferences, as updateCyclePreferences",
}

func guestsSeeAllMatrixRows() []matrixRow {
	const v = guestsSeeAllVariant
	return []matrixRow{
		{op: "createCycle", variant: v, write: true, columns: cycleColumns, prepare: guestsSeeAll,
			request: toProject(http.MethodPost, "/cycles", newCycle),
			cells:   onTheCycles(cellCreated, cellCreated, cellForbidden, cellCreated), check: createsTheCycle},
		{op: "updateCycle", variant: v, write: true, columns: cycleColumns, prepare: guestsSeeAll,
			request: toCycle(http.MethodPatch, "", `{"sort_order":12345}`),
			cells:   onACycle(cellOK, cellOK, cellOK, cellForbidden, cellOK), check: reordersTheCycle},
		// cycle.delete: the workspace's admin, as the project's admins.
		{op: "deleteCycle", variant: v, write: true, columns: cycleColumns, prepare: guestsSeeAll,
			request: toCycle(http.MethodDelete, "", ""),
			cells:   onACycle(cellNoContent, cellForbidden, cellNoContent, cellForbidden, cellNoContent)},
		{op: "archiveCycle", variant: v, write: true, columns: cycleColumns, prepare: guestsSeeAll,
			request: toCycle(http.MethodPost, "/archive", ""),
			cells:   onACycle(cellOK, cellOK, cellOK, cellForbidden, cellOK), check: archivesTheCycle(true)},
		{op: "unarchiveCycle", variant: v, write: true, columns: cycleColumns, prepare: preparing(guestsSeeAll, cycleArchived),
			request: toCycle(http.MethodPost, "/unarchive", ""),
			cells:   onACycle(cellOK, cellOK, cellOK, cellForbidden, cellOK), check: archivesTheCycle(false)},
		{op: "createModule", variant: v, write: true, columns: cycleColumns, prepare: guestsSeeAll,
			request: toProject(http.MethodPost, "/modules", newModule),
			cells:   onTheCycles(cellCreated, cellCreated, cellForbidden, cellCreated), check: createsTheModule},
		{op: "updateModule", variant: v, write: true, columns: cycleColumns, prepare: guestsSeeAll,
			request: toModule(http.MethodPatch, "", `{"sort_order":12345}`),
			cells:   moduleRows.inCycleColumns(cellOK, cellOK, cellOK, cellForbidden, cellOK), check: reordersTheModule},
		// module.delete: as cycle.delete.
		{op: "deleteModule", variant: v, write: true, columns: cycleColumns, prepare: guestsSeeAll,
			request: toModule(http.MethodDelete, "", ""),
			cells:   moduleRows.inCycleColumns(cellNoContent, cellForbidden, cellNoContent, cellForbidden, cellNoContent)},
		{op: "archiveModule", variant: v, write: true, columns: cycleColumns, prepare: preparing(guestsSeeAll, moduleCompleted),
			request: toModule(http.MethodPost, "/archive", ""),
			cells:   moduleRows.inCycleColumns(cellOK, cellOK, cellOK, cellForbidden, cellOK), check: archivesTheModule(true)},
		{op: "unarchiveModule", variant: v, write: true, columns: cycleColumns, prepare: preparing(guestsSeeAll, moduleArchived),
			request: toModule(http.MethodPost, "/unarchive", ""),
			cells:   moduleRows.inCycleColumns(cellOK, cellOK, cellOK, cellForbidden, cellOK), check: archivesTheModule(false)},
		// A link's write is its module's (module.write).
		{op: "createModuleLink", variant: v, write: true, columns: cycleColumns, prepare: guestsSeeAll,
			request: toModule(http.MethodPost, "/links", `{"url":"http://169.254.169.254/matrix-create"}`),
			cells:   moduleRows.inCycleColumns(cellCreated, cellCreated, cellCreated, cellForbidden, cellCreated), check: createsTheModuleLink},
		{op: "updateModuleLink", variant: v, write: true, columns: cycleColumns, prepare: guestsSeeAll,
			request: toModuleLink(http.MethodPatch, `{"title":"Edited"}`),
			cells:   moduleLinkRows.inCycleColumns(cellOK, cellOK, cellOK, cellForbidden, cellOK), check: editsTheModuleLink},
		{op: "deleteModuleLink", variant: v, write: true, columns: cycleColumns, prepare: guestsSeeAll,
			request: toModuleLink(http.MethodDelete, ""),
			cells:   moduleLinkRows.inCycleColumns(cellNoContent, cellNoContent, cellNoContent, cellForbidden, cellNoContent)},
	}
}

// Each write of the planning module (an operation tagged planning that is
// not a GET) has a row of a project that shows its guests every work item,
// but those noGuestsSeeAllRow lists, and each it lists is such a write
// without one: a write the planning module adds is held to it too.
func TestEachPlanningWriteHasAGuestsSeeAllMatrixRow(t *testing.T) {
	rowed := map[string]bool{}
	for _, r := range matrixRows() {
		if r.variant == guestsSeeAllVariant {
			rowed[r.op] = true
		}
	}
	writes := map[string]bool{}
	for _, op := range apitest.Load(t).Operations() {
		if slices.Contains(op.Tags, "planning") && op.Method != http.MethodGet {
			writes[op.ID] = true
		}
	}
	for _, id := range slices.Sorted(maps.Keys(writes)) {
		_, listed := noGuestsSeeAllRow[id]
		switch {
		case listed && rowed[id]:
			t.Errorf("the planning write %s has a row of a project that shows its guests every work item, and is listed as without one", id)
		case !listed && !rowed[id]:
			t.Errorf("the planning write %s has no row of a project that shows its guests every work item", id)
		}
	}
	for _, id := range slices.Sorted(maps.Keys(noGuestsSeeAllRow)) {
		if !writes[id] {
			t.Errorf("%s, listed as without a row of a project that shows its guests every work item, is no planning write", id)
		}
	}
}
