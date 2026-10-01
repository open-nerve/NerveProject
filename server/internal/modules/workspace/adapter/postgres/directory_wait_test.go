package postgresadapter_test

import (
	"context"
	"fmt"
	"maps"
	"testing"
	"time"
	"uuid"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ShareMembers answers the memberships as they are after its wait (M3
// design 3.6 conventions 3 and 6), as the directory's lock does the
// workspace (TestTheDirectorysLockSeesADeletionItWaitedFor): Postgres
// evaluates deleted_at IS NULL again on a row's newest version, and the
// roles come from the rows it locked. A transaction ends bob's membership,
// makes carol a guest and deletes dave's, each one row; ShareMembers waits
// for it and, once it commits, answers alice the admin and carol the guest,
// not bob's ended membership nor dave's deleted one. An answer read before
// the lock would have bob, carol and dave as members.
func TestShareMembersAnswersTheRowsAsTheyAreAfterItsWait(t *testing.T) {
	s, pool := newStore(t)
	d := postgresadapter.NewDirectory(pool)
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	carol, dave := newAccount(t, pool, "carol@corp.com"), newAccount(t, pool, "dave@corp.com")
	acme := newWorkspace(t, s, "Acme", "acme", alice)
	for _, user := range []uuid.UUID{bob, carol, dave} {
		join(t, s, acme.ID, user, shared.RoleMember)
	}
	tx := postgres.NewTxManager(pool, 2*time.Second)
	end := hold(t, tx, func(ctx context.Context) error {
		for _, change := range []struct {
			sql  string
			user uuid.UUID
		}{
			{"UPDATE workspace_members SET is_active = false WHERE workspace_id = $1 AND member_id = $2", bob},
			{"UPDATE workspace_members SET role = 5 WHERE workspace_id = $1 AND member_id = $2", carol},
			{"UPDATE workspace_members SET deleted_at = now() WHERE workspace_id = $1 AND member_id = $2", dave},
		} {
			tag, err := postgres.DB(ctx, pool).Exec(ctx, change.sql, acme.ID, change.user)
			if err == nil && tag.RowsAffected() != 1 {
				err = fmt.Errorf("%q changed %d rows; want 1", change.sql, tag.RowsAffected())
			}
			if err != nil {
				return err
			}
		}
		return nil
	})
	type answer struct {
		roles map[uuid.UUID]shared.Role
		err   error
	}
	done := make(chan answer, 1)
	go func() {
		var a answer
		a.err = tx.WithinTx(context.Background(), func(ctx context.Context) error {
			var err error
			a.roles, err = d.ShareMembers(ctx, acme.ID, []uuid.UUID{alice, bob, carol, dave})
			return err
		})
		done <- a
	}()
	pgtest.WaitForLockWaitOn(t, pool, "workspace_members", 10*time.Second)
	if err := end(); err != nil {
		t.Fatal(err)
	}
	select {
	case a := <-done:
		want := map[uuid.UUID]shared.Role{alice: shared.RoleAdmin, carol: shared.RoleGuest}
		if a.err != nil || !maps.Equal(a.roles, want) {
			t.Errorf("ShareMembers() after the wait = %v, %v; want %v", a.roles, a.err, want)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("ShareMembers() did not end within 10s")
	}
}
