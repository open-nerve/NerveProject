package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Interleaving 7 of M3 design 9.3 (Codex S1): bob's acceptance of his
// invitation to acme, a workspace he is not in, against his deactivation,
// `nerve users deactivate` as bootstrap wires it (deactivating), each in
// its own transaction on a real database. Both lock bob's account row
// first (3.6 convention 6), the acceptance FOR SHARE and the deactivation
// FOR NO KEY UPDATE, so the deactivation's enumeration of his workspaces
// comes after any acceptance that committed first. Every wait has a
// deadline.

// s1Race is answerRace with bob invited to acme as its admin, and Web,
// acme's public project, of which alice is the admin; when former, bob has
// a membership of acme, as a member, which alice's removal ended before she
// invited him again (3.8).
type s1Race struct {
	answerRace
	web, aliceIn uuid.UUID
}

func newS1Race(t *testing.T, former bool) s1Race {
	t.Helper()
	r := s1Race{answerRace: newAnswerRace(t, shared.RoleAdmin), web: uuid.NewV7()}
	ctx, before := soon(t), time.Now().Add(-time.Hour)
	r.aliceIn = queryIDs(t, r.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", r.acme, r.alice)[0]
	projects := projectpg.New(r.pool)
	if err := projects.CreateProject(ctx, projectapp.ProjectRow{ID: r.web, WorkspaceID: r.acme, Name: "Web", Identifier: "WEB",
		Network: projectdomain.NetworkPublic, Timezone: "UTC", CreatedBy: r.alice, Now: before}); err != nil {
		t.Fatal(err)
	}
	if err := projects.CreateMember(ctx, projectapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: r.acme, ProjectID: r.web, MemberID: r.alice,
		Role: shared.RoleAdmin, CreatedBy: r.alice, Now: before}); err != nil {
		t.Fatal(err)
	}
	if former {
		workspaces := workspacepg.New(r.pool)
		if err := workspaces.CreateMember(ctx, workspaceapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: r.acme, MemberID: r.bob,
			Role: shared.RoleMember, CreatedBy: r.alice, Now: before}); err != nil {
			t.Fatal(err)
		}
		if err := workspaces.EndMember(ctx, r.acme, r.bob, r.alice, before); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

// standing is bob's account, active or deactivated; each of his
// memberships of acme, in the order they were made, and of Web, with its
// role, and its ender when it has ended; the invitation, accepted or
// unanswered, and who deleted it when it is deleted; and alice's role in
// acme.
func (r s1Race) standing(t *testing.T) string {
	t.Helper()
	var s string
	if err := r.pool.QueryRow(soon(t), `SELECT concat_ws('; ', CASE WHEN u.is_active THEN 'bob active' ELSE 'bob deactivated' END,
		(SELECT string_agg('acme ' || m.role || CASE WHEN m.is_active THEN ' active' ELSE ' ended by ' || split_part(e.email, '@', 1) END,
			', ' ORDER BY m.created_at) FROM workspace_members m JOIN users e ON e.id = m.updated_by_id
			WHERE m.workspace_id = $2 AND m.member_id = u.id),
		(SELECT 'Web ' || m.role || CASE WHEN m.is_active THEN ' active' ELSE ' ended by ' || split_part(e.email, '@', 1) END
			FROM project_members m JOIN users e ON e.id = m.updated_by_id WHERE m.project_id = $3 AND m.member_id = u.id),
		(SELECT 'invitation ' || CASE WHEN i.accepted THEN 'accepted' ELSE 'unanswered' END ||
			CASE WHEN i.deleted_at IS NULL THEN '' ELSE ' deleted by ' || split_part(e.email, '@', 1) END
			FROM workspace_member_invites i LEFT JOIN users e ON e.id = i.updated_by_id WHERE i.id = $4),
		(SELECT 'alice ' || m.role FROM workspace_members m WHERE m.id = $5))
		FROM users u WHERE u.id = $1`, r.bob, r.acme, r.web, r.invitation.id, r.aliceIn).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// changeRole is by's change of the role of the workspace membership id to
// role, updateWorkspaceMember over members and cascade, with identity's
// profiles and the Authorizer as bootstrap wires them, on pool and the
// system's clock: its time is read when it reads it. It takes no test, so
// it runs on any goroutine.
func changeRole(ctx context.Context, pool *pgxpool.Pool, by, id uuid.UUID, role shared.Role, members workspaceapp.MemberUpdater,
	cascade workspaceapp.ProjectCascade) error {
	_, err := workspaceapp.NewUpdateWorkspaceMember(members, cascade, workspaceProfiles{profiles: identity.Provide(pool).PublicProfiles},
		authorizerOn(pool), postgres.NewTxManager(pool, 2*time.Second), clock.System{}).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: by}), id, role)
	return err
}

// Interleaving 7 (a), (d) and (e):
//   - (a) the acceptance first holds bob's account row FOR SHARE, and acme's
//     FOR NO KEY UPDATE, at its gate; the deactivation waits for his
//     account's row. Once the acceptance has committed, the deactivation
//     finds acme among his workspaces and ends his membership of it, as
//     his: he is no admin acme lacks, alice being one.
//   - (d) the deactivation first holds his account row at its gate, before
//     its memberships' step; the acceptance waits for it, then reads his
//     account deactivated under its lock: 401, no membership. The
//     deactivation deleted the invitation, as his.
//   - (e) bob has a former membership of acme: the acceptance restores that
//     row rather than inserting one, and serializes on his account row as
//     an insertion does: (a) ends the restored row, his only one; (d) leaves
//     it as alice's removal did.
//
// The probe sees the one wait on users that either order has.
func TestAcceptingAndDeactivating(t *testing.T) {
	for _, tt := range []struct {
		former, acceptFirst bool
		accept              error
		want                string
	}{
		{false, true, nil, "bob deactivated; acme 20 ended by bob; invitation accepted deleted by bob; alice 20"},
		{false, false, shared.Unauthenticated(), "bob deactivated; invitation unanswered deleted by bob; alice 20"},
		{true, true, nil, "bob deactivated; acme 20 ended by bob; invitation accepted deleted by bob; alice 20"},
		{true, false, shared.Unauthenticated(), "bob deactivated; acme 15 ended by alice; invitation unanswered deleted by bob; alice 20"},
	} {
		t.Run(fmt.Sprintf("former %v, the acceptance first %v", tt.former, tt.acceptFirst), func(t *testing.T) {
			r := newS1Race(t, tt.former)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			store, users := workspacepg.New(r.pool), identitypg.New(r.pool)
			var accepted, deactivated <-chan error
			if tt.acceptFirst {
				accepted = run(func() error { return r.accept(ctx, gatedAccepter{store, g}) })
				held(t, ctx, g, accepted, "the acceptance")
				deactivated = run(func() error { return r.deactivateBob(ctx, users, store) })
			} else {
				deactivated = run(func() error { return r.deactivateBob(ctx, gatedSessions{users, g}, store) })
				held(t, ctx, g, deactivated, "the deactivation")
				accepted = run(func() error { return r.accept(ctx, store) })
			}
			pgtest.WaitForLockWaitOn(t, r.pool, "users", 5*time.Second)
			close(g.open)

			if err := result(t, ctx, accepted, "the acceptance"); !errors.Is(err, tt.accept) {
				t.Errorf("the acceptance = %v, want %v", err, tt.accept)
			}
			if err := result(t, ctx, deactivated, "the deactivation"); err != nil {
				t.Errorf("the deactivation = %v, want it done", err)
			}
			if got := r.standing(t); got != tt.want {
				t.Errorf("after both: %s; want %s", got, tt.want)
			}
		})
	}
}

// Interleaving 7 (b) and (c), S1's steps 5 and 6: the acceptance first
// holds bob's account row at its gate; the deactivation waits for it, then,
// once the acceptance has committed, stops at its own gate holding his
// account row, before its memberships' step. bob, acme's admin now, begins
// another write of acme, which holds acme's row at its gate: (b) his change
// of alice's role to member, holding acme FOR NO KEY UPDATE before its
// write; (c) his joining Web, holding acme FOR SHARE and his membership of
// it (3.6 convention 2, option E). The deactivation goes on and waits for
// acme's row: nothing else waits then, and neither side writes a row of
// workspaces, so only that wait satisfies the probe. The tables are read
// while both stand at their gates, neither having committed a write. Once
// bob's write commits:
//   - (b) bob is acme's only active admin, alice its member:
//     workspace.sole_admin, and no row of any table changes from what they
//     were then but alice's membership, which the change wrote;
//   - (c) the deactivation finds his membership of Web, made after its lock
//     of his account, and ends it with his membership of acme, as his;
//     acme keeps alice, its admin.
func TestAnAdmittedAdminsWriteAndHisDeactivation(t *testing.T) {
	for _, tt := range []struct {
		name  string
		write func(r s1Race, t *testing.T, ctx context.Context, g *gate) <-chan error
		err   error
		want  string
	}{
		{"(b) he makes alice a member", func(r s1Race, t *testing.T, ctx context.Context, g *gate) <-chan error {
			cascade := project.NewCascade(project.CascadeDeps{Pool: r.pool})
			return run(func() error {
				return changeRole(ctx, r.pool, r.bob, r.aliceIn, shared.RoleMember, gatedMembers{workspacepg.New(r.pool), g}, cascade)
			})
		}, workspacedomain.ErrSoleAdmin, "bob active; acme 20 active; invitation accepted deleted by bob; alice 15"},
		{"(c) he joins Web", func(r s1Race, t *testing.T, ctx context.Context, g *gate) <-chan error {
			route := newProjectRoute(t, r.pool, authorizerOn(r.pool), gatedShares{workspace.Provide(r.pool).WorkspaceMembers, g})
			return run(func() error {
				if _, rec := route.send(http.MethodPost, "/api/v0/projects/"+r.web.String()+"/join", r.bob, ""); rec.Code != http.StatusOK {
					return fmt.Errorf("bob's joining Web = %d %s", rec.Code, rec.Body)
				}
				return nil
			})
		}, nil, "bob deactivated; acme 20 ended by bob; Web 20 ended by bob; invitation accepted deleted by bob; alice 20"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := newS1Race(t, false)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			admitted, stopped, writing := newGate(), newGate(), newGate()
			store, users := workspacepg.New(r.pool), identitypg.New(r.pool)
			accepted := run(func() error { return r.accept(ctx, gatedAccepter{store, admitted}) })
			held(t, ctx, admitted, accepted, "the acceptance")
			deactivated := run(func() error { return r.deactivateBob(ctx, gatedSessions{users, stopped}, store) })
			pgtest.WaitForLockWaitOn(t, r.pool, "users", 5*time.Second)
			close(admitted.open)
			if err := result(t, ctx, accepted, "the acceptance"); err != nil {
				t.Fatalf("the acceptance = %v, want it done", err)
			}
			held(t, ctx, stopped, deactivated, "the deactivation")
			written := tt.write(r, t, ctx, writing)
			held(t, ctx, writing, written, "bob's write")
			// Every table but River's and workspace_members, and every
			// membership but alice's, which bob's change writes, as row_to_json
			// writes it, in id order.
			notMemberships := func(table string) bool { return riversOwn(table) || table == "public.workspace_members" }
			membershipsButAlices := func() string {
				t.Helper()
				var s string
				if err := r.pool.QueryRow(soon(t), `SELECT coalesce(string_agg(row_to_json(m)::text, E'\n' ORDER BY m.id), '')
					FROM workspace_members m WHERE m.id <> $1`, r.aliceIn).Scan(&s); err != nil {
					t.Fatal(err)
				}
				return s
			}
			before, memberships := tableRows(t, r.pool, notMemberships), membershipsButAlices()
			close(stopped.open)
			pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
			close(writing.open)

			if err := result(t, ctx, written, "bob's write"); err != nil {
				t.Fatalf("bob's write = %v, want it done", err)
			}
			if err := result(t, ctx, deactivated, "the deactivation"); !sameOutcome(err, tt.err) {
				t.Errorf("the deactivation = %v, want %v", err, tt.err)
			}
			if after := tableRows(t, r.pool, notMemberships); tt.err != nil && !maps.Equal(after, before) {
				t.Errorf("the tables after the refused deactivation changed:\n%v\nwant them as they were at the gates:\n%v", after, before)
			}
			if after := membershipsButAlices(); tt.err != nil && after != memberships {
				t.Errorf("the memberships but alice's after the refused deactivation:\n%s\nwant them as they were at the gates:\n%s", after,
					memberships)
			}
			if got := r.standing(t); got != tt.want {
				t.Errorf("after all: %s; want %s", got, tt.want)
			}
		})
	}
}
