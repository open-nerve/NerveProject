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
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The rows of the permission matrix of a project's cycles (M6 P1 spec R10):
// listing and creating them under the project of each column, and reading
// one, naming it by its id (/cycles/{cycle_id}) among its column's
// project's seeded cycles; and the cycles they rest on, matrixCycles, which prepareMatrix seeds in each project through the
// planning store (cycles) once the project store has turned the project's
// cycles on (cyclesOn).

var (
	cellCycleDatesOverlap = cell{http.StatusConflict, "planning.cycle_dates_overlap"}
	cellCycleNotFound     = cell{http.StatusNotFound, "planning.cycle_not_found"}
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

// matrixCycles are the cycles prepareMatrix seeds in each project of
// matrixProjects, by its workspace's admin: Sprint, from the 1st to the 14th
// of October 2026, at the sort order of a project's first cycle. Their
// writer in acme, as for the labels, is its admin, WA-'s account, a member
// of none of its projects, so no column that may write a cycle calls as
// him: a check that a write made its caller a cycle's owner cannot pass on
// the seed's.
var matrixCycles = []struct {
	name, start, end string
	sortOrder        float64
}{
	{"Sprint", "2026-10-01", "2026-10-14", 65535},
}

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
		{op: "getCycle", columns: cycleColumns, request: cycleRows.to(http.MethodGet, "", "Sprint", ""),
			cells: onACycle(cellOK, cellOK, cellOK, cellOK, cellOK), check: readsTheCycle},
		{op: "getCycle", variant: "archived", columns: archivedColumns, request: cycleRows.to(http.MethodGet, "", "Sprint", ""),
			cells: cycleRows.ofArchived(cellOK, cellForbidden), check: readsTheCycle},
		// The project's admins and members, not its guests, nor a caller who
		// sees only his own work items in it (M6 design 3.4); a workspace
		// admin or member who is no member of the project sees it and may
		// not.
		{op: "createCycle", write: true, columns: cycleColumns, request: toProject(http.MethodPost, "/cycles", newCycle),
			cells: onTheCycles(cellCreated, cellCreated, cellForbidden, cellForbidden), check: createsTheCycle},
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
	}
}

// listsTheCycles: the column's project's Sprint alone, as seeded.
func listsTheCycles(t *testing.T, c caller, s seeded, answer string) {
	var list struct{ Data []cycleAnswer }
	decodeAnswer(t, answer, &list)
	if len(list.Data) != 1 {
		t.Fatalf("%s lists %s; want %s's Sprint alone", c, answer, projectOf(c))
	}
	isTheSeededSprint(t, c, s, list.Data[0], answer)
}

// readsTheCycle: the column's project's Sprint, as seeded.
func readsTheCycle(t *testing.T, c caller, s seeded, answer string) {
	var got cycleAnswer
	decodeAnswer(t, answer, &got)
	isTheSeededSprint(t, c, s, got, answer)
}

// isTheSeededSprint checks that got, of answer, is the column's project's
// Sprint as matrixCycles seeds it, completed (its days are past), owned and
// written by the project's workspace's admin.
func isTheSeededSprint(t *testing.T, c caller, s seeded, got cycleAnswer, answer string) {
	t.Helper()
	slug, _, _ := strings.Cut(projectOf(c), "/")
	by := s.account(matrixAdmins[slug])
	if got.ID != s.cycle(projectOf(c), "Sprint") || got.ProjectID != s.project(projectOf(c)) || got.WorkspaceID != s.workspace(slug) ||
		got.Name != "Sprint" || got.StartDate == nil || *got.StartDate != "2026-10-01" || got.EndDate == nil || *got.EndDate != "2026-10-14" ||
		got.Status != "completed" || got.SortOrder != 65535 || got.OwnedByID == nil || *got.OwnedByID != by {
		t.Errorf("%s reads %s; want %s's Sprint as seeded", c, answer, projectOf(c))
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

// cyclesOn turns each project's cycles on through the project store, by
// its workspace's admin (M6 P1 spec R7: they are off by default).
func (s projectSeed) cyclesOn() {
	s.t.Helper()
	on := true
	for _, p := range matrixProjects {
		slug, _, _ := strings.Cut(p.key, "/")
		if err := s.store.UpdateProject(context.Background(), s.projects[p.key], projectdomain.ProjectPatch{CycleView: &on},
			s.ids[matrixAdmins[slug]], s.now); err != nil {
			s.t.Fatal(err)
		}
	}
}

// cycles stores each project's matrixCycles through the planning store, with
// the ids sd names, by its workspace's admin.
func (s projectSeed) cycles(sd seeded, store *planningpg.Store) {
	s.t.Helper()
	for _, p := range matrixProjects {
		slug, _, _ := strings.Cut(p.key, "/")
		by := s.ids[matrixAdmins[slug]]
		for _, c := range matrixCycles {
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
