package postgresadapter_test

import (
	"context"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The cycles and the modules favorites name (M7 P3c spec 2.4, 2.5), read by the ids asked, in one statement each and
// without a lock: each undeleted one's workspace, project, name and whether it is archived, another workspace's and
// another project's too, each once though asked twice; none for a deleted one, an id of none or no ids. A FOR UPDATE
// of one read does not wait.
func TestNavigationTargets(t *testing.T) {
	s, pool := newStore(t)
	w := newWorld(t, pool)
	beta, site := uuid.NewV7(), uuid.NewV7()
	exec(t, pool, "INSERT INTO workspaces (id, name, slug) VALUES ($1, 'Beta', 'beta')", beta)
	exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, 'Site', 'SITE')", site, beta)
	for _, c := range []struct {
		table string
		read  func(context.Context, []uuid.UUID) (map[uuid.UUID]app.NavigationTarget, error)
	}{
		{"cycles", s.NavigationCycles},
		{"modules", s.NavigationModules},
	} {
		t.Run(c.table, func(t *testing.T) {
			ids := sortedIDs(5)
			live, archived, deleted, other, elsewhere := ids[0], ids[1], ids[2], ids[3], ids[4]
			plant(t, pool, w, c.table,
				planned{id: live, project: w.web, name: "Live"},
				planned{id: archived, project: w.web, name: "Done", archived: true},
				planned{id: deleted, project: w.web, name: "Gone", deleted: true},
				planned{id: other, project: w.ops, name: "Ops"})
			exec(t, pool, "INSERT INTO "+c.table+" (id, workspace_id, project_id, name) VALUES ($1, $2, $3, 'Site')", elsewhere, beta, site)
			all := map[uuid.UUID]app.NavigationTarget{
				live:      {WorkspaceID: w.workspace, ProjectID: w.web, Name: "Live"},
				archived:  {WorkspaceID: w.workspace, ProjectID: w.web, Name: "Done", Archived: true},
				other:     {WorkspaceID: w.workspace, ProjectID: w.ops, Name: "Ops"},
				elsewhere: {WorkspaceID: beta, ProjectID: site, Name: "Site"},
			}
			for _, tt := range []struct {
				name string
				ids  []uuid.UUID
				want map[uuid.UUID]app.NavigationTarget
			}{
				{"each kind", []uuid.UUID{live, archived, deleted, other, elsewhere, uuid.NewV7(), live}, all},
				{"none", nil, map[uuid.UUID]app.NavigationTarget{}},
			} {
				err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
					got, err := c.read(ctx, tt.ids)
					if err != nil {
						return err
					}
					if !reflect.DeepEqual(got, tt.want) {
						t.Errorf("%s: read = %+v; want %+v", tt.name, got, tt.want)
					}
					for id := range tt.want {
						if pgtest.RowWaits(t, pool, c.table, id, "FOR UPDATE") {
							t.Errorf("%s: a FOR UPDATE of %s waits; want no lock", tt.name, id)
						}
					}
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
