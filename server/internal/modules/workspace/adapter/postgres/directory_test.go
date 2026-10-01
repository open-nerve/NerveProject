package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Both of the directory's reads find the undeleted workspace a slug names,
// with its id and its time zone: not a deleted one, not by another case of
// the slug, and each workspace its own.
func TestWorkspaceDirectoryFindsTheUndeletedWorkspace(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	beta := newWorkspace(t, s, "Beta", "beta", alice)
	gone := newWorkspace(t, s, "Gone", "gone", alice)
	exec(t, pool, "UPDATE workspaces SET timezone = 'Asia/Shanghai' WHERE id = $1", beta.ID)
	exec(t, pool, "UPDATE workspaces SET deleted_at = $2 WHERE id = $1", gone.ID, now)
	finds := map[string]func(context.Context, string) (app.DirectoryEntry, bool, error){
		"WorkspaceBySlug": d.WorkspaceBySlug, "ShareWorkspaceBySlug": d.ShareWorkspaceBySlug,
	}
	for name, find := range finds {
		tx := postgres.NewTxManager(pool, 2*time.Second)
		err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
			for slug, want := range map[string]app.DirectoryEntry{"acme": {ID: acme.ID, Timezone: "UTC"}, "beta": {ID: beta.ID, Timezone: "Asia/Shanghai"}} {
				if got, found, err := find(ctx, slug); err != nil || !found || got != want {
					t.Errorf("%s(%s) = %+v, %v, %v; want %+v", name, slug, got, found, err, want)
				}
			}
			for _, slug := range []string{"gone", "ACME", "acm", "nothing"} {
				if got, found, err := find(ctx, slug); err != nil || found || got != (app.DirectoryEntry{}) {
					t.Errorf("%s(%q) = %+v, %v, %v; want not found", name, slug, got, found, err)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// The directory's lock is convention 2's FOR SHARE: it waits for the
// workspace's FOR NO KEY UPDATE and makes it wait, not another FOR SHARE,
// and no lock of another workspace. A lock that waits ends with
// lock_not_available under a lock_timeout.
func TestTheDirectorysLockIsForShare(t *testing.T) {
	const directory = "Directory.ShareWorkspaceBySlug"
	tests := []struct {
		held, then string
		slug       string // then's
		waits      bool
	}{
		{directory, noKeyUpdate.name, "acme", true},
		{noKeyUpdate.name, directory, "acme", true},
		{directory, forShare.name, "acme", false},
		{forShare.name, directory, "acme", false},
		{directory, noKeyUpdate.name, "beta", false},
		{noKeyUpdate.name, directory, "beta", false},
	}
	for _, tt := range tests {
		t.Run(tt.held+" held, "+tt.then+" of "+tt.slug, func(t *testing.T) {
			s, pool := newStore(t)
			d := postgresadapter.NewDirectory(pool)
			locks := map[string]lock{noKeyUpdate.name: noKeyUpdate, forShare.name: forShare, directory: {directory,
				func(ctx context.Context, _ *postgresadapter.Store, w named) (uuid.UUID, error) {
					e, found, err := d.ShareWorkspaceBySlug(ctx, w.slug)
					if err == nil && !found {
						err = app.ErrNotFound
					}
					return e.ID, err
				}}}
			alice := newAccount(t, pool, "alice@corp.com")
			ids := map[string]uuid.UUID{"acme": newWorkspace(t, s, "Acme", "acme", alice).ID, "beta": newWorkspace(t, s, "Beta", "beta", alice).ID}
			tx := postgres.NewTxManager(pool, 2*time.Second)
			hold(t, tx, func(ctx context.Context) error {
				_, err := locks[tt.held].take(ctx, s, named{"acme", ids["acme"]})
				return err
			})

			var got uuid.UUID
			err := withLockTimeout(tx, pool, func(ctx context.Context) error {
				var err error
				got, err = locks[tt.then].take(ctx, s, named{tt.slug, ids[tt.slug]})
				return err
			})

			var pgErr *pgconn.PgError
			switch {
			case tt.waits && (!errors.As(err, &pgErr) || pgErr.Code != "55P03"):
				t.Errorf("%s() = %s, %v; want lock_not_available after waiting", tt.then, got, err)
			case !tt.waits && (err != nil || got != ids[tt.slug]):
				t.Errorf("%s() = %s, %v; want %s's id %s without waiting", tt.then, got, err, tt.slug, ids[tt.slug])
			}
		})
	}
}

// A workspace deleted while the directory's lock waits is not found: the
// lock's statement has deleted_at IS NULL, which Postgres evaluates again
// on the row's newest version after the wait.
func TestTheDirectorysLockSeesADeletionItWaitedFor(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	end := hold(t, tx, func(ctx context.Context) error {
		if _, err := s.LockWorkspaceBySlug(ctx, "acme"); err != nil {
			return err
		}
		return s.DeleteWorkspace(ctx, acme.ID, alice, now)
	})
	type answer struct {
		w     app.DirectoryEntry
		found bool
		err   error
	}
	done := make(chan answer, 1)
	go func() {
		var a answer
		a.err = tx.WithinTx(context.Background(), func(ctx context.Context) error {
			var err error
			a.w, a.found, err = d.ShareWorkspaceBySlug(ctx, "acme")
			return err
		})
		done <- a
	}()
	pgtest.WaitForLockWaitOn(t, pool, "workspaces", 10*time.Second)
	if err := end(); err != nil {
		t.Fatal(err)
	}
	select {
	case a := <-done:
		if a.err != nil || a.found {
			t.Errorf("ShareWorkspaceBySlug() after the deletion = %+v, %v, %v; want not found", a.w, a.found, a.err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("ShareWorkspaceBySlug() did not end within 10s")
	}
}

// ShareMembers answers the active members' roles among the accounts asked
// for, in the workspace asked for: not an ended membership, not a deleted
// one, not another workspace's, not an account not asked for.
func TestShareMembersAnswersTheActiveMembersRoles(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	var ids []uuid.UUID
	for _, email := range []string{"alice@corp.com", "bob@corp.com", "carol@corp.com", "dave@corp.com", "erin@corp.com", "frank@corp.com", "gina@corp.com"} {
		ids = append(ids, newAccount(t, pool, email))
	}
	alice, bob, carol, dave, erin, frank, gina := ids[0], ids[1], ids[2], ids[3], ids[4], ids[5], ids[6]
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", frank)
	join(t, s, acme.ID, bob, shared.RoleMember)
	join(t, s, acme.ID, carol, shared.RoleGuest)
	join(t, s, acme.ID, dave, shared.RoleMember)
	join(t, s, acme.ID, erin, shared.RoleMember)
	join(t, s, acme.ID, gina, shared.RoleMember)
	join(t, s, beta.ID, bob, shared.RoleAdmin)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2", acme.ID, dave)
	exec(t, pool, "UPDATE workspace_members SET deleted_at = $3 WHERE workspace_id = $1 AND member_id = $2", acme.ID, erin, now)

	var got map[uuid.UUID]shared.Role
	err := postgres.NewTxManager(pool, 2*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
		var err error
		got, err = d.ShareMembers(ctx, acme.ID, []uuid.UUID{alice, bob, carol, dave, erin, frank, uuid.NewV7()})
		return err
	})

	want := map[uuid.UUID]shared.Role{alice: shared.RoleAdmin, bob: shared.RoleMember, carol: shared.RoleGuest}
	if err != nil || !maps.Equal(got, want) {
		t.Errorf("ShareMembers() = %v, %v; want %v", got, err, want)
	}
}

// ShareMembers locks FOR SHARE the undeleted memberships asked for in the
// workspace, the ended one too, and no other row: an update of a locked row
// waits; one of another account's, of the same account in another
// workspace, or of a deleted row does not.
func TestShareMembersLocksTheRowsAskedFor(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	alice, bob, carol := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "carol@corp.com")
	dave, erin := newAccount(t, pool, "dave@corp.com"), newAccount(t, pool, "erin@corp.com")
	acme, beta := newWorkspace(t, s, "Acme", "acme", alice), newWorkspace(t, s, "Beta", "beta", bob)
	join(t, s, acme.ID, bob, shared.RoleMember)
	join(t, s, acme.ID, carol, shared.RoleMember)
	join(t, s, acme.ID, dave, shared.RoleMember)
	join(t, s, acme.ID, erin, shared.RoleMember)
	exec(t, pool, "UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2", acme.ID, carol)
	exec(t, pool, "UPDATE workspace_members SET deleted_at = $3 WHERE workspace_id = $1 AND member_id = $2", acme.ID, erin, now)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	hold(t, tx, func(ctx context.Context) error {
		_, err := d.ShareMembers(ctx, acme.ID, []uuid.UUID{bob, carol, erin})
		return err
	})
	for _, tt := range []struct {
		name            string
		workspace, user uuid.UUID
		waits           bool
	}{
		{"bob's in acme, asked for", acme.ID, bob, true},
		{"carol's ended one, asked for", acme.ID, carol, true},
		{"dave's, not asked for", acme.ID, dave, false},
		{"erin's deleted one", acme.ID, erin, false},
		{"bob's in beta", beta.ID, bob, false},
	} {
		err := withLockTimeout(tx, pool, func(ctx context.Context) error {
			_, err := postgres.DB(ctx, pool).Exec(ctx, "UPDATE workspace_members SET role = role WHERE workspace_id = $1 AND member_id = $2",
				tt.workspace, tt.user)
			return err
		})
		var pgErr *pgconn.PgError
		if waited := errors.As(err, &pgErr) && pgErr.Code == "55P03"; waited != tt.waits || (!waited && err != nil) {
			t.Errorf("%s: updating it = %v; want a wait %v", tt.name, err, tt.waits)
		}
	}
}

// ShareMembers takes its locks in the memberships' id order, whatever
// order the rows lie in, in the table or in an index: bob's membership has
// the smaller id, but lies after carol's in the table, and bob's account
// has the greater id, so the (workspace_id, member_id) index lists carol's
// first too. carol's row is held. ShareMembers waits for carol's holding
// bob's, which an update of bob's row then waits for; in any other order it
// would reach carol's first and wait holding nothing.
func TestShareMembersLocksInIDOrder(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	alice, carol, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "carol@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	bobsID := uuid.NewV7() // drawn first: the smaller id
	exec(t, pool, "INSERT INTO workspace_members (id, workspace_id, member_id, role) VALUES ($1, $2, $3, 15)", uuid.NewV7(), acme.ID, carol)
	exec(t, pool, "INSERT INTO workspace_members (id, workspace_id, member_id, role) VALUES ($1, $2, $3, 15)", bobsID, acme.ID, bob)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	end := hold(t, tx, func(ctx context.Context) error {
		_, err := postgres.DB(ctx, pool).Exec(ctx, "SELECT id FROM workspace_members WHERE member_id = $1 FOR NO KEY UPDATE", carol)
		return err
	})
	done := make(chan error, 1)
	go func() {
		done <- tx.WithinTx(context.Background(), func(ctx context.Context) error {
			_, err := d.ShareMembers(ctx, acme.ID, []uuid.UUID{carol, bob})
			return err
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "workspace_members", 10*time.Second)

	err := withLockTimeout(tx, pool, func(ctx context.Context) error {
		_, err := postgres.DB(ctx, pool).Exec(ctx, "UPDATE workspace_members SET role = role WHERE id = $1", bobsID)
		return err
	})

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
		t.Errorf("updating bob's row while ShareMembers waits for carol's = %v; want lock_not_available: bob's is locked first", err)
	}
	if err := end(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("ShareMembers() = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("ShareMembers() did not end within 10s")
	}
}
