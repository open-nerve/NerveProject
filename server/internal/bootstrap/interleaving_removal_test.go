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

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	workspacedomain "github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The interleavings 5 and 6 of M3 design 9.3, on a real database, in both
// orders: a removal against the removed member's own write, his creating a
// project, through the project module behind the API, or his deleting the
// workspace, through workspace's use case; each with the Authorizer as
// bootstrap wires it, and gated as interleaving_endings_test.go gates.

// opsOf is bob's standing in Ops, the project his creation makes: "no
// Ops" when there is none, else his membership's role, "ended" when it
// is.
func (r growthRace) opsOf(t *testing.T) string {
	t.Helper()
	var s string
	if err := r.pool.QueryRow(context.Background(), `SELECT coalesce((SELECT 'Ops ' || m.role || CASE WHEN m.is_active THEN '' ELSE ' ended' END
		FROM projects p JOIN project_members m ON m.project_id = p.id WHERE p.name = 'Ops' AND m.member_id = $1), 'no Ops')`, r.bob).
		Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// opsWritten is when bob's membership of Ops was last written.
func (r growthRace) opsWritten(t *testing.T) time.Time {
	t.Helper()
	var written time.Time
	if err := r.pool.QueryRow(context.Background(), `SELECT m.updated_at FROM projects p JOIN project_members m ON m.project_id = p.id
		WHERE p.name = 'Ops' AND m.member_id = $1`, r.bob).Scan(&written); err != nil {
		t.Fatal(err)
	}
	return written
}

// Interleaving 5: alice's removal of bob from acme and his creating the
// project Ops in it serialize on acme's row (M3 design 3.6 conventions 2, 3
// and 6). The creation first: it holds acme FOR SHARE and his membership of
// acme FOR SHARE (lockOn), and waits at its gate before it inserts Ops;
// the removal waits for acme's row. Once the creation has committed, 201,
// the removal's step over his projects finds his membership of Ops, Ops's
// admin and only member, which he wrote, and ends it as alice, at the
// moment it wrote his membership of acme, read once it held acme. The
// removal first: it holds acme and his ended membership's row; the
// creation waits for acme's row, then decides once he is no member: 404
// workspace.not_found, and there is no Ops.
func TestARemovalAndTheRemovedMembersProjectSerialize(t *testing.T) {
	contract := apitest.Load(t)
	for _, creationFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("creation first %v", creationFirst), func(t *testing.T) {
			r := newGrowthRace(t, false)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			var req *http.Request
			var rec *httptest.ResponseRecorder
			create := func(gated *gate) func() error {
				route := newProjectRoute(t, r.pool, r.authorizer(), gatedShares{workspace.Provide(r.pool).WorkspaceMembers, gated})
				return func() error {
					req, rec = route.send(http.MethodPost, "/api/v0/workspaces/acme/projects", r.bob, `{"name":"Ops","identifier":"OPS"}`)
					return nil
				}
			}
			var created, removed <-chan error
			if creationFirst {
				created = run(create(g))
				held(t, ctx, g, created, "the creation")
				if got, want := "acme "+lockOn(t, r.pool, "workspaces WHERE slug = 'acme'")+", his membership "+lockOn(t, r.pool,
					"workspace_members WHERE id = $1", r.bobIn), "acme FOR SHARE, his membership FOR SHARE"; got != want {
					t.Errorf("the creation at its gate holds %s; want %s", got, want)
				}
				removed = run(func() error { return r.remove(ctx, workspacepg.New(r.pool)) })
			} else {
				removed = run(func() error { return r.remove(ctx, endedHolding{workspacepg.New(r.pool), g}) })
				held(t, ctx, g, removed, "the removal")
				created = run(create(nil))
			}
			pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
			opened := time.Now()
			close(g.open)

			removal := result(t, ctx, removed, "the removal")
			if err := result(t, ctx, created, "the creation"); err != nil {
				t.Fatal(err)
			}
			contract.CheckResponse(t, req, rec.Result())
			var problem struct {
				Code string `json:"code"`
			}
			_ = json.Unmarshal(rec.Body.Bytes(), &problem) // a 201 has no code
			want, answered := "15 ended, Web none, Ops 20 ended", rec.Code == http.StatusCreated
			if !creationFirst {
				want, answered = "15 ended, Web none, no Ops", rec.Code == http.StatusNotFound && problem.Code == "workspace.not_found"
			}
			if got := r.standing(t) + ", " + r.opsOf(t); removal != nil || !answered || got != want {
				t.Errorf("the removal = %v, the creation = %d %s, bob %s; want the removal done, the creation %s, bob %s", removal, rec.Code,
					rec.Body, got, map[bool]string{true: "201", false: "404 workspace.not_found"}[creationFirst], want)
			}
			if creationFirst {
				acme, ops := r.lastWritten(t, "Ops")
				if written := r.opsWritten(t); written.Before(opened) || ops != acme {
					t.Errorf("bob's membership of Ops last written at %v, the gate opened at %v; written %s, his membership of acme %s; want it "+
						"ended by the removal, as it wrote acme's, at a time read once it held acme", written, opened, ops, acme)
				}
			}
		})
	}
}

// removeAlice is bob's removal of alice from acme, over members, with
// project's cascade, identity's profiles and the Authorizer as bootstrap
// wires them.
func (r adminRace) removeAlice(ctx context.Context, members workspaceapp.MemberRemover) error {
	return workspaceapp.NewRemoveWorkspaceMember(members, workspaceProfiles{profiles: identity.Provide(r.pool).PublicProfiles},
		project.NewCascade(project.CascadeDeps{Pool: r.pool}), authorizerOn(r.pool), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: r.bob}), r.aliceIn)
}

// acmeAndAlice is whether acme is deleted, then alice's membership of it:
// "active", "ended" or "deleted".
func (r adminRace) acmeAndAlice(t *testing.T) string {
	t.Helper()
	var s string
	if err := r.pool.QueryRow(pgtest.Soon(t), `SELECT CASE WHEN w.deleted_at IS NULL THEN 'acme' ELSE 'acme deleted' END || ', alice ' ||
		CASE WHEN m.deleted_at IS NOT NULL THEN 'deleted' WHEN m.is_active THEN 'active' ELSE 'ended' END
		FROM workspaces w JOIN workspace_members m ON m.workspace_id = w.id WHERE m.id = $1`, r.aliceIn).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// waitsAtAWrite reports whether a backend that waits for a lock holds
// RowExclusiveLock on acme's table: an UPDATE of it takes that lock before
// it waits for a row; a locking read takes RowShareLock.
func (r adminRace) waitsAtAWrite(t *testing.T) bool {
	t.Helper()
	var written bool
	if err := r.pool.QueryRow(pgtest.Soon(t), `SELECT EXISTS (SELECT 1 FROM pg_stat_activity a JOIN pg_locks l ON l.pid = a.pid
		WHERE a.datname = current_database() AND a.wait_event_type = 'Lock' AND l.locktype = 'relation'
			AND l.relation = 'workspaces'::regclass AND l.mode = 'RowExclusiveLock' AND l.granted)`).Scan(&written); err != nil {
		t.Fatal(err)
	}
	return written
}

// Interleaving 6: bob's removal of alice, acme's other admin, and her
// deleting acme serialize on acme's row (M3 design 3.6 convention 2). The
// deletion first: it holds acme FOR NO KEY UPDATE after its decision
// (gatedDeleter); the removal, which has read her membership, waits for
// acme's row, then finds acme deleted: 404 workspace.member_not_found;
// that its lock then takes no row is the store's
// TestTheWorkspaceLocksSkipAWorkspaceDeletedWhileTheyWait. Her membership
// is deleted with acme. The removal first: it holds acme and her ended
// membership's row; the deletion waits for acme's row at its lock, then
// decides once she is no member: 404 workspace.not_found. acme stays, her
// membership ended. In either order the second side waits at a lock of
// acme's row, not at a write of acme's table (waitsAtAWrite): a deletion
// that decided without its lock would wait for acme's row at its write of
// it, which WaitForLockWaitOn counts too.
func TestARemovalAndTheRemovedAdminsDeletionSerialize(t *testing.T) {
	for _, deletionFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("deletion first %v", deletionFirst), func(t *testing.T) {
			r := newAdminRace(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			var deleted, removed <-chan error
			if deletionFirst {
				deleted = run(func() error { return r.deleteAcme(ctx, gatedDeleter{workspacepg.New(r.pool), g}) })
				held(t, ctx, g, deleted, "the deletion")
				removed = run(func() error { return r.removeAlice(ctx, workspacepg.New(r.pool)) })
			} else {
				removed = run(func() error { return r.removeAlice(ctx, endedHolding{workspacepg.New(r.pool), g}) })
				held(t, ctx, g, removed, "the removal")
				deleted = run(func() error { return r.deleteAcme(ctx, workspacepg.New(r.pool)) })
			}
			pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
			if r.waitsAtAWrite(t) {
				t.Error("the second side waits for acme's row at its write of acme; want it waiting at its lock of acme")
			}
			if got := r.acmeAndAlice(t); got != "acme, alice active" {
				t.Errorf("while the first holds acme: %s, want acme, alice active", got)
			}
			close(g.open)

			deletion, removal := result(t, ctx, deleted, "the deletion"), result(t, ctx, removed, "the removal")
			wantDeletion, wantRemoval, want := error(nil), error(workspacedomain.ErrMemberNotFound), "acme deleted, alice deleted"
			if !deletionFirst {
				wantDeletion, wantRemoval, want = workspacedomain.ErrNotFound, nil, "acme, alice ended"
			}
			if !errors.Is(deletion, wantDeletion) || !errors.Is(removal, wantRemoval) || r.acmeAndAlice(t) != want {
				t.Errorf("the deletion = %v, the removal = %v, %s; want %v, %v, %s", deletion, removal, r.acmeAndAlice(t), wantDeletion,
					wantRemoval, want)
			}
		})
	}
}
