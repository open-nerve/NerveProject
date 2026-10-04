package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	identitydomain "github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Interleavings 3, 12 and 19 of M3 design 9.3: an answer to an invitation
// against the deletion of its workspace, against the change of the
// invitee's address, and against his deactivation; each in both orders, on
// a real database, through the real use cases as bootstrap wires them. The
// side that goes first stops at a gate inside its transaction, holding its
// locks; the test waits until pgtest shows the other waiting on the table
// the design says it waits on, then opens the gate. Only the two sides run
// on the database, so the one wait each probe sees is the one meant: the
// deletion's and the acceptance's lock of the workspace row
// (WaitForLockWaitOn "workspaces"), and the lock of the invitee's account
// row (WaitForLockWaitOn "users"), which the answer takes first. Every wait
// has a deadline.

// answerRace is a database with acme, whose admin is alice, and bob, who is
// invited to acme at his address, each account with its profile;
// invitation is the link of bob's invitation, as a member.
type answerRace struct {
	pool       *pgxpool.Pool
	alice, bob uuid.UUID
	acme       uuid.UUID
	invitation invitationLink
	mac        workspaceapp.InvitationMAC
}

func newAnswerRace(t *testing.T) answerRace {
	t.Helper()
	pool := openPool(t, pgtest.NewDatabase(t))
	r := answerRace{pool: pool, alice: uuid.NewV7(), bob: uuid.NewV7(), acme: uuid.NewV7(), mac: testInvitationMAC(t)}
	now := time.Now()
	users := identitypg.New(pool)
	for id, email := range map[uuid.UUID]string{r.alice: "alice@example.com", r.bob: "bob@example.com"} {
		if err := errors.Join(users.CreateUser(context.Background(), identityapp.NewUser{ID: id, Email: email, PasswordHash: "x", DisplayName: "x", Now: now}),
			users.CreateDefaultProfile(context.Background(), uuid.NewV7(), id, now)); err != nil {
			t.Fatal(err)
		}
	}
	store := workspacepg.New(pool)
	if _, err := store.CreateWorkspace(context.Background(), workspaceapp.WorkspaceRow{ID: r.acme, Name: "Acme", Slug: "acme", Timezone: "UTC",
		CreatedBy: r.alice, Now: now}); err != nil {
		t.Fatal(err)
	}
	r.join(t, r.alice, shared.RoleAdmin)
	id := uuid.NewV7()
	if _, err := store.CreateInvitations(context.Background(), []workspaceapp.InvitationRow{
		{ID: id, WorkspaceID: r.acme, Email: "bob@example.com", Role: shared.RoleMember, CreatedBy: r.alice, Now: now},
	}); err != nil {
		t.Fatal(err)
	}
	r.invitation = invitationLink{id, invitationToken(t, id)}
	return r
}

// join makes user a member of acme with role.
func (r answerRace) join(t *testing.T, user uuid.UUID, role shared.Role) {
	t.Helper()
	if err := workspacepg.New(r.pool).CreateMember(context.Background(), workspaceapp.MemberRow{
		ID: uuid.NewV7(), WorkspaceID: r.acme, MemberID: user, Role: role, CreatedBy: r.alice, Now: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
}

func (r answerRace) tx() *postgres.TxManager { return postgres.NewTxManager(r.pool, 2*time.Second) }

func (r answerRace) accounts() workspaceapp.Accounts {
	return workspaceAccounts{accounts: identity.Provide(r.pool).Accounts}
}

// projects is project's cascade as bootstrap wires it.
func (r answerRace) projects() workspaceapp.ProjectCascade {
	return project.NewCascade(project.CascadeDeps{Pool: r.pool})
}

// accept is bob's acceptance of his invitation, over invitations.
func (r answerRace) accept(ctx context.Context, invitations workspaceapp.InvitationAccepter) error {
	_, err := workspaceapp.NewAcceptWorkspaceInvitation(workspaceapp.AcceptInvitationDeps{
		Accounts: r.accounts(), Invitations: invitations, Projects: r.projects(), Tx: r.tx(), Clock: clocktest.At(time.Now()), MAC: r.mac,
	}).Execute(shared.WithActor(ctx, shared.Actor{UserID: r.bob}), r.invitation.id, r.invitation.token)
	return err
}

// decline is bob's answer no, over invitations.
func (r answerRace) decline(ctx context.Context, invitations workspaceapp.InvitationDecliner) error {
	return workspaceapp.NewDeclineWorkspaceInvitation(r.accounts(), invitations, r.tx(), clocktest.At(time.Now()), r.mac).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: r.bob}), r.invitation.id, r.invitation.token)
}

// deleteAcme is alice's deletion of acme, over workspaces.
func (r answerRace) deleteAcme(ctx context.Context, workspaces workspaceapp.WorkspaceDeleter) error {
	return workspaceapp.NewDeleteWorkspace(workspaces, r.projects(), authorizerOn(r.pool), r.tx(), clocktest.At(time.Now()),
		slog.New(slog.DiscardHandler)).Execute(shared.WithActor(ctx, shared.Actor{UserID: r.alice}), "acme")
}

// bobIn is bob's membership of acme as it stands: whether there is an
// undeleted one; and whether acme is deleted.
func (r answerRace) bobIn(t *testing.T) (member, acmeDeleted bool) {
	t.Helper()
	if err := r.pool.QueryRow(context.Background(), `SELECT
		EXISTS (SELECT 1 FROM workspace_members WHERE workspace_id = $1 AND member_id = $2 AND deleted_at IS NULL),
		(SELECT deleted_at IS NOT NULL FROM workspaces WHERE id = $1)`, r.acme, r.bob).Scan(&member, &acmeDeleted); err != nil {
		t.Fatal(err)
	}
	return member, acmeDeleted
}

// answered is the invitation's state: accepted, answered, deleted.
func (r answerRace) answered(t *testing.T) (accepted, responded, deleted bool) {
	t.Helper()
	if err := r.pool.QueryRow(context.Background(), `SELECT accepted, responded_at IS NOT NULL, deleted_at IS NOT NULL
		FROM workspace_member_invites WHERE id = $1`, r.invitation.id).Scan(&accepted, &responded, &deleted); err != nil {
		t.Fatal(err)
	}
	return accepted, responded, deleted
}

// gatedAccepter stops an acceptance after its locks, before it reads the
// membership.
type gatedAccepter struct {
	*workspacepg.Store
	gate *gate
}

func (a gatedAccepter) MemberOf(ctx context.Context, workspaceID, userID uuid.UUID) (workspacedomain.Membership, bool, error) {
	if err := a.gate.wait(ctx); err != nil {
		return workspacedomain.Membership{}, false, err
	}
	return a.Store.MemberOf(ctx, workspaceID, userID)
}

// gatedDeleter stops a deletion after its lock and decision, before its
// first write.
type gatedDeleter struct {
	*workspacepg.Store
	gate *gate
}

func (d gatedDeleter) DeleteWorkspace(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := d.gate.wait(ctx); err != nil {
		return err
	}
	return d.Store.DeleteWorkspace(ctx, id, by, now)
}

// Interleaving 3: the acceptance first holds acme's row; the deletion waits
// on it, then deletes acme with bob's new membership. The deletion first
// holds it; the acceptance waits on it, then finds no workspace: 404, and
// bob is no member.
func TestAcceptingAndDeletingTheWorkspace(t *testing.T) {
	for _, acceptFirst := range []bool{true, false} {
		name := map[bool]string{true: "the acceptance first", false: "the deletion first"}[acceptFirst]
		t.Run(name, func(t *testing.T) {
			r := newAnswerRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			store := workspacepg.New(r.pool)
			var accepted, deleted <-chan error
			if acceptFirst {
				accepted = run(func() error { return r.accept(ctx, gatedAccepter{store, g}) })
				held(t, ctx, g, accepted, "the acceptance")
				deleted = run(func() error { return r.deleteAcme(ctx, store) })
			} else {
				deleted = run(func() error { return r.deleteAcme(ctx, gatedDeleter{store, g}) })
				held(t, ctx, g, deleted, "the deletion")
				accepted = run(func() error { return r.accept(ctx, store) })
			}
			pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
			close(g.open)

			wantAccept := error(nil)
			if !acceptFirst {
				wantAccept = workspacedomain.ErrInvitationNotFound
			}
			if err := result(t, ctx, accepted, "the acceptance"); !errors.Is(err, wantAccept) {
				t.Errorf("the acceptance = %v, want %v", err, wantAccept)
			}
			if err := result(t, ctx, deleted, "the deletion"); err != nil {
				t.Errorf("the deletion = %v, want it done", err)
			}
			var gone int
			if err := r.pool.QueryRow(context.Background(), `SELECT count(*) FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
				WHERE m.member_id = $1 AND m.deleted_at = w.deleted_at`, r.bob).Scan(&gone); err != nil {
				t.Fatal(err)
			}
			wantGone := map[bool]int{true: 1, false: 0}[acceptFirst]
			if member, acmeDeleted := r.bobIn(t); member || !acmeDeleted || gone != wantGone {
				t.Errorf("bob a member %v, acme deleted %v, bob's memberships deleted with acme %d; want none left, acme deleted, %d",
					member, acmeDeleted, gone, wantGone)
			}
			if acc, _, deleted := r.answered(t); acc != acceptFirst || !deleted {
				t.Errorf("the invitation accepted %v, deleted %v; want accepted %v, deleted", acc, deleted, acceptFirst)
			}
		})
	}
}

// setBobsEmail is `nerve users set-email` of bob's address, over sessions.
func (r answerRace) setBobsEmail(ctx context.Context, sessions identityapp.SessionRevoker) error {
	store := identitypg.New(r.pool)
	_, err := identityapp.NewSetEmail(identityapp.SetEmailDeps{Accounts: store, Users: store, Sessions: sessions, Tx: r.tx(),
		Clock: clocktest.At(time.Now()), Logger: slog.New(slog.DiscardHandler)}).Execute(ctx, "bob@example.com", "robert@example.com")
	return err
}

// Interleaving 12: the change of bob's address first holds his account row;
// the acceptance waits on it, then reads the new address under its lock:
// 403 workspace.invitation_email_mismatch, and nothing changes. The
// acceptance first holds the row FOR SHARE; the change waits, the
// acceptance makes bob a member by his address as it was, then the change
// goes on.
func TestAcceptingAndChangingTheAddress(t *testing.T) {
	for _, acceptFirst := range []bool{true, false} {
		name := map[bool]string{true: "the acceptance first", false: "the change first"}[acceptFirst]
		t.Run(name, func(t *testing.T) {
			r := newAnswerRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			store, users := workspacepg.New(r.pool), identitypg.New(r.pool)
			var accepted, changed <-chan error
			if acceptFirst {
				accepted = run(func() error { return r.accept(ctx, gatedAccepter{store, g}) })
				held(t, ctx, g, accepted, "the acceptance")
				changed = run(func() error { return r.setBobsEmail(ctx, users) })
			} else {
				changed = run(func() error { return r.setBobsEmail(ctx, gatedSessions{users, g}) })
				held(t, ctx, g, changed, "the change")
				accepted = run(func() error { return r.accept(ctx, store) })
			}
			pgtest.WaitForLockWaitOn(t, r.pool, "users", 5*time.Second)
			close(g.open)

			wantAccept := error(nil)
			if !acceptFirst {
				wantAccept = workspacedomain.ErrInvitationEmailMismatch
			}
			if err := result(t, ctx, accepted, "the acceptance"); !errors.Is(err, wantAccept) {
				t.Errorf("the acceptance = %v, want %v", err, wantAccept)
			}
			if err := result(t, ctx, changed, "the change"); err != nil {
				t.Errorf("the change = %v, want it done", err)
			}
			member, _ := r.bobIn(t)
			acc, responded, deleted := r.answered(t)
			if member != acceptFirst || acc != acceptFirst || responded != acceptFirst || deleted != acceptFirst {
				t.Errorf("bob a member %v; the invitation accepted %v, answered %v, deleted %v; want all %v", member, acc, responded, deleted, acceptFirst)
			}
		})
	}
}

// gatedDeactivation takes, after it revokes the sessions, the lock of acme
// that P6's deactivation takes (M3 design 3.9: the account row, profiles,
// auth_sessions, then each workspace FOR NO KEY UPDATE), and stops there
// when it has a gate.
type gatedDeactivation struct {
	identityapp.SessionRevoker
	pool *pgxpool.Pool
	acme uuid.UUID
	gate *gate // nil: never stops
}

func (d gatedDeactivation) RevokeSessions(ctx context.Context, userID, keep uuid.UUID, reason identitydomain.RevokeReason, now time.Time) (int, error) {
	revoked, err := d.SessionRevoker.RevokeSessions(ctx, userID, keep, reason, now)
	if err == nil {
		_, err = postgres.DB(ctx, d.pool).Exec(ctx, "SELECT id FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", d.acme)
	}
	if err == nil && d.gate != nil {
		err = d.gate.wait(ctx)
	}
	return revoked, err
}

// deactivateBob is `nerve users deactivate` of bob, over sessions.
func (r answerRace) deactivateBob(ctx context.Context, sessions identityapp.SessionRevoker) error {
	store := identitypg.New(r.pool)
	_, err := identityapp.NewDeactivate(identityapp.DeactivateDeps{Accounts: store, Users: store, Profiles: store, Sessions: sessions, Tx: r.tx(),
		Clock: clocktest.At(time.Now()), Logger: slog.New(slog.DiscardHandler)}).ExecuteByEmail(ctx, "bob@example.com")
	return err
}

// gatedDecliner stops a decline after its locks, before its write.
type gatedDecliner struct {
	*workspacepg.Store
	gate *gate
}

func (d gatedDecliner) DeclineInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := d.gate.wait(ctx); err != nil {
		return err
	}
	return d.Store.DeclineInvitation(ctx, id, by, now)
}

// Interleaving 19: bob is acme's member and has an invitation to it, as
// after reactivate-member or a change of address. The decline first holds
// his account row FOR SHARE, then acme's; the deactivation waits on the
// account row, then goes on through acme. The deactivation first holds the
// account row and acme's; the decline waits on the account row, then reads
// it deactivated under its lock: 401, and the invitation stays pending.
// Neither deadlocks: both lock the account row first (3.6 convention 1).
func TestDecliningAndDeactivating(t *testing.T) {
	for _, declineFirst := range []bool{true, false} {
		name := map[bool]string{true: "the decline first", false: "the deactivation first"}[declineFirst]
		t.Run(name, func(t *testing.T) {
			r := newAnswerRace(t)
			r.join(t, r.bob, shared.RoleMember)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			store, users := workspacepg.New(r.pool), identitypg.New(r.pool)
			var declined, deactivated <-chan error
			if declineFirst {
				declined = run(func() error { return r.decline(ctx, gatedDecliner{store, g}) })
				held(t, ctx, g, declined, "the decline")
				deactivated = run(func() error { return r.deactivateBob(ctx, gatedDeactivation{users, r.pool, r.acme, nil}) })
			} else {
				deactivated = run(func() error { return r.deactivateBob(ctx, gatedDeactivation{users, r.pool, r.acme, g}) })
				held(t, ctx, g, deactivated, "the deactivation")
				declined = run(func() error { return r.decline(ctx, store) })
			}
			pgtest.WaitForLockWaitOn(t, r.pool, "users", 5*time.Second)
			close(g.open)

			wantDecline := error(nil)
			if !declineFirst {
				wantDecline = shared.Unauthenticated()
			}
			if err := result(t, ctx, declined, "the decline"); !errors.Is(err, wantDecline) {
				t.Errorf("the decline = %v, want %v", err, wantDecline)
			}
			if err := result(t, ctx, deactivated, "the deactivation"); err != nil {
				t.Errorf("the deactivation = %v, want it done", err)
			}
			var active bool
			if err := r.pool.QueryRow(context.Background(), "SELECT is_active FROM users WHERE id = $1", r.bob).Scan(&active); err != nil {
				t.Fatal(err)
			}
			acc, responded, deleted := r.answered(t)
			if active || acc || responded != declineFirst || deleted {
				t.Errorf("bob active %v; the invitation accepted %v, answered %v, deleted %v; want deactivated, answered %v, undeleted",
					active, acc, responded, deleted, declineFirst)
			}
		})
	}
}
