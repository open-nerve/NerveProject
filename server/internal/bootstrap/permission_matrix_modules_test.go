package bootstrap

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"uuid"

	planningpg "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres"
	planningdomain "github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The rows of the permission matrix of a project's modules (M6 P2 spec
// R10): listing and creating them under the project of each column, and
// reading, changing, archiving, unarchiving and deleting one, naming its
// column's own (moduleOf) by its id (/modules/{module_id}); and the modules
// they rest on, matrixModules, which prepareMatrix seeds in each project
// through the planning store (modules) once the project store has turned
// the project's modules on (planningOn). The module table's columns and
// cells are the cycle table's (cycleColumns, onTheCycles): the member who
// made a module (callerCreator) is PM's account, as for the cycles.

var (
	cellModuleNameTaken = cell{http.StatusConflict, "planning.module_name_taken"}
	cellModuleNotFound  = cell{http.StatusNotFound, "planning.module_not_found"}
	cellModuleNotClosed = cell{http.StatusConflict, "planning.module_not_closed"}
)

// moduleRows are the modules the rows name by id: a column that does not
// see its project answers planning.module_not_found.
var moduleRows = rowsByID{path: "modules", notFound: cellModuleNotFound, find: seeded.module}

// inCycleColumns are the cells of a row of the cycle table's columns
// (cycleColumns) on one row of k, named by its id (k.of), PM+WA as an admin
// of its project; the member who made the module as creator, and the
// workspace admin who is a guest as adminGuest.
func (k rowsByID) inCycleColumns(admin, member, creator, guest, adminGuest cell) map[caller]cell {
	cells := k.of(admin, member, guest, admin, cellForbidden, cellForbidden)
	cells[callerCreator] = creator
	cells[callerAdminGuest] = adminGuest
	return cells
}

// matrixModules are the modules prepareMatrix seeds, both planned: Release,
// at the sort order of a project's first module, in each project of
// matrixProjects, by its workspace's admin; and Mine, after Release, in
// acme's public project alone, by PM's account, the member who made it
// (callerCreator). Release's writer in acme is its admin, WA-'s account, a
// member of none of its projects, as for the cycles.
var matrixModules = []struct {
	name      string
	sortOrder float64
	project   string // its project's key, "" for each project
	author    caller // its writer, "" for its workspace's admin
}{
	{"Release", 65535, "", ""},
	{"Mine", 75535, "acme/public", callerProjectMember},
}

// moduleOf is the name of the module of its project (projectOf) that a
// column's cells name: Mine, his own, for the member who made it; Release,
// made by the workspace's admin, for every other.
func moduleOf(c caller) string {
	if c == callerCreator {
		return "Mine"
	}
	return "Release"
}

// toModule is the request of a row whose callers each send method, with
// body, to their column's own module (moduleOf), after its id.
func toModule(method, after, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return moduleRows.to(method, after, moduleOf(c), body)(c, s)
	}
}

// moduleSettingsRows are the modules' pages the rows name by their module's
// id (/me/modules/{module_id}/preferences), as moduleRows.
var moduleSettingsRows = rowsByID{path: "me/modules", notFound: cellModuleNotFound, find: seeded.module}

// toModulePreferences is toCyclePreferences of their column's own module's
// page (moduleOf).
func toModulePreferences(method, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return moduleSettingsRows.to(method, "/preferences", moduleOf(c), body)(c, s)
	}
}

// moduleSettings are cycleSettings of a module's page: a filter of the
// cycles'.
var moduleSettings = pageSettingsJSON(`{"priority":["high"],"label_id":[],"cycle_id":["none"]}`, customDisplay, customProperties)

// newModule is the body of the module the rows create: Release 2, its name
// alone.
const newModule = `{"name":"Release 2"}`

func moduleMatrixRows() []matrixRow {
	return []matrixRow{
		// Every active member of the project, its guests and a caller who sees
		// only his own work items there too (M6 P2 spec R6); a workspace admin
		// or member who is no member of it sees it and may not. An archived
		// project's modules, and an archived module, are read as any other's.
		{op: "listModules", columns: cycleColumns, request: toProject(http.MethodGet, "/modules", ""),
			cells: onTheCycles(cellOK, cellOK, cellOK, cellOK), check: listsTheModules},
		{op: "listModules", variant: "archived", columns: archivedColumns, request: toProject(http.MethodGet, "/modules", ""),
			cells: ofArchived(cellOK, cellForbidden), check: listsTheModules},
		// The archive alone (M6 P2 spec R9): no seeded module is archived.
		{op: "listModules", variant: "the archive", columns: cycleColumns, request: toProject(http.MethodGet, "/modules?archived=true", ""),
			cells: onTheCycles(cellOK, cellOK, cellOK, cellOK), check: listsNoModule},
		{op: "getModule", columns: cycleColumns, request: toModule(http.MethodGet, "", ""),
			cells: moduleRows.inCycleColumns(cellOK, cellOK, cellOK, cellOK, cellOK), check: readsTheModule},
		{op: "getModule", variant: "archived", columns: archivedColumns, request: toModule(http.MethodGet, "", ""),
			cells: moduleRows.ofArchived(cellOK, cellForbidden), check: readsTheModule},
		// As getModule (M6 P4 spec R10): no seeded module has work items.
		{op: "getModuleProgress", columns: cycleColumns, request: toModule(http.MethodGet, "/progress", ""),
			cells: moduleRows.inCycleColumns(cellOK, cellOK, cellOK, cellOK, cellOK), check: answersAnEmptyModuleProgress},
		{op: "getModuleProgress", variant: "archived", columns: archivedColumns, request: toModule(http.MethodGet, "/progress", ""),
			cells: moduleRows.ofArchived(cellOK, cellForbidden), check: answersAnEmptyModuleProgress},
		// The project's admins and members, not its guests, nor a caller who
		// sees only his own work items in it (M6 P2 spec R6); a workspace
		// admin or member who is no member of the project sees it and may
		// not.
		{op: "createModule", write: true, columns: cycleColumns, request: toProject(http.MethodPost, "/modules", newModule),
			cells: onTheCycles(cellCreated, cellCreated, cellForbidden, cellForbidden), check: createsTheModule},
		// Release's name in another case, and a lead who is no member of the
		// project: refused under the lock, after the decision, so who may not
		// create modules learns nothing of the project's (M6 P2 spec R2).
		{op: "createModule", variant: "a taken name", write: true, columns: cycleColumns,
			request: toProject(http.MethodPost, "/modules", `{"name":"RELEASE"}`),
			cells:   onTheCycles(cellModuleNameTaken, cellModuleNameTaken, cellForbidden, cellForbidden)},
		{op: "createModule", variant: "a lead who is no member", write: true, columns: cycleColumns,
			request: toProject(http.MethodPost, "/modules", `{"name":"Release 2","lead_id":"`+uuid.Nil().String()+`"}`),
			cells:   onTheCycles(cellValidationFailed, cellValidationFailed, cellForbidden, cellForbidden), refusal: "lead_id not_allowed"},
		// An archived project takes no modules (M6 design 3.2), after the
		// decision too.
		{op: "createModule", variant: "archived", write: true, columns: archivedColumns, request: toProject(http.MethodPost, "/modules", newModule),
			cells: ofArchived(cellProjectArchived, cellForbidden)},
		// The project's admins and members, as createModule, whoever made
		// the module.
		{op: "updateModule", write: true, columns: cycleColumns, request: toModule(http.MethodPatch, "", `{"sort_order":12345}`),
			cells: moduleRows.inCycleColumns(cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: reordersTheModule},
		{op: "updateModule", variant: "archived", write: true, columns: archivedColumns, request: toModule(http.MethodPatch, "", `{"sort_order":12345}`),
			cells: moduleRows.ofArchived(cellProjectArchived, cellForbidden)},
		// The column's module archived first in each cell's copy
		// (moduleArchived): it takes no change, one of nothing too, after the
		// decision (M6 P2 spec R2).
		{op: "updateModule", variant: "an archived module", write: true, columns: cycleColumns, prepare: moduleArchived,
			request: toModule(http.MethodPatch, "", `{}`),
			cells:   moduleRows.inCycleColumns(cellModuleArchived, cellModuleArchived, cellModuleArchived, cellForbidden, cellForbidden)},
		// The project's admins and members, as updateModule, whoever made
		// the module, once it is completed (moduleCompleted); a module not
		// closed, as seeded, is refused after the decision (M6 P2 spec R3).
		{op: "archiveModule", write: true, columns: cycleColumns, prepare: moduleCompleted, request: toModule(http.MethodPost, "/archive", ""),
			cells: moduleRows.inCycleColumns(cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: archivesTheModule(true)},
		{op: "archiveModule", variant: "a module not closed", write: true, columns: cycleColumns, request: toModule(http.MethodPost, "/archive", ""),
			cells: moduleRows.inCycleColumns(cellModuleNotClosed, cellModuleNotClosed, cellModuleNotClosed, cellForbidden, cellForbidden)},
		{op: "archiveModule", variant: "archived", write: true, columns: archivedColumns, request: toModule(http.MethodPost, "/archive", ""),
			cells: moduleRows.ofArchived(cellProjectArchived, cellForbidden)},
		// The column's module archived first in each cell's copy
		// (moduleArchived), whatever its status.
		{op: "unarchiveModule", write: true, columns: cycleColumns, prepare: moduleArchived, request: toModule(http.MethodPost, "/unarchive", ""),
			cells: moduleRows.inCycleColumns(cellOK, cellOK, cellOK, cellForbidden, cellForbidden), check: archivesTheModule(false)},
		{op: "unarchiveModule", variant: "archived", write: true, columns: archivedColumns, prepare: moduleArchived,
			request: toModule(http.MethodPost, "/unarchive", ""), cells: moduleRows.ofArchived(cellProjectArchived, cellForbidden)},
		// The project's admins, and the member who made the module
		// (AllowCreator): PM may not delete Release, the workspace admin's,
		// and deletes Mine, his own; an archived module too (M6 P2 spec 2.4).
		{op: "deleteModule", write: true, columns: cycleColumns, request: toModule(http.MethodDelete, "", ""),
			cells: moduleRows.inCycleColumns(cellNoContent, cellForbidden, cellNoContent, cellForbidden, cellForbidden)},
		{op: "deleteModule", variant: "archived", write: true, columns: archivedColumns, request: toModule(http.MethodDelete, "", ""),
			cells: moduleRows.ofArchived(cellProjectArchived, cellForbidden)},
		{op: "deleteModule", variant: "an archived module", write: true, columns: cycleColumns, prepare: moduleArchived,
			request: toModule(http.MethodDelete, "", ""), cells: moduleRows.inCycleColumns(cellNoContent, cellForbidden, cellNoContent, cellForbidden, cellForbidden)},
		// As getCyclePreferences and updateCyclePreferences, on the module's
		// page (M6 P3b spec 2.3, R6).
		{op: "getModulePreferences", columns: cycleColumns, request: toModulePreferences(http.MethodGet, ""),
			cells: moduleRows.inCycleColumns(cellOK, cellOK, cellOK, cellOK, cellOK), check: answersPageSettings(pageDefaults)},
		{op: "getModulePreferences", variant: "archived", columns: archivedColumns, request: toModulePreferences(http.MethodGet, ""),
			cells: moduleRows.ofArchived(cellOK, cellForbidden), check: answersPageSettings(pageDefaults)},
		{op: "updateModulePreferences", write: true, columns: cycleColumns, request: toModulePreferences(http.MethodPatch, moduleSettings),
			cells: moduleRows.inCycleColumns(cellOK, cellOK, cellOK, cellOK, cellOK), check: answersPageSettings(moduleSettings)},
		{op: "updateModulePreferences", variant: "archived", write: true, columns: archivedColumns,
			request: toModulePreferences(http.MethodPatch, moduleSettings), cells: moduleRows.ofArchived(cellOK, cellForbidden),
			check: answersPageSettings(moduleSettings)},
		{op: "updateModulePreferences", variant: "an archived module", write: true, columns: cycleColumns, prepare: moduleArchived,
			request: toModulePreferences(http.MethodPatch, moduleSettings), cells: moduleRows.inCycleColumns(cellOK, cellOK, cellOK, cellOK, cellOK),
			check: answersPageSettings(moduleSettings)},
	}
}

// moduleCompleted completes the column's own module in the cell's copy.
func moduleCompleted(t *testing.T, c caller, s seeded, url string) {
	t.Helper()
	pool := openPool(t, url)
	defer pool.Close()
	tag, err := pool.Exec(pgtest.Soon(t), "UPDATE modules SET status = 'completed' WHERE id = $1", s.module(projectOf(c), moduleOf(c)))
	if err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("completing %s's %s: %v, %d rows", c, moduleOf(c), err, tag.RowsAffected())
	}
}

// archivesTheModule: the column's own module, completed and archived, or
// planned and not archived, as archived says, last changed by the column's
// account at the moment it is archived; the rest as seeded.
func archivesTheModule(archived bool) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, c caller, s seeded, answer string) {
		var got moduleAnswer
		decodeAnswer(t, answer, &got)
		seeded := got
		seeded.Status, seeded.ArchivedAt, seeded.UpdatedByID = "planned", nil, got.CreatedByID
		isTheSeededModule(t, c, s, moduleOf(c), seeded, answer)
		ok := got.UpdatedByID != nil && *got.UpdatedByID == s.account(accountOf(c)) && got.UpdatedAt.After(got.CreatedAt)
		if archived {
			ok = ok && got.Status == "completed" && got.ArchivedAt != nil && got.ArchivedAt.Equal(got.UpdatedAt)
		} else {
			ok = ok && got.Status == "planned" && got.ArchivedAt == nil
		}
		if !ok {
			t.Errorf("%s answers %s; want %s's %s archived=%t, changed by %s", c, answer, projectOf(c), moduleOf(c), archived, accountOf(c))
		}
	}
}

// reordersTheModule: the column's own module at 12345, the rest as seeded,
// last changed by the column's account.
func reordersTheModule(t *testing.T, c caller, s seeded, answer string) {
	var got moduleAnswer
	decodeAnswer(t, answer, &got)
	if got.SortOrder != 12345 || got.UpdatedByID == nil || *got.UpdatedByID != s.account(accountOf(c)) || !got.UpdatedAt.After(got.CreatedAt) {
		t.Errorf("%s reorders %s; want %s's %s at 12345, changed by %s", c, answer, projectOf(c), moduleOf(c), accountOf(c))
	}
	for _, m := range matrixModules {
		if m.name == moduleOf(c) {
			seeded := got
			seeded.SortOrder, seeded.UpdatedByID = m.sortOrder, got.CreatedByID
			isTheSeededModule(t, c, s, m.name, seeded, answer)
		}
	}
}

// listsTheModules: the column's project's seeded modules, by sort order,
// each as seeded.
func listsTheModules(t *testing.T, c caller, s seeded, answer string) {
	got := listIn[moduleAnswer](t, answer)
	var want []string
	for _, m := range matrixModules {
		if m.project == "" || m.project == projectOf(c) {
			want = append(want, m.name)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("%s lists %s; want %s's %v", c, answer, projectOf(c), want)
	}
	for i, name := range want {
		isTheSeededModule(t, c, s, name, got[i], answer)
	}
}

// listsNoModule: an empty archive.
func listsNoModule(t *testing.T, c caller, _ seeded, answer string) {
	if got := listIn[moduleAnswer](t, answer); len(got) != 0 {
		t.Errorf("%s lists %s; want no archived module", c, answer)
	}
}

// readsTheModule: the column's own module (moduleOf), as seeded.
func readsTheModule(t *testing.T, c caller, s seeded, answer string) {
	var got moduleAnswer
	decodeAnswer(t, answer, &got)
	isTheSeededModule(t, c, s, moduleOf(c), got, answer)
}

// isTheSeededModule checks that got, of answer, is the module name of the
// column's project as matrixModules seeds it: planned, without dates, lead
// or members, at its sort order, created and last changed by its writer.
func isTheSeededModule(t *testing.T, c caller, s seeded, name string, got moduleAnswer, answer string) {
	t.Helper()
	slug, _, _ := strings.Cut(projectOf(c), "/")
	for _, m := range matrixModules {
		if m.name != name {
			continue
		}
		by := s.account(matrixAdmins[slug])
		if m.author != "" {
			by = s.account(m.author)
		}
		if got.ID != s.module(projectOf(c), name) || got.ProjectID != s.project(projectOf(c)) || got.WorkspaceID != s.workspace(slug) ||
			got.Name != name || got.Status != "planned" || got.StartDate != nil || got.TargetDate != nil || got.LeadID != nil ||
			got.MemberIDs == nil || len(got.MemberIDs) != 0 || got.SortOrder != m.sortOrder || got.ArchivedAt != nil ||
			got.CreatedByID == nil || *got.CreatedByID != by || got.UpdatedByID == nil || *got.UpdatedByID != by {
			t.Errorf("%s answers %s; want %s's %s as seeded", c, answer, projectOf(c), name)
		}
		return
	}
	t.Fatalf("no module %s in matrixModules", name)
}

// createsTheModule: Release 2 in the column's project and workspace,
// planned, without dates, lead or members, first among the project's
// modules (10000 below the seeded Release), created and last changed by
// the column's account at one moment.
func createsTheModule(t *testing.T, c caller, s seeded, answer string) {
	var got moduleAnswer
	decodeAnswer(t, answer, &got)
	slug, _, _ := strings.Cut(projectOf(c), "/")
	by := s.account(accountOf(c))
	if got.ProjectID != s.project(projectOf(c)) || got.WorkspaceID != s.workspace(slug) || got.Name != "Release 2" || got.Description != "" ||
		got.Status != "planned" || got.StartDate != nil || got.TargetDate != nil || got.LeadID != nil || got.MemberIDs == nil ||
		len(got.MemberIDs) != 0 || got.SortOrder != 55535 || got.ArchivedAt != nil || got.CreatedByID == nil || *got.CreatedByID != by ||
		got.UpdatedByID == nil || *got.UpdatedByID != by || !got.CreatedAt.Equal(got.UpdatedAt) {
		t.Errorf("%s creates %s; want Release 2 in %s, planned, at 55535, created and changed by %s at one moment", c, answer,
			projectOf(c), accountOf(c))
	}
}

// modules stores each project's matrixModules through the planning store,
// with the ids sd names, each by its writer.
func (s projectSeed) modules(sd seeded, store *planningpg.Store) {
	s.t.Helper()
	for _, p := range matrixProjects {
		slug, _, _ := strings.Cut(p.key, "/")
		for _, m := range matrixModules {
			if m.project != "" && m.project != p.key {
				continue
			}
			by := s.ids[matrixAdmins[slug]]
			if m.author != "" {
				by = s.ids[m.author]
			}
			if err := store.CreateModule(context.Background(), planningdomain.Module{ID: sd.module(p.key, m.name), ProjectID: s.projects[p.key],
				WorkspaceID: s.workspaces[slug], Name: m.name, Status: planningdomain.ModulePlanned, SortOrder: m.sortOrder, CreatedAt: s.now,
				UpdatedAt: s.now, CreatedByID: &by, UpdatedByID: &by}); err != nil {
				s.t.Fatal(err)
			}
		}
	}
}

// module is the id of the module name of the project key; one never seeded
// fails the test at once, as cycle's does.
func (s seeded) module(key, name string) uuid.UUID {
	id, ok := s.modules[key+"|"+name]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no module %s of %s is seeded", name, key)
	}
	return id
}
