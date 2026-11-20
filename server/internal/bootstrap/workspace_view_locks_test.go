package bootstrap

import (
	"context"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Each write on a workspace's own view, updateView and deleteView of a
// view without a project, takes its workspace's row FOR SHARE first, in
// its transaction, then the view's row FOR UPDATE, and no project's row
// (M7 P3b spec 2.6, re-90 I7): TestEachWriteOnAProjectSharesItsWorkspaceFirst
// classes the two operations by their id (writesOnAProject) and its rows
// prove a project's view's branch alone. alice, acme's admin, makes
// acme's project Web and a view of acme's own for each write in each
// phase; once the start-time jobs have run (waitForStartupRuns), and with
// no fan-out pending (waitForFanouts), each write:
//   - Waits for its workspace first: another transaction holds acme's row
//     FOR NO KEY UPDATE, as a change of a role in it or its deletion does.
//     The write waits for that row, and meanwhile holds no lock on the
//     view's row: a FOR UPDATE NOWAIT of it succeeds.
//   - Takes the view's row after its workspace: another transaction holds
//     the view's row FOR UPDATE. The write waits for it, and meanwhile
//     holds acme's row at FOR SHARE, no stronger, which it took in its
//     transaction before (lockOn), and no lock on Web's row.
//
// Once the other transaction ends, the write answers as it would alone.
// Each write in each phase is a subtest, which names it when it fails; the
// first that fails ends the test, as a write left waiting could hold what
// the next one's probes look at.
func TestEachWriteOnAWorkspaceViewSharesItsWorkspaceFirst(t *testing.T) {
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	requests := &requestQueries{}
	base := startObservedApp(t, testConfig(t, dbURL, false), requests)
	pool := openPool(t, dbURL)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	var acme struct {
		ID uuid.UUID `json:"id"`
	}
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice, `{"name":"Acme","slug":"acme"}`)
	if status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	decodeAnswer(t, body, &acme)
	web := createdProject(t, contract, base, alice, "acme", "Web", "WEB")
	writes := []struct {
		op, method, body string
		want             int
	}{
		{"updateView", http.MethodPatch, `{"name":"Checked"}`, http.StatusOK},
		{"deleteView", http.MethodDelete, "", http.StatusNoContent},
	}
	phases := [2]string{"its workspace first", "its row after its workspace"}
	views := map[string]uuid.UUID{}
	for _, w := range writes {
		for _, phase := range phases {
			var view struct {
				ID uuid.UUID `json:"id"`
			}
			status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/acme/views", alice, `{"name":"`+w.op+`, `+phase+`"}`)
			if status != http.StatusCreated {
				t.Fatalf("creating the view for %s, %s = %d %s", w.op, phase, status, body)
			}
			decodeAnswer(t, body, &view)
			views[w.op+", "+phase] = view.ID
		}
	}
	waitForStartupRuns(t, pool)
	viewLock := "SELECT 1 FROM issue_views WHERE id = $1 FOR UPDATE"
	for _, w := range writes {
		for phase, name := range phases {
			if !t.Run(w.op+", "+name, func(t *testing.T) {
				view := views[w.op+", "+name]
				req := newRequest(t, w.method, base+"/api/v0/views/"+view.String(), alice, []byte(w.body))
				contract.CheckRequest(t, req)
				waitForFanouts(t, pool)
				var other pgx.Tx
				if phase == 0 {
					other = holding(t, pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", acme.ID)
				} else {
					other = holding(t, pool, viewLock, view)
				}
				probe := requests.observe(req)
				answered := sendInBackground(req)
				if phase == 0 {
					requestBlocked(t, pool, probe, other.Conn().PgConn().PID(), "workspaces")
					if heldBy(t, pool, viewLock, view) {
						t.Errorf("%s holds the view's row while it waits for its workspace", w.op)
					}
				} else {
					requestBlocked(t, pool, probe, other.Conn().PgConn().PID(), "issue_views")
					switch lock := lockOn(t, pool, "workspaces WHERE id = $1", acme.ID); lock {
					case "FOR SHARE":
					case "no lock", "FOR KEY SHARE":
						t.Errorf("%s does not hold its workspace FOR SHARE in its transaction while it waits for the view's row: %s", w.op, lock)
					default:
						t.Errorf("%s holds its workspace %s, stronger than FOR SHARE, while it waits for the view's row", w.op, lock)
					}
					if lock := lockOn(t, pool, "projects WHERE id = $1", web); lock != "no lock" {
						t.Errorf("%s holds Web's row %s, though its view is of no project", w.op, lock)
					}
				}
				if err := other.Rollback(context.Background()); err != nil {
					t.Fatal(err)
				}
				a := receiveWithin(t, answered, 10*time.Second, "the answer to "+w.op)
				if a.err != nil {
					t.Fatal(a.err)
				}
				contract.CheckResponse(t, req, a.res)
				if a.res.StatusCode != w.want {
					t.Errorf("%s, %s = %d %s, want %d", w.op, name, a.res.StatusCode, a.body, w.want)
				}
			}) {
				t.FailNow()
			}
		}
	}
}
