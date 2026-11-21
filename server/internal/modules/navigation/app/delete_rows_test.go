package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
)

// fakeRows holds a deleted parent's undeleted row ids under each statement of either scope; a batch soft-deletes the
// first Limit after After, in a transaction only, and records which statement it ran.
type fakeRows struct {
	live    map[string][]uuid.UUID
	batches []app.RowBatch
	ran     []string
	inTx    bool
	fail    error
}

func (f *fakeRows) delete(statement string, b app.RowBatch) ([]uuid.UUID, error) {
	if !f.inTx {
		return nil, errors.New("batch outside a transaction")
	}
	f.batches, f.ran = append(f.batches, b), append(f.ran, statement)
	if f.fail != nil {
		return nil, f.fail
	}
	var out []uuid.UUID
	for _, id := range f.live[statement] {
		if id.Compare(b.After) > 0 && len(out) < b.Limit {
			out = append(out, id)
		}
	}
	f.live[statement] = slices.DeleteFunc(f.live[statement], func(id uuid.UUID) bool { return slices.Contains(out, id) })
	return out, nil
}

func (f *fakeRows) DeleteProjectViews(_ context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.delete("project views", b)
}

func (f *fakeRows) DeleteWorkspaceViews(_ context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.delete("workspace views", b)
}

func (f *fakeRows) DeleteProjectFavorites(_ context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.delete("project favorites", b)
}

func (f *fakeRows) DeleteWorkspaceFavoriteFolders(_ context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.delete("workspace folders", b)
}

func (f *fakeRows) DeleteWorkspaceFavoriteItems(_ context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.delete("workspace favorite items", b)
}

type fakeTx struct{ rows *fakeRows }

func (f fakeTx) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	f.rows.inTx = true
	defer func() { f.rows.inTx = false }()
	return fn(ctx)
}

// statements are each scope's statements in the deletion's order (P3c spec 2.6, R11): the views, then the favorites;
// a workspace's own favorites its folders first, then the others without a project (the total order of favorite rows,
// spec 2.8).
var statements = map[app.DeletionScope][]string{
	app.ProjectScope:   {"project views", "project favorites"},
	app.WorkspaceScope: {"workspace views", "workspace folders", "workspace favorite items"},
}

func sortedIDs(n int) []uuid.UUID {
	out := make([]uuid.UUID, n)
	for i := range out {
		out[i] = uuid.NewV7()
	}
	slices.SortFunc(out, uuid.UUID.Compare)
	return out
}

// rowsUnder is a deleted parent's n undeleted rows under each statement of either scope, a later step's ids below an
// earlier one's: a cursor carried from one step into the next would pass over them.
func rowsUnder(n int) *fakeRows {
	f := &fakeRows{live: map[string][]uuid.UUID{}}
	for _, steps := range statements {
		for _, s := range slices.Backward(steps) {
			f.live[s] = sortedIDs(n)
		}
	}
	return f
}

func newDeleteRows(rows *fakeRows, now func() time.Time, batch int, budget time.Duration) *app.DeleteRows {
	return app.NewDeleteRows(app.DeleteRowsDeps{Views: rows, Favorites: rows, Tx: fakeTx{rows}, RuntimeNow: now, Batch: batch, Budget: budget})
}

var (
	parent, by = uuid.NewV7(), uuid.NewV7()
	at         = time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
)

// ranBatch is a batch as the fake saw it: its statement and its cursor.
type ranBatch struct {
	statement string
	after     uuid.UUID
}

func ran(rows *fakeRows) []ranBatch {
	out := make([]ranBatch, len(rows.batches))
	for i, b := range rows.batches {
		out[i] = ranBatch{rows.ran[i], b.After}
	}
	return out
}

// Without waiting for anything, the scope's steps run in order, each draining its table in batches: each batch its own
// transaction, carrying the deletion's original moment and actor, and starting after the previous batch of its step's
// greatest id, a step's first from the start; only an empty batch ends a step, and the last step's completes the work
// (P3b spec 2.7, P3c spec 2.6). A project's deletion runs the project's statements alone, a workspace's the
// workspace's own: the rows of the workspace's projects are their projects' deletions'.
func TestDeleteRowsDrainsInBatches(t *testing.T) {
	for _, scope := range []app.DeletionScope{app.ProjectScope, app.WorkspaceScope} {
		t.Run(string(scope), func(t *testing.T) {
			rows := rowsUnder(3)
			var want []ranBatch
			for _, s := range statements[scope] {
				ids := rows.live[s]
				want = append(want, ranBatch{s, uuid.UUID{}}, ranBatch{s, ids[1]}, ranBatch{s, ids[2]})
			}
			got, err := newDeleteRows(rows, time.Now, 2, time.Hour).Execute(t.Context(), scope, parent, at, by)
			if err != nil || got != (app.DeleteRowsResult{Deleted: 3 * len(statements[scope]), Complete: true}) {
				t.Fatalf("Execute = %+v, %v", got, err)
			}
			if !slices.Equal(ran(rows), want) {
				t.Fatalf("batches %+v; want %+v", ran(rows), want)
			}
			for i, b := range rows.batches {
				if b.ParentID != parent || b.Limit != 2 || !b.At.Equal(at) || b.By != by {
					t.Errorf("batch %d = %+v; want of %s at %s by %s", i, b, parent, at, by)
				}
			}
			for s, ids := range rows.live {
				if mine := slices.Contains(statements[scope], s); mine && len(ids) != 0 || !mine && len(ids) != 3 {
					t.Errorf("the %s: %d left of 3; want none of the scope's, all of the other's", s, len(ids))
				}
			}
		})
	}
	failed := errors.New("lock timeout")
	rows := rowsUnder(3)
	rows.fail = failed
	if _, err := newDeleteRows(rows, time.Now, 2, time.Hour).Execute(t.Context(), app.ProjectScope, parent, at, by); !errors.Is(err, failed) || len(rows.batches) != 1 {
		t.Fatalf("a failed batch = %v after %d batches; want its failure, at once", err, len(rows.batches))
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	rows = rowsUnder(3)
	if _, err := newDeleteRows(rows, time.Now, 2, time.Hour).Execute(cancelled, app.WorkspaceScope, parent, at, by); !errors.Is(err, context.Canceled) || len(rows.batches) != 0 {
		t.Fatalf("a cancelled run=%v batches=%v", err, rows.batches)
	}
}

// The steps share one budget: at it the job stops before the next batch, incomplete, without writing, whichever step
// it is in, and River snoozes it. A later run scans from the start: a step done before finds nothing left, and the
// step it stopped in starts over from the start and continues with what is left.
func TestDeleteRowsStopsAtItsBudget(t *testing.T) {
	rows := rowsUnder(3)
	start := time.Now()
	calls := 0
	// Each reading ten seconds after the one before: the first starts the 45 seconds; those before the views' three
	// batches and the favorites' first are within them; the next, before the favorites' second batch, is past them.
	now := func() time.Time {
		calls++
		return start.Add(time.Duration(calls) * 10 * time.Second)
	}
	got, err := newDeleteRows(rows, now, 2, 45*time.Second).Execute(t.Context(), app.ProjectScope, parent, at, by)
	if err != nil || got != (app.DeleteRowsResult{Deleted: 5}) || len(rows.live["project views"]) != 0 || len(rows.live["project favorites"]) != 1 {
		t.Fatalf("Execute = %+v, %v, %v left; want the views and two favorites deleted, one favorite left", got, err, rows.live)
	}
	left := rows.live["project favorites"][0]
	rows.batches, rows.ran = nil, nil
	got, err = newDeleteRows(rows, time.Now, 2, time.Hour).Execute(t.Context(), app.ProjectScope, parent, at, by)
	want := []ranBatch{{"project views", uuid.UUID{}}, {"project favorites", uuid.UUID{}}, {"project favorites", left}}
	if err != nil || got != (app.DeleteRowsResult{Deleted: 1, Complete: true}) || !slices.Equal(ran(rows), want) {
		t.Fatalf("the later run = %+v, %v, batches %+v; want %+v", got, err, ran(rows), want)
	}
}

// A job without a known scope, its parent or its deletion's moment and actor writes nothing.
func TestDeleteRowsRefusesIncompleteArguments(t *testing.T) {
	for name, run := range map[string]func(*app.DeleteRows) error{
		"scope": func(u *app.DeleteRows) error {
			_, err := u.Execute(t.Context(), "workspace_drafts", parent, at, by)
			return err
		},
		"parent": func(u *app.DeleteRows) error {
			_, err := u.Execute(t.Context(), app.ProjectScope, uuid.UUID{}, at, by)
			return err
		},
		"actor": func(u *app.DeleteRows) error {
			_, err := u.Execute(t.Context(), app.WorkspaceScope, parent, at, uuid.UUID{})
			return err
		},
		"moment": func(u *app.DeleteRows) error {
			_, err := u.Execute(t.Context(), app.ProjectScope, parent, time.Time{}, by)
			return err
		},
	} {
		rows := rowsUnder(1)
		if err := run(newDeleteRows(rows, time.Now, 2, time.Hour)); err == nil || len(rows.batches) != 0 {
			t.Errorf("%s: err=%v batches=%v; want a refusal before any write", name, err, rows.batches)
		}
	}
}
