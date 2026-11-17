package app_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
)

// fakeProject is one deleted project: whether its work items remain, and its undeleted notification ids; a batch
// soft-deletes the first Limit after After, in a transaction only. calls records the reads and batches in order.
type fakeProject struct {
	remain  bool
	live    []uuid.UUID
	batches []app.RowBatch
	calls   []string
	inTx    bool
	fail    error
	read    error
}

func (f *fakeProject) ProjectIssuesRemain(_ context.Context, project uuid.UUID) (bool, error) {
	f.calls = append(f.calls, "remain "+project.String())
	return f.remain, f.read
}

func (f *fakeProject) DeleteProjectNotifications(_ context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	if !f.inTx {
		return nil, errors.New("batch outside a transaction")
	}
	f.calls = append(f.calls, "batch")
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

type fakeTx struct{ rows *fakeProject }

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

func newDeleteRows(rows *fakeProject, now func() time.Time, batch int, budget time.Duration) *app.DeleteRows {
	return app.NewDeleteRows(app.DeleteRowsDeps{Issues: rows, Rows: rows, Tx: fakeTx{rows}, RuntimeNow: now, Batch: batch, Budget: budget})
}

var (
	project, by = uuid.NewV7(), uuid.NewV7()
	at          = time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
)

// While the project's work items remain, issue.delete_rows still deleting them, the job writes nothing and waits
// (M7 P1b spec 2.8, R3), until an hour after its enqueueing: from then on each run fails, for River's retries and
// attempt limit, as a snooze consumes no attempt.
func TestDeleteRowsWaitsForTheWorkItemsAnHourAtMost(t *testing.T) {
	now := time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name   string
		waited time.Duration
		fails  bool
	}{{"just enqueued", 0, false}, {"short of the hour", time.Hour - time.Nanosecond, false}, {"the hour", time.Hour, true}, {"past it", 3 * time.Hour, true}} {
		t.Run(tt.name, func(t *testing.T) {
			rows := &fakeProject{remain: true, live: ids(3)}
			got, err := newDeleteRows(rows, func() time.Time { return now }, 2, time.Hour).Execute(t.Context(), project, at, by, now.Add(-tt.waited))
			if tt.fails {
				if err == nil || !strings.Contains(err.Error(), project.String()) || got != (app.DeleteRowsResult{}) {
					t.Fatalf("Execute = %+v, %v; want a failure naming the project", got, err)
				}
			} else if err != nil || got != (app.DeleteRowsResult{Waiting: true}) {
				t.Fatalf("Execute = %+v, %v; want to wait", got, err)
			}
			if !slices.Equal(rows.calls, []string{"remain " + project.String()}) || len(rows.live) != 3 {
				t.Fatalf("calls=%v live=%d; want the work items asked about and nothing written", rows.calls, len(rows.live))
			}
		})
	}
	failed := errors.New("database unavailable")
	rows := &fakeProject{read: failed, live: ids(1)}
	if _, err := newDeleteRows(rows, func() time.Time { return now }, 2, time.Hour).Execute(t.Context(), project, at, by, now); !errors.Is(err, failed) || len(rows.batches) != 0 {
		t.Fatalf("a failed read=%v batches=%v; want the failure, nothing written", err, rows.batches)
	}
}

// Once the work items are deleted, each batch is its own transaction, carries the deletion's original moment and
// actor and starts after the previous batch's greatest id; only an empty batch completes the work.
func TestDeleteRowsDrainsInBatches(t *testing.T) {
	notes := ids(5)
	rows := &fakeProject{live: slices.Clone(notes)}
	got, err := newDeleteRows(rows, time.Now, 2, time.Hour).Execute(t.Context(), project, at, by, time.Now().Add(-2*time.Hour))
	if err != nil || got != (app.DeleteRowsResult{Deleted: 5, Complete: true}) {
		t.Fatalf("Execute = %+v, %v", got, err)
	}
	afters := []uuid.UUID{{}, notes[1], notes[3], notes[4]}
	if len(rows.batches) != len(afters) || rows.calls[0] != "remain "+project.String() {
		t.Fatalf("calls=%v batches=%+v", rows.calls, rows.batches)
	}
	for i, b := range rows.batches {
		if b.ProjectID != project || b.Limit != 2 || !b.At.Equal(at) || b.By != by || b.After != afters[i] {
			t.Errorf("batch %d = %+v; want after %s at %s by %s", i, b, afters[i], at, by)
		}
	}
	failed := errors.New("lock timeout")
	rows = &fakeProject{live: ids(3), fail: failed}
	if _, err := newDeleteRows(rows, time.Now, 2, time.Hour).Execute(t.Context(), project, at, by, time.Now()); !errors.Is(err, failed) {
		t.Fatalf("a failed batch = %v", err)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	rows = &fakeProject{live: ids(3)}
	if _, err := newDeleteRows(rows, time.Now, 2, time.Hour).Execute(cancelled, project, at, by, time.Now()); !errors.Is(err, context.Canceled) || len(rows.batches) != 0 {
		t.Fatalf("a cancelled run=%v batches=%v", err, rows.batches)
	}
}

// At the budget the job stops before the next batch, incomplete, without writing: River snoozes it and a later run
// continues.
func TestDeleteRowsStopsAtItsBudget(t *testing.T) {
	rows := &fakeProject{live: ids(3)}
	start := time.Now()
	calls := 0
	// The first reading starts the budget; the second, before the first batch, is within it; the third, before the
	// second batch, is past it.
	now := func() time.Time {
		calls++
		if calls < 3 {
			return start
		}
		return start.Add(time.Minute)
	}
	got, err := newDeleteRows(rows, now, 2, time.Minute).Execute(t.Context(), project, at, by, start)
	if err != nil || got != (app.DeleteRowsResult{Deleted: 2}) || len(rows.live) != 1 {
		t.Fatalf("Execute = %+v, %v, %d left", got, err, len(rows.live))
	}
}

// A job without its project, its deletion's moment and actor or its enqueueing reads and writes nothing.
func TestDeleteRowsRefusesIncompleteArguments(t *testing.T) {
	for name, run := range map[string]func(*app.DeleteRows) error{
		"project": func(u *app.DeleteRows) error { _, err := u.Execute(t.Context(), uuid.UUID{}, at, by, at); return err },
		"actor": func(u *app.DeleteRows) error {
			_, err := u.Execute(t.Context(), project, at, uuid.UUID{}, at)
			return err
		},
		"moment": func(u *app.DeleteRows) error {
			_, err := u.Execute(t.Context(), project, time.Time{}, by, at)
			return err
		},
		"enqueued": func(u *app.DeleteRows) error {
			_, err := u.Execute(t.Context(), project, at, by, time.Time{})
			return err
		},
	} {
		rows := &fakeProject{live: ids(1)}
		if err := run(newDeleteRows(rows, time.Now, 2, time.Hour)); err == nil || len(rows.calls) != 0 {
			t.Errorf("%s: err=%v calls=%v; want a refusal before any read", name, err, rows.calls)
		}
	}
}
