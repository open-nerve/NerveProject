package postgresadapter_test

import (
	"context"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The projects favorites name (M7 P3c spec 2.4, 2.5), read by the ids asked, in one statement and without a lock: each
// undeleted project's workspace, name, icon and whether it is archived, another workspace's too, each once though asked
// twice; none for a deleted project, an id of no project or no ids. A FOR UPDATE of a project read does not wait.
func TestNavigationTargets(t *testing.T) {
	s, pool := newStore(t)
	maker := newAccount(t, pool, "maker@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, old := newProject(t, s, acme, "Web", "WEB", maker), newProject(t, s, acme, "Old", "OLD", maker)
	gone, site := newProject(t, s, acme, "Gone", "GONE", maker), newProject(t, s, beta, "Site", "SITE", maker)
	exec(t, pool, `UPDATE projects SET logo_props = '{"in_use":"emoji","emoji":{"value":"128512"}}' WHERE id = $1`, web)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", old, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)
	emoji, smile := shared.LogoInUseEmoji, "128512"
	all := map[uuid.UUID]app.NavigationTarget{
		web:  {WorkspaceID: acme, Name: "Web", LogoProps: shared.LogoProps{InUse: &emoji, Emoji: &shared.LogoEmoji{Value: &smile}}},
		old:  {WorkspaceID: acme, Name: "Old", Archived: true},
		site: {WorkspaceID: beta, Name: "Site"},
	}
	for _, tt := range []struct {
		name string
		ids  []uuid.UUID
		want map[uuid.UUID]app.NavigationTarget
	}{
		{"each kind", []uuid.UUID{web, old, gone, site, uuid.NewV7(), web}, all},
		{"none", nil, map[uuid.UUID]app.NavigationTarget{}},
	} {
		err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			got, err := s.NavigationTargets(ctx, tt.ids)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s: NavigationTargets = %+v; want %+v", tt.name, got, tt.want)
			}
			for id := range tt.want {
				if waits(t, pool, id, "FOR UPDATE") {
					t.Errorf("%s: a FOR UPDATE of %s waits; want no lock", tt.name, id)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
