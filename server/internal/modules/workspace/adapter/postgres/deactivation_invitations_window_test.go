package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// An invitation to bob's address created after his deactivation's
// invitations' lock, before its delete, is not his deactivation's to
// delete, and two deactivations do not wait for each other over it (ruling
// F-1, P6 spec section 3 item 20). bob is alone in acme, which invites
// carol; carol is alone in gamma. bob's deactivation, as the Deactivator
// runs it, locks acme and acme's invitation to carol, x, the only one it
// finds. Then z, an invitation to bob's address in gamma, is created and
// committed; its id is smaller than x's, drawn before x's was, as a
// request draws it before its transaction waits. carol's deactivation
// locks gamma and, in id order, z, then waits for x. bob's deletes x
// alone, the one its lock returned, and commits; had it deleted every
// invitation to his address, it would have waited for z, which carol's
// holds, while carol's waits for x: 40P01, one rolled back. carol's then
// leaves x out, deleted meanwhile, and deletes z: both are done, x deleted
// by bob, z by carol.
func TestAnInvitationCreatedAfterADeactivationsLockIsNotItsToDelete(t *testing.T) {
	s, pool := newStore(t)
	bob, carol := newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme, gamma := newWorkspace(t, s, "acme", "acme", bob).ID, newWorkspace(t, s, "gamma", "gamma", carol).ID
	z := uuid.NewV7()
	x := invite(t, s, acme, "carol@corp.com", shared.RoleMember, bob).ID
	if z.Compare(x) >= 0 {
		t.Fatalf("z %s, x %s: want z's id the smaller", z, x)
	}
	tx := postgres.NewTxManager(pool, 10*time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	bobsAt, carolsAt := now.Add(time.Hour), now.Add(2*time.Hour)
	// deactivation runs the Deactivator's invitation steps for account,
	// once it holds his workspaces: what its lock returned goes to locked,
	// then it waits for proceed, if any, before it deletes them.
	deactivation := func(account uuid.UUID, email string, at time.Time, locked chan<- []uuid.UUID, proceed <-chan struct{}) <-chan error {
		done := make(chan error, 1)
		go func() {
			done <- tx.WithinTx(ctx, func(ctx context.Context) error {
				workspaces, err := s.LockMemberWorkspaces(ctx, account)
				if err != nil {
					return err
				}
				ids, err := s.LockInvitationsToDelete(ctx, workspaces, account, email)
				if err != nil {
					return err
				}
				locked <- ids
				if proceed != nil {
					select {
					case <-proceed:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
				return s.DeleteInvitations(ctx, ids, account, at)
			})
		}()
		return done
	}
	bobsLock, carolsLock, proceed := make(chan []uuid.UUID, 1), make(chan []uuid.UUID, 1), make(chan struct{})
	bobs := deactivation(bob, "bob@corp.com", bobsAt, bobsLock, proceed)
	var bobsIDs []uuid.UUID
	select {
	case bobsIDs = <-bobsLock:
	case err := <-bobs:
		t.Fatalf("bob's deactivation before its lock was taken: %v", err)
	}
	exec(t, pool, `INSERT INTO workspace_member_invites (id, workspace_id, email, role, created_by_id, updated_by_id, created_at,
		updated_at) VALUES ($1, $2, 'bob@corp.com', 15, $3, $3, $4, $4)`, z, gamma, carol, now)
	carols := deactivation(carol, "carol@corp.com", carolsAt, carolsLock, nil)
	pgtest.WaitForLockWaitOn(t, pool, "workspace_member_invites", 10*time.Second)
	close(proceed)

	for name, done := range map[string]<-chan error{"bob's": bobs, "carol's": carols} {
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("%s deactivation = %v; want it done", name, err)
			}
		case <-ctx.Done():
			t.Fatalf("%s deactivation did not end within 10s", name)
		}
	}
	// Both have ended: carol's lock has sent its ids by now, or never will
	// (her deactivation failed in it).
	var carolsIDs []uuid.UUID
	select {
	case carolsIDs = <-carolsLock:
	default:
	}
	if !slices.Equal(bobsIDs, []uuid.UUID{x}) || !slices.Equal(carolsIDs, []uuid.UUID{z}) {
		t.Errorf("the locks returned %v to bob's deactivation, %v to carol's; want x %s, then z %s", bobsIDs, carolsIDs, x, z)
	}
	for _, tt := range []struct {
		name string
		id   uuid.UUID
		want string
	}{
		{"x", x, "deleted at " + at(t, pool, bobsAt) + " at " + at(t, pool, bobsAt) + " by " + bob.String()},
		{"z", z, "deleted at " + at(t, pool, carolsAt) + " at " + at(t, pool, carolsAt) + " by " + carol.String()},
	} {
		if got := stamp(t, pool, "workspace_member_invites", tt.id); got != tt.want {
			t.Errorf("%s: %s, want %s", tt.name, got, tt.want)
		}
	}
}
