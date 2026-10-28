package bootstrap

import (
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// These are real routes, locks and rows. Each subtree names the lost behavior
// it catches: lock mode/order, precheck isolation, terminal-state precedence,
// final-D clock, or resource-free admission to the publication gate.
func TestDraftWritesLockShape(t *testing.T) {
	t.Run("directory modes and nil project", func(t *testing.T) {
		w := newIssueWorld(t)
		checkIssueWriteDirectories(t, w.contract)
		for _, row := range draftCRUDWrites {
			t.Run(row.op, func(t *testing.T) { row.checkCRUDLocks(t, w) })
		}
		for _, row := range draftPublishWrites {
			t.Run(row.op, func(t *testing.T) { row.checkPublishLocks(t, w) })
		}
	})
	t.Run("refusals use no write resources", testDraftPrecheckResources)
	t.Run("workspace terminal states", testDraftWorkspaceTerminal)
	t.Run("P and D terminal states", testDraftPDTerminal)
	t.Run("clock follows final D", testDraftFinalClock)
	t.Run("gate holds no connection", testDraftGateConnection)
	t.Run("request blocking chain", testDraftWaitChain)
}

func testDraftFinalClock(t *testing.T) {
	for _, method := range []string{"PATCH", "DELETE", "POST"} {
		t.Run(method, func(t *testing.T) {
			w := newIssueWorld(t)
			d := createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{"name":"Clock"}}`)
			at := time.Date(2044, 5, 6, 7, 8, 9, 123456000, time.UTC)
			controlled := clockAt(at)
			clock := &eventCountingClock{base: controlled}
			tx := requestTx(w.pool)
			route := newIssueRouteWithClock(t, w.pool, tx, clock)
			path, body, want := draftPath(d.ID), "", 204
			if method == "PATCH" {
				body = `{"payload":{"name":"Later"}}`
				want = 200
			}
			if method == "POST" {
				path = publishPath(d.ID)
				want = 201
			}
			holder := holding(t, w.pool, "SELECT 1 FROM draft_issues WHERE id=$1 FOR UPDATE", d.ID)
			var e exchange
			collect := requestBackground(t, w.sending(&e, route, "alice", method, path, body), func() { requestRelease(t, holder) })
			requestBlocked(t, w.pool, tx, holder.Conn().PgConn().PID(), "draft_issues")
			if clock.calls.Load() != 0 {
				t.Errorf("Clock.Now called %d times before final D lock", clock.calls.Load())
			}
			controlled.advance(time.Hour)
			at = at.Add(time.Hour)
			requestRelease(t, holder)
			collect()
			raw := w.answered(t, e, want)
			if clock.calls.Load() != 1 {
				t.Fatal("expected sole lock-time clock", clock.calls.Load())
			}
			var updated time.Time
			var deleted *time.Time
			var by uuid.UUID
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT updated_at,deleted_at,updated_by_id FROM draft_issues WHERE id=$1", d.ID).Scan(&updated, &deleted, &by); err != nil {
				t.Fatal(err)
			}
			if !updated.Equal(at) || by != w.ids["alice"] {
				t.Fatal("draft retained pre-wait clock", updated, at, by)
			}
			if method != "PATCH" && (deleted == nil || !deleted.Equal(at)) {
				t.Fatal("consumption clock", deleted, at)
			}
			if method == "POST" {
				var issue storedIssue
				decodeAnswer(t, raw, &issue)
				if !issue.CreatedAt.Equal(at) || !issue.UpdatedAt.Equal(at) {
					t.Fatal("issue retained pre-D clock", issue)
				}
				for _, table := range []string{"issue_activities", "issue_subscribers"} {
					var n, stamped int
					if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*),count(*) FILTER(WHERE created_at=$2 AND updated_at=$2 AND created_by_id=$3 AND updated_by_id=$3) FROM "+table+" WHERE issue_id=$1", issue.ID, at, w.ids["alice"]).Scan(&n, &stamped); err != nil || n != 1 || stamped != n {
						t.Fatal(table, n, stamped, err)
					}
				}
			}
		})
	}
}

func draftSameState(t *testing.T, w issueWorld, before map[string]string) {
	t.Helper()
	if !reflect.DeepEqual(before, publishSnapshot(t, w)) {
		t.Fatal("refused request changed draft, issue, project counter or queued jobs")
	}
}

// The publication's project wait is caused by this PATCH request, which in
// turn waits for the independently identified D holder. A worker cannot satisfy
// either edge, and publication must consume the PATCH's committed payload.
func testDraftWaitChain(t *testing.T) {
	w := newIssueWorld(t)
	d := createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{"name":"Old"}}`)
	holder := holding(t, w.pool, "SELECT 1 FROM draft_issues WHERE id=$1 FOR UPDATE", d.ID)
	patchTx, publishTx := requestTx(w.pool), requestTx(w.pool)
	var patch, publish exchange
	patched := requestBackground(t, draftWrite(t, w, patchTx, &patch, "PATCH", draftPath(d.ID), `{"payload":{"name":"Chain new"}}`), func() { requestRelease(t, holder) })
	requestBlocked(t, w.pool, patchTx, holder.Conn().PgConn().PID(), "draft_issues")
	published := requestBackground(t, draftWrite(t, w, publishTx, &publish, "POST", publishPath(d.ID), ""), func() { requestRelease(t, holder) })
	requestBlocked(t, w.pool, publishTx, requestPID(t, patchTx), "projects")
	requestBlocked(t, w.pool, patchTx, holder.Conn().PgConn().PID(), "draft_issues")
	requestRelease(t, holder)
	patched()
	published()
	w.answered(t, patch, 200)
	var issue storedIssue
	decodeAnswer(t, w.answered(t, publish, 201), &issue)
	if issue.Name != "Chain new" || issue.ProjectID != w.web || issue.SequenceID != 1 {
		t.Fatal(issue)
	}
	var deleted time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM draft_issues WHERE id=$1", d.ID).Scan(&deleted); err != nil || !deleted.Equal(issue.CreatedAt) {
		t.Fatal(deleted, err)
	}
}
