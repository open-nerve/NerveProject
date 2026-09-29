package postgresadapter_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateWorkspace changes the fields the patch sets and no other, of that
// workspace only; it writes the updater and the clock's time, and answers
// the row as stored with the number of active members.
func TestUpdateWorkspace(t *testing.T) {
	size, other := "2-10", "500+"
	tests := []struct {
		name string
		p    domain.WorkspacePatch
		want func(w domain.Workspace) domain.Workspace
	}{
		{"every field", domain.WorkspacePatch{Name: ptr("Acme Inc"), OrganizationSize: &other, Timezone: ptr("Europe/Paris")},
			func(w domain.Workspace) domain.Workspace {
				w.Name, w.OrganizationSize, w.Timezone = "Acme Inc", &other, "Europe/Paris"
				return w
			}},
		{"the name", domain.WorkspacePatch{Name: ptr("研发部")}, func(w domain.Workspace) domain.Workspace { w.Name = "研发部"; return w }},
		{"the organization size", domain.WorkspacePatch{OrganizationSize: &other},
			func(w domain.Workspace) domain.Workspace { w.OrganizationSize = &other; return w }},
		{"the time zone", domain.WorkspacePatch{Timezone: ptr("Asia/Tokyo")},
			func(w domain.Workspace) domain.Workspace { w.Timezone = "Asia/Tokyo"; return w }},
		{"nothing", domain.WorkspacePatch{}, func(w domain.Workspace) domain.Workspace { return w }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
			acme, err := s.CreateWorkspace(context.Background(), app.WorkspaceRow{
				ID: uuid.NewV7(), Name: "Acme", Slug: "acme", OrganizationSize: &size, Timezone: "UTC", CreatedBy: alice, Now: now,
			})
			if err != nil {
				t.Fatal(err)
			}
			join(t, s, acme.ID, alice, shared.RoleAdmin)
			join(t, s, acme.ID, bob, shared.RoleMember)
			join(t, s, acme.ID, carol, shared.RoleGuest)
			exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE member_id = $1", carol)
			beta := newWorkspace(t, s, "Beta", "beta", alice)
			later := now.Add(time.Hour)

			got, err := s.UpdateWorkspace(context.Background(), acme.ID, tt.p, bob, later)

			want := tt.want(acme)
			want.TotalMembers, want.UpdatedAt = 2, later
			if err != nil || !sameWorkspace(got, want) {
				t.Fatalf("UpdateWorkspace() = %+v, %v; want %+v", got, err, want)
			}
			if read, err := s.WorkspaceBySlug(context.Background(), "acme"); err != nil || !sameWorkspace(read, want) {
				t.Errorf("read back: %+v, %v; want %+v", read, err, want)
			}
			var updatedBy, createdBy uuid.UUID
			if err := pool.QueryRow(context.Background(), "SELECT updated_by_id, created_by_id FROM workspaces WHERE id = $1", acme.ID).
				Scan(&updatedBy, &createdBy); err != nil || updatedBy != bob || createdBy != alice {
				t.Errorf("updated_by_id %s, created_by_id %s, %v; want bob, alice", updatedBy, createdBy, err)
			}
			beta.TotalMembers = 1
			if read, err := s.WorkspaceBySlug(context.Background(), "beta"); err != nil || !sameWorkspace(read, beta) {
				t.Errorf("the other workspace: %+v, %v; want it unchanged, %+v", read, err, beta)
			}
		})
	}
}

// A workspace id without a row is an error, not app.ErrNotFound: the caller
// holds the row's lock, so its absence is not an answer to give.
func TestUpdateWorkspaceWithoutARowIsAnError(t *testing.T) {
	s, _ := newStore(t)
	got, err := s.UpdateWorkspace(context.Background(), uuid.NewV7(), domain.WorkspacePatch{Name: ptr("Acme")}, uuid.NewV7(), now)
	if err == nil || errors.Is(err, app.ErrNotFound) || got != (domain.Workspace{}) {
		t.Errorf("UpdateWorkspace() = %+v, %v; want an error that is not app.ErrNotFound", got, err)
	}
}

func ptr[T any](v T) *T { return &v }
