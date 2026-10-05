package postgresadapter_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// now is the fixed clock's time: whole microseconds, as timestamptz stores
// them, so the audit columns read back equal to it (M2 design 3.13).
var now = clocktest.At(time.Date(2026, 10, 1, 10, 0, 0, 123456789, time.UTC)).Now()

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

// newAccount inserts the users row that the project tables reference.
// identity owns the table: the test writes it directly, as a fixture.
func newAccount(t *testing.T, pool *pgxpool.Pool, email string) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, $2, 'x', 'x')", id, email)
	return id
}

// newWorkspace inserts the workspaces row the project tables reference.
// workspace owns the table: the test writes it directly, as a fixture.
func newWorkspace(t *testing.T, pool *pgxpool.Pool, slug string) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, "INSERT INTO workspaces (id, name, slug) VALUES ($1, $2, $2)", id, slug)
	return id
}

// newProject stores a public project of workspace named name, with the
// identifier identifier, created by by at now, and returns its id.
func newProject(t *testing.T, s *postgresadapter.Store, workspace uuid.UUID, name, identifier string, by uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	if err := s.CreateProject(context.Background(), app.ProjectRow{
		ID: id, WorkspaceID: workspace, Name: name, Identifier: identifier, Network: domain.NetworkPublic, Timezone: "UTC",
		CreatedBy: by, Now: now,
	}); err != nil {
		t.Fatal(err)
	}
	return id
}

// tableRows is every row of table as text but the rows ids, in order:
// what a write of the rows ids must leave as it was.
func tableRows(t *testing.T, pool *pgxpool.Pool, table string, ids ...uuid.UUID) string {
	t.Helper()
	var s string
	if err := pool.QueryRow(context.Background(), "SELECT coalesce(string_agg(r::text, E'\\n' ORDER BY r::text), '') FROM "+table+
		" r WHERE r.id <> ALL (coalesce($1::uuid[], '{}'))", ids).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// rowsBut is every row of table as text, by id: the one id without the
// columns cols, which a write of it writes.
func rowsBut(t *testing.T, pool *pgxpool.Pool, table string, id uuid.UUID, cols ...string) string {
	t.Helper()
	var rows string
	if err := pool.QueryRow(context.Background(), `SELECT coalesce(string_agg(CASE WHEN r.id = $1 THEN (to_jsonb(r) - $2::text[])::text
		ELSE r::text END, E'\n' ORDER BY r.id), '') FROM `+table+` r`, id, cols).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}
