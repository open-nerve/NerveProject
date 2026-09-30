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

// MemberByID reads an undeleted membership, an ended one too;
// app.ErrNotFound for a deleted row and an id no row has.
func TestMemberByID(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	bobIn := joinAt(t, s, acme.ID, bob, shared.RoleMember, now)
	carolIn := joinAt(t, s, acme.ID, carol, shared.RoleGuest, now)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE id = $1", carolIn.ID)
	carolIn.IsActive = false
	gone := joinAt(t, s, newWorkspace(t, s, "Beta", "beta", alice).ID, bob, shared.RoleGuest, now)
	exec(t, pool, "UPDATE workspace_members SET deleted_at = $2 WHERE id = $1", gone.ID, now)

	for _, want := range []domain.Membership{bobIn, carolIn} {
		if got, err := s.MemberByID(context.Background(), want.ID); err != nil || got != want {
			t.Errorf("MemberByID(%s) = %+v, %v; want %+v", want.ID, got, err, want)
		}
	}
	for _, id := range []uuid.UUID{gone.ID, uuid.NewV7()} {
		if got, err := s.MemberByID(context.Background(), id); !errors.Is(err, app.ErrNotFound) || got != (domain.Membership{}) {
			t.Errorf("MemberByID(%s) = %+v, %v; want app.ErrNotFound", id, got, err)
		}
	}
}

// UpdateMemberRole sets the role of that membership only, with the updater
// and the time, and answers it as stored; a missing row is an error, not
// app.ErrNotFound.
func TestUpdateMemberRole(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	bobIn := joinAt(t, s, acme.ID, bob, shared.RoleMember, now)
	bobInBeta := joinAt(t, s, beta.ID, bob, shared.RoleMember, now)
	later := now.Add(time.Hour)

	got, err := s.UpdateMemberRole(context.Background(), bobIn.ID, shared.RoleGuest, alice, later)

	want := bobIn
	want.Role = shared.RoleGuest
	if err != nil || got != want {
		t.Errorf("UpdateMemberRole() = %+v, %v; want %+v", got, err, want)
	}
	var updatedBy uuid.UUID
	var updatedAt time.Time
	if err := pool.QueryRow(context.Background(), "SELECT updated_by_id, updated_at FROM workspace_members WHERE id = $1", bobIn.ID).
		Scan(&updatedBy, &updatedAt); err != nil || updatedBy != alice || !updatedAt.Equal(later) {
		t.Errorf("updated_by_id %s at %v, %v; want alice at %v", updatedBy, updatedAt, err, later)
	}
	if other, err := s.MemberByID(context.Background(), bobInBeta.ID); err != nil || other != bobInBeta {
		t.Errorf("bob in beta: %+v, %v; want it unchanged", other, err)
	}
	if _, err := s.UpdateMemberRole(context.Background(), uuid.NewV7(), shared.RoleGuest, alice, later); err == nil || errors.Is(err, app.ErrNotFound) {
		t.Errorf("UpdateMemberRole() of no row = %v, want an error that is not app.ErrNotFound", err)
	}
}
