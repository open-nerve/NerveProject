package postgresadapter_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
)

// moduleTables are the module's tables in the purge's order, children first.
var moduleTables = []string{"module_links", "module_members", "modules"}

// moduleRow inserts the row id of table, one of moduleTables, in project, deleted at deleted when it is not nil. A
// member or a link hangs from a live module of its own, named after the row, so rows of one table never share a
// module (a member or an address is unique in its module).
func moduleRow(t *testing.T, pool *pgxpool.Pool, w world, table string, project, id uuid.UUID, deleted *time.Time) {
	t.Helper()
	module := id
	if table != "modules" {
		module = uuid.NewV7()
		exec(t, pool, `INSERT INTO modules (id, workspace_id, project_id, name, created_by_id, updated_by_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $5, $6, $6)`, module, w.workspace, project, module.String(), w.by, now.Add(-time.Hour))
	}
	switch table {
	case "modules":
		exec(t, pool, `INSERT INTO modules (id, workspace_id, project_id, name, created_by_id, updated_by_id, created_at, updated_at, deleted_at)
			VALUES ($1, $2, $3, $4, $5, $5, $6, $6, $7)`, id, w.workspace, project, id.String(), w.by, now.Add(-time.Hour), deleted)
	case "module_members":
		exec(t, pool, `INSERT INTO module_members (id, workspace_id, project_id, module_id, member_id, created_by_id, updated_by_id, created_at, updated_at, deleted_at)
			VALUES ($1, $2, $3, $4, $5, $5, $5, $6, $6, $7)`, id, w.workspace, project, module, w.by, now.Add(-time.Hour), deleted)
	case "module_links":
		exec(t, pool, `INSERT INTO module_links (id, workspace_id, project_id, module_id, url, created_by_id, updated_by_id, created_at, updated_at, deleted_at)
			VALUES ($1, $2, $3, $4, $5, $6, $6, $7, $7, $8)`, id, w.workspace, project, module, "https://nerve.dev/"+id.String(), w.by, now.Add(-time.Hour), deleted)
	default:
		t.Fatalf("no module table %s", table)
	}
}

// tableRows is table's rows of project, each as JSON, in id order.
func tableRows(t *testing.T, pool *pgxpool.Pool, table string, project uuid.UUID) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(context.Background(), "SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id),'[]')::text FROM "+table+" r WHERE project_id=$1", project).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// Each module table's step soft-deletes, a batch at a time after its cursor and in id order, at most its limit of the
// project's undeleted rows at the batch's moment and by its actor; a row deleted before keeps its own deletion, the
// other project's rows are untouched.
func TestDeleteProjectModuleRows(t *testing.T) {
	for _, table := range moduleTables {
		t.Run(table, func(t *testing.T) {
			s, pool := newStore(t)
			ctx := context.Background()
			w := newWorld(t, pool)
			live := sortedIDs(3)
			for _, id := range live {
				moduleRow(t, pool, w, table, w.web, id, nil)
			}
			earlier := now.Add(-2 * time.Hour)
			gone := uuid.NewV7()
			moduleRow(t, pool, w, table, w.web, gone, &earlier)
			moduleRow(t, pool, w, table, w.ops, uuid.NewV7(), nil)
			ops := tableRows(t, pool, table, w.ops)
			deleter := uuid.NewV7()
			exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, 'dave@corp.com', 'x', 'dave')", deleter)

			step := map[string]func(context.Context, app.RowBatch) ([]uuid.UUID, error){
				"modules": s.DeleteProjectModules, "module_members": s.DeleteProjectModuleMembers, "module_links": s.DeleteProjectModuleLinks,
			}[table]
			batch := app.RowBatch{ProjectID: w.web, Limit: 2, At: now, By: deleter}
			var got [][]uuid.UUID
			for {
				ids, err := step(ctx, batch)
				if err != nil {
					t.Fatal(err)
				}
				slices.SortFunc(ids, uuid.UUID.Compare)
				got = append(got, ids)
				if len(ids) == 0 {
					break
				}
				batch.After = ids[len(ids)-1]
			}
			if want := [][]uuid.UUID{live[:2], live[2:], nil}; fmt.Sprint(got) != fmt.Sprint(want) {
				t.Fatalf("batches = %v, want %v", got, want)
			}
			var stamped, kept int
			if err := pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE deleted_at=$2 AND updated_at=$2 AND updated_by_id=$3 AND id=ANY($7)),
				count(*) FILTER (WHERE id=$4 AND deleted_at=$5 AND updated_by_id=$6) FROM `+table+` WHERE project_id=$1`,
				w.web, now, deleter, gone, earlier, w.by, live).Scan(&stamped, &kept); err != nil || stamped != 3 || kept != 1 {
				t.Fatalf("stamped=%d kept=%d, %v; want 3 and the earlier deletion kept", stamped, kept, err)
			}
			if tableRows(t, pool, table, w.ops) != ops {
				t.Fatal("the other project's rows changed")
			}
		})
	}
}

// The remainder probe reads, without writing, its step after its cursor and every later step whole; the steps before
// it are done and not read, nor are the other project's rows. An unknown step is an error.
func TestProjectDeletionRemainingFromItsStep(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	cycle(t, pool, w, w.web, uuid.NewV7(), nil)
	link := uuid.NewV7()
	moduleRow(t, pool, w, "module_links", w.web, link, nil)
	snapshot := func() string {
		return rowsOf(t, pool, w.web) + tableRows(t, pool, "modules", w.web) + tableRows(t, pool, "module_links", w.web)
	}
	before := snapshot()
	for _, tt := range []struct {
		r    app.ProjectRemainder
		want bool
	}{
		{app.ProjectRemainder{ProjectID: w.web, Step: "cycles"}, true},
		{app.ProjectRemainder{ProjectID: w.web, Step: "cycles", After: uuid.Max()}, true},
		{app.ProjectRemainder{ProjectID: w.web, Step: "module_members", After: uuid.Max()}, true},
		{app.ProjectRemainder{ProjectID: w.web, Step: "module_links", After: link}, false},
		{app.ProjectRemainder{ProjectID: w.ops, Step: "cycles"}, false},
	} {
		if got, err := s.ProjectDeletionRemaining(ctx, tt.r); err != nil || got != tt.want {
			t.Errorf("remaining %+v = %t, %v; want %t", tt.r, got, err, tt.want)
		}
	}
	if snapshot() != before {
		t.Error("the probe wrote")
	}
	exec(t, pool, "UPDATE module_links SET deleted_at=now() WHERE id=$1", link)
	// The live cycle and the link's live module are before the members' step.
	if got, err := s.ProjectDeletionRemaining(ctx, app.ProjectRemainder{ProjectID: w.web, Step: "module_members"}); err != nil || got {
		t.Errorf("remaining from the members after the link's deletion = %t, %v; want false", got, err)
	}
	if _, err := s.ProjectDeletionRemaining(ctx, app.ProjectRemainder{ProjectID: w.web, Step: "issues"}); err == nil {
		t.Error("an unknown step was read")
	}
}

// Each module table's purge deletes, a bounded page at a time after its cursor, its rows deleted before the cutoff;
// the row deleted at the cutoff, a recent one and a live one stay. Dropping the cursor, the limit or using <= fails
// here.
func TestPurgeModuleTables(t *testing.T) {
	for _, table := range moduleTables {
		t.Run(table, func(t *testing.T) {
			s, pool := newStore(t)
			ctx := context.Background()
			w := newWorld(t, pool)
			cutoff := now.Add(-60 * 24 * time.Hour)
			old, older, recent := now.Add(-61*24*time.Hour), now.Add(-62*24*time.Hour), now.Add(-59*24*time.Hour)
			for _, row := range []struct {
				n  int
				at *time.Time
			}{{30, &older}, {10, &old}, {20, &cutoff}, {40, &recent}, {50, nil}} {
				moduleRow(t, pool, w, table, w.web, purgeID(row.n), row.at)
			}
			page := map[string]func(context.Context, app.PurgeBatch) (app.PurgePage, error){
				"modules": s.PurgeModules, "module_members": s.PurgeModuleMembers, "module_links": s.PurgeModuleLinks,
			}[table]
			first, err := page(ctx, app.PurgeBatch{Cutoff: cutoff, Limit: 1})
			if err != nil || first.Scanned != 1 || first.Deleted != 1 || first.After != purgeID(10) {
				t.Fatalf("first batch=%+v,%v; want the lowest eligible id only", first, err)
			}
			skipped, err := page(ctx, app.PurgeBatch{Cutoff: cutoff, After: purgeID(30), Limit: 1})
			if err != nil || skipped.Scanned != 0 || skipped.Deleted != 0 || skipped.After != purgeID(30) {
				t.Fatalf("after the greatest=%+v,%v; want empty", skipped, err)
			}
			if deleted := purgeAll(t, planning.Provide(pool).Purger, cutoff); deleted != 1 {
				t.Fatalf("module purge deleted %d; want the remaining one", deleted)
			}
			var left []uuid.UUID
			if err := pool.QueryRow(ctx, "SELECT coalesce(array_agg(id ORDER BY id),'{}') FROM "+table+" WHERE id=ANY($1)",
				[]uuid.UUID{purgeID(10), purgeID(20), purgeID(30), purgeID(40), purgeID(50)}).Scan(&left); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(left, []uuid.UUID{purgeID(20), purgeID(40), purgeID(50)}) {
				t.Fatalf("boundary/recent/live=%v; want 20,40,50", left)
			}
			invalid := map[string]func(context.Context, app.PurgeBatch) (app.PurgePage, error){
				"modules": postgresadapter.New(nil).PurgeModules, "module_members": postgresadapter.New(nil).PurgeModuleMembers,
				"module_links": postgresadapter.New(nil).PurgeModuleLinks,
			}[table]
			for _, limit := range []int{0, 2147483648} {
				if _, err := invalid(ctx, app.PurgeBatch{Cutoff: cutoff, Limit: limit}); err == nil {
					t.Fatalf("accepted invalid batch %d", limit)
				}
			}
		})
	}
}

// The module purges its tables children first: a module deleted before the cutoff with its member and link, and a
// cycle, are each deleted by their own table's page, none taken by a parent's CASCADE first.
func TestModulePurgeChildrenFirst(t *testing.T) {
	_, pool := newStore(t)
	w := newWorld(t, pool)
	purger := planning.Provide(pool).Purger
	cutoff, old := now.Add(-60*24*time.Hour), now.Add(-61*24*time.Hour)
	module := uuid.NewV7()
	exec(t, pool, `INSERT INTO modules (id, workspace_id, project_id, name, deleted_at) VALUES ($1, $2, $3, 'Gone', $4)`, module, w.workspace, w.web, old)
	exec(t, pool, `INSERT INTO module_members (id, workspace_id, project_id, module_id, member_id, deleted_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.NewV7(), w.workspace, w.web, module, w.by, old)
	exec(t, pool, `INSERT INTO module_links (id, workspace_id, project_id, module_id, url, deleted_at) VALUES ($1, $2, $3, $4, 'https://nerve.dev', $5)`,
		uuid.NewV7(), w.workspace, w.web, module, old)
	cycle(t, pool, w, w.web, uuid.NewV7(), &old)
	if deleted := purgeAll(t, purger, cutoff); deleted != 4 {
		t.Errorf("direct deletions=%d; want 4, none taken by a parent's CASCADE", deleted)
	}
	if !slices.Equal(purger.Tables(), []string{"module_links", "module_members", "modules", "cycles"}) {
		t.Errorf("directory=%v", purger.Tables())
	}
}

// purgeAll pages purger to its end a row at a time, counting its own deletions.
func purgeAll(t *testing.T, purger *app.Purger, cutoff time.Time) int {
	t.Helper()
	deleted := 0
	cursor := planning.PurgeCursor{}
	for {
		p, err := purger.Page(context.Background(), cutoff, 1, cursor)
		if err != nil {
			t.Fatal(err)
		}
		deleted += p.Deleted
		if p.Done {
			return deleted
		}
		cursor = p.Cursor
	}
}
