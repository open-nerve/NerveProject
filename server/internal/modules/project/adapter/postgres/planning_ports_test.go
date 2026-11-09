package postgresadapter_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// planningProjects are Web (Asia/Shanghai, cycles on), Arch (archived, modules on) and Gone (deleted) of acme.
func planningProjects(t *testing.T) (*postgresadapter.Store, *pgxpool.Pool, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, arch, gone := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Arch", "ARCH", alice), newProject(t, s, acme,
		"Gone", "GONE", alice)
	exec(t, pool, "UPDATE projects SET timezone = 'Asia/Shanghai', cycle_view = true WHERE id = $1", web)
	exec(t, pool, "UPDATE projects SET archived_at = $2, module_view = true WHERE id = $1", arch, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)
	return s, pool, acme, web, arch, gone
}

// The planning module's facts of a project (M6 P1 spec 2.1): its workspace, whether it is archived, its time zone and
// its two toggles, an archived one's too, none for a deleted one or one that is not there. PlanningFacts reads them
// without a lock; LockPlanning holds the project FOR NO KEY UPDATE until the transaction ends, which a FOR SHARE waits
// for and a foreign key's FOR KEY SHARE does not, and leaves the workspace's row unlocked.
func TestPlanningFacts(t *testing.T) {
	s, pool, acme, web, arch, gone := planningProjects(t)
	cases := map[string]struct {
		id    uuid.UUID
		want  app.PlanningFacts
		found bool
	}{
		"Web":     {web, app.PlanningFacts{WorkspaceID: acme, Timezone: "Asia/Shanghai", CycleView: true}, true},
		"Arch":    {arch, app.PlanningFacts{WorkspaceID: acme, Archived: true, Timezone: "UTC", ModuleView: true}, true},
		"Gone":    {gone, app.PlanningFacts{}, false},
		"Missing": {uuid.NewV7(), app.PlanningFacts{}, false},
	}
	for name, tt := range cases {
		err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			f, found, err := s.PlanningFacts(ctx, tt.id)
			if err != nil {
				return err
			}
			if found != tt.found || jsonOf(t, f) != jsonOf(t, tt.want) {
				t.Errorf("PlanningFacts(%s) = %+v, %v; want %+v, %v", name, f, found, tt.want, tt.found)
			}
			if name != "Missing" && waits(t, pool, tt.id, "FOR UPDATE") {
				t.Errorf("%s after PlanningFacts: a FOR UPDATE waits; want no lock", name)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for name, tt := range cases {
		err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			f, found, err := s.LockPlanning(ctx, tt.id)
			if err != nil {
				return err
			}
			if found != tt.found || jsonOf(t, f) != jsonOf(t, tt.want) {
				t.Errorf("LockPlanning(%s) = %+v, %v; want %+v, %v", name, f, found, tt.want, tt.found)
			}
			if name == "Missing" {
				return nil // no row to probe
			}
			if share, keyShare := waits(t, pool, tt.id, "FOR SHARE"), waits(t, pool, tt.id, "FOR KEY SHARE"); share != tt.found || keyShare {
				t.Errorf("%s while locked: a FOR SHARE waits %v, a FOR KEY SHARE waits %v; want %v, false", name, share, keyShare, tt.found)
			}
			if pgtest.RowWaits(t, pool, "workspaces", acme, "FOR UPDATE") {
				t.Errorf("%s while locked: a FOR UPDATE of acme waits; want the workspace's row not locked", name)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// A project deleted while LockPlanning waits for its row is not found: Postgres reads deleted_at again on the row's
// newest version after the wait.
func TestLockPlanningAfterAConcurrentDeletion(t *testing.T) {
	s, pool, _, web, _, _ := planningProjects(t)
	holder, err := pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Rollback(context.Background()) }()
	if _, err := holder.Exec(pgtest.Soon(t), "UPDATE projects SET deleted_at = $2 WHERE id = $1", web, now); err != nil {
		t.Fatal(err)
	}
	type result struct {
		found bool
		err   error
	}
	done := make(chan result, 1)
	go func() {
		err := postgres.NewTxManager(pool, 5*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			_, found, err := s.LockPlanning(ctx, web)
			done <- result{found, err}
			return err
		})
		if err != nil {
			done <- result{false, err}
		}
	}()
	if !pgtest.RowWaits(t, pool, "projects", web, "FOR NO KEY UPDATE") {
		t.Fatal("the deletion does not hold the project's row")
	}
	time.Sleep(50 * time.Millisecond)
	select {
	case r := <-done:
		t.Fatalf("LockPlanning returned before the deletion committed: %+v", r)
	default:
	}
	if err := holder.Commit(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
	if r := <-done; r.err != nil || r.found {
		t.Fatalf("LockPlanning after the concurrent deletion = found %v, %v; want not found", r.found, r.err)
	}
}

// The planning module's facts of a project under the lock of a change of one's own display settings on a page (M6 P3b
// spec R6): SharePlanning reads them as PlanningFacts does, an archived project's too, none for a deleted one or one
// that is not there, and holds the project FOR SHARE until the transaction ends: a planning write's or a deletion's FOR
// NO KEY UPDATE waits for it, and so does a FOR UPDATE; another FOR SHARE does not, nor a foreign key's FOR KEY SHARE;
// the workspace's row is not locked.
func TestSharePlanning(t *testing.T) {
	s, pool, acme, web, arch, gone := planningProjects(t)
	for name, tt := range map[string]struct {
		id    uuid.UUID
		want  app.PlanningFacts
		found bool
	}{
		"Web":     {web, app.PlanningFacts{WorkspaceID: acme, Timezone: "Asia/Shanghai", CycleView: true}, true},
		"Arch":    {arch, app.PlanningFacts{WorkspaceID: acme, Archived: true, Timezone: "UTC", ModuleView: true}, true},
		"Gone":    {gone, app.PlanningFacts{}, false},
		"Missing": {uuid.NewV7(), app.PlanningFacts{}, false},
	} {
		err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			f, found, err := s.SharePlanning(ctx, tt.id)
			if err != nil {
				return err
			}
			if found != tt.found || jsonOf(t, f) != jsonOf(t, tt.want) {
				t.Errorf("SharePlanning(%s) = %+v, %v; want %+v, %v", name, f, found, tt.want, tt.found)
			}
			if name == "Missing" {
				return nil // no row to probe
			}
			noKeyUpdate, update := waits(t, pool, tt.id, "FOR NO KEY UPDATE"), waits(t, pool, tt.id, "FOR UPDATE")
			share, keyShare := waits(t, pool, tt.id, "FOR SHARE"), waits(t, pool, tt.id, "FOR KEY SHARE")
			if noKeyUpdate != tt.found || update != tt.found || share || keyShare {
				t.Errorf("%s while shared: FOR NO KEY UPDATE waits %v, FOR UPDATE %v, FOR SHARE %v, FOR KEY SHARE %v; want %v, %v, false, false",
					name, noKeyUpdate, update, share, keyShare, tt.found, tt.found)
			}
			if pgtest.RowWaits(t, pool, "workspaces", acme, "FOR UPDATE") {
				t.Errorf("%s while shared: a FOR UPDATE of acme waits; want the workspace's row not locked", name)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if waits(t, pool, web, "FOR NO KEY UPDATE") {
		t.Error("after the transactions: a FOR NO KEY UPDATE of Web waits; want it free")
	}
}

// A project deleted while SharePlanning waits for its row, under the deletion's FOR NO KEY UPDATE, is not found:
// Postgres reads deleted_at again on the row's newest version after the wait.
func TestSharePlanningAfterAConcurrentDeletion(t *testing.T) {
	s, pool, _, web, _, _ := planningProjects(t)
	holder, err := pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Rollback(context.Background()) }()
	if _, err := holder.Exec(pgtest.Soon(t), "UPDATE projects SET deleted_at = $2 WHERE id = $1", web, now); err != nil {
		t.Fatal(err)
	}
	type result struct {
		found bool
		err   error
	}
	done := make(chan result, 1)
	go func() {
		var r result
		r.err = postgres.NewTxManager(pool, 5*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			var err error
			_, r.found, err = s.SharePlanning(ctx, web)
			return err
		})
		done <- r
	}()
	pgtest.WaitForLockWaitOn(t, pool, "projects", 5*time.Second)
	select {
	case r := <-done:
		t.Fatalf("SharePlanning returned before the deletion committed: %+v", r)
	default:
	}
	if err := holder.Commit(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
	if r := <-done; r.err != nil || r.found {
		t.Fatalf("SharePlanning after the concurrent deletion = found %v, %v; want not found", r.found, r.err)
	}
}
