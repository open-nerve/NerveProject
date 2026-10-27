package bootstrap

import (
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Every publication directory row proves its W(S)/P(N)/D(U) lock shape
// at actual PostgreSQL waits before any creation writes.
func (row draftWriteLock) checkPublishLocks(t *testing.T, w issueWorld) {
	for _, stage := range []string{"workspace", "project", "draft", "core"} {
		t.Run(stage, func(t *testing.T) {
			d := createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{"name":"Publish"}}`)
			tables, mode, wantTable := issueTables, "EXCLUSIVE", "workspaces"
			switch stage {
			case "project":
				tables, wantTable = "projects", "projects"
			case "draft":
				tables, wantTable = "draft_issues", "draft_issues"
			case "core":
				tables, mode, wantTable = "states", "ACCESS EXCLUSIVE", "states"
			}
			held := lockedTables(t, w.pool, tables, mode)
			req := newRequest(t, row.method, w.base+publishPath(d.ID), w.tokens["alice"], nil)
			w.contract.CheckRequest(t, req)
			pending := sendInBackground(req)
			pgtest.WaitForLockWaitBehind(t, w.pool, held.Conn().PgConn(), 10*time.Second)
			if got := waitedTable(t, w.pool, held.Conn().PgConn()); got != wantTable {
				t.Errorf("wait=%s want %s", got, wantTable)
			}
			if got := writtenTables(t, w.pool, held.Conn().PgConn()); len(got) != 0 {
				t.Errorf("wrote %v at %s", got, stage)
			}
			if stage != "workspace" {
				if got := lockOn(t, w.pool, "workspaces WHERE id=$1", w.acme); got != row.workspace {
					t.Errorf("W=%s", got)
				}
			}
			if stage == "draft" || stage == "core" {
				if got := lockOn(t, w.pool, "projects WHERE id=$1", w.web); got != row.project {
					t.Errorf("P=%s", got)
				}
			}
			if stage == "core" {
				if got := lockOn(t, w.pool, "draft_issues WHERE id=$1", d.ID); got != row.draft {
					t.Errorf("D=%s", got)
				}
			}
			release(t, held)
			answer := receiveWithin(t, pending, 10*time.Second, row.op)
			if answer.err != nil {
				t.Fatal(answer.err)
			}
			if answer.res.StatusCode != row.want {
				t.Fatalf("publish=%d %s", answer.res.StatusCode, answer.body)
			}
			w.contract.CheckResponse(t, req, answer.res)
		})
	}
}

// Publication must read the post-wait payload. PATCH already holds P(S) and
// is paused at D(U); publish prechecks Old, waits at P(N), then creates New.
func TestDraftPublishUsesPayloadAfterPatchWait(t *testing.T) {
	w := newIssueWorld(t)
	d := createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{"name":"Old"}}`)
	held := holding(t, w.pool, "SELECT 1 FROM draft_issues WHERE id=$1 FOR UPDATE", d.ID)
	patch := newRequest(t, "PATCH", w.base+draftPath(d.ID), w.tokens["alice"], []byte(`{"payload":{"name":"Locked new"}}`))
	patched := sendInBackground(patch)
	pgtest.WaitForLockWaitBehind(t, w.pool, held.Conn().PgConn(), 10*time.Second)
	req := newRequest(t, "POST", w.base+publishPath(d.ID), w.tokens["alice"], nil)
	published := sendInBackground(req)
	pgtest.WaitForLockWaitOn(t, w.pool, "projects", 10*time.Second)
	release(t, held)
	a := receiveWithin(t, patched, 10*time.Second, "patch")
	if a.err != nil || a.res.StatusCode != 200 {
		t.Fatal(a)
	}
	b := receiveWithin(t, published, 10*time.Second, "publish")
	if b.err != nil || b.res.StatusCode != 201 {
		t.Fatal(b)
	}
	w.contract.CheckResponse(t, patch, a.res)
	w.contract.CheckResponse(t, req, b.res)
	var got storedIssue
	decodeAnswer(t, string(b.body), &got)
	if got.Name != "Locked new" {
		t.Fatal(got)
	}
}
