package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// MemberProjectFacts reads, of a workspace's undeleted projects, archived
// or not, those the user is an active member of, each with ProjectFacts'
// facts and his project role, in id order (M7 P1c spec 2.5), on
// TestProjectFacts' projects and memberships and a project of another
// workspace: not bob's ended membership of Web nor his deleted one of
// Secret, not alice's of the deleted Gone, not her membership of the other
// workspace's project when acme is asked, nor anyone's of a workspace that
// does not exist.
func TestMemberProjectFacts(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	carol := newAccount(t, pool, "carol@corp.com")
	acme, other := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "other")
	public := newProject(t, s, acme, "Web", "WEB", alice)
	exec(t, pool, "UPDATE projects SET guest_view_all_features = true WHERE id = $1", public)
	private := newProject(t, s, acme, "Secret", "SEC", alice)
	exec(t, pool, "UPDATE projects SET network = 0 WHERE id = $1", private)
	archived := newProject(t, s, acme, "Old", "OLD", alice)
	exec(t, pool, "UPDATE projects SET archived_at = now() WHERE id = $1", archived)
	deleted := newProject(t, s, acme, "Gone", "GONE", alice)
	exec(t, pool, "UPDATE projects SET deleted_at = now() WHERE id = $1", deleted)
	elsewhere := newProject(t, s, other, "Elsewhere", "ELS", alice)
	for _, m := range []struct {
		workspace, project, user uuid.UUID
		role                     shared.Role
	}{{acme, public, alice, shared.RoleGuest}, {acme, private, alice, shared.RoleAdmin}, {acme, archived, alice, shared.RoleMember},
		{acme, deleted, alice, shared.RoleAdmin}, {other, elsewhere, alice, shared.RoleMember}, {acme, public, bob, shared.RoleAdmin},
		{acme, private, bob, shared.RoleMember}, {acme, public, carol, shared.RoleAdmin}} {
		if err := s.CreateMember(ctx, app.MemberRow{ID: uuid.NewV7(), WorkspaceID: m.workspace, ProjectID: m.project, MemberID: m.user,
			Role: m.role, CreatedBy: alice, Now: now}); err != nil {
			t.Fatal(err)
		}
	}
	exec(t, pool, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2", public, bob)
	exec(t, pool, "UPDATE project_members SET deleted_at = now() WHERE project_id = $1 AND member_id = $2", private, bob)

	byID := func(facts ...app.MemberAccessFacts) []app.MemberAccessFacts {
		return slices.SortedFunc(slices.Values(facts), func(a, b app.MemberAccessFacts) int { return a.ProjectID.Compare(b.ProjectID) })
	}
	tests := []struct {
		name            string
		workspace, user uuid.UUID
		want            []app.MemberAccessFacts
	}{
		{"alice in acme", acme, alice, byID(
			app.MemberAccessFacts{ProjectID: public, Public: true, Role: shared.RoleGuest, GuestViewAll: true},
			app.MemberAccessFacts{ProjectID: private, Role: shared.RoleAdmin},
			app.MemberAccessFacts{ProjectID: archived, Public: true, Role: shared.RoleMember})},
		{"alice in the other workspace", other, alice, []app.MemberAccessFacts{{ProjectID: elsewhere, Public: true, Role: shared.RoleMember}}},
		{"bob, ended and deleted", acme, bob, nil},
		{"carol", acme, carol, []app.MemberAccessFacts{{ProjectID: public, Public: true, Role: shared.RoleAdmin, GuestViewAll: true}}},
		{"carol in the other workspace", other, carol, nil},
		{"no account", acme, uuid.NewV7(), nil},
		{"no workspace", uuid.NewV7(), alice, nil},
	}
	for _, tt := range tests {
		got, err := s.MemberProjectFacts(ctx, tt.workspace, tt.user)
		if err != nil || !slices.Equal(got, tt.want) {
			t.Errorf("%s: MemberProjectFacts() = %+v, %v; want %+v", tt.name, got, err, tt.want)
		}
	}
}

// MemberProjectFacts reads in the transaction ctx carries, as ProjectFacts
// does: it sees the membership the transaction wrote before committing, and
// once that is rolled back, none. A failed read is its error.
func TestMemberProjectFactsReadsInTheTransaction(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web := newProject(t, s, acme, "Web", "WEB", alice)
	var inside []app.MemberAccessFacts
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		if err := s.CreateMember(ctx, app.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, MemberID: bob,
			Role: shared.RoleMember, CreatedBy: alice, Now: now}); err != nil {
			return err
		}
		var err error
		inside, err = s.MemberProjectFacts(ctx, acme, bob)
		return errors.Join(err, errors.New("roll back"))
	})
	if want := []app.MemberAccessFacts{{ProjectID: web, Public: true, Role: shared.RoleMember}}; err == nil || !slices.Equal(inside, want) {
		t.Errorf("in the transaction: %+v, %v; want %+v", inside, err, want)
	}
	if got, err := s.MemberProjectFacts(context.Background(), acme, bob); err != nil || len(got) != 0 {
		t.Errorf("after the rollback: %+v, %v; want none", got, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := s.MemberProjectFacts(cancelled, acme, alice); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("on a cancelled context: %+v, %v; want context.Canceled and none", got, err)
	}
}
