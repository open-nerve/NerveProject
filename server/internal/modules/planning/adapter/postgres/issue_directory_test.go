package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
)

// The names a list ordered by the cycles' or the modules' names keys its work items by (M6 P3b spec R2): of the
// projects asked about, every undeleted cycle or module, archived ones too, with its name, in id order across the
// projects, whether the project has its cycles and modules turned on or not; another project's and a deleted one are
// not among them, nor is anything when no project, or a project of none, is asked about.
func TestIssueDirectoryOfTheProjects(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	docs := uuid.NewV7()
	exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, 'Docs', 'DOC')", docs, w.workspace)
	// web has its cycles and modules turned on; ops, as a project is made, has neither.
	exec(t, pool, "UPDATE projects SET cycle_view = true, module_view = true WHERE id = $1", w.web)
	for _, c := range []struct {
		table string
		read  func(context.Context, []uuid.UUID) ([]app.IssueDirectoryName, error)
	}{
		{"cycles", s.CycleNames},
		{"modules", s.ModuleNames},
	} {
		t.Run(c.table, func(t *testing.T) {
			ids := sortedIDs(6)
			webLive, opsLive, webArchived, webDeleted, opsArchived, docsLive := ids[0], ids[1], ids[2], ids[3], ids[4], ids[5]
			// Inserted out of id order and the projects interleaved, so the answer's order is the statement's.
			plant(t, pool, w, c.table,
				planned{id: docsLive, project: docs, name: "Docs " + c.table},
				planned{id: opsArchived, project: w.ops, name: "Old ops " + c.table, archived: true},
				planned{id: webDeleted, project: w.web, name: "Gone " + c.table, deleted: true},
				planned{id: webArchived, project: w.web, name: "Done " + c.table, archived: true},
				planned{id: opsLive, project: w.ops, name: "Ops " + c.table},
				planned{id: webLive, project: w.web, name: "Live " + c.table})
			got, err := c.read(ctx, []uuid.UUID{w.ops, w.web, w.web})
			want := []app.IssueDirectoryName{{ID: webLive, Name: "Live " + c.table}, {ID: opsLive, Name: "Ops " + c.table},
				{ID: webArchived, Name: "Done " + c.table}, {ID: opsArchived, Name: "Old ops " + c.table}}
			if err != nil || !slices.Equal(got, want) {
				t.Errorf("names of web and ops = %+v, %v\nwant %+v", got, err, want)
			}
			if got, err := c.read(ctx, []uuid.UUID{w.ops}); err != nil || !slices.Equal(got, []app.IssueDirectoryName{{ID: opsLive, Name: "Ops " + c.table}, {ID: opsArchived, Name: "Old ops " + c.table}}) {
				t.Errorf("names of ops, whose toggles are off = %+v, %v; want its own alone, archived too", got, err)
			}
			for _, none := range [][]uuid.UUID{nil, {}, {uuid.NewV7()}} {
				if got, err := c.read(ctx, none); err != nil || got == nil || len(got) != 0 {
					t.Errorf("names of %v = %#v, %v; want none", none, got, err)
				}
			}
			cancelled, cancel := context.WithCancel(ctx)
			cancel()
			if got, err := c.read(cancelled, []uuid.UUID{w.web}); !errors.Is(err, context.Canceled) || got != nil {
				t.Errorf("names in a cancelled context = %+v, %v; want the failure", got, err)
			}
		})
	}
}
