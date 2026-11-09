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
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

var now = time.Date(2026, 11, 7, 9, 0, 0, 0, time.UTC)

func newStore(t *testing.T) (*postgresadapter.Store, *pgxpool.Pool) {
	t.Helper()
	pool, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: pgtest.NewDatabase(t), MaxConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return postgresadapter.New(pool), pool
}

func exec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

// world is an account, a workspace and two of its projects, written directly: identity, workspace and project own
// those tables, and the tests write them as fixtures.
type world struct{ by, workspace, web, ops uuid.UUID }

func newWorld(t *testing.T, pool *pgxpool.Pool) world {
	t.Helper()
	w := world{uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()}
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, 'bob@corp.com', 'x', 'bob')", w.by)
	exec(t, pool, "INSERT INTO workspaces (id, name, slug) VALUES ($1, 'Acme', 'acme')", w.workspace)
	exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $3, 'Web', 'WEB'), ($2, $3, 'Ops', 'OPS')", w.web, w.ops, w.workspace)
	return w
}

// cycle inserts the cycle id of project, deleted at deleted when it is not nil.
func cycle(t *testing.T, pool *pgxpool.Pool, w world, project, id uuid.UUID, deleted *time.Time) {
	t.Helper()
	exec(t, pool, `INSERT INTO cycles (id, workspace_id, project_id, name, created_by_id, updated_by_id, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, 'Sprint', $4, $4, $5, $5, $6)`, id, w.workspace, project, w.by, now.Add(-time.Hour), deleted)
}

func rowsOf(t *testing.T, pool *pgxpool.Pool, project uuid.UUID) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(context.Background(), "SELECT coalesce(jsonb_agg(to_jsonb(c) ORDER BY id),'[]')::text FROM cycles c WHERE project_id=$1", project).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func sortedIDs(n int) []uuid.UUID {
	out := make([]uuid.UUID, n)
	for i := range out {
		out[i] = uuid.NewV7()
	}
	slices.SortFunc(out, uuid.UUID.Compare)
	return out
}

// A batch soft-deletes at most its limit of the project's undeleted cycles after its cursor, in id order, at the
// batch's moment and by its actor; a cycle deleted before keeps its own deletion, the other project's are untouched.
// Remaining reads the same candidates without writing.
func TestDeleteProjectCycles(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	live := sortedIDs(3)
	for _, id := range live {
		cycle(t, pool, w, w.web, id, nil)
	}
	earlier := now.Add(-2 * time.Hour)
	gone := uuid.NewV7()
	cycle(t, pool, w, w.web, gone, &earlier)
	cycle(t, pool, w, w.ops, uuid.NewV7(), nil)
	ops := rowsOf(t, pool, w.ops)
	deleter := uuid.NewV7()
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, 'dave@corp.com', 'x', 'dave')", deleter)

	if pending, err := s.ProjectDeletionRemaining(ctx, app.ProjectRemainder{ProjectID: w.web, Step: "cycles"}); err != nil || !pending {
		t.Fatalf("remaining before = %t, %v", pending, err)
	}
	batch := app.RowBatch{ProjectID: w.web, Limit: 2, At: now, By: deleter}
	var got [][]uuid.UUID
	for {
		ids, err := s.DeleteProjectCycles(ctx, batch)
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
	if pending, err := s.ProjectDeletionRemaining(ctx, app.ProjectRemainder{ProjectID: w.web, Step: "cycles"}); err != nil || pending {
		t.Fatalf("remaining after = %t, %v", pending, err)
	}
	var stamped, kept int
	if err := pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE deleted_at=$2 AND updated_at=$2 AND updated_by_id=$3),
		count(*) FILTER (WHERE id=$4 AND deleted_at=$5 AND updated_by_id=$6) FROM cycles WHERE project_id=$1`, w.web, now, deleter, gone, earlier, w.by).Scan(&stamped, &kept); err != nil || stamped != 3 || kept != 1 {
		t.Fatalf("stamped=%d kept=%d, %v; want 3 and the earlier deletion kept", stamped, kept, err)
	}
	if rowsOf(t, pool, w.ops) != ops {
		t.Fatal("the other project's cycles changed")
	}
	// The remainder reads after its cursor only.
	cycle(t, pool, w, w.web, uuid.NewV7(), nil)
	if pending, err := s.ProjectDeletionRemaining(ctx, app.ProjectRemainder{ProjectID: w.web, Step: "cycles", After: uuid.Max()}); err != nil || pending {
		t.Fatalf("remaining after the greatest id = %t, %v", pending, err)
	}
}

func purgeID(n int) uuid.UUID { return uuid.MustParse(fmt.Sprintf("0199a2b4-0000-7000-8000-%012x", n)) }

// The purge deletes, a bounded page at a time after its cursor, the cycles deleted before the cutoff; the cycle
// deleted at the cutoff, a recent one and a live one stay. Dropping the cursor, the limit or using <= fails here.
func TestPurgeCycles(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	cutoff := now.Add(-60 * 24 * time.Hour)
	old, older, recent := now.Add(-61*24*time.Hour), now.Add(-62*24*time.Hour), now.Add(-59*24*time.Hour)
	for _, row := range []struct {
		n  int
		at *time.Time
	}{{30, &older}, {10, &old}, {20, &cutoff}, {40, &recent}, {50, nil}} {
		cycle(t, pool, w, w.web, purgeID(row.n), row.at)
	}
	first, err := s.PurgeCycles(ctx, app.PurgeBatch{Cutoff: cutoff, Limit: 1})
	if err != nil || first.Scanned != 1 || first.Deleted != 1 || first.After != purgeID(10) {
		t.Fatalf("first batch=%+v,%v; want the lowest eligible id only", first, err)
	}
	skipped, err := s.PurgeCycles(ctx, app.PurgeBatch{Cutoff: cutoff, After: purgeID(30), Limit: 1})
	if err != nil || skipped.Scanned != 0 || skipped.Deleted != 0 || skipped.After != purgeID(30) {
		t.Fatalf("after the greatest=%+v,%v; want empty", skipped, err)
	}
	purger := planning.Provide(pool, nil).Purger
	if !slices.Equal(purger.Tables(), []string{"module_user_properties", "module_links", "module_members", "modules", "cycle_user_properties", "cycles"}) {
		t.Fatalf("directory=%v", purger.Tables())
	}
	deleted := 0
	cursor := planning.PurgeCursor{}
	for {
		p, err := purger.Page(ctx, cutoff, 1, cursor)
		if err != nil {
			t.Fatal(err)
		}
		deleted += p.Deleted
		if p.Done {
			break
		}
		cursor = p.Cursor
	}
	if deleted != 1 {
		t.Fatalf("module purge deleted %d; want the remaining one", deleted)
	}
	var left []uuid.UUID
	if err := pool.QueryRow(ctx, "SELECT coalesce(array_agg(id ORDER BY id),'{}') FROM cycles").Scan(&left); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(left, []uuid.UUID{purgeID(20), purgeID(40), purgeID(50)}) {
		t.Fatalf("boundary/recent/live=%v; want 20,40,50", left)
	}
	for _, limit := range []int{0, 2147483648} {
		if _, err := postgresadapter.New(nil).PurgeCycles(ctx, app.PurgeBatch{Cutoff: cutoff, Limit: limit}); err == nil {
			t.Fatalf("accepted invalid batch %d", limit)
		}
	}
}
