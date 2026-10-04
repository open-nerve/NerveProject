package postgresadapter_test

import (
	"context"
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// invitationsToBob stores, in alice's workspaces slugs, the invitations
// TestDeleteInvitationsTo deletes and keeps: bob's pending one to the
// first, his declined one to the second, his pending one to the third,
// whose member he is not, all three by alice; two earlier ones to his
// address in the first, one accepted, one deleted while pending; carol's
// pending one to the first, and one to the second to an address that only
// ends with his. It returns the three that DeleteInvitationsTo(bob's
// address) deletes.
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
	deleted := []uuid.UUID{
		invite(t, s, ws[0], "bob@corp.com", shared.RoleGuest, alice).ID,
		invite(t, s, ws[1], "bob@corp.com", shared.RoleMember, alice).ID,
		invite(t, s, ws[2], "bob@corp.com", shared.RoleAdmin, alice).ID,
	}
	exec(t, pool, "UPDATE workspace_member_invites SET responded_at = $2 WHERE id = $1", deleted[1], now)
	invite(t, s, ws[0], "carol@corp.com", shared.RoleMember, alice)
	invite(t, s, ws[1], "rebob@corp.com", shared.RoleMember, alice)
	return deleted
}

// DeleteInvitationsTo soft-deletes every undeleted invitation to the
// address, of every workspace, pending or declined, by the account and at
// the time given, and changes nothing else (M3 design 3.8, 3.9): bob's
// pending one to acme, his declined one to beta, his pending one to gamma
// (invitationsToBob). Two earlier ones to his address in acme, one
// accepted, one deleted while pending, keep their moments; carol's pending
// one to acme, and an address that only ends with his, every column.
func TestDeleteInvitationsTo(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	deleted := invitationsToBob(t, s, pool, alice, [3]string{"acme", "beta", "gamma"})
	before := tableRows(t, pool, "workspace_member_invites", deleted, "deleted_at", "updated_at", "updated_by_id")
	later := now.Add(time.Hour)

	if err := s.DeleteInvitationsTo(context.Background(), "bob@corp.com", bob, later); err != nil {
		t.Fatal(err)
	}

	for i, id := range deleted {
		if got, want := stamp(t, pool, "workspace_member_invites", id), fmt.Sprintf("deleted at %[1]s at %[1]s by %[2]s", at(t, pool, later),
			bob); got != want {
			t.Errorf("bob's invitation %d: %s, want %s", i, got, want)
		}
	}
	if after := tableRows(t, pool, "workspace_member_invites", deleted, "deleted_at", "updated_at", "updated_by_id"); after != before {
		t.Errorf("the invitations, bob's three without the columns written:\n%s\nwant them as they were:\n%s", after, before)
	}
}

// leftEmpty is the world of TestDeleteInvitationsOfWorkspacesLeftEmpty:
// asked are bob's workspaces acme, where he is alone; beta, where carol is
// an active guest, an active member all the same; gamma, where her
// membership ended; delta, where it is deleted. Each has a pending
// invitation of alice's, to dave: deleted are acme's, gamma's and delta's,
// though alice wrote them; beta's stays. acme's declined one, to erin, and
// its one deleted while pending, to frank, stay; so does the pending one of
// zeta, alice's, a workspace not asked about with no active member.
type leftEmpty struct {
	alice, bob     uuid.UUID
	asked, deleted []uuid.UUID
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
			w.deleted = append(w.deleted, id)
		}
	}
	declined := invite(t, s, ws["acme"], "erin@corp.com", shared.RoleGuest, w.alice).ID
	exec(t, pool, "UPDATE workspace_member_invites SET responded_at = $2 WHERE id = $1", declined, now)
	exec(t, pool, `INSERT INTO workspace_member_invites (id, workspace_id, email, role, created_by_id, updated_by_id, created_at,
		updated_at, deleted_at) VALUES ($1, $2, 'frank@corp.com', 15, $3, $3, $4, $4, $4)`, uuid.NewV7(), ws["acme"], w.alice, now.Add(-time.Hour))
	return w
}

// DeleteInvitationsOfWorkspacesLeftEmpty soft-deletes the pending
// invitations of each workspace asked about where bob has no other active
// member, by the account and at the time given, and changes nothing else
// (M3 design 3.7, 3.9; the P6 pre-flight's M1): in leftEmpty, acme's,
// gamma's and delta's. The account given is grace, not bob, so that the
// member and the writer cannot be taken for each other.
func TestDeleteInvitationsOfWorkspacesLeftEmpty(t *testing.T) {
	s, pool := newStore(t)
	w := newLeftEmpty(t, s, pool)
	grace := newAccount(t, pool, "grace@corp.com")
	before := tableRows(t, pool, "workspace_member_invites", w.deleted, "deleted_at", "updated_at", "updated_by_id")
	later := now.Add(time.Hour)

	if err := s.DeleteInvitationsOfWorkspacesLeftEmpty(context.Background(), w.asked, w.bob, grace, later); err != nil {
		t.Fatal(err)
	}

	for i, id := range w.deleted {
		if got, want := stamp(t, pool, "workspace_member_invites", id), fmt.Sprintf("deleted at %[1]s at %[1]s by %[2]s", at(t, pool, later),
			grace); got != want {
			t.Errorf("the invitation to dave of the workspace %d left empty: %s, want %s", i, got, want)
		}
	}
	if after := tableRows(t, pool, "workspace_member_invites", w.deleted, "deleted_at", "updated_at", "updated_by_id"); after != before {
		t.Errorf("the invitations, the three deleted without the columns written:\n%s\nwant them as they were:\n%s", after, before)
	}
}
