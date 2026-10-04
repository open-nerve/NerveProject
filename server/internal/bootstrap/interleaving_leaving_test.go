package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Interleaving 1, the project's side, and a leaving of a project against
// P5a's ending of a workspace membership (M3 design 9.3, 3.7 rules 1 and
// 2), on memberWorld's real database, in both orders: project's
// LeaveProject over Locks, and workspace's removal with project's cascade,
// each with the Authorizer as bootstrap wires them, and gated as
// interleaving_endings_test.go gates. Every wait has a deadline.

// projectOtherFoundHolding stops a leaving of a project between its check
// and its ending: once it has asked whether the project has another admin
// and holds its locks, the workspace FOR SHARE and the project FOR NO KEY
// UPDATE, before its write of the membership.
type projectOtherFoundHolding struct {
	*projectpg.Store
	gate *gate
}

func (m projectOtherFoundHolding) EndMember(ctx context.Context, projectID, userID, by uuid.UUID, now time.Time) error {
	if err := m.gate.wait(ctx); err != nil {
		return err
	}
	return m.Store.EndMember(ctx, projectID, userID, by, now)
}

// projectEndedHolding stops a leaving of a project after its write of the
// membership, holding the workspace, the project and the membership's row.
type projectEndedHolding struct {
	*projectpg.Store
	gate *gate
}

func (m projectEndedHolding) EndMember(ctx context.Context, projectID, userID, by uuid.UUID, now time.Time) error {
	if err := m.Store.EndMember(ctx, projectID, userID, by, now); err != nil {
		return err
	}
	return m.gate.wait(ctx)
}

// leaving is how a leaving of a project is held at its gate.
type leaving struct {
	name    string
	holding func(store *projectpg.Store, g *gate) projectapp.MemberLeaver
}

var leavings = []leaving{
	{"before the membership's end", func(store *projectpg.Store, g *gate) projectapp.MemberLeaver {
		return projectOtherFoundHolding{store, g}
	}},
	{"past the membership's end", func(store *projectpg.Store, g *gate) projectapp.MemberLeaver {
		return projectEndedHolding{store, g}
	}},
}

// leaveProject is name's leaving of project, through project's use case
// over leavers, with Locks over project's store, workspace's directory and
// members' locks and the Authorizer, as project.New wires them.
func (w memberWorld) leaveProject(ctx context.Context, name string, project uuid.UUID, leavers projectapp.MemberLeaver) error {
	provided := workspace.Provide(w.pool)
	locks := projectapp.NewLocks(projectpg.New(w.pool), projectWorkspaces{directory: provided.WorkspaceDirectory}, provided.WorkspaceMembers,
		authorizerOn(w.pool))
	return projectapp.NewLeaveProject(locks, leavers, postgres.NewTxManager(w.pool, 2*time.Second), clock.System{}).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: w.ids[name]}), project)
}

// removeFromAcme is by's removal of the membership of acme id, over
// members, with project's cascade, identity's profiles and the Authorizer
// as bootstrap wires them. It takes no test, so it runs on any goroutine.
func (w memberWorld) removeFromAcme(ctx context.Context, by string, id uuid.UUID, members workspaceapp.MemberRemover) error {
	return workspaceapp.NewRemoveWorkspaceMember(members, workspaceProfiles{profiles: identity.Provide(w.pool).PublicProfiles},
		project.NewCascade(project.CascadeDeps{Pool: w.pool}), authorizerOn(w.pool), postgres.NewTxManager(w.pool, 2*time.Second), clock.System{}).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: w.ids[by]}), id)
}

// endedBy is the name of who ended name's membership of project last, or
// "active" while it is.
func (w memberWorld) endedBy(t *testing.T, project uuid.UUID, name string) string {
	t.Helper()
	var by string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT CASE WHEN m.is_active THEN 'active' ELSE split_part(u.email, '@', 1) END
		FROM project_members m JOIN users u ON u.id = m.updated_by_id WHERE m.project_id = $1 AND m.member_id = $2`, project, w.ids[name]).
		Scan(&by); err != nil {
		t.Fatal(err)
	}
	return by
}

// The world's standing (worldStanding) with carol's memberships of acme and
// of its projects ended, and with alice's or carol's membership of Ops
// ended.
const (
	carolOutOfAcme = "acme: alice 20, bob 15, dave 15, erin 5, gina 20; beta: bob 20, carol 15; Lab: bob 20, carol 15; " +
		"Ops: alice 20, bob 15; Web: alice 20, bob 20, dave 20, erin 5, gina 15"
	aliceOutOfOps = "acme: alice 20, bob 15, carol 15, dave 15, erin 5, gina 20; beta: bob 20, carol 15; Lab: bob 20, carol 15; " +
		"Ops: bob 15, carol 20; Web: alice 20, bob 20, carol 15, dave 20, erin 5, gina 15"
	carolOutOfOps = "acme: alice 20, bob 15, carol 15, dave 15, erin 5, gina 20; beta: bob 20, carol 15; Lab: bob 20, carol 15; " +
		"Ops: alice 20, bob 15; Web: alice 20, bob 20, carol 15, dave 20, erin 5, gina 15"
)

// Interleaving 1, the project's side: Ops's two admins, alice and carol,
// leave it at once (M3 design 3.6, 3.7 rule 1); bob is its member. The
// first holds acme FOR SHARE and Ops FOR NO KEY UPDATE and waits at his
// gate: once he has found the other admin, before he ends anything; or once
// he has ended his membership. The second shares acme and waits for Ops's
// row: neither writes a row of projects, so only that lock's wait satisfies
// the probe. Once the first has committed, the second finds no other active
// admin: 409 project.sole_admin. The first one's membership is ended, the
// second's active: Ops keeps an admin. Held between his check and his
// ending, the first shows that he asks rule 1 under the lock his ending
// holds: a leaving that let Ops go between the two would leave the second
// nothing to wait for, and one that asked before its lock would let the
// second find him still active, so that both leave.
func TestTwoProjectAdminsLeavingLeaveAnAdmin(t *testing.T) {
	for _, l := range leavings {
		for _, aliceFirst := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s, alice first %v", l.name, aliceFirst), func(t *testing.T) {
				w := newMemberWorld(t)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				first, second, want := "alice", "carol", aliceOutOfOps
				if !aliceFirst {
					first, second, want = second, first, carolOutOfOps
				}
				g := newGate()
				left := run(func() error { return w.leaveProject(ctx, first, w.ops, l.holding(projectpg.New(w.pool), g)) })
				held(t, ctx, g, left, "the first leaving")
				refused := run(func() error { return w.leaveProject(ctx, second, w.ops, projectpg.New(w.pool)) })
				pgtest.WaitForLockWaitOn(t, w.pool, "projects", 5*time.Second)
				if got := w.standing(t); got != worldStanding {
					t.Errorf("while the first holds Ops: %s; want %s", got, worldStanding)
				}
				close(g.open)

				if err := result(t, ctx, left, "the first leaving"); err != nil {
					t.Errorf("the first leaving = %v, want it done", err)
				}
				if err := result(t, ctx, refused, "the second leaving"); !sameOutcome(err, projectdomain.ErrSoleAdmin) {
					t.Errorf("the second leaving = %v, want 409 project.sole_admin as its first problem", err)
				}
				if got := w.standing(t); got != want {
					t.Errorf("after both: %s; want %s", got, want)
				}
			})
		}
	}
}

// A leaving of Ops and gina's removal of carol from acme, whose cascade
// ends carol's memberships of acme's projects (3.7 rule 2), serialize on
// acme's row in both orders, and never form a cycle: each takes acme's
// row first, the leaving FOR SHARE, the removal FOR NO KEY UPDATE, then
// Ops's (M3 design 3.6's lock table). The second waits for acme's row: no
// side writes a row of workspaces, so only that lock's wait satisfies the
// probe.
//   - carol leaves, the same member: the leaving first, held at its gate,
//     ends her membership of Ops, as hers; the removal then ends the others,
//     Web's. The removal first ends all of them, as gina's; the leaving then
//     finds carol no member of acme: 404 project.not_found.
//   - alice leaves, Ops's other admin: the leaving first leaves carol its
//     only admin, beside bob, its member; the removal, under Ops's lock,
//     sees it, and refuses: 409 project.sole_admin (rule 2). The removal
//     first ends carol's membership of Ops; the leaving, under Ops's lock,
//     finds no other active admin: 409 project.sole_admin (rule 1). Either
//     way Ops keeps an admin.
func TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize(t *testing.T) {
	for _, tt := range []struct {
		leaver                  string
		leaveFirst, removeFirst serialized // by which side came first
	}{
		{"carol", serialized{nil, nil, carolOutOfAcme, "carol", "gina"},
			serialized{projectdomain.ErrNotFound, nil, carolOutOfAcme, "gina", "gina"}},
		{"alice", serialized{nil, projectdomain.ErrSoleAdmin, aliceOutOfOps, "active", "active"},
			serialized{projectdomain.ErrSoleAdmin, nil, carolOutOfAcme, "gina", "gina"}},
	} {
		for _, at := range slices.Concat(leavings, []leaving{{name: "the removal first"}}) {
			t.Run(tt.leaver+" leaves, "+at.name, func(t *testing.T) {
				w := newMemberWorld(t)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				g := newGate()
				carol := w.acmeMembership(t, "carol")
				var left, removed <-chan error
				want := tt.leaveFirst
				if at.holding != nil {
					left = run(func() error { return w.leaveProject(ctx, tt.leaver, w.ops, at.holding(projectpg.New(w.pool), g)) })
					held(t, ctx, g, left, "the leaving")
					removed = run(func() error { return w.removeFromAcme(ctx, "gina", carol, workspacepg.New(w.pool)) })
				} else {
					want = tt.removeFirst
					removed = run(func() error { return w.removeFromAcme(ctx, "gina", carol, endedHolding{workspacepg.New(w.pool), g}) })
					held(t, ctx, g, removed, "the removal")
					left = run(func() error { return w.leaveProject(ctx, tt.leaver, w.ops, projectpg.New(w.pool)) })
				}
				pgtest.WaitForLockWaitOn(t, w.pool, "workspaces", 5*time.Second)
				if got := w.standing(t); got != worldStanding {
					t.Errorf("while the first holds acme: %s; want %s", got, worldStanding)
				}
				close(g.open)

				leave, removal := result(t, ctx, left, "the leaving"), result(t, ctx, removed, "the removal")
				if !sameOutcome(leave, want.leave) || !sameOutcome(removal, want.remove) {
					t.Errorf("the leaving = %v, the removal = %v; want %v, %v", leave, removal, want.leave, want.remove)
				}
				got, ops, web := w.standing(t), w.endedBy(t, w.ops, "carol"), w.endedBy(t, w.web, "carol")
				if got != want.standing || ops != want.ops || web != want.web {
					t.Errorf("after both: %s, carol's membership of Ops %s, of Web %s; want %s, %s, %s", got, ops, web, want.standing, want.ops,
						want.web)
				}
			})
		}
	}
}

// serialized is how a leaving of Ops and a removal from acme end: the
// leaving's and the removal's errors, the world after them, and who ended
// carol's memberships of Ops and of Web ("active" while each is). alice
// made both, so that their ender shows.
type serialized struct {
	leave, remove      error
	standing, ops, web string
}

// sameOutcome is whether err is want: nil for nil, else an error whose
// first problem in its chain, the one the API would answer, is want's: of
// its kind, with its code and its detail, so that a problem made anew by
// each call, as shared.Unauthenticated() is, compares too.
func sameOutcome(err, want error) bool {
	if want == nil {
		return err == nil
	}
	var first, problem *shared.Error
	return errors.As(err, &first) && errors.As(want, &problem) && first.Kind == problem.Kind && first.Code == problem.Code &&
		first.Detail == problem.Detail
}
