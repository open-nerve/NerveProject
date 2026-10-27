package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

type purgeControlArgs struct{}

func (purgeControlArgs) Kind() string { return "test.purge_control" }

type purgeControlWorker struct {
	river.WorkerDefaults[purgeControlArgs]
}

func (*purgeControlWorker) Work(context.Context, *river.Job[purgeControlArgs]) error { return nil }

// The runner and periodic producer are real. Only the module boundary is held,
// so removing production UniqueOpts exposes another active job immediately.
func TestPurgePeriodicRunsDoNotOverlap(t *testing.T) {
	for _, retry := range []bool{false, true} {
		name := "running"
		if retry {
			name = "retryable"
		}
		t.Run(name, func(t *testing.T) {
			pool := openPool(t, pgtest.NewDatabase(t))
			if retry {
				// Persist the short available transition: a slow test process
				// must not need to poll within River's one-second first retry.
				for _, statement := range []string{
					"CREATE TABLE purge_first_failure (job_id bigint PRIMARY KEY)",
					`CREATE FUNCTION observe_first_purge_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO purge_first_failure VALUES (NEW.id) ON CONFLICT DO NOTHING; RETURN NEW; END $$`,
					`CREATE TRIGGER observe_first_purge_failure AFTER UPDATE OF state ON river_job FOR EACH ROW WHEN (NEW.kind='cleanup.purge_deleted' AND NEW.state='available' AND NEW.attempt=1 AND cardinality(NEW.errors)=1) EXECUTE FUNCTION observe_first_purge_failure()`,
				} {
					if _, err := pool.Exec(pgtest.Soon(t), statement); err != nil {
						t.Fatal(err)
					}
				}
			}
			ctx, cancel := context.WithTimeout(t.Context(), 35*time.Second)
			defer cancel()
			release := make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			var calls, current, maximum atomic.Int32
			entered := make(chan int32, 32)
			uc := newPurgeDeleted([]Purger{purgeFunc(func(workctx context.Context, _ time.Time, _ int) (int, error) {
				n := calls.Add(1)
				active := current.Add(1)
				defer current.Add(-1)
				for old := maximum.Load(); active > old; old = maximum.Load() {
					if maximum.CompareAndSwap(old, active) {
						break
					}
				}
				entered <- n
				// River makes the default first short retry available. The second
				// real failure of this same job receives the longer retryable backoff.
				if retry && n <= 2 {
					return 0, errors.New("purge retry fixture")
				}
				select {
				case <-release:
					return 1, nil
				case <-workctx.Done():
					return 0, workctx.Err()
				case <-ctx.Done():
					return 0, ctx.Err()
				}
			})}, newClock(testConfig(t, "", false)), 60*24*time.Hour, 1)
			control := jobs.Job{Worker: jobs.Register(&purgeControlWorker{}), Periodic: &jobs.PeriodicSpec{
				Args: purgeControlArgs{}, ID: "test.purge_control", Interval: time.Second, RunOnStart: true, InsertOpts: &river.InsertOpts{Queue: "periodic"},
			}}
			var logs lockedBuffer
			runner, err := jobs.New(pool, jobs.Config{DefaultWorkers: 1, PeriodicWorkers: 4, ShutdownTimeout: time.Second, Logger: slog.New(slog.NewTextHandler(&logs, nil))}, []jobs.Job{purgeJob(uc, time.Second), control}, nil)
			if err != nil {
				t.Fatal(err)
			}
			runner.Start(ctx)
			defer func() {
				unblock()
				stopctx, stopcancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer stopcancel()
				if err := runner.Stop(stopctx); err != nil {
					t.Errorf("stop: %v", err)
				}
			}()
			awaitPurge(t, ctx, &logs, "first module entry", func() bool {
				select {
				case <-entered:
					return true
				default:
					return false
				}
			})
			var first int64
			state := "running"
			if retry {
				state = "retryable"
			}
			awaitPurge(t, ctx, &logs, "first "+state, func() bool {
				rows := purgeActive(t, ctx, pool)
				if len(rows) > 1 {
					t.Fatalf("second active purge before observation: %v", rows)
				}
				if len(rows) == 1 && rows[0].state == state {
					first = rows[0].id
					return true
				}
				return false
			})
			if retry {
				var sawFirstAvailable bool
				if err := pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM purge_first_failure WHERE job_id=$1)", first).Scan(&sawFirstAvailable); err != nil {
					t.Fatal(err)
				}
				if !sawFirstAvailable {
					t.Fatal("first failed attempt was not observed available before retryable")
				}
				t.Logf("originalID=%d: real failure attempt=1 available, attempt=2 retryable", first)
				var id int64
				// Keep the real failed job beyond the observation window without changing
				// production retry policy. The state predicate rejects a racing retry.
				err := pool.QueryRow(ctx, `UPDATE river_job SET scheduled_at=now()+interval '1 minute' WHERE id=$1 AND state='retryable' AND attempt=2 AND cardinality(errors)=2 RETURNING id`, first).Scan(&id)
				if err != nil || id != first {
					t.Fatalf("defer real retry %d: id=%d err=%v", first, id, err)
				}
			}
			var boundary time.Time
			if err := pool.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&boundary); err != nil {
				t.Fatal(err)
			}
			var periods int
			awaitPurge(t, ctx, &logs, "two independent control periods", func() bool {
				active := purgeActive(t, ctx, pool)
				if len(active) != 1 || active[0].id != first || maximum.Load() != 1 {
					t.Fatalf("second active/concurrent purge: active=%v first=%d maxConcurrent=%d", active, first, maximum.Load())
				}
				if active[0].state != state {
					t.Fatalf("original purge left %s across periods: %v", state, active)
				}
				if err := pool.QueryRow(ctx, `SELECT count(DISTINCT scheduled_at) FROM river_job WHERE kind='test.purge_control' AND state='completed' AND scheduled_at>$1`, boundary).Scan(&periods); err != nil {
					t.Fatal(err)
				}
				return periods >= 2
			})
			t.Logf("%s across %d independent periods: originalID=%d active=1 maxConcurrent=%d", name, periods, first, maximum.Load())
			if retry {
				if calls.Load() != 2 {
					t.Fatalf("retry ran during deferral: calls=%d", calls.Load())
				}
				tag, err := pool.Exec(ctx, "UPDATE river_job SET scheduled_at=now() WHERE id=$1 AND state='retryable'", first)
				if err != nil || tag.RowsAffected() != 1 {
					t.Fatalf("release retry: %v %v", tag, err)
				}
				awaitPurge(t, ctx, &logs, "same job retries", func() bool {
					active := purgeActive(t, ctx, pool)
					if len(active) != 1 || active[0].id != first || maximum.Load() != 1 {
						t.Fatalf("retry replaced or overlapped: %v first=%d max=%d", active, first, maximum.Load())
					}
					return active[0].state == "running" && active[0].attempt == 3 && calls.Load() == 3
				})
				t.Logf("retry uses originalID=%d attempt=3", first)
			}
			unblock()
			awaitPurge(t, ctx, &logs, "original completes", func() bool {
				var completed bool
				if err := pool.QueryRow(ctx, "SELECT state='completed' FROM river_job WHERE id=$1", first).Scan(&completed); err != nil {
					t.Fatal(err)
				}
				return completed
			})
			var next int64
			awaitPurge(t, ctx, &logs, "new job after completed", func() bool {
				if err := pool.QueryRow(ctx, `SELECT coalesce(min(id),0) FROM river_job WHERE kind='cleanup.purge_deleted' AND id<>$1 AND state='completed'`, first).Scan(&next); err != nil {
					t.Fatal(err)
				}
				return next != 0
			})
			t.Logf("completed originalID=%d allows nextID=%d", first, next)
		})
	}
}

type activePurge struct {
	id      int64
	state   string
	attempt int
}

func purgeActive(t *testing.T, ctx context.Context, pool *pgxpool.Pool) []activePurge {
	t.Helper()
	rows, err := pool.Query(ctx, `SELECT id,state::text,attempt FROM river_job WHERE kind='cleanup.purge_deleted' AND state IN ('available','pending','running','retryable','scheduled') ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var active []activePurge
	for rows.Next() {
		var job activePurge
		if err := rows.Scan(&job.id, &job.state, &job.attempt); err != nil {
			t.Fatal(err)
		}
		active = append(active, job)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return active
}
func awaitPurge(t *testing.T, ctx context.Context, logs *lockedBuffer, condition string, ready func() bool) {
	t.Helper()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		if ready() {
			return
		}
		select {
		case <-tick.C:
		case <-ctx.Done():
			t.Fatalf("waiting for %s: %v; logs:\n%s", condition, ctx.Err(), logs.String())
		}
	}
}
