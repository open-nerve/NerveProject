package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Real River first snoozes, then is stopped during its next attempt. A new
// runner and a different business clock must resume that exact job and cutoff.
func TestPurgeCursorRiverSnoozeAndRestart(t *testing.T) {
	pool := openPool(t, pgtest.NewDatabase(t))
	if _, err := pool.Exec(pgtest.Soon(t), "CREATE TABLE purge_resume_calls(n bigint GENERATED ALWAYS AS IDENTITY, after_id uuid, cutoff timestamptz)"); err != nil {
		t.Fatal(err)
	}
	inserter, err := jobs.NewInserter(pool)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err := inserter.InsertTx(pgtest.Soon(t), tx, purgeArgs{}, &river.InsertOpts{Queue: "periodic"}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
	var jobID int64
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT id FROM river_job WHERE kind=$1", purgeArgs{}.Kind()).Scan(&jobID); err != nil {
		t.Fatal(err)
	}
	low, high := uuid.NewV7(), uuid.NewV7()
	if low.Compare(high) > 0 {
		low, high = high, low
	}
	business := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	var elapsed atomic.Int64
	runtimeNow := func() time.Time { return business.Add(time.Duration(elapsed.Load())) }
	var calls atomic.Int32
	record := func(ctx context.Context, cursor purgePageCursor, cutoff time.Time) error {
		_, err := pool.Exec(ctx, "INSERT INTO purge_resume_calls(after_id,cutoff) VALUES($1,$2)", cursor.After, cutoff)
		return err
	}
	firstRows := purgeFunc(func(ctx context.Context, cutoff time.Time, _ int, cursor purgePageCursor) (purgePageProgress, error) {
		if err := record(ctx, cursor, cutoff); err != nil {
			return purgePageProgress{Cursor: cursor}, err
		}
		if calls.Add(1) == 1 {
			elapsed.Add(int64(time.Second))
			return purgePageProgress{Cursor: purgePageCursor{"issues", low}, Scanned: 1, Blocked: 1}, nil
		}
		<-ctx.Done()
		return purgePageProgress{Cursor: cursor}, ctx.Err()
	})
	var logs lockedBuffer
	start := func(uc *purgeDeleted) *jobs.Runner {
		t.Helper()
		runner, err := jobs.New(pool, jobs.Config{DefaultWorkers: 1, PeriodicWorkers: 1, ShutdownTimeout: 100 * time.Millisecond, Logger: slog.New(slog.NewTextHandler(&logs, nil))}, []jobs.Job{{Worker: jobs.Register(&purgeWorker{uc: uc})}}, []string{purgeArgs{}.Kind()})
		if err != nil {
			t.Fatal(err)
		}
		runner.Start(t.Context())
		return runner
	}
	stop := func(r *jobs.Runner) {
		t.Helper()
		if err := r.Stop(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	first := start(newPurgeDeleted([]purgeModule{{"issue", firstRows}}, clockAt(business), runtimeNow, 60*24*time.Hour, 1, time.Second))
	stopped := false
	defer func() {
		if !stopped {
			stop(first)
		}
	}()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	var state string
	awaitPurge(t, ctx, &logs, "second attempt or erroneous early completion", func() bool {
		if err := pool.QueryRow(ctx, "SELECT state FROM river_job WHERE id=$1", jobID).Scan(&state); err != nil {
			t.Fatal(err)
		}
		return calls.Load() >= 2 || state == "completed"
	})
	if calls.Load() != 2 {
		t.Fatalf("budget work completed without durable resume: calls=%d state=%s", calls.Load(), state)
	}
	var metadata []byte
	if err := pool.QueryRow(ctx, "SELECT metadata FROM river_job WHERE id=$1", jobID).Scan(&metadata); err != nil {
		t.Fatal(err)
	}
	var output struct {
		Output purgeResult `json:"output"`
	}
	if err := json.Unmarshal(metadata, &output); err != nil {
		t.Fatal(err)
	}
	if output.Output.Cursor.After != low || output.Output.Cursor.Cutoff != business.Add(-60*24*time.Hour) || output.Output.Complete {
		t.Fatalf("snoozed metadata=%s", metadata)
	}
	stop(first)
	stopped = true
	secondRows := purgeFunc(func(ctx context.Context, cutoff time.Time, _ int, cursor purgePageCursor) (purgePageProgress, error) {
		if err := record(ctx, cursor, cutoff); err != nil {
			return purgePageProgress{Cursor: cursor}, err
		}
		elapsed.Add(int64(time.Second))
		switch cursor.After {
		case low:
			return purgePageProgress{Cursor: purgePageCursor{"issues", high}, Scanned: 1, Deleted: 1}, nil
		case high:
			return purgePageProgress{Cursor: cursor, Done: true}, nil
		default:
			return purgePageProgress{Cursor: cursor}, fmt.Errorf("lost persisted position: %+v", cursor)
		}
	})
	second := start(newPurgeDeleted([]purgeModule{{"issue", secondRows}}, clockAt(business.Add(30*24*time.Hour)), runtimeNow, 60*24*time.Hour, 1, time.Second))
	defer stop(second)
	awaitPurge(t, ctx, &logs, "same job completed after restart", func() bool {
		if err := pool.QueryRow(ctx, "SELECT state,metadata FROM river_job WHERE id=$1", jobID).Scan(&state, &metadata); err != nil {
			t.Fatal(err)
		}
		return state == "completed"
	})
	if err := json.Unmarshal(metadata, &output); err != nil {
		t.Fatal(err)
	}
	if !output.Output.Complete || output.Output.Cursor.After != high || output.Output.Cursor.Cutoff != business.Add(-60*24*time.Hour) {
		t.Fatalf("completed output=%s", metadata)
	}
	var ids []uuid.UUID
	var cutoffs []time.Time
	if err := pool.QueryRow(ctx, "SELECT array_agg(after_id ORDER BY n),array_agg(cutoff ORDER BY n) FROM purge_resume_calls").Scan(&ids, &cutoffs); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ids, []uuid.UUID{uuid.Nil(), low, low, high}) {
		t.Fatalf("resumed positions=%v", ids)
	}
	for _, cutoff := range cutoffs {
		if !cutoff.Equal(business.Add(-60 * 24 * time.Hour)) {
			t.Fatalf("cutoff moved: %v", cutoffs)
		}
	}
}
