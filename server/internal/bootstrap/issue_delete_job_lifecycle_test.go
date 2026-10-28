package bootstrap

import (
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func TestDeleteRowsRiverSnoozeDoesNotSpendAttempts(t *testing.T) {
	for _, offset := range []time.Duration{-61 * 24 * time.Hour, 61 * 24 * time.Hour} {
		t.Run(offset.String(), func(t *testing.T) {
			base := time.Now()
			var calls atomic.Int64
			f := newDeletionLifecycle(t, func(c *config.Config) { c.Clock.Offset = offset; c.Issue.DeleteBudget = 20 * time.Millisecond }, func() time.Time { return base.Add(time.Duration(calls.Add(1)) * time.Millisecond) }, nil)
			w := f.w
			var ids []uuid.UUID
			for i := 0; i < 45; i++ {
				ids = append(ids, createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{"name":"Budget"}}`).ID)
			}
			old := createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{}}`).ID
			w.send(t, "alice", "DELETE", draftPath(old), "", 204)
			key := foreignKey{table: "draft_issues"}
			before := key.asText(t, w.pool, []uuid.UUID{old})
			w.send(t, "alice", "DELETE", "/api/v0/projects/"+w.web.String(), "", 204)
			job := projectDeletionJob(t, w.pool, w.web)
			got := awaitObservedJob(t, w.pool, job, func(j observedJob) bool { return j.State == "completed" })
			if got.Attempt != 1 || got.Failures != 0 || got.Snoozes < 2 {
				t.Fatalf("not real multi-snooze: %+v", got)
			}
			attempts := make([]int, got.Snoozes+1)
			for i := range attempts {
				attempts[i] = 1
			}
			assertJobRuns(t, w.pool, job, attempts, "completed")
			snoozes := 0
			for _, tr := range jobTransitions(t, w.pool, job.ID) {
				if tr.From == "running" && (tr.To == "available" || tr.To == "scheduled") {
					snoozes++
					if tr.Attempt != 0 {
						t.Fatal("snooze spent attempt", tr)
					}
				}
			}
			if snoozes != got.Snoozes {
				t.Fatal("metadata/transition snoozes differ", snoozes, got.Snoozes)
			}
			assertDeletionAudit(t, w.pool, "draft_issues", ids, job)
			if key.asText(t, w.pool, []uuid.UUID{old}) != before {
				t.Fatal("snooze overwrote old audit")
			}
			committed := 0
			for _, run := range f.execution.snapshot() {
				if run.Err != nil {
					t.Fatal(run.Err)
				}
				committed += run.Result.Deleted
			}
			if committed != len(ids) {
				t.Fatalf("committed rows=%d want=%d", committed, len(ids))
			}
		})
	}
}

func TestDeleteRowsRiverRetriesAndExhausts(t *testing.T) {
	for _, last := range []bool{false, true} {
		name := "partial_commit_retry"
		if last {
			name = "last_attempt"
		}
		t.Run(name, func(t *testing.T) {
			f := newDeletionLifecycle(t, func(*config.Config) {}, time.Now, nil)
			w := f.w
			var ids []uuid.UUID
			for i := 0; i < 3; i++ {
				ids = append(ids, createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{}}`).ID)
			}
			w.send(t, "alice", "DELETE", draftPath(ids[2]), "", 204)
			key := foreignKey{table: "draft_issues"}
			old := key.asText(t, w.pool, ids[2:])
			// The real second batch violates a PostgreSQL CHECK, after batch one
			// committed. No Go error replacement or SQLSTATE fabrication is used.
			if _, err := w.pool.Exec(pgtest.Soon(t), `ALTER TABLE draft_issues ADD CONSTRAINT deletion_retry_fixture CHECK(id<>'`+ids[1].String()+`'::uuid OR deleted_at IS NULL) NOT VALID`); err != nil {
				t.Fatal(err)
			}
			setDeletionControl(t, w.pool, last, true)
			w.send(t, "alice", "DELETE", "/api/v0/projects/"+w.web.String(), "", 204)
			job := projectDeletionJob(t, w.pool, w.web)
			if last {
				tag, err := w.pool.Exec(pgtest.Soon(t), `UPDATE river_job SET attempt=24 WHERE id=$1 AND state='available' AND attempt=0`, job.ID)
				if err != nil || tag.RowsAffected() != 1 {
					t.Fatal(tag, err)
				}
				setDeletionControl(t, w.pool, false, true)
				releaseDeletionJob(t, w.pool, job, 24)
			}
			got := awaitObservedJob(t, w.pool, job, func(j observedJob) bool { return j.Failures == 1 && j.State != "running" })
			assertDeletionAudit(t, w.pool, "draft_issues", ids[:1], job)
			var live bool
			if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT deleted_at IS NULL FROM draft_issues WHERE id=$1`, ids[1]).Scan(&live); err != nil || !live {
				t.Fatal("failed batch committed", live, err)
			}
			failures := f.tx.failures()
			if len(failures) != 1 || failures[0].Inside != failures[0].Outside {
				t.Fatal("transaction changed original error", failures)
			}
			var pgerr *pgconn.PgError
			if !errors.As(failures[0].Outside, &pgerr) || pgerr.Code != "23514" || !strings.Contains(string(got.Errors), pgerr.ConstraintName) {
				t.Fatal("not original database constraint error", failures, got)
			}
			runs := f.execution.snapshot()
			if len(runs) != 1 || runs[0].Result.Deleted != 1 || runs[0].Err != failures[0].Outside {
				t.Fatal("partial result or original error lost", runs)
			}
			request := deletionWindowRequest{name: "failed deletion remains hidden", actor: "alice", method: "GET", path: "/api/v0/drafts/" + ids[1].String(), want: 404, absentPath: "/api/v0/drafts/" + uuid.NewV7().String()}
			before := request.checked(t, w)
			if last {
				if got.State != "discarded" || got.Attempt != 25 || got.Finalized == nil {
					t.Fatal("not real last attempt", got)
				}
				assertJobRuns(t, w.pool, job, []int{25}, "discarded")
				count := 0
				for _, line := range strings.Split(f.logs.String(), "\n") {
					var log map[string]any
					if json.Unmarshal([]byte(line), &log) != nil {
						continue
					}
					if log["msg"] == "issue row deletion exhausted" {
						count++
						if log["level"] != "ERROR" || log["job_id"] != float64(job.ID) || log["scope"] != job.Scope || log["id"] != job.Parent.String() || log["error"] != failures[0].Outside.Error() {
							t.Fatal("wrong exhausted log", log)
						}
					}
				}
				if count != 1 {
					t.Fatal("exhausted logs", count, f.logs.String())
				}
			} else {
				if got.Attempt != 1 || (got.State != "available" && got.State != "retryable") {
					t.Fatal("missing failed state", got)
				}
				if _, err := w.pool.Exec(pgtest.Soon(t), `ALTER TABLE draft_issues DROP CONSTRAINT deletion_retry_fixture`); err != nil {
					t.Fatal(err)
				}
				setDeletionControl(t, w.pool, false, false)
				releaseDeletionJob(t, w.pool, job, 1)
				got = awaitObservedJob(t, w.pool, job, func(j observedJob) bool { return j.State == "completed" })
				if got.Attempt != 2 || got.Failures != 1 {
					t.Fatal("not same job retry", got)
				}
				assertJobRuns(t, w.pool, job, []int{1, 2}, "completed")
				assertDeletionAudit(t, w.pool, "draft_issues", ids[:2], job)
				if after := request.checked(t, w); after != before {
					t.Fatal("retry changed hidden response")
				}
			}
			if key.asText(t, w.pool, ids[2:]) != old {
				t.Fatal("retry overwrote old audit")
			}
		})
	}
}
