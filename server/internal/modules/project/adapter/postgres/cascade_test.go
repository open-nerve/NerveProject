package postgresadapter_test

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// deletion is a row's audit columns, as the tests read them.
type deletion struct {
	deletedAt *time.Time
	updatedAt time.Time
	updatedBy *uuid.UUID
}

// deletedAtBy reports whether d was deleted at when by by.
func (d deletion) deletedAtBy(when time.Time, by uuid.UUID) bool {
	return d.deletedAt != nil && d.deletedAt.Equal(when) && d.updatedAt.Equal(when) && d.updatedBy != nil && *d.updatedBy == by
}

// projectTables are the tables a workspace's deletion deletes the projects'
// rows of, in its order.
var projectTables = []string{"projects", "project_members", "project_user_properties", "states"}

// deletions reads the audit columns of the workspace's rows of each project
// table, keyed by table then id.
func deletions(t *testing.T, pool *pgxpool.Pool, workspace uuid.UUID) map[string]map[uuid.UUID]deletion {
	t.Helper()
	out := map[string]map[uuid.UUID]deletion{}
	for _, table := range projectTables {
		rows, err := pool.Query(context.Background(),
			"SELECT id, deleted_at, updated_at, updated_by_id FROM "+table+" WHERE workspace_id = $1", workspace)
		if err != nil {
			t.Fatal(err)
		}
		out[table] = map[uuid.UUID]deletion{}
		for rows.Next() {
			var id uuid.UUID
			var d deletion
			if err := rows.Scan(&id, &d.deletedAt, &d.updatedAt, &d.updatedBy); err != nil {
				t.Fatal(err)
			}
			out[table][id] = d
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

// seedProject writes a project of workspace named name, by alice at now,
// with alice's and bob's memberships and display settings in it, and a
// backlog state and the triage state, and returns its id. The rows are
// written directly: the delete statements read no column the inserts of
// the use cases would set otherwise.
func seedProject(t *testing.T, pool *pgxpool.Pool, workspace uuid.UUID, name string, alice, bob uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, `INSERT INTO projects (id, workspace_id, name, identifier, created_by_id, updated_by_id, created_at, updated_at)
		VALUES ($1, $2, $3::text, upper($3::text), $4, $4, $5, $5)`, id, workspace, name, alice, now)
	for _, user := range []uuid.UUID{alice, bob} {
		exec(t, pool, `INSERT INTO project_members (id, workspace_id, project_id, member_id, role, created_by_id, updated_by_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 20, $5, $5, $6, $6)`, uuid.NewV7(), workspace, id, user, alice, now)
		exec(t, pool, `INSERT INTO project_user_properties (id, workspace_id, project_id, user_id, created_by_id, updated_by_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $5, $6, $6)`, uuid.NewV7(), workspace, id, user, alice, now)
	}
	for _, group := range []string{"backlog", "triage"} {
		exec(t, pool, `INSERT INTO states (id, workspace_id, project_id, name, color, "group", created_by_id, updated_by_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, '#60646C', $4, $5, $5, $6, $6)`, uuid.NewV7(), workspace, id, group, alice, now)
	}
	return id
}

// seedProjects writes two projects of workspace (seedProject), the second
// archived and bob's membership of it inactive.
func seedProjects(t *testing.T, pool *pgxpool.Pool, workspace, alice, bob uuid.UUID) {
	t.Helper()
	seedProject(t, pool, workspace, "Web", alice, bob)
	ops := seedProject(t, pool, workspace, "Ops", alice, bob)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", ops, now)
	exec(t, pool, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2", ops, bob)
}

// The four steps, in one transaction, soft-delete the workspace's
// projects, archived ones too, every membership of them, active or not,
// every member's display settings in them and every state, the triage ones
// too, at the same moment and by the same account; a row deleted before
// keeps its time, and another workspace keeps everything. Running the
// steps again changes nothing.
func TestDeletingAWorkspaceSoftDeletesItsProjects(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	seedProjects(t, pool, acme, alice, bob)
	seedProjects(t, pool, beta, alice, bob)
	// A project of acme deleted before, with the rows under it.
	earlier, later := now.Add(-time.Hour), now.Add(time.Hour)
	old := seedProject(t, pool, acme, "Old", alice, bob)
	for _, table := range projectTables {
		column := map[string]string{"projects": "id"}[table]
		if column == "" {
			column = "project_id"
		}
		exec(t, pool, "UPDATE "+table+" SET deleted_at = $2 WHERE "+column+" = $1", old, earlier)
	}
	steps := []func(ctx context.Context, id, by uuid.UUID, now time.Time) error{
		s.DeleteWorkspaceProjects, s.DeleteWorkspaceProjectMembers, s.DeleteWorkspaceProjectPreferences, s.DeleteWorkspaceStates,
	}
	run := func(ctx context.Context, by uuid.UUID, at time.Time) error {
		for _, step := range steps {
			if err := step(ctx, acme, by, at); err != nil {
				return err
			}
		}
		return nil
	}

	if err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		return run(ctx, bob, later)
	}); err != nil {
		t.Fatal(err)
	}
	// Once more, by alice an hour later: nothing is left undeleted.
	if err := run(context.Background(), alice, later.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	perProject := map[string]int{"projects": 1, "project_members": 2, "project_user_properties": 2, "states": 2}
	for table, rows := range deletions(t, pool, acme) {
		deleted, before := 0, 0
		for id, d := range rows {
			switch {
			case d.deletedAtBy(later, bob):
				deleted++
			case d.deletedAt != nil && d.deletedAt.Equal(earlier) && d.updatedAt.Equal(now):
				before++
			default:
				t.Errorf("acme's %s %s: %+v, want deleted at %v by bob, or at %v as before", table, id, d, later, earlier)
			}
		}
		if deleted != 2*perProject[table] || before != perProject[table] {
			t.Errorf("acme's %s: %d deleted by bob, %d deleted before; want %d, %d", table, deleted, before, 2*perProject[table], perProject[table])
		}
	}
	var active []bool
	if err := pool.QueryRow(context.Background(), `SELECT array_agg(is_active ORDER BY is_active) FROM project_members
		WHERE workspace_id = $1 AND deleted_at = $2`, acme, later).Scan(&active); err != nil || len(active) != 4 || active[0] || !active[1] {
		t.Errorf("is_active of acme's deleted memberships: %v, %v; want one false, three true, as they were", active, err)
	}
	for table, rows := range deletions(t, pool, beta) {
		if len(rows) != 2*perProject[table] {
			t.Errorf("beta's %s: %d rows, want %d", table, len(rows), 2*perProject[table])
		}
		for id, d := range rows {
			if d.deletedAt != nil || !d.updatedAt.Equal(now) {
				t.Errorf("beta's %s %s: %+v, want it untouched", table, id, d)
			}
		}
	}
}
