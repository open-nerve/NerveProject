package postgresadapter_test

import (
	"context"
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// membershipRows is every membership as text, by id: the one id without
// the columns cols, which a write of it writes.
func membershipRows(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, cols ...string) string {
	t.Helper()
	var rows string
	if err := pool.QueryRow(context.Background(), `SELECT string_agg(CASE WHEN r.id = $1 THEN (to_jsonb(r) - $2::text[])::text
		ELSE r::text END, E'\n' ORDER BY r.id) FROM project_members r`, id, cols).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

// seedDeleted writes user's deleted membership of project, of role,
// active as a deletion leaves it, beside a live one or none, written by user
// at now, and returns its id.
func seedDeleted(t *testing.T, pool *pgxpool.Pool, workspace, project, user uuid.UUID, role int) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	exec(t, pool, `INSERT INTO project_members (id, workspace_id, project_id, member_id, role, created_by_id, updated_by_id, created_at,
		updated_at, deleted_at) VALUES ($1, $2, $3, $4, $5, $4, $4, $6, $6, $6)`, id, workspace, project, user, role, now)
	return id
}

// written reads the role, is_active, updated_at and updated_by_id of the
// membership id.
func written(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) string {
	t.Helper()
	var role int
	var active bool
	var at time.Time
	var by uuid.UUID
	if err := pool.QueryRow(context.Background(), "SELECT role, is_active, updated_at, updated_by_id FROM project_members WHERE id = $1",
		id).Scan(&role, &active, &at, &by); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("role %d, active %v, at %s by %s", role, active, at.UTC().Format(time.RFC3339Nano), by)
}

// madeByAnother makes every membership stored so far created and last
// written by an account that is none of their members, nor the writer of
// any write of the tests: seedMember and seedDeleted write the member as
// both, so that a query reading created_by_id or updated_by_id where it
// means member_id would answer the same.
func madeByAnother(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	exec(t, pool, "UPDATE project_members SET created_by_id = $1, updated_by_id = $1", newAccount(t, pool, "maker@corp.com"))
}

// MemberByID reads the undeleted membership the id names, active or ended,
// with its workspace, project, member and role: not another membership,
// which a read of whichever row lies first would answer for one of the two
// asked about; not a deleted one, which is none, nor an id of none. Every
// membership was made by another account (madeByAnother).
func TestMemberByID(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, site := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, beta, "Site", "SITE", alice)
	alices := seedMember(t, pool, acme, web, alice, 20, true)
	bobs := seedMember(t, pool, beta, site, bob, 5, false)
	deleted := seedDeleted(t, pool, acme, web, bob, 15)
	madeByAnother(t, pool)
	for _, tt := range []struct {
		name  string
		id    uuid.UUID
		want  app.ProjectMembership
		found bool
	}{
		{"alice's of Web, active", alices, app.ProjectMembership{ID: alices, WorkspaceID: acme, ProjectID: web, MemberID: alice,
			Role: shared.RoleAdmin, Active: true}, true},
		{"bob's of beta's Site, ended", bobs, app.ProjectMembership{ID: bobs, WorkspaceID: beta, ProjectID: site, MemberID: bob,
			Role: shared.RoleGuest}, true},
		{"bob's of Web, deleted", deleted, app.ProjectMembership{}, false},
		{"none", uuid.NewV7(), app.ProjectMembership{}, false},
	} {
		if m, found, err := s.MemberByID(context.Background(), tt.id); err != nil || found != tt.found || m != tt.want {
			t.Errorf("%s: MemberByID() = %+v, %v, %v; want %+v, %v", tt.name, m, found, err, tt.want, tt.found)
		}
	}
}

// UpdateMemberRole gives bob's active membership of Web the role given, at
// the moment and by the account given, and answers it as stored: its id,
// project, member, new role and the time it was made. It demotes him from
// a member to a guest when his deleted membership is stored first, which a
// change that wrote a member's or an admin's, kept his role or took the
// greater of the two would get wrong, and promotes him from a guest to a
// member when it is stored after, which a change that wrote the column's
// default, a guest's, or an admin's, kept his role or took the lesser of
// the two would get wrong. The row keeps its other columns, and every other
// row every column: his memberships of Ops and of beta's Site, his deleted
// one of Web, stored before his live one or after it, carol's of Web. His
// ended membership of Docs, and his deleted one of Web, are each an error,
// and not written. Every membership was made and last written by another
// account (madeByAnother).
func TestUpdateMemberRole(t *testing.T) {
	for _, deletedFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("his deleted membership of Web stored first %v", deletedFirst), func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
			acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
			web, ops, docs := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice),
				newProject(t, s, acme, "Docs", "DOCS", alice)
			site := newProject(t, s, beta, "Site", "SITE", alice)
			var deleted uuid.UUID
			deleteOne := func() { deleted = seedDeleted(t, pool, acme, web, bob, 15) }
			if deletedFirst {
				deleteOne()
			}
			from, to := 15, shared.RoleGuest
			if !deletedFirst {
				from, to = 5, shared.RoleMember
			}
			bobs := seedMember(t, pool, acme, web, bob, from, true)
			if !deletedFirst {
				deleteOne()
			}
			seedMember(t, pool, acme, ops, bob, 15, true)
			seedMember(t, pool, beta, site, bob, 15, true)
			seedMember(t, pool, acme, web, carol, 15, true)
			ended := seedMember(t, pool, acme, docs, bob, 15, false)
			madeByAnother(t, pool)
			before := membershipRows(t, pool, bobs, "role", "updated_at", "updated_by_id")
			later := now.Add(time.Hour)

			m, err := s.UpdateMemberRole(context.Background(), bobs, to, alice, later)
			if want := (domain.Member{ID: bobs, ProjectID: web, MemberID: bob, Role: to, CreatedAt: now}); err != nil || m != want {
				t.Errorf("UpdateMemberRole() = %+v, %v; want %+v", m, err, want)
			}
			if got, want := written(t, pool, bobs), fmt.Sprintf("role %d, active true, at ", to)+later.UTC().Format(time.RFC3339Nano)+" by "+alice.String(); got != want {
				t.Errorf("bob's membership of Web: %s; want %s", got, want)
			}
			if after := membershipRows(t, pool, bobs, "role", "updated_at", "updated_by_id"); after != before {
				t.Errorf("the memberships, bob's of Web without role, updated_at and updated_by_id:\n%s\nwant them as they were:\n%s", after, before)
			}
			for name, id := range map[string]uuid.UUID{"his ended one of Docs": ended, "his deleted one of Web": deleted} {
				before := membershipRows(t, pool, uuid.UUID{})
				if m, err := s.UpdateMemberRole(context.Background(), id, shared.RoleMember, alice, later.Add(time.Hour)); err == nil {
					t.Errorf("UpdateMemberRole() of %s = %+v, nil; want an error", name, m)
				}
				if after := membershipRows(t, pool, uuid.UUID{}); after != before {
					t.Errorf("changing %s wrote:\n%s\nwant the memberships as they were:\n%s", name, after, before)
				}
			}
		})
	}
}

// EndMember ends bob's active membership of Web, at the moment and by the
// account given; its role stays: a guest's when his deleted membership is
// stored first, which an ending that wrote a member's or an admin's would
// change, and an admin's when it is stored after, which an ending that
// wrote the column's default, a guest's, would. The row keeps its other
// columns, and every other row every column: his memberships of Ops and of
// beta's Site, his deleted one of Web, active as a deletion leaves it,
// stored before his live one or after it, and carol's of Web. Asked again,
// with nothing active left to end, it is an error and writes nothing; so
// is asking about Docs, where his membership ended, and about Gone, where
// his only membership is deleted, though active. Every membership was made
// and last written by another account (madeByAnother).
func TestEndMember(t *testing.T) {
	for _, deletedFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("his deleted membership of Web stored first %v", deletedFirst), func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
			acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
			web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
			docs, gone := newProject(t, s, acme, "Docs", "DOCS", alice), newProject(t, s, acme, "Gone", "GONE", alice)
			site := newProject(t, s, beta, "Site", "SITE", alice)
			deleted := func(project uuid.UUID) { seedDeleted(t, pool, acme, project, bob, 15) }
			if deletedFirst {
				deleted(web)
			}
			role := 5
			if !deletedFirst {
				role = 20
			}
			bobs := seedMember(t, pool, acme, web, bob, role, true)
			if !deletedFirst {
				deleted(web)
			}
			seedMember(t, pool, acme, ops, bob, 15, true)
			seedMember(t, pool, beta, site, bob, 15, true)
			seedMember(t, pool, acme, web, carol, 15, true)
			seedMember(t, pool, acme, docs, bob, 15, false)
			deleted(gone)
			madeByAnother(t, pool)
			before := membershipRows(t, pool, bobs, "is_active", "updated_at", "updated_by_id")
			later := now.Add(time.Hour)

			if err := s.EndMember(context.Background(), web, bob, alice, later); err != nil {
				t.Fatalf("EndMember() = %v", err)
			}
			if got, want := written(t, pool, bobs), fmt.Sprintf("role %d, active false, at ", role)+later.UTC().Format(time.RFC3339Nano)+" by "+alice.String(); got != want {
				t.Errorf("bob's membership of Web: %s; want %s", got, want)
			}
			if after := membershipRows(t, pool, bobs, "is_active", "updated_at", "updated_by_id"); after != before {
				t.Errorf("the memberships, bob's of Web without is_active, updated_at and updated_by_id:\n%s\nwant them as they were:\n%s", after,
					before)
			}
			for name, project := range map[string]uuid.UUID{"Web again": web, "Docs": docs, "Gone": gone} {
				before := membershipRows(t, pool, uuid.UUID{})
				if err := s.EndMember(context.Background(), project, bob, alice, later.Add(time.Hour)); err == nil {
					t.Errorf("EndMember() of %s = nil; want an error", name)
				}
				if after := membershipRows(t, pool, uuid.UUID{}); after != before {
					t.Errorf("ending bob's membership of %s wrote:\n%s\nwant the memberships as they were:\n%s", name, after, before)
				}
			}
		})
	}
}

// HasOtherAdmin reports whether the project asked about has an active
// admin other than bob (M3 design 3.7 rule 1). Every project is acme's,
// and in each case one other project has an active admin other than bob,
// so that a check of another project answers otherwise; bob is an active
// admin of the projects where he is asked about as one, so that a check
// that counts him answers otherwise. The other active admin is stored
// after bob and before him, so that a check of whichever admin lies last or
// first answers otherwise; every membership was made by another account
// (madeByAnother).
func TestHasOtherAdmin(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme := newWorkspace(t, pool, "acme")
	type member struct {
		user   uuid.UUID
		role   int
		active bool
	}
	n := 0
	// project stores a project of acme with members, the last of them
	// deleted when deleteLast is set, and returns its id.
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
	project(false, member{alice, 20, true}, member{bob, 20, true})
	cases := []struct {
		name    string
		project uuid.UUID
		want    bool
	}{
		{"another active admin", project(false, member{bob, 20, true}, member{alice, 20, true}), true},
		{"another active admin stored first", project(false, member{alice, 20, true}, member{bob, 20, true}), true},
		{"another active admin, bob a member", project(false, member{bob, 15, true}, member{alice, 20, true}), true},
		{"bob alone", project(false, member{bob, 20, true}), false},
		{"another active member", project(false, member{bob, 20, true}, member{carol, 15, true}), false},
		{"another active guest", project(false, member{bob, 20, true}, member{carol, 5, true}), false},
		{"the other admin's membership ended", project(false, member{bob, 20, true}, member{alice, 20, false}), false},
		{"the other admin's membership deleted", project(true, member{bob, 20, true}, member{alice, 20, true}), false},
		{"no member", project(false), false},
	}
	madeByAnother(t, pool)
	for _, tt := range cases {
		if got, err := s.HasOtherAdmin(context.Background(), tt.project, bob); err != nil || got != tt.want {
			t.Errorf("%s: HasOtherAdmin() = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
}
