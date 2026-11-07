package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
)

// fakeCycles holds one project's undeleted cycle ids; a batch soft-deletes the first Limit after After.
type fakeCycles struct {
	live      []uuid.UUID
	batches   []app.CycleBatch
	remaining []uuid.UUID
	inTx      bool
	fail      error
}

func (f *fakeCycles) DeleteProjectCycles(_ context.Context, b app.CycleBatch) ([]uuid.UUID, error) {
	if !f.inTx {
		return nil, errors.New("batch outside a transaction")
	}
	f.batches = append(f.batches, b)
	if f.fail != nil {
		return nil, f.fail
	}
	var out []uuid.UUID
	for _, id := range f.live {
		if id.Compare(b.After) > 0 && len(out) < b.Limit {
			out = append(out, id)
		}
	}
	f.live = slices.DeleteFunc(f.live, func(id uuid.UUID) bool { return slices.Contains(out, id) })
	return out, nil
}

func (f *fakeCycles) ProjectCyclesRemaining(_ context.Context, project, after uuid.UUID) (bool, error) {
	f.remaining = append(f.remaining, after)
	return slices.ContainsFunc(f.live, func(id uuid.UUID) bool { return id.Compare(after) > 0 }), nil
}

type fakeTx struct{ rows *fakeCycles }

func (f fakeTx) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	f.rows.inTx = true
	defer func() { f.rows.inTx = false }()
	return fn(ctx)
}

func ids(n int) []uuid.UUID {
	out := make([]uuid.UUID, n)
	for i := range out {
		out[i] = uuid.NewV7()
	}
	slices.SortFunc(out, uuid.UUID.Compare)
	return out
}

// Each batch is its own transaction, carries the deletion's original moment and actor, and starts after the
// previous batch's greatest id; only an empty batch ends the work.
func TestDeleteRowsDrainsInBatches(t *testing.T) {
	project, by := uuid.NewV7(), uuid.NewV7()
	at := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	cycles := ids(5)
	rows := &fakeCycles{live: slices.Clone(cycles)}
	uc := app.NewDeleteRows(app.DeleteRowsDeps{Rows: rows, Tx: fakeTx{rows}, RuntimeNow: time.Now, Batch: 2, Budget: time.Hour})
	got, err := uc.Execute(t.Context(), project, at, by)
	if err != nil || got != (app.DeleteRowsResult{Deleted: 5, Complete: true}) {
		t.Fatalf("Execute = %+v, %v", got, err)
	}
	afters := []uuid.UUID{{}, cycles[1], cycles[3], cycles[4]}
	if len(rows.batches) != len(afters) {
		t.Fatalf("batches=%+v", rows.batches)
	}
	for i, b := range rows.batches {
		if b.ProjectID != project || b.Limit != 2 || !b.At.Equal(at) || b.By != by || b.After != afters[i] {
			t.Errorf("batch %d = %+v; want after %s at %s by %s", i, b, afters[i], at, by)
		}
	}
}

// At the budget the job stops before the next batch and reports whether rows remain after its cursor, without
// writing: River snoozes it and a later run continues.
func TestDeleteRowsStopsAtItsBudget(t *testing.T) {
	project, by := uuid.NewV7(), uuid.NewV7()
	at := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name     string
		cycles   int
		complete bool
	}{{"rows remain", 3, false}, {"nothing remains", 2, true}} {
		t.Run(tt.name, func(t *testing.T) {
			all := ids(tt.cycles)
			rows := &fakeCycles{live: slices.Clone(all)}
			start := time.Now()
			calls := 0
			// The first reading starts the budget; the second, before the first batch, is within it; the third,
			// before the second batch, is past it.
			now := func() time.Time {
				calls++
				if calls < 3 {
					return start
				}
				return start.Add(time.Minute)
			}
			uc := app.NewDeleteRows(app.DeleteRowsDeps{Rows: rows, Tx: fakeTx{rows}, RuntimeNow: now, Batch: 2, Budget: time.Minute})
			got, err := uc.Execute(t.Context(), project, at, by)
			if err != nil || got != (app.DeleteRowsResult{Deleted: 2, Complete: tt.complete}) {
				t.Fatalf("Execute = %+v, %v", got, err)
			}
			// The probe reads after the committed batch's greatest id.
			if len(rows.batches) != 1 || len(rows.remaining) != 1 || rows.remaining[0] != all[1] {
				t.Fatalf("batches=%d remaining probes=%v", len(rows.batches), rows.remaining)
			}
		})
	}
}

// A failed batch, invalid audit data or a cancelled context stops the job with an error, River retrying it.
func TestDeleteRowsFails(t *testing.T) {
	project, by := uuid.NewV7(), uuid.NewV7()
	at := time.Now()
	failed := errors.New("write failed")
	rows := &fakeCycles{live: ids(1), fail: failed}
	uc := app.NewDeleteRows(app.DeleteRowsDeps{Rows: rows, Tx: fakeTx{rows}, RuntimeNow: time.Now, Batch: 2, Budget: time.Hour})
	if got, err := uc.Execute(t.Context(), project, at, by); !errors.Is(err, failed) || got.Complete {
		t.Fatalf("failed batch = %+v, %v", got, err)
	}
	for name, args := range map[string]struct {
		project, by uuid.UUID
		at          time.Time
	}{"project": {uuid.UUID{}, by, at}, "actor": {project, uuid.UUID{}, at}, "moment": {project, by, time.Time{}}} {
		if _, err := uc.Execute(t.Context(), args.project, args.at, args.by); err == nil {
			t.Errorf("invalid %s accepted", name)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	ok := &fakeCycles{live: ids(1)}
	if _, err := app.NewDeleteRows(app.DeleteRowsDeps{Rows: ok, Tx: fakeTx{ok}, RuntimeNow: time.Now, Batch: 2, Budget: time.Hour}).Execute(ctx, project, at, by); !errors.Is(err, context.Canceled) || len(ok.batches) != 0 {
		t.Fatalf("cancelled = %v, batches=%d", err, len(ok.batches))
	}
}
