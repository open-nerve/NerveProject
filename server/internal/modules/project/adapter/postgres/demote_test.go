package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// seedMember writes user's membership of project, of role, active or not,
// written by user at now, and returns its id.
func seedMember(t *testing.T, pool *pgxpool.Pool, workspace, project, user uuid.UUID, role int, active bool) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, `INSERT INTO project_members (id, workspace_id, project_id, member_id, role, is_active, created_by_id, updated_by_id,
		created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $4, $4, $7, $7)`, id, workspace, project, user, role, active, now)
	return id
}

// waits reports whether a statement of another transaction on project's
// row, lock, waits for a lock held on it: NOWAIT answers lock_not_available
// (55P03) at once instead of waiting.
func waits(t *testing.T, pool *pgxpool.Pool, project uuid.UUID, lock string) bool {
	t.Helper()
	_, err := pool.Exec(context.Background(), "SELECT 1 FROM projects WHERE id = $1 "+lock+" NOWAIT", project)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "55P03" {
		return true
	}
	if err != nil {
		t.Fatal(err)
	}
	return false
}

// DemoteToGuest's two steps in one transaction (M3 design 3.3, 3.6). The
// lock returns, in id order, acme's undeleted projects, the archived one
// too, in which bob has an undeleted membership, active, ended, or a
// guest's already; until the transaction ends each of them is held FOR NO
// KEY UPDATE, which a FOR SHARE waits for and a foreign key's FOR KEY SHARE
// does not, and no other project is: not HR, where alice is a member and
// bob's membership is deleted. The write makes bob's memberships of those a
// guest's, at the moment and by the account given, the ended one still
// ended, and leaves their other columns as they were; his guest's one,
// alice's, his deleted ones, his one of a deleted project and his one in
// beta keep every column.
func TestDemotingAMemberToGuest(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	arch, docs := newProject(t, s, acme, "Arch", "ARCH", alice), newProject(t, s, acme, "Docs", "DOCS", alice)
	hr, gone := newProject(t, s, acme, "HR", "HR", alice), newProject(t, s, acme, "Gone", "GONE", alice)
	betas := newProject(t, s, beta, "Web", "WEB", alice)
	exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", arch, now)
	exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)
	// bob's deleted memberships: of Web, before the one he has now, and of HR.
	for _, project := range []uuid.UUID{web, hr} {
		exec(t, pool, "UPDATE project_members SET deleted_at = $2 WHERE id = $1", seedMember(t, pool, acme, project, bob, 15, true), now)
	}
	demoted := map[uuid.UUID]bool{ // by id, whether active
		seedMember(t, pool, acme, web, bob, 20, true): true, seedMember(t, pool, acme, ops, bob, 15, false): false,
		seedMember(t, pool, acme, arch, bob, 15, true): true,
	}
	seedMember(t, pool, acme, docs, bob, 5, true)
	seedMember(t, pool, acme, web, alice, 20, true)
	seedMember(t, pool, acme, hr, alice, 20, true)
	seedMember(t, pool, acme, gone, bob, 20, true)
	seedMember(t, pool, beta, betas, bob, 20, true)
	// kept is every membership, a demoted one without the columns the write
	// sets.
	kept := func() string {
		t.Helper()
		var rows string
		if err := pool.QueryRow(context.Background(), `SELECT string_agg(CASE WHEN r.id = ANY ($1)
			THEN (to_jsonb(r) - 'role' - 'updated_at' - 'updated_by_id')::text ELSE r::text END, E'\n' ORDER BY r.id)
			FROM project_members r`, slices.Collect(maps.Keys(demoted))).Scan(&rows); err != nil {
			t.Fatal(err)
		}
		return rows
	}
	before := kept()
	later := now.Add(time.Hour)

	var locked []uuid.UUID
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		var err error
		if locked, err = s.LockMemberProjects(ctx, acme, bob); err != nil {
			return err
		}
		for name, p := range map[string]struct {
			id   uuid.UUID
			held bool
		}{"Web": {web, true}, "Ops": {ops, true}, "Arch": {arch, true}, "Docs": {docs, true}, "HR": {hr, false}, "Gone": {gone, false},
			"beta's Web": {betas, false}} {
			if share, keyShare := waits(t, pool, p.id, "FOR SHARE"), waits(t, pool, p.id, "FOR KEY SHARE"); share != p.held || keyShare {
				t.Errorf("%s while locked: a FOR SHARE waits %v, a FOR KEY SHARE waits %v; want %v, false", name, share, keyShare, p.held)
			}
		}
		return s.DemoteMemberships(ctx, locked, bob, alice, later)
	})
	if err != nil {
		t.Fatal(err)
	}

	if want := slices.SortedFunc(slices.Values([]uuid.UUID{web, ops, arch, docs}), uuid.UUID.Compare); !slices.Equal(locked, want) {
		t.Errorf("LockMemberProjects() = %v, want %v", locked, want)
	}
	for id, active := range demoted {
		var role int
		var isActive bool
		var updatedAt time.Time
		var updatedBy uuid.UUID
		if err := pool.QueryRow(context.Background(), "SELECT role, is_active, updated_at, updated_by_id FROM project_members WHERE id = $1", id).
			Scan(&role, &isActive, &updatedAt, &updatedBy); err != nil {
			t.Fatal(err)
		}
		if role != 5 || isActive != active || !updatedAt.Equal(later) || updatedBy != alice {
			t.Errorf("membership %s: role %d, active %v, at %v by %s; want 5, %v, at %v by alice", id, role, isActive, updatedAt, updatedBy, active, later)
		}
	}
	if after := kept(); after != before {
		t.Errorf("the memberships, the demoted ones without role, updated_at and updated_by_id:\n%s\nwant them as they were:\n%s", after, before)
	}
}

// LockMemberProjects takes its locks in the projects' id order, whatever
// order the rows lie in: Web has the smaller id, but lies after Alpha in
// the table and in the indexes on the name and on the identifier. Alpha's
// row is held. LockMemberProjects waits for it holding Web's, which a FOR
// SHARE then waits for; in any other order it would reach Alpha first and
// wait holding nothing.
func TestLockMemberProjectsLocksInIDOrder(t *testing.T) {
	s, pool := newStore(t)
	bob := newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, alpha := uuid.NewV7(), uuid.NewV7() // web drawn first: the smaller id
	for _, p := range []struct {
		id   uuid.UUID
		name string
	}{{alpha, "Alpha"}, {web, "Web"}} {
		exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, $3::text, upper($3::text))", p.id, acme, p.name)
		seedMember(t, pool, acme, p.id, bob, 15, true)
	}
	held, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = held.Rollback(context.Background()) }()
	if _, err := held.Exec(context.Background(), "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", alpha); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- postgres.NewTxManager(pool, 10*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			_, err := s.LockMemberProjects(ctx, acme, bob)
			return err
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "projects", 10*time.Second)

	if !waits(t, pool, web, "FOR SHARE") {
		t.Error("a FOR SHARE of Web while LockMemberProjects waits for Alpha does not wait; want Web locked first")
	}
	if err := held.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("LockMemberProjects() = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockMemberProjects() did not end within 10s")
	}
}
