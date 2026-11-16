package bootstrap

import (
	"context"
	"encoding/json"
	"math"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	planningpg "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres"
	planningapp "github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// freezeScaleTrace keeps the candidates' statement as the store sends it.
type freezeScaleTrace struct {
	mu   sync.Mutex
	sql  string
	args []any
}

func (tr *freezeScaleTrace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	if strings.HasPrefix(d.SQL, "-- name: FreezeDueProjects ") {
		tr.mu.Lock()
		tr.sql, tr.args = d.SQL, d.Args
		tr.mu.Unlock()
	}
	return ctx
}

func (*freezeScaleTrace) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

// freezeScalePool is a pool on url with postgres.NewPool's planner
// settings, its statements traced.
func freezeScalePool(t *testing.T, url string, trace pgx.QueryTracer) *pgxpool.Pool {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["jit"] = "off"
	cfg.ConnConfig.RuntimeParams["plan_cache_mode"] = "force_custom_plan"
	cfg.ConnConfig.Tracer = trace
	cfg.MaxConns = 4
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

type freezeScalePlan struct {
	SQL                 string
	Args                []any
	Candidates          []uuid.UUID
	ReadMilliseconds    float64
	Nodes, Indexes      []string
	Cost, SeqScanCost   float64
	Plan, SeqScan, Size json.RawMessage
}

// explainFreezeDue reads the candidates at at through the store, as the
// job does, then EXPLAINs (ANALYZE, BUFFERS) the statement it sent with
// its arguments, read-only. Beside it, for the margin alone, the cost of
// the plan without index scans and the sizes the planner saw.
func explainFreezeDue(t *testing.T, pool *pgxpool.Pool, trace *freezeScaleTrace, at time.Time) freezeScalePlan {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	start := time.Now()
	candidates, err := planningpg.New(pool).FreezeDueProjects(ctx, shared.DateOf(at))
	if err != nil {
		t.Fatal(err)
	}
	p := freezeScalePlan{Candidates: candidates, ReadMilliseconds: float64(time.Since(start)) / float64(time.Millisecond)}
	trace.mu.Lock()
	p.SQL, p.Args = trace.sql, trace.args
	trace.mu.Unlock()
	if p.SQL == "" {
		t.Fatal("the candidates' statement was not traced")
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err = tx.QueryRow(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) "+p.SQL, p.Args...).Scan(&p.Plan); err != nil {
		t.Fatal(err)
	}
	var plans []struct{ Plan freezeScalePlanNode }
	if err = json.Unmarshal(p.Plan, &plans); err != nil || len(plans) != 1 {
		t.Fatalf("EXPLAIN answered %s: %v", p.Plan, err)
	}
	p.Cost = plans[0].Plan.TotalCost
	if err = tx.QueryRow(ctx, `SELECT json_agg(json_build_object('relation',relname,'pages',relpages,'tuples',reltuples) ORDER BY relname) FROM pg_class
		WHERE relname IN ('cycles','cycles_freeze_due_idx')`).Scan(&p.Size); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, "SET LOCAL enable_bitmapscan=off; SET LOCAL enable_indexscan=off; SET LOCAL enable_indexonlyscan=off"); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, "EXPLAIN (FORMAT JSON) "+p.SQL, p.Args...).Scan(&p.SeqScan); err != nil {
		t.Fatal(err)
	}
	var seq []struct{ Plan freezeScalePlanNode }
	if err = json.Unmarshal(p.SeqScan, &seq); err != nil || len(seq) != 1 {
		t.Fatalf("EXPLAIN answered %s: %v", p.SeqScan, err)
	}
	p.SeqScanCost = seq[0].Plan.TotalCost
	plans[0].Plan.walk(func(n freezeScalePlanNode) {
		p.Nodes = append(p.Nodes, n.NodeType)
		if n.IndexName != "" {
			p.Indexes = append(p.Indexes, n.IndexName)
		}
	})
	return p
}

type freezeScalePlanNode struct {
	NodeType  string                `json:"Node Type"`
	IndexName string                `json:"Index Name"`
	TotalCost float64               `json:"Total Cost"`
	Plans     []freezeScalePlanNode `json:"Plans"`
}

func (n freezeScalePlanNode) walk(visit func(freezeScalePlanNode)) {
	visit(n)
	for _, c := range n.Plans {
		c.walk(visit)
	}
}

// freezeHold is one project's transaction in a run: begun, its project's
// lock granted, and ended, its commit's or rollback's end; the lock is held
// from its grant to the end.
type freezeHold struct {
	Project              uuid.UUID
	Begun, Locked, Ended time.Time
	WaitMS, HeldMS       float64
	Failed               bool
}

// freezeHolds are a run's transactions, one at a time, as the job takes
// its candidates in turn.
type freezeHolds struct {
	current freezeHold
	done    []freezeHold
}

type timedFreezeLocks struct {
	planningapp.ProjectLocker
	holds *freezeHolds
}

func (l timedFreezeLocks) Lock(ctx context.Context, id uuid.UUID) (planningapp.ProjectFacts, bool, error) {
	f, found, err := l.ProjectLocker.Lock(ctx, id)
	l.holds.current.Project, l.holds.current.Locked = id, time.Now()
	return f, found, err
}

type timedFreezeTx struct {
	shared.TxManager
	holds *freezeHolds
}

func (x timedFreezeTx) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	x.holds.current = freezeHold{Begun: time.Now()}
	err := x.TxManager.WithinTx(ctx, fn)
	h := x.holds.current
	h.Ended, h.Failed = time.Now(), err != nil
	h.WaitMS, h.HeldMS = float64(h.Locked.Sub(h.Begun))/float64(time.Millisecond), float64(h.Ended.Sub(h.Locked))/float64(time.Millisecond)
	x.holds.done = append(x.holds.done, h)
	return err
}

type freezeScaleResult struct {
	At             time.Time
	Frozen         int
	Err            string
	Seconds        float64
	HeldMS, WaitMS map[string]float64
	Holds          []freezeHold
}

// freezeScaleRun runs the freeze once on its clock at at, wired as
// freezeCyclesWith wires it, in the job's own transactions on pool; the
// project locks and the transactions noted.
func freezeScaleRun(t *testing.T, pool *pgxpool.Pool, at time.Time) freezeScaleResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Minute)
	defer cancel()
	holds := &freezeHolds{}
	projects := planningProjects{projects: project.Provide(pool, nil).PlanningProjects}
	tx := newIssueJobTx(pool, config.Config{Database: config.DatabaseConfig{CommitTimeout: 2 * time.Second}, Issue: testIssueConfig})
	uc := planningapp.NewFreezeCycles(planningapp.FreezeCyclesDeps{Projects: timedFreezeLocks{projects, holds}, Groups: projects, Cycles: planningpg.New(pool),
		Stats: planningIssues{stats: issue.Provide(pool, nil).PlanningStats}, Tx: timedFreezeTx{tx, holds}, Clock: clocktest.At(at)})
	start := time.Now()
	n, err := uc.Execute(ctx)
	r := freezeScaleResult{At: at, Frozen: n, Seconds: time.Since(start).Seconds(), Holds: holds.done}
	if err != nil {
		r.Err = err.Error()
	}
	held, wait := make([]float64, 0, len(r.Holds)), make([]float64, 0, len(r.Holds))
	for _, h := range r.Holds {
		held, wait = append(held, h.HeldMS), append(wait, h.WaitMS)
	}
	r.HeldMS, r.WaitMS = freezeScaleQuantiles(held), freezeScaleQuantiles(wait)
	return r
}

// freezeScaleQuantiles are the nearest-rank quantiles of values, as
// scaleSummary's, and their sum.
func freezeScaleQuantiles(values []float64) map[string]float64 {
	if len(values) == 0 {
		return nil
	}
	sorted := slices.Sorted(slices.Values(values))
	rank := func(p float64) float64 { return sorted[int(math.Ceil(p*float64(len(sorted))))-1] }
	sum := 0.0
	for _, v := range sorted {
		sum += v
	}
	return map[string]float64{"n": float64(len(sorted)), "p50": rank(.5), "p95": rank(.95), "max": sorted[len(sorted)-1], "sum": sum}
}
