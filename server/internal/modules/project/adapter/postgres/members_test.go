package postgresadapter_test

import (
	"context"
	"maps"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Memberships answers the accounts asked about, each by his undeleted
// membership of the project, active or ended, with its id and role: not
// another project's, not a deleted one, not an account not asked about.
// carol's deleted membership was stored before her ended one, and dave's
// one of ops before his none in web, so that neither row order answers
// the other's.
func TestMemberships(t *testing.T) {
	s, pool := newStore(t)
	var ids []uuid.UUID
	for _, email := range []string{"alice@corp.com", "bob@corp.com", "carol@corp.com", "dave@corp.com", "erin@corp.com", "frank@corp.com"} {
		ids = append(ids, newAccount(t, pool, email))
	}
	alice, bob, carol, dave, erin, frank := ids[0], ids[1], ids[2], ids[3], ids[4], ids[5]
	acme := newWorkspace(t, pool, "acme")
	ops, web := newProject(t, s, acme, "Ops", "OPS", alice), newProject(t, s, acme, "Web", "WEB", alice)
	seedMember(t, pool, acme, ops, dave, 20, true)
	deleted := seedMember(t, pool, acme, web, carol, 20, true)
	exec(t, pool, "UPDATE project_members SET deleted_at = $2 WHERE id = $1", deleted, now)
	carols := seedMember(t, pool, acme, web, carol, 5, false)
	alices := seedMember(t, pool, acme, web, alice, 20, true)
	bobs := seedMember(t, pool, acme, web, bob, 15, false)
	seedMember(t, pool, acme, web, erin, 15, true)
	frank2 := seedMember(t, pool, acme, web, frank, 20, true)
	exec(t, pool, "UPDATE project_members SET deleted_at = $2 WHERE id = $1", frank2, now)

	got, err := s.Memberships(context.Background(), web, []uuid.UUID{alice, bob, carol, dave, frank})
	want := map[uuid.UUID]app.Membership{
		alice: {ID: alices, Role: shared.RoleAdmin, Active: true},
		bob:   {ID: bobs, Role: shared.RoleMember},
		carol: {ID: carols, Role: shared.RoleGuest},
	}
	if err != nil || !maps.Equal(got, want) {
		t.Errorf("Memberships() = %v, %v; want %v", got, err, want)
	}
	if got, err := s.Memberships(context.Background(), web, nil); err != nil || len(got) != 0 {
		t.Errorf("Memberships() of nobody = %v, %v; want none", got, err)
	}
}

// ListMembers lists the project's active undeleted memberships alone, by
// the time they were made, then by id: erin's, made last but stored first,
// comes last; bob's, gina's and alice's were made at one moment, gina's
// stored first, bob's with the smallest id and alice's the largest, so that
// neither the table's order of the three, nor its reverse, nor their
// accounts' order is the answer's. Not carol's, ended; not dave's, deleted;
// not frank's of another project.
func TestListMembers(t *testing.T) {
	s, pool := newStore(t)
	var ids []uuid.UUID
	for _, email := range []string{"alice@corp.com", "bob@corp.com", "carol@corp.com", "dave@corp.com", "erin@corp.com", "frank@corp.com",
		"gina@corp.com"} {
		ids = append(ids, newAccount(t, pool, email))
	}
	alice, bob, carol, dave, erin, frank, gina := ids[0], ids[1], ids[2], ids[3], ids[4], ids[5], ids[6]
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	erins := seedMember(t, pool, acme, web, erin, 5, true)
	exec(t, pool, "UPDATE project_members SET created_at = $2 WHERE id = $1", erins, now.Add(time.Hour))
	seedMember(t, pool, acme, web, carol, 15, false)
	bobs, ginas, alices := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	want := []domain.Member{{ID: bobs, ProjectID: web, MemberID: bob, Role: shared.RoleMember, CreatedAt: now},
		{ID: ginas, ProjectID: web, MemberID: gina, Role: shared.RoleGuest, CreatedAt: now},
		{ID: alices, ProjectID: web, MemberID: alice, Role: shared.RoleAdmin, CreatedAt: now},
		{ID: erins, ProjectID: web, MemberID: erin, Role: shared.RoleGuest, CreatedAt: now.Add(time.Hour)}}
	for _, m := range []domain.Member{want[1], want[0], want[2]} {
		if err := s.CreateMember(context.Background(), app.MemberRow{ID: m.ID, WorkspaceID: acme, ProjectID: web, MemberID: m.MemberID,
			Role: m.Role, CreatedBy: alice, Now: now}); err != nil {
			t.Fatal(err)
		}
	}
	exec(t, pool, "UPDATE project_members SET deleted_at = $2 WHERE id = $1", seedMember(t, pool, acme, web, dave, 15, true), now)
	seedMember(t, pool, acme, ops, frank, 20, true)
	got, err := s.ListMembers(context.Background(), web)
	if err != nil || len(got) != len(want) {
		t.Fatalf("ListMembers() = %+v, %v; want %+v", got, err, want)
	}
	for i := range want {
		if got[i].ID != want[i].ID || got[i].ProjectID != want[i].ProjectID || got[i].MemberID != want[i].MemberID || got[i].Role != want[i].Role ||
			!got[i].CreatedAt.Equal(want[i].CreatedAt) {
			t.Errorf("ListMembers()[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
	if got, err := s.ListMembers(context.Background(), uuid.NewV7()); err != nil || len(got) != 0 {
		t.Errorf("ListMembers() of no project = %v, %v; want none", got, err)
	}
}

// RestoreMember makes the ended membership active again with the role
// given, at the moment and by the account given, and changes no other
// column of it, its id and created_at kept; every other membership keeps
// every column. bob's, an admin's, is restored as a guest's; then carol's,
// a guest's, as an admin's: the role given, whether below or above the
// ended one. An ended membership is stored before bob's (his in Ops) and
// one after it (carol's), so that neither the first nor the last ended row
// is the one restored.
func TestRestoreMember(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	seedMember(t, pool, acme, ops, bob, 20, false)
	bobs := seedMember(t, pool, acme, web, bob, 20, false)
	seedMember(t, pool, acme, web, alice, 15, true)
	carols := seedMember(t, pool, acme, web, carol, 5, false)
	for i, tt := range []struct {
		ended uuid.UUID
		role  shared.Role
		want  string
		by    uuid.UUID
	}{{bobs, shared.RoleGuest, "5", alice}, {carols, shared.RoleAdmin, "20", bob}} {
		others := tableRows(t, pool, "project_members", tt.ended)
		before := columns(t, pool, "project_members", tt.ended)
		later := now.Add(time.Duration(i+1) * time.Hour)

		if err := s.RestoreMember(context.Background(), tt.ended, tt.role, tt.by, later); err != nil {
			t.Fatal(err)
		}

		want := changed(before, map[string]string{"is_active": "true", "role": tt.want, "updated_by_id": `"` + tt.by.String() + `"`,
			"updated_at": `"` + later.Format("2006-01-02T15:04:05.999999") + `+00:00"`})
		if got := columns(t, pool, "project_members", tt.ended); !maps.Equal(got, want) {
			t.Errorf("the restored membership %d: %v\nwant %v", i, got, want)
		}
		if after := tableRows(t, pool, "project_members", tt.ended); after != others {
			t.Errorf("the other memberships after %d:\n%s\nwant\n%s", i, after, others)
		}
	}
}

// CountInactive counts the account's ended undeleted memberships of the
// workspace's projects: bob's of Web and of Ops in acme, 2; not his active
// one of Docs, nor his ended one of Old, deleted, nor his ended one of
// beta's project, which beta's count is; not carol's ended one of Web,
// which hers is. alice has none.
func TestCountInactive(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	web, ops := newProject(t, s, acme, "Web", "WEB", alice), newProject(t, s, acme, "Ops", "OPS", alice)
	docs, old := newProject(t, s, acme, "Docs", "DOC", alice), newProject(t, s, acme, "Old", "OLD", alice)
	betas := newProject(t, s, beta, "Web", "WEB", alice)
	seedMember(t, pool, acme, web, bob, 20, false)
	seedMember(t, pool, acme, ops, bob, 15, false)
	seedMember(t, pool, acme, docs, bob, 15, true)
	exec(t, pool, "UPDATE project_members SET deleted_at = $2 WHERE id = $1", seedMember(t, pool, acme, old, bob, 15, false), now)
	seedMember(t, pool, beta, betas, bob, 15, false)
	seedMember(t, pool, acme, web, carol, 5, false)
	seedMember(t, pool, acme, web, alice, 20, true)
	for _, tt := range []struct {
		name            string
		workspace, user uuid.UUID
		want            int
	}{{"bob in acme", acme, bob, 2}, {"bob in beta", beta, bob, 1}, {"carol in acme", acme, carol, 1}, {"alice in acme", acme, alice, 0}} {
		if got, err := s.CountInactive(context.Background(), tt.workspace, tt.user); err != nil || got != tt.want {
			t.Errorf("CountInactive() of %s = %d, %v; want %d", tt.name, got, err, tt.want)
		}
	}
}
