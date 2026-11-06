package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A purge statement has one of three shapes. A page scans and deletes one
// bounded page of its own table in one statement; the rows it deleted are in
// its result row, which a tracer cannot read. A protected table runs a
// candidates SELECT, then a DELETE of exactly those IDs whose command tag
// counts the rows it removed. A run's budget may cancel the statement in
// flight; it commits nothing and the next run repeats it.
const (
	purgePage       = "page"
	purgeCandidates = "candidates"
	purgeDelete     = "delete"
)

type purgeScaleStatement struct {
	SQL, Name, Role, Table, User, Error string
	PID                                 uint32
	Cutoff                              time.Time
	After                               uuid.UUID
	Limit                               int32
	IDs                                 int
	Rows                                int64
	Deadline                            bool
	Started, Ended                      time.Time
}

var (
	purgeScaleName     = regexp.MustCompile(`^-- name: ((?:Delete)?Purge\w*) :\w+\n`)
	purgeScaleDeletes  = regexp.MustCompile(`\bDELETE FROM ([a-z_]+)\b`)
	purgeScaleSelects  = regexp.MustCompile(`^SELECT id FROM ([a-z_]+)\n`)
	purgeScalePageHead = "WITH candidates AS MATERIALIZED ("
)

type purgeScaleTrace struct {
	mu      sync.Mutex
	records []*purgeScaleStatement
}
type purgeScaleTraceKey struct{}

func (trace *purgeScaleTrace) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	name := purgeScaleName.FindStringSubmatch(data.SQL)
	if name == nil {
		return ctx
	}
	r := &purgeScaleStatement{SQL: data.SQL, Name: name[1], User: conn.Config().User, PID: conn.PgConn().PID(), Started: time.Now()}
	body := data.SQL[len(name[0]):]
	deletes := purgeScaleDeletes.FindStringSubmatch(body)
	selects := purgeScaleSelects.FindStringSubmatch(body)
	switch {
	case strings.HasPrefix(r.Name, "DeletePurge") && deletes != nil:
		r.Role, r.Table = purgeDelete, deletes[1]
	case strings.HasSuffix(r.Name, "Candidates") && selects != nil && deletes == nil:
		r.Role, r.Table = purgeCandidates, selects[1]
	case strings.HasPrefix(body, purgeScalePageHead) && deletes != nil:
		r.Role, r.Table = purgePage, deletes[1]
	default:
		r.Error = "unknown purge statement shape"
	}
	if r.Error == "" {
		r.Error = r.bind(data.Args)
	}
	trace.mu.Lock()
	trace.records = append(trace.records, r)
	trace.mu.Unlock()
	return context.WithValue(ctx, purgeScaleTraceKey{}, r)
}

// bind reads parameters by type: sqlc orders them by first use, which differs
// between queries.
func (r *purgeScaleStatement) bind(args []any) string {
	var cutoffs, afters, limits, lists int
	types := make([]string, 0, len(args))
	for _, arg := range args {
		types = append(types, fmt.Sprintf("%T", arg))
		switch v := arg.(type) {
		case *time.Time:
			if v != nil {
				r.Cutoff = *v
			}
			cutoffs++
		case uuid.UUID:
			r.After, afters = v, afters+1
		case int32:
			r.Limit, limits = v, limits+1
		case []uuid.UUID:
			r.IDs, lists = len(v), lists+1
		}
	}
	want := [4]int{1, 1, 1, 0}
	if r.Role == purgeDelete {
		want = [4]int{1, 0, 0, 1}
	}
	if [4]int{cutoffs, afters, limits, lists} != want || len(args) != cutoffs+afters+limits+lists || r.Cutoff.IsZero() {
		return fmt.Sprintf("unexpected %s parameters %v", r.Role, types)
	}
	return ""
}

func (trace *purgeScaleTrace) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	r, _ := ctx.Value(purgeScaleTraceKey{}).(*purgeScaleStatement)
	if r == nil {
		return
	}
	trace.mu.Lock()
	defer trace.mu.Unlock()
	r.Ended, r.Rows = time.Now(), data.CommandTag.RowsAffected()
	if data.Err != nil {
		r.Deadline = r.Error == "" && errors.Is(data.Err, context.DeadlineExceeded)
		r.Error += data.Err.Error()
	}
}

func (trace *purgeScaleTrace) snapshot() []purgeScaleStatement {
	trace.mu.Lock()
	defer trace.mu.Unlock()
	out := make([]purgeScaleStatement, 0, len(trace.records))
	for _, row := range trace.records {
		out = append(out, *row)
	}
	return out
}

func (trace *purgeScaleTrace) pool(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	source, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		return nil, err
	}
	copied := source.Config()
	source.Close()
	copied.ConnConfig.Tracer = trace
	return pgxpool.NewWithConfig(ctx, copied)
}

type purgeScaleJob struct {
	ID                       int64
	Kind, Queue, State       string
	Attempt, Errors, Snoozes int
	Created                  time.Time
	Attempted, Finalized     *time.Time
	ErrorRows, Metadata      json.RawMessage
}

// The purge is one attempt resumed across budget-bounded runs: a run that
// spends its budget records its cursor and snoozes, which River does not count
// as an attempt.
func awaitPurgeScaleJob(t *testing.T, pool *pgxpool.Pool, before int64, dir string) purgeScaleJob {
	t.Helper()
	var job purgeScaleJob
	requestEventuallyWithin(t, "new purge scale job completed", 15*time.Minute, func(ctx context.Context) bool {
		rows, err := pool.Query(ctx, `SELECT id,kind,queue,state::text,attempt,coalesce(cardinality(errors),0),
 coalesce((metadata->>'snoozes')::int,0),created_at,attempted_at,finalized_at,to_json(errors),metadata
 FROM river_job WHERE kind='cleanup.purge_deleted' AND id>$1 ORDER BY id`, before)
		if err != nil {
			t.Fatal(err)
		}
		jobs, err := pgx.CollectRows(rows, pgx.RowToStructByPos[purgeScaleJob])
		if err != nil || len(jobs) > 1 {
			t.Fatalf("purge scale fresh job set=%+v err=%v", jobs, err)
		}
		if len(jobs) == 0 {
			return false
		}
		job = jobs[0]
		if job.ID <= before || job.Errors != 0 || job.Attempt > 1 || job.State == "discarded" || job.State == "cancelled" {
			listEvidence(t, dir, "job-failure", job)
			t.Fatalf("purge scale job is not one fresh successful attempt: %+v watermark=%d", job, before)
		}
		return job.State == "completed"
	})
	return job
}

// purgeScaleRun is one execution of the purge job and the output it recorded.
type purgeScaleRun struct {
	Started, Ended time.Time
	To             string
	Output         purgeResult
}

// purgeScaleRuns pairs each observed start with the transition that ended it;
// assertJobRuns has already checked that they pair.
func purgeScaleRuns(t *testing.T, pool *pgxpool.Pool, id int64) []purgeScaleRun {
	t.Helper()
	rows, err := pool.Query(pgtest.Soon(t), `SELECT old_state,new_state,metadata,at FROM job_observer.transitions WHERE job_id=$1 ORDER BY ordinal`, id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var runs []purgeScaleRun
	var started time.Time
	for rows.Next() {
		var from, to string
		var metadata []byte
		var at time.Time
		if err := rows.Scan(&from, &to, &metadata, &at); err != nil {
			t.Fatal(err)
		}
		if to == "running" {
			started = at
			continue
		}
		if from != "running" {
			continue
		}
		output, present, err := decodePurgeOutput(metadata)
		if err != nil || !present {
			t.Fatalf("purge run ended without its output: present=%t err=%v metadata=%s", present, err, metadata)
		}
		runs = append(runs, purgeScaleRun{Started: started, Ended: at, To: to, Output: output})
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return runs
}

// purgeScaleRunSummary reads the purge's own account across its runs: each run
// reports the rows its statements removed from their own tables. When nothing
// outside the old scope is purgeable and that scope is gone, a total equal to
// the scope proves no row of any table was removed by a cascade instead.
func purgeScaleRunSummary(t *testing.T, runs []purgeScaleRun, cutoff time.Time, cut int, want map[string]int64) map[string]any {
	t.Helper()
	var scope int64
	for _, n := range want {
		scope += n
	}
	var scanned, deleted, blocked int
	var seconds float64
	for i, run := range runs {
		o, last := run.Output, i == len(runs)-1
		ended := run.To == "completed"
		if !last {
			ended = run.To == "available" || run.To == "scheduled"
		}
		if !ended || o.Cursor.Version != 1 || !o.Cursor.Cutoff.Equal(cutoff) || o.Complete != last || run.Started.IsZero() || run.Ended.Before(run.Started) {
			t.Fatalf("purge run %d is not one budget slice of this cycle (cutoff %s): %+v", i, cutoff, run)
		}
		scanned, deleted, blocked = scanned+o.Scanned, deleted+o.Deleted, blocked+o.Blocked
		seconds += run.Ended.Sub(run.Started).Seconds()
	}
	if blocked != 0 || scanned != deleted || int64(deleted) != scope {
		t.Errorf("purge runs scanned=%d deleted=%d blocked=%d; old scope rows=%d", scanned, deleted, blocked, scope)
	}
	if cut > len(runs)-1 {
		t.Errorf("%d statements cancelled by a budget, but only %d runs snoozed", cut, len(runs)-1)
	}
	return map[string]any{"runs": len(runs), "running_seconds": seconds, "scanned": scanned, "deleted": deleted, "blocked": blocked, "budget_cut_statements": cut}
}

func purgeScaleSQLSummary(t *testing.T, records []purgeScaleStatement, role string, started, ended time.Time, want map[string]int64) (time.Time, int, map[string]any) {
	t.Helper()
	if len(records) == 0 {
		t.Fatal("purge task completed without executing any purge SQL")
	}
	cutoff := records[0].Cutoff
	if cutoff.Before(started.Add(-60*24*time.Hour)) || cutoff.After(ended.Add(-60*24*time.Hour)) {
		t.Fatalf("purge cutoff outside this app's business clock: %s not %s..%s", cutoff, started, ended)
	}
	direct := map[string]int64{}
	kinds := map[string]string{}
	statements := map[string]int{}
	ms := map[string]float64{}
	var order []string
	cut := 0
	for i, r := range records {
		bounded := r.Role == purgePage && r.Limit == 1000 && r.Rows == 1 ||
			r.Role == purgeCandidates && r.Limit == 1000 && r.Rows <= 1000 ||
			r.Role == purgeDelete && r.IDs >= 1 && r.IDs <= 1000 && r.Rows <= int64(r.IDs)
		if r.Error != "" && budgetCut(records, i) {
			bounded, cut = r.Rows == 0, cut+1
		} else if r.Error != "" {
			bounded = false
		}
		if !bounded || r.User != role || r.Rows < 0 || r.PID == 0 || r.Ended.IsZero() || r.Ended.Before(r.Started) || r.Started.Before(started) || r.Ended.After(ended) || !r.Cutoff.Equal(cutoff) {
			t.Fatalf("invalid purge SQL evidence: %+v", r)
		}
		kind := "protected"
		if r.Role == purgePage {
			kind = purgePage
		}
		if seen, ok := kinds[r.Table]; ok && seen != kind {
			t.Fatalf("purge table %s mixes page and protected statements", r.Table)
		}
		kinds[r.Table] = kind
		if r.Role == purgeDelete && !r.Deadline {
			direct[r.Table] += r.Rows
		}
		if len(order) == 0 || order[len(order)-1] != r.Table {
			order = append(order, r.Table)
		}
		statements[r.Table]++
		ms[r.Table] += float64(r.Ended.Sub(r.Started)) / float64(time.Millisecond)
	}
	// Zero-row workspace and asset statements still execute; their rows are
	// retained. A page's deletions are checked through the runs' account.
	for table, n := range want {
		if _, ok := kinds[table]; !ok {
			t.Errorf("purge executed no statement on %s", table)
		}
		if kinds[table] == "protected" && direct[table] != n {
			t.Errorf("purge direct DELETE %s=%d want=%d", table, direct[table], n)
		}
	}
	if len(kinds) != len(want) {
		t.Errorf("purge statement table set=%v want=%v", kinds, want)
	}
	return cutoff, cut, map[string]any{"cutoff": cutoff, "protected_direct_rows": direct, "table_kinds": kinds, "statements_by_table": statements, "sql_ms_by_table": ms, "table_order": order, "statements": len(records), "budget_cut_statements": cut}
}

// budgetCut reports whether records[i] is a statement its run's budget
// cancelled: the next statement must repeat its page from the same cursor. A
// cancelled DELETE repeats from its page's candidates.
func budgetCut(records []purgeScaleStatement, i int) bool {
	r := records[i]
	if !r.Deadline || i+1 == len(records) {
		return false
	}
	next, from := records[i+1], r
	if r.Role == purgeDelete {
		k := i - 1
		for k >= 0 && records[k].Role == purgeDelete {
			k--
		}
		if k < 0 {
			return false
		}
		from = records[k]
	}
	return from.Table == r.Table && from.Role != purgeDelete && next.Table == from.Table && next.Role == from.Role && next.After == from.After
}

func purgeScaleBoundary(t *testing.T, pool *pgxpool.Pool) (int64, time.Time) {
	t.Helper()
	var before int64
	var watermark time.Time
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT coalesce(max(id),0),clock_timestamp() FROM river_job WHERE kind='cleanup.purge_deleted'").Scan(&before, &watermark); err != nil || before == 0 {
		t.Fatalf("missing completed preparation boundary: id=%d err=%v", before, err)
	}
	return before, watermark
}
