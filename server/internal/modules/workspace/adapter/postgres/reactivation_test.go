package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ReactivateMember makes the user's ended, undeleted membership of the
// workspace active again at the time given, its role kept, and writes
// nothing else: updated_by_id stays alice's, who ended it (M3 design 3.11).
// bob's deleted membership of acme, stored before his live one or after it,
// his ended membership of beta and carol's ended one of acme keep every
// column, and his reactivated one its other columns. carol's, a guest's,
// reactivated next, keeps her role and her other columns, as every other
// row keeps each of its own: no reactivation makes its member an admin,
// which bob, one already, can't show. A pair with no ended, undeleted
// membership is an error that is not app.ErrNotFound, and changes nothing:
// carol's in beta, of which there is none, bob's in gamma, deleted, and
// bob's in acme, active by then, whose moment stays.
func TestReactivateMember(t *testing.T) {
	for _, deletedFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("the deleted membership stored first %v", deletedFirst), func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
			acme, beta, gamma := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice),
				newWorkspace(t, s, "Gamma", "gamma", alice)
			deleted := func(workspace uuid.UUID) {
				exec(t, pool, `INSERT INTO workspace_members (id, workspace_id, member_id, role, is_active, created_by_id, updated_by_id,
					created_at, updated_at, deleted_at) VALUES ($1, $2, $3, 15, false, $3, $3, $4, $4, $4)`, uuid.NewV7(), workspace, bob,
					now.Add(-time.Hour))
			}
			if deletedFirst {
				deleted(acme.ID)
			}
			bobIn := joinAt(t, s, acme.ID, bob, shared.RoleAdmin, now)
			if !deletedFirst {
				deleted(acme.ID)
			}
			carolIn := joinAt(t, s, acme.ID, carol, shared.RoleGuest, now)
			joinAt(t, s, beta.ID, bob, shared.RoleMember, now)
			deleted(gamma.ID)
			for _, w := range []uuid.UUID{acme.ID, beta.ID} {
				if err := s.EndMember(context.Background(), w, bob, alice, now); err != nil {
					t.Fatal(err)
				}
			}
			if err := s.EndMember(context.Background(), acme.ID, carol, alice, now); err != nil {
				t.Fatal(err)
			}
			before := tableRows(t, pool, "workspace_members", []uuid.UUID{bobIn.ID}, "is_active", "updated_at")
			later := now.Add(time.Hour)

			if err := s.ReactivateMember(context.Background(), acme.ID, bob, later); err != nil {
				t.Fatal(err)
			}

			if got, want := stamp(t, pool, "workspace_members", bobIn.ID), fmt.Sprintf("active at %s by %s", at(t, pool, later), alice); got != want {
				t.Errorf("bob's membership of acme: %s, want %s", got, want)
			}
			if after := tableRows(t, pool, "workspace_members", []uuid.UUID{bobIn.ID}, "is_active", "updated_at"); after != before {
				t.Errorf("the memberships, bob's in acme without the columns written:\n%s\nwant them as they were:\n%s", after, before)
			}
			before = tableRows(t, pool, "workspace_members", []uuid.UUID{carolIn.ID}, "is_active", "updated_at")
			if err := s.ReactivateMember(context.Background(), acme.ID, carol, later); err != nil {
				t.Fatal(err)
			}
			if got, want := stamp(t, pool, "workspace_members", carolIn.ID), fmt.Sprintf("active at %s by %s", at(t, pool, later), alice); got != want {
				t.Errorf("carol's membership of acme: %s, want %s", got, want)
			}
			if after := tableRows(t, pool, "workspace_members", []uuid.UUID{carolIn.ID}, "is_active", "updated_at"); after != before {
				t.Errorf("the memberships, carol's in acme without the columns written:\n%s\nwant them as they were:\n%s", after, before)
			}
			before = tableRows(t, pool, "workspace_members", nil)
			for _, pair := range []struct {
				name            string
				workspace, user uuid.UUID
			}{{"carol in beta", beta.ID, carol}, {"bob in gamma", gamma.ID, bob}, {"bob in acme, active", acme.ID, bob}} {
				if err := s.ReactivateMember(context.Background(), pair.workspace, pair.user, later.Add(time.Hour)); err == nil ||
					errors.Is(err, app.ErrNotFound) {
					t.Errorf("ReactivateMember() of %s = %v, want an error that is not app.ErrNotFound", pair.name, err)
				}
			}
			if after := tableRows(t, pool, "workspace_members", nil); after != before {
				t.Errorf("the memberships after reactivating none:\n%s\nwant them as they were:\n%s", after, before)
			}
		})
	}
}
