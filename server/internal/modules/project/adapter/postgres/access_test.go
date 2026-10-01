package postgresadapter_test

import (
	"context"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ProjectFacts reads the undeleted project, archived or not, and the user's
// membership of it while it is active: a membership ended or deleted is
// none, a project deleted is not found. Each case is one fact changed from
// a project where the user is its member.
func TestProjectFacts(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	public := newProject(t, s, acme, "Web", "WEB", alice)
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
		{deleted, alice, shared.RoleAdmin}, {public, bob, shared.RoleAdmin}, {private, bob, shared.RoleMember}} {
		if err := s.CreateMember(ctx, app.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: m.project, MemberID: m.user, Role: m.role,
			CreatedBy: alice, Now: now}); err != nil {
			t.Fatal(err)
		}
	}
	exec(t, pool, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2", public, bob)
	exec(t, pool, "UPDATE project_members SET deleted_at = now() WHERE project_id = $1 AND member_id = $2", private, bob)

	tests := []struct {
		name          string
		project, user uuid.UUID
		want          app.AccessFacts
		found         bool
	}{
		{"a public project's guest", public, alice, app.AccessFacts{WorkspaceID: acme, Public: true, Member: true, Role: shared.RoleGuest}, true},
		{"a private project's admin", private, alice, app.AccessFacts{WorkspaceID: acme, Member: true, Role: shared.RoleAdmin}, true},
		{"an archived project's member", archived, alice, app.AccessFacts{WorkspaceID: acme, Public: true, Member: true, Role: shared.RoleMember}, true},
		{"a membership ended", public, bob, app.AccessFacts{WorkspaceID: acme, Public: true}, true},
		{"a membership deleted", private, bob, app.AccessFacts{WorkspaceID: acme}, true},
		{"no membership", archived, bob, app.AccessFacts{WorkspaceID: acme, Public: true}, true},
		{"a project deleted", deleted, alice, app.AccessFacts{}, false},
		{"no project", uuid.NewV7(), alice, app.AccessFacts{}, false},
	}
	for _, tt := range tests {
		got, found, err := s.ProjectFacts(ctx, tt.project, tt.user)
		if err != nil || found != tt.found || got != tt.want {
			t.Errorf("%s: ProjectFacts() = %+v, %v, %v; want %+v, %v", tt.name, got, found, err, tt.want, tt.found)
		}
	}
}
