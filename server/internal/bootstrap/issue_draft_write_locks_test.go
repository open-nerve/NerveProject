package bootstrap

import (
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The three explicit directories share contract coverage, not lock semantics:
// ordinary writes P(N), draft CRUD W(S)/P(S)/D(U), publication W(S)/P(N)/D(U).
// Publication is populated when its HTTP operation is registered in T6.
type draftWriteLock struct {
	op, method, path, workspace, project, draft string
	want                                        int
}

var draftCRUDWrites = []draftWriteLock{
	{"createDraft", "POST", "/api/v0/workspaces/{slug}/drafts", "FOR SHARE", "FOR SHARE", "INSERT", 201},
	{"updateDraft", "PATCH", "/api/v0/drafts/{draft_id}", "FOR SHARE", "FOR SHARE", "FOR UPDATE", 200},
	{"deleteDraft", "DELETE", "/api/v0/drafts/{draft_id}", "FOR SHARE", "FOR SHARE", "FOR UPDATE", 204},
}
var draftPublishWrites = []draftWriteLock{}

func checkIssueWriteDirectories(t *testing.T, contract *apitest.Contract) {
	t.Helper()
	var ops []string
	check := func(op, method, path string) {
		t.Helper()
		ops = append(ops, op)
		if !slices.ContainsFunc(contract.Operations(), func(o apitest.Operation) bool { return o.ID == op && o.Method == method && o.Path == path }) {
			t.Fatalf("%s sends %s %s outside its contract", op, method, path)
		}
	}
	for _, row := range issueWrites {
		check(row.op, row.method, row.path)
		if row.workspaceLock != "no lock" || row.projectLock != "FOR NO KEY UPDATE" {
			t.Fatalf("%s has invalid ordinary issue lock shape", row.op)
		}
	}
	for _, directory := range []struct {
		rows    []draftWriteLock
		project string
	}{{draftCRUDWrites, "FOR SHARE"}, {draftPublishWrites, "FOR NO KEY UPDATE"}} {
		for _, row := range directory.rows {
			check(row.op, row.method, row.path)
			wantDraft := "FOR UPDATE"
			if row.op == "createDraft" {
				wantDraft = "INSERT"
			}
			if row.workspace != "FOR SHARE" || row.project != directory.project || row.draft != wantDraft {
				t.Fatalf("%s has an invalid declared lock shape: %+v", row.op, row)
			}
		}
	}
	slices.Sort(ops)
	if want := writesOnIssues(contract.Operations()); !slices.Equal(ops, want) {
		t.Fatalf("ordinary issue / draft CRUD / draft publish union %q, want every tagged non-GET issue write %q", ops, want)
	}
}

// The same HTTP routes in the coverage union are observed while blocked at
// each lock boundary. Blocking the write statement after LockDraft proves
// exact D(U), independently of an UPDATE's implicit row lock.
func (row draftWriteLock) checkCRUDLocks(t *testing.T, w issueWorld) {
	t.Helper()
	for _, attached := range []bool{true, false} {
		t.Run(map[bool]string{true: "project", false: "unassigned"}[attached], func(t *testing.T) {
			for _, stage := range []string{"workspace", "project", "draft"} {
				if stage == "project" && !attached {
					continue
				}
				t.Run(stage, func(t *testing.T) {
					body := `{"payload":{"name":"Saved"}}`
					if attached {
						body = `{"project_id":"` + w.web.String() + `","payload":{"name":"Saved"}}`
					}
					path := draftsPath
					var id uuid.UUID
					if row.op != "createDraft" {
						d := createDraftHTTP(t, w, "alice", body)
						id = d.ID
						path = draftPath(id)
						body = `{"payload":{"name":"Changed"}}`
						if row.op == "deleteDraft" {
							body = ""
						}
					}
					// Prechecks read through EXCLUSIVE locks; the first row lock must be W.
					tables, mode := issueTables, "EXCLUSIVE"
					if stage == "project" {
						tables = "projects"
					}
					if stage == "draft" {
						tables, mode = "draft_issues", "SHARE"
					}
					held := lockedTables(t, w.pool, tables, mode)
					req := newRequest(t, row.method, w.base+path, w.tokens["alice"], []byte(body))
					w.contract.CheckRequest(t, req)
					pending := sendInBackground(req)
					pgtest.WaitForLockWaitBehind(t, w.pool, held.Conn().PgConn(), 10*time.Second)
					wantTable := "workspaces"
					if stage == "project" {
						wantTable = "projects"
					}
					if stage == "draft" {
						wantTable = "draft_issues"
					}
					if got := waitedTable(t, w.pool, held.Conn().PgConn()); got != wantTable {
						t.Errorf("%s waits at %s, want %s", row.op, got, wantTable)
					}
					if got := writtenTables(t, w.pool, held.Conn().PgConn()); len(got) != 0 {
						t.Errorf("%s already wrote %v at %s", row.op, got, stage)
					}
					if stage != "workspace" {
						if got := lockOn(t, w.pool, "workspaces WHERE id=$1", w.acme); got != row.workspace {
							t.Errorf("workspace lock=%s want %s", got, row.workspace)
						}
					}
					if stage == "draft" {
						wantProject := "no lock"
						if attached {
							wantProject = row.project
						}
						if got := lockOn(t, w.pool, "projects WHERE id=$1", w.web); got != wantProject {
							t.Errorf("project lock=%s want %s", got, wantProject)
						}
						if row.op != "createDraft" {
							if got := lockOn(t, w.pool, "draft_issues WHERE id=$1", id); got != row.draft {
								t.Errorf("draft lock=%s want %s", got, row.draft)
							}
						}
					}
					release(t, held)
					a := receiveWithin(t, pending, 10*time.Second, row.op)
					if a.err != nil {
						t.Fatal(a.err)
					}
					if a.res.StatusCode != row.want {
						t.Fatalf("%s = %d %s want %d", row.op, a.res.StatusCode, a.body, row.want)
					}
					w.contract.CheckResponse(t, req, a.res)
				})
			}
		})
	}
}
