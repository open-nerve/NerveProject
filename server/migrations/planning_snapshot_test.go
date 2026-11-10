package migrations_test

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// snapshotCycle is the cycle of TestCyclesProgressSnapshotMigration, written before 00048.
const snapshotCycle = "0199a2b4-0000-7000-8000-000000000034"

// A cycle's frozen progress (M6 P4 spec 2.4): Up adds the column, none on a cycle written before it, with its CHECK
// and the freeze job's partial index; Down drops them and keeps the cycle as it was, a frozen one too; Up works once
// more.
func TestCyclesProgressSnapshotMigration(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t, pgtest.NewEmptyDatabase(t))
	before, through := fstest.MapFS{}, fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.FS(), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() > "00048_z" {
			continue
		}
		data, err := fs.ReadFile(migrations.FS(), e.Name())
		if err != nil {
			t.Fatal(err)
		}
		through[e.Name()] = &fstest.MapFile{Data: data}
		if e.Name() <= "00047_z" {
			before[e.Name()] = through[e.Name()]
		}
	}
	m, err := postgres.NewMigrator(pool, before)
	if err != nil {
		t.Fatal(err)
	}
	if up, err := m.Up(ctx); err != nil || len(up) != 47 {
		t.Fatalf("old Up: %v, %v", up, err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		`INSERT INTO users(id,email,password,display_name) VALUES('0199a2b4-0000-7000-8000-000000000031','snapshot@example.com','x','snapshot')`,
		`INSERT INTO workspaces(id,name,slug) VALUES('0199a2b4-0000-7000-8000-000000000032','Snapshot','snapshot')`,
		`INSERT INTO projects(id,workspace_id,name,identifier) VALUES('0199a2b4-0000-7000-8000-000000000033','0199a2b4-0000-7000-8000-000000000032','Snapshot','SNP')`,
		`INSERT INTO cycles(id,workspace_id,project_id,name,start_date,end_date,created_by_id,updated_by_id,created_at,updated_at)
		 VALUES('` + snapshotCycle + `','0199a2b4-0000-7000-8000-000000000032','0199a2b4-0000-7000-8000-000000000033','Sprint',
		 '2026-11-01','2026-11-07','0199a2b4-0000-7000-8000-000000000031','0199a2b4-0000-7000-8000-000000000031',
		 '2026-11-01T08:00:00Z','2026-11-02T08:00:00Z')`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	kept := cycleWithoutSnapshot(t, pool)
	m, err = postgres.NewMigrator(pool, through)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	for pass := range 2 {
		up, err := m.Up(ctx)
		if err != nil || len(up) != 1 || up[0].Version != 48 || up[0].Source != "00048_planning_cycles_progress_snapshot.sql" {
			t.Fatalf("Up %d = %v, %v; want only 48", pass, up, err)
		}
		if got := snapshotSchema(t, pool); !slices.Equal(got, []string{"cycles_freeze_due_idx", "cycles_progress_snapshot_check", "progress_snapshot"}) {
			t.Fatalf("Up %d: %q; want the column, its CHECK and the index", pass, got)
		}
		var none bool
		if err := pool.QueryRow(ctx, "SELECT progress_snapshot IS NULL FROM cycles WHERE id=$1", snapshotCycle).Scan(&none); err != nil || !none {
			t.Fatalf("Up %d: the earlier cycle is frozen (%t, %v); want none", pass, !none, err)
		}
		if got := cycleWithoutSnapshot(t, pool); got != kept {
			t.Fatalf("Up %d changed the cycle:\n%s\nwant\n%s", pass, got, kept)
		}
		_, err = pool.Exec(ctx, "UPDATE cycles SET progress_snapshot='[]' WHERE id=$1", snapshotCycle)
		var pe *pgconn.PgError
		if !errors.As(err, &pe) || pe.Code != "23514" || pe.ConstraintName != "cycles_progress_snapshot_check" {
			t.Fatalf("Up %d: a list stored as a frozen progress: %v", pass, err)
		}
		if pass == 1 {
			break
		}
		if _, err := pool.Exec(ctx, `UPDATE cycles SET progress_snapshot='{"total_issues":1}' WHERE id=$1`, snapshotCycle); err != nil {
			t.Fatal(err)
		}
		down, err := m.Down(ctx)
		if err != nil || down == nil || down.Version != 48 {
			t.Fatalf("Down = %v, %v", down, err)
		}
		if got := snapshotSchema(t, pool); len(got) != 0 {
			t.Fatalf("Down left %q", got)
		}
		if got := cycleWithoutSnapshot(t, pool); got != kept {
			t.Fatalf("Down changed the cycle:\n%s\nwant\n%s", got, kept)
		}
	}
}

// cycleWithoutSnapshot is snapshotCycle's row as JSON, but its frozen progress.
func cycleWithoutSnapshot(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var row string
	if err := pool.QueryRow(context.Background(), "SELECT (to_jsonb(c) - 'progress_snapshot')::text FROM cycles c WHERE id=$1", snapshotCycle).Scan(&row); err != nil {
		t.Fatal(err)
	}
	return row
}

// snapshotSchema are those of 00048's column, CHECK and index on cycles that the schema has, by name.
func snapshotSchema(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	return names(t, pool, `SELECT column_name::text FROM information_schema.columns
 WHERE table_schema='public' AND table_name='cycles' AND column_name='progress_snapshot'
 UNION ALL SELECT conname::text FROM pg_constraint WHERE conrelid='public.cycles'::regclass AND conname='cycles_progress_snapshot_check'
 UNION ALL SELECT indexname::text FROM pg_indexes WHERE schemaname='public' AND tablename='cycles' AND indexname='cycles_freeze_due_idx'
 ORDER BY 1`)
}
