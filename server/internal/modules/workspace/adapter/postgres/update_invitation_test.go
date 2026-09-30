package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// invitationReads are the two reads of an invitation by its id, the lock
// in a transaction of its own.
func invitationReads(s *postgresadapter.Store, tx *postgres.TxManager) map[string]func(ctx context.Context, id uuid.UUID) (domain.Invitation, error) {
	return map[string]func(ctx context.Context, id uuid.UUID) (domain.Invitation, error){
		"InvitationByID": s.InvitationByID,
		"LockInvitation": func(ctx context.Context, id uuid.UUID) (got domain.Invitation, err error) {
			// The transaction begins on a live context and the lock runs in it
			// on one that is cancelled when ctx is: a cancelled ctx fails the
			// lock's own statement, not the BEGIN.
			err = tx.WithinTx(context.Background(), func(inTx context.Context) error {
				lockCtx, cancel := context.WithCancel(inTx)
				defer cancel()
				if ctx.Err() != nil {
					cancel()
				}
				got, err = s.LockInvitation(lockCtx, id)
				return err
			})
			return got, err
		},
	}
}

// InvitationByID and LockInvitation read an undeleted invitation, pending
// or declined, as stored; app.ErrNotFound for an accepted or deleted one
// and an id no row has. A failed read is its error, never app.ErrNotFound.
func TestTheInvitationReadsFindOnlyAnUndeletedInvitation(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	pending := invite(t, s, acme.ID, "carol@corp.com", shared.RoleMember, alice)
	declined := invite(t, s, acme.ID, "dave@corp.com", shared.RoleGuest, alice)
	exec(t, pool, "UPDATE workspace_member_invites SET responded_at = $1 WHERE id = $2", now, declined.ID)
	declined.RespondedAt = &now
	accepted := invite(t, s, acme.ID, "erin@corp.com", shared.RoleMember, alice)
	exec(t, pool, "UPDATE workspace_member_invites SET accepted = true, responded_at = $1, deleted_at = $1 WHERE id = $2", now, accepted.ID)
	deleted := invite(t, s, acme.ID, "frank@corp.com", shared.RoleMember, alice)
	exec(t, pool, "UPDATE workspace_member_invites SET deleted_at = $1 WHERE id = $2", now, deleted.ID)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	for name, read := range invitationReads(s, postgres.NewTxManager(pool, 2*time.Second)) {
		for _, want := range []domain.Invitation{pending, declined} {
			if got, err := read(context.Background(), want.ID); err != nil || !sameInvitation(got, want) {
				t.Errorf("%s(%s) = %+v, %v; want %+v", name, want.Email, got, err, want)
			}
		}
		for _, id := range []uuid.UUID{accepted.ID, deleted.ID, uuid.NewV7()} {
			if got, err := read(context.Background(), id); !errors.Is(err, app.ErrNotFound) || got.ID != (uuid.UUID{}) {
				t.Errorf("%s(%s) = %+v, %v; want app.ErrNotFound", name, id, got, err)
			}
		}
		if got, err := read(cancelled, pending.ID); !errors.Is(err, context.Canceled) || errors.Is(err, app.ErrNotFound) || got.ID != (uuid.UUID{}) {
			t.Errorf("%s() on a cancelled context = %+v, %v; want context.Canceled, not app.ErrNotFound", name, got, err)
		}
	}
}

// LockInvitation holds the row FOR UPDATE: another LockInvitation of it
// waits, and so does a FOR KEY SHARE, which a FOR NO KEY UPDATE would let
// pass; another invitation's lock does not wait. A wait ends with
// lock_not_available under a lock_timeout.
func TestLockInvitationLocksTheRowForUpdate(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	carol := invite(t, s, acme.ID, "carol@corp.com", shared.RoleMember, alice)
	dave := invite(t, s, acme.ID, "dave@corp.com", shared.RoleMember, alice)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	hold(t, tx, func(ctx context.Context) error {
		_, err := s.LockInvitation(ctx, carol.ID)
		return err
	})
	var pgErr *pgconn.PgError
	for name, take := range map[string]func(ctx context.Context) error{
		"LockInvitation": func(ctx context.Context) error {
			_, err := s.LockInvitation(ctx, carol.ID)
			return err
		},
		"FOR KEY SHARE": func(ctx context.Context) error {
			_, err := postgres.DB(ctx, pool).Exec(ctx, "SELECT id FROM workspace_member_invites WHERE id = $1 FOR KEY SHARE", carol.ID)
			return err
		},
	} {
		if err := withLockTimeout(tx, pool, take); !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
			t.Errorf("%s of carol's while it is locked: %v; want lock_not_available after waiting", name, err)
		}
	}
	var got domain.Invitation
	err := withLockTimeout(tx, pool, func(ctx context.Context) error {
		var err error
		got, err = s.LockInvitation(ctx, dave.ID)
		return err
	})
	if err != nil || got.ID != dave.ID {
		t.Errorf("LockInvitation of dave's = %+v, %v; want it without a wait", got, err)
	}
}

// A LockInvitation that waits for the transaction deleting the invitation
// reads no row once that one commits: app.ErrNotFound. WaitForLockWaitOn
// sees it waiting on the invitation's row before the deletion commits.
func TestLockInvitationSkipsAnInvitationDeletedWhileItWaits(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	carol := invite(t, s, acme.ID, "carol@corp.com", shared.RoleMember, alice)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	commit := hold(t, tx, func(ctx context.Context) error {
		if _, err := s.LockInvitation(ctx, carol.ID); err != nil {
			return err
		}
		return s.DeleteInvitation(ctx, carol.ID, alice, now)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	answered := make(chan error, 1)
	go func() {
		answered <- tx.WithinTx(ctx, func(ctx context.Context) error {
			_, err := s.LockInvitation(ctx, carol.ID)
			return err
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "workspace_member_invites", 5*time.Second)
	if err := commit(); err != nil {
		t.Fatalf("the deletion: %v", err)
	}
	select {
	case err := <-answered:
		if !errors.Is(err, app.ErrNotFound) {
			t.Errorf("LockInvitation() after the deletion = %v; want app.ErrNotFound", err)
		}
	case <-ctx.Done():
		t.Fatal("the lock did not end within 10s")
	}
}

// audit is an invitation's audit columns and deletion time.
type audit struct {
	updatedBy uuid.UUID
	updatedAt time.Time
	deletedAt *time.Time
}

func auditOf(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) audit {
	t.Helper()
	var a audit
	if err := pool.QueryRow(context.Background(), "SELECT updated_by_id, updated_at, deleted_at FROM workspace_member_invites WHERE id = $1", id).
		Scan(&a.updatedBy, &a.updatedAt, &a.deletedAt); err != nil {
		t.Fatal(err)
	}
	return a
}

// UpdateInvitationRole sets the role of that invitation only, with the
// updater and the time, and answers it as stored: another invitation of the
// workspace and one in another workspace keep their role and audit columns.
// A missing row is an error, not app.ErrNotFound; a failed write is its
// error.
func TestUpdateInvitationRole(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	carol := invite(t, s, acme.ID, "carol@corp.com", shared.RoleMember, alice)
	others := []domain.Invitation{invite(t, s, acme.ID, "dave@corp.com", shared.RoleMember, alice),
		invite(t, s, beta.ID, "carol@corp.com", shared.RoleMember, alice)}
	later := now.Add(time.Hour)

	got, err := s.UpdateInvitationRole(context.Background(), carol.ID, shared.RoleAdmin, bob, later)

	want := carol
	want.Role = shared.RoleAdmin
	if err != nil || !sameInvitation(got, want) {
		t.Errorf("UpdateInvitationRole() = %+v, %v; want %+v", got, err, want)
	}
	if a := auditOf(t, pool, carol.ID); a.updatedBy != bob || !a.updatedAt.Equal(later) || a.deletedAt != nil {
		t.Errorf("carol's: %+v; want updated by bob at %v, not deleted", a, later)
	}
	for _, o := range others {
		if got, err := s.InvitationByID(context.Background(), o.ID); err != nil || !sameInvitation(got, o) {
			t.Errorf("%s in %s: %+v, %v; want it unchanged", o.Email, o.WorkspaceID, got, err)
		}
		if a := auditOf(t, pool, o.ID); a.updatedBy != alice || !a.updatedAt.Equal(now) {
			t.Errorf("%s in %s: %+v; want updated by alice at %v", o.Email, o.WorkspaceID, a, now)
		}
	}
	if _, err := s.UpdateInvitationRole(context.Background(), uuid.NewV7(), shared.RoleGuest, bob, later); err == nil || errors.Is(err, app.ErrNotFound) {
		t.Errorf("UpdateInvitationRole() of no row = %v, want an error that is not app.ErrNotFound", err)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := s.UpdateInvitationRole(cancelled, carol.ID, shared.RoleGuest, bob, later); !errors.Is(err, context.Canceled) || got.ID != (uuid.UUID{}) {
		t.Errorf("UpdateInvitationRole() on a cancelled context = %+v, %v; want context.Canceled", got, err)
	}
}

// DeleteInvitation soft-deletes that invitation only, with the deleter and
// the time; its address is free again at once. Another invitation of the
// workspace and one in another workspace stay. A failed write is its
// error.
func TestDeleteInvitation(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	carol := invite(t, s, acme.ID, "carol@corp.com", shared.RoleMember, alice)
	invite(t, s, acme.ID, "dave@corp.com", shared.RoleMember, alice)
	invite(t, s, beta.ID, "carol@corp.com", shared.RoleMember, alice)
	later := now.Add(time.Hour)

	if err := s.DeleteInvitation(context.Background(), carol.ID, bob, later); err != nil {
		t.Fatalf("DeleteInvitation() = %v", err)
	}

	if a := auditOf(t, pool, carol.ID); a.updatedBy != bob || !a.updatedAt.Equal(later) || a.deletedAt == nil || !a.deletedAt.Equal(later) {
		t.Errorf("carol's: %+v; want deleted by bob at %v", a, later)
	}
	if got := pendingEmails(t, pool, acme.ID); !slices.Equal(got, []string{"dave@corp.com"}) {
		t.Errorf("acme's invitations: %q; want dave's", got)
	}
	if got := pendingEmails(t, pool, beta.ID); !slices.Equal(got, []string{"carol@corp.com"}) {
		t.Errorf("beta's invitations: %q; want carol's", got)
	}
	invite(t, s, acme.ID, "carol@corp.com", shared.RoleGuest, alice)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.DeleteInvitation(cancelled, carol.ID, bob, later); !errors.Is(err, context.Canceled) {
		t.Errorf("DeleteInvitation() on a cancelled context = %v; want context.Canceled", err)
	}
}
