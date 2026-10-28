package bootstrap

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueriver "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

type deletionRun struct {
	Result issueapp.DeleteRowsResult
	Err    error
}
type deletionExecution struct {
	issueriver.DeleteRowsUseCase
	mu   sync.Mutex
	runs []deletionRun
}

func (o *deletionExecution) Execute(ctx context.Context, scope string, id uuid.UUID, at time.Time, by uuid.UUID) (issueapp.DeleteRowsResult, error) {
	r, err := o.DeleteRowsUseCase.Execute(ctx, scope, id, at, by)
	o.mu.Lock()
	o.runs = append(o.runs, deletionRun{r, err})
	o.mu.Unlock()
	return r, err
}
func (o *deletionExecution) snapshot() []deletionRun {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]deletionRun(nil), o.runs...)
}

type deletionTxObservation struct{ Inside, Outside error }
type deletionTransactions struct {
	shared.TxManager
	mu     sync.Mutex
	errors []deletionTxObservation
}

func (o *deletionTransactions) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	var inside error
	err := o.TxManager.WithinTx(ctx, func(ctx context.Context) error { inside = fn(ctx); return inside })
	if err != nil {
		o.mu.Lock()
		o.errors = append(o.errors, deletionTxObservation{inside, err})
		o.mu.Unlock()
	}
	return err
}
func (o *deletionTransactions) failures() []deletionTxObservation {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]deletionTxObservation(nil), o.errors...)
}

type deletionLifecycle struct {
	w         issueWorld
	execution *deletionExecution
	tx        *deletionTransactions
	logs      *lockedBuffer
}

// Only the unstarted Runner is replaced. HTTP, authentication, request gates,
// publisher and Inserter still come from the real app. Business requests enqueue
// deletion jobs only; extra may register production periodic jobs as well.
func newDeletionLifecycle(t *testing.T, configure func(*config.Config), runtimeNow func() time.Time, extra func(*pgxpool.Pool) []jobs.Job) deletionLifecycle {
	t.Helper()
	db := pgtest.NewDatabase(t)
	pool := openPool(t, db)
	observeJobStates(t, pool)
	for _, sql := range []string{
		`CREATE TABLE job_observer.control (hold_insert boolean NOT NULL, hold_retry boolean NOT NULL); INSERT INTO job_observer.control VALUES(false,false)`,
		`CREATE FUNCTION job_observer.defer_deletion() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.kind='issue.delete_rows' AND ((TG_OP='INSERT' AND (SELECT hold_insert FROM job_observer.control)) OR (TG_OP='UPDATE' AND NEW.state IN ('available','retryable') AND cardinality(NEW.errors)>0 AND (SELECT hold_retry FROM job_observer.control))) THEN NEW.scheduled_at=clock_timestamp()+interval '1 day'; END IF; RETURN NEW; END $$`,
		`CREATE TRIGGER defer_test_deletion BEFORE INSERT OR UPDATE OF state ON river_job FOR EACH ROW EXECUTE FUNCTION job_observer.defer_deletion()`,
	} {
		if _, err := pool.Exec(pgtest.Soon(t), sql); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := pool.Exec(ctx, `DROP TRIGGER defer_test_deletion ON river_job`); err != nil {
			t.Error(err)
		}
	})
	cfg := testConfig(t, db, false)
	cfg.Issue.DeleteBatch = 1
	cfg.Issue.DeleteBudget = time.Second
	configure(&cfg)
	logs := &lockedBuffer{}
	logger := slog.New(slog.NewJSONHandler(logs, nil))
	a := buildAppLogging(t, cfg, migrations.FS(), logger)
	tx := &deletionTransactions{TxManager: newIssueJobTx(a.pool, cfg)}
	store := issuepg.New(a.pool)
	uc := issueapp.NewDeleteRows(issueapp.DeleteRowsDeps{Rows: store, Relations: issueapp.NewRelationDeletions(store, issueProjects{projects: project.Provide(a.pool).IssueProjects}), Remaining: store, Tx: tx, RuntimeNow: runtimeNow, Batch: cfg.Issue.DeleteBatch, Budget: cfg.Issue.DeleteBudget})
	execution := &deletionExecution{DeleteRowsUseCase: uc}
	definitions := []jobs.Job{issueriver.DeleteRowsJob(execution, cfg.Issue.DeleteBudget, logger)}
	if extra != nil {
		definitions = append(definitions, extra(a.pool)...)
	}
	runner, err := jobs.New(a.pool, jobs.Config{ShutdownTimeout: cfg.Jobs.ShutdownTimeout, Logger: logger, DefaultWorkers: cfg.Jobs.DefaultWorkers, PeriodicWorkers: cfg.Jobs.PeriodicWorkers}, definitions, []string{issueriver.DeleteRowsKind})
	if err != nil {
		t.Fatal(err)
	}
	// The old Runner has never started or borrowed a listener connection. Its
	// pool belongs to app.close; Stop is only valid on the replacement we start.
	a.jobs = runner
	w := prepareIssueWorld(t, issueWorld{contract: apitest.Load(t), base: runBuiltApp(t, a), pool: pool, tokens: map[string]string{}, ids: map[string]uuid.UUID{}})
	return deletionLifecycle{w, execution, tx, logs}
}

func setDeletionControl(t testing.TB, pool *pgxpool.Pool, insert, retry bool) {
	t.Helper()
	if _, err := pool.Exec(pgtest.Soon(t), `UPDATE job_observer.control SET hold_insert=$1,hold_retry=$2`, insert, retry); err != nil {
		t.Fatal(err)
	}
}
func releaseDeletionJob(t testing.TB, pool *pgxpool.Pool, job deletionJob, attempt int) {
	t.Helper()
	tag, err := pool.Exec(pgtest.Soon(t), `UPDATE river_job SET scheduled_at=clock_timestamp() WHERE id=$1 AND attempt=$2 AND state IN ('available','scheduled','retryable')`, job.ID, attempt)
	if err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("release job %d attempt %d: %v %v", job.ID, attempt, tag, err)
	}
}

func assertDeletionAudit(t testing.TB, pool *pgxpool.Pool, table string, ids []uuid.UUID, job deletionJob) {
	t.Helper()
	var count, bad int
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT count(*),count(*) FILTER(WHERE deleted_at IS DISTINCT FROM $2 OR updated_at IS DISTINCT FROM $2 OR updated_by_id IS DISTINCT FROM $3) FROM `+table+` WHERE id=ANY($1)`, ids, job.At, job.By).Scan(&count, &bad); err != nil {
		t.Fatal(err)
	}
	if count != len(ids) || bad != 0 {
		t.Fatalf("%s count=%d/%d bad stamps=%d", table, count, len(ids), bad)
	}
}
