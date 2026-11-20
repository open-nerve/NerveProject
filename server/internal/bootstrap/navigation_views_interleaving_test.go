package bootstrap

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The interleavings of the views' writes (M7 P3b spec 2.5, 2.6, 4), on
// issueWorld's real database, each write through its module's route as
// bootstrap wires it (issue_interleaving_test.go). A write of a view takes
// its workspace's row FOR SHARE, then, for a project's view, its project's
// FOR SHARE, then the view's row FOR UPDATE; a deletion of the project or
// of the workspace, a change of the project's toggle or of what it shows
// its guests, and a change of a member or a role take the project's or the
// workspace's row FOR NO KEY UPDATE, and another write of the view its
// row. The view's write, second, is proven to wait at its own statement
// for the row the first holds past its writes (viewWaits); once the first
// has committed, it reads again under its locks what decides it and
// decides again, and that decision is its answer.

// A write of a view that waits for a change of what decides it is decided
// under its locks, once the change has committed (M7 P3b spec 2.5, 2.6,
// 4): the decision under the lock wins. One issueWorld: acme, whose admin
// alice made Web, which shows its views, bob Web's member, walt acme's
// member and wanda its admin; Vault, acme's private project, which shows
// its views too, bob and walt its members, and adam its guest who is
// acme's admin (Grant.ProjectAdmin, adminGuest's case), who joins acme as
// its member first. Once the start-time runs have run, so that none
// queues on a row a write holds (03 §5), each group is a subtest:
// viewsWaitForADeletion, viewsWaitForTheToggle, viewsWaitForAMember and
// twoWritesOfAView. No work item is made, so no fan-out is pending.
func TestViewWritesInterleave(t *testing.T) {
	w := newIssueWorld(t)
	w.changeProject(t, w.web, `{"issue_views_view":true}`)
	vault := createdProjectFrom(t, w.contract, w.base, w.tokens["alice"], "acme", `{"name":"Vault","identifier":"VLT","network":0}`)
	w.changeProject(t, vault, `{"issue_views_view":true}`)
	for _, name := range []string{"bob", "walt"} {
		w.makeMember(t, name, vault, shared.RoleMember)
	}
	w.joinAcme(t, "adam", shared.RoleMember)
	w.makeMember(t, "adam", vault, shared.RoleGuest)
	w.promoteInAcme(t, "adam")
	waitForStartupRuns(t, w.pool)
	t.Run("a deletion", func(t *testing.T) { viewsWaitForADeletion(t, w) })
	t.Run("the toggle", func(t *testing.T) { viewsWaitForTheToggle(t, w) })
	t.Run("a member", func(t *testing.T) { viewsWaitForAMember(t, w, vault) })
	t.Run("another write of the view", func(t *testing.T) { twoWritesOfAView(t, w) })
}

// newNavigationRoute is the navigation module (navigation.New) on pool,
// with tx its writes' transactions, read-only snapshots of its own for its
// reads, the system's clock, the Authorizer on pool, and project's,
// workspace's and issue's ports converted, as bootstrap wires them.
func newNavigationRoute(t *testing.T, pool *pgxpool.Pool, tx shared.TxManager) moduleRoute {
	t.Helper()
	issues := issue.Provide(pool, nil)
	return newRoute(t, navigation.New(navigation.Deps{Pool: pool, Tx: tx,
		ReadTx: postgres.NewTxManager(pool, 2*time.Second, postgres.ReadOnlySnapshot()), Clock: clock.System{}, Authorizer: authorizerOn(pool),
		Projects:     navigationProjects{projects: project.Provide(pool, nil).NavigationProjects},
		Workspaces:   navigationWorkspaces{directory: workspace.Provide(pool, nil).WorkspaceDirectory},
		ListSettings: issues.ListSettings, WorkspaceListSettings: issues.WorkspaceListSettings}).Register)
}

// writesView is the account name's method on path, a view's or a
// project's or a workspace's views, with body, through the navigation
// module's route.
func writesView(name, method, path, body string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newNavigationRoute(t, w.pool, tx), name, method, path, body)
	}
}

// alicesProjectWrite is alice's method on path, with body, through the
// project module's route (aliceOnWeb): a project's deletion or change, or
// a change of a membership of it.
func alicesProjectWrite(method, path, body string) racingWrite {
	return aliceOnWeb(method, func(*testing.T, issueWorld) string { return path }, body)
}

// alicesWorkspaceWrite is alice's method on path, with body, through the
// workspace module's route: a workspace's deletion, or a change of a
// membership of it.
func alicesWorkspaceWrite(method, path, body string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newWorkspaceRoute(t, w.pool, tx), "alice", method, path, body)
	}
}

// viewWaits is serializeAtObserved with a write of a view second: once the
// first holds its row of table past its writes, the second waits for that
// row at statement, its own sqlc query, and meanwhile holds no lock on the
// row of view, when it names one: a view's row is its write's last lock
// (P3b spec 2.6).
func (w issueWorld) viewWaits(t *testing.T, table, statement string, view *uuid.UUID, first, second racingWrite, a, b *exchange) {
	t.Helper()
	w.serializeAtObserved(t, table, first, second, a, b, func(_, waiter *requestProbe) {
		sample, waiting := readRequestLock(t, pgtest.Soon(t), w.pool, waiter)
		if f := strings.Fields(sample.query); !waiting || len(f) < 3 || f[0] != "--" || f[1] != "name:" || f[2] != statement {
			t.Errorf("the view's write waits (%v) running %q; want it waiting at %s", waiting, sample.query, statement)
		}
		if view != nil && heldBy(t, w.pool, "SELECT 1 FROM issue_views WHERE id = $1 FOR UPDATE", *view) {
			t.Errorf("the view's write holds the view's row while it waits at %s", statement)
		}
	})
}

// refusedAsAbsent checks that e, the account name's write with body,
// answered 404 code, byte for byte as the same write on absent, a path
// that names nothing that ever was, answers (P3b spec 2.2, 2.5).
func (w issueWorld) refusedAsAbsent(t *testing.T, e exchange, name, absent, body, code string) {
	t.Helper()
	got := w.answered(t, e, http.StatusNotFound)
	none := newNavigationRoute(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second)).answerWithin(t, w.contract, e.req.Method, absent,
		w.ids[name], body, http.StatusNotFound)
	if got != none || refusalOf(t, []byte(got), false) != code {
		t.Errorf("%s %s = %s; want 404 %s, byte for byte as on %s: %s", e.req.Method, e.req.URL.Path, got, code, absent, none)
	}
}

// sameView tells whether a and b are the same view, each moment of them
// the same whatever its zone: a module's route on the test's pool, whose
// connections scan timestamptz in time.Local, answers its moments in that
// zone, the app in UTC (postgres.NewPool).
func sameView(a, b viewHTTP) bool {
	for _, v := range []*viewHTTP{&a, &b} {
		v.CreatedAt, v.UpdatedAt = v.CreatedAt.UTC(), v.UpdatedAt.UTC()
	}
	return reflect.DeepEqual(a, b)
}

// viewColumns is the row of the view id, deleted or not, every column of
// it, as JSON.
func viewColumns(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) string {
	t.Helper()
	var row string
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT to_jsonb(v)::text FROM issue_views AS v WHERE id = $1", id).Scan(&row); err != nil {
		t.Fatalf("the view %s: %v", id, err)
	}
	return row
}

// unchanged checks that a write refused under its lock left the view id's
// row as before, every column of it.
func unchanged(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, before string) {
	t.Helper()
	if after := viewColumns(t, pool, id); after != before {
		t.Errorf("the view %s is %s after the refused write; want it as before, %s", id, after, before)
	}
}

// unchangedButDeletedWith checks that a write refused under its lock left
// the view id's row as before, every column of it, or as before but
// deleted with its parent, the deleted row parent of the table parents, at
// its deletion's moment and by its deleter, as the parent's deletion job
// deletes it, which may have run since (P3b spec 2.7, deletedWith).
func unchangedButDeletedWith(t *testing.T, pool *pgxpool.Pool, id uuid.UUID, before, parents string, parent uuid.UUID) {
	t.Helper()
	var asBefore, asDeleted bool
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT to_jsonb(v) = $2::jsonb, to_jsonb(v) = $2::jsonb || jsonb_build_object('deleted_at', p.deleted_at,
			'updated_at', p.deleted_at, 'updated_by_id', p.updated_by_id)
		FROM issue_views AS v, `+parents+` AS p WHERE v.id = $1 AND p.id = $3 AND p.deleted_at IS NOT NULL`, id, before, parent).Scan(&asBefore,
		&asDeleted); err != nil {
		t.Fatalf("the view %s under the deleted %s %s: %v", id, parents, parent, err)
	}
	if !asBefore && !asDeleted {
		t.Errorf("the view %s is %s after the refused write; want it as before, %s, or so deleted with its parent", id, viewColumns(t, pool, id), before)
	}
}

// viewsNamed counts the views named name, deleted or not, of the project
// or the workspace parent.
func viewsNamed(t *testing.T, pool *pgxpool.Pool, parent uuid.UUID, name string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM issue_views WHERE (project_id = $1 OR workspace_id = $1) AND name = $2", parent,
		name).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
