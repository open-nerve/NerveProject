package postgresadapter_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// now is the fixed clock's time: whole microseconds, as timestamptz stores
// them, so the audit columns read back equal to it (M2 design 3.13).
var now = clocktest.At(time.Date(2026, 10, 17, 10, 0, 0, 123456789, time.UTC)).Now()

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

// newAccount inserts the users row that the issue tables reference.
// identity owns the table: the test writes it directly, as a fixture.
func newAccount(t *testing.T, pool *pgxpool.Pool, email string) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, $2, 'x', 'x')", id, email)
	return id
}

// newWorkspace inserts the workspaces row the issue tables reference.
// workspace owns the table: the test writes it directly, as a fixture.
func newWorkspace(t *testing.T, pool *pgxpool.Pool, slug string) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, "INSERT INTO workspaces (id, name, slug) VALUES ($1, $2, $2)", id, slug)
	return id
}

// project is a project the issue tables reference, with its default
// state: project owns the tables, the test writes them directly, as a
// fixture.
type project struct {
	workspace, id, state uuid.UUID
}

// newProject inserts a project of workspace with the identifier, and its
// default state.
func newProject(t *testing.T, pool *pgxpool.Pool, workspace uuid.UUID, identifier string) project {
	t.Helper()
	p := project{workspace: workspace, id: uuid.NewV7(), state: uuid.NewV7()}
	exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, $3, $3)", p.id, workspace, identifier)
	exec(t, pool, `INSERT INTO states (id, workspace_id, project_id, name, color, "default") VALUES ($1, $2, $3, 'Backlog', '#60646C', true)`,
		p.state, workspace, p.id)
	return p
}

// newLabel inserts a label of p named name and returns its id. project
// owns the table: the test writes it directly, as a fixture.
func newLabel(t *testing.T, pool *pgxpool.Pool, p project, name string) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, "INSERT INTO labels (id, workspace_id, project_id, name) VALUES ($1, $2, $3, $4)", id, p.workspace, p.id, name)
	return id
}

// newIssue inserts the work item id of p, numbered sequence, in p's
// default state, by by at now.
func newIssue(t *testing.T, pool *pgxpool.Pool, p project, id uuid.UUID, sequence int, by uuid.UUID) {
	t.Helper()
	exec(t, pool, `INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id, created_by_id, updated_by_id, created_at,
		updated_at) VALUES ($1, $2, $3, $4, 'Work', $5, $6, $6, $7, $7)`, id, p.workspace, p.id, p.state, sequence, by, now)
}
