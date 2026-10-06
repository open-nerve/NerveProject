package postgresadapter_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// deletion is a row's audit columns, as the tests read them, and the
// project the row is under.
type deletion struct {
	project   uuid.UUID
	deletedAt *time.Time
	updatedAt time.Time
	updatedBy *uuid.UUID
}

// deletedAtBy reports whether d was deleted at when by by.
func (d deletion) deletedAtBy(when time.Time, by uuid.UUID) bool {
	return d.deletedAt != nil && d.deletedAt.Equal(when) && d.updatedAt.Equal(when) && d.updatedBy != nil && *d.updatedBy == by
}

// String is d as a failure prints it: the times in RFC 3339 to the
// microsecond and the account's id, each null when the column is.
func (d deletion) String() string {
	by := "null"
	if d.updatedBy != nil {
		by = d.updatedBy.String()
	}
	return fmt.Sprintf("deleted_at %s, updated_at %s, updated_by %s", instant(d.deletedAt), instant(&d.updatedAt), by)
}

// instant is *t in RFC 3339 to the microsecond, as timestamptz keeps it, or
// null.
func instant(t *time.Time) string {
	if t == nil {
		return "null"
	}
	return t.UTC().Format("2006-01-02T15:04:05.000000Z07:00")
}

// projectTables are the tables a deletion of projects deletes the rows of,
// in its order, each with its column of the project and how many rows of
// it seedProject writes under each project. A table here without a row
// under a project fails checkDeletions: seed it.
var projectTables = []struct {
	name, project string
	perProject    int
}{
	{"projects", "id", 1}, {"project_members", "project_id", 2}, {"project_user_properties", "project_id", 2}, {"states", "project_id", 2},
	{"labels", "project_id", 2},
}

// deletions reads the audit columns of the workspace's rows of each project
// table, keyed by table then id.
func deletions(t *testing.T, pool *pgxpool.Pool, workspace uuid.UUID) map[string]map[uuid.UUID]deletion {
	t.Helper()
	out := map[string]map[uuid.UUID]deletion{}
	for _, table := range projectTables {
		rows, err := pool.Query(context.Background(),
			"SELECT id, "+table.project+", deleted_at, updated_at, updated_by_id FROM "+table.name+" WHERE workspace_id = $1", workspace)
		if err != nil {
			t.Fatal(err)
		}
		out[table.name] = map[uuid.UUID]deletion{}
		for rows.Next() {
			var id uuid.UUID
			var d deletion
			if err := rows.Scan(&id, &d.project, &d.deletedAt, &d.updatedAt, &d.updatedBy); err != nil {
				t.Fatal(err)
			}
			out[table.name][id] = d
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

// seedProject writes a project of workspace named name, by alice at now,
// with alice's and bob's memberships and display settings in it, a backlog
// state and the triage state, and a label and its child, and returns its
// id. The rows are written directly: the delete statements read no column
// the inserts of the use cases would set otherwise.
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
	exec(t, pool, `INSERT INTO labels (id, workspace_id, project_id, parent_id, name, created_by_id, updated_by_id, created_at, updated_at)
		VALUES ($1, $2, $3, NULL, 'Bug', $4, $4, $5, $5), ($6, $2, $3, $1, 'UI', $4, $4, $5, $5)`, uuid.NewV7(), workspace, id, alice, now, uuid.NewV7())
	return id
}

// seedProjects writes three projects of workspace (seedProject) and
// returns their ids: web; ops, archived, bob's membership of it inactive;
// old, deleted at earlier with the rows under it.
func seedProjects(t *testing.T, pool *pgxpool.Pool, workspace, alice, bob uuid.UUID, earlier time.Time) (web, ops, old uuid.UUID) {
	t.Helper()
	web = seedProject(t, pool, workspace, "Web", alice, bob)
	ops = seedProject(t, pool, workspace, "Ops", alice, bob)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", ops, now)
	exec(t, pool, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2", ops, bob)
	old = seedProject(t, pool, workspace, "Old", alice, bob)
	for _, table := range projectTables {
		exec(t, pool, "UPDATE "+table.name+" SET deleted_at = $2 WHERE "+table.project+" = $1", old, earlier)
	}
	return web, ops, old
}

// deletionStep is a step of a deletion of projects, by its name.
type deletionStep struct {
	name string
	run  func(context.Context, app.Deletion) error
}

// deletionSteps are s's steps of a deletion of projects: every method of
// app.ProjectsDeleter, in the order of their names, so that a step added
// to the port is run here without a list to extend. Each step is one
// statement that reads no row another step writes, so their order does
// not change what they write; the use cases' order is app's to test.
func deletionSteps(s *postgresadapter.Store) []deletionStep {
	port := reflect.TypeFor[app.ProjectsDeleter]()
	deleter := reflect.ValueOf(app.ProjectsDeleter(s))
	steps := make([]deletionStep, port.NumMethod())
	for i := range steps {
		name := port.Method(i).Name
		steps[i] = deletionStep{name, deleter.MethodByName(name).Interface().(func(context.Context, app.Deletion) error)}
	}
	return steps
}

// deleteAll runs every step of d (deletionSteps), in the transaction ctx
// carries if any.
func deleteAll(ctx context.Context, s *postgresadapter.Store, d app.Deletion) error {
	for _, step := range deletionSteps(s) {
		if err := step.run(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

// checkDeletions holds the workspace's rows of each project table to want,
// each row's state by its project: "deleted" at later by bob; "before",
// deleted at earlier and not changed since; "kept", neither deleted nor
// changed. Each project of want has the table's perProject rows, at least
// one, and no row is under another.
func checkDeletions(t *testing.T, pool *pgxpool.Pool, workspace uuid.UUID, want map[uuid.UUID]string, later, earlier time.Time, bob uuid.UUID) {
	t.Helper()
	read := deletions(t, pool, workspace)
	for _, table := range projectTables {
		under := map[uuid.UUID]int{}
		for id, d := range read[table.name] {
			under[d.project]++
			var ok bool
			switch want[d.project] {
			case "deleted":
				ok = d.deletedAtBy(later, bob)
			case "before":
				ok = d.deletedAt != nil && d.deletedAt.Equal(earlier) && d.updatedAt.Equal(now)
			case "kept":
				ok = d.deletedAt == nil && d.updatedAt.Equal(now)
			}
			if !ok {
				t.Errorf("%s %s of project %s: %s; want it %q (deleted: at %s by bob %s; before: deleted_at %s, updated_at %s; kept: "+
					"deleted_at null, updated_at %s)", table.name, id, d.project, d, want[d.project], instant(&later), bob, instant(&earlier),
					instant(&now), instant(&now))
			}
		}
		for project, state := range want {
			switch {
			case under[project] == 0:
				t.Errorf("%s: no row under the %s project %s, so nothing of the table is checked: seed one (seedProject)", table.name, state,
					project)
			case under[project] != table.perProject:
				t.Errorf("%s: %d rows under the %s project %s, want %d", table.name, under[project], state, project, table.perProject)
			}
		}
	}
}

// The five steps, on every project of a workspace, in one transaction,
// soft-delete the workspace's projects, archived ones too, every membership
// of them, active or not, every member's display settings in them, every
// state, the triage ones too, and every label, parents and children alike,
// at the same moment and by the same account; a row deleted before keeps
// its time, and another workspace keeps everything. Running the steps again
// changes nothing.
func TestDeletingAWorkspaceSoftDeletesItsProjects(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	earlier, later := now.Add(-time.Hour), now.Add(time.Hour)
	betaWeb, betaOps, betaOld := seedProjects(t, pool, beta, alice, bob, earlier)
	web, ops, old := seedProjects(t, pool, acme, alice, bob, earlier)

	if err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		return deleteAll(ctx, s, app.Deletion{WorkspaceID: acme, By: bob, Now: later})
	}); err != nil {
		t.Fatal(err)
	}
	// Once more, by alice an hour later: nothing is left undeleted.
	if err := deleteAll(context.Background(), s, app.Deletion{WorkspaceID: acme, By: alice, Now: later.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}

	checkDeletions(t, pool, acme, map[uuid.UUID]string{web: "deleted", ops: "deleted", old: "before"}, later, earlier, bob)
	var active []bool
	if err := pool.QueryRow(context.Background(), `SELECT array_agg(is_active ORDER BY is_active) FROM project_members
		WHERE workspace_id = $1 AND deleted_at = $2`, acme, later).Scan(&active); err != nil || len(active) != 4 || active[0] || !active[1] {
		t.Errorf("is_active of acme's deleted memberships: %v, %v; want one false, three true, as they were", active, err)
	}
	checkDeletions(t, pool, beta, map[uuid.UUID]string{betaWeb: "kept", betaOps: "kept", betaOld: "before"}, later, earlier, bob)
}

// unwritten is every row of each project table under workspace, as text,
// without the three columns a deletion writes: what a deletion must leave.
func unwritten(t *testing.T, pool *pgxpool.Pool, workspace uuid.UUID) string {
	t.Helper()
	var all string
	for _, table := range projectTables {
		var rows string
		if err := pool.QueryRow(context.Background(), `SELECT coalesce(string_agg((to_jsonb(r) - 'deleted_at' - 'updated_at' - 'updated_by_id')::text,
			E'\n' ORDER BY r.id), '') FROM `+table.name+` r WHERE r.workspace_id = $1`, workspace).Scan(&rows); err != nil {
			t.Fatal(err)
		}
		all += table.name + ":\n" + rows + "\n"
	}
	return all
}

// The five steps, on one project, soft-delete it and the rows under it
// alone, archived or not, the first project of its workspace or the last
// undeleted one, at the same moment and by the same account, writing no
// other column: the workspace's other project, its project deleted before
// and another workspace's projects keep everything. Running the steps again
// changes nothing.
func TestDeletingAProjectSoftDeletesItsRowsAlone(t *testing.T) {
	for _, target := range []string{"Web", "Ops"} {
		t.Run(target, func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
			acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
			earlier, later := now.Add(-time.Hour), now.Add(time.Hour)
			web, ops, old := seedProjects(t, pool, acme, alice, bob, earlier)
			betaWeb, betaOps, betaOld := seedProjects(t, pool, beta, alice, bob, earlier)
			want := map[uuid.UUID]string{web: "kept", ops: "kept", old: "before"}
			id := map[string]uuid.UUID{"Web": web, "Ops": ops}[target]
			want[id] = "deleted"
			before := unwritten(t, pool, acme)

			if err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
				return deleteAll(ctx, s, app.Deletion{WorkspaceID: acme, ProjectID: &id, By: bob, Now: later})
			}); err != nil {
				t.Fatal(err)
			}
			if err := deleteAll(context.Background(), s, app.Deletion{WorkspaceID: acme, ProjectID: &id, By: alice, Now: later.Add(time.Hour)}); err != nil {
				t.Fatal(err)
			}

			checkDeletions(t, pool, acme, want, later, earlier, bob)
			checkDeletions(t, pool, beta, map[uuid.UUID]string{betaWeb: "kept", betaOps: "kept", betaOld: "before"}, later, earlier, bob)
			if after := unwritten(t, pool, acme); after != before {
				t.Errorf("acme's rows but the deletion's columns:\n%s\nwant\n%s", after, before)
			}
		})
	}
}

// The five steps, on a project of another workspace than the deletion's,
// change nothing: the workspace bounds a project's deletion too, so
// neither workspace loses a row, the named project included.
func TestDeletingAnotherWorkspacesProjectChangesNothing(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	earlier, later := now.Add(-time.Hour), now.Add(time.Hour)
	web, ops, old := seedProjects(t, pool, acme, alice, bob, earlier)
	betaWeb, betaOps, betaOld := seedProjects(t, pool, beta, alice, bob, earlier)
	before := unwritten(t, pool, acme) + unwritten(t, pool, beta)

	if err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		return deleteAll(ctx, s, app.Deletion{WorkspaceID: acme, ProjectID: &betaWeb, By: bob, Now: later})
	}); err != nil {
		t.Fatal(err)
	}

	checkDeletions(t, pool, acme, map[uuid.UUID]string{web: "kept", ops: "kept", old: "before"}, later, earlier, bob)
	checkDeletions(t, pool, beta, map[uuid.UUID]string{betaWeb: "kept", betaOps: "kept", betaOld: "before"}, later, earlier, bob)
	if after := unwritten(t, pool, acme) + unwritten(t, pool, beta); after != before {
		t.Errorf("acme's and beta's rows but the deletion's columns:\n%s\nwant\n%s", after, before)
	}
}
