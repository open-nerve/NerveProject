package postgresadapter_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// endedRows is every membership as text, by id: one of ended without
// is_active, updated_at and updated_by_id, which ending it writes.
func endedRows(t *testing.T, pool *pgxpool.Pool, ended []uuid.UUID) string {
	t.Helper()
	var rows string
	if err := pool.QueryRow(context.Background(), `SELECT string_agg(CASE WHEN r.id = ANY ($1)
		THEN (to_jsonb(r) - 'is_active' - 'updated_at' - 'updated_by_id')::text ELSE r::text END, E'\n' ORDER BY r.id)
		FROM project_members r`, ended).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

// EndMemberships' first and last steps in one transaction (M3 design 3.6
// convention 6). The lock returns, in id order, the undeleted projects of
// acme and gamma, the archived one too, in which bob has an active
// membership; until the transaction ends each of them is held FOR NO KEY
// UPDATE, which a FOR SHARE waits for and a foreign key's FOR KEY SHARE
// does not, and no other project is: not Docs, where his membership ended;
// not HR, where his is deleted and alice's active; not Free, where only
// alice is; not Gone, deleted, though his membership of it is not; not
// beta's Web, beta not asked for. The write ends his memberships of those,
// at the moment and by the account given, each keeping its role, and leaves
// their other columns as they were; his deleted membership of Web, stored
// before his live one or after it, alice's, his ended one and his ones in
// Gone and beta keep every column. Asked to end his ended membership of
// Docs, it writes nothing.
func TestEndingAMembersProjectMemberships(t *testing.T) {
	for _, deletedFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("his deleted membership of Web stored first %v", deletedFirst), func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
			acme, beta, gamma := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta"), newWorkspace(t, pool, "gamma")
			web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
			arch, docs := newProject(t, s, acme, "Arch", "ARCH", alice), newProject(t, s, acme, "Docs", "DOCS", alice)
			hr, free := newProject(t, s, acme, "HR", "HR", alice), newProject(t, s, acme, "Free", "FREE", alice)
			gone, site := newProject(t, s, acme, "Gone", "GONE", alice), newProject(t, s, gamma, "Site", "SITE", alice)
			betas := newProject(t, s, beta, "Web", "WEB", alice)
			exec(t, pool, "UPDATE projects SET archived_at = $2 WHERE id = $1", arch, now)
			exec(t, pool, "UPDATE projects SET deleted_at = $2 WHERE id = $1", gone, now)
			// deleted stores bob's deleted membership of project, active as
			// a deletion leaves it, beside a live one or none.
			deleted := func(project uuid.UUID) {
				exec(t, pool, `INSERT INTO project_members (id, workspace_id, project_id, member_id, role, created_by_id, updated_by_id,
					created_at, updated_at, deleted_at) VALUES ($1, $2, $3, $4, 20, $4, $4, $5, $5, $5)`, uuid.NewV7(), acme, project, bob, now)
			}
			if deletedFirst {
				deleted(web)
			}
			ended := []uuid.UUID{seedMember(t, pool, acme, web, bob, 20, true)}
			if !deletedFirst {
				deleted(web)
			}
			ended = append(ended, seedMember(t, pool, acme, ops, bob, 15, true), seedMember(t, pool, acme, arch, bob, 5, true),
				seedMember(t, pool, gamma, site, bob, 15, true))
			deleted(hr)
			docsMembership := seedMember(t, pool, acme, docs, bob, 15, false)
			for _, p := range []uuid.UUID{web, hr, free} {
				seedMember(t, pool, acme, p, alice, 20, true)
			}
			seedMember(t, pool, acme, gone, bob, 20, true)
			seedMember(t, pool, beta, betas, bob, 20, true)
			before := endedRows(t, pool, ended)
			later := now.Add(time.Hour)

			var locked []uuid.UUID
			err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
				var err error
				if locked, err = s.LockActiveMemberProjects(ctx, []uuid.UUID{acme, gamma}, bob); err != nil {
					return err
				}
				for name, p := range map[string]struct {
					id   uuid.UUID
					held bool
				}{"Web": {web, true}, "Ops": {ops, true}, "Arch": {arch, true}, "gamma's Site": {site, true}, "Docs": {docs, false},
					"HR": {hr, false}, "Free": {free, false}, "Gone": {gone, false}, "beta's Web": {betas, false}} {
					if share, keyShare := waits(t, pool, p.id, "FOR SHARE"), waits(t, pool, p.id, "FOR KEY SHARE"); share != p.held || keyShare {
						t.Errorf("%s while locked: a FOR SHARE waits %v, a FOR KEY SHARE waits %v; want %v, false", name, share, keyShare, p.held)
					}
				}
				return s.EndMemberships(ctx, locked, bob, alice, later)
			})
			if err != nil {
				t.Fatal(err)
			}

			if want := slices.SortedFunc(slices.Values([]uuid.UUID{web, ops, arch, site}), uuid.UUID.Compare); !slices.Equal(locked, want) {
				t.Errorf("LockActiveMemberProjects() = %v, want %v", locked, want)
			}
			for i, id := range ended {
				var role int
				var active bool
				var updatedAt time.Time
				var updatedBy uuid.UUID
				if err := pool.QueryRow(context.Background(), "SELECT role, is_active, updated_at, updated_by_id FROM project_members WHERE id = $1",
					id).Scan(&role, &active, &updatedAt, &updatedBy); err != nil {
					t.Fatal(err)
				}
				if want := []int{20, 15, 5, 15}[i]; role != want || active || !updatedAt.Equal(later) || updatedBy != alice {
					t.Errorf("membership %s: role %d, active %v, at %v by %s; want %d, ended, at %v by alice", id, role, active, updatedAt, updatedBy,
						want, later)
				}
			}
			if after := endedRows(t, pool, ended); after != before {
				t.Errorf("the memberships, the ended ones without is_active, updated_at and updated_by_id:\n%s\nwant them as they were:\n%s", after,
					before)
			}
			before = endedRows(t, pool, nil)
			if err := s.EndMemberships(context.Background(), []uuid.UUID{docs}, bob, alice, later.Add(time.Hour)); err != nil {
				t.Fatal(err)
			}
			if after := endedRows(t, pool, nil); after != before {
				t.Errorf("ending his ended membership of Docs (%s) wrote:\n%s\nwant the memberships as they were:\n%s", docsMembership, after, before)
			}
		})
	}
}

// SoleAdmin reports whether bob is the only active admin of one of the
// projects asked about that has another active member (M3 design 3.7 rule
// 2). Every project is acme's, so that a check of another project's members
// or admins, which the other cases' projects have, answers otherwise. Most
// cases ask about one project, in the state the case names; three ask about
// two of those projects at once, one of them with another admin or other
// members, so that a check of the set asked about, not of each of its
// projects, answers otherwise too; one asks about none. Plane's checks got
// this set wrong: the workspace membership's id compared with a project
// member's account, the projects of one member only, those where he is
// alone.
func TestSoleAdmin(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme := newWorkspace(t, pool, "acme")
	type member struct {
		user   uuid.UUID
		role   int
		active bool
	}
	// project stores a project of acme with members, the last of them
	// deleted when deleteLast is set, and returns its id.
	n := 0
	project := func(deleteLast bool, members ...member) uuid.UUID {
		t.Helper()
		n++
		p := newProject(t, s, acme, fmt.Sprintf("P%d", n), fmt.Sprintf("P%d", n), alice)
		for i, m := range members {
			id := seedMember(t, pool, acme, p, m.user, m.role, m.active)
			if deleteLast && i == len(members)-1 {
				exec(t, pool, "UPDATE project_members SET deleted_at = $2 WHERE id = $1", id, now)
			}
		}
		return p
	}
	alone := project(false, member{bob, 20, true})
	withAMember := project(false, member{bob, 20, true}, member{carol, 15, true})
	twoAdmins := project(false, member{bob, 20, true}, member{alice, 20, true}, member{carol, 15, true})
	tests := []struct {
		name     string
		projects []uuid.UUID
		want     bool
	}{
		{"the only admin, with an active member", []uuid.UUID{withAMember}, true},
		{"the only admin, with an active guest", []uuid.UUID{project(false, member{bob, 20, true}, member{carol, 5, true})}, true},
		{"the only admin, alone", []uuid.UUID{alone}, false},
		{"the only admin, the other's membership ended", []uuid.UUID{project(false, member{bob, 20, true}, member{carol, 15, false})}, false},
		{"the only admin, the other's membership deleted", []uuid.UUID{project(true, member{bob, 20, true}, member{carol, 15, true})}, false},
		{"one of two active admins", []uuid.UUID{twoAdmins}, false},
		{"the other admin's membership ended", []uuid.UUID{project(false, member{bob, 20, true}, member{alice, 20, false},
			member{carol, 15, true})}, true},
		{"the other admin's membership deleted", []uuid.UUID{project(true, member{bob, 20, true}, member{carol, 15, true},
			member{alice, 20, true})}, true},
		{"a member, beside the only admin and another member", []uuid.UUID{project(false, member{bob, 15, true}, member{alice, 20, true},
			member{carol, 15, true})}, false},
		{"a member, beside another member and no admin", []uuid.UUID{project(false, member{bob, 15, true}, member{carol, 15, true})}, false},
		{"an admin whose membership ended", []uuid.UUID{project(false, member{bob, 20, false}, member{carol, 15, true})}, false},
		{"an admin whose membership is deleted", []uuid.UUID{project(true, member{carol, 15, true}, member{bob, 20, true})}, false},
		{"a member of the project asked about, the only admin of another", []uuid.UUID{project(false, member{bob, 15, true},
			member{alice, 20, true})}, false},
		{"the only admin of one of two asked about", []uuid.UUID{alone, withAMember}, true},
		{"the only admin of one, another asked about having another admin", []uuid.UUID{withAMember, twoAdmins}, true},
		{"alone in one, another asked about having other members", []uuid.UUID{alone, twoAdmins}, false},
		{"none asked about", nil, false},
	}
	for _, tt := range tests {
		if got, err := s.SoleAdmin(context.Background(), tt.projects, bob); err != nil || got != tt.want {
			t.Errorf("%s: SoleAdmin() = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
}

// LockActiveMemberProjects takes its locks in the projects' id order,
// whatever order the rows lie in: Web has the smaller id, but lies after
// Alpha in the table and in the indexes on the name and on the identifier.
// Alpha's row is held. LockActiveMemberProjects waits for it holding
// Web's, which a FOR SHARE then waits for; in any other order it would
// reach Alpha first and wait holding nothing.
func TestLockActiveMemberProjectsLocksInIDOrder(t *testing.T) {
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
			_, err := s.LockActiveMemberProjects(ctx, []uuid.UUID{acme}, bob)
			return err
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "projects", 10*time.Second)

	if !waits(t, pool, web, "FOR SHARE") {
		t.Error("a FOR SHARE of Web while LockActiveMemberProjects waits for Alpha does not wait; want Web locked first")
	}
	if err := held.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("LockActiveMemberProjects() = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockActiveMemberProjects() did not end within 10s")
	}
}

// A project deleted while LockActiveMemberProjects waits for its row is
// left out (M3 design 3.6): another transaction holds Web FOR NO KEY
// UPDATE, as deleteProject does, and soft-deletes it, its memberships too;
// LockActiveMemberProjects, which found Web undeleted, waits for it; once
// the deletion commits, it evaluates deleted_at again on the row's newest
// version and returns Ops alone.
func TestLockActiveMemberProjectsLeavesOutAProjectDeletedWhileItWaited(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	for _, p := range []uuid.UUID{web, ops} {
		seedMember(t, pool, acme, p, bob, 15, true)
	}
	deletion, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = deletion.Rollback(context.Background()) }()
	for _, sql := range []string{"SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", "UPDATE projects SET deleted_at = now() WHERE id = $1",
		"UPDATE project_members SET deleted_at = now() WHERE project_id = $1"} {
		if _, err := deletion.Exec(context.Background(), sql, web); err != nil {
			t.Fatal(err)
		}
	}
	type locked struct {
		ids []uuid.UUID
		err error
	}
	done := make(chan locked, 1)
	go func() {
		var l locked
		l.err = postgres.NewTxManager(pool, 10*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			var err error
			l.ids, err = s.LockActiveMemberProjects(ctx, []uuid.UUID{acme}, bob)
			return err
		})
		done <- l
	}()
	pgtest.WaitForLockWaitOn(t, pool, "projects", 10*time.Second)
	if err := deletion.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case l := <-done:
		if l.err != nil || !slices.Equal(l.ids, []uuid.UUID{ops}) {
			t.Errorf("LockActiveMemberProjects() = %v, %v; want Ops (%s) alone, Web (%s) deleted while it waited", l.ids, l.err, ops, web)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockActiveMemberProjects() did not end within 10s")
	}
}
