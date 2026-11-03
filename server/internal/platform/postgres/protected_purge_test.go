package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func protectedID(n int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("01990000-0000-7000-8000-%012x", n))
}
func protectedRefs() []postgres.ProtectedReference {
	return []postgres.ProtectedReference{{Schema: "public", Table: "probe_asset", Constraint: "probe_asset_issue_fk"}}
}
func protectedError() error {
	return &pgconn.PgError{Code: "23503", SchemaName: "public", TableName: "probe_asset", ConstraintName: "probe_asset_issue_fk"}
}

type protectedQueries struct{ pool *pgxpool.Pool }

func (q protectedQueries) Candidates(ctx context.Context, cutoff time.Time, after uuid.UUID, limit int) ([]uuid.UUID, error) {
	rows, err := q.pool.Query(ctx, "SELECT id FROM probe_parent WHERE deleted_at < $1 AND id > $2 ORDER BY id LIMIT $3", cutoff, after, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
}
func (q protectedQueries) Delete(ctx context.Context, cutoff time.Time, ids []uuid.UUID) ([]uuid.UUID, error) {
	rows, err := q.pool.Query(ctx, "DELETE FROM probe_parent WHERE deleted_at < $1 AND id = ANY($2) RETURNING id", cutoff, ids)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
}

// The production algorithm runs on a real NO ACTION leaf below two CASCADE
// edges, including a self edge. A failed batch may not consume its middle row.
func TestProtectedPurgeProgress(t *testing.T) {
	pool := newPool(t, pgtest.NewEmptyDatabase(t))
	ctx := pgtest.Soon(t)
	for _, sql := range []string{
		"CREATE TABLE probe_parent(id uuid PRIMARY KEY, deleted_at timestamptz)",
		"CREATE TABLE probe_issue(id uuid PRIMARY KEY, parent_id uuid REFERENCES probe_parent ON DELETE CASCADE, issue_id uuid REFERENCES probe_issue ON DELETE CASCADE)",
		"CREATE TABLE probe_asset(id int PRIMARY KEY, issue_id uuid CONSTRAINT probe_asset_issue_fk REFERENCES probe_issue ON DELETE NO ACTION NOT DEFERRABLE)",
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	cutoff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, n := range []int{3, 1, 2} {
		if _, err := pool.Exec(ctx, "INSERT INTO probe_parent VALUES($1,$2);", protectedID(n), cutoff.Add(-time.Hour)); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, "INSERT INTO probe_issue VALUES($1,$2,NULL)", protectedID(n*10), protectedID(n)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, "INSERT INTO probe_issue VALUES($1,$2,$3)", protectedID(11), protectedID(1), protectedID(10)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO probe_asset VALUES(1,$1),(2,$2)", protectedID(11), protectedID(30)); err != nil {
		t.Fatal(err)
	}
	q := protectedQueries{pool}
	_, err := q.Delete(ctx, cutoff, []uuid.UUID{protectedID(1), protectedID(2), protectedID(3)})
	if !postgres.ProtectedForeignKey(err, protectedRefs()) {
		t.Fatalf("leaf FK identity lost: %v", err)
	}
	var before int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM probe_parent").Scan(&before); err != nil || before != 3 {
		t.Fatalf("bulk rollback: %d %v", before, err)
	}
	got, err := postgres.PurgeProtected(ctx, q, protectedRefs(), cutoff, uuid.Nil(), 3)
	want := postgres.PurgePage{Scanned: 3, Deleted: 1, Blocked: 2, After: protectedID(3)}
	if err != nil || got != want {
		t.Fatalf("protected page=%+v %v; want %+v", got, err, want)
	}
	rows, err := q.Candidates(ctx, cutoff, uuid.Nil(), 3)
	if err != nil || !slices.Equal(rows, []uuid.UUID{protectedID(1), protectedID(3)}) {
		t.Fatalf("survivors=%v %v", rows, err)
	}
	end, err := postgres.PurgeProtected(ctx, q, protectedRefs(), cutoff, got.After, 3)
	if err != nil || end.Scanned != 0 || end.After != got.After {
		t.Fatalf("end=%+v %v", end, err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM probe_asset"); err != nil {
		t.Fatal(err)
	}
	recovered, err := postgres.PurgeProtected(ctx, q, protectedRefs(), cutoff, uuid.Nil(), 3)
	if err != nil || recovered.Scanned != 2 || recovered.Deleted != 2 || recovered.Blocked != 0 {
		t.Fatalf("recovery=%+v %v", recovered, err)
	}
}

func TestProtectedPurgeExactForeignKey(t *testing.T) {
	base := protectedError().(*pgconn.PgError)
	for _, field := range []string{"exact", "wrapped", "schema", "table", "constraint", "23505", "23001", "40P01"} {
		t.Run(field, func(t *testing.T) {
			p := *base
			switch field {
			case "schema":
				p.SchemaName = "other"
			case "table":
				p.TableName = "other"
			case "constraint":
				p.ConstraintName = "other"
			case "23505", "23001", "40P01":
				p.Code = field
			}
			var err error = &p
			if field == "wrapped" {
				err = fmt.Errorf("owner: %w", err)
			}
			if got := postgres.ProtectedForeignKey(err, protectedRefs()); got != (field == "exact" || field == "wrapped") {
				t.Fatalf("classification %t: %v", got, err)
			}
		})
	}
	for _, err := range []error{nil, errors.New("23503 probe_asset_issue_fk"), (*pgconn.PgError)(nil)} {
		if postgres.ProtectedForeignKey(err, protectedRefs()) {
			t.Fatal("accepted non PG failure")
		}
	}
}
