package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"time"
	"uuid"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// joinAt makes user a member of workspace with role at the time at, and
// returns the membership as stored.
func joinAt(t *testing.T, s *postgresadapter.Store, workspace, user uuid.UUID, role shared.Role, at time.Time) domain.Membership {
	t.Helper()
	return joinAtWithID(t, s, uuid.NewV7(), workspace, user, role, at)
}

// joinAtWithID is joinAt with the membership's id given.
func joinAtWithID(t *testing.T, s *postgresadapter.Store, id, workspace, user uuid.UUID, role shared.Role, at time.Time) domain.Membership {
	t.Helper()
	m := app.MemberRow{ID: id, WorkspaceID: workspace, MemberID: user, Role: role, CreatedBy: user, Now: at}
	if err := s.CreateMember(context.Background(), m); err != nil {
		t.Fatal(err)
	}
	return domain.Membership{ID: m.ID, WorkspaceID: workspace, MemberID: user, Role: role, IsActive: true, CreatedAt: at}
}

// ListMembers is the workspace's undeleted memberships, the ended ones too,
// by the time they began, then by id; not a deleted row, not another
// workspace's.
func TestListMembers(t *testing.T) {
	s, pool := newStore(t)
	var ids []uuid.UUID
	for _, email := range []string{"alice@corp.com", "bob@corp.com", "carol@corp.com", "dave@corp.com", "eve@corp.com"} {
		ids = append(ids, newAccount(t, pool, email))
	}
	alice, bob, carol, dave, eve := ids[0], ids[1], ids[2], ids[3], ids[4]
	acme, err := s.CreateWorkspace(context.Background(), app.WorkspaceRow{ID: uuid.NewV7(), Name: "Acme", Slug: "acme", Timezone: "UTC", CreatedBy: alice, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	beta := newWorkspace(t, s, "Beta", "beta", bob)
	carolIn := joinAt(t, s, acme.ID, carol, shared.RoleGuest, now.Add(time.Minute))
	bobIn := joinAt(t, s, acme.ID, bob, shared.RoleMember, now.Add(2*time.Minute))
	// alice and eve begin at the same time: eve's membership has the smaller
	// id and is stored after alice's, so only the tie-break on id lists it
	// first.
	eveID := uuid.NewV7()
	aliceIn := joinAt(t, s, acme.ID, alice, shared.RoleAdmin, now)
	eveIn := joinAtWithID(t, s, eveID, acme.ID, eve, shared.RoleMember, now)
	joinAt(t, s, acme.ID, dave, shared.RoleMember, now)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE member_id = $1", carol)
	exec(t, pool, "UPDATE workspace_members SET deleted_at = $2 WHERE member_id = $1", dave, now)
	joinAt(t, s, beta.ID, alice, shared.RoleGuest, now)

	got, err := s.ListMembers(context.Background(), acme.ID)

	carolIn.IsActive = false
	if want := []domain.Membership{eveIn, aliceIn, carolIn, bobIn}; err != nil || !slices.Equal(got, want) {
		t.Errorf("ListMembers(acme) = %+v, %v; want %+v", got, err, want)
	}
	if got, err := s.ListMembers(context.Background(), uuid.NewV7()); err != nil || len(got) != 0 {
		t.Errorf("ListMembers() of no workspace = %+v, %v; want none", got, err)
	}
}
