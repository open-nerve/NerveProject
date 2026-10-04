package postgresadapter_test

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// noProjectRows is the project module's cascade over a database without
// projects: no step has a row to write.
type noProjectRows struct{}

func (noProjectRows) DeleteWorkspaceProjects(context.Context, uuid.UUID, uuid.UUID, time.Time) error {
	return nil
}

func (noProjectRows) DemoteToGuest(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, time.Time) error {
	return nil
}

func (noProjectRows) EndMemberships(context.Context, []uuid.UUID, uuid.UUID, uuid.UUID, time.Time) error {
	return nil
}

// connOf is the connection of the transaction ctx carries.
func connOf(ctx context.Context, pool *pgxpool.Pool) (*pgconn.PgConn, error) {
	tx, ok := postgres.DB(ctx, pool).(pgx.Tx)
	if !ok {
		return nil, errors.New("no transaction in the context")
	}
	return tx.Conn().PgConn(), nil
}

// DeleteWorkspaceInvitations takes the workspace's undeleted invitations
// in id order, whatever order the rows lie in, FOR NO KEY UPDATE, before it
// deletes them, and takes no other row (M3 design 3.6 convention 5; ruling
// G-1). acme's undeleted invitations, to zoe, yan and xia, have ids in that
// order, but lie the other way round, in the table as in each index but the
// primary key's: xia's was inserted first, zoe's last, and the addresses
// sort the other way. beta's invitation and acme's deleted one, to wes, have
// the smallest ids. Three transactions hold zoe's, yan's and xia's FOR
// SHARE, one each, and let go one at a time, each step's probe naming the
// holder the deletion then waits behind: zoe's first, holding no row; then
// yan's, holding zoe's; then xia's, holding zoe's and yan's. In the table's
// order, or that of an index other than the primary key's, it would wait
// behind xia's first. Once xia's lets go it deletes the three, at its moment
// and by its account; wes's keeps its deletion, and beta's is not deleted.
func TestDeleteWorkspaceInvitationsLocksInIDOrder(t *testing.T) {
	s, pool := newStoreWithConns(t, 8)
	alice := newAccount(t, pool, "alice@corp.com")
	acme, beta := newWorkspace(t, s, "acme", "acme", alice).ID, newWorkspace(t, s, "beta", "beta", alice).ID
	elsewhere := inviteWithID(t, s, uuid.NewV7(), beta, "zoe@corp.com", shared.RoleMember, alice).ID
	gone := inviteWithID(t, s, uuid.NewV7(), acme, "wes@corp.com", shared.RoleMember, alice).ID
	exec(t, pool, "UPDATE workspace_member_invites SET deleted_at = $2 WHERE id = $1", gone, now)
	ids, to := []uuid.UUID{uuid.NewV7(), uuid.NewV7(), uuid.NewV7()}, []string{"zoe", "yan", "xia"}
	names := map[uuid.UUID]string{elsewhere: "beta's", gone: "wes's"}
	for i := len(ids) - 1; i >= 0; i-- {
		inviteWithID(t, s, ids[i], acme, to[i]+"@corp.com", shared.RoleMember, alice)
		names[ids[i]] = to[i] + "'s"
	}
	all := slices.Concat([]uuid.UUID{elsewhere, gone}, ids)
	holders := make([]pgx.Tx, len(ids))
	for i, id := range ids {
		tx, err := pool.Begin(pgtest.Soon(t))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
		if _, err := tx.Exec(pgtest.Soon(t), "SELECT 1 FROM workspace_member_invites WHERE id = $1 FOR SHARE", id); err != nil {
			t.Fatal(err)
		}
		holders[i] = tx
	}
	later := now.Add(time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- postgres.NewTxManager(pool, 10*time.Second).WithinTx(ctx, func(ctx context.Context) error {
			return s.DeleteWorkspaceInvitations(ctx, acme, alice, later)
		})
	}()
	// locked names the rows the deletion holds FOR NO KEY UPDATE: a FOR
	// SHARE of each waits, a FOR KEY SHARE does not.
	locked := func() string {
		var held []string
		for _, id := range all {
			if waitsFor(t, pool, "workspace_member_invites", id, "FOR SHARE") {
				held = append(held, names[id])
			}
			if waitsFor(t, pool, "workspace_member_invites", id, "FOR KEY SHARE") {
				held = append(held, names[id]+" FOR UPDATE")
			}
		}
		return strings.Join(held, ", ")
	}

	for i := range ids {
		if i > 0 {
			if err := holders[i-1].Rollback(context.Background()); err != nil {
				t.Fatal(err)
			}
		}
		pgtest.WaitForLockWaitBehind(t, pool, holders[i].Conn().PgConn(), 5*time.Second)
		want := make([]string, i)
		for j := range i {
			want[j] = names[ids[j]]
		}
		if got := locked(); got != strings.Join(want, ", ") {
			t.Errorf("the deletion waiting for %s holds %q; want %q", names[ids[i]], got, strings.Join(want, ", "))
		}
	}
	if err := holders[len(ids)-1].Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("DeleteWorkspaceInvitations() = %v", err)
		}
	case <-ctx.Done():
		t.Fatal("DeleteWorkspaceInvitations() did not end within 20s")
	}
	for _, id := range all {
		want := "deleted at " + at(t, pool, later) + " at " + at(t, pool, later) + " by " + alice.String()
		switch id {
		case elsewhere:
			want = "undeleted at " + at(t, pool, now) + " by " + alice.String()
		case gone:
			want = "deleted at " + at(t, pool, now) + " at " + at(t, pool, now) + " by " + alice.String()
		}
		if got := stamp(t, pool, "workspace_member_invites", id); got != want {
			t.Errorf("%s invitation: %s, want %s", names[id], got, want)
		}
	}
}

// A workspace's deletion and two deactivations all go through, where the
// workspace's invitations lie out of their id order (M3 design 3.6
// convention 5; the P6 re-review's cycle, ruling G-1). erin's y invites
// bob and carol; carol's x, where she is alone, and erin's v invite bob.
// The ids run carol's to y, bob's to x, his to v, his to y; bob's to y lies
// first in y's rows, in the table as in the address's index. Another
// transaction, an updateWorkspaceInvitation of bob's to v, holds that row.
// bob's deactivation (the Deactivator, under his account's row) locks his
// to x, then waits for it; carol's locks hers to y, then waits for bob's to
// x, which it deletes too, x left with no active member; y's deletion (the
// DeleteWorkspace use case) waits for carol's to y. Taking y's invitations
// in id order, it holds none of them meanwhile; once the update commits,
// bob's takes his to v and to y, and all three are done, each invitation
// deleted by the deactivation that locked it first, and y by erin. In the
// table's order, the deletion would hold bob's to y while it waited, which
// bob's would wait for: a cycle of three, 40P01. In the control, bob's
// address is zed's, which sorts after carol's, and his invitation to y
// lies after hers: the table's order is the ids' too.
func TestAWorkspaceDeletionAndTwoDeactivationsAllGoThrough(t *testing.T) {
	for _, bobs := range []string{"bob", "zed"} {
		name := "y's rows out of their id order"
		if bobs == "zed" {
			name = "the control, y's rows in their id order"
		}
		t.Run(name, func(t *testing.T) {
			s, pool := newStoreWithConns(t, 10)
			bobAt := bobs + "@corp.com"
			bob, carol, erin := newAccount(t, pool, bobAt), newAccount(t, pool, "carol@corp.com"), newAccount(t, pool, "erin@corp.com")
			x, y, v := newWorkspace(t, s, "x", "x", carol).ID, newWorkspace(t, s, "y", "y", erin).ID, newWorkspace(t, s, "v", "v", erin).ID
			toCarol, toX, toV, toY := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			ys := []struct {
				id    uuid.UUID
				email string
			}{{toY, bobAt}, {toCarol, "carol@corp.com"}}
			if bobs == "zed" {
				ys[0], ys[1] = ys[1], ys[0]
			}
			for _, inv := range ys {
				inviteWithID(t, s, inv.id, y, inv.email, shared.RoleMember, erin)
			}
			inviteWithID(t, s, toX, x, bobAt, shared.RoleMember, carol)
			inviteWithID(t, s, toV, v, bobAt, shared.RoleMember, erin)
			tx := postgres.NewTxManager(pool, 10*time.Second)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			var update *pgconn.PgConn
			commit := hold(t, tx, func(ctx context.Context) (err error) {
				if update, err = connOf(ctx, pool); err != nil {
					return err
				}
				if err := s.ShareWorkspace(ctx, v); err != nil {
					return err
				}
				if _, err := s.LockInvitation(ctx, toV); err != nil {
					return err
				}
				_, err = s.UpdateInvitationRole(ctx, toV, shared.RoleGuest, erin, now)
				return err
			})
			deactivate := func(user uuid.UUID, email string) (<-chan error, *pgconn.PgConn) {
				t.Helper()
				conns, done := make(chan *pgconn.PgConn, 1), make(chan error, 1)
				d := app.NewDeactivator(s, noProjectRows{}, clocktest.At(now))
				go func() {
					done <- tx.WithinTx(ctx, func(ctx context.Context) error {
						conn, err := connOf(ctx, pool)
						if err != nil {
							return err
						}
						conns <- conn
						if _, err := postgres.DB(ctx, pool).Exec(ctx, "SELECT 1 FROM users WHERE id = $1 FOR NO KEY UPDATE", user); err != nil {
							return err
						}
						return d.DeactivateMemberships(ctx, user, email)
					})
				}()
				select {
				case conn := <-conns:
					return done, conn
				case err := <-done:
					t.Fatalf("%s's deactivation ended before its first statement: %v", email, err)
				case <-ctx.Done():
					t.Fatalf("%s's deactivation did not begin within 20s", email)
				}
				return nil, nil
			}
			bobsDone, bobsConn := deactivate(bob, bobAt)
			pgtest.WaitForLockWaitBehind(t, pool, update, 5*time.Second)
			carolsDone, carolsConn := deactivate(carol, "carol@corp.com")
			pgtest.WaitForLockWaitBehind(t, pool, bobsConn, 5*time.Second)
			deletion := app.NewDeleteWorkspace(s, noProjectRows{}, allowAll{}, tx, clocktest.At(now), slog.New(slog.DiscardHandler))
			deleted := make(chan error, 1)
			go func() { deleted <- deletion.Execute(shared.WithActor(ctx, shared.Actor{UserID: erin}), "y") }()
			pgtest.WaitForLockWaitBehind(t, pool, carolsConn, 5*time.Second)
			if err := commit(); err != nil {
				t.Fatal(err)
			}

			for _, r := range []struct {
				what string
				done <-chan error
			}{{bobs + "'s deactivation", bobsDone}, {"carol's deactivation", carolsDone}, {"y's deletion", deleted}} {
				select {
				case err := <-r.done:
					var pgErr *pgconn.PgError
					switch {
					case errors.As(err, &pgErr) && pgErr.Code == "40P01":
						t.Errorf("%s: a deadlock, 40P01: %v", r.what, err)
					case err != nil:
						t.Errorf("%s = %v; want it done", r.what, err)
					}
				case <-ctx.Done():
					t.Fatalf("%s did not end within 20s", r.what)
				}
			}
			var got string
			if err := pool.QueryRow(pgtest.Soon(t), `SELECT concat_ws('; ',
				(SELECT string_agg(w.slug || ' ' || split_part(i.email, '@', 1) || CASE WHEN i.deleted_at IS NULL THEN ' undeleted'
					ELSE ' deleted by ' || split_part(u.email, '@', 1) END, ', ' ORDER BY i.id)
					FROM workspace_member_invites i JOIN workspaces w ON w.id = i.workspace_id JOIN users u ON u.id = i.updated_by_id),
				(SELECT string_agg(w.slug || CASE WHEN w.deleted_at IS NULL THEN ' undeleted' ELSE ' deleted by ' || split_part(u.email, '@', 1) END,
					', ' ORDER BY w.slug) FROM workspaces w LEFT JOIN users u ON u.id = w.updated_by_id))`).Scan(&got); err != nil {
				t.Fatal(err)
			}
			if want := "y carol deleted by carol, x " + bobs + " deleted by " + bobs + ", v " + bobs + " deleted by " + bobs + ", y " + bobs +
				" deleted by " + bobs + "; v undeleted, x undeleted, y deleted by erin"; got != want {
				t.Errorf("after all three: %s; want %s", got, want)
			}
		})
	}
}
