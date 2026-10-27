package bootstrap

import (
	"context"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Missing P serialization would permit publication after project deletion;
// missing post-lock existence checking would expose project404 or create rows.
func TestDraftPublishAndProjectDeletionSerialize(t *testing.T) {
	for _, publishFirst := range []bool{true, false} {
		t.Run(map[bool]string{true: "publish first", false: "delete first"}[publishFirst], func(t *testing.T) {
			w := newIssueWorld(t)
			d := createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{"name":"Serialized"}}`)
			before := collaborationState(t, w)
			var publication, deletion exchange
			if publishFirst {
				draftSerialize(t, w, draftPublication(d.ID), deletesWeb, &publication, &deletion, "projects", func() {
					if got := lockOn(t, w.pool, "draft_issues WHERE id=$1", d.ID); got != "FOR UPDATE" {
						t.Fatal("publication has not locked D", got)
					}
				})
			} else {
				draftSerialize(t, w, deletesWeb, draftPublication(d.ID), &deletion, &publication, "projects", nil)
			}
			w.answered(t, deletion, 204)
			var projectDeleted *time.Time
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM projects WHERE id=$1", w.web).Scan(&projectDeleted); err != nil || projectDeleted == nil {
				t.Fatal(projectDeleted, err)
			}
			if !publishFirst {
				w.refused(t, publication, 404, "issue.draft_not_found")
				if !reflect.DeepEqual(before, collaborationState(t, w)) {
					t.Fatal("refused publication created issue effects")
				}
				var at time.Time
				var by uuid.UUID
				if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at,updated_by_id FROM draft_issues WHERE id=$1", d.ID).Scan(&at, &by); err != nil || !at.Equal(*projectDeleted) || by != w.ids["alice"] {
					t.Fatal("draft must be consumed by deletion, not publication", at, by, err)
				}
				var sequence int
				if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT last_issue_sequence FROM projects WHERE id=$1", w.web).Scan(&sequence); err != nil || sequence != 0 {
					t.Fatal(sequence, err)
				}
				return
			}
			var issue storedIssue
			decodeAnswer(t, w.answered(t, publication, 201), &issue)
			if issue.ProjectID != w.web || issue.SequenceID != 1 || issue.Name != "Serialized" {
				t.Fatal(issue)
			}
			var consumed, updated time.Time
			var by uuid.UUID
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at,updated_at,updated_by_id FROM draft_issues WHERE id=$1", d.ID).Scan(&consumed, &updated, &by); err != nil {
				t.Fatal(err)
			}
			if !consumed.Equal(issue.CreatedAt) || !updated.Equal(issue.CreatedAt) || by != w.ids["alice"] || projectDeleted.Before(consumed) {
				t.Fatal("draft/creation/deletion ordering", consumed, updated, issue.CreatedAt, projectDeleted, by)
			}
			var all, gone int
			if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*), count(*) FILTER(WHERE deleted_at=$2) FROM (
    SELECT deleted_at FROM issues WHERE project_id=$1 UNION ALL
    SELECT deleted_at FROM issue_subscribers WHERE project_id=$1 UNION ALL
    SELECT deleted_at FROM issue_activities WHERE project_id=$1) r`, w.web, *projectDeleted).Scan(&all, &gone); err != nil || all != 3 || gone != 3 {
				t.Fatal("creation effects must be deleted with project", all, gone, err)
			}
		})
	}
	t.Run("P9 window keeps draft live", func(t *testing.T) {
		w := newIssueWorld(t)
		d := createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{"name":"Never"}}`)
		before := draftState(t, w)
		holder := holding(t, w.pool, "SELECT 1 FROM projects WHERE id=$1 FOR NO KEY UPDATE", w.web)
		draftExec(t, holder, "UPDATE projects SET deleted_at=now() WHERE id=$1", w.web)
		tx := draftTx(w.pool)
		var e exchange
		collect := draftBackground(t, draftWrite(t, w, tx, &e, "POST", publishPath(d.ID), ""), func() { draftRelease(t, holder) })
		draftBlocked(t, w.pool, draftPID(t, tx), holder.Conn().PgConn().PID(), "projects")
		draftCommit(t, holder)
		collect()
		w.refused(t, e, 404, "issue.draft_not_found")
		if !reflect.DeepEqual(before, draftState(t, w)) {
			t.Fatal("live-draft deletion window changed draft or issue rows")
		}
	})
}
func assertDraftLive(t *testing.T, w issueWorld, id, project uuid.UUID) {
	t.Helper()
	var alive bool
	var got uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at IS NULL,project_id FROM draft_issues WHERE id=$1", id).Scan(&alive, &got); err != nil || !alive || got != project {
		t.Fatal("draft final state", alive, got, err)
	}
}

// Removing the initial/final project comparison publishes under a stale P1
// grant after a committed move. Taking D before P would block the sorted-lock
// probes, while reversing P order immediately names the wrong holder PID.
func TestDraftProjectMoveAndPublishSerialize(t *testing.T) {
	for _, originHigh := range []bool{false, true} {
		t.Run(map[bool]string{false: "P1 low", true: "P1 high"}[originHigh], func(t *testing.T) {
			w := newIssueWorld(t)
			other := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
			low, high := w.web, other
			if low.Compare(high) > 0 {
				low, high = high, low
			}
			if low.Compare(high) >= 0 {
				t.Fatal("fixture UUIDs are not distinct ascending", low, high)
			}
			p1, p2 := low, high
			if originHigh {
				p1, p2 = high, low
			}
			t.Logf("initialP1=%s targetP2=%s low=%s high=%s", p1, p2, low, high)
			t.Run("move commits after publish precheck", func(t *testing.T) {
				d := createDraftHTTP(t, w, "alice", `{"project_id":"`+p1.String()+`","payload":{"name":"Stale"}}`)
				g := newGate()
				tx := draftTx(w.pool)
				tx.before = g
				var e exchange
				collect := draftBackground(t, draftWrite(t, w, tx, &e, "POST", publishPath(d.ID), ""), func() { draftOpen(g) })
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				select {
				case <-g.held:
				case <-ctx.Done():
					t.Fatal("publish did not finish its precheck")
				}
				draftNoLock(t, w, "projects", p1)
				draftNoLock(t, w, "draft_issues", d.ID)
				draftRequest(t, w, "alice", "PATCH", draftPath(d.ID), `{"project_id":"`+p2.String()+`"}`, 200)
				before := publishSnapshot(t, w)
				draftOpen(g)
				collect()
				w.refused(t, e, 409, "issue.draft_changed")
				if !reflect.DeepEqual(before, publishSnapshot(t, w)) {
					t.Fatal("stale publication changed draft/project/issue/job state")
				}
				assertDraftLive(t, w, d.ID, p2)
			})
			t.Run("publication owns D before move", func(t *testing.T) {
				d := createDraftHTTP(t, w, "alice", `{"project_id":"`+p1.String()+`","payload":{"name":"Published"}}`)
				var published, moved exchange
				draftSerialize(t, w, draftPublication(d.ID), draftMove(d.ID, p2), &published, &moved, "projects", func() {
					if got := lockOn(t, w.pool, "draft_issues WHERE id=$1", d.ID); got != "FOR UPDATE" {
						t.Fatal("missing publication D lock", got)
					}
				})
				var issue storedIssue
				decodeAnswer(t, w.answered(t, published, 201), &issue)
				w.refused(t, moved, 404, "issue.draft_not_found")
				if issue.ProjectID != p1 || issue.Name != "Published" {
					t.Fatal(issue)
				}
				var original uuid.UUID
				var consumed time.Time
				if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT project_id,deleted_at FROM draft_issues WHERE id=$1", d.ID).Scan(&original, &consumed); err != nil || original != p1 || !consumed.Equal(issue.CreatedAt) {
					t.Fatal(original, consumed, err)
				}
			})
			t.Run("move locks old and new ascending before D", func(t *testing.T) {
				d := createDraftHTTP(t, w, "alice", `{"project_id":"`+p1.String()+`","payload":{"name":"Moving"}}`)
				lo := holding(t, w.pool, "SELECT 1 FROM projects WHERE id=$1 FOR UPDATE", low)
				hi := holding(t, w.pool, "SELECT 1 FROM projects WHERE id=$1 FOR UPDATE", high)
				tx := draftTx(w.pool)
				var e exchange
				collect := draftBackground(t, draftMove(d.ID, p2)(t, w, tx, &e), func() { draftRelease(t, lo); draftRelease(t, hi) })
				pid := draftPID(t, tx)
				draftBlocked(t, w.pool, pid, lo.Conn().PgConn().PID(), "projects")
				draftNoLock(t, w, "draft_issues", d.ID)
				draftRelease(t, lo)
				draftBlocked(t, w.pool, pid, hi.Conn().PgConn().PID(), "projects")
				if got := lockOn(t, w.pool, "projects WHERE id=$1", low); got != "FOR SHARE" {
					t.Fatal("low P must be SHARE before high P", got)
				}
				draftNoLock(t, w, "draft_issues", d.ID)
				draftRelease(t, hi)
				collect()
				w.answered(t, e, 200)
				assertDraftLive(t, w, d.ID, p2)
			})
		})
	}
}
