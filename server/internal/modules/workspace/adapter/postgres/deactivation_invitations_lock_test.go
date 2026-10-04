package postgresadapter_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// invitationNames names every invitation by its workspace's slug, its
// address and its state: pending, declined, accepted or deleted.
func invitationNames(t *testing.T, pool *pgxpool.Pool) map[uuid.UUID]string {
	t.Helper()
	rows, err := pool.Query(pgtest.Soon(t), `SELECT i.id, w.slug || ' ' || i.email || ' ' || CASE WHEN i.accepted THEN 'accepted'
		WHEN i.deleted_at IS NOT NULL THEN 'deleted' WHEN i.responded_at IS NOT NULL THEN 'declined' ELSE 'pending' END
		FROM workspace_member_invites i JOIN workspaces w ON w.id = i.workspace_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	names := map[uuid.UUID]string{}
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			t.Fatal(err)
		}
		names[id] = name
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return names
}

// inOrder is the names of ids, in their order.
func inOrder(names map[uuid.UUID]string, ids []uuid.UUID) string {
	list := make([]string, len(ids))
	for i, id := range ids {
		list[i] = names[id]
	}
	return strings.Join(list, "; ")
}

// LockInvitationsToDelete returns the invitations a deactivation deletes,
// in id order, and holds exactly those FOR NO KEY UPDATE in the
// transaction it runs in, until that transaction ends: a FOR SHARE of each
// waits, a foreign key's FOR KEY SHARE does not; no other invitation is
// locked. The world is leftEmpty, and invitationsToBob in omega, sigma and
// tau, alice's workspaces, where each of the statement's predicates is
// decided by a row of its own: the deleted ones to his address, and
// frank's (the invitation's deleted_at); carol's and rebob's (the
// address); zeta's (the workspaces asked about); erin's, declined
// (pending); beta's, where carol is an active guest (another member, of
// that workspace, other than him); acme's, where he is alone (other than
// him); gamma's, where her membership ended (active); delta's, where it is
// deleted (undeleted).
func TestLockInvitationsToDeleteLocksWhatItReturns(t *testing.T) {
	s, pool := newStore(t)
	w := newLeftEmpty(t, s, pool)
	want := slices.Concat(w.found, invitationsToBob(t, s, pool, w.alice, [3]string{"omega", "sigma", "tau"}))
	slices.SortFunc(want, uuid.UUID.Compare)
	names := invitationNames(t, pool)
	var got []uuid.UUID
	end := hold(t, postgres.NewTxManager(pool, 2*time.Second), func(ctx context.Context) (err error) {
		got, err = s.LockInvitationsToDelete(ctx, w.asked, w.bob, "bob@corp.com")
		return err
	})

	if !slices.Equal(got, want) {
		t.Errorf("LockInvitationsToDelete() =\n%s\nwant, in id order,\n%s", inOrder(names, got), inOrder(names, want))
	}
	var locked, stronger []uuid.UUID
	for id := range names {
		if waitsFor(t, pool, "workspace_member_invites", id, "FOR SHARE") {
			locked = append(locked, id)
		}
		if waitsFor(t, pool, "workspace_member_invites", id, "FOR KEY SHARE") {
			stronger = append(stronger, id)
		}
	}
	slices.SortFunc(locked, uuid.UUID.Compare)
	if !slices.Equal(locked, want) || len(stronger) > 0 {
		t.Errorf("LockInvitationsToDelete() holds\n%s\nFOR NO KEY UPDATE or stronger, and these FOR UPDATE: %s;\nwant\n%s\nFOR NO KEY UPDATE",
			inOrder(names, locked), inOrder(names, stronger), inOrder(names, want))
	}
	if err := end(); err != nil {
		t.Fatal(err)
	}
}

// LockInvitationsToDelete takes its locks in the invitations' id order,
// whatever order the rows lie in, and returns them in that order: dave's
// pending one to acme, where bob is alone, has the smaller id, but lies
// after the one to bob's address in omega, alice's, in the table, as in
// the address's index. omega's row is held. LockInvitationsToDelete waits
// for it holding acme's, which a FOR SHARE then waits for; in the table's
// order, or the address's first, it would reach omega's first and wait
// holding nothing.
func TestLockInvitationsToDeleteLocksInIDOrder(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, omega := newWorkspace(t, s, "acme", "acme", bob).ID, newWorkspace(t, s, "omega", "omega", alice).ID
	toAcme, toBob := uuid.NewV7(), uuid.NewV7() // acme's drawn first: the smaller id
	for _, inv := range []struct {
		id, workspace uuid.UUID
		email         string
	}{{toBob, omega, "bob@corp.com"}, {toAcme, acme, "dave@corp.com"}} {
		exec(t, pool, `INSERT INTO workspace_member_invites (id, workspace_id, email, role, created_by_id, updated_by_id, created_at,
			updated_at) VALUES ($1, $2, $3, 15, $4, $4, $5, $5)`, inv.id, inv.workspace, inv.email, alice, now)
	}
	tx := postgres.NewTxManager(pool, 10*time.Second)
	release := hold(t, tx, func(ctx context.Context) error {
		_, err := postgres.DB(ctx, pool).Exec(ctx, "SELECT 1 FROM workspace_member_invites WHERE id = $1 FOR NO KEY UPDATE", toBob)
		return err
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var got []uuid.UUID
	done := make(chan error, 1)
	go func() {
		done <- tx.WithinTx(ctx, func(ctx context.Context) (err error) {
			got, err = s.LockInvitationsToDelete(ctx, []uuid.UUID{acme}, bob, "bob@corp.com")
			return err
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "workspace_member_invites", 10*time.Second)

	if !waitsFor(t, pool, "workspace_member_invites", toAcme, "FOR SHARE") {
		t.Error("a FOR SHARE of acme's invitation while LockInvitationsToDelete waits for omega's does not wait; want acme's locked first")
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if want := []uuid.UUID{toAcme, toBob}; err != nil || !slices.Equal(got, want) {
			t.Errorf("LockInvitationsToDelete() = %v, %v; want %v, acme's then omega's", got, err, want)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockInvitationsToDelete() did not end within 10s")
	}
}

// An invitation deleted while LockInvitationsToDelete waits for its row is
// left out, not an error: another transaction holds bob's invitation to
// omega FOR UPDATE, as deleteWorkspaceInvitation does, and deletes it;
// LockInvitationsToDelete, which found it undeleted, waits for it; once
// the deletion commits, it goes on and returns his other one, to sigma,
// alone, which DeleteInvitations then deletes; omega's stays as its
// deletion left it.
func TestLockInvitationsToDeleteLeavesOutAnInvitationDeletedWhileItWaited(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	omega, sigma := newWorkspace(t, s, "omega", "omega", alice).ID, newWorkspace(t, s, "sigma", "sigma", alice).ID
	toOmega, toSigma := invite(t, s, omega, "bob@corp.com", shared.RoleMember, alice).ID, invite(t, s, sigma, "bob@corp.com", shared.RoleMember, alice).ID
	tx := postgres.NewTxManager(pool, 10*time.Second)
	commit := hold(t, tx, func(ctx context.Context) error {
		if _, err := s.LockInvitation(ctx, toOmega); err != nil {
			return err
		}
		return s.DeleteInvitation(ctx, toOmega, alice, now)
	})
	later := now.Add(time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var got []uuid.UUID
	done := make(chan error, 1)
	go func() {
		done <- tx.WithinTx(ctx, func(ctx context.Context) (err error) {
			if got, err = s.LockInvitationsToDelete(ctx, nil, bob, "bob@corp.com"); err != nil {
				return err
			}
			return s.DeleteInvitations(ctx, got, bob, later)
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "workspace_member_invites", 10*time.Second)
	if err := commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if want := []uuid.UUID{toSigma}; err != nil || !slices.Equal(got, want) {
			t.Errorf("LockInvitationsToDelete(), then DeleteInvitations() = %v, %v; want %v, omega's left out", got, err, want)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockInvitationsToDelete() did not end within 10s")
	}
	for _, tt := range []struct {
		id   uuid.UUID
		want string
	}{
		{toOmega, "deleted at " + at(t, pool, now) + " at " + at(t, pool, now) + " by " + alice.String()},
		{toSigma, "deleted at " + at(t, pool, later) + " at " + at(t, pool, later) + " by " + bob.String()},
	} {
		if got := stamp(t, pool, "workspace_member_invites", tt.id); got != tt.want {
			t.Errorf("bob's invitation %s: %s, want %s", tt.id, got, tt.want)
		}
	}
}
