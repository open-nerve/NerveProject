package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// MemberOf finds the user's undeleted membership of that workspace, active
// or ended; none for a deleted row, another workspace's row, or no row. A
// failed read is its error, never "none".
func TestMemberOf(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	bobIn := joinAt(t, s, acme.ID, bob, shared.RoleMember, now)
	carolIn := joinAt(t, s, acme.ID, carol, shared.RoleGuest, now)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE id = $1", carolIn.ID)
	carolIn.IsActive = false
	gone := joinAt(t, s, beta.ID, carol, shared.RoleMember, now)
	exec(t, pool, "UPDATE workspace_members SET deleted_at = $2 WHERE id = $1", gone.ID, now)

	for _, want := range []domain.Membership{bobIn, carolIn} {
		if got, found, err := s.MemberOf(context.Background(), acme.ID, want.MemberID); err != nil || !found || got != want {
			t.Errorf("MemberOf(acme, %s) = %+v, %v, %v; want %+v", want.MemberID, got, found, err, want)
		}
	}
	for _, tt := range []struct {
		name            string
		workspace, user uuid.UUID
	}{{"a deleted row", beta.ID, carol}, {"another workspace's", beta.ID, bob}, {"no row", acme.ID, uuid.NewV7()}} {
		if got, found, err := s.MemberOf(context.Background(), tt.workspace, tt.user); err != nil || found || got != (domain.Membership{}) {
			t.Errorf("%s: MemberOf() = %+v, %v, %v; want none", tt.name, got, found, err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, found, err := s.MemberOf(cancelled, acme.ID, bob); !errors.Is(err, context.Canceled) || found || got != (domain.Membership{}) {
		t.Errorf("MemberOf() on a cancelled context = %+v, %v, %v; want context.Canceled", got, found, err)
	}
}

// memberAudit is a membership's state and audit columns.
type memberAudit struct {
	role      shared.Role
	active    bool
	updatedBy uuid.UUID
	updatedAt time.Time
}

func memberAuditOf(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) memberAudit {
	t.Helper()
	var a memberAudit
	if err := pool.QueryRow(context.Background(), "SELECT role, is_active, updated_by_id, updated_at FROM workspace_members WHERE id = $1", id).
		Scan(&a.role, &a.active, &a.updatedBy, &a.updatedAt); err != nil {
		t.Fatal(err)
	}
	return a
}

// RestoreMember makes that membership active again with the role, the
// restorer and the time; another ended membership of the workspace and the
// member's ended one in another workspace stay ended. A failed write is its
// error.
func TestRestoreMember(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	bobIn, carolIn, bobInBeta := joinAt(t, s, acme.ID, bob, shared.RoleAdmin, now), joinAt(t, s, acme.ID, carol, shared.RoleMember, now),
		joinAt(t, s, beta.ID, bob, shared.RoleMember, now)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE id = ANY($1)", []uuid.UUID{bobIn.ID, carolIn.ID, bobInBeta.ID})
	later := now.Add(time.Hour)

	if err := s.RestoreMember(context.Background(), bobIn.ID, shared.RoleGuest, bob, later); err != nil {
		t.Fatalf("RestoreMember() = %v", err)
	}

	if got, want := memberAuditOf(t, pool, bobIn.ID), (memberAudit{shared.RoleGuest, true, bob, later}); got.role != want.role ||
		got.active != want.active || got.updatedBy != want.updatedBy || !got.updatedAt.Equal(want.updatedAt) {
		t.Errorf("bob's in acme: %+v, want %+v", got, want)
	}
	for _, m := range []domain.Membership{carolIn, bobInBeta} {
		if got := memberAuditOf(t, pool, m.ID); got.active || got.role != m.Role || !got.updatedAt.Equal(now) {
			t.Errorf("%s in %s: %+v; want it ended, unchanged", m.MemberID, m.WorkspaceID, got)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.RestoreMember(cancelled, carolIn.ID, shared.RoleGuest, bob, later); !errors.Is(err, context.Canceled) {
		t.Errorf("RestoreMember() on a cancelled context = %v; want context.Canceled", err)
	}
}

// AcceptInvitation records that invitation accepted and deletes it, at the
// same moment, with the accepter; its address is free again. DeclineInvitation
// records it declined, undeleted: it holds its address. Another invitation
// of the workspace and one to the same address in another workspace stay
// pending. A failed write is its error.
func TestAnsweringAnInvitation(t *testing.T) {
	for _, tt := range []struct {
		name     string
		accepted bool
	}{{"AcceptInvitation", true}, {"DeclineInvitation", false}} {
		t.Run(tt.name, func(t *testing.T) {
			s, pool := newStore(t)
			answer := s.DeclineInvitation
			if tt.accepted {
				answer = s.AcceptInvitation
			}
			alice, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "carol@corp.com")
			acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
			answered := invite(t, s, acme.ID, "carol@corp.com", shared.RoleMember, alice)
			invite(t, s, acme.ID, "dave@corp.com", shared.RoleMember, alice)
			invite(t, s, beta.ID, "carol@corp.com", shared.RoleMember, alice)
			later := now.Add(time.Hour)

			if err := answer(context.Background(), answered.ID, carol, later); err != nil {
				t.Fatalf("%s() = %v", tt.name, err)
			}

			var accepted bool
			var responded, deleted *time.Time
			var updatedBy uuid.UUID
			var updatedAt time.Time
			if err := pool.QueryRow(context.Background(), `SELECT accepted, responded_at, deleted_at, updated_by_id, updated_at
				FROM workspace_member_invites WHERE id = $1`, answered.ID).Scan(&accepted, &responded, &deleted, &updatedBy, &updatedAt); err != nil {
				t.Fatal(err)
			}
			wantDeleted := tt.accepted
			if accepted != tt.accepted || responded == nil || !responded.Equal(later) || (deleted != nil) != wantDeleted ||
				(deleted != nil && !deleted.Equal(later)) || updatedBy != carol || !updatedAt.Equal(later) {
				t.Errorf("the answered invitation: accepted %v, responded %v, deleted %v, by %s at %v; want accepted %v at %v, deleted %v",
					accepted, responded, deleted, updatedBy, updatedAt, tt.accepted, later, wantDeleted)
			}
			wantAcme := []string{"dave@corp.com"}
			if !tt.accepted {
				wantAcme = []string{"carol@corp.com", "dave@corp.com"}
			}
			if got := pendingEmails(t, pool, acme.ID); !slices.Equal(got, wantAcme) {
				t.Errorf("acme's undeleted invitations: %q, want %q", got, wantAcme)
			}
			var pending int
			if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM workspace_member_invites
				WHERE id <> $1 AND responded_at IS NULL AND NOT accepted AND deleted_at IS NULL AND updated_at = created_at`, answered.ID).
				Scan(&pending); err != nil || pending != 2 {
				t.Errorf("the other invitations still pending, unchanged: %d, %v; want 2", pending, err)
			}
			_, err := s.CreateInvitations(context.Background(), []app.InvitationRow{
				{ID: uuid.NewV7(), WorkspaceID: acme.ID, Email: "carol@corp.com", Role: shared.RoleGuest, CreatedBy: alice, Now: later},
			})
			var dup *app.DuplicateInvitation
			if tt.accepted == errors.As(err, &dup) {
				t.Errorf("inviting carol@corp.com again = %v; want it free again only once accepted", err)
			}
			cancelled, cancel := context.WithCancel(context.Background())
			cancel()
			if err := answer(cancelled, answered.ID, carol, later); !errors.Is(err, context.Canceled) {
				t.Errorf("%s() on a cancelled context = %v; want context.Canceled", tt.name, err)
			}
		})
	}
}

// WorkspaceByID reads the undeleted workspace, counting its active
// memberships only, as the transaction that reads it sees them; a deleted
// workspace, which its caller's lock rules out, and a failed read are
// errors, never app.ErrNotFound.
func TestWorkspaceByID(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", alice)
	joinAt(t, s, acme.ID, bob, shared.RoleMember, now)
	ended := joinAt(t, s, acme.ID, carol, shared.RoleMember, now)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE id = $1", ended.ID)
	exec(t, pool, "UPDATE workspaces SET deleted_at = $2 WHERE id = $1", beta.ID, now)

	got, err := s.WorkspaceByID(context.Background(), acme.ID)
	want := acme
	want.Role, want.TotalMembers = 0, 2
	if err != nil || !sameWorkspace(got, want) || got.TotalMembers != 2 {
		t.Errorf("WorkspaceByID(acme) = %+v, %v; want %+v with 2 members", got, err, want)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for name, tt := range map[string]struct {
		ctx context.Context
		id  uuid.UUID
	}{"deleted": {context.Background(), beta.ID}, "cancelled": {cancelled, acme.ID}} {
		if got, err := s.WorkspaceByID(tt.ctx, tt.id); err == nil || errors.Is(err, app.ErrNotFound) || got.ID != (uuid.UUID{}) {
			t.Errorf("WorkspaceByID(), %s = %+v, %v; want an error that is not app.ErrNotFound", name, got, err)
		}
	}
}
