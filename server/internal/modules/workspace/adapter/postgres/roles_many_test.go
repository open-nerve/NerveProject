package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ActiveRoles is ActiveRole for many accounts at once (M7 P1b spec 2.7): in
// each workspace of the fixture, of the accounts asked, those whose
// membership is active, its row not deleted and the workspace not deleted,
// with their roles, and no one else: not carol, removed from beta, nor bob
// from beta2, nor alice from left; not alice in gone, deleted, nor in
// dropped, her row deleted; not an account that does not exist. An account
// asked twice is answered once; one not asked is not answered, though his
// membership is active.
func TestActiveRoles(t *testing.T) {
	f := newFixture(t)
	nobody := uuid.NewV7()
	everyone := []uuid.UUID{f.alice, f.bob, f.carol, nobody, f.alice}
	tests := []struct {
		name      string
		workspace uuid.UUID
		accounts  []uuid.UUID
		want      map[uuid.UUID]shared.Role
	}{
		{"beta", f.beta.ID, everyone, map[uuid.UUID]shared.Role{f.alice: shared.RoleAdmin, f.bob: shared.RoleMember}},
		{"acme", f.acme.ID, everyone, map[uuid.UUID]shared.Role{f.alice: shared.RoleGuest, f.bob: shared.RoleAdmin}},
		{"beta2", f.beta2.ID, everyone, map[uuid.UUID]shared.Role{f.alice: shared.RoleMember}},
		{"gone, deleted", f.gone.ID, everyone, map[uuid.UUID]shared.Role{}},
		{"left", f.left.ID, everyone, map[uuid.UUID]shared.Role{f.bob: shared.RoleAdmin}},
		{"dropped", f.dropped.ID, everyone, map[uuid.UUID]shared.Role{f.bob: shared.RoleAdmin}},
		{"bobs", f.bobs.ID, everyone, map[uuid.UUID]shared.Role{f.bob: shared.RoleAdmin}},
		{"no workspace", uuid.NewV7(), everyone, map[uuid.UUID]shared.Role{}},
		{"beta, alice and nobody asked", f.beta.ID, []uuid.UUID{f.alice, nobody}, map[uuid.UUID]shared.Role{f.alice: shared.RoleAdmin}},
		{"acme, carol asked", f.acme.ID, []uuid.UUID{f.carol}, map[uuid.UUID]shared.Role{}},
		{"beta, no one asked", f.beta.ID, nil, map[uuid.UUID]shared.Role{}},
	}
	for _, tt := range tests {
		got, err := f.s.ActiveRoles(context.Background(), tt.workspace, tt.accounts)
		if err != nil || got == nil || !maps.Equal(got, tt.want) {
			t.Errorf("%s: ActiveRoles() = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
}

// ActiveRoles reads in the transaction ctx carries, as ActiveRole does: it
// sees the membership the transaction wrote before committing, and once that
// is rolled back, none. A failed read is its error, not "no member".
func TestActiveRolesReadsInTheTransaction(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	w := newWorkspace(t, s, "Acme", "acme", alice)
	var inside map[uuid.UUID]shared.Role
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		if err := s.CreateMember(ctx, app.MemberRow{ID: uuid.NewV7(), WorkspaceID: w.ID, MemberID: bob, Role: shared.RoleMember, CreatedBy: alice,
			Now: now}); err != nil {
			return err
		}
		var err error
		inside, err = s.ActiveRoles(ctx, w.ID, []uuid.UUID{alice, bob})
		return errors.Join(err, errors.New("roll back"))
	})
	if want := map[uuid.UUID]shared.Role{alice: shared.RoleAdmin, bob: shared.RoleMember}; err == nil || !maps.Equal(inside, want) {
		t.Errorf("in the transaction: %v, %v; want %v", inside, err, want)
	}
	if got, err := s.ActiveRoles(context.Background(), w.ID, []uuid.UUID{alice, bob}); err != nil ||
		!maps.Equal(got, map[uuid.UUID]shared.Role{alice: shared.RoleAdmin}) {
		t.Errorf("after the rollback: %v, %v; want alice alone", got, err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := s.ActiveRoles(cancelled, w.ID, []uuid.UUID{alice}); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("on a cancelled context: %v, %v; want context.Canceled, no answer", got, err)
	}
}
