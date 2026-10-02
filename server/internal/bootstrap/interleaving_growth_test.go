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

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	identitypg "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
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

// The interleaving 17 of M3 design 9.3, and a demotion against deleteProject,
// on a real database: the project side through the project module as
// bootstrap wires it (project.New, behind the API), the demotion through
// workspace's use case, both with the Authorizer as bootstrap wires it. A
// gate inside the first side's transaction holds it open at a lock the
// second side needs; pgtest.WaitForLockWaitOn proves that the second side
// waits on that table's row before the gate opens: nothing else runs on
// the database, and the table is the one design 3.6 says the second side
// meets first. Every wait has a deadline.

// growthRace is a database with acme, whose admin is alice and whose
// member is bob, and alice's public project Web, of which bob has an ended
// membership as a member when ended is set: P5's removal ends one, and SQL
// stands in for it.
type growthRace struct {
	race
	bob, bobIn, web uuid.UUID
}

func newGrowthRace(t *testing.T, ended bool) growthRace {
	t.Helper()
	r := growthRace{race: newRace(t), bob: uuid.NewV7(), bobIn: uuid.NewV7(), web: uuid.NewV7()}
	ctx, now := context.Background(), time.Now()
	users := identitypg.New(r.pool)
	if err := users.CreateUser(ctx, identityapp.NewUser{ID: r.bob, Email: "bob@example.com", PasswordHash: "x", DisplayName: "bob", Now: now}); err != nil {
		t.Fatal(err)
	}
	if err := users.CreateDefaultProfile(ctx, uuid.NewV7(), r.bob, now); err != nil {
		t.Fatal(err)
	}
	workspaces := workspacepg.New(r.pool)
	w, err := workspaces.CreateWorkspace(ctx, workspaceapp.WorkspaceRow{ID: uuid.NewV7(), Name: "Acme", Slug: "acme", Timezone: "UTC",
		CreatedBy: r.alice, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	for id, m := range map[uuid.UUID]struct {
		user uuid.UUID
		role shared.Role
	}{uuid.NewV7(): {r.alice, shared.RoleAdmin}, r.bobIn: {r.bob, shared.RoleMember}} {
		if err := workspaces.CreateMember(ctx, workspaceapp.MemberRow{ID: id, WorkspaceID: w.ID, MemberID: m.user, Role: m.role, CreatedBy: r.alice,
			Now: now}); err != nil {
			t.Fatal(err)
		}
	}
	projects := projectpg.New(r.pool)
	if err := projects.CreateProject(ctx, projectapp.ProjectRow{ID: r.web, WorkspaceID: w.ID, Name: "Web", Identifier: "WEB",
		Network: projectdomain.NetworkPublic, Timezone: "UTC", CreatedBy: r.alice, Now: now}); err != nil {
		t.Fatal(err)
	}
	members := []uuid.UUID{r.alice}
	if ended {
		members = append(members, r.bob)
	}
	for _, user := range members {
		if err := projects.CreateMember(ctx, projectapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: w.ID, ProjectID: r.web, MemberID: user,
			Role: map[uuid.UUID]shared.Role{r.alice: shared.RoleAdmin, r.bob: shared.RoleMember}[user], CreatedBy: r.alice, Now: now}); err != nil {
			t.Fatal(err)
		}
	}
	if ended {
		if tag, err := r.pool.Exec(ctx, "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2", r.web, r.bob); err != nil ||
			tag.RowsAffected() != 1 {
			t.Fatalf("ending bob's membership of Web: %v, %v", tag, err)
		}
	}
	return r
}

// authorizer is access's Authorizer as bootstrap wires it, on r's pool.
func (r growthRace) authorizer() shared.Authorizer {
	return authorizerOn(r.pool)
}

// authorizerOn is access's Authorizer as bootstrap wires it, on pool.
func authorizerOn(pool *pgxpool.Pool) shared.Authorizer {
	return access.New(access.Deps{WorkspaceRoles: workspace.Provide(pool).WorkspaceRoles,
		ProjectAccess: accessProjects{projects: project.Provide(pool).ProjectAccess}})
}

// gatedShares is workspace's WorkspaceMembers; with a gate, the growth
// waits at it once its workspace and its targets' memberships of it are
// held FOR SHARE, before it locks the project (M3 design 3.6 conventions 2
// and 3).
type gatedShares struct {
	projectapp.WorkspaceMembers
	gate *gate
}

func (m gatedShares) ShareMembers(ctx context.Context, workspaceID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	roles, err := m.WorkspaceMembers.ShareMembers(ctx, workspaceID, userIDs)
	if err != nil || m.gate == nil {
		return roles, err
	}
	return roles, m.gate.wait(ctx)
}

// growth is bob's joining Web, or alice's adding him to it as a member,
// through the project module as bootstrap wires it, its WorkspaceMembers
// gated by g when g is not nil: a function that sends the request and
// answers it with its answer, which another goroutine may call.
func (r growthRace) growth(t *testing.T, add bool, g *gate) func() (*http.Request, *httptest.ResponseRecorder) {
	t.Helper()
	route := newProjectRoute(t, r.pool, r.authorizer(), gatedShares{workspace.Provide(r.pool).WorkspaceMembers, g})
	web := "/api/v0/projects/" + r.web.String()
	if add {
		return func() (*http.Request, *httptest.ResponseRecorder) {
			return route.send(http.MethodPost, web+"/members", r.alice, `{"members":[{"member_id":"`+r.bob.String()+`","role":15}]}`)
		}
	}
	return func() (*http.Request, *httptest.ResponseRecorder) {
		return route.send(http.MethodPost, web+"/join", r.bob, "")
	}
}

// join makes bob Web's member: his joining it, through the project module.
func (r growthRace) join(t *testing.T) {
	t.Helper()
	if _, rec := r.growth(t, false, nil)(); rec.Code != http.StatusOK {
		t.Fatalf("bob's joining Web = %d %s", rec.Code, rec.Body)
	}
}

// demote is alice's change of bob's role in acme to guest, over members
// and cascade, on the system's clock: its time is read when it reads it.
func (r growthRace) demote(ctx context.Context, members workspaceapp.MemberUpdater, cascade workspaceapp.ProjectCascade) error {
	_, err := workspaceapp.NewUpdateWorkspaceMember(members, cascade, workspaceProfiles{profiles: identity.Provide(r.pool).PublicProfiles},
		r.authorizer(), postgres.NewTxManager(r.pool, 2*time.Second), clock.System{}).
		Execute(shared.WithActor(ctx, shared.Actor{UserID: r.alice}), r.bobIn, shared.RoleGuest)
	return err
}

// standing is bob's role in acme, then his membership of Web: its role,
// "ended" when it is, "deleted" when Web is; "none" when he has none.
func (r growthRace) standing(t *testing.T) string {
	t.Helper()
	var s string
	if err := r.pool.QueryRow(context.Background(), `SELECT (SELECT role::text FROM workspace_members WHERE id = $1) || ', Web ' ||
		coalesce((SELECT role || CASE WHEN is_active THEN '' ELSE ' ended' END || CASE WHEN deleted_at IS NULL THEN '' ELSE ' deleted' END
		          FROM project_members WHERE project_id = $2 AND member_id = $3), 'none')`, r.bobIn, r.web, r.bob).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// webFree reports whether no transaction holds Web's row: a FOR UPDATE
// NOWAIT of it answers lock_not_available (55P03) at once while one does.
func (r growthRace) webFree(t *testing.T) bool {
	t.Helper()
	_, err := r.pool.Exec(context.Background(), "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE NOWAIT", r.web)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "55P03" {
		return false
	}
	if err != nil {
		t.Fatal(err)
	}
	return true
}

// demotedHolding stops a role change after its write of the membership,
// holding the membership's row, before the projects' step.
type demotedHolding struct {
	*workspacepg.Store
	gate *gate
}

func (m demotedHolding) UpdateMemberRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (
	workspacedomain.Membership, error) {
	updated, err := m.Store.UpdateMemberRole(ctx, id, role, by, now)
	if err != nil {
		return updated, err
	}
	return updated, m.gate.wait(ctx)
}

// A demotion to guest and the project side's growth, bob's joining Web or
// alice's adding him to it, serialize on acme's row (M3 design 3.6
// conventions 2, 3 and 6, 9.3's interleaving 17), in both orders, for a
// new membership of Web and for his ended one. The growth first: it holds
// acme and his membership of acme FOR SHARE, no stronger (lockOn), and
// waits at its gate before it locks Web; the demotion waits for acme's
// row, holding nothing. Once the growth has committed, the demotion's step
// over his projects, a statement run after its write of his membership,
// finds his membership of Web and makes it a guest's, at the demotion's
// time, read once it held acme: after the gate opened (3.3). The demotion
// first: it holds acme FOR NO KEY UPDATE and his membership's row after its
// write; the growth waits for acme's row, then reads him a guest of acme:
// his joining is refused as one who does not see Web (404), alice's adding
// him as a member as a role a guest may not have (422 members[0].role); an
// ended membership is a guest's, by the demotion. In either order, while
// the second side waits for acme's row, no transaction holds Web (FOR
// UPDATE NOWAIT): the growth takes his membership of acme before Web, and
// the demotion his membership before his projects (P4a's F-M2 order: a
// growth that locked Web first would hold it at its gate; a demotion that
// locked his projects first would hold Web, of which he has an ended
// membership). His membership of Web, once there, was last written no
// earlier than it was made.
func TestADemotionAndTheProjectSidesGrowthSerialize(t *testing.T) {
	contract := apitest.Load(t)
	for _, add := range []bool{false, true} {
		for _, ended := range []bool{false, true} {
			for _, growthFirst := range []bool{true, false} {
				name := fmt.Sprintf("join, ended %v, growth first %v", ended, growthFirst)
				if add {
					name = fmt.Sprintf("add, ended %v, growth first %v", ended, growthFirst)
				}
				t.Run(name, func(t *testing.T) {
					r := newGrowthRace(t, ended)
					before := "15, Web none"
					if ended {
						before = "15, Web 15 ended"
					}
					if got := r.standing(t); got != before {
						t.Fatalf("before: %s, want %s", got, before)
					}
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					g, cascade := newGate(), project.New(project.Deps{Pool: r.pool}).Cascade()
					var req *http.Request
					var rec *httptest.ResponseRecorder
					var grew, demoted <-chan error
					if growthFirst {
						grow := r.growth(t, add, g)
						grew = run(func() error { req, rec = grow(); return nil })
						held(t, ctx, g, grew, "the growth")
						if got, want := "acme "+lockOn(t, r.pool, "workspaces WHERE slug = 'acme'")+", his membership "+lockOn(t, r.pool,
							"workspace_members WHERE id = $1", r.bobIn), "acme FOR SHARE, his membership FOR SHARE"; got != want {
							t.Errorf("the growth at its gate holds %s; want %s", got, want)
						}
						demoted = run(func() error { return r.demote(ctx, workspacepg.New(r.pool), cascade) })
					} else {
						demoted = run(func() error { return r.demote(ctx, demotedHolding{workspacepg.New(r.pool), g}, cascade) })
						held(t, ctx, g, demoted, "the demotion")
						grow := r.growth(t, add, nil)
						grew = run(func() error { req, rec = grow(); return nil })
					}
					pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
					if !r.webFree(t) {
						t.Error("Web is held while the second side waits for acme's row; want it locked after that row")
					}
					opened := time.Now()
					close(g.open)

					demotion := result(t, ctx, demoted, "the demotion")
					if err := result(t, ctx, grew, "the growth"); err != nil {
						t.Fatal(err)
					}
					contract.CheckResponse(t, req, rec.Result())
					want, grown := "5, Web 5", rec.Code == map[bool]int{false: http.StatusOK, true: http.StatusCreated}[add]
					if !growthFirst {
						want, grown = map[bool]string{false: "5, Web none", true: "5, Web 5 ended"}[ended], refusedAsAGuest(rec, add)
					}
					if got := r.standing(t); demotion != nil || !grown || got != want {
						t.Errorf("the demotion = %v, the growth = %d %s, bob %s; want the demotion done, the growth %s, bob %s", demotion, rec.Code,
							rec.Body, got, map[bool]string{true: "done", false: "refused"}[growthFirst], want)
					}
					if made, written := r.membershipTimes(t); written.Before(made) || (growthFirst && written.Before(opened)) {
						t.Errorf("bob's membership of Web made at %v, last written at %v, the gate opened at %v; want it written last by the "+
							"demotion, at a time read once it held acme", made, written, opened)
					}
				})
			}
		}
	}
}

// membershipTimes are bob's membership of Web's created_at and updated_at.
func (r growthRace) membershipTimes(t *testing.T) (made, written time.Time) {
	t.Helper()
	if err := r.pool.QueryRow(context.Background(), "SELECT created_at, updated_at FROM project_members WHERE project_id = $1 AND member_id = $2",
		r.web, r.bob).Scan(&made, &written); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal(err)
	}
	return made, written
}

// refusedAsAGuest reports whether rec is the growth's refusal of bob as
// acme's guest: his joining, 404 project.not_found; alice's adding him as a
// member, 422 members[0].role not_allowed alone.
func refusedAsAGuest(rec *httptest.ResponseRecorder, add bool) bool {
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
	if !add {
		return rec.Code == http.StatusNotFound && problem.Code == "project.not_found"
	}
	return rec.Code == http.StatusUnprocessableEntity && problem.Code == "validation_failed" && len(problem.Errors) == 1 &&
		problem.Errors[0].Field == "members[0].role" && problem.Errors[0].Code == "not_allowed"
}

// gatedDemoter stops the demotion's step over the projects after it has
// locked them, before its write.
type gatedDemoter struct {
	*projectpg.Store
	gate *gate
}

func (d gatedDemoter) DemoteMemberships(ctx context.Context, projectIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	if err := d.gate.wait(ctx); err != nil {
		return err
	}
	return d.Store.DemoteMemberships(ctx, projectIDs, userID, by, now)
}

// A demotion to guest and the deletion of a project its member is in
// serialize on acme's row, in both orders (M3 design 3.6): bob is a member
// of Web. The deletion first holds acme FOR SHARE and Web FOR NO KEY
// UPDATE, and waits after its decision; the demotion waits for acme's row,
// then its step over his projects finds Web deleted and leaves it out, so
// his membership is deleted as it was, a member's. The demotion first holds
// acme and, after locking it, Web; the deletion waits for acme's row, then
// deletes his membership, a guest's by then.
func TestADemotionAndAProjectsDeletionSerialize(t *testing.T) {
	contract := apitest.Load(t)
	for _, deletionFirst := range []bool{true, false} {
		t.Run(fmt.Sprintf("deletion first %v", deletionFirst), func(t *testing.T) {
			r := newGrowthRace(t, false)
			r.join(t)
			if got := r.standing(t); got != "15, Web 15" {
				t.Fatalf("before: %s, want 15, Web 15", got)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g, store := newGate(), projectpg.New(r.pool)
			auth, cascade := r.authorizer(), projectapp.NewCascade(store, store, store)
			if deletionFirst {
				auth = gatedAuthorizer{Authorizer: auth, action: projectdomain.ActionDelete, gate: g}
			} else {
				cascade = projectapp.NewCascade(store, gatedDemoter{store, g}, store)
			}
			route := newProjectRoute(t, r.pool, auth, workspace.Provide(r.pool).WorkspaceMembers)
			var req *http.Request
			var rec *httptest.ResponseRecorder
			deletion := func() error {
				req, rec = route.send(http.MethodDelete, "/api/v0/projects/"+r.web.String(), r.alice, "")
				return nil
			}
			demotion := func() error { return r.demote(ctx, workspacepg.New(r.pool), cascade) }
			first, second := deletion, demotion
			if !deletionFirst {
				first, second = demotion, deletion
			}
			firstDone := run(first)
			held(t, ctx, g, firstDone, "the first")
			secondDone := run(second)
			pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
			close(g.open)

			want := map[bool]string{true: "5, Web 15 deleted", false: "5, Web 5 deleted"}[deletionFirst]
			a, b := result(t, ctx, firstDone, "the first"), result(t, ctx, secondDone, "the second")
			contract.CheckResponse(t, req, rec.Result())
			if got := r.standing(t); a != nil || b != nil || rec.Code != http.StatusNoContent || got != want {
				t.Errorf("the first = %v, the second = %v, the deletion = %d %s, bob %s; want both done, the deletion 204, bob %s", a, b, rec.Code,
					rec.Body, got, want)
			}
		})
	}
}
