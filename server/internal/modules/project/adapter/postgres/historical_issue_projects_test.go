package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Historical scope retains deleted and archived memberships, but excludes ended
// memberships, other actors and workspaces, and projects with no membership.
func TestHistoricalIssueProjects(t *testing.T) {
	s, pool := newStore(t)
	owner := newAccount(t, pool, "owner@corp.com")
	guest := newAccount(t, pool, "guest@corp.com")
	adminGuest := newAccount(t, pool, "admin-guest@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	for actor, role := range map[uuid.UUID]int{guest: 5, adminGuest: 20} {
		exec(t, pool, "INSERT INTO workspace_members (id, workspace_id, member_id, role) VALUES ($1,$2,$3,$4)", uuid.NewV7(), acme, actor, role)
	}
	deletedProject := newProject(t, s, acme, "Z deleted", "ZDEL", owner)
	deletedMember := newProject(t, s, acme, "Y member", "YMEM", owner)
	archived := newProject(t, s, acme, "X archived", "XARC", owner)
	liveProject := newProject(t, s, acme, "W live", "WLIV", owner)
	allGuest := newProject(t, s, acme, "V all", "VALL", owner)
	ended := newProject(t, s, acme, "Ended", "END", owner)
	foreign := newProject(t, s, beta, "Foreign", "FOR", owner)
	noMembership := newProject(t, s, acme, "Public without membership", "NONE", owner)
	seedMember(t, pool, acme, noMembership, owner, 20, true)
	exec(t, pool, "UPDATE projects SET network=0 WHERE id=$1", liveProject)
	exec(t, pool, "UPDATE projects SET guest_view_all_features=false WHERE id=ANY($1::uuid[])",
		[]uuid.UUID{deletedProject, deletedMember, archived, liveProject})
	exec(t, pool, "UPDATE projects SET guest_view_all_features=true WHERE id=$1", allGuest)
	for _, actor := range []uuid.UUID{guest, adminGuest} {
		// Insert in reverse project order so insertion order cannot satisfy sorting.
		for _, id := range []uuid.UUID{allGuest, liveProject, archived, deletedMember, deletedProject} {
			role := 5
			if id == liveProject {
				role = 15
			}
			membership := seedMember(t, pool, acme, id, actor, role, true)
			if id == deletedMember || id == deletedProject {
				exec(t, pool, "UPDATE project_members SET deleted_at=$2 WHERE id=$1", membership, now)
			}
		}
		seedMember(t, pool, acme, ended, actor, 5, false)
		seedMember(t, pool, beta, foreign, actor, 15, true)
	}
	exec(t, pool, "UPDATE projects SET deleted_at=$2 WHERE id=$1", deletedProject, now)
	exec(t, pool, "UPDATE projects SET archived_at=$2 WHERE id=$1", archived, now)
	for _, actor := range []uuid.UUID{guest, adminGuest} {
		got, err := s.HistoricalIssueProjects(context.Background(), acme, actor)
		want := []app.HistoricalIssueProject{
			{ID: deletedProject, Role: shared.RoleGuest, OwnOnly: true},
			{ID: deletedMember, Role: shared.RoleGuest, OwnOnly: true},
			{ID: archived, Role: shared.RoleGuest, OwnOnly: true},
			{ID: liveProject, Role: shared.RoleMember},
			{ID: allGuest, Role: shared.RoleGuest},
		}
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("historical scope and project role visibility for %s: got %+v, %v; want %+v", actor, got, err, want)
		}
	}
	got, err := s.HistoricalIssueProjects(context.Background(), acme, uuid.NewV7())
	if err != nil || len(got) != 0 {
		t.Fatalf("actor without memberships: got %+v, %v", got, err)
	}
}
