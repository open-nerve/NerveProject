package postgresadapter_test

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
)

// The asset path reads its host's project, under the project lock its
// caller holds, in one statement (M7 P2c spec 2.4, R4): whether it is
// archived and whether its intake is turned on, each of the four together
// as stored; a deleted project, or an id of no project, is not found; an
// error as it came.
func TestTheAssetProjectReadCarriesTheArchiveAndTheIntakeToggle(t *testing.T) {
	s, pool := newStore(t)
	owner := newAccount(t, pool, "owner@corp.com")
	acme := newWorkspace(t, pool, "acme")
	for _, tt := range []struct {
		identifier string
		want       app.AssetProject
	}{
		{"OFF", app.AssetProject{}},
		{"ON", app.AssetProject{IntakeView: true}},
		{"ARC", app.AssetProject{Archived: true}},
		{"ARO", app.AssetProject{Archived: true, IntakeView: true}},
	} {
		id := newProject(t, s, acme, tt.identifier, tt.identifier, owner)
		exec(t, pool, "UPDATE projects SET intake_view=$2 WHERE id=$1", id, tt.want.IntakeView)
		if tt.want.Archived {
			exec(t, pool, "UPDATE projects SET archived_at=$2 WHERE id=$1", id, now)
		}
		if got, found, err := s.AssetProject(context.Background(), id); err != nil || !found || got != tt.want {
			t.Errorf("AssetProject(%s) = %+v, %v, %v; want %+v, found", tt.identifier, got, found, err, tt.want)
		}
	}
	deleted := newProject(t, s, acme, "Deleted", "DEL", owner)
	exec(t, pool, "UPDATE projects SET intake_view=true, archived_at=$2, deleted_at=$2 WHERE id=$1", deleted, now)
	for _, id := range []uuid.UUID{deleted, uuid.NewV7()} {
		if got, found, err := s.AssetProject(context.Background(), id); err != nil || found || got != (app.AssetProject{}) {
			t.Errorf("AssetProject(%s) = %+v, %v, %v; want none, not found", id, got, found, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, found, err := s.AssetProject(ctx, deleted); !errors.Is(err, context.Canceled) || found || got != (app.AssetProject{}) {
		t.Errorf("cancelled: %+v, %v, %v", got, found, err)
	}
}
