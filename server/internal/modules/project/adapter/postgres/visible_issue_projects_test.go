package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestVisibleIssueProjects(t *testing.T) {
	s, pool := newStore(t)
	owner := newAccount(t, pool, "owner@corp.com")
	guest := newAccount(t, pool, "guest@corp.com")
	admin := newAccount(t, pool, "admin@corp.com")
	outsider := newAccount(t, pool, "outsider@corp.com")
	acme, beta := newWorkspace(t, pool, "acme"), newWorkspace(t, pool, "beta")
	for actor, role := range map[uuid.UUID]int{guest: 5, admin: 20, outsider: 20} {
		exec(t, pool, "INSERT INTO workspace_members (id,workspace_id,member_id,role) VALUES ($1,$2,$3,$4)", uuid.NewV7(), acme, actor, role)
	}
	member := newProject(t, s, acme, "Member", "MEM", owner)
	own := newProject(t, s, acme, "Guest own", "OWN", owner)
	all := newProject(t, s, acme, "Guest all", "ALL", owner)
	ended := newProject(t, s, acme, "Ended", "END", owner)
	deletedMember := newProject(t, s, acme, "Deleted member", "DM", owner)
	deletedProject := newProject(t, s, acme, "Deleted project", "DP", owner)
	archived := newProject(t, s, acme, "Archived", "ARC", owner)
	foreign := newProject(t, s, beta, "Foreign", "FOR", owner)
	newProject(t, s, acme, "Public without membership", "PUB", owner)
	exec(t, pool, "UPDATE projects SET guest_view_all_features=false WHERE id=$1", own)
	exec(t, pool, "UPDATE projects SET guest_view_all_features=true WHERE id=$1", all)
	for _, actor := range []uuid.UUID{guest, admin} {
		for _, id := range []uuid.UUID{archived, deletedProject, deletedMember, ended, all, own, member} {
			role := 15
			if id == own || id == all {
				role = 5
			}
			mid := seedMember(t, pool, acme, id, actor, role, id != ended)
			if id == deletedMember {
				exec(t, pool, "UPDATE project_members SET deleted_at=$2 WHERE id=$1", mid, now)
			}
		}
		seedMember(t, pool, beta, foreign, actor, 15, true)
	}
	exec(t, pool, "UPDATE projects SET deleted_at=$2 WHERE id=$1", deletedProject, now)
	exec(t, pool, "UPDATE projects SET archived_at=$2 WHERE id=$1", archived, now)
	for _, actor := range []uuid.UUID{guest, admin} {
		got, err := s.VisibleIssueProjects(context.Background(), acme, actor)
		want := []app.VisibleIssueProject{{ID: member, Role: shared.RoleMember}, {ID: own, Role: shared.RoleGuest, OwnOnly: true}, {ID: all, Role: shared.RoleGuest}}
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("ordinary scope for %s: got %+v, %v; want %+v", actor, got, err, want)
		}
		// On precisely the same rows Historical must retain deleted and archived
		// projects/members while still excluding ended memberships.
		history, err := s.HistoricalIssueProjects(context.Background(), acme, actor)
		wantHistory := []app.HistoricalIssueProject{
			{ID: member, Role: shared.RoleMember}, {ID: own, Role: shared.RoleGuest, OwnOnly: true}, {ID: all, Role: shared.RoleGuest},
			{ID: deletedMember, Role: shared.RoleMember}, {ID: deletedProject, Role: shared.RoleMember}, {ID: archived, Role: shared.RoleMember},
		}
		if err != nil || !slices.Equal(history, wantHistory) {
			t.Errorf("historical scope for %s: got %+v, %v; want %+v", actor, history, err, wantHistory)
		}
	}
	for _, actor := range []uuid.UUID{outsider, uuid.NewV7()} {
		got, err := s.VisibleIssueProjects(context.Background(), acme, actor)
		if err != nil || got == nil || len(got) != 0 {
			t.Errorf("nonmember scope: got %+v, %v; want non-nil empty", got, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := s.VisibleIssueProjects(ctx, acme, guest); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("cancelled scope: %v, %v", got, err)
	}
}
