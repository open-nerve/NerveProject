package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The interleavings 1 (its workspace side) and 4 of M3 design 9.3, on a
// real database, in both orders: an ending of a workspace membership, a
// leaving or a removal, through workspace's use case with project's
// cascade, identity's profiles and the Authorizer as bootstrap wires them,
// against another leaving, or against the project side's growth through
// the project module behind the API. A gate inside the first side's
// transaction holds it open at a lock the second side needs;
// pgtest.WaitForLockWaitOn proves that the second side waits on that
// table's row before the gate opens. Every wait has a deadline.

// endedHolding stops an ending after its write of the membership, holding
// the workspace's row and the membership's, before the projects' step.
type endedHolding struct {
	*workspacepg.Store
	gate *gate
}

func (m endedHolding) EndMember(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error {
	if err := m.Store.EndMember(ctx, workspaceID, userID, by, now); err != nil {
		return err
	}
	return m.gate.wait(ctx)
}

// otherFoundHolding stops a leaving once it has asked whether the
// workspace has another admin, holding the workspace's row, before it ends
// anything.
type otherFoundHolding struct {
	*workspacepg.Store
	gate *gate
}

func (m otherFoundHolding) HasOtherAdmin(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	other, err := m.Store.HasOtherAdmin(ctx, workspaceID, userID)
	if err != nil {
		return false, err
	}
	return other, m.gate.wait(ctx)
}

// leave is user's leaving of acme, over workspaces, with project's cascade,
// identity's profiles and the Authorizer as bootstrap wires them.
func (r adminRace) leave(ctx context.Context, user uuid.UUID, workspaces workspaceapp.WorkspaceLeaver) error {
	return workspaceapp.NewLeaveWorkspace(workspaces, workspaceProfiles{profiles: identity.Provide(r.pool).PublicProfiles},
		project.NewCascade(project.CascadeDeps{Pool: r.pool}), authorizerOn(r.pool), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: user}), "acme")
}

// active is whether alice's and bob's memberships of acme are active.
func (r adminRace) active(t *testing.T) (alice, bob bool) {
	t.Helper()
	if err := r.pool.QueryRow(pgtest.Soon(t), "SELECT (SELECT is_active FROM workspace_members WHERE id = $1), "+
		"(SELECT is_active FROM workspace_members WHERE id = $2)", r.aliceIn, r.bobIn).Scan(&alice, &bob); err != nil {
		t.Fatal(err)
	}
	return alice, bob
}

// Interleaving 1, the workspace's side: acme's two admins, alice and bob,
// leave it at once (M3 design 3.6, 3.7 rule 1). The first holds acme FOR NO
// KEY UPDATE and waits at his gate: once he has found the other admin,
// before he ends anything; or once he has ended his membership. The second
// waits for acme's row. Once the first has committed, the second finds no
// other active admin: 409 workspace.sole_admin. The first one's membership
// is ended, the second's active: acme keeps an admin. Held at his check,
// the first shows that he asks rule 1 under the lock his ending holds: a
// leaving that let acme go between the two would let the second find him
// still active, and both would leave.
func TestTwoAdminsLeavingLeaveAnAdmin(t *testing.T) {
	for _, at := range []struct {
		name    string
		holding func(store *workspacepg.Store, g *gate) workspaceapp.WorkspaceLeaver
	}{
		{"at his check", func(store *workspacepg.Store, g *gate) workspaceapp.WorkspaceLeaver {
			return otherFoundHolding{store, g}
		}},
		{"past his membership's end", func(store *workspacepg.Store, g *gate) workspaceapp.WorkspaceLeaver {
			return endedHolding{store, g}
		}},
	} {
		for _, aliceFirst := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s, alice first %v", at.name, aliceFirst), func(t *testing.T) {
				r := newAdminRace(t)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				first, second := r.alice, r.bob
				if !aliceFirst {
					first, second = second, first
				}
				g := newGate()
				left := run(func() error { return r.leave(ctx, first, at.holding(workspacepg.New(r.pool), g)) })
				held(t, ctx, g, left, "the first leaving")
				refused := run(func() error { return r.leave(ctx, second, workspacepg.New(r.pool)) })
				pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
				if alice, bob := r.active(t); !alice || !bob {
					t.Errorf("while the first holds the lock: alice active %v, bob active %v; want both", alice, bob)
				}
				close(g.open)

				if err := result(t, ctx, left, "the first leaving"); err != nil {
					t.Errorf("the first leaving = %v, want it done", err)
				}
				if err := result(t, ctx, refused, "the second leaving"); !errors.Is(err, workspacedomain.ErrSoleAdmin) {
					t.Errorf("the second leaving = %v, want 409 workspace.sole_admin", err)
				}
				if alice, bob := r.active(t); alice != !aliceFirst || bob != aliceFirst {
					t.Errorf("alice active %v, bob active %v; want the first one's membership ended, the second's active", alice, bob)
				}
			})
		}
	}
}

// remove is alice's removal of bob from acme, over members, with project's
// cascade, identity's profiles and the Authorizer as bootstrap wires them,
// on the system's clock: its time is read when it reads it.
func (r growthRace) remove(ctx context.Context, members workspaceapp.MemberRemover) error {
	return workspaceapp.NewRemoveWorkspaceMember(members, workspaceProfiles{profiles: identity.Provide(r.pool).PublicProfiles},
		project.NewCascade(project.CascadeDeps{Pool: r.pool}), r.authorizer(), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: r.alice}), r.bobIn)
}

// lastWritten is when, and by whom, bob's membership of acme and his
// membership of the project named project were last written, each as
// "<updated_at> by <account id>": the two are the same when one
// transaction wrote both, at its one moment, as one account.
func (r growthRace) lastWritten(t *testing.T, project string) (acme, of string) {
	t.Helper()
	if err := r.pool.QueryRow(pgtest.Soon(t), `SELECT (SELECT updated_at::text || ' by ' || updated_by_id FROM workspace_members WHERE id = $1),
		(SELECT m.updated_at::text || ' by ' || m.updated_by_id FROM project_members m JOIN projects p ON p.id = m.project_id
		 WHERE p.name = $2 AND m.member_id = $3)`, r.bobIn, project, r.bob).Scan(&acme, &of); err != nil {
		t.Fatal(err)
	}
	return acme, of
}

// Interleaving 4: alice's removal of bob from acme and the project side's
// growth, alice's adding him to Web or his joining it, serialize on acme's
// row (M3 design 3.6 conventions 2, 3 and 6), in both orders, for a new
// membership of Web and for his ended one. The growth first: it holds acme
// and his membership of acme FOR SHARE, no stronger (lockOn), and waits at
// its gate before it locks Web; the removal waits for acme's row. Once the
// growth has committed, the removal's step over his projects, a statement
// run after its write of his membership, finds his membership of Web and
// ends it as alice, at the moment it wrote his membership of acme, read
// once it held acme: after the gate opened (3.3). In the joining's cells
// bob wrote it last before, so that alice's writing shows. The removal
// first: it holds acme FOR NO KEY UPDATE and his membership's row after its
// write; the growth waits for acme's row, then reads his membership ended:
// alice's adding him is refused as one who is no active member of acme
// (422 members[0].member_id not_allowed), his joining as one who does not
// see Web (404); his membership of Web is as it was. In either order, while
// the second side waits for acme's row, no transaction holds Web (FOR
// UPDATE NOWAIT): the growth locks Web after acme, and the removal, at its
// gate, has locked none of his projects, having no active membership of
// one. The order of the removal's own locks is
// TestEachLockOfAnEndingIsItsStrength's.
func TestARemovalAndTheProjectSidesGrowthSerialize(t *testing.T) {
	contract := apitest.Load(t)
	for _, add := range []bool{true, false} {
		for _, ended := range []bool{false, true} {
			for _, growthFirst := range []bool{true, false} {
				name := fmt.Sprintf("%s, ended %v, growth first %v", map[bool]string{true: "add", false: "join"}[add], ended, growthFirst)
				t.Run(name, func(t *testing.T) {
					r := newGrowthRace(t, ended)
					before := map[bool]string{false: "15, Web none", true: "15, Web 15 ended"}[ended]
					if got := r.standing(t); got != before {
						t.Fatalf("before: %s, want %s", got, before)
					}
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					g := newGate()
					var req *http.Request
					var rec *httptest.ResponseRecorder
					var grew, removed <-chan error
					if growthFirst {
						grow := r.growth(t, add, g)
						grew = run(func() error { req, rec = grow(); return nil })
						held(t, ctx, g, grew, "the growth")
						r.sharesAcme(t, "the growth")
						removed = run(func() error { return r.remove(ctx, workspacepg.New(r.pool)) })
					} else {
						removed = run(func() error { return r.remove(ctx, endedHolding{workspacepg.New(r.pool), g}) })
						held(t, ctx, g, removed, "the removal")
						grow := r.growth(t, add, nil)
						grew = run(func() error { req, rec = grow(); return nil })
					}
					pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
					if !r.webFree(t) {
						t.Error("Web is held while the second side waits for acme's row; want it locked after that row")
					}
					opened := time.Now()
					close(g.open)

					removal := result(t, ctx, removed, "the removal")
					if err := result(t, ctx, grew, "the growth"); err != nil {
						t.Fatal(err)
					}
					contract.CheckResponse(t, req, rec.Result())
					want, grown := "15 ended, Web 15 ended", rec.Code == map[bool]int{false: http.StatusOK, true: http.StatusCreated}[add]
					if !growthFirst {
						want, grown = "15 ended, "+before[len("15, "):], refusedAsNoMember(rec, add)
					}
					if got := r.standing(t); removal != nil || !grown || got != want {
						t.Errorf("the removal = %v, the growth = %d %s, bob %s; want the removal done, the growth %s, bob %s", removal, rec.Code,
							rec.Body, got, map[bool]string{true: "done", false: "refused"}[growthFirst], want)
					}
					if growthFirst {
						made, written := r.membershipTimes(t)
						acme, web := r.lastWritten(t, "Web")
						if written.Before(made) || written.Before(opened) || web != acme {
							t.Errorf("bob's membership of Web made at %v, last written at %v, the gate opened at %v; written %s, his membership of "+
								"acme %s; want Web's ended last by the removal, as it wrote acme's, at a time read once it held acme", made, written,
								opened, web, acme)
						}
					}
				})
			}
		}
	}
}

// refusedAsNoMember reports whether rec is the growth's refusal of bob as
// no active member of acme: alice's adding him, 422 members[0].member_id
// not_allowed alone; his joining, 404 project.not_found.
func refusedAsNoMember(rec *httptest.ResponseRecorder, add bool) bool {
	return refusedAt(rec, map[bool]string{true: "members[0].member_id"}[add])
}

// refusedAt reports whether rec is the project side's refusal of bob: with
// field empty, 404 project.not_found, the answer to one who does not see
// the project; else 422 validation_failed, not_allowed at field alone.
func refusedAt(rec *httptest.ResponseRecorder, field string) bool {
	var problem struct {
		Code   string `json:"code"`
		Errors []struct {
			Field string `json:"field"`
			Code  string `json:"code"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		return false
	}
	if field == "" {
		return rec.Code == http.StatusNotFound && problem.Code == "project.not_found"
	}
	return rec.Code == http.StatusUnprocessableEntity && problem.Code == "validation_failed" && len(problem.Errors) == 1 &&
		problem.Errors[0].Field == field && problem.Errors[0].Code == "not_allowed"
}

// sharesAcme checks that side, at its gate, holds acme's row and bob's
// membership of acme FOR SHARE, no stronger (lockOn; M3 design 3.6
// conventions 2 and 3).
func (r growthRace) sharesAcme(t *testing.T, side string) {
	t.Helper()
	if got, want := "acme "+lockOn(t, r.pool, "workspaces WHERE slug = 'acme'")+", his membership "+lockOn(t, r.pool,
		"workspace_members WHERE id = $1", r.bobIn), "acme FOR SHARE, his membership FOR SHARE"; got != want {
		t.Errorf("%s at its gate holds %s; want %s", side, got, want)
	}
}
