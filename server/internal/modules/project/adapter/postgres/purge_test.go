package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func purgeID(n int) uuid.UUID { return uuid.MustParse(fmt.Sprintf("0199a2b4-0000-7000-8000-%012x", n)) }

// Each table is nonempty and has an ineligible parent chain for its controls.
// Dropping a binding, using <=, dropping the cursor or ignoring the limit
// loses either a direct deletion or a retained control.
func TestModulePurgeOrderAndCutoff(t *testing.T) {
	for _, table := range []string{"project_user_properties", "project_members", "labels", "states", "projects"} {
		t.Run(table, func(t *testing.T) {
			s, pool := newStore(t)
			ctx := context.Background()
			cutoff := now.Add(-60 * 24 * time.Hour)
			old, older, recent := now.Add(-61*24*time.Hour), now.Add(-62*24*time.Hour), now.Add(-59*24*time.Hour)
			parent := purgeParent(t, s, pool)
			// Lower id has the later deletion time; insertion order is high to low.
			for _, row := range []struct {
				n  int
				at *time.Time
			}{{30, &older}, {10, &old}, {20, &cutoff}, {40, &recent}, {50, nil}} {
				purgeRow(t, s, pool, table, purgeID(row.n), parent, nil, row.at)
			}
			deleteBatch := map[string]func(context.Context, app.PurgeBatch) ([]uuid.UUID, error){"project_user_properties": s.PurgeProjectUserProperties, "project_members": s.PurgeProjectMembers, "labels": s.PurgeLabels, "states": s.PurgeStates, "projects": s.PurgeProjects}[table]
			first, err := deleteBatch(ctx, app.PurgeBatch{Cutoff: cutoff, Limit: 1})
			if err != nil || !slices.Equal(first, []uuid.UUID{purgeID(10)}) {
				t.Fatalf("%s first batch=%v,%v; want lowest eligible id only", table, first, err)
			}
			skipped, err := deleteBatch(ctx, app.PurgeBatch{Cutoff: cutoff, After: purgeID(30), Limit: 1})
			if err != nil || len(skipped) != 0 {
				t.Fatalf("%s after maximum=%v,%v; want empty", table, skipped, err)
			}
			purger := project.Provide(pool).Purger
			if !slices.Equal(purger.Tables(), []string{"project_user_properties", "project_members", "labels", "states", "projects"}) {
				t.Fatalf("directory=%v", purger.Tables())
			}
			count, err := purger.Purge(ctx, cutoff, 1)
			if err != nil || count != 1 {
				t.Fatalf("%s module direct count=%d,%v; want remaining one", table, count, err)
			}
			var left []uuid.UUID
			if err := pool.QueryRow(ctx, "SELECT coalesce(array_agg(id ORDER BY id),'{}') FROM "+table+" WHERE id = ANY($1)", []uuid.UUID{purgeID(10), purgeID(20), purgeID(30), purgeID(40), purgeID(50)}).Scan(&left); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(left, []uuid.UUID{purgeID(20), purgeID(40), purgeID(50)}) {
				t.Fatalf("%s boundary/recent/live=%v; want 20,40,50", table, left)
			}
			for _, limit := range []int{0, 2147483648} {
				invalid := map[string]func(context.Context, app.PurgeBatch) ([]uuid.UUID, error){"project_user_properties": postgresadapter.New(nil).PurgeProjectUserProperties, "project_members": postgresadapter.New(nil).PurgeProjectMembers, "labels": postgresadapter.New(nil).PurgeLabels, "states": postgresadapter.New(nil).PurgeStates, "projects": postgresadapter.New(nil).PurgeProjects}[table]
				if _, err := invalid(ctx, app.PurgeBatch{Cutoff: cutoff, Limit: limit}); err == nil {
					t.Fatalf("%s accepted invalid batch %d", table, limit)
				}
			}
			// Every static adapter must commit independently of the caller's tx.
			// The second statement fails after the first has committed.
			purgeRow(t, s, pool, table, purgeID(60), parent, nil, &old)
			purgeRow(t, s, pool, table, purgeID(70), parent, nil, &old)
			exec(t, pool, `CREATE FUNCTION fail_purge() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF OLD.id='0199a2b4-0000-7000-8000-000000000046'::uuid THEN RAISE EXCEPTION 'purge fixture failure'; END IF; RETURN OLD; END $$`)
			exec(t, pool, "CREATE TRIGGER fail_purge BEFORE DELETE ON "+table+" FOR EACH ROW EXECUTE FUNCTION fail_purge()")
			observer, err := pool.Acquire(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer observer.Release()
			rollback := errors.New("outer rollback")
			var deleted int
			var purgeErr error
			err = postgres.NewTxManager(pool, 2*time.Second).WithinTx(ctx, func(txctx context.Context) error { deleted, purgeErr = purger.Purge(txctx, cutoff, 1); return rollback })
			var pe *pgconn.PgError
			if !errors.Is(err, rollback) || deleted != 1 || !errors.As(purgeErr, &pe) || pe.Message != "purge fixture failure" {
				t.Fatalf("%s independent batches=%d,%v outer=%v; want 1, original PG error, rollback", table, deleted, purgeErr, err)
			}
			var committed, failed bool
			if err := observer.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM "+table+" WHERE id=$1),EXISTS(SELECT 1 FROM "+table+" WHERE id=$2)", purgeID(60), purgeID(70)).Scan(&committed, &failed); err != nil {
				t.Fatal(err)
			}
			if committed || !failed {
				t.Fatalf("%s after caller rollback: committed row exists=%v failed row exists=%v; want false,true", table, committed, failed)
			}
		})
	}
	t.Run("child before parent direct counts", func(t *testing.T) {
		s, pool := newStore(t)
		cutoff, old := now.Add(-60*24*time.Hour), now.Add(-61*24*time.Hour)
		parent := purgeParent(t, s, pool)
		root := purgeID(90)
		purgeRow(t, s, pool, "projects", root, parent, nil, &old)
		purgeRow(t, s, pool, "project_user_properties", purgeID(91), root, nil, &old)
		purgeRow(t, s, pool, "project_members", purgeID(92), root, nil, &old)
		labelParent := purgeID(95)
		purgeRow(t, s, pool, "labels", labelParent, root, nil, &old)
		purgeRow(t, s, pool, "labels", purgeID(93), root, &labelParent, &old)
		purgeRow(t, s, pool, "states", purgeID(94), root, nil, &old)
		count, err := project.Provide(pool).Purger.Purge(context.Background(), cutoff, 1)
		if err != nil || count != 6 {
			t.Fatalf("child-before-parent direct count=%d,%v; want 6 (no CASCADE stealing eligible rows)", count, err)
		}
	})
	t.Run("old parent cascades recent children", func(t *testing.T) {
		s, pool := newStore(t)
		cutoff, old, recent := now.Add(-60*24*time.Hour), now.Add(-61*24*time.Hour), now.Add(-59*24*time.Hour)
		parent := purgeParent(t, s, pool)
		root := purgeID(90)
		purgeRow(t, s, pool, "projects", root, parent, nil, &old)
		purgeRow(t, s, pool, "project_user_properties", purgeID(91), root, nil, &recent)
		purgeRow(t, s, pool, "project_members", purgeID(92), root, nil, &recent)
		labelParent := purgeID(95)
		purgeRow(t, s, pool, "labels", labelParent, root, nil, &recent)
		purgeRow(t, s, pool, "labels", purgeID(93), root, &labelParent, &recent)
		purgeRow(t, s, pool, "states", purgeID(94), root, nil, &recent)
		count, err := project.Provide(pool).Purger.Purge(context.Background(), cutoff, 1)
		if err != nil || count != 1 {
			t.Fatalf("recent CASCADE direct count=%d,%v; want parent only", count, err)
		}
		for _, table := range []string{"project_user_properties", "project_members", "labels", "states", "projects"} {
			var exists bool
			if err := pool.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM "+table+" WHERE id=ANY($1))", []uuid.UUID{purgeID(90), purgeID(91), purgeID(92), purgeID(93), purgeID(94), purgeID(95)}).Scan(&exists); err != nil || exists {
				t.Fatalf("%s cascade rows exist=%v err=%v", table, exists, err)
			}
		}
	})
}

func purgeParent(t *testing.T, s *postgresadapter.Store, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	w := newWorkspace(t, pool, uuid.NewV7().String())
	user := newAccount(t, pool, uuid.NewV7().String()+"@corp.com")
	return newProject(t, s, w, "Parent", "PARENT", user)
}

func purgeRow(t *testing.T, s *postgresadapter.Store, pool *pgxpool.Pool, table string, id, parent uuid.UUID, labelParent *uuid.UUID, deleted *time.Time) {
	t.Helper()
	ctx := context.Background()
	var workspace uuid.UUID
	if err := pool.QueryRow(ctx, "SELECT workspace_id FROM projects WHERE id=$1", parent).Scan(&workspace); err != nil {
		t.Fatal(err)
	}
	user := newAccount(t, pool, uuid.NewV7().String()+"@corp.com")
	name := fmt.Sprintf("P%x", id[10:])
	var err error
	switch table {
	case "projects":
		err = s.CreateProject(ctx, app.ProjectRow{ID: id, WorkspaceID: workspace, Name: name, Identifier: fmt.Sprintf("P%X", id[12:]), Network: domain.NetworkPublic, Timezone: "UTC", CreatedBy: user, Now: now})
	case "project_user_properties":
		err = s.EnsurePreferences(ctx, app.PreferencesRow{ID: id, WorkspaceID: workspace, ProjectID: parent, UserID: user, CreatedBy: user, Now: now})
	case "project_members":
		err = s.CreateMember(ctx, app.MemberRow{ID: id, WorkspaceID: workspace, ProjectID: parent, MemberID: user, Role: shared.RoleMember, CreatedBy: user, Now: now})
	case "labels":
		_, err = s.CreateLabel(ctx, app.LabelRow{ID: id, WorkspaceID: workspace, ProjectID: parent, ParentID: labelParent, Name: name, CreatedBy: user, Now: now})
	case "states":
		_, err = s.CreateState(ctx, app.StateRow{ID: id, WorkspaceID: workspace, ProjectID: parent, State: domain.NewState{Name: name, Color: "#123456", Group: domain.GroupBacklog}, CreatedBy: user, Now: now})
	default:
		t.Fatalf("unknown fixture table %s", table)
	}
	if err != nil {
		t.Fatal(err)
	}
	if deleted != nil {
		exec(t, pool, "UPDATE "+table+" SET deleted_at=$2 WHERE id=$1", id, *deleted)
	}
}
