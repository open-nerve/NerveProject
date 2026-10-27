package bootstrap

import (
	"context"
	"encoding/json"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Only the issue-owned portion is asynchronous; unknown catalog tables still
// participate in the immediate assertions rather than receiving an exemption.
func projectIssueTable(table string) bool {
	return table == "draft_issues" || table == "issues" || table == "comment_reactions" || len(table) > 6 && table[:6] == "issue_"
}

type deletionJob struct {
	ID     int64     `json:"-"`
	Scope  string    `json:"scope"`
	Parent uuid.UUID `json:"id"`
	At     time.Time `json:"at"`
	By     uuid.UUID `json:"by"`
}

func projectDeletionJob(t *testing.T, pool *pgxpool.Pool, project uuid.UUID) deletionJob {
	t.Helper()
	rows, err := pool.Query(pgtest.Soon(t), `SELECT id,args,queue,max_attempts FROM river_job WHERE kind='issue.delete_rows' AND args->>'scope'='project' AND args->>'id'=$1`, project.String())
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var job deletionJob
	count := 0
	for rows.Next() {
		var raw []byte
		var queue string
		var attempts int
		if err := rows.Scan(&job.ID, &raw, &queue, &attempts); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &job); err != nil {
			t.Fatal(err)
		}
		var fields map[string]any
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		if queue != "default" || attempts != 25 || len(fields) != 4 {
			t.Fatalf("deletion job options/payload=%s %s/%d", raw, queue, attempts)
		}
		count++
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("project %s deletion jobs=%d want exactly one", project, count)
	}
	var at time.Time
	var by uuid.UUID
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at,updated_by_id FROM projects WHERE id=$1", project).Scan(&at, &by); err != nil {
		t.Fatal(err)
	}
	if job.Scope != "project" || job.Parent != project || !job.At.Equal(at) || job.By != by {
		t.Fatalf("deletion job=%+v want project %s at %s by %s", job, project, at, by)
	}
	t.Logf("project deletion job=%d scope=%s id=%s at=%s by=%s", job.ID, job.Scope, job.Parent, job.At.Format(time.RFC3339Nano), job.By)
	return job
}

func waitProjectDeletion(t *testing.T, pool *pgxpool.Pool, project uuid.UUID) {
	t.Helper()
	job := projectDeletionJob(t, pool, project)
	waitDeletionJob(t, pool, job, "completed")
}

func waitDeletionJob(t *testing.T, pool *pgxpool.Pool, job deletionJob, want string) {
	t.Helper()
	requestEventuallyWithin(t, "deletion job "+want, 15*time.Second, func(ctx context.Context) bool {
		var state, kind string
		var raw []byte
		var attempt, failures int
		if err := pool.QueryRow(ctx, "SELECT state::text,kind,args,attempt,coalesce(cardinality(errors),0) FROM river_job WHERE id=$1", job.ID).Scan(&state, &kind, &raw, &attempt, &failures); err != nil {
			t.Fatal(err)
		}
		got := deletionJob{ID: job.ID}
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		if kind != "issue.delete_rows" || got.ID != job.ID || got.Scope != job.Scope || got.Parent != job.Parent || !got.At.Equal(job.At) || got.By != job.By {
			t.Fatalf("job identity changed: %+v %s", got, kind)
		}
		if state == "discarded" || state == "cancelled" {
			t.Fatalf("deletion job %d ended %s", job.ID, state)
		}
		if failures != 0 || attempt > 1 {
			t.Fatalf("ordinary deletion job %d retried or failed: state=%s attempt=%d failures=%d", job.ID, state, attempt, failures)
		}
		if state == want {
			if attempt != 1 {
				t.Fatalf("job %d reached %s without one real attempt: %d", job.ID, want, attempt)
			}
			t.Logf("deletion job=%d state=%s attempt=%d failures=%d", job.ID, state, attempt, failures)
			return true
		}
		return false
	})
}

// A real first-batch row lock leaves the complete issue catalog untouched while
// the app and its production River worker continue running.
func holdProjectDeletion(t *testing.T, pool *pgxpool.Pool, project uuid.UUID) pgx.Tx {
	t.Helper()
	var draft uuid.UUID
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT id FROM draft_issues WHERE project_id=$1 AND deleted_at IS NULL ORDER BY id LIMIT 1", project).Scan(&draft); err != nil {
		t.Fatal(err)
	}
	holder := holding(t, pool, "SELECT 1 FROM draft_issues WHERE id=$1 FOR UPDATE", draft)
	if !pgtest.RowWaits(t, pool, "draft_issues", draft, "FOR UPDATE") {
		t.Fatal("draft barrier not held")
	}
	return holder
}

// The sole committed deletion job is running and its worker is directly
// blocked by this fixture. Stable backend/query identities exclude other waits.
func deletionWorkerBlocked(t *testing.T, pool *pgxpool.Pool, holder pgx.Tx, job deletionJob, table string) uint32 {
	t.Helper()
	return deletionWorkerBlockedBy(t, pool, holder.Conn().PgConn().PID(), job, table)
}

func deletionWorkerBlockedBy(t *testing.T, pool *pgxpool.Pool, holder uint32, job deletionJob, table string) uint32 {
	t.Helper()
	waitDeletionJob(t, pool, job, "running")
	var pid uint32
	requestEventuallyWithin(t, "deletion worker at row barrier", 10*time.Second, func(ctx context.Context) bool {
		rows, err := pool.Query(ctx, "SELECT pid FROM pg_stat_activity WHERE $1::int=ANY(pg_blocking_pids(pid)) AND wait_event_type='Lock'", holder)
		if err != nil {
			t.Fatal(err)
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[uint32])
		if err != nil {
			t.Fatal(err)
		}
		if len(ids) != 1 {
			return false
		}
		pid = ids[0]
		return true
	})
	probe := &requestProbe{}
	probe.pid.Store(pid)
	requestBlocked(t, pool, probe, holder, table)
	return pid
}

// Small routes use the same producer union and real InsertTx subscribers as newApp.
func projectDeletionPublisher(t *testing.T, pool *pgxpool.Pool, observers ...shared.EventSubscriber) shared.EventPublisher {
	t.Helper()
	client, err := jobs.NewInserter(pool)
	if err != nil {
		t.Fatal(err)
	}
	subscribers := append(issue.Subscribers(issueInserter{client: client}), observers...)
	d, err := newDispatcher(eventTypes(), subscribers)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func assertProjectDeletionStamp(t *testing.T, pool *pgxpool.Pool, k foreignKey, ids []uuid.UUID, project, by uuid.UUID) {
	t.Helper()
	if len(ids) == 0 {
		t.Fatalf("%s catalog has no live fixture", k)
	}
	var count, bad int
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT count(*),count(*) FILTER(WHERE r.deleted_at IS DISTINCT FROM p.deleted_at OR r.updated_at IS DISTINCT FROM p.deleted_at OR r.updated_by_id IS DISTINCT FROM $3) FROM "+k.table+" r,projects p WHERE p.id=$1 AND r.id=ANY($2)", project, ids, by).Scan(&count, &bad); err != nil {
		t.Fatal(err)
	}
	if count != len(ids) || bad != 0 {
		t.Errorf("%s: kept %d/%d rows, %d lack original deleted_at/updated_at/updated_by_id", k, count, len(ids), bad)
	}
}
