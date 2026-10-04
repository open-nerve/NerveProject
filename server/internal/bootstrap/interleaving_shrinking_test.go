package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"uuid"

	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The interleavings 13, 14 and 15 of M3 design 9.3, and rule 2 under two
// deactivations at once: a deactivation, `nerve users deactivate` as
// bootstrap wires it (deactivating), against a growth of the account's set
// of memberships on the project side, through the project module as
// bootstrap wires it, or against another deactivation, on a real database,
// in both orders, as TestALeavingAndAnEndingOfAWorkspaceMembershipSerialize
// is: the second side waits for the workspace's row, and neither side
// writes a row of workspaces, so only that lock's wait satisfies the
// probe. Every wait has a deadline.

// deactivateBob is bob's deactivation on r's database (deactivating), over
// memberships.
func (r growthRace) deactivateBob(ctx context.Context, memberships workspaceapp.AllMembershipsEnder) error {
	_, err := deactivating(r.pool, identitypg.New(r.pool), memberships).ExecuteByEmail(ctx, "bob@example.com")
	return err
}

// lastWrittenByAlice stamps bob's membership of acme, his own, as last
// written by alice, as her change of his role would leave it: so that its
// ending as his shows the deactivation's write.
func (r growthRace) lastWrittenByAlice(t *testing.T) {
	t.Helper()
	stampWriter(t, r.pool, r.alice, 1, "workspace_members", "id = $2", r.bobIn)
}

// Interleavings 13 and 14: bob's deactivation and the project side's
// growth, alice's adding him to Web or his joining it, serialize on acme's
// row, in both orders, for a new membership of Web and for his ended one,
// which the growth restores (3.6 conventions 2 and 6). The test stamps his
// membership of acme as last written by alice first. The growth first: it
// holds acme and his membership of acme FOR SHARE, no stronger (lockOn),
// and waits at its gate, before it locks Web; the deactivation waits for
// acme's row, holding his account. Once the growth has committed, the
// deactivation, which finds his projects when its step over them runs,
// after its end of his membership of acme, finds Web and ends his
// membership of it, as his, at the moment it ends his membership of acme,
// read once it held acme: after the gate opened (3.3). The growth reads its
// time under its locks, after the gate opened too, so it is that one
// moment of both rows, not the gate, that tells the deactivation's write of
// Web from the growth's; both endings are his, though where he joins, his
// join wrote Web as his too, and only the moment shows the deactivation's
// write of it. The deactivation first: it holds acme FOR NO KEY UPDATE and
// his membership's row after its end; the growth waits for acme's row,
// then reads his membership ended: alice's adding is 422 at
// members[0].member_id, his joining 404; his membership of acme is ended
// as his, and an ended membership of Web stays as alice's removal left it.
// In either order no transaction holds Web while the second side waits,
// and no membership of alice's ends.
func TestADeactivationAndTheProjectSidesGrowthSerialize(t *testing.T) {
	contract := apitest.Load(t)
	for _, add := range []bool{true, false} {
		for _, ended := range []bool{false, true} {
			for _, growthFirst := range []bool{true, false} {
				t.Run(fmt.Sprintf("add %v, ended %v, growth first %v", add, ended, growthFirst), func(t *testing.T) {
					r := newGrowthRace(t, ended)
					before, enders := "15, Web none", "acme bob"
					if ended {
						before, enders = "15, Web 15 ended", "Web alice, acme bob"
					}
					if got := r.standing(t); got != before {
						t.Fatalf("before: %s, want %s", got, before)
					}
					r.lastWrittenByAlice(t)
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					g := newGate()
					var req *http.Request
					var rec *httptest.ResponseRecorder
					var grew, deactivated <-chan error
					if growthFirst {
						grow := r.growth(t, add, g)
						grew = run(func() error { req, rec = grow(); return nil })
						held(t, ctx, g, grew, "the growth")
						r.sharesAcme(t, "the growth")
						deactivated = run(func() error { return r.deactivateBob(ctx, workspacepg.New(r.pool)) })
					} else {
						deactivated = run(func() error { return r.deactivateBob(ctx, endedHoldingAll{workspacepg.New(r.pool), g}) })
						held(t, ctx, g, deactivated, "the deactivation")
						grow := r.growth(t, add, nil)
						grew = run(func() error { req, rec = grow(); return nil })
					}
					pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
					if !r.webFree(t) {
						t.Error("Web is held while the second side waits for acme's row; want it locked after that row")
					}
					opened := time.Now()
					close(g.open)

					deactivation := result(t, ctx, deactivated, "the deactivation")
					if err := result(t, ctx, grew, "the growth"); err != nil {
						t.Fatal(err)
					}
					contract.CheckResponse(t, req, rec.Result())
					want, grown := "15 ended, Web 15 ended", rec.Code == map[bool]int{false: http.StatusOK, true: http.StatusCreated}[add]
					if growthFirst {
						enders = "Web bob, acme bob"
					} else {
						want = map[bool]string{false: "15 ended, Web none", true: "15 ended, Web 15 ended"}[ended]
						grown = refusedAsNoMember(rec, add)
					}
					got, by, alices := r.standing(t), endersOf(t, r.pool, r.bob), endersOf(t, r.pool, r.alice)
					if deactivation != nil || !grown || got != want || by != enders || alices != "none" {
						t.Errorf("the deactivation = %v, the growth = %d %s, bob %s, his ended memberships by their last writers %s, alice's %s; "+
							"want the deactivation done, the growth %s, bob %s, %s, none of alice's", deactivation, rec.Code, rec.Body, got, by, alices,
							map[bool]string{true: "done", false: "refused"}[growthFirst], want, enders)
					}
					if growthFirst {
						made, written := r.membershipTimes(t)
						acme, web := r.lastWritten(t, "Web")
						if written.Before(made) || written.Before(opened) || web != acme {
							t.Errorf("bob's membership of Web made at %v, ended at %v, the gate opened at %v; written %s, his membership of "+
								"acme %s; want Web's ended at the moment the deactivation ended acme's, read once it held acme", made, written,
								opened, web, acme)
						}
					}
				})
			}
		}
	}
}

// Interleaving 15: bob's deactivation and alice's creation of Ops with him
// as its lead serialize on acme's row, in both orders, once the test has
// read him acme's member with no membership of Web, and stamped his
// membership of acme as last written by alice. The creation first: it
// holds acme and his membership of acme FOR SHARE, no stronger (lockOn),
// at its gate; the deactivation waits for acme's row. Once the creation
// has committed, the deactivation finds Ops among his projects and ends his
// membership of it, which alice's creation wrote, as his, with his
// membership of acme. The deactivation first: the creation waits for
// acme's row, then reads his membership ended, as his: 422 at
// project_lead_id, and no project is named Ops, where the creation first
// leaves one. In either order no membership of alice's ends.
func TestADeactivationAndACreationHeLeadsSerialize(t *testing.T) {
	contract := apitest.Load(t)
	for _, creationFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("creation first %v", creationFirst), func(t *testing.T) {
			r := newGrowthRace(t, false)
			if got := r.standing(t); got != "15, Web none" {
				t.Fatalf("before: %s, want 15, Web none", got)
			}
			r.lastWrittenByAlice(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			var req *http.Request
			var rec *httptest.ResponseRecorder
			create := func(gate *gate) func() error {
				route := newProjectRoute(t, r.pool, r.authorizer(), gatedShares{workspace.Provide(r.pool).WorkspaceMembers, gate})
				return func() error {
					req, rec = route.send(http.MethodPost, "/api/v0/workspaces/acme/projects", r.alice,
						`{"name":"Ops","identifier":"OPS","project_lead_id":"`+r.bob.String()+`"}`)
					return nil
				}
			}
			var created, deactivated <-chan error
			if creationFirst {
				created = run(create(g))
				held(t, ctx, g, created, "the creation")
				r.sharesAcme(t, "the creation")
				deactivated = run(func() error { return r.deactivateBob(ctx, workspacepg.New(r.pool)) })
			} else {
				deactivated = run(func() error { return r.deactivateBob(ctx, endedHoldingAll{workspacepg.New(r.pool), g}) })
				held(t, ctx, g, deactivated, "the deactivation")
				created = run(create(nil))
			}
			pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
			close(g.open)

			deactivation := result(t, ctx, deactivated, "the deactivation")
			if err := result(t, ctx, created, "the creation"); err != nil {
				t.Fatal(err)
			}
			contract.CheckResponse(t, req, rec.Result())
			done, by, alices := rec.Code == http.StatusCreated, endersOf(t, r.pool, r.bob), endersOf(t, r.pool, r.alice)
			if want := map[bool]string{true: "Ops bob, acme bob", false: "acme bob"}[creationFirst]; creationFirst != done || deactivation != nil ||
				by != want || alices != "none" {
				t.Errorf("the deactivation = %v, the creation = %d %s, bob's ended memberships by their last writers %s, alice's %s; want the "+
					"deactivation done, the creation %s, %s, none of alice's", deactivation, rec.Code, rec.Body, by, alices, map[bool]string{true: "done",
					false: "refused at project_lead_id, and no Ops"}[creationFirst], want)
			}
			if !creationFirst && !refusedAt(rec, "project_lead_id") {
				t.Errorf("the creation = %d %s, want 422 at project_lead_id", rec.Code, rec.Body)
			}
			var ops int
			if err := r.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM projects WHERE name = 'Ops'").Scan(&ops); err != nil ||
				ops != map[bool]int{true: 1, false: 0}[creationFirst] {
				t.Errorf("%d projects named Ops (%v); want %d", ops, err, map[bool]int{true: 1, false: 0}[creationFirst])
			}
		})
	}
}

// soleAdminCheckedHolding stops a deactivation between its check of rule 2
// and its writes: once it has asked whether the account is the only active
// admin of one of its workspaces, holding the account and every workspace
// of it FOR NO KEY UPDATE.
type soleAdminCheckedHolding struct {
	*workspacepg.Store
	gate *gate
}

func (m soleAdminCheckedHolding) SoleAdmin(ctx context.Context, workspaceIDs []uuid.UUID, userID uuid.UUID) (bool, error) {
	sole, err := m.Store.SoleAdmin(ctx, workspaceIDs, userID)
	if err != nil {
		return false, err
	}
	return sole, m.gate.wait(ctx)
}

// The world's standing (worldStanding) with alice's or gina's memberships
// ended.
const (
	aliceDeactivated = "acme: bob 15, carol 15, dave 15, erin 5, gina 20; beta: bob 20, carol 15; Lab: bob 20, carol 15; " +
		"Ops: bob 15, carol 20; Web: bob 20, carol 15, dave 20, erin 5, gina 15"
	ginaDeactivated = "acme: alice 20, bob 15, carol 15, dave 15, erin 5; beta: bob 20, carol 15; Lab: bob 20, carol 15; " +
		"Ops: alice 20, bob 15, carol 20; Web: alice 20, bob 20, carol 15, dave 20, erin 5"
)

// Rule 2 under two deactivations at once (M3 design 3.7, 3.6 convention
// 6): acme's two admins, alice and gina, deactivated at once; bob, carol,
// dave and erin are its other active members. The first holds her account
// and acme FOR NO KEY UPDATE and waits at her gate: between her check of
// rule 2 and her writes; or once her membership has ended. The second
// waits for acme's row. Once the first has committed, the second finds
// herself acme's only active admin, with other active members:
// workspace.sole_admin, and her account is active still. acme keeps an
// admin; each membership of the first ended, of acme and of its projects,
// is hers: the test stamped both admins' memberships as last written by
// dave first, alice's creations of acme, Web and Ops having written hers
// as hers. Held between her check and her writes, the first shows that the
// check holds the lock her writes do: a deactivation that let acme go
// between the two would leave the second nothing to wait for, and both
// would end.
func TestTwoAdminsDeactivatedAtOnceLeaveAnAdmin(t *testing.T) {
	for _, at := range []struct {
		name    string
		holding func(store *workspacepg.Store, g *gate) workspaceapp.AllMembershipsEnder
	}{
		{"at her check", func(store *workspacepg.Store, g *gate) workspaceapp.AllMembershipsEnder {
			return soleAdminCheckedHolding{store, g}
		}},
		{"past her membership's end", func(store *workspacepg.Store, g *gate) workspaceapp.AllMembershipsEnder {
			return endedHoldingAll{store, g}
		}},
	} {
		for _, aliceFirst := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s, alice first %v", at.name, aliceFirst), func(t *testing.T) {
				w := newMemberWorld(t)
				admins := []uuid.UUID{w.ids["alice"], w.ids["gina"]}
				stampWriter(t, w.pool, w.ids["dave"], 2, "workspace_members", "member_id = ANY ($2::uuid[])", admins)
				stampWriter(t, w.pool, w.ids["dave"], 3, "project_members", "member_id = ANY ($2::uuid[])", admins)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				first, second, want := "alice", "gina", aliceDeactivated
				if !aliceFirst {
					first, second, want = second, first, ginaDeactivated
				}
				deactivate := func(name string, memberships workspaceapp.AllMembershipsEnder) func() error {
					return func() error {
						_, err := deactivating(w.pool, identitypg.New(w.pool), memberships).ExecuteByEmail(ctx, name+"@example.com")
						return err
					}
				}
				g := newGate()
				ended := run(deactivate(first, at.holding(workspacepg.New(w.pool), g)))
				held(t, ctx, g, ended, "the first deactivation")
				refused := run(deactivate(second, workspacepg.New(w.pool)))
				pgtest.WaitForLockWaitOn(t, w.pool, "workspaces", 5*time.Second)
				if got := w.standing(t); got != worldStanding {
					t.Errorf("while the first holds acme: %s; want %s", got, worldStanding)
				}
				close(g.open)

				if err := result(t, ctx, ended, "the first deactivation"); err != nil {
					t.Errorf("the first deactivation = %v, want it done", err)
				}
				if err := result(t, ctx, refused, "the second deactivation"); !sameOutcome(err, workspacedomain.ErrSoleAdmin) {
					t.Errorf("the second deactivation = %v, want workspace.sole_admin", err)
				}
				var active bool
				if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT is_active FROM users WHERE id = $1", w.ids[second]).Scan(&active); err != nil || !active {
					t.Errorf("%s's account active %v (%v); want it active still", second, active, err)
				}
				enders := map[string]string{"alice": "Ops alice, Web alice, acme alice", "gina": "Web gina, acme gina"}[first]
				if got := endersOf(t, w.pool, w.ids[first]); got != enders {
					t.Errorf("%s's ended memberships by their last writers: %s; want %s", first, got, enders)
				}
				if got := w.standing(t); got != want {
					t.Errorf("after both: %s; want %s", got, want)
				}
			})
		}
	}
}
