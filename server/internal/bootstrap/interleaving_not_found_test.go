package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	projectdomain "github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A write on a project that waits for a deletion, of its project or of its
// workspace, finds no project once the deletion commits: it answers 404
// project.not_found, as for a project never there, never 403, which would
// tell its caller that the project was there; and it changes no row (M3
// design 3.6 convention 2, 6.4). Each write is bob's, through the project
// module as bootstrap wires it, behind the API; the project's deletion
// too, the workspace's through workspace's use case (deleteAcme). Every
// wait has a deadline.

// underWeb is every row under Web, deleted ones too, in each table the
// catalog ties to projects (keysTo), by table and id: its columns but the
// three a deletion writes, then ", deleted with Web" when alice's deletion
// of Web, or of acme, wrote those last: deleted_at and updated_at Web's
// deleted_at, updated_by_id alice.
func (r growthRace) underWeb(t *testing.T) []string {
	t.Helper()
	var all []string
	for _, k := range keysTo(t, r.pool, "projects") {
		rows, err := r.pool.Query(context.Background(), `SELECT (to_jsonb(r) - 'deleted_at' - 'updated_at' - 'updated_by_id')::text ||
			CASE WHEN (r.deleted_at, r.updated_at, r.updated_by_id) = (w.deleted_at, w.deleted_at, $2::uuid) THEN ', deleted with Web' ELSE '' END
			FROM `+k.table+` r JOIN projects w ON w.id = $1 WHERE r.`+pgx.Identifier{k.column}.Sanitize()+` = $1 ORDER BY r.id`, r.web, r.alice)
		if err != nil {
			t.Fatalf("%s: %v", k, err)
		}
		texts, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			t.Fatalf("%s: %v", k, err)
		}
		for _, s := range texts {
			all = append(all, k.String()+": "+s)
		}
	}
	return all
}

// webWrite is one of bob's writes on Web, as the tests of a write whose
// project or workspace is deleted while it waits send it.
type webWrite struct {
	name, method, path, body string
	joins, archived          bool // bob is not Web's member, and joins it; Web is archived before the write
}

// webWrites are bob's change of Web, its archive, its unarchive and the
// change of his display settings in it, as Web's admin, and his joining
// it, not its member yet.
var webWrites = []webWrite{
	{"updateProject", http.MethodPatch, "/api/v0/projects/%s", `{"name":"Site"}`, false, false},
	{"archiveProject", http.MethodPost, "/api/v0/projects/%s/archive", "", false, false},
	{"unarchiveProject", http.MethodPost, "/api/v0/projects/%s/unarchive", "", false, true},
	{"updateProjectPreferences", http.MethodPatch, "/api/v0/me/projects/%s/preferences", `{"sort_order":1}`, false, false},
	{"joinProject", http.MethodPost, "/api/v0/projects/%s/join", "", true, false},
}

// race is a growthRace in which bob sends w: Web's admin
// (bobAdministersWeb), or not its member when w joins it; Web is archived
// when w unarchives it.
func (w webWrite) race(t *testing.T) growthRace {
	t.Helper()
	var r growthRace
	if w.joins {
		r = newGrowthRace(t, false)
	} else {
		r, _ = bobAdministersWeb(t)
	}
	if _, err := r.pool.Exec(context.Background(), "UPDATE projects SET archived_at = CASE WHEN $2 THEN now() END WHERE id = $1", r.web,
		w.archived); err != nil {
		t.Fatal(err)
	}
	return r
}

// send sends w as bob on r's Web through route.
func (w webWrite) send(route moduleRoute, r growthRace) (*http.Request, *httptest.ResponseRecorder) {
	return route.send(w.method, fmt.Sprintf(w.path, r.web), r.bob, w.body)
}

// foundNoProject fails the test unless rec, bob's write, is 404
// project.not_found, and every row under Web is as it was before (as
// underWeb read it then), deleted with Web: a write that found no project,
// and changed nothing.
func (r growthRace) foundNoProject(t *testing.T, rec *httptest.ResponseRecorder, before []string) {
	t.Helper()
	var problem struct {
		Code string `json:"code"`
	}
	if rec.Code != http.StatusNotFound || json.Unmarshal(rec.Body.Bytes(), &problem) != nil || problem.Code != "project.not_found" {
		t.Errorf("bob's write = %d %s; want 404 project.not_found", rec.Code, rec.Body)
	}
	want := make([]string, len(before))
	for i, row := range before {
		want[i] = row + ", deleted with Web"
	}
	if got := r.underWeb(t); !slices.Equal(got, want) {
		t.Errorf("the rows under Web:\n%s\nwant them as the deletion left them:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// A write on a project whose deletion commits while the write waits for
// the project's row finds no project: it answers 404 project.not_found,
// as for a project never there, never 403, which would tell its caller
// that the project was there; and it changes no row (M3 design 3.6
// convention 2, 6.4). Alice deletes Web and waits after her decision,
// holding acme FOR SHARE and Web FOR NO KEY UPDATE. Bob, Web's admin,
// changes Web, archives it, unarchives it or changes his display settings
// in it; or, not its member yet, joins it. His write shares acme with the
// deletion and waits for Web's row; once the deletion commits, his lock of
// Web, FOR NO KEY UPDATE or, for his settings, FOR SHARE, reads no row.
// Every row under Web is then as the deletion left it: as it was before,
// deleted with Web (foundNoProject).
func TestAWriteOnAProjectDeletedWhileItWaitsFindsNoProject(t *testing.T) {
	contract := apitest.Load(t)
	for _, w := range webWrites {
		t.Run(w.name, func(t *testing.T) {
			r := w.race(t)
			before := r.underWeb(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			deleting := newProjectRoute(t, r.pool, gatedAuthorizer{Authorizer: r.authorizer(), action: projectdomain.ActionDelete, gate: g},
				workspace.Provide(r.pool).WorkspaceMembers)
			route := newProjectRoute(t, r.pool, r.authorizer(), workspace.Provide(r.pool).WorkspaceMembers)
			var deleteReq, writeReq *http.Request
			var deleteRec, writeRec *httptest.ResponseRecorder
			deleted := run(func() error {
				deleteReq, deleteRec = deleting.send(http.MethodDelete, "/api/v0/projects/"+r.web.String(), r.alice, "")
				return nil
			})
			held(t, ctx, g, deleted, "the deletion")
			if got, want := "acme "+lockOn(t, r.pool, "workspaces WHERE slug = 'acme'")+", Web "+lockOn(t, r.pool, "projects WHERE id = $1", r.web),
				"acme FOR SHARE, Web FOR NO KEY UPDATE"; got != want {
				t.Errorf("the deletion at its gate holds %s; want %s", got, want)
			}
			wrote := run(func() error {
				writeReq, writeRec = w.send(route, r)
				return nil
			})
			pgtest.WaitForLockWaitOn(t, r.pool, "projects", 5*time.Second)
			close(g.open)

			if err := errors.Join(result(t, ctx, deleted, "the deletion"), result(t, ctx, wrote, "bob's write")); err != nil {
				t.Fatal(err)
			}
			contract.CheckResponse(t, deleteReq, deleteRec.Result())
			contract.CheckResponse(t, writeReq, writeRec.Result())
			if deleteRec.Code != http.StatusNoContent {
				t.Errorf("the deletion = %d %s, want 204", deleteRec.Code, deleteRec.Body)
			}
			r.foundNoProject(t, writeRec, before)
		})
	}
}

// A write on a project whose workspace's deletion commits while the write
// waits for the workspace's row finds no project: it answers 404
// project.not_found, never 403, and changes no row (M3 design 3.6
// convention 2, 6.4), as one whose project's deletion commits while it
// waits (TestAWriteOnAProjectDeletedWhileItWaitsFindsNoProject). Alice's
// deletion of acme waits after its decision, before its first write,
// holding acme FOR NO KEY UPDATE and nothing under it (gatedDeleter). Each
// of bob's writes on Web (webWrites) waits for acme's row, its first lock;
// once the deletion commits, his share of acme reads no row. Every row
// under Web is then as the deletion left it: as it was before, deleted
// with Web.
func TestAWriteOnAProjectWhoseWorkspaceIsDeletedWhileItWaitsFindsNoProject(t *testing.T) {
	contract := apitest.Load(t)
	for _, w := range webWrites {
		t.Run(w.name, func(t *testing.T) {
			r := w.race(t)
			before := r.underWeb(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			g := newGate()
			route := newProjectRoute(t, r.pool, r.authorizer(), workspace.Provide(r.pool).WorkspaceMembers)
			deleted := run(func() error { return r.deleteAcme(ctx, gatedDeleter{workspacepg.New(r.pool), g}) })
			held(t, ctx, g, deleted, "the deletion")
			if got, want := "acme "+lockOn(t, r.pool, "workspaces WHERE slug = 'acme'")+", Web "+lockOn(t, r.pool, "projects WHERE id = $1", r.web),
				"acme FOR NO KEY UPDATE, Web no lock"; got != want {
				t.Errorf("the deletion at its gate holds %s; want %s", got, want)
			}
			var req *http.Request
			var rec *httptest.ResponseRecorder
			wrote := run(func() error {
				req, rec = w.send(route, r)
				return nil
			})
			pgtest.WaitForLockWaitOn(t, r.pool, "workspaces", 5*time.Second)
			close(g.open)

			if err := errors.Join(result(t, ctx, deleted, "the deletion"), result(t, ctx, wrote, "bob's write")); err != nil {
				t.Fatal(err)
			}
			contract.CheckResponse(t, req, rec.Result())
			r.foundNoProject(t, rec, before)
		})
	}
}
