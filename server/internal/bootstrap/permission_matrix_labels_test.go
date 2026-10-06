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
// creating them, under the project of each column; and the labels they
// rest on, matrixLabels, which prepareMatrix seeds in each project through
// the project store (labels) and reads back (seededLabels). They are here
// rather than in permission_matrix_seed_test.go, which has no room for
// them.

var cellLabelNameTaken = cell{http.StatusConflict, "project.label_name_taken"}

// matrixLabels are the labels prepareMatrix seeds in each project of
// matrixProjects, by its workspace's admin, at the sort orders createLabel
// gives labels created one after another: Bug and Feature at the top, UI
// under Bug. gone's are deleted with it.
var matrixLabels = []struct {
	name, parent string // parent: the name of the label it is under, "" at the top
	sortOrder    float64
}{
	{"Bug", "", 65535}, {"UI", "Bug", 75535}, {"Feature", "", 85535},
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
				ProjectID: s.projects[p.key], ParentID: parent, Name: l.name, SortOrder: l.sortOrder, CreatedBy: s.ids[matrixAdmins[slug]],
				Now: s.now}); err != nil {
				s.t.Fatal(err)
			}
		}
	}
}

// seededLabel is a label of a matrix project as seededLabels reads it: its
// parent by name, "" at the top.
type seededLabel struct {
	Name, Parent string
	SortOrder    float64
	Deleted      bool
}

// seededLabels checks the labels the cells of each matrix project rest on,
// read back by name: those of matrixLabels exactly, each under its parent,
// at its sort order; undeleted, but gone's, deleted with gone. A label
// missing or seeded otherwise would let a cell answer as it wants for
// another reason.
func (s projectSeed) seededLabels(pool *pgxpool.Pool) {
	s.t.Helper()
	for _, p := range matrixProjects {
		gone := strings.HasPrefix(p.key, "gone/")
		var want []seededLabel
		for _, l := range matrixLabels {
			want = append(want, seededLabel{Name: l.name, Parent: l.parent, SortOrder: l.sortOrder, Deleted: gone})
		}
		slices.SortFunc(want, func(a, b seededLabel) int { return strings.Compare(a.Name, b.Name) })
		rows, err := pool.Query(context.Background(), `SELECT l.name, coalesce(p.name, ''), l.sort_order, l.deleted_at IS NOT NULL
			FROM labels l LEFT JOIN labels p ON p.id = l.parent_id WHERE l.project_id = $1 ORDER BY l.name COLLATE "C"`, s.projects[p.key])
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
