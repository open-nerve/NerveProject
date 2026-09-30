package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	identitydomain "github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Interleavings 9 and 18 of M3 design 9.3: creating invitations against
// the administrator's reset of the inviter's password, and two admins
// inviting overlapping batches; in both orders, on a real database, through
// the real use cases and identity's credential lock as bootstrap wires
// them. The reset and the creation wait on the inviter's account row
// (WaitForLockWaitOn "users"): the only row both lock. The later batch
// waits on no row: it inserts a key the earlier one has inserted and waits
// for that transaction to end, which WaitForKeyWaitOn sees and nothing else
// in these tests does; its transaction writes no other table, upgrades no
// row lock, and checks its foreign keys against rows no live transaction
// has updated.

// inviteRace is answerRace with carol, acme's second admin, and a live
// session of each admin, which the credential lock checks.
type inviteRace struct {
	answerRace
	carol                      uuid.UUID
	aliceSession, carolSession uuid.UUID
}

func newInviteRace(t *testing.T) inviteRace {
	t.Helper()
	r := inviteRace{answerRace: newAnswerRace(t), carol: uuid.NewV7(), aliceSession: uuid.NewV7(), carolSession: uuid.NewV7()}
	now := time.Now()
	users := identitypg.New(r.pool)
	if err := errors.Join(
		users.CreateUser(context.Background(), identityapp.NewUser{ID: r.carol, Email: "carol@example.com", PasswordHash: "x", DisplayName: "carol", Now: now}),
		users.CreateDefaultProfile(context.Background(), uuid.NewV7(), r.carol, now),
		users.CreateSession(context.Background(), identityapp.NewSession{ID: r.aliceSession, UserID: r.alice, TokenHash: make([]byte, 32),
			ExpiresAt: now.Add(time.Hour), Now: now}),
		users.CreateSession(context.Background(), identityapp.NewSession{ID: r.carolSession, UserID: r.carol, TokenHash: make([]byte, 32),
			ExpiresAt: now.Add(time.Hour), Now: now}),
	); err != nil {
		t.Fatal(err)
	}
	r.join(t, r.carol, shared.RoleAdmin)
	return r
}

// invite is the admin's creation of batch in acme, as his session, over
// invitations.
func (r inviteRace) invite(ctx context.Context, admin, session uuid.UUID, invitations workspaceapp.InvitationCreator, emails ...string) error {
	var batch []workspacedomain.NewInvitation
	for _, email := range emails {
		batch = append(batch, workspacedomain.NewInvitation{Email: email, Role: shared.RoleMember})
	}
	_, err := workspaceapp.NewCreateWorkspaceInvitations(workspaceapp.CreateInvitationsDeps{
		Caller: identity.Provide(r.pool).CredentialLock, Invitations: invitations, Profiles: workspaceProfiles{profiles: identity.Provide(r.pool).PublicProfiles},
		Auth: access.New(access.Deps{WorkspaceRoles: workspace.Provide(r.pool).WorkspaceRoles}), Tx: r.tx(), Clock: clocktest.At(time.Now()), MAC: r.mac,
	}).Execute(shared.WithActor(ctx, shared.Actor{UserID: admin, SessionID: session}), "acme", batch)
	return err
}

// invited are the addresses of acme's undeleted invitations but bob's,
// sorted, each with the admin who sent it.
func (r inviteRace) invited(t *testing.T) []string {
	t.Helper()
	rows, err := r.pool.Query(context.Background(), `SELECT i.email || ' by ' || u.email FROM workspace_member_invites i
		JOIN users u ON u.id = i.created_by_id WHERE i.workspace_id = $1 AND i.deleted_at IS NULL AND i.email <> 'bob@example.com'`, r.acme)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		got = append(got, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	return got
}

// gatedInviter stops a creation after the first row it inserts, holding
// that key, the workspace's FOR SHARE and the inviter's account row; the
// rest follow when the gate opens. A later batch that inserted out of order
// would hold a key the earlier still has to insert, and one of them would
// be chosen as a deadlock's victim (40P01).
type gatedInviter struct {
	*workspacepg.Store
	gate *gate
}

func (i gatedInviter) CreateInvitations(ctx context.Context, rows []workspaceapp.InvitationRow) ([]workspacedomain.Invitation, error) {
	first, err := i.Store.CreateInvitations(ctx, rows[:1])
	if err != nil {
		return nil, err
	}
	if err := i.gate.wait(ctx); err != nil {
		return nil, err
	}
	rest, err := i.Store.CreateInvitations(ctx, rows[1:])
	if err != nil {
		return nil, err
	}
	return append(first, rest...), nil
}

// fixedHasher hashes password as "hashed:<password>".
type fixedHasher struct{}

func (fixedHasher) Hash(_ context.Context, password string) (string, error) {
	return "hashed:" + password, nil
}

func (fixedHasher) Verify(_ context.Context, password, hash string) (bool, bool, error) {
	return hash == "hashed:"+password, false, nil
}

// gatedPasswords stops a reset before it writes the hash, holding the
// account row.
type gatedPasswords struct {
	identityapp.PasswordHashWriter
	gate *gate
}

func (p gatedPasswords) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, now time.Time) error {
	if err := p.gate.wait(ctx); err != nil {
		return err
	}
	return p.PasswordHashWriter.UpdatePasswordHash(ctx, id, hash, now)
}

// resetAlice is `nerve users reset-password` of alice, over passwords.
func (r inviteRace) resetAlice(ctx context.Context, passwords identityapp.PasswordHashWriter) error {
	store := identitypg.New(r.pool)
	_, err := identityapp.NewResetPassword(identityapp.ResetPasswordDeps{Accounts: store, Passwords: passwords, Sessions: store, APITokens: store,
		Hasher: fixedHasher{}, Rules: identitydomain.NewPasswordRules(), Tx: r.tx(), Clock: clocktest.At(time.Now()),
		Logger: slog.New(slog.DiscardHandler)}).Execute(ctx, "alice@example.com", "N3w-Passw0rd!")
	return err
}

// Interleaving 9: the reset first holds alice's account row; her creation
// waits on it, then finds her session revoked under the lock: 401, and no
// invitation. The creation first holds the row; the reset waits, and the
// invitation stays: it belongs to acme, not to her credential (3.8).
func TestInvitingAndResettingThePassword(t *testing.T) {
	for _, inviteFirst := range []bool{true, false} {
		name := map[bool]string{true: "the creation first", false: "the reset first"}[inviteFirst]
		t.Run(name, func(t *testing.T) {
			r := newInviteRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			store := workspacepg.New(r.pool)
			var invited, reset <-chan error
			if inviteFirst {
				invited = run(func() error {
					return r.invite(ctx, r.alice, r.aliceSession, gatedInviter{store, g}, "dave@example.com")
				})
				held(t, ctx, g, invited, "the creation")
				reset = run(func() error { return r.resetAlice(ctx, identitypg.New(r.pool)) })
			} else {
				reset = run(func() error { return r.resetAlice(ctx, gatedPasswords{identitypg.New(r.pool), g}) })
				held(t, ctx, g, reset, "the reset")
				invited = run(func() error { return r.invite(ctx, r.alice, r.aliceSession, store, "dave@example.com") })
			}
			pgtest.WaitForLockWaitOn(t, r.pool, "users", 5*time.Second)
			close(g.open)

			wantInvite := error(nil)
			if !inviteFirst {
				wantInvite = shared.Unauthenticated()
			}
			if err := result(t, ctx, invited, "the creation"); !errors.Is(err, wantInvite) {
				t.Errorf("the creation = %v, want %v", err, wantInvite)
			}
			if err := result(t, ctx, reset, "the reset"); err != nil {
				t.Errorf("the reset = %v, want it done", err)
			}
			var want []string
			if inviteFirst {
				want = []string{"dave@example.com by alice@example.com"}
			}
			if got := r.invited(t); !slices.Equal(got, want) {
				t.Errorf("acme's invitations: %q, want %q", got, want)
			}
		})
	}
}

// Interleaving 18: alice invites [x, y] and carol [y, x], each under acme's
// FOR SHARE, inserting in the order of the addresses. The later batch also
// invites w, an address of its own that sorts first. The earlier stops
// after x and inserts y when the gate opens; the later inserts w, then
// waits on x, the key the earlier inserted, not on a row, and once the
// earlier commits is refused 422 duplicate on x's index in its own
// request, its whole batch rolled back: w is gone too. Inserting in its
// request's order, the later would first hold y, which the earlier still
// has to insert: a deadlock (40P01). None, whichever goes first. The same
// for one address in both, in both orders.
func TestInvitingOverlappingBatches(t *testing.T) {
	w, x, y := "wendy@example.com", "xavier@example.com", "yvonne@example.com"
	for _, tt := range []struct {
		name         string
		aliceFirst   bool
		alice, carol []string // the later one's has w, which the earlier's must not: its first row is x
		field        string   // the later one's; x's place in the request is not its place in order
	}{
		{"alice first", true, []string{x, y}, []string{y, w, x}, "invitations[2].email"},
		{"carol first", false, []string{x, y, w}, []string{y, x}, "invitations[0].email"},
		{"one address, alice first", true, []string{x}, []string{x}, "invitations[0].email"},
		{"one address, carol first", false, []string{x}, []string{x}, "invitations[0].email"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := newInviteRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			store := workspacepg.New(r.pool)
			first, second := func(inv workspaceapp.InvitationCreator) error {
				return r.invite(ctx, r.alice, r.aliceSession, inv, tt.alice...)
			}, func(inv workspaceapp.InvitationCreator) error {
				return r.invite(ctx, r.carol, r.carolSession, inv, tt.carol...)
			}
			winner := "alice@example.com"
			if !tt.aliceFirst {
				first, second, winner = second, first, "carol@example.com"
			}
			earlier := run(func() error { return first(gatedInviter{store, g}) })
			held(t, ctx, g, earlier, "the earlier batch")
			later := run(func() error { return second(store) })
			pgtest.WaitForKeyWaitOn(t, r.pool, "workspace_member_invites", 5*time.Second)
			close(g.open)

			if err := result(t, ctx, earlier, "the earlier batch"); err != nil {
				t.Errorf("the earlier batch = %v, want it done", err)
			}
			err := result(t, ctx, later, "the later batch")
			var se *shared.Error
			if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || len(se.Fields) != 1 || se.Fields[0].Field != tt.field ||
				se.Fields[0].Code != shared.FieldDuplicate {
				t.Errorf("the later batch = %v, want 422 duplicate on %s alone", err, tt.field)
			}
			batch := tt.alice
			if !tt.aliceFirst {
				batch = tt.carol
			}
			var want []string
			for _, email := range batch {
				want = append(want, email+" by "+winner)
			}
			slices.Sort(want)
			if got := r.invited(t); !slices.Equal(got, want) {
				t.Errorf("acme's invitations: %q, want %q: the earlier batch's, none of the later", got, want)
			}
		})
	}
}
