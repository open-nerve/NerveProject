package bootstrap

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestDeletionInterleaving24(t *testing.T) {
	t.Run("all-skipped_then_mixed_then_all-skipped", testDeletionCandidateProgress)
	for _, reverse := range []bool{false, true} {
		for _, liveFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("reverseUUID=%t/liveFirst=%t", reverse, liveFirst), func(t *testing.T) {
				w := newIssueWorldWith(t, func(c *config.Config) {
					u, err := url.Parse(c.Database.URL)
					if err != nil {
						t.Fatal(err)
					}
					q := u.Query()
					q.Set("enable_indexscan", "off")
					q.Set("enable_bitmapscan", "off")
					q.Set("enable_hashjoin", "off")
					q.Set("enable_mergejoin", "off")
					u.RawQuery = q.Encode()
					c.Database.URL = u.String()
				})
				p, q := w.web, createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Other", "OTHER")
				w.makeMember(t, "bob", q, shared.RoleMember)
				if reverse {
					p, q = q, p
				}
				x := w.newWorkItemIn(t, p, `{"name":"X"}`)
				x2 := w.newWorkItemIn(t, p, `{"name":"X2"}`)
				y := w.newWorkItemBy(t, "bob", q, `{"name":"Y"}`)
				ids := []uuid.UUID{
					listIn[answeredRelation](t, w.send(t, "alice", "POST", relationsOf(x), relating("duplicate", y), 201))[0].ID,
					listIn[answeredRelation](t, w.send(t, "alice", "POST", relationsOf(y), relating("blocked_by", x2), 201))[0].ID,
				}
				// Move the first row's physical tuple behind the second without changing
				// any business value: heap order must not substitute for relation.id.
				reorderRelationTuple(t, w, ids[0])
				probe := requestTx(w.pool)
				g := newGate()
				var reply exchange
				remove := func(tx shared.TxManager) func() error {
					return w.sending(&reply, newIssueRoute(t, w.pool, tx), "bob", "DELETE", "/api/v0/issues/"+y.String(), "")
				}
				var done <-chan error
				defer func() { releaseAndJoinGate(t, g, done) }()
				var second pgx.Tx
				var joinNeighbor func()
				if liveFirst {
					done = run(remove(heldTx{probe, g}))
					held(t, pgtest.Soon(t), g, done, "live issue deletion")
					// A periodic archive read can wait on the same transaction's
					// project lock. Observe the relation worker without assuming
					// that it is this transaction's only blocked backend.
					neighbor, err := w.pool.Acquire(pgtest.Soon(t))
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(neighbor.Release)
					neighborCtx, cancelNeighbor := context.WithTimeout(context.Background(), 20*time.Second)
					t.Cleanup(cancelNeighbor)
					joinNeighbor = requestBackground(t, func() error {
						_, err := neighbor.Exec(neighborCtx, "SELECT 1 FROM projects WHERE id=$1 FOR NO KEY UPDATE", q)
						return err
					}, func() { requestOpen(g) })
					observer := &requestProbe{}
					observer.pid.Store(neighbor.Conn().PgConn().PID())
					requestBlocked(t, w.pool, observer, requestPID(t, probe), "projects")
				} else {
					second = holding(t, w.pool, `SELECT 1 FROM issue_relations WHERE id=$1 FOR UPDATE`, ids[1])
					defer requestRelease(t, second)
				}
				w.send(t, "alice", "DELETE", "/api/v0/projects/"+p.String(), "", 204)
				job := projectDeletionJob(t, w.pool, p)
				if liveFirst {
					deletionWorkerBlockedBy(t, w.pool, requestPID(t, probe), job, "issue_relations")
					requestOpen(g)
					if err := result(t, pgtest.Soon(t), done, "live deletion"); err != nil {
						t.Fatal(err)
					}
					done = nil
					joinNeighbor()
				} else {
					pid := deletionWorkerBlocked(t, w.pool, second, job, "issue_relations")
					logRelationLockPlan(t, w, pid, p)
					owner := &requestProbe{}
					owner.pid.Store(pid)
					if mode := requestRowLock(t, w.pool, w.pool, owner, "issue_relations WHERE id=$1", ids[0]); mode != "FOR NO KEY UPDATE" {
						t.Fatal("worker did not lock lower relation first", mode)
					}
					if !pgtest.RowWaits(t, w.pool, "issue_relations", ids[0], "FOR UPDATE") {
						t.Fatal("relation row not held")
					}
					join := requestBackground(t, remove(probe), func() { requestRelease(t, second) })
					requestBlocked(t, w.pool, probe, pid, "issue_relations")
					requestRelease(t, second)
					join()
				}
				w.answered(t, reply, 204)
				waitDeletionJob(t, w.pool, job, "completed")
				stamp := job
				if liveFirst {
					stamp.At = w.deletedAt(t, "issues", y)
					stamp.By = w.ids["bob"]
				}
				assertDeletionAudit(t, w.pool, "issue_relations", ids, stamp)
				assertDeletionAudit(t, w.pool, "issues", []uuid.UUID{x, x2}, job)
			})
		}
	}
}

// A non-HOT update gives the legal heap plan a deliberately different order.
// Both statements commit together with the original ID and every audit value.
func reorderRelationTuple(t *testing.T, w issueWorld, id uuid.UUID) {
	t.Helper()
	key := foreignKey{table: "issue_relations"}
	before := key.asText(t, w.pool, []uuid.UUID{id})
	allBefore := registrationRows(t, w.pool)
	tx, err := w.pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer requestRelease(t, tx)
	temporary := uuid.NewV7()
	for _, pair := range [][2]uuid.UUID{{id, temporary}, {temporary, id}} {
		if _, err := tx.Exec(pgtest.Soon(t), `UPDATE issue_relations SET id=$2 WHERE id=$1`, pair[0], pair[1]); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
	if key.asText(t, w.pool, []uuid.UUID{id}) != before || registrationRows(t, w.pool) != allBefore {
		t.Fatal("physical layout fixture changed relation values")
	}
	var layout string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT string_agg(id::text||'@'||ctid::text,',' ORDER BY ctid) FROM issue_relations`).Scan(&layout); err != nil {
		t.Fatal(err)
	}
	heap := queryIDs(t, w.pool, `SELECT id FROM issue_relations ORDER BY ctid`)
	if len(heap) != 2 || heap[0] == id || heap[1] != id {
		t.Fatal("fixture did not reverse heap order", layout)
	}
	t.Logf("unchanged relation values; actual heap layout=%s", layout)
}

// EXPLAIN the observed statement with the worker's test planner settings and
// actual arguments. No ANALYZE: this observation never takes its row locks.
func logRelationLockPlan(t *testing.T, w issueWorld, pid uint32, p uuid.UUID) {
	t.Helper()
	var query string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT query FROM pg_stat_activity WHERE pid=$1`, pid).Scan(&query); err != nil {
		t.Fatal(err)
	}
	tx, err := w.pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer requestRelease(t, tx)
	if _, err := tx.Exec(pgtest.Soon(t), `SET LOCAL enable_indexscan=off; SET LOCAL enable_bitmapscan=off; SET LOCAL enable_hashjoin=off; SET LOCAL enable_mergejoin=off`); err != nil {
		t.Fatal(err)
	}
	rows, err := tx.Query(pgtest.Soon(t), "EXPLAIN (COSTS OFF) "+query, p, uuid.UUID{}, 1000)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("observed worker statement, matching planner settings EXPLAIN: %v", plan)
}

func testDeletionCandidateProgress(t *testing.T) {
	w := newIssueWorldWith(t, func(c *config.Config) { c.Issue.DeleteBatch = 2 })
	old := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Old", "OLD")
	createDraftHTTP(t, w, "alice", `{"project_id":"`+old.String()+`","payload":{}}`)
	x := w.newWorkItem(t)
	var skipped []uuid.UUID
	var owned uuid.UUID
	for i := 0; i < 6; i++ {
		p := old
		if i == 3 {
			p = w.web
		}
		y := w.newWorkItemIn(t, p, `{"name":"Candidate"}`)
		from, to := x, y
		if i%2 == 1 {
			from, to = to, from
		}
		id := listIn[answeredRelation](t, w.send(t, "alice", "POST", relationsOf(from), relating("relates_to", to), 201))[0].ID
		if i == 3 {
			owned = id
		} else {
			skipped = append(skipped, id)
		}
	}
	holder := holdProjectDeletion(t, w.pool, old)
	w.send(t, "alice", "DELETE", "/api/v0/projects/"+old.String(), "", 204)
	oldJob := projectDeletionJob(t, w.pool, old)
	deletionWorkerBlocked(t, w.pool, holder, oldJob, "draft_issues")
	w.send(t, "alice", "DELETE", "/api/v0/projects/"+w.web.String(), "", 204)
	job := projectDeletionJob(t, w.pool, w.web)
	waitDeletionJob(t, w.pool, job, "completed")
	assertDeletionAudit(t, w.pool, "issue_relations", []uuid.UUID{owned}, job)
	var live int
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*) FROM issue_relations WHERE id=ANY($1) AND deleted_at IS NULL`, skipped).Scan(&live); err != nil || live != len(skipped) {
		t.Fatal("later job consumed Old's candidates", live, err)
	}
	requestRelease(t, holder)
	waitDeletionJob(t, w.pool, oldJob, "completed")
	assertDeletionAudit(t, w.pool, "issue_relations", skipped, oldJob)
}

func TestDeletionInterleaving25(t *testing.T) {
	for _, oldFirst := range []bool{false, true} {
		t.Run(fmt.Sprintf("OldCompletesFirst=%t", oldFirst), func(t *testing.T) {
			w := newIssueWorldWith(t, func(c *config.Config) { c.Issue.DeleteBatch = 2 })
			old := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Old", "OLD")
			q := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Q", "Q")
			for _, p := range []uuid.UUID{old, w.web, q} {
				createDraftHTTP(t, w, "alice", `{"project_id":"`+p.String()+`","payload":{}}`)
			}
			loose := createDraftHTTP(t, w, "alice", `{"payload":{}}`).ID
			x := w.newWorkItem(t)
			y := w.newWorkItemIn(t, q, `{"name":"Q"}`)
			var skipped, sharedIDs []uuid.UUID
			var oldItem uuid.UUID
			makeOld := func() {
				z := w.newWorkItemIn(t, old, `{"name":"Old"}`)
				oldItem = z
				skipped = append(skipped, listIn[answeredRelation](t, w.send(t, "alice", "POST", relationsOf(x), relating("relates_to", z), 201))[0].ID)
			}
			makeOld()
			makeOld()
			for i, kind := range []string{"duplicate", "blocked_by"} {
				if i > 0 {
					y = w.newWorkItemIn(t, q, `{"name":"Q2"}`)
				}
				sharedIDs = append(sharedIDs, listIn[answeredRelation](t, w.send(t, "alice", "POST", relationsOf(x), relating(kind, y), 201))[0].ID)
			}
			makeOld() // The final all-skipped batch must also terminate.
			for _, item := range []struct{ project, issue uuid.UUID }{{old, oldItem}, {w.web, x}, {q, y}} {
				furnishDeletionIssue(t, w, item.project, item.issue)
			}
			keys := keysTo(t, w.pool, "projects")
			catalog := map[uuid.UUID]map[foreignKey][]uuid.UUID{}
			for _, p := range []uuid.UUID{old, w.web, q} {
				catalog[p] = map[foreignKey][]uuid.UUID{}
				for _, k := range keys {
					catalog[p][k] = k.undeleted(t, w.pool, p)
					if projectIssueTable(k.table) && k.table != "issue_relations" && len(catalog[p][k]) == 0 {
						t.Fatalf("fixture lacks %s under %s", k.table, p)
					}
				}
			}
			oldHolder := holdProjectDeletion(t, w.pool, old)
			w.send(t, "alice", "DELETE", "/api/v0/projects/"+old.String(), "", 204)
			oldJob := projectDeletionJob(t, w.pool, old)
			deletionWorkerBlocked(t, w.pool, oldHolder, oldJob, "draft_issues")
			qHolder := holdProjectDeletion(t, w.pool, q)
			lastRelation := holding(t, w.pool, `SELECT 1 FROM issue_relations WHERE id=$1 FOR UPDATE`, sharedIDs[1])
			w.send(t, "wanda", "DELETE", "/api/v0/workspaces/acme", "", 204)
			if got := projectDeletionJob(t, w.pool, old); got != oldJob {
				t.Fatal("workspace replaced Old's job or request audit", got, oldJob)
			}
			pj, qj := projectDeletionJob(t, w.pool, w.web), projectDeletionJob(t, w.pool, q)
			wj := parentDeletionJob(t, w.pool, "workspace_drafts", w.acme)
			if pj.At != qj.At || pj.By != qj.By || pj.By == oldJob.By || !oldJob.At.Before(pj.At) {
				t.Fatal("fixture lacks distinct ownership", oldJob, pj, qj)
			}
			pPID := deletionWorkerBlocked(t, w.pool, lastRelation, pj, "issue_relations")
			pProbe := &requestProbe{}
			pProbe.pid.Store(pPID)
			if mode := requestRowLock(t, w.pool, w.pool, pProbe, "issue_relations WHERE id=$1", sharedIDs[0]); mode != "FOR NO KEY UPDATE" {
				t.Fatal("zero-written low batch stopped before high batch", mode)
			}
			deletionWorkerBlocked(t, w.pool, qHolder, qj, "draft_issues")
			requestRelease(t, qHolder)
			deletionWorkerBlockedBy(t, w.pool, pPID, qj, "issue_relations")
			if oldFirst {
				// Old owns only skipped relations and can finish while P/Q compete.
				requestRelease(t, oldHolder)
				waitDeletionJob(t, w.pool, oldJob, "completed")
			}
			requestRelease(t, lastRelation)
			waitDeletionJob(t, w.pool, pj, "completed")
			waitDeletionJob(t, w.pool, qj, "completed")
			waitDeletionJob(t, w.pool, wj, "completed")
			if !oldFirst {
				requestRelease(t, oldHolder)
				waitDeletionJob(t, w.pool, oldJob, "completed")
			}
			assertDeletionAudit(t, w.pool, "issue_relations", skipped, oldJob)
			assertDeletionAudit(t, w.pool, "issue_relations", sharedIDs, pj)
			assertDeletionAudit(t, w.pool, "draft_issues", []uuid.UUID{loose}, wj)
			for _, p := range []uuid.UUID{old, w.web, q} {
				job := pj
				if p == old {
					job = oldJob
				}
				for k, ids := range catalog[p] {
					// Cross-project relations have their own earlier-owner rule.
					if k.table == "issue_relations" {
						continue
					}
					if len(ids) > 0 {
						assertDeletionAudit(t, w.pool, k.table, ids, job)
					}
				}
			}
			before := registrationRows(t, w.pool)
			// A direct repeat is an idempotency check only, not a River retry.
			uc := newDeletionRowsForTest(w)
			for _, j := range []deletionJob{oldJob, pj, qj, wj} {
				r, err := uc.Execute(context.Background(), j.Scope, j.Parent, j.At, j.By)
				if err != nil || !r.Complete || r.Deleted != 0 {
					t.Fatal("repeat changed deletion", r, err)
				}
			}
			if registrationRows(t, w.pool) != before {
				t.Fatal("repeat overwrote old rows")
			}
		})
	}
}

// Fill every issue-owned table without adding relations that would change the
// ordered candidate fixture. Links use the existing catalog seed; the remaining
// rows are created through the real authenticated HTTP handlers.
func furnishDeletionIssue(t *testing.T, w issueWorld, projectID, issueID uuid.UUID) {
	t.Helper()
	var label struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", "POST", "/api/v0/projects/"+projectID.String()+"/labels", `{"name":"Audit coverage"}`, 201), &label)
	w.send(t, "alice", "PATCH", "/api/v0/issues/"+issueID.String(), fmt.Sprintf(`{"assignee_ids":["%s"],"label_ids":["%s"]}`, w.ids["alice"], label.ID), 200)
	patchDescription(t, w, "alice", issueID, mentionHTML(w.ids["alice"]), false)
	seedIssueLinks(t, w.pool, issueID, w.ids["alice"])
	w.send(t, "alice", "POST", reactionsOf(issueID), `{"reaction":"128077"}`, 201)
	comment := w.newComment(t, "alice", issueID)
	addCommentReaction(t, w, "alice", comment.ID, "128077")
}

// Ordinary paths retain the production wall-clock budget and raw JobTx.
func newDeletionRowsForTest(w issueWorld) *issueapp.DeleteRows {
	store := issuepg.New(w.pool)
	cfg := config.Config{Database: config.DatabaseConfig{CommitTimeout: 2 * time.Second}, Issue: testIssueConfig}
	return issueapp.NewDeleteRows(issueapp.DeleteRowsDeps{Rows: store, Relations: issueapp.NewRelationDeletions(store, issueProjects{projects: project.Provide(w.pool).IssueProjects}), Remaining: store, Tx: newIssueJobTx(w.pool, cfg), RuntimeNow: time.Now, Batch: 2, Budget: time.Second})
}
