package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// invitationsToBob stores, in alice's workspaces slugs, invitations to bob's
// address that LockInvitationsToDelete finds and leaves: bob's pending one
// to the first, his declined one to the second, his pending one to the
// third, whose member he is not, all three by alice; two earlier ones to
// his address in the first, one accepted, one deleted while pending;
// carol's pending one to the first, and one to the second to an address
// that only ends with his. It returns the three it finds for his address.
func invitationsToBob(t *testing.T, s *postgresadapter.Store, pool *pgxpool.Pool, alice uuid.UUID, slugs [3]string) []uuid.UUID {
	t.Helper()
	var ws [3]uuid.UUID
	for i, slug := range slugs {
		ws[i] = newWorkspace(t, s, slug, slug, alice).ID
	}
	exec(t, pool, `INSERT INTO workspace_member_invites (id, workspace_id, email, role, accepted, responded_at, created_by_id,
		updated_by_id, created_at, updated_at, deleted_at) VALUES ($1, $2, 'bob@corp.com', 15, true, $4, $3, $3, $4, $4, $4)`,
		uuid.NewV7(), ws[0], alice, now.Add(-time.Hour))
	exec(t, pool, `INSERT INTO workspace_member_invites (id, workspace_id, email, role, created_by_id, updated_by_id, created_at,
		updated_at, deleted_at) VALUES ($1, $2, 'bob@corp.com', 5, $3, $3, $4, $4, $4)`, uuid.NewV7(), ws[0], alice, now.Add(-time.Hour))
	found := []uuid.UUID{
		invite(t, s, ws[0], "bob@corp.com", shared.RoleGuest, alice).ID,
		invite(t, s, ws[1], "bob@corp.com", shared.RoleMember, alice).ID,
		invite(t, s, ws[2], "bob@corp.com", shared.RoleAdmin, alice).ID,
	}
	exec(t, pool, "UPDATE workspace_member_invites SET responded_at = $2 WHERE id = $1", found[1], now)
	invite(t, s, ws[0], "carol@corp.com", shared.RoleMember, alice)
	invite(t, s, ws[1], "rebob@corp.com", shared.RoleMember, alice)
	return found
}

// leftEmpty is a world of LockInvitationsToDelete's: asked are bob's
// workspaces acme, where he is alone; beta, where carol is an active
// guest, an active member all the same; gamma, where her membership ended;
// delta, where it is deleted. Each has a pending invitation of alice's, to
// dave: found are acme's, gamma's and delta's, though alice wrote them;
// beta's is left. acme's declined one, to erin, and its one deleted while
// pending, to frank, are left; so is the pending one of zeta, alice's, a
// workspace not asked about with no active member.
type leftEmpty struct {
	alice, bob   uuid.UUID
	asked, found []uuid.UUID
}

func newLeftEmpty(t *testing.T, s *postgresadapter.Store, pool *pgxpool.Pool) leftEmpty {
	t.Helper()
	w := leftEmpty{alice: newAccount(t, pool, "alice@corp.com"), bob: newAccount(t, pool, "bob@corp.com")}
	carol := newAccount(t, pool, "carol@corp.com")
	ws := map[string]uuid.UUID{"zeta": newWorkspace(t, s, "zeta", "zeta", w.alice).ID}
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1", ws["zeta"])
	for _, name := range []string{"acme", "beta", "gamma", "delta"} {
		ws[name] = newWorkspace(t, s, name, name, w.bob).ID
		w.asked = append(w.asked, ws[name])
	}
	join(t, s, ws["beta"], carol, shared.RoleGuest)
	for _, name := range []string{"gamma", "delta"} {
		join(t, s, ws[name], carol, shared.RoleMember)
	}
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2", ws["gamma"], carol)
	exec(t, pool, "UPDATE workspace_members SET deleted_at = $3 WHERE workspace_id = $1 AND member_id = $2", ws["delta"], carol, now)
	for _, name := range []string{"acme", "beta", "gamma", "delta", "zeta"} {
		id := invite(t, s, ws[name], "dave@corp.com", shared.RoleMember, w.alice).ID
		if name == "acme" || name == "gamma" || name == "delta" {
			w.found = append(w.found, id)
		}
	}
	declined := invite(t, s, ws["acme"], "erin@corp.com", shared.RoleGuest, w.alice).ID
	exec(t, pool, "UPDATE workspace_member_invites SET responded_at = $2 WHERE id = $1", declined, now)
	exec(t, pool, `INSERT INTO workspace_member_invites (id, workspace_id, email, role, created_by_id, updated_by_id, created_at,
		updated_at, deleted_at) VALUES ($1, $2, 'frank@corp.com', 15, $3, $3, $4, $4, $4)`, uuid.NewV7(), ws["acme"], w.alice, now.Add(-time.Hour))
	return w
}

// DeleteInvitations soft-deletes the undeleted invitations of the ids it is
// given, by the account and at the time given, in the transaction it runs
// in, and changes nothing else (M3 design 3.9; ruling F-1). Given bob's
// pending invitation to acme, his declined one to beta, both alice's, and
// his one to acme deleted earlier, it deletes the first two, by grace,
// whom no invitation names, and the third keeps its moment; his pending
// one to gamma, which it is not given, and carol's keep every column. Run
// first in a transaction that rolls back, it leaves every row as it was.
func TestDeleteInvitations(t *testing.T) {
	s, pool := newStore(t)
	alice, grace := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "grace@corp.com")
	acme, beta, gamma := newWorkspace(t, s, "acme", "acme", alice).ID, newWorkspace(t, s, "beta", "beta", alice).ID,
		newWorkspace(t, s, "gamma", "gamma", alice).ID
	pending, declined := invite(t, s, acme, "bob@corp.com", shared.RoleGuest, alice).ID, invite(t, s, beta, "bob@corp.com", shared.RoleMember, alice).ID
	exec(t, pool, "UPDATE workspace_member_invites SET responded_at = $2 WHERE id = $1", declined, now)
	deleted := uuid.NewV7()
	exec(t, pool, `INSERT INTO workspace_member_invites (id, workspace_id, email, role, created_by_id, updated_by_id, created_at,
		updated_at, deleted_at) VALUES ($1, $2, 'bob@corp.com', 5, $3, $3, $4, $4, $4)`, deleted, acme, alice, now.Add(-time.Hour))
	invite(t, s, gamma, "bob@corp.com", shared.RoleMember, alice)
	invite(t, s, acme, "carol@corp.com", shared.RoleMember, alice)
	given, written := []uuid.UUID{pending, declined, deleted}, []uuid.UUID{pending, declined}
	untouched := tableRows(t, pool, "workspace_member_invites", nil)
	before := tableRows(t, pool, "workspace_member_invites", written, "deleted_at", "updated_at", "updated_by_id")
	later := now.Add(time.Hour)

	rollback := errors.New("rolled back")
	if err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		if err := s.DeleteInvitations(ctx, given, grace, later); err != nil {
			return err
		}
		return rollback
	}); !errors.Is(err, rollback) {
		t.Fatalf("DeleteInvitations() in a transaction = %v; want it done, then the transaction rolled back", err)
	}
	if got := tableRows(t, pool, "workspace_member_invites", nil); got != untouched {
		t.Errorf("the invitations once the transaction rolled back:\n%s\nwant them as they were:\n%s", got, untouched)
	}
	if err := s.DeleteInvitations(context.Background(), given, grace, later); err != nil {
		t.Fatal(err)
	}

	for i, id := range written {
		if got, want := stamp(t, pool, "workspace_member_invites", id), fmt.Sprintf("deleted at %[1]s at %[1]s by %[2]s", at(t, pool, later),
			grace); got != want {
			t.Errorf("bob's invitation %d: %s, want %s", i, got, want)
		}
	}
	if after := tableRows(t, pool, "workspace_member_invites", written, "deleted_at", "updated_at", "updated_by_id"); after != before {
		t.Errorf("the invitations, the two deleted without the columns written:\n%s\nwant them as they were:\n%s", after, before)
	}
}
