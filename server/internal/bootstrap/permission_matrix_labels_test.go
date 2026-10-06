package bootstrap

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
)

// The rows of the permission matrix of a project's labels (M3 design 9.2):
// creating them, under the project of each column, and the writes on one,
// naming it by its id (/labels/{label_id}) among its column's project's
// seeded labels; and the labels they rest on, matrixLabels, which
// prepareMatrix seeds in each project through the project store (labels)
// and reads back (seededLabels). They are here rather than in
// permission_matrix_seed_test.go, which has no room for them.

var (
	cellLabelNameTaken = cell{http.StatusConflict, "project.label_name_taken"}
	cellLabelNotFound  = cell{http.StatusNotFound, "project.label_not_found"}
)

// labelRows are the labels the writes of the matrix name by id: a column
// that does not see its project answers project.label_not_found.
var labelRows = rowsByID{path: "labels", notFound: cellLabelNotFound, find: seeded.label}

// matrixLabels are the labels prepareMatrix seeds in each project of
// matrixProjects, by its workspace's admin, at the sort orders createLabel
// gives labels created one after another: Bug and Feature at the top, UI
// under Bug, with a color, the others without one (the column's default):
// a write that lost UI's parent, sort order or color would not give it
// back by chance (renamesTheLabel). gone's are deleted with it. Their
// writer in acme, as for the projects and states the matrix seeds, is its
// admin, WA-'s account, a member of none of its projects, so no column
// that may write a label calls as him: nothing reads the labels' writers,
// and a check that a write made its caller a label's writer cannot pass on
// the seed's.
var matrixLabels = []struct {
	name, parent string // parent: the name of the label it is under, "" at the top
	color        string
	sortOrder    float64
}{
	{"Bug", "", "", 65535}, {"UI", "Bug", "#3B82F6", 75535}, {"Feature", "", "", 85535},
}

// newLabel is the body of the label the rows create: QA, at the top, a
// name no seeded label has in any case.
const newLabel = `{"name":"QA","color":"#0EA5E9"}`

// withParent is the request of a row whose callers each create QA under
// the label parent names for their column.
func withParent(parent func(c caller, s seeded) uuid.UUID) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return toProject(http.MethodPost, "/labels", `{"name":"QA","parent_id":"`+parent(c, s).String()+`"}`)(c, s)
	}
}

func labelMatrixRows() []matrixRow {
	return []matrixRow{
		// The project's admins, and its members who are the workspace's
		// admins (M3 design 3.4), as createState.
		{op: "createLabel", write: true, columns: projectColumns, request: toProject(http.MethodPost, "/labels", newLabel),
			cells: ofProject(cellCreated, cellForbidden, cellForbidden, cellCreated, cellForbidden, cellForbidden), check: createsTheLabel},
		// Bug in another case: the 409 comes after the decision, so who may
		// not create labels learns nothing of the project's.
		{op: "createLabel", variant: "a name taken", write: true, columns: projectColumns,
			request: toProject(http.MethodPost, "/labels", `{"name":"bug"}`),
			cells:   ofProject(cellLabelNameTaken, cellForbidden, cellForbidden, cellLabelNameTaken, cellForbidden, cellForbidden)},
		// UI, under Bug, as the parent: labels have two levels (M3 design
		// 3.16), and the parent is checked after the decision.
		{op: "createLabel", variant: "a parent with a parent", write: true, columns: projectColumns,
			request: withParent(func(c caller, s seeded) uuid.UUID { return s.label(projectOf(c), "UI") }),
			cells:   ofProject(cellValidationFailed, cellForbidden, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "parent_id not_allowed"},
		// The archived project's Bug, a label of another project of the same
		// workspace: the parent's project is checked, not only its workspace.
		{op: "createLabel", variant: "a parent of another project", write: true, columns: projectColumns,
			request: withParent(func(_ caller, s seeded) uuid.UUID { return s.label("acme/archived", "Bug") }),
			cells:   ofProject(cellValidationFailed, cellForbidden, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "parent_id not_allowed"},
		// An archived project's labels are created as any other's (M3 design
		// 3.19).
		{op: "createLabel", variant: "archived", write: true, columns: archivedColumns, request: toProject(http.MethodPost, "/labels", newLabel),
			cells: ofArchived(cellCreated, cellForbidden), check: createsTheLabel},
		// As createLabel: UI, under Bug, renamed.
		{op: "updateLabel", write: true, columns: projectColumns, request: labelRows.to(http.MethodPatch, "", "UI", `{"name":"Story"}`),
			cells: labelRows.of(cellOK, cellForbidden, cellForbidden, cellOK, cellForbidden, cellForbidden), check: renamesTheLabel},
		{op: "updateLabel", variant: "a name taken", write: true, columns: projectColumns,
			request: labelRows.to(http.MethodPatch, "", "Feature", `{"name":"bug"}`),
			cells:   labelRows.of(cellLabelNameTaken, cellForbidden, cellForbidden, cellLabelNameTaken, cellForbidden, cellForbidden)},
		// Bug, which has UI under it, under Feature: the parent is checked
		// after the decision.
		{op: "updateLabel", variant: "a label with labels under it given a parent", write: true, columns: projectColumns,
			request: func(c caller, s seeded) (string, string, string) {
				return labelRows.to(http.MethodPatch, "", "Bug", `{"parent_id":"`+s.label(projectOf(c), "Feature").String()+`"}`)(c, s)
			},
			cells:   labelRows.of(cellValidationFailed, cellForbidden, cellForbidden, cellValidationFailed, cellForbidden, cellForbidden),
			refusal: "parent_id not_allowed"},
		// A value refused before the label is looked at: the same 422 in
		// every column.
		{op: "updateLabel", variant: "a value refused", write: true, columns: projectColumns,
			request: labelRows.to(http.MethodPatch, "", "Feature", `{"name":""}`), cells: func() map[caller]cell {
				cells := map[caller]cell{}
				for _, c := range projectColumns {
					cells[c] = cellValidationFailed
				}
				return cells
			}(), refusal: "name too_short"},
		{op: "updateLabel", variant: "archived", write: true, columns: archivedColumns,
			request: labelRows.to(http.MethodPatch, "", "UI", `{"name":"Story"}`), cells: labelRows.ofArchived(cellOK, cellForbidden),
			check: renamesTheLabel},
	}
}

// createsTheLabel: QA at the top of the column's project, after its
// Feature, the greatest sort order of its labels.
func createsTheLabel(t *testing.T, c caller, s seeded, answer string) {
	var l struct {
		ProjectID uuid.UUID  `json:"project_id"`
		ParentID  *uuid.UUID `json:"parent_id"`
		Name      string     `json:"name"`
		Color     string     `json:"color"`
		SortOrder float64    `json:"sort_order"`
	}
	decodeAnswer(t, answer, &l)
	if l.ProjectID != s.project(projectOf(c)) || l.ParentID != nil || l.Name != "QA" || l.Color != "#0EA5E9" || l.SortOrder != 95535 {
		t.Errorf("%s creates %s; want QA in %s, at the top, at 95535", c, answer, projectOf(c))
	}
}

// renamesTheLabel: the column's project's UI renamed Story, as stored, and
// what the body leaves out kept as seeded: under its project's Bug, at
// 75535, with its color. None of these is what a write that lost one would
// give: the top, a column's default, a place after the project's labels.
func renamesTheLabel(t *testing.T, c caller, s seeded, answer string) {
	var l struct {
		ID        uuid.UUID  `json:"id"`
		ProjectID uuid.UUID  `json:"project_id"`
		ParentID  *uuid.UUID `json:"parent_id"`
		Name      string     `json:"name"`
		Color     string     `json:"color"`
		SortOrder float64    `json:"sort_order"`
	}
	decodeAnswer(t, answer, &l)
	if l.ID != s.label(projectOf(c), "UI") || l.ProjectID != s.project(projectOf(c)) || l.ParentID == nil ||
		*l.ParentID != s.label(projectOf(c), "Bug") || l.Name != "Story" || l.Color != "#3B82F6" || l.SortOrder != 75535 {
		t.Errorf("%s renames %s; want %s's UI, under its Bug, at 75535, with #3B82F6, named Story", c, answer, projectOf(c))
	}
}

// labels stores each project's matrixLabels, each parent before the labels
// under it, with the ids sd names, by its workspace's admin.
func (s projectSeed) labels(sd seeded) {
	s.t.Helper()
	for _, p := range matrixProjects {
		slug, _, _ := strings.Cut(p.key, "/")
		for _, l := range matrixLabels {
			var parent *uuid.UUID
			if l.parent != "" {
				id := sd.label(p.key, l.parent)
				parent = &id
			}
			if _, err := s.store.CreateLabel(context.Background(), projectapp.LabelRow{ID: sd.label(p.key, l.name), WorkspaceID: s.workspaces[slug],
				ProjectID: s.projects[p.key], ParentID: parent, Name: l.name, Color: l.color, SortOrder: l.sortOrder,
				CreatedBy: s.ids[matrixAdmins[slug]], Now: s.now}); err != nil {
				s.t.Fatal(err)
			}
		}
	}
}

// seededLabel is a label of a matrix project as seededLabels reads it: its
// parent's id, uuid.Nil at the top.
type seededLabel struct {
	ID, WorkspaceID, ParentID uuid.UUID
	Name, Color               string
	SortOrder                 float64
	Deleted                   bool
}

// seededLabels checks the labels the cells of each matrix project rest on,
// read back by name: those of matrixLabels exactly, each with the id sd
// names for it, in its project's workspace, under the label sd names for
// its parent in its project, with its color, at its sort order; undeleted,
// but gone's, deleted with gone. A label missing or seeded otherwise would
// let a cell answer as it wants for another reason: a row names its parent
// by the id sd names, and a parent that is no label, or another project's,
// is refused as one under another is (parent_id not_allowed), and a
// rename shows UI's color kept only if UI was seeded with it.
func (s projectSeed) seededLabels(pool *pgxpool.Pool, sd seeded) {
	s.t.Helper()
	for _, p := range matrixProjects {
		slug, _, _ := strings.Cut(p.key, "/")
		var want []seededLabel
		for _, l := range matrixLabels {
			parent := uuid.Nil()
			if l.parent != "" {
				parent = sd.label(p.key, l.parent)
			}
			want = append(want, seededLabel{ID: sd.label(p.key, l.name), WorkspaceID: sd.workspace(slug), ParentID: parent, Name: l.name,
				Color: l.color, SortOrder: l.sortOrder, Deleted: slug == "gone"})
		}
		slices.SortFunc(want, func(a, b seededLabel) int { return strings.Compare(a.Name, b.Name) })
		rows, err := pool.Query(context.Background(), `SELECT id, workspace_id, coalesce(parent_id, $2), name, color, sort_order,
			deleted_at IS NOT NULL FROM labels WHERE project_id = $1 ORDER BY name COLLATE "C"`, sd.project(p.key), uuid.Nil())
		if err != nil {
			s.t.Fatal(err)
		}
		got, err := pgx.CollectRows(rows, pgx.RowToStructByPos[seededLabel])
		if err != nil {
			s.t.Fatal(err)
		}
		if !slices.Equal(got, want) {
			s.t.Fatalf("%s's labels by name = %+v; want %+v", p.key, got, want)
		}
	}
}
