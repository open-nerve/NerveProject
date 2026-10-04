package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// waitsFor reports whether the lock mode of the workspace id, taken in a
// transaction of its own, waits for another transaction's lock: it ends
// with lock_not_available under withLockTimeout's 300ms. A lock that does
// not wait must find the row.
func waitsFor(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, mode string) bool {
	t.Helper()
	var locked int64
	err := withLockTimeout(postgres.NewTxManager(pool, 2*time.Second), pool, func(ctx context.Context) error {
		tag, err := postgres.DB(ctx, pool).Exec(ctx, "SELECT id FROM workspaces WHERE id = $1 "+mode, id)
		locked = tag.RowsAffected()
		return err
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "55P03" {
		return true
	}
	if err != nil || locked != 1 {
		t.Fatalf("%s of %s: %d rows, %v; want its row", mode, id, locked, err)
	}
	return false
}

// LockMemberWorkspaces returns, in id order, the undeleted workspaces of
// which bob is an active member, acme and gamma, and holds each FOR NO KEY
// UPDATE until its transaction ends: a FOR SHARE of it waits, a foreign
// key's FOR KEY SHARE does not. It holds no other: not beta, where his
// membership ended; not delta, where it is deleted; not gone, deleted,
// though his membership of it is not; not epsilon, carol's alone; not
// zeta, where he is no member.
func TestLockMemberWorkspaces(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	ws := map[string]uuid.UUID{}
	for _, name := range []string{"acme", "beta", "gamma", "delta", "gone", "epsilon", "zeta"} {
		ws[name] = newWorkspace(t, s, name, name, alice).ID
	}
	for _, name := range []string{"acme", "beta", "gamma", "delta", "gone"} {
		join(t, s, ws[name], bob, shared.RoleMember)
	}
	join(t, s, ws["epsilon"], carol, shared.RoleAdmin)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2", ws["beta"], bob)
	exec(t, pool, "UPDATE workspace_members SET deleted_at = $3 WHERE workspace_id = $1 AND member_id = $2", ws["delta"], bob, now)
	exec(t, pool, "UPDATE workspaces SET deleted_at = $2 WHERE id = $1", ws["gone"], now)
	var locked []uuid.UUID
	end := hold(t, postgres.NewTxManager(pool, 2*time.Second), func(ctx context.Context) error {
		var err error
		locked, err = s.LockMemberWorkspaces(ctx, bob)
		return err
	})

	if want := slices.SortedFunc(slices.Values([]uuid.UUID{ws["acme"], ws["gamma"]}), uuid.UUID.Compare); !slices.Equal(locked, want) {
		t.Errorf("LockMemberWorkspaces() = %v, want acme and gamma, %v", locked, want)
	}
	for name, id := range ws {
		held := name == "acme" || name == "gamma"
		if share, keyShare := waitsFor(t, pool, id, "FOR SHARE"), waitsFor(t, pool, id, "FOR KEY SHARE"); share != held || keyShare {
			t.Errorf("%s while locked: a FOR SHARE waits %v, a FOR KEY SHARE waits %v; want %v, false", name, share, keyShare, held)
		}
	}
	if err := end(); err != nil {
		t.Fatal(err)
	}
}

// LockMemberWorkspaces takes its locks in the workspaces' id order,
// whatever order the rows lie in: web has the smaller id, but lies after
// alpha in the table and in the slug's index. alpha's row is held.
// LockMemberWorkspaces waits for it holding web's, which a FOR SHARE then
// waits for; in any other order it would reach alpha first and wait
// holding nothing.
func TestLockMemberWorkspacesLocksInIDOrder(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web, alpha := uuid.NewV7(), uuid.NewV7() // web drawn first: the smaller id
	for _, w := range []struct {
		id   uuid.UUID
		slug string
	}{{alpha, "alpha"}, {web, "web"}} {
		newWorkspaceWithID(t, s, w.id, w.slug, w.slug, alice)
		join(t, s, w.id, bob, shared.RoleMember)
	}
	tx := postgres.NewTxManager(pool, 10*time.Second)
	release := hold(t, tx, func(ctx context.Context) error {
		_, err := postgres.DB(ctx, pool).Exec(ctx, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", alpha)
		return err
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- tx.WithinTx(ctx, func(ctx context.Context) error {
			_, err := s.LockMemberWorkspaces(ctx, bob)
			return err
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "workspaces", 10*time.Second)

	if !waitsFor(t, pool, web, "FOR SHARE") {
		t.Error("a FOR SHARE of web while LockMemberWorkspaces waits for alpha does not wait; want web locked first")
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("LockMemberWorkspaces() = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockMemberWorkspaces() did not end within 10s")
	}
}

// A workspace deleted while LockMemberWorkspaces waits for its row is left
// out, not an error (M3 design 3.9, review spike 15): another transaction
// holds acme FOR NO KEY UPDATE, as deleteWorkspace does, and soft-deletes
// it, its memberships too; LockMemberWorkspaces, which found acme
// undeleted, waits for it; once the deletion commits, it evaluates
// deleted_at again on the row's newest version and returns beta alone.
func TestLockMemberWorkspacesLeavesOutAWorkspaceDeletedWhileItWaited(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice).ID, newWorkspace(t, s, "Beta", "beta", alice).ID
	for _, w := range []uuid.UUID{acme, beta} {
		join(t, s, w, bob, shared.RoleMember)
	}
	tx := postgres.NewTxManager(pool, 10*time.Second)
	commit := hold(t, tx, func(ctx context.Context) error {
		for _, sql := range []string{"SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", "UPDATE workspaces SET deleted_at = now() WHERE id = $1",
			"UPDATE workspace_members SET deleted_at = now() WHERE workspace_id = $1"} {
			if _, err := postgres.DB(ctx, pool).Exec(ctx, sql, acme); err != nil {
				return err
			}
		}
		return nil
	})
	type locked struct {
		ids []uuid.UUID
		err error
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	done := make(chan locked, 1)
	go func() {
		var l locked
		l.err = tx.WithinTx(ctx, func(ctx context.Context) error {
			var err error
			l.ids, err = s.LockMemberWorkspaces(ctx, bob)
			return err
		})
		done <- l
	}()
	pgtest.WaitForLockWaitOn(t, pool, "workspaces", 10*time.Second)
	if err := commit(); err != nil {
		t.Fatal(err)
	}
	select {
	case l := <-done:
		if l.err != nil || !slices.Equal(l.ids, []uuid.UUID{beta}) {
			t.Errorf("LockMemberWorkspaces() = %v, %v; want beta (%s) alone, acme (%s) deleted while it waited", l.ids, l.err, beta, acme)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LockMemberWorkspaces() did not end within 10s")
	}
}

// SoleAdmin reports whether bob is the only active admin of one of the
// workspaces asked about that has another active member (M3 design 3.7
// rule 2). Most cases ask about one workspace, in the state the case
// names; three ask about two at once, one of them with another admin or
// other members, so that a check of the set asked about, not of each of
// its workspaces, answers otherwise; one asks about none. Each workspace
// has the members its case names alone, its creator's membership deleted;
// the cases' workspaces all stand beside each other, so that a check of a
// workspace not asked about answers otherwise.
func TestSoleAdminOfAWorkspace(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	type member struct {
		user   uuid.UUID
		role   shared.Role
		active bool
	}
	// workspace stores a workspace with members, its creator's membership
	// deleted, the last of them deleted when deleteLast is set, and returns
	// its id.
	n := 0
	workspace := func(deleteLast bool, members ...member) uuid.UUID {
		t.Helper()
		n++
		w := newWorkspace(t, s, fmt.Sprintf("W%d", n), fmt.Sprintf("w%d", n), carol).ID
		exec(t, pool, "UPDATE workspace_members SET deleted_at = $2 WHERE workspace_id = $1", w, now)
		for i, m := range members {
			join(t, s, w, m.user, m.role)
			if !m.active {
				exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2 AND deleted_at IS NULL",
					w, m.user)
			}
			if deleteLast && i == len(members)-1 {
				exec(t, pool, "UPDATE workspace_members SET deleted_at = $3 WHERE workspace_id = $1 AND member_id = $2", w, m.user, now)
			}
		}
		return w
	}
	alone := workspace(false, member{bob, shared.RoleAdmin, true})
	withAMember := workspace(false, member{bob, shared.RoleAdmin, true}, member{carol, shared.RoleMember, true})
	twoAdmins := workspace(false, member{bob, shared.RoleAdmin, true}, member{alice, shared.RoleAdmin, true}, member{carol, shared.RoleMember, true})
	workspace(false, member{alice, shared.RoleAdmin, true}, member{carol, shared.RoleMember, true}) // not asked about
	tests := []struct {
		name       string
		workspaces []uuid.UUID
		want       bool
	}{
		{"the only admin, with an active member", []uuid.UUID{withAMember}, true},
		{"the only admin, with an active guest", []uuid.UUID{workspace(false, member{bob, shared.RoleAdmin, true},
			member{carol, shared.RoleGuest, true})}, true},
		{"the only admin, alone", []uuid.UUID{alone}, false},
		{"the only admin, the other's membership ended", []uuid.UUID{workspace(false, member{bob, shared.RoleAdmin, true},
			member{carol, shared.RoleMember, false})}, false},
		{"the only admin, the other's membership deleted", []uuid.UUID{workspace(true, member{bob, shared.RoleAdmin, true},
			member{carol, shared.RoleMember, true})}, false},
		{"one of two active admins", []uuid.UUID{twoAdmins}, false},
		{"the other admin's membership ended", []uuid.UUID{workspace(false, member{bob, shared.RoleAdmin, true},
			member{alice, shared.RoleAdmin, false}, member{carol, shared.RoleMember, true})}, true},
		{"the other admin's membership deleted", []uuid.UUID{workspace(true, member{bob, shared.RoleAdmin, true},
			member{carol, shared.RoleMember, true}, member{alice, shared.RoleAdmin, true})}, true},
		{"a member, beside the only admin and another member", []uuid.UUID{workspace(false, member{bob, shared.RoleMember, true},
			member{alice, shared.RoleAdmin, true}, member{carol, shared.RoleMember, true})}, false},
		{"a member, beside another member and no admin", []uuid.UUID{workspace(false, member{bob, shared.RoleMember, true},
			member{carol, shared.RoleMember, true})}, false},
		{"an admin whose membership ended", []uuid.UUID{workspace(false, member{bob, shared.RoleAdmin, false},
			member{carol, shared.RoleMember, true})}, false},
		{"an admin whose membership is deleted", []uuid.UUID{workspace(true, member{carol, shared.RoleMember, true},
			member{bob, shared.RoleAdmin, true})}, false},
		{"the only admin of one of two asked about", []uuid.UUID{alone, withAMember}, true},
		{"the only admin of one, another asked about having another admin", []uuid.UUID{withAMember, twoAdmins}, true},
		{"alone in one, another asked about having other members", []uuid.UUID{alone, twoAdmins}, false},
		{"none asked about", nil, false},
	}
	for _, tt := range tests {
		if got, err := s.SoleAdmin(context.Background(), tt.workspaces, bob); err != nil || got != tt.want {
			t.Errorf("%s: SoleAdmin() = %v, %v; want %v", tt.name, got, err, tt.want)
		}
	}
}

// EndWorkspaceMemberships ends bob's active, undeleted memberships of the
// workspaces asked about, acme's as a member and gamma's as an admin, each
// keeping its role, by the account and at the time given, though alice
// wrote them last, and changes
// nothing else (M3 design 3.6 convention 6): his membership of beta,
// asked about but ended by alice before, keeps its ender and moment; his
// deleted membership of acme, carol's of acme and his of delta, not asked
// about, every column.
func TestEndWorkspaceMemberships(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice).ID, newWorkspace(t, s, "Beta", "beta", alice).ID
	gamma, delta := newWorkspace(t, s, "Gamma", "gamma", alice).ID, newWorkspace(t, s, "Delta", "delta", alice).ID
	exec(t, pool, `INSERT INTO workspace_members (id, workspace_id, member_id, role, created_by_id, updated_by_id, created_at, updated_at,
		deleted_at) VALUES ($1, $2, $3, 20, $3, $3, $4, $4, $4)`, uuid.NewV7(), acme, bob, now.Add(-time.Hour))
	ended := []uuid.UUID{joinAt(t, s, acme, bob, shared.RoleMember, now).ID, joinAt(t, s, gamma, bob, shared.RoleAdmin, now).ID}
	exec(t, pool, "UPDATE workspace_members SET updated_by_id = $2 WHERE id = ANY($1)", ended, alice)
	joinAt(t, s, beta, bob, shared.RoleGuest, now)
	exec(t, pool, "UPDATE workspace_members SET is_active = false, updated_by_id = $3 WHERE workspace_id = $1 AND member_id = $2", beta, bob, alice)
	joinAt(t, s, acme, carol, shared.RoleMember, now)
	joinAt(t, s, delta, bob, shared.RoleMember, now)
	before := tableRows(t, pool, "workspace_members", ended, "is_active", "updated_at", "updated_by_id")
	later := now.Add(time.Hour)

	if err := s.EndWorkspaceMemberships(context.Background(), []uuid.UUID{acme, beta, gamma}, bob, bob, later); err != nil {
		t.Fatal(err)
	}

	for i, id := range ended {
		if got, want := stamp(t, pool, "workspace_members", id), fmt.Sprintf("ended at %s by %s", at(t, pool, later), bob); got != want {
			t.Errorf("bob's membership %d: %s, want %s", i, got, want)
		}
	}
	if after := tableRows(t, pool, "workspace_members", ended, "is_active", "updated_at", "updated_by_id"); after != before {
		t.Errorf("the memberships, bob's two ended without the columns written:\n%s\nwant them as they were:\n%s", after, before)
	}
}
