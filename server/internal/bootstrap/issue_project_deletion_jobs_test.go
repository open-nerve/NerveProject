package bootstrap

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Only the explicitly owned portions are asynchronous; unknown catalog tables still
// participate in the immediate assertions rather than receiving an exemption.
// The issue module's memberships (M6 P3a spec 2.5) are named for their targets,
// and its intake rows (M7 P2a spec 2.10) for the intake. The saved views are
// named for the work items they show, but are the navigation module's.
func projectIssueTable(table string) bool {
	return table == "draft_issues" || table == "issues" || table == "comment_reactions" || table == "cycle_issues" || table == "module_issues" ||
		table == "intake_issues" || len(table) > 6 && table[:6] == "issue_" && !navigationTable(table)
}

// The saved views, a project's and a workspace's, are deleted by the navigation module's own navigation.delete_rows
// job (M7 P3b spec 2.7, R3), and so are the favorites, a project's and a workspace's own (M7 P3c spec 2.6, R11), and a
// project's recent visits (M7 P3d spec 2.6).
func navigationTable(table string) bool {
	return table == "issue_views" || table == "user_favorites" || table == "user_recent_visits"
}

// The planning module's tables are deleted by its own planning.delete_rows job (M6 P1 spec 2.4, P2 spec 2.4, P3b
// spec 2.5).
func projectPlanningTable(table string) bool {
	return table == "cycles" || table == "modules" || table == "module_members" || table == "module_links" ||
		table == "cycle_user_properties" || table == "module_user_properties"
}

// The notifications are deleted by the notification module's own notification.delete_rows job once the project's work
// items are (M7 P1b spec 2.8).
func parentAsyncTable(table string) bool {
	return table == "file_assets" || table == "notifications" || projectIssueTable(table) || projectPlanningTable(table) || navigationTable(table)
}

type deletionJob struct {
	ID     int64     `json:"-"`
	Scope  string    `json:"scope"`
	Parent uuid.UUID `json:"id"`
	At     time.Time `json:"at"`
	By     uuid.UUID `json:"by"`
}

func projectDeletionJob(t *testing.T, pool *pgxpool.Pool, project uuid.UUID) deletionJob {
	return parentDeletionJob(t, pool, "project", project)
}

func parentDeletionJob(t *testing.T, pool *pgxpool.Pool, scope string, project uuid.UUID) deletionJob {
	t.Helper()
	rows, err := pool.Query(pgtest.Soon(t), `SELECT id,args,queue,max_attempts FROM river_job WHERE kind='issue.delete_rows' AND args->>'scope'=$2 AND args->>'id'=$1`, project.String(), scope)
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
	query := "SELECT deleted_at,updated_by_id FROM projects WHERE id=$1"
	if scope == "workspace_drafts" {
		query = "SELECT deleted_at,updated_by_id FROM workspaces WHERE id=$1"
	}
	if err := pool.QueryRow(pgtest.Soon(t), query, project).Scan(&at, &by); err != nil {
		t.Fatal(err)
	}
	if job.Scope != scope || job.Parent != project || !job.At.Equal(at) || job.By != by {
		t.Fatalf("deletion job=%+v want project %s at %s by %s", job, project, at, by)
	}
	t.Logf("project deletion job=%d scope=%s id=%s at=%s by=%s", job.ID, job.Scope, job.Parent, job.At.Format(time.RFC3339Nano), job.By)
	return job
}

func waitProjectDeletion(t *testing.T, pool *pgxpool.Pool, project uuid.UUID) {
	t.Helper()
	job := projectDeletionJob(t, pool, project)
	waitDeletionJob(t, pool, job, "completed")
	waitPlanningDeletion(t, pool, project)
	waitNotificationDeletion(t, pool, project)
	waitNavigationDeletion(t, pool, "project", project)
	waitAssetScopeDeletion(t, pool, "project", project)
}

func waitDeletionJob(t *testing.T, pool *pgxpool.Pool, job deletionJob, want string) {
	t.Helper()
	waitDeletionJobWithin(t, pool, job, want, 15*time.Second)
}

func waitDeletionJobWithin(t *testing.T, pool *pgxpool.Pool, job deletionJob, want string, timeout time.Duration) {
	t.Helper()
	requestEventuallyWithin(t, "deletion job "+want, timeout, func(ctx context.Context) bool {
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

// barriersOutlasted configures the world of a test that holds deletion
// workers at its barriers (holdProjectDeletion, holdPlanningDeletion,
// holding) across work its design does not bound: a loop over the whole
// catalog or API, or other jobs' pickup, completion and snoozes. Each lock
// wait of the work-item side's transactions, the issue, planning and
// notification jobs' among them, then ends far past any such hold and under
// those jobs' River timeout (delete_budget + 5 s), so a held worker keeps its
// first attempt however slowly the runner goes. In production a job waiting
// past issue.write_lock_timeout fails and River retries it; the barriers are
// the test's own, not the contention that timeout is for.
func barriersOutlasted(cfg *config.Config) {
	cfg.Issue.WriteLockTimeout = 30 * time.Second
}

// The sole committed deletion job is running and its worker is directly
// blocked by this fixture on the expected table. A periodic worker can also
// wait on this transaction; stable backend/query identities exclude that wait.
// The worker is known by the statement with which it waits for table
// (deletionWorkerStatement), not by a query naming the table: another job's
// query may name it too, as notification.fanout's lock of a work item names
// issues (M7 P1b spec R10).
func deletionWorkerBlocked(t *testing.T, pool *pgxpool.Pool, holder pgx.Tx, job deletionJob, table string) uint32 {
	t.Helper()
	return deletionWorkerBlockedBy(t, pool, holder.Conn().PgConn().PID(), job, table)
}

func deletionWorkerBlockedBy(t *testing.T, pool *pgxpool.Pool, holder uint32, job deletionJob, table string) uint32 {
	t.Helper()
	statement := deletionWorkerStatement(t, job, table)
	waitDeletionJob(t, pool, job, "running")
	var pid uint32
	requestEventuallyWithin(t, "deletion worker at row barrier", 10*time.Second, func(ctx context.Context) bool {
		rows, err := pool.Query(ctx, "SELECT pid FROM pg_stat_activity WHERE datname=current_database() AND $1::int=ANY(pg_blocking_pids(pid)) AND wait_event_type='Lock' AND "+pgtest.StatementIn(2), holder, []string{statement})
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

// deletionWorkerStatement is the statement with which the worker of job, an
// issue.delete_rows, waits for a row of table: the project's drafts, its
// relations or its work items, or a workspace's drafts without a project.
func deletionWorkerStatement(t *testing.T, job deletionJob, table string) string {
	t.Helper()
	statement, ok := map[[2]string]string{
		{"project", "draft_issues"}:          "DeleteProjectDraftRows",
		{"project", "issue_relations"}:       "LockProjectRelationRows",
		{"project", "issues"}:                "DeleteProjectIssueRows",
		{"workspace_drafts", "draft_issues"}: "DeleteWorkspaceDraftRows",
	}[[2]string{job.Scope, table}]
	if !ok {
		t.Fatalf("no statement of a %s deletion waits for a row of %s", job.Scope, table)
	}
	return statement
}

// Small routes use the same producer union and real InsertTx subscribers as newApp.
func projectDeletionPublisher(t *testing.T, pool *pgxpool.Pool, observers ...shared.EventSubscriber) shared.EventPublisher {
	t.Helper()
	client, err := jobs.NewInserter(pool)
	if err != nil {
		t.Fatal(err)
	}
	subscribers := slices.Concat(issue.Subscribers(issue.Provide(pool, nil), jobInserter{client: client}), planning.Subscribers(jobInserter{client: client}), asset.Subscribers(asset.Provide(pool), jobInserter{client: client}),
		notification.Subscribers(notification.Provide(pool), jobInserter{client: client}), navigation.Subscribers(navigation.Provide(pool), jobInserter{client: client}))
	subscribers = append(subscribers, observers...)
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

func waitWorkspaceDeletion(t *testing.T, pool *pgxpool.Pool, workspace uuid.UUID) {
	t.Helper()
	// These are this request's projects, excluding earlier deletions.
	projects := queryIDs(t, pool, "SELECT p.id FROM projects p JOIN workspaces w ON w.id=p.workspace_id WHERE w.id=$1 AND p.deleted_at=w.deleted_at ORDER BY p.id", workspace)
	for _, project := range projects {
		waitDeletionJob(t, pool, projectDeletionJob(t, pool, project), "completed")
		waitPlanningDeletion(t, pool, project)
		waitNotificationDeletion(t, pool, project)
		waitNavigationDeletion(t, pool, "project", project)
	}
	waitDeletionJob(t, pool, parentDeletionJob(t, pool, "workspace_drafts", workspace), "completed")
	waitNavigationDeletion(t, pool, "workspace", workspace)
	waitAssetScopeDeletion(t, pool, "workspace", workspace)
}
