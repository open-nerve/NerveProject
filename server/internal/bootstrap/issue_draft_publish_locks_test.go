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
			tx := requestTx(w.pool)
			var e exchange
			collect := requestBackground(t, draftWrite(t, w, tx, &e, row.method, publishPath(d.ID), ""), func() { requestRelease(t, held) })
			pid := requestPID(t, tx)
			requestTableBlocked(t, w.pool, tx, held.Conn().PgConn().PID(), wantTable)
			if got := requestWrittenTables(t, w.pool, pid); len(got) != 0 {
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
			requestRelease(t, held)
			collect()
			w.answered(t, e, row.want)
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
