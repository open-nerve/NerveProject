package bootstrap

import (
	"context"
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
// acme's public project, of which alice is the admin; aliceIn and aliceWeb
// are her memberships of acme and of Web. When former, bob has a membership
// of acme, as a member, his by an acceptance, which alice's removal ended
// before she invited him again (3.8).
type s1Race struct {
	answerRace
	web, aliceIn, aliceWeb uuid.UUID
}

func newS1Race(t *testing.T, former bool) s1Race {
	t.Helper()
	r := s1Race{answerRace: newAnswerRace(t, shared.RoleAdmin), web: uuid.NewV7(), aliceWeb: uuid.NewV7()}
	ctx, before := pgtest.Soon(t), time.Now().Add(-time.Hour)
	r.aliceIn = queryIDs(t, r.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", r.acme, r.alice)[0]
	projects := projectpg.New(r.pool)
	if err := projects.CreateProject(ctx, projectapp.ProjectRow{ID: r.web, WorkspaceID: r.acme, Name: "Web", Identifier: "WEB",
		Network: projectdomain.NetworkPublic, Timezone: "UTC", CreatedBy: r.alice, Now: before}); err != nil {
		t.Fatal(err)
	}
	if err := projects.CreateMember(ctx, projectapp.MemberRow{ID: r.aliceWeb, WorkspaceID: r.acme, ProjectID: r.web, MemberID: r.alice,
		Role: shared.RoleAdmin, CreatedBy: r.alice, Now: before}); err != nil {
		t.Fatal(err)
	}
	if former {
		workspaces := workspacepg.New(r.pool)
		if err := workspaces.CreateMember(ctx, workspaceapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: r.acme, MemberID: r.bob,
			Role: shared.RoleMember, CreatedBy: r.bob, Now: before}); err != nil {
			t.Fatal(err)
		}
		if err := workspaces.EndMember(ctx, r.acme, r.bob, r.alice, before); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

// standing is bob's account, active or deactivated; each of his
// memberships of acme, in the order they were made, then of Web; the
// invitation, accepted or unanswered, and who deleted it when it is
// deleted; and alice's memberships of acme and of Web. A membership is its
// place and its role, then "active", or "ended by" its last writer when it
// has ended; a membership of Web that ended at the moment one of his of
// acme did, "ended with acme": there the deactivation ended both, and the
// writer would show no more than his joining did, which wrote it as his.
func (r s1Race) standing(t *testing.T) string {
	t.Helper()
	var s string
	if err := r.pool.QueryRow(pgtest.Soon(t), `WITH membership AS (
			SELECT 1 AS ord, 'acme' AS place, m.id, m.member_id, m.created_at, m.role, m.is_active, m.updated_by_id, m.updated_at
			FROM workspace_members m WHERE m.workspace_id = $2
			UNION ALL
			SELECT 2, 'Web', m.id, m.member_id, m.created_at, m.role, m.is_active, m.updated_by_id, m.updated_at
			FROM project_members m WHERE m.project_id = $3),
		state AS (
			SELECT m.ord, m.id, m.member_id, m.created_at,
				m.place || ' ' || m.role || CASE WHEN m.is_active THEN ' active'
					WHEN m.ord = 2 AND EXISTS (SELECT 1 FROM membership a WHERE a.ord = 1 AND a.member_id = m.member_id AND NOT a.is_active
						AND a.updated_at = m.updated_at) THEN ' ended with acme'
					ELSE ' ended by ' || split_part(e.email, '@', 1) END AS text
			FROM membership m JOIN users e ON e.id = m.updated_by_id)
		SELECT concat_ws('; ', CASE WHEN u.is_active THEN 'bob active' ELSE 'bob deactivated' END,
			(SELECT string_agg(s.text, ', ' ORDER BY s.created_at) FROM state s WHERE s.member_id = u.id AND s.ord = 1),
			(SELECT string_agg(s.text, ', ' ORDER BY s.created_at) FROM state s WHERE s.member_id = u.id AND s.ord = 2),
			(SELECT 'invitation ' || CASE WHEN i.accepted THEN 'accepted' ELSE 'unanswered' END ||
				CASE WHEN i.deleted_at IS NULL THEN '' ELSE ' deleted by ' || split_part(e.email, '@', 1) END
				FROM workspace_member_invites i LEFT JOIN users e ON e.id = i.updated_by_id WHERE i.id = $4),
			(SELECT 'alice ' || string_agg(s.text, ', ' ORDER BY s.ord) FROM state s
				WHERE (s.ord = 1 AND s.id = $5) OR (s.ord = 2 AND s.id = $6)))
		FROM users u WHERE u.id = $1`, r.bob, r.acme, r.web, r.invitation.id, r.aliceIn, r.aliceWeb).Scan(&s); err != nil {
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
//     stops at a gate of its own, holding his account row, before its
//     memberships' step, while the test stamps the membership the
//     acceptance wrote as last written by alice; then it finds acme among
//     his workspaces and ends his membership of it, as his: he is no admin
//     acme lacks, alice being one.
//   - (d) the deactivation first holds his account row at its gate, before
//     its memberships' step; the acceptance waits for it, then reads his
//     account deactivated under its lock: 401, no membership. The
//     deactivation deleted the invitation, as his.
//   - (e) bob has a former membership of acme: the acceptance restores that
//     row rather than inserting one, and serializes on his account row as
//     an insertion does: (a) ends the restored row, his only one; (d) leaves
//     it as alice's removal did.
//
// In each, alice keeps her memberships of acme and of Web, active, as their
// admin. The probe sees the one wait on users that either order has.
func TestAcceptingAndDeactivating(t *testing.T) {
	for _, tt := range []struct {
		former, acceptFirst bool
		accept              error
		want                string
	}{
		{false, true, nil, "bob deactivated; acme 20 ended by bob; invitation accepted deleted by bob; alice acme 20 active, Web 20 active"},
		{false, false, shared.Unauthenticated(), "bob deactivated; invitation unanswered deleted by bob; alice acme 20 active, Web 20 active"},
		{true, true, nil, "bob deactivated; acme 20 ended by bob; invitation accepted deleted by bob; alice acme 20 active, Web 20 active"},
		{true, false, shared.Unauthenticated(),
			"bob deactivated; acme 15 ended by alice; invitation unanswered deleted by bob; alice acme 20 active, Web 20 active"},
	} {
		t.Run(fmt.Sprintf("former %v, the acceptance first %v", tt.former, tt.acceptFirst), func(t *testing.T) {
			r := newS1Race(t, tt.former)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g, stopped := newGate(), newGate()
			store, users := workspacepg.New(r.pool), identitypg.New(r.pool)
			var accepted, deactivated <-chan error
			if tt.acceptFirst {
				accepted = run(func() error { return r.accept(ctx, gatedAccepter{store, g}) })
				held(t, ctx, g, accepted, "the acceptance")
				deactivated = run(func() error { return r.deactivateBob(ctx, gatedSessions{users, stopped}, store) })
			} else {
				deactivated = run(func() error { return r.deactivateBob(ctx, gatedSessions{users, g}, store) })
				held(t, ctx, g, deactivated, "the deactivation")
				accepted = run(func() error { return r.accept(ctx, store) })
			}
			pgtest.WaitForLockWaitOn(t, r.pool, "users", 5*time.Second)
			close(g.open)

			if err := result(t, ctx, accepted, "the acceptance"); !sameOutcome(err, tt.accept) {
				t.Errorf("the acceptance = %v, want %v", err, tt.accept)
			}
			if tt.acceptFirst {
				held(t, ctx, stopped, deactivated, "the deactivation")
				r.bobsLastWrittenByAlice(t)
				close(stopped.open)
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
// it (3.6 convention 2, option E); before it, the test stamps his
// membership of acme, which the acceptance wrote, as last written by alice.
// The deactivation goes on and waits for acme's row: nothing else waits
// then, and neither side writes a row of workspaces, so only that wait
// satisfies the probe. The tables are read while both stand at their
// gates, neither having committed a write. Once bob's write commits:
//   - (b) bob is acme's only active admin, alice its member:
//     workspace.sole_admin, and no row of any table changes from what they
//     were then but alice's membership, which the change wrote;
//   - (c) the deactivation finds his membership of Web, made after its lock
//     of his account, and ends it at the moment it ends his membership of
//     acme, as his; acme and Web keep alice, their admin, her memberships
//     active.
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
		}, workspacedomain.ErrSoleAdmin, "bob active; acme 20 active; invitation accepted deleted by bob; alice acme 15 active, Web 20 active"},
		{"(c) he joins Web", func(r s1Race, t *testing.T, ctx context.Context, g *gate) <-chan error {
			route := newProjectRoute(t, r.pool, authorizerOn(r.pool), gatedShares{workspace.Provide(r.pool).WorkspaceMembers, g})
			return run(func() error {
				if _, rec := route.send(http.MethodPost, "/api/v0/projects/"+r.web.String()+"/join", r.bob, ""); rec.Code != http.StatusOK {
					return fmt.Errorf("bob's joining Web = %d %s", rec.Code, rec.Body)
				}
				return nil
			})
		}, nil, "bob deactivated; acme 20 ended by bob; Web 20 ended with acme; invitation accepted deleted by bob; alice acme 20 active, " +
			"Web 20 active"},
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
			r.bobsLastWrittenByAlice(t)
			written := tt.write(r, t, ctx, writing)
			held(t, ctx, writing, written, "bob's write")
			// Every table but River's and workspace_members, and every
			// membership but alice's, which bob's change writes, as row_to_json
			// writes it, in id order.
			notMemberships := func(table string) bool { return riversOwn(table) || table == "public.workspace_members" }
			membershipsButAlices := func() string {
				t.Helper()
				var s string
				if err := r.pool.QueryRow(pgtest.Soon(t), `SELECT coalesce(string_agg(row_to_json(m)::text, E'\n' ORDER BY m.id), '')
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
