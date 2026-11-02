package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
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

type purgeScaleStatement struct {
	SQL, Table, User, Error string
	PID                     uint32
	Cutoff                  time.Time
	After                   uuid.UUID
	Limit                   int32
	Rows                    int64
	Started, Ended          time.Time
}

type purgeScaleTrace struct {
	mu      sync.Mutex
	records []*purgeScaleStatement
}
type purgeScaleTraceKey struct{}

func (trace *purgeScaleTrace) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if !strings.HasPrefix(data.SQL, "-- name: Purge") {
		return ctx
	}
	r := &purgeScaleStatement{SQL: data.SQL, User: conn.Config().User, PID: conn.PgConn().PID(), Started: time.Now()}
	_, tail, found := strings.Cut(data.SQL, "\nDELETE FROM ")
	if !found || len(strings.Fields(tail)) == 0 || len(data.Args) != 3 {
		r.Error = "unknown purge statement or argument count"
	} else {
		r.Table = strings.Fields(tail)[0]
		cutoff, cutoffOK := data.Args[0].(*time.Time)
		after, afterOK := data.Args[1].(uuid.UUID)
		limit, limitOK := data.Args[2].(int32)
		if !cutoffOK || cutoff == nil || !afterOK || !limitOK {
			r.Error = fmt.Sprintf("unexpected purge parameter types: %T/%T/%T", data.Args[0], data.Args[1], data.Args[2])
		} else {
			r.Cutoff, r.After, r.Limit = *cutoff, after, limit
		}
	}
	trace.mu.Lock()
	trace.records = append(trace.records, r)
	trace.mu.Unlock()
	return context.WithValue(ctx, purgeScaleTraceKey{}, r)
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
		if job.ID <= before || job.Errors != 0 || job.Snoozes != 0 || job.Attempt > 1 || job.State == "discarded" || job.State == "cancelled" {
			listEvidence(t, dir, "job-failure", job)
			t.Fatalf("purge scale job is not one fresh successful attempt: %+v watermark=%d", job, before)
		}
		return job.State == "completed"
	})
	return job
}

func purgeScaleSQLSummary(t *testing.T, records []purgeScaleStatement, role string, started, ended time.Time, want map[string]int64) map[string]any {
	t.Helper()
	if len(records) == 0 {
		t.Fatal("purge task completed without executing any deletion SQL")
	}
	cutoff := records[0].Cutoff
	if cutoff.Before(started.Add(-60*24*time.Hour)) || cutoff.After(ended.Add(-60*24*time.Hour)) {
		t.Fatalf("purge cutoff outside this app's business clock: %s not %s..%s", cutoff, started, ended)
	}
	counts := map[string]int64{}
	ms := map[string]float64{}
	var order []string
	for _, r := range records {
		if r.Error != "" || r.User != role || r.Limit != 1000 || r.Rows < 0 || r.Rows > 1000 || r.PID == 0 || r.Ended.IsZero() || r.Ended.Before(r.Started) || r.Started.Before(started) || r.Ended.After(ended) || !r.Cutoff.Equal(cutoff) {
			t.Fatalf("invalid purge SQL evidence: %+v", r)
		}
		if len(order) == 0 || order[len(order)-1] != r.Table {
			order = append(order, r.Table)
		}
		counts[r.Table] += r.Rows
		ms[r.Table] += float64(r.Ended.Sub(r.Started)) / float64(time.Millisecond)
	}
	// Zero-row workspace statements still execute; their rows are retained.
	for table, n := range want {
		if got, ok := counts[table]; !ok || got != n {
			t.Errorf("purge direct DELETE %s=%d present=%t want=%d", table, got, ok, n)
		}
	}
	if len(counts) != len(want) {
		t.Errorf("purge statement table set=%v want=%v", counts, want)
	}
	return map[string]any{"cutoff": cutoff, "direct_rows": counts, "sql_ms_by_table": ms, "table_order": order, "statements": len(records)}
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
