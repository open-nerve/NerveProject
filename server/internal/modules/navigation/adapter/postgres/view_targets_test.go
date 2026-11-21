package postgresadapter_test

import (
	"context"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The views favorites name (P3c spec 2.4, 2.5), read by the ids asked, without a lock: each undeleted view of either
// kind, another workspace's too, each once though asked twice, as a read of it by its id reads it; none for a deleted
// view, an id of none or no ids. A FOR UPDATE of a view read does not wait.
func TestViewTargets(t *testing.T) {
	s, pool := newStore(t)
	w := newViews(t, pool)
	project, workspace := view(1, w.workspace, &w.project, w.owner, now.Add(-time.Hour)), view(2, w.workspace, nil, w.other, now.Add(-time.Hour))
	elsewhere, gone := view(3, w.elsewhere, &w.elsewhereProject, w.owner, now.Add(-time.Hour)), view(4, w.workspace, nil, w.owner, now.Add(-time.Hour))
	project.Name, workspace.Name = "Triage", "Everything"
	insert(t, s, project, workspace, elsewhere, gone)
	if err := s.DeleteView(context.Background(), gone.ID, now, w.owner); err != nil {
		t.Fatal(err)
	}
	within(t, pool, func(ctx context.Context) {
		got, err := s.ViewTargets(ctx, []uuid.UUID{project.ID, workspace.ID, elsewhere.ID, gone.ID, uuid.NewV7(), project.ID})
		want := map[uuid.UUID]any{project.ID: project, workspace.ID: workspace, elsewhere.ID: elsewhere}
		if err != nil || len(got) != len(want) {
			t.Fatalf("ViewTargets = %+v, %v; want %d views", got, err, len(want))
		}
		for id, v := range got {
			if !reflect.DeepEqual(any(v), want[id]) {
				t.Errorf("view %s = %+v; want %+v", id, v, want[id])
			}
			if pgtest.RowWaits(t, pool, "issue_views", id, "FOR UPDATE") {
				t.Errorf("a FOR UPDATE of %s waits; want no lock", id)
			}
		}
		if got, err := s.ViewTargets(ctx, nil); err != nil || got == nil || len(got) != 0 {
			t.Fatalf("ViewTargets(none) = %#v, %v; want none", got, err)
		}
	})
}

// A view a favorite names is locked FOR SHARE until the transaction ends, after its workspace's and its project's FOR
// SHARE (spec 2.8): its deletion and its change (FOR UPDATE) wait for it, another FOR SHARE and a foreign key's FOR KEY
// SHARE do not. A deleted view and an id of none take no lock and are no failure: the reads under the locks find
// them gone.
func TestShareAView(t *testing.T) {
	s, pool := newStore(t)
	w := newViews(t, pool)
	live, gone := view(1, w.workspace, &w.project, w.owner, now.Add(-time.Hour)), view(2, w.workspace, nil, w.owner, now.Add(-time.Hour))
	insert(t, s, live, gone)
	if err := s.DeleteView(context.Background(), gone.ID, now, w.owner); err != nil {
		t.Fatal(err)
	}
	waits := func(id uuid.UUID, lock string) bool { return pgtest.RowWaits(t, pool, "issue_views", id, lock) }
	within(t, pool, func(ctx context.Context) {
		for _, id := range []uuid.UUID{gone.ID, uuid.NewV7()} {
			if err := s.ShareView(ctx, id); err != nil {
				t.Fatalf("share %s: %v", id, err)
			}
		}
		if waits(gone.ID, "FOR UPDATE") {
			t.Fatal("the deleted view is locked; want no lock")
		}
		if err := s.ShareView(ctx, live.ID); err != nil {
			t.Fatal(err)
		}
		if !waits(live.ID, "FOR UPDATE") || !waits(live.ID, "FOR NO KEY UPDATE") || waits(live.ID, "FOR SHARE") || waits(live.ID, "FOR KEY SHARE") {
			t.Fatal("the view is not held FOR SHARE")
		}
	})
	if waits(live.ID, "FOR UPDATE") {
		t.Fatal("after the transaction: the view is still locked")
	}
}
