package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A workspace's deletion against the writes on its projects, on a real
// database: the writes through the project module as bootstrap wires it,
// behind the API, the deletion through workspace's use case with project's
// cascade as bootstrap wires it, each on the system's clock. Every write on
// a project holds its workspace FOR SHARE (M3 design 3.6 convention 2); the
// deletion holds it FOR NO KEY UPDATE and reads its time once it does, so
// it waits for the writes in flight and writes one time, after theirs, into
// every row it deletes (3.3). Every wait has a deadline.

// deleteAcme is alice's deletion of acme over workspaces, with project's
// cascade and the Authorizer as bootstrap wires them, on the system's
// clock.
func (r race) deleteAcme(ctx context.Context, workspaces workspaceapp.WorkspaceDeleter) error {
	return workspaceapp.NewDeleteWorkspace(workspaces, project.New(project.Deps{Pool: r.pool}).Cascade(), authorizerOn(r.pool),
		postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}, slog.New(slog.DiscardHandler)).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: r.alice}), "acme")
}

// deletedWithAcme counts the rows statement reads, as deleted_at,
// updated_at, updated_by_id and created_at, and fails the test for each
// that acme's deletion did not write last: deleted and last written at the
// deletion's time, by alice, and not before it was made.
func (r growthRace) deletedWithAcme(t *testing.T, statement string, args ...any) int {
	t.Helper()
	ctx := context.Background()
	var acme *time.Time
	if err := r.pool.QueryRow(ctx, "SELECT deleted_at FROM workspaces WHERE slug = 'acme'").Scan(&acme); err != nil || acme == nil {
		t.Fatalf("acme's deletion time: %v, %v", acme, err)
	}
	rows, err := r.pool.Query(ctx, statement, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for ; rows.Next(); n++ {
		var deleted *time.Time
		var written, made time.Time
		var by uuid.UUID
		if err := rows.Scan(&deleted, &written, &by, &made); err != nil {
			t.Fatal(err)
		}
		if deleted == nil || !deleted.Equal(*acme) || !written.Equal(*acme) || by != r.alice || deleted.Before(made) {
			t.Errorf("row %d: deleted at %v, written last at %v by %v, made at %v; want deleted and written last at acme's deletion's time %v, "+
				"by alice, not before it was made", n, deleted, written, by, made, *acme)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return n
}

// A workspace's deletion waits for the writes on its projects in flight and
// deletes their rows at its one time, after theirs (pre-flight L1): bob,
// Web's admin, changes Web; or bob, acme's member, joins it, which makes
// his membership of it and his display settings in it. His write waits
// after its decision, holding acme FOR SHARE; alice's deletion of acme
// waits for acme's row. Once his write commits, the deletion deletes Web
// and the rows under it: Web, and each row his write made, deleted and last
// written at acme's deletion's time, by alice, and not before it was made;
// Web's time no earlier than his change's, which his answer gives. A
// deletion that went past acme's row would wait for Web's, or for his
// membership of acme, with a time read before his write committed.
func TestAWorkspacesDeletionWaitsForTheWritesOnItsProjects(t *testing.T) {
	contract := apitest.Load(t)
	for _, w := range []struct {
		name, method, path, body string
		action                   shared.Action
		rows                     string // the rows the write wrote, by $1 Web's id and $2 bob's
		want                     int
	}{
		{"updateProject", http.MethodPatch, "", `{"name":"Site"}`, projectdomain.ActionUpdate,
			"SELECT deleted_at, updated_at, updated_by_id, created_at FROM projects WHERE id = $1 AND $2::uuid IS NOT NULL", 1},
		{"joinProject", http.MethodPost, "/join", "", projectdomain.ActionJoin, `
			SELECT deleted_at, updated_at, updated_by_id, created_at FROM project_members WHERE project_id = $1 AND member_id = $2
			UNION ALL
			SELECT deleted_at, updated_at, updated_by_id, created_at FROM project_user_properties WHERE project_id = $1 AND user_id = $2`, 2},
	} {
		t.Run(w.name, func(t *testing.T) {
			var r growthRace
			if w.action == projectdomain.ActionUpdate {
				r, _ = bobAdministersWeb(t)
			} else {
				r = newGrowthRace(t, false)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			route := newProjectRoute(t, r.pool, gatedAuthorizer{Authorizer: r.authorizer(), action: w.action, gate: g},
				workspace.Provide(r.pool).WorkspaceMembers)
			var req *http.Request
			var rec *httptest.ResponseRecorder
			wrote := run(func() error {
				req, rec = route.send(w.method, "/api/v0/projects/"+r.web.String()+w.path, r.bob, w.body)
				return nil
			})
			held(t, ctx, g, wrote, "bob's write")
			deleted := run(func() error { return r.deleteAcme(ctx, workspacepg.New(r.pool)) })
			pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
			close(g.open)

			if err := errors.Join(result(t, ctx, wrote, "bob's write"), result(t, ctx, deleted, "the deletion")); err != nil {
				t.Fatal(err)
			}
			contract.CheckResponse(t, req, rec.Result())
			if rec.Code != http.StatusOK {
				t.Fatalf("bob's write = %d %s, want 200", rec.Code, rec.Body)
			}
			if n := r.deletedWithAcme(t, w.rows, r.web, r.bob); n != w.want {
				t.Errorf("%d rows of bob's write, want %d", n, w.want)
			}
			if w.action != projectdomain.ActionUpdate {
				return
			}
			var changed struct {
				UpdatedAt time.Time `json:"updated_at"`
			}
			var deletedAt time.Time
			if err := errors.Join(json.Unmarshal(rec.Body.Bytes(), &changed),
				r.pool.QueryRow(context.Background(), "SELECT deleted_at FROM projects WHERE id = $1", r.web).Scan(&deletedAt)); err != nil {
				t.Fatal(err)
			}
			if deletedAt.Before(changed.UpdatedAt) {
				t.Errorf("Web deleted at %v, before bob's change at %v", deletedAt, changed.UpdatedAt)
			}
		})
	}
}

// answeredOrWaiting returns true once done yields, or false once n backends
// of pool's database wait for a lock; it fails the test when neither has
// happened within 5s. It suits a database on which only the test's sides
// can wait.
func answeredOrWaiting(t *testing.T, pool *pgxpool.Pool, n int, done <-chan error) bool {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(10 * time.Millisecond) {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			return true
		default:
		}
		var waiting int
		if err := pool.QueryRow(soon(t),
			"SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'").Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting >= n {
			return false
		}
		if time.Now().After(deadline) {
			t.Fatalf("no answer, and %d backends waiting, within 5s; want an answer or %d waiting", waiting, n)
		}
	}
}

// Adding several members to a project and deleting its workspace serialize
// on the workspace's row, with no 40P01, in the schedule of the P4b
// pre-flight's M1: carol and dave are acme's members; carol's membership
// has the lesser id, so the adding's lock of its targets (in id order)
// meets hers first; dave's row comes first in the table and in member_id
// order, so the deletion's update of acme's memberships meets his first.
// Carol joins Ops and waits after her decision, holding acme and her
// membership of it FOR SHARE (her own growth elsewhere, convention 3).
// Alice's deletion of acme waits for acme's row. Alice adds carol and dave
// to Web: the adding shares acme's row, and both memberships, with the
// joining, and answers 201 while the deletion waits. Once the joining
// commits, the deletion deletes the three new memberships at its one time.
// Without the workspace's share, the deletion would hold acme, update
// dave's membership and wait for carol's; the adding would share carol's
// and wait for dave's; once the joining ended, the deletion would wait for
// the adding: a cycle, 40P01. The test lets the adding's one deadlock check
// pass before the joining ends, so that the deletion is the one to find it.
func TestAddingSeveralMembersAndDeletingTheWorkspaceSerialize(t *testing.T) {
	contract := apitest.Load(t)
	r, ops := bobAdministersWeb(t)
	ctx, now := context.Background(), time.Now()
	dave, carol := uuid.NewV7(), uuid.NewV7()
	carolIn, daveIn := uuid.NewV7(), uuid.NewV7()
	var acme uuid.UUID
	if err := r.pool.QueryRow(ctx, "SELECT id FROM workspaces WHERE slug = 'acme'").Scan(&acme); err != nil {
		t.Fatal(err)
	}
	users := identitypg.New(r.pool)
	for _, m := range []struct{ id, user uuid.UUID }{{daveIn, dave}, {carolIn, carol}} {
		if err := errors.Join(users.CreateUser(ctx, identityapp.NewUser{ID: m.user, Email: m.user.String() + "@example.com", PasswordHash: "x",
			DisplayName: "x", Now: now}), users.CreateDefaultProfile(ctx, uuid.NewV7(), m.user, now),
			workspacepg.New(r.pool).CreateMember(ctx, workspaceapp.MemberRow{ID: m.id, WorkspaceID: acme, MemberID: m.user, Role: shared.RoleMember,
				CreatedBy: r.alice, Now: now})); err != nil {
			t.Fatal(err)
		}
	}
	if carolIn.String() >= daveIn.String() || dave.String() >= carol.String() {
		t.Fatal("carol's membership's id is not the lesser, or dave's account's is not")
	}
	tctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	g := newGate()
	joining := newProjectRoute(t, r.pool, gatedAuthorizer{Authorizer: r.authorizer(), action: projectdomain.ActionJoin, gate: g},
		workspace.Provide(r.pool).WorkspaceMembers)
	adding := newProjectRoute(t, r.pool, r.authorizer(), workspace.Provide(r.pool).WorkspaceMembers)
	var joinRec, addRec *httptest.ResponseRecorder
	var addReq *http.Request
	joined := run(func() error {
		_, joinRec = joining.send(http.MethodPost, "/api/v0/projects/"+ops.String()+"/join", carol, "")
		return nil
	})
	held(t, tctx, g, joined, "carol's joining")
	deleted := run(func() error { return r.deleteAcme(tctx, workspacepg.New(r.pool)) })
	pgtest.WaitForLockWait(t, r.pool, 5*time.Second)
	added := run(func() error {
		addReq, addRec = adding.send(http.MethodPost, "/api/v0/projects/"+r.web.String()+"/members", r.alice,
			`{"members":[{"member_id":"`+carol.String()+`","role":15},{"member_id":"`+dave.String()+`","role":15}]}`)
		return nil
	})
	answered := answeredOrWaiting(t, r.pool, 2, added)
	if !answered {
		time.Sleep(1200 * time.Millisecond) // past the adding's one deadlock check (deadlock_timeout, 1s)
	}
	close(g.open)

	errs := []error{result(t, tctx, joined, "carol's joining"), result(t, tctx, deleted, "the deletion")}
	if !answered {
		errs = append(errs, result(t, tctx, added, "the adding"))
	}
	if err := errors.Join(errs...); err != nil {
		t.Fatal(err)
	}
	if !answered {
		t.Error("the adding did not answer while the deletion waited: it waited for a lock the joining or the deletion held, where it " +
			"shares acme and both memberships with the joining")
	}
	contract.CheckResponse(t, addReq, addRec.Result())
	if joinRec.Code != http.StatusOK || addRec.Code != http.StatusCreated {
		t.Errorf("carol's joining = %d %s, the adding = %d %s; want 200 and 201", joinRec.Code, joinRec.Body, addRec.Code, addRec.Body)
	}
	if n := r.deletedWithAcme(t, "SELECT deleted_at, updated_at, updated_by_id, created_at FROM project_members WHERE member_id = ANY ($1::uuid[])",
		[]uuid.UUID{carol, dave}); n != 3 {
		t.Errorf("%d memberships of carol's and dave's, want 3: hers of Ops and Web, his of Web", n)
	}
}
