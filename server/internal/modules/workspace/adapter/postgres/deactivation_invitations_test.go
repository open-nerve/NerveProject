package postgresadapter_test

import (
	"context"
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteInvitationsTo soft-deletes every undeleted invitation to the
// address, of every workspace, pending or declined, by the account and at
// the time given, and changes nothing else (M3 design 3.8, 3.9): bob's
// pending one to acme, his declined one to beta, his pending one to gamma,
// whose member he is not. Two earlier ones to his address in acme, one
// accepted, one deleted while pending, keep their moments; carol's pending
// one to acme, and an address that only ends with his, every column.
func TestDeleteInvitationsTo(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta, gamma := newWorkspace(t, s, "Acme", "acme", alice).ID, newWorkspace(t, s, "Beta", "beta", alice).ID,
		newWorkspace(t, s, "Gamma", "gamma", alice).ID
	exec(t, pool, `INSERT INTO workspace_member_invites (id, workspace_id, email, role, accepted, responded_at, created_by_id,
		updated_by_id, created_at, updated_at, deleted_at) VALUES ($1, $2, 'bob@corp.com', 15, true, $4, $3, $3, $4, $4, $4)`,
		uuid.NewV7(), acme, alice, now.Add(-time.Hour))
	exec(t, pool, `INSERT INTO workspace_member_invites (id, workspace_id, email, role, created_by_id, updated_by_id, created_at,
		updated_at, deleted_at) VALUES ($1, $2, 'bob@corp.com', 5, $3, $3, $4, $4, $4)`, uuid.NewV7(), acme, alice, now.Add(-time.Hour))
	deleted := []uuid.UUID{
		invite(t, s, acme, "bob@corp.com", shared.RoleGuest, alice).ID,
		invite(t, s, beta, "bob@corp.com", shared.RoleMember, alice).ID,
		invite(t, s, gamma, "bob@corp.com", shared.RoleAdmin, alice).ID,
	}
	exec(t, pool, "UPDATE workspace_member_invites SET responded_at = $2 WHERE id = $1", deleted[1], now)
	invite(t, s, acme, "carol@corp.com", shared.RoleMember, alice)
	invite(t, s, beta, "rebob@corp.com", shared.RoleMember, alice)
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

// DeleteInvitationsOfWorkspacesLeftEmpty soft-deletes the pending
// invitations of each workspace asked about where bob has no other active
// member, by the account and at the time given, and changes nothing else
// (M3 design 3.7, 3.9; the P6 pre-flight's M1). Asked about at once: acme,
// where he is alone; beta, where carol is an active member; gamma, where
// her membership ended; delta, where it is deleted. Each has a pending
// invitation of alice's, to dave: acme's, gamma's and delta's are deleted,
// though alice wrote them; beta's stays. acme's declined one, to erin, and
// its one deleted while pending, to frank, keep every column; so does
// zeta's pending one, a workspace not asked about with no active member.
func TestDeleteInvitationsOfWorkspacesLeftEmpty(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	ws := map[string]uuid.UUID{"zeta": newWorkspace(t, s, "zeta", "zeta", alice).ID}
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1", ws["zeta"])
	for _, name := range []string{"acme", "beta", "gamma", "delta"} {
		ws[name] = newWorkspace(t, s, name, name, bob).ID
	}
	for _, name := range []string{"beta", "gamma", "delta"} {
		join(t, s, ws[name], carol, shared.RoleMember)
	}
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2", ws["gamma"], carol)
	exec(t, pool, "UPDATE workspace_members SET deleted_at = $3 WHERE workspace_id = $1 AND member_id = $2", ws["delta"], carol, now)
	var deleted []uuid.UUID
	for _, name := range []string{"acme", "beta", "gamma", "delta", "zeta"} {
		id := invite(t, s, ws[name], "dave@corp.com", shared.RoleMember, alice).ID
		if name == "acme" || name == "gamma" || name == "delta" {
			deleted = append(deleted, id)
		}
	}
	declined := invite(t, s, ws["acme"], "erin@corp.com", shared.RoleGuest, alice).ID
	exec(t, pool, "UPDATE workspace_member_invites SET responded_at = $2 WHERE id = $1", declined, now)
	exec(t, pool, `INSERT INTO workspace_member_invites (id, workspace_id, email, role, created_by_id, updated_by_id, created_at,
		updated_at, deleted_at) VALUES ($1, $2, 'frank@corp.com', 15, $3, $3, $4, $4, $4)`, uuid.NewV7(), ws["acme"], alice, now.Add(-time.Hour))
	before := tableRows(t, pool, "workspace_member_invites", deleted, "deleted_at", "updated_at", "updated_by_id")
	later := now.Add(time.Hour)

	if err := s.DeleteInvitationsOfWorkspacesLeftEmpty(context.Background(), []uuid.UUID{ws["acme"], ws["beta"], ws["gamma"], ws["delta"]},
		bob, bob, later); err != nil {
		t.Fatal(err)
	}

	for i, id := range deleted {
		if got, want := stamp(t, pool, "workspace_member_invites", id), fmt.Sprintf("deleted at %[1]s at %[1]s by %[2]s", at(t, pool, later),
			bob); got != want {
			t.Errorf("the invitation to dave of the workspace %d left empty: %s, want %s", i, got, want)
		}
	}
	if after := tableRows(t, pool, "workspace_member_invites", deleted, "deleted_at", "updated_at", "updated_by_id"); after != before {
		t.Errorf("the invitations, the three deleted without the columns written:\n%s\nwant them as they were:\n%s", after, before)
	}
}
