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
// workspace, or of a deleted row does not, and updates its one row, so it
// cannot pass by matching none.
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
		var updated int64
		err := withLockTimeout(tx, pool, func(ctx context.Context) error {
			tag, err := postgres.DB(ctx, pool).Exec(ctx, "UPDATE workspace_members SET role = role WHERE workspace_id = $1 AND member_id = $2",
				tt.workspace, tt.user)
			updated = tag.RowsAffected()
			return err
		})
		var pgErr *pgconn.PgError
		if waited := errors.As(err, &pgErr) && pgErr.Code == "55P03"; waited != tt.waits || (!waited && (err != nil || updated != 1)) {
			t.Errorf("%s: updating it = %d rows, %v; want a wait %v, or else its one row", tt.name, updated, err, tt.waits)
		}
	}
}

// ShareMembers takes its locks in the memberships' id order, whatever
// order the rows lie in, in the table or in an index. By id the
// memberships are bob's, carol's, dave's; in the table carol's, bob's,
// dave's; by account, as the (workspace_id, member_id) index lists them,
// bob's, dave's, carol's. No order of the table or of an index, forwards
// or backwards, is the ids'. carol's row, the middle one, is held.
// ShareMembers waits for it holding bob's and not yet dave's: an update of
// bob's row waits, one of dave's does not. In any other order it would
// wait for carol's holding nothing, or holding dave's.
func TestShareMembersLocksInIDOrder(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	alice := newAccount(t, pool, "alice@corp.com")
	bob, dave, carol := newAccount(t, pool, "bob@corp.com"), newAccount(t, pool, "dave@corp.com"), newAccount(t, pool, "carol@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	bobs, carols, daves := uuid.NewV7(), uuid.NewV7(), uuid.NewV7() // drawn in the id order
	for _, m := range []struct{ id, user uuid.UUID }{{carols, carol}, {bobs, bob}, {daves, dave}} {
		exec(t, pool, "INSERT INTO workspace_members (id, workspace_id, member_id, role) VALUES ($1, $2, $3, 15)", m.id, acme.ID, m.user)
	}
	tx := postgres.NewTxManager(pool, 2*time.Second)
	end := hold(t, tx, func(ctx context.Context) error {
		_, err := postgres.DB(ctx, pool).Exec(ctx, "SELECT id FROM workspace_members WHERE id = $1 FOR NO KEY UPDATE", carols)
		return err
	})
	done := make(chan error, 1)
	go func() {
		done <- tx.WithinTx(context.Background(), func(ctx context.Context) error {
			_, err := d.ShareMembers(ctx, acme.ID, []uuid.UUID{carol, dave, bob})
			return err
		})
	}()
	pgtest.WaitForLockWaitOn(t, pool, "workspace_members", 10*time.Second)

	for _, tt := range []struct {
		name  string
		id    uuid.UUID
		waits bool
	}{
		{"bob's row (before carol's by id)", bobs, true},
		{"dave's row (after carol's by id)", daves, false},
	} {
		var updated int64
		err := withLockTimeout(tx, pool, func(ctx context.Context) error {
			tag, err := postgres.DB(ctx, pool).Exec(ctx, "UPDATE workspace_members SET role = role WHERE id = $1", tt.id)
			updated = tag.RowsAffected()
			return err
		})
		var pgErr *pgconn.PgError
		if waited := errors.As(err, &pgErr) && pgErr.Code == "55P03"; waited != tt.waits || (!waited && (err != nil || updated != 1)) {
			t.Errorf("updating %s while ShareMembers waits for carol's = %d rows, %v; want a wait %v, or else its one row", tt.name, updated, err, tt.waits)
		}
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

// ShareMembers' lock is convention 3's FOR SHARE: while one holds the rows,
// a second ShareMembers, as two writes adding projects with the same admins
// take, answers without a wait, and an update of a held row waits. (That
// ShareMembers waits for an update is TestShareMembersLocksInIDOrder's.)
// A wait ends with lock_not_available under a lock_timeout.
func TestShareMembersLockIsForShare(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	join(t, s, acme.ID, bob, shared.RoleMember)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	hold(t, tx, func(ctx context.Context) error {
		_, err := d.ShareMembers(ctx, acme.ID, []uuid.UUID{alice, bob})
		return err
	})

	var got map[uuid.UUID]shared.Role
	err := withLockTimeout(tx, pool, func(ctx context.Context) error {
		var err error
		got, err = d.ShareMembers(ctx, acme.ID, []uuid.UUID{alice, bob})
		return err
	})
	want := map[uuid.UUID]shared.Role{alice: shared.RoleAdmin, bob: shared.RoleMember}
	if err != nil || !maps.Equal(got, want) {
		t.Errorf("ShareMembers() while another holds the rows = %v, %v; want %v without a wait", got, err, want)
	}

	err = withLockTimeout(tx, pool, func(ctx context.Context) error {
		_, err := postgres.DB(ctx, pool).Exec(ctx, "UPDATE workspace_members SET role = role WHERE workspace_id = $1 AND member_id = $2",
			acme.ID, bob)
		return err
	})
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
		t.Errorf("updating bob's row while ShareMembers holds it = %v; want lock_not_available after waiting", err)
	}
}

// The directory's read takes no lock: it answers a workspace whose row a
// transaction holds FOR UPDATE, which every row lock waits for, without
// waiting. The holder locks acme's row, one row, so it cannot pass by
// holding none.
func TestTheDirectorysReadTakesNoLock(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	hold(t, tx, func(ctx context.Context) error {
		tag, err := postgres.DB(ctx, pool).Exec(ctx, "SELECT id FROM workspaces WHERE id = $1 FOR UPDATE", acme.ID)
		if err == nil && tag.RowsAffected() != 1 {
			err = errors.New("acme's row is not held")
		}
		return err
	})

	var got app.DirectoryEntry
	var found bool
	err := withLockTimeout(tx, pool, func(ctx context.Context) error {
		var err error
		got, found, err = d.WorkspaceBySlug(ctx, "acme")
		return err
	})

	if want := (app.DirectoryEntry{ID: acme.ID, Timezone: "UTC"}); err != nil || !found || got != want {
		t.Errorf("WorkspaceBySlug() while acme's row is held FOR UPDATE = %+v, %v, %v; want %+v without a wait", got, found, err, want)
	}
}

// A directory call that fails answers its error, never an answer: not "no
// such workspace", which a use case would turn into workspace.not_found,
// and not "no active member", which would refuse a project's lead. Each
// runs on a cancelled context against a workspace alice administers, so
// that neither empty answer is right.
func TestAFailedDirectoryCallIsAnErrorNotAnAnswer(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	alice := newAccount(t, pool, "alice@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	failed := func(err error) bool { return errors.Is(err, context.Canceled) }

	for name, find := range map[string]func(context.Context, string) (app.DirectoryEntry, bool, error){
		"WorkspaceBySlug": d.WorkspaceBySlug, "ShareWorkspaceBySlug": d.ShareWorkspaceBySlug,
	} {
		if got, found, err := find(cancelled, "acme"); !failed(err) || found || got != (app.DirectoryEntry{}) {
			t.Errorf("%s() = %+v, %v, %v; want context.Canceled, not no workspace", name, got, found, err)
		}
	}
	if roles, err := d.ShareMembers(cancelled, acme.ID, []uuid.UUID{alice}); !failed(err) || roles != nil {
		t.Errorf("ShareMembers() = %v, %v; want context.Canceled, no roles", roles, err)
	}
}
