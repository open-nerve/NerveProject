package bootstrap

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"uuid"

	planningpg "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres"
	planningdomain "github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The rows of the permission matrix of a project's cycles (M6 P1 spec R10):
// listing and creating them under the project of each column, and reading,
// changing, archiving, unarchiving and deleting one, naming its column's
// own (cycleOf) by its id (/cycles/{cycle_id}) among its column's
// project's seeded cycles; and the cycles they rest on, matrixCycles, which prepareMatrix seeds in each project through the
// planning store (cycles) once the project store has turned the project's
// cycles on (planningOn).

var (
	cellCycleDatesOverlap = cell{http.StatusConflict, "planning.cycle_dates_overlap"}
	cellCycleNotFound     = cell{http.StatusNotFound, "planning.cycle_not_found"}
	cellCycleCompleted    = cell{http.StatusConflict, "planning.cycle_completed"}
)

// cycleRows are the cycles the rows name by id: a column that does not see
// its project answers planning.cycle_not_found.
var cycleRows = rowsByID{path: "cycles", notFound: cellCycleNotFound, find: seeded.cycle}

// cycleColumns are the columns of the cycle table (M6 P1 spec R10): PA, PM
// and the member who made a cycle; the project's guest; the workspace admin
// who is a guest of the project, who sees only his own work items there
// (Grant.OwnOnly) and so writes none of its cycles; then the project
// level's others, as projectColumns has them.
var cycleColumns = []caller{callerProjectAdmin, callerProjectMember, callerCreator, callerProjectGuest, callerAdminGuest,
	callerMemberAndAdmin, callerAdminOnly, callerMemberPublic, callerMemberPrivate, callerGuestOnly, callerBefore, callerNever,
	callerRemoved, callerDeleted}

// onTheCycles are the cells of a row of the cycle table on its project's
// cycles, none named (createCycle): a project-level row's on the project
// (ofProject), PM+WA as an admin of it; the member who made a cycle as PM,
// and the workspace admin who is a guest as adminGuest.
func onTheCycles(admin, member, guest, adminGuest cell) map[caller]cell {
	cells := ofProject(admin, member, guest, admin, cellForbidden, cellForbidden)
	cells[callerCreator] = member
	cells[callerAdminGuest] = adminGuest
	return cells
}

// onACycle are the cells of a row of the cycle table on one cycle, named by
// its id (cycleRows): a project-level row's on one (cycleRows.of), PM+WA as
// an admin of its project; the member who made a cycle as creator, and the
// workspace admin who is a guest as adminGuest.
func onACycle(admin, member, creator, guest, adminGuest cell) map[caller]cell {
	cells := cycleRows.of(admin, member, guest, admin, cellForbidden, cellForbidden)
	cells[callerCreator] = creator
	cells[callerAdminGuest] = adminGuest
	return cells
}

// matrixCycles are the cycles prepareMatrix seeds: Sprint, from the 1st to
// the 14th of October 2026, at the sort order of a project's first cycle, in
// each project of matrixProjects, by its workspace's admin; and Mine, from
// the 1st to the 14th of September, after Sprint, in acme's public project
// alone, by PM's account, the member who made it (callerCreator). Both are
// completed. Sprint's writer in acme, as for the labels, is its admin, WA-'s
// account, a member of none of its projects, so no column that may write a
// cycle calls as him: a check that a write made its caller a cycle's owner
// cannot pass on the seed's.
var matrixCycles = []struct {
	name, start, end string
	sortOrder        float64
	project          string // its project's key, "" for each project
	author           caller // its writer, "" for its workspace's admin
}{
	{"Sprint", "2026-10-01", "2026-10-14", 65535, "", ""},
	{"Mine", "2026-09-01", "2026-09-14", 75535, "acme/public", callerProjectMember},
}

// seededCycles are the cycles of matrixCycles seeded in the project key, in
// the order its list answers them (by sort order).
func seededCycles(key string) []string {
	var names []string
	for _, c := range matrixCycles {
		if c.project == "" || c.project == key {
			names = append(names, c.name)
		}
	}
	return names
}

// cycleOf is the name of the cycle of its project (projectOf) that a
// column's cells name: Mine, his own, for the member who made it; Sprint,
// made by the workspace's admin, for every other.
func cycleOf(c caller) string {
	if c == callerCreator {
		return "Mine"
	}
	return "Sprint"
}

// toCycle is the request of a row whose callers each send method, with
// body, to their column's own cycle (cycleOf), after its id.
func toCycle(method, after, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return cycleRows.to(method, after, cycleOf(c), body)(c, s)
	}
}

// cycleSettingsRows are the cycles' pages the rows name by their cycle's id
// (/me/cycles/{cycle_id}/preferences), as cycleRows.
var cycleSettingsRows = rowsByID{path: "me/cycles", notFound: cellCycleNotFound, find: seeded.cycle}

// toCyclePreferences is the request of a row whose callers each send
// method, with body, to their own display settings on their column's own
// cycle's page (cycleOf).
func toCyclePreferences(method, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return cycleSettingsRows.to(method, "/preferences", cycleOf(c), body)(c, s)
	}
}

// cycleSettings are the settings on a cycle's page the rows store: a
// filter of the modules', and display settings apart from the defaults.
var cycleSettings = pageSettingsJSON(`{"priority":["high"],"label_id":[],"module_id":["none"]}`, customDisplay, customProperties)

// newCycle is the body of the cycle the rows create: Sprint 2, from the day
// after Sprint's last, which it does not share.
const newCycle = `{"name":"Sprint 2","start_date":"2026-10-15","end_date":"2026-10-28"}`

// overlappingCycle shares Sprint's last day.
const overlappingCycle = `{"name":"Overlap","start_date":"2026-10-14","end_date":"2026-10-20"}`

func cycleMatrixRows() []matrixRow {
	return []matrixRow{
		// Every active member of the project, its guests and a caller who sees
		// only his own work items there too (M6 P1 spec R6); a workspace admin
		// or member who is no member of it sees it and may not.
		{op: "listCycles", columns: cycleColumns, request: toProject(http.MethodGet, "/cycles", ""),
			cells: onTheCycles(cellOK, cellOK, cellOK, cellOK), check: listsTheCycles},
		// An archived project's cycles are listed as any other's.
		{op: "listCycles", variant: "archived", columns: archivedColumns, request: toProject(http.MethodGet, "/cycles", ""),
			cells: ofArchived(cellOK, cellForbidden), check: listsTheCycles},
		{op: "getCycle", columns: cycleColumns, request: toCycle(http.MethodGet, "", ""),
			cells: onACycle(cellOK, cellOK, cellOK, cellOK, cellOK), check: readsTheCycle},
		{op: "getCycle", variant: "archived", columns: archivedColumns, request: toCycle(http.MethodGet, "", ""),
			cells: cycleRows.ofArchived(cellOK, cellForbidden), check: readsTheCycle},
		// As getCycle (M6 P4 spec R10): no seeded cycle has work items.
		{op: "getCycleProgress", columns: cycleColumns, request: toCycle(http.MethodGet, "/progress", ""),
			cells: onACycle(cellOK, cellOK, cellOK, cellOK, cellOK), check: answersAnEmptyCycleProgress},
		{op: "getCycleProgress", variant: "archived", columns: archivedColumns, request: toCycle(http.MethodGet, "/progress", ""),
			cells: cycleRows.ofArchived(cellOK, cellForbidden), check: answersAnEmptyCycleProgress},
		// The project's admins and members, not its guests, nor a caller who
		// sees only his own work items in it (M6 design 3.4); a workspace
		// admin or member who is no member of the project sees it and may
		// not.
		{op: "createCycle", write: true, columns: cycleColumns, request: toProject(http.MethodPost, "/cycles", newCycle),
			cells: onTheCycles(cellCreated, cellCreated, cellForbidden, cellForbidden), check: createsTheCycle},
		// The project's admins and members, as createCycle. Each cycle is
		// completed: its sort order is all it takes (M6 P1 spec R5).
		{op: "updateCycle", write: true, columns: cycleColumns, request: toCycle(http.MethodPatch, "", `{"sort_order":12345}`),
			cells: onACycle(cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: reordersTheCycle},
		// Its name: refused after the decision, so who may not change it
		// learns nothing of its status.
		{op: "updateCycle", variant: "a completed cycle's name", write: true, columns: cycleColumns,
			request: toCycle(http.MethodPatch, "", `{"name":"Renamed"}`),
			cells:   onACycle(cellCycleCompleted, cellCycleCompleted, cellCycleCompleted, cellForbidden, cellForbidden)},
		{op: "updateCycle", variant: "archived", write: true, columns: archivedColumns,
			request: toCycle(http.MethodPatch, "", `{"sort_order":12345}`), cells: cycleRows.ofArchived(cellProjectArchived, cellForbidden)},
		// As updateCycle: each cycle is completed, so it may be archived.
		{op: "archiveCycle", write: true, columns: cycleColumns, request: toCycle(http.MethodPost, "/archive", ""),
			cells: onACycle(cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: archivesTheCycle(true)},
		{op: "archiveCycle", variant: "archived", write: true, columns: archivedColumns, request: toCycle(http.MethodPost, "/archive", ""),
			cells: cycleRows.ofArchived(cellProjectArchived, cellForbidden)},
		// The column's cycle archived first in each cell's copy (cycleArchived).
		{op: "unarchiveCycle", write: true, columns: cycleColumns, prepare: cycleArchived,
			request: toCycle(http.MethodPost, "/unarchive", ""),
			cells:   onACycle(cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: archivesTheCycle(false)},
		{op: "unarchiveCycle", variant: "archived", write: true, columns: archivedColumns, prepare: cycleArchived,
			request: toCycle(http.MethodPost, "/unarchive", ""), cells: cycleRows.ofArchived(cellProjectArchived, cellForbidden)},
		// The project's admins, and the member who made the cycle
		// (AllowCreator): PM may not delete Sprint, the workspace admin's, and
		// deletes Mine, his own; whatever their status.
		{op: "deleteCycle", write: true, columns: cycleColumns, request: toCycle(http.MethodDelete, "", ""),
			cells: onACycle(cellNoContent, cellForbidden, cellNoContent, cellForbidden, cellForbidden)},
		{op: "deleteCycle", variant: "archived", write: true, columns: archivedColumns, request: toCycle(http.MethodDelete, "", ""),
			cells: cycleRows.ofArchived(cellProjectArchived, cellForbidden)},
		// Dates sharing a day with Sprint: refused under the lock, after the
		// decision, so who may not create cycles learns nothing of the
		// project's (M6 P1 spec R2).
		{op: "createCycle", variant: "overlapping dates", write: true, columns: cycleColumns,
			request: toProject(http.MethodPost, "/cycles", overlappingCycle),
			cells:   onTheCycles(cellCycleDatesOverlap, cellCycleDatesOverlap, cellForbidden, cellForbidden)},
		// An archived project takes no cycles (M6 design 3.1), after the
		// decision too.
		{op: "createCycle", variant: "archived", write: true, columns: archivedColumns, request: toProject(http.MethodPost, "/cycles", newCycle),
			cells: ofArchived(cellProjectArchived, cellForbidden)},
		// Every active member of the project, its guests and a caller who
		// sees only his own work items there, each his own display settings
		// on the cycle's page, the project or the cycle archived too (M6 P3b
		// spec 2.3, R6); a workspace admin or member who is no member of it
		// sees it and may not.
		{op: "getCyclePreferences", columns: cycleColumns, request: toCyclePreferences(http.MethodGet, ""),
			cells: onACycle(cellOK, cellOK, cellOK, cellOK, cellOK), check: answersPageSettings(pageDefaults)},
		{op: "getCyclePreferences", variant: "archived", columns: archivedColumns, request: toCyclePreferences(http.MethodGet, ""),
			cells: cycleRows.ofArchived(cellOK, cellForbidden), check: answersPageSettings(pageDefaults)},
		{op: "updateCyclePreferences", write: true, columns: cycleColumns, request: toCyclePreferences(http.MethodPatch, cycleSettings),
			cells: onACycle(cellOK, cellOK, cellOK, cellOK, cellOK), check: answersPageSettings(cycleSettings)},
		{op: "updateCyclePreferences", variant: "archived", write: true, columns: archivedColumns,
			request: toCyclePreferences(http.MethodPatch, cycleSettings), cells: cycleRows.ofArchived(cellOK, cellForbidden),
			check: answersPageSettings(cycleSettings)},
		{op: "updateCyclePreferences", variant: "an archived cycle", write: true, columns: cycleColumns, prepare: cycleArchived,
			request: toCyclePreferences(http.MethodPatch, cycleSettings), cells: onACycle(cellOK, cellOK, cellOK, cellOK, cellOK),
			check: answersPageSettings(cycleSettings)},
	}
}

// listsTheCycles: the column's project's seeded cycles, by sort order, each
// as seeded.
func listsTheCycles(t *testing.T, c caller, s seeded, answer string) {
	var list struct{ Data []cycleAnswer }
	decodeAnswer(t, answer, &list)
	want := seededCycles(projectOf(c))
	if len(list.Data) != len(want) {
		t.Fatalf("%s lists %s; want %s's %v", c, answer, projectOf(c), want)
	}
	for i, name := range want {
		isTheSeededCycle(t, c, s, name, list.Data[i], answer)
	}
}

// readsTheCycle: the column's own cycle (cycleOf), as seeded.
func readsTheCycle(t *testing.T, c caller, s seeded, answer string) {
	var got cycleAnswer
	decodeAnswer(t, answer, &got)
	isTheSeededCycle(t, c, s, cycleOf(c), got, answer)
}

// isTheSeededCycle checks that got, of answer, is the cycle name of the
// column's project as matrixCycles seeds it, completed (its days are past),
// owned by its writer.
func isTheSeededCycle(t *testing.T, c caller, s seeded, name string, got cycleAnswer, answer string) {
	t.Helper()
	slug, _, _ := strings.Cut(projectOf(c), "/")
	for _, m := range matrixCycles {
		if m.name != name {
			continue
		}
		by := s.account(matrixAdmins[slug])
		if m.author != "" {
			by = s.account(m.author)
		}
		if got.ID != s.cycle(projectOf(c), name) || got.ProjectID != s.project(projectOf(c)) || got.WorkspaceID != s.workspace(slug) ||
			got.Name != name || got.StartDate == nil || *got.StartDate != m.start || got.EndDate == nil || *got.EndDate != m.end ||
			got.Status != "completed" || got.SortOrder != m.sortOrder || got.OwnedByID == nil || *got.OwnedByID != by {
			t.Errorf("%s answers %s; want %s's %s as seeded", c, answer, projectOf(c), name)
		}
		return
	}
	t.Fatalf("no cycle %s in matrixCycles", name)
}

// reordersTheCycle: the column's own cycle at 12345, the rest as seeded,
// last changed by the column's account.
func reordersTheCycle(t *testing.T, c caller, s seeded, answer string) {
	var got cycleAnswer
	decodeAnswer(t, answer, &got)
	for _, m := range matrixCycles {
		if m.name == cycleOf(c) {
			seeded := got
			seeded.SortOrder = m.sortOrder
			isTheSeededCycle(t, c, s, m.name, seeded, answer)
		}
	}
	if got.SortOrder != 12345 || got.UpdatedByID == nil || *got.UpdatedByID != s.account(accountOf(c)) || !got.UpdatedAt.After(got.CreatedAt) {
		t.Errorf("%s reorders %s; want %s's %s at 12345, changed by %s", c, answer, projectOf(c), cycleOf(c), accountOf(c))
	}
}

// cycleArchived archives the column's own cycle in the cell's copy, at its
// last change.
func cycleArchived(t *testing.T, c caller, s seeded, url string) {
	t.Helper()
	pool := openPool(t, url)
	defer pool.Close()
	tag, err := pool.Exec(pgtest.Soon(t), "UPDATE cycles SET archived_at = updated_at WHERE id = $1", s.cycle(projectOf(c), cycleOf(c)))
	if err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("archiving %s's %s: %v, %d rows", c, cycleOf(c), err, tag.RowsAffected())
	}
}

// archivesTheCycle: the column's own cycle, archived or not as archived
// says, last changed by the column's account at the moment it is archived;
// the rest as seeded.
func archivesTheCycle(archived bool) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		var got cycleAnswer
		decodeAnswer(t, answer, &got)
		isTheSeededCycle(t, c, s, cycleOf(c), got, answer)
		ok := got.UpdatedByID != nil && *got.UpdatedByID == s.account(accountOf(c)) && got.UpdatedAt.After(got.CreatedAt)
		if archived {
			ok = ok && got.ArchivedAt != nil && got.ArchivedAt.Equal(got.UpdatedAt)
		} else {
			ok = ok && got.ArchivedAt == nil
		}
		if !ok {
			t.Errorf("%s answers %s; want %s's %s archived=%t, changed by %s", c, answer, projectOf(c), cycleOf(c), archived, accountOf(c))
		}
	}
}

// createsTheCycle: Sprint 2 in the column's project and workspace, as given,
// completed (its days are past), first among the project's cycles (10000
// below the seeded Sprint), owned, created and last changed by the column's
// account at one moment.
func createsTheCycle(t *testing.T, c caller, s seeded, answer string) {
	var got cycleAnswer
	decodeAnswer(t, answer, &got)
	slug, _, _ := strings.Cut(projectOf(c), "/")
	by := s.account(accountOf(c))
	if got.ProjectID != s.project(projectOf(c)) || got.WorkspaceID != s.workspace(slug) || got.Name != "Sprint 2" ||
		got.StartDate == nil || *got.StartDate != "2026-10-15" || got.EndDate == nil || *got.EndDate != "2026-10-28" ||
		got.Status != "completed" || got.SortOrder != 55535 || got.OwnedByID == nil || *got.OwnedByID != by ||
		got.CreatedByID == nil || *got.CreatedByID != by || got.UpdatedByID == nil || *got.UpdatedByID != by ||
		!got.CreatedAt.Equal(got.UpdatedAt) {
		t.Errorf("%s creates %s; want Sprint 2 in %s, completed, at 55535, owned, created and changed by %s at one moment", c, answer,
			projectOf(c), accountOf(c))
	}
}

// planningOn turns each project's cycles and modules on through the project
// store, by its workspace's admin (M6 P1 spec R7, P2 spec R7: they are off
// by default).
func (s projectSeed) planningOn() {
	s.t.Helper()
	on := true
	for _, p := range matrixProjects {
		slug, _, _ := strings.Cut(p.key, "/")
		if err := s.store.UpdateProject(context.Background(), s.projects[p.key], projectdomain.ProjectPatch{CycleView: &on, ModuleView: &on},
			s.ids[matrixAdmins[slug]], s.now); err != nil {
			s.t.Fatal(err)
		}
	}
}

// cycles stores each project's matrixCycles through the planning store, with
// the ids sd names, each by its writer.
func (s projectSeed) cycles(sd seeded, store *planningpg.Store) {
	s.t.Helper()
	for _, p := range matrixProjects {
		slug, _, _ := strings.Cut(p.key, "/")
		for _, c := range matrixCycles {
			if c.project != "" && c.project != p.key {
				continue
			}
			by := s.ids[matrixAdmins[slug]]
			if c.author != "" {
				by = s.ids[c.author]
			}
			start, end := seedDay(s.t, c.start), seedDay(s.t, c.end)
			if err := store.CreateCycle(context.Background(), planningdomain.Cycle{ID: sd.cycle(p.key, c.name), ProjectID: s.projects[p.key],
				WorkspaceID: s.workspaces[slug], Name: c.name, Dates: planningdomain.Dates{Start: &start, End: &end}, OwnedByID: &by,
				SortOrder: c.sortOrder, CreatedAt: s.now, UpdatedAt: s.now, CreatedByID: &by, UpdatedByID: &by}); err != nil {
				s.t.Fatal(err)
			}
		}
	}
}

// seedDay is the day day, YYYY-MM-DD.
func seedDay(t testing.TB, day string) shared.Date {
	t.Helper()
	d, err := shared.ParseDate(day)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// cycle is the id of the cycle name of the project key; one never seeded
// fails the test at once, as label's does.
func (s seeded) cycle(key, name string) uuid.UUID {
	id, ok := s.cycles[key+"|"+name]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no cycle %s of %s is seeded", name, key)
	}
	return id
}
