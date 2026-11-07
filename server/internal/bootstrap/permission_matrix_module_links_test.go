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
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The rows of the permission matrix of a module's links (M6 P2 spec R10):
// listing them and adding one to the column's own module (moduleOf), named
// by its id (/modules/{module_id}/links), and changing and removing that
// module's seeded link (/module-links/{module_link_id}); the links each
// module of matrixModules has, which prepareMatrix seeds through the
// planning store (moduleLinks). The table's columns and cells are the cycle
// table's (cycleColumns); a link's write is its module's (module.write), so
// the member who made Mine (callerCreator) writes its link as any member.

var (
	cellModuleLinkNotFound = cell{http.StatusNotFound, "planning.module_link_not_found"}
	cellModuleLinkExists   = cell{http.StatusConflict, "planning.module_link_exists"}
	cellModuleArchived     = cell{http.StatusConflict, "planning.archived"}
)

// moduleLinkRows are the links the rows name by id: a column that does not
// see its project answers planning.module_link_not_found.
var moduleLinkRows = rowsByID{path: "module-links", notFound: cellModuleLinkNotFound, find: seeded.moduleLink}

// matrixModuleLink is the address of each seeded module's one link: a
// link-local one, which no crawl may reach.
const matrixModuleLink = "http://169.254.169.254/matrix-module"

// toModuleLink is the request of a row whose callers each send method, with
// body, to the link of their column's own module (moduleOf).
func toModuleLink(method, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return moduleLinkRows.to(method, "", moduleOf(c), body)(c, s)
	}
}

func moduleLinkMatrixRows() []matrixRow {
	list := toModule(http.MethodGet, "/links", "")
	create := toModule(http.MethodPost, "/links", `{"url":"http://169.254.169.254/matrix-create"}`)
	rows := []matrixRow{
		// Every active member of the project, its guests and a caller who
		// sees only his own work items there too (M6 P2 spec R6), an archived
		// module's and project's as any other's.
		{op: "listModuleLinks", columns: cycleColumns, request: list, cells: moduleRows.inCycleColumns(cellOK, cellOK, cellOK, cellOK, cellOK),
			check: listsTheModuleLinks},
		{op: "listModuleLinks", variant: "archived", columns: archivedColumns, request: list, cells: moduleRows.ofArchived(cellOK, cellForbidden),
			check: listsTheModuleLinks},
		// The project's admins and members, not its guests, nor a caller who
		// sees only his own work items in it.
		{op: "createModuleLink", write: true, columns: cycleColumns, request: create,
			cells: moduleRows.inCycleColumns(cellCreated, cellCreated, cellCreated, cellForbidden, cellForbidden), check: createsTheModuleLink},
		// The seeded link's address, refused under the lock, after the
		// decision.
		{op: "createModuleLink", variant: "a taken address", write: true, columns: cycleColumns,
			request: toModule(http.MethodPost, "/links", `{"url":"`+matrixModuleLink+`"}`),
			cells:   moduleRows.inCycleColumns(cellModuleLinkExists, cellModuleLinkExists, cellModuleLinkExists, cellForbidden, cellForbidden)},
		{op: "createModuleLink", variant: "archived", write: true, columns: archivedColumns, request: create,
			cells: moduleRows.ofArchived(cellProjectArchived, cellForbidden)},
		{op: "createModuleLink", variant: "an archived module", write: true, columns: cycleColumns, prepare: moduleArchived, request: create,
			cells: moduleRows.inCycleColumns(cellModuleArchived, cellModuleArchived, cellModuleArchived, cellForbidden, cellForbidden)},
	}
	for _, edit := range []bool{true, false} {
		op, request, success := "deleteModuleLink", toModuleLink(http.MethodDelete, ""), cellNoContent
		var check func(*testing.T, caller, seeded, string)
		if edit {
			op, request, success, check = "updateModuleLink", toModuleLink(http.MethodPatch, `{"title":"Edited"}`), cellOK, editsTheModuleLink
		}
		rows = append(rows,
			matrixRow{op: op, write: true, columns: cycleColumns, request: request,
				cells: moduleLinkRows.inCycleColumns(success, success, success, cellForbidden, cellForbidden), check: check},
			matrixRow{op: op, variant: "archived", write: true, columns: archivedColumns, request: request,
				cells: moduleLinkRows.ofArchived(cellProjectArchived, cellForbidden)},
			matrixRow{op: op, variant: "an archived module", write: true, columns: cycleColumns, prepare: moduleArchived, request: request,
				cells: moduleLinkRows.inCycleColumns(cellModuleArchived, cellModuleArchived, cellModuleArchived, cellForbidden, cellForbidden)})
	}
	return rows
}

// isTheSeededLink checks that got, of answer, is the seeded link of the
// column's own module, as seeded but for its title.
func isTheSeededLink(t *testing.T, c caller, s seeded, got moduleLinkAnswer, answer string) {
	t.Helper()
	if got.ID != s.moduleLink(projectOf(c), moduleOf(c)) || got.ModuleID != s.module(projectOf(c), moduleOf(c)) || got.URL != matrixModuleLink ||
		got.Metadata == nil || len(got.Metadata) != 0 {
		t.Errorf("%s answers %s; want the link of %s's %s as seeded", c, answer, projectOf(c), moduleOf(c))
	}
}

// listsTheModuleLinks: the column's own module's one link, without a title.
func listsTheModuleLinks(t *testing.T, c caller, s seeded, answer string) {
	got := listIn[moduleLinkAnswer](t, answer)
	if len(got) != 1 || got[0].Title != nil {
		t.Fatalf("%s lists %s; want the one link of %s's %s", c, answer, projectOf(c), moduleOf(c))
	}
	isTheSeededLink(t, c, s, got[0], answer)
}

// createsTheModuleLink: the column's own module's new link, its metadata
// empty, added by the column's account.
func createsTheModuleLink(t *testing.T, c caller, s seeded, answer string) {
	var got moduleLinkAnswer
	decodeAnswer(t, answer, &got)
	by := s.account(accountOf(c))
	if got.ModuleID != s.module(projectOf(c), moduleOf(c)) || got.URL != "http://169.254.169.254/matrix-create" || got.Title != nil ||
		got.Metadata == nil || len(got.Metadata) != 0 || got.CreatedByID == nil || *got.CreatedByID != by || !got.CreatedAt.Equal(got.UpdatedAt) {
		t.Errorf("%s creates %s; want a link of %s's %s, added by %s", c, answer, projectOf(c), moduleOf(c), accountOf(c))
	}
}

// editsTheModuleLink: the column's own module's link, titled Edited, changed
// after it was added.
func editsTheModuleLink(t *testing.T, c caller, s seeded, answer string) {
	var got moduleLinkAnswer
	decodeAnswer(t, answer, &got)
	isTheSeededLink(t, c, s, got, answer)
	if got.Title == nil || *got.Title != "Edited" || !got.UpdatedAt.After(got.CreatedAt) {
		t.Errorf("%s edits %s; want it titled Edited", c, answer)
	}
}

// moduleArchived archives the column's own module in the cell's copy, at
// its last change.
func moduleArchived(t *testing.T, c caller, s seeded, url string) {
	t.Helper()
	pool := openPool(t, url)
	defer pool.Close()
	tag, err := pool.Exec(pgtest.Soon(t), "UPDATE modules SET archived_at = updated_at WHERE id = $1", s.module(projectOf(c), moduleOf(c)))
	if err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("archiving %s's %s: %v, %d rows", c, moduleOf(c), err, tag.RowsAffected())
	}
}

// moduleLinks stores the one link of each module of each project's
// matrixModules through the planning store, with the ids sd names, by the
// module's writer.
func (s projectSeed) moduleLinks(sd seeded, store *planningpg.Store) {
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
			if err := store.InsertModuleLink(context.Background(), planningdomain.ModuleLink{ID: sd.moduleLink(p.key, m.name),
				WorkspaceID: s.workspaces[slug], ProjectID: s.projects[p.key], ModuleID: sd.module(p.key, m.name), URL: matrixModuleLink,
				Metadata: shared.LinkMetadata{}, CreatedByID: &by, CreatedAt: s.now, UpdatedAt: s.now}); err != nil {
				s.t.Fatal(err)
			}
		}
	}
}

// moduleLink is the id of the link of the module name of the project key;
// one never seeded fails the test at once, as module's does.
func (s seeded) moduleLink(key, name string) uuid.UUID {
	id, ok := s.moduleLinks[key+"|"+name]
	if !ok {
		s.t.Helper()
		s.t.Fatalf("no link of the module %s of %s is seeded", name, key)
	}
	return id
}
