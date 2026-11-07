package bootstrap

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"uuid"

	planningpg "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres"
	planningdomain "github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

// The rows of the permission matrix of a project's modules (M6 P2 spec
// R10): creating them under the project of each column; and the modules
// they rest on, matrixModules, which prepareMatrix seeds in each project
// through the planning store (modules) once the project store has turned
// the project's modules on (planningOn). The module table's columns and
// cells are the cycle table's (cycleColumns, onTheCycles): the member who
// made a module (callerCreator) is PM's account, as for the cycles.

var cellModuleNameTaken = cell{http.StatusConflict, "planning.module_name_taken"}

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

// newModule is the body of the module the rows create: Release 2, its name
// alone.
const newModule = `{"name":"Release 2"}`

func moduleMatrixRows() []matrixRow {
	return []matrixRow{
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
	}
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
