package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"uuid"

	projectpg "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	projectapp "github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A write on a project decides under its locks, its workspace's FOR SHARE
// and its project's, which it holds until it commits (M3 design 3.6
// convention 2, 6.7): against a demotion to guest, which holds the
// workspace FOR NO KEY UPDATE; against another write on the project, which
// holds the project; and beside a write on another project of the
// workspace, which shares the workspace. The writes run as bootstrap wires
// them (project.New), behind the API, so the locks are the ones of the
// transaction project.New is given.

// gatedAuthorizer is an Authorizer whose decision on action, once made,
// waits at the gate: inside the write's transaction, after its lock.
type gatedAuthorizer struct {
	shared.Authorizer
	action shared.Action
	gate   *gate
}

func (a gatedAuthorizer) Authorize(ctx context.Context, actor shared.Actor, action shared.Action, t shared.Target) (shared.Grant, error) {
	grant, err := a.Authorizer.Authorize(ctx, actor, action, t)
	if err == nil && action == a.action {
		if err := a.gate.wait(ctx); err != nil {
			return shared.Grant{}, err
		}
	}
	return grant, err
}

// Bob, acme's member, is the admin of Web; alice makes him acme's guest
// while he changes Web, archives it, unarchives it, or changes his display
// settings in it, in both orders. His write first: it holds acme FOR SHARE
// and Web, FOR NO KEY UPDATE or, for his settings, FOR SHARE, each no
// stronger (lockOn), and waits after its decision; the demotion waits for
// acme's row, then makes him Web's guest. The demotion first: it holds
// acme FOR NO KEY UPDATE and, by its step over his projects, Web; his write
// waits for acme's row, then decides on what the demotion committed: Web's
// guest may not change, archive or unarchive it (403 forbidden), and may
// change his own settings.
func TestAProjectWriteAndADemotionSerialize(t *testing.T) {
	writes := []struct {
		name, method, path, body string
		action                   shared.Action // the write's decision, where its gate holds it
		archived                 bool          // Web is archived before the write
		done                     string        // the statement of whether the write is done, by $1 Web's id and $2 bob's
		guests                   bool          // Web's guest may write it
		lock                     string        // the write's lock of Web, which it holds at its gate
	}{
		{"updateProject", http.MethodPatch, "/api/v0/projects/%s", `{"name":"Site"}`, projectdomain.ActionUpdate, false,
			"SELECT name = 'Site' FROM projects WHERE id = $1 AND $2::uuid IS NOT NULL", false, "FOR NO KEY UPDATE"},
		{"archiveProject", http.MethodPost, "/api/v0/projects/%s/archive", "", projectdomain.ActionArchive, false,
			"SELECT archived_at IS NOT NULL FROM projects WHERE id = $1 AND $2::uuid IS NOT NULL", false, "FOR NO KEY UPDATE"},
		{"unarchiveProject", http.MethodPost, "/api/v0/projects/%s/unarchive", "", projectdomain.ActionUnarchive, true,
			"SELECT archived_at IS NULL FROM projects WHERE id = $1 AND $2::uuid IS NOT NULL", false, "FOR NO KEY UPDATE"},
		{"updateProjectPreferences", http.MethodPatch, "/api/v0/me/projects/%s/preferences", `{"sort_order":1}`,
			projectdomain.ActionPreferencesUpdate, false,
			"SELECT sort_order = 1 FROM project_user_properties WHERE project_id = $1 AND user_id = $2 AND deleted_at IS NULL", true,
			"FOR SHARE"},
	}
	contract := apitest.Load(t)
	for _, w := range writes {
		for _, writeFirst := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s, the write first %v", w.name, writeFirst), func(t *testing.T) {
				r := newGrowthRace(t, false)
				r.join(t)
				if _, err := r.pool.Exec(context.Background(), "UPDATE project_members SET role = 20 WHERE project_id = $1 AND member_id = $2",
					r.web, r.bob); err != nil {
					t.Fatal(err)
				}
				if _, err := r.pool.Exec(context.Background(), "UPDATE projects SET archived_at = CASE WHEN $2 THEN now() END WHERE id = $1",
					r.web, w.archived); err != nil {
					t.Fatal(err)
				}
				if got := r.standing(t); got != "15, Web 20" {
					t.Fatalf("before: %s, want 15, Web 20", got)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				g, store := newGate(), projectpg.New(r.pool)
				auth := r.authorizer()
				if writeFirst {
					auth = gatedAuthorizer{Authorizer: auth, action: w.action, gate: g}
				}
				route := newProjectRoute(t, r.pool, auth, workspace.Provide(r.pool).WorkspaceMembers)
				var req *http.Request
				var rec *httptest.ResponseRecorder
				write := func() error {
					req, rec = route.send(w.method, fmt.Sprintf(w.path, r.web), r.bob, w.body)
					return nil
				}
				var wrote, demoted <-chan error
				if writeFirst {
					wrote = run(write)
					held(t, ctx, g, wrote, "the write")
					if got, want := "acme "+lockOn(t, r.pool, "workspaces WHERE slug = 'acme'")+", Web "+lockOn(t, r.pool, "projects WHERE id = $1",
						r.web), "acme FOR SHARE, Web "+w.lock; got != want {
						t.Errorf("the write at its gate holds %s; want %s", got, want)
					}
					demoted = run(func() error {
						return r.demote(ctx, workspacepg.New(r.pool), projectapp.NewCascade(store, store, store))
					})
				} else {
					demoted = run(func() error {
						return r.demote(ctx, workspacepg.New(r.pool), projectapp.NewCascade(store, gatedDemoter{store, g}, store))
					})
					held(t, ctx, g, demoted, "the demotion")
					wrote = run(write)
				}
				pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
				close(g.open)

				demotion := result(t, ctx, demoted, "the demotion")
				if err := result(t, ctx, wrote, "the write"); err != nil {
					t.Fatal(err)
				}
				contract.CheckResponse(t, req, rec.Result())
				var done bool
				if err := r.pool.QueryRow(context.Background(), w.done, r.web, r.bob).Scan(&done); err != nil {
					t.Fatal(err)
				}
				wantDone, want := writeFirst || w.guests, http.StatusOK
				if !wantDone {
					want = http.StatusForbidden
				}
				if got := r.standing(t); demotion != nil || rec.Code != want || done != wantDone || got != "5, Web 5" {
					t.Errorf("the demotion = %v, the write = %d %s (done %v), bob %s; want the demotion done, the write %d (done %v), "+
						"bob 5, Web 5", demotion, rec.Code, rec.Body, done, got, want, wantDone)
				}
			})
		}
	}
}

// bobAdministersWeb is a growthRace in which bob, acme's member, has joined
// Web and is its admin (SQL stands in for P5's role change), and Ops is
// another project of acme, of which alice is the admin.
func bobAdministersWeb(t *testing.T) (r growthRace, ops uuid.UUID) {
	t.Helper()
	r = newGrowthRace(t, false)
	r.join(t)
	if _, err := r.pool.Exec(context.Background(), "UPDATE project_members SET role = 20 WHERE project_id = $1 AND member_id = $2", r.web,
		r.bob); err != nil {
		t.Fatal(err)
	}
	ops = uuid.NewV7()
	var acme uuid.UUID
	if err := r.pool.QueryRow(context.Background(), "SELECT workspace_id FROM projects WHERE id = $1", r.web).Scan(&acme); err != nil {
		t.Fatal(err)
	}
	store := projectpg.New(r.pool)
	if err := errors.Join(store.CreateProject(context.Background(), projectapp.ProjectRow{ID: ops, WorkspaceID: acme, Name: "Ops", Identifier: "OPS",
		Network: projectdomain.NetworkPublic, Timezone: "UTC", CreatedBy: r.alice, Now: time.Now()}),
		store.CreateMember(context.Background(), projectapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: ops, MemberID: r.alice,
			Role: shared.RoleAdmin, CreatedBy: r.alice, Now: time.Now()})); err != nil {
		t.Fatal(err)
	}
	return r, ops
}

// Two writes on one project serialize on its row (M3 design 3.6 convention
// 2: a write that changes the project row holds it FOR NO KEY UPDATE): bob
// archives Web and waits after his decision, holding acme FOR SHARE and
// Web; his change of Web takes acme's row beside it and waits for Web's.
// Once the archive commits, the change decides on Web archived and answers
// 409 project.archived. Under a FOR SHARE lock of Web the change would take
// Web beside the archive and wait only at its UPDATE, which upgrades a lock
// it shares: PostgreSQL takes no tuple lock for that wait, so the probe of
// a wait on projects times out, and the test fails there, not at a 40P01.
func TestTwoWritesOnAProjectSerialize(t *testing.T) {
	contract := apitest.Load(t)
	r, _ := bobAdministersWeb(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	g := newGate()
	gated := newProjectRoute(t, r.pool, gatedAuthorizer{Authorizer: r.authorizer(), action: projectdomain.ActionArchive, gate: g},
		workspace.Provide(r.pool).WorkspaceMembers)
	route := newProjectRoute(t, r.pool, r.authorizer(), workspace.Provide(r.pool).WorkspaceMembers)
	web := "/api/v0/projects/" + r.web.String()
	var archiveReq, updateReq *http.Request
	var archiveRec, updateRec *httptest.ResponseRecorder
	archived := run(func() error {
		archiveReq, archiveRec = gated.send(http.MethodPost, web+"/archive", r.bob, "")
		return nil
	})
	held(t, ctx, g, archived, "the archive")
	updated := run(func() error {
		updateReq, updateRec = route.send(http.MethodPatch, web, r.bob, `{"name":"Site"}`)
		return nil
	})
	pgtest.WaitForLockWaitOn(t, r.pool, "projects", 5*time.Second)
	close(g.open)

	if err := errors.Join(result(t, ctx, archived, "the archive"), result(t, ctx, updated, "the change")); err != nil {
		t.Fatal(err)
	}
	contract.CheckResponse(t, archiveReq, archiveRec.Result())
	contract.CheckResponse(t, updateReq, updateRec.Result())
	if archiveRec.Code != http.StatusOK || updateRec.Code != http.StatusConflict || !strings.Contains(updateRec.Body.String(), `"project.archived"`) {
		t.Errorf("the archive = %d %s, the change = %d %s; want 200 and 409 project.archived", archiveRec.Code, archiveRec.Body, updateRec.Code,
			updateRec.Body)
	}
}

// Writes on two projects of one workspace do not wait for each other: each
// holds the workspace FOR SHARE, which the other shares (M3 design 3.6
// convention 2). Bob archives Web and waits after his decision, holding
// acme and Web; meanwhile alice changes Ops, and her change answers 200.
// Under a stronger lock of the workspace her change would wait for the
// archive.
func TestWritesOnTwoProjectsOfAWorkspaceDoNotWait(t *testing.T) {
	contract := apitest.Load(t)
	r, ops := bobAdministersWeb(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	g := newGate()
	gated := newProjectRoute(t, r.pool, gatedAuthorizer{Authorizer: r.authorizer(), action: projectdomain.ActionArchive, gate: g},
		workspace.Provide(r.pool).WorkspaceMembers)
	route := newProjectRoute(t, r.pool, r.authorizer(), workspace.Provide(r.pool).WorkspaceMembers)
	archived := run(func() error {
		gated.send(http.MethodPost, "/api/v0/projects/"+r.web.String()+"/archive", r.bob, "")
		return nil
	})
	held(t, ctx, g, archived, "the archive")
	type answer struct {
		req *http.Request
		rec *httptest.ResponseRecorder
	}
	changed := make(chan answer, 1)
	go func() {
		req, rec := route.send(http.MethodPatch, "/api/v0/projects/"+ops.String(), r.alice, `{"name":"Site"}`)
		changed <- answer{req, rec}
	}()
	select {
	case a := <-changed:
		contract.CheckResponse(t, a.req, a.rec.Result())
		if a.rec.Code != http.StatusOK {
			t.Errorf("alice's change of Ops while the archive holds acme = %d %s, want 200", a.rec.Code, a.rec.Body)
		}
	case <-time.After(5 * time.Second):
		t.Error("alice's change of Ops did not answer within 5s while the archive of Web held acme: it waited for it")
	}
	close(g.open)
	if err := result(t, ctx, archived, "the archive"); err != nil {
		t.Fatal(err)
	}
}
