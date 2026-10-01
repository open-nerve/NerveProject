package postgresadapter_test

import (
	"context"
	"maps"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
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
