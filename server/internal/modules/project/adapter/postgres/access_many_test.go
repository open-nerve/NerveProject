package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ProjectFactsMany is ProjectFacts for many accounts at once (M7 P1b spec
// 2.7), on TestProjectFacts' projects and memberships: the undeleted
// project's facts, archived or not, and of the accounts asked, those whose
// membership of it is active, with their project roles, and no one else:
// not bob's ended membership of Web nor his deleted one of Secret, not
// carol's of Web in Secret, not an account that does not exist, nor one
// not asked. A project with no active member among them is found with no
// role; a deleted project, or none, is not found.
func TestProjectFactsMany(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	carol := newAccount(t, pool, "carol@corp.com")
	acme := newWorkspace(t, pool, "acme")
	public := newProject(t, s, acme, "Web", "WEB", alice)
	exec(t, pool, "UPDATE projects SET guest_view_all_features = true WHERE id = $1", public)
	private := newProject(t, s, acme, "Secret", "SEC", alice)
	exec(t, pool, "UPDATE projects SET network = 0 WHERE id = $1", private)
	archived := newProject(t, s, acme, "Old", "OLD", alice)
	exec(t, pool, "UPDATE projects SET archived_at = now() WHERE id = $1", archived)
	deleted := newProject(t, s, acme, "Gone", "GONE", alice)
	exec(t, pool, "UPDATE projects SET deleted_at = now() WHERE id = $1", deleted)
	for _, m := range []struct {
		project, user uuid.UUID
		role          shared.Role
	}{{public, alice, shared.RoleGuest}, {private, alice, shared.RoleAdmin}, {archived, alice, shared.RoleMember},
		{deleted, alice, shared.RoleAdmin}, {public, bob, shared.RoleAdmin}, {private, bob, shared.RoleMember},
		{public, carol, shared.RoleAdmin}} {
		if err := s.CreateMember(ctx, app.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: m.project, MemberID: m.user, Role: m.role,
			CreatedBy: alice, Now: now}); err != nil {
			t.Fatal(err)
		}
	}
	exec(t, pool, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2", public, bob)
	exec(t, pool, "UPDATE project_members SET deleted_at = now() WHERE project_id = $1 AND member_id = $2", private, bob)

	nobody := uuid.NewV7()
	everyone := []uuid.UUID{alice, bob, carol, nobody, alice}
	type roles = map[uuid.UUID]shared.Role
	tests := []struct {
		name     string
		project  uuid.UUID
		accounts []uuid.UUID
		want     app.AccessFactsMany
		found    bool
	}{
		{"Web", public, everyone, app.AccessFactsMany{WorkspaceID: acme, Public: true, GuestViewAll: true,
			Roles: roles{alice: shared.RoleGuest, carol: shared.RoleAdmin}}, true},
		{"Secret", private, everyone, app.AccessFactsMany{WorkspaceID: acme, Roles: roles{alice: shared.RoleAdmin}}, true},
		{"Old, archived", archived, everyone, app.AccessFactsMany{WorkspaceID: acme, Public: true, Roles: roles{alice: shared.RoleMember}}, true},
		{"Web, bob and nobody asked", public, []uuid.UUID{bob, nobody}, app.AccessFactsMany{WorkspaceID: acme, Public: true,
			GuestViewAll: true, Roles: roles{}}, true},
		{"Secret, carol asked", private, []uuid.UUID{carol}, app.AccessFactsMany{WorkspaceID: acme, Roles: roles{}}, true},
		{"Web, no one asked", public, nil, app.AccessFactsMany{WorkspaceID: acme, Public: true, GuestViewAll: true, Roles: roles{}}, true},
		{"Gone, deleted", deleted, everyone, app.AccessFactsMany{}, false},
		{"no project", uuid.NewV7(), everyone, app.AccessFactsMany{}, false},
	}
	for _, tt := range tests {
		got, found, err := s.ProjectFactsMany(ctx, tt.project, tt.accounts)
		if err != nil || found != tt.found || !sameFactsMany(got, tt.want) || found && got.Roles == nil {
			t.Errorf("%s: ProjectFactsMany() = %+v, %v, %v; want %+v, %v", tt.name, got, found, err, tt.want, tt.found)
		}
	}
}

// sameFactsMany reports whether a and b hold the same facts and roles.
func sameFactsMany(a, b app.AccessFactsMany) bool {
	return a.WorkspaceID == b.WorkspaceID && a.Public == b.Public && a.GuestViewAll == b.GuestViewAll && maps.Equal(a.Roles, b.Roles)
}

// ProjectFactsMany reads in the transaction ctx carries, as ProjectFacts
// does: it sees the membership the transaction wrote before committing, and
// once that is rolled back, none. A failed read is its error, not "not
// found".
func TestProjectFactsManyReadsInTheTransaction(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	web := newProject(t, s, acme, "Web", "WEB", alice)
	var inside app.AccessFactsMany
	var found bool
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		if err := s.CreateMember(ctx, app.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, MemberID: bob,
			Role: shared.RoleMember, CreatedBy: alice, Now: now}); err != nil {
			return err
		}
		var err error
		inside, found, err = s.ProjectFactsMany(ctx, web, []uuid.UUID{bob})
		return errors.Join(err, errors.New("roll back"))
	})
	if want := (app.AccessFactsMany{WorkspaceID: acme, Public: true, Roles: map[uuid.UUID]shared.Role{bob: shared.RoleMember}}); err == nil ||
		!found || !sameFactsMany(inside, want) {
		t.Errorf("in the transaction: %+v, %v, %v; want %+v, found", inside, found, err, want)
	}
	if f, found, err := s.ProjectFactsMany(context.Background(), web, []uuid.UUID{bob}); err != nil || !found || len(f.Roles) != 0 {
		t.Errorf("after the rollback: %+v, %v, %v; want bob no member", f, found, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if f, found, err := s.ProjectFactsMany(cancelled, web, []uuid.UUID{alice}); !errors.Is(err, context.Canceled) || found ||
		!sameFactsMany(f, app.AccessFactsMany{}) {
		t.Errorf("on a cancelled context: %+v, %v, %v; want context.Canceled, not found", f, found, err)
	}
}
