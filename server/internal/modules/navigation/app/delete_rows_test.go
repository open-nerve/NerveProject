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

// fakeViews holds a deleted parent's undeleted view ids under each scope; a batch soft-deletes the first Limit after
// After, in a transaction only, and records which scope's statement it ran.
type fakeViews struct {
	live    map[app.DeletionScope][]uuid.UUID
	batches []app.RowBatch
	scopes  []app.DeletionScope
	inTx    bool
	fail    error
}

func (f *fakeViews) delete(scope app.DeletionScope, b app.RowBatch) ([]uuid.UUID, error) {
	if !f.inTx {
		return nil, errors.New("batch outside a transaction")
	}
	f.batches, f.scopes = append(f.batches, b), append(f.scopes, scope)
	if f.fail != nil {
		return nil, f.fail
	}
	var out []uuid.UUID
	for _, id := range f.live[scope] {
		if id.Compare(b.After) > 0 && len(out) < b.Limit {
			out = append(out, id)
		}
	}
	f.live[scope] = slices.DeleteFunc(f.live[scope], func(id uuid.UUID) bool { return slices.Contains(out, id) })
	return out, nil
}

func (f *fakeViews) DeleteProjectViews(_ context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.delete(app.ProjectScope, b)
}

func (f *fakeViews) DeleteWorkspaceViews(_ context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	return f.delete(app.WorkspaceScope, b)
}

type fakeTx struct{ views *fakeViews }

func (f fakeTx) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	f.views.inTx = true
	defer func() { f.views.inTx = false }()
	return fn(ctx)
}

func sortedIDs(n int) []uuid.UUID {
	out := make([]uuid.UUID, n)
	for i := range out {
		out[i] = uuid.NewV7()
	}
	slices.SortFunc(out, uuid.UUID.Compare)
	return out
}

// viewsUnder is a deleted parent's n undeleted views under scope, and as many under the other scope, which a
// deletion of scope must not reach.
func viewsUnder(scope app.DeletionScope, n int) *fakeViews {
	other := app.WorkspaceScope
	if scope == app.WorkspaceScope {
		other = app.ProjectScope
	}
	return &fakeViews{live: map[app.DeletionScope][]uuid.UUID{scope: sortedIDs(n), other: sortedIDs(n)}}
}

func newDeleteRows(views *fakeViews, now func() time.Time, batch int, budget time.Duration) *app.DeleteRows {
	return app.NewDeleteRows(app.DeleteRowsDeps{Rows: views, Tx: fakeTx{views}, RuntimeNow: now, Batch: batch, Budget: budget})
}

var (
	parent, by = uuid.NewV7(), uuid.NewV7()
	at         = time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
)

// Without waiting for anything, each batch of the scope's views is its own transaction, carries the deletion's
// original moment and actor and starts after the previous batch's greatest id; only an empty batch completes the work
// (P3b spec 2.7). A project's deletion runs the project's statement alone, a workspace's the workspace's own: the
// views of the workspace's projects are their projects' deletions'.
func TestDeleteRowsDrainsInBatches(t *testing.T) {
	for _, scope := range []app.DeletionScope{app.ProjectScope, app.WorkspaceScope} {
		t.Run(string(scope), func(t *testing.T) {
			views := viewsUnder(scope, 5)
			deleted := slices.Clone(views.live[scope])
			got, err := newDeleteRows(views, time.Now, 2, time.Hour).Execute(t.Context(), scope, parent, at, by)
			if err != nil || got != (app.DeleteRowsResult{Deleted: 5, Complete: true}) {
				t.Fatalf("Execute = %+v, %v", got, err)
			}
			afters := []uuid.UUID{{}, deleted[1], deleted[3], deleted[4]}
			if len(views.batches) != len(afters) || len(views.live[scope]) != 0 {
				t.Fatalf("batches=%+v live=%v", views.batches, views.live)
			}
			for i, b := range views.batches {
				if views.scopes[i] != scope || b.ParentID != parent || b.Limit != 2 || !b.At.Equal(at) || b.By != by || b.After != afters[i] {
					t.Errorf("batch %d = %s %+v; want %s after %s at %s by %s", i, views.scopes[i], b, scope, afters[i], at, by)
				}
			}
			for s, ids := range views.live {
				if s != scope && len(ids) != 5 {
					t.Errorf("the %s views: %d left of 5; want none deleted", s, len(ids))
				}
			}
		})
	}
	failed := errors.New("lock timeout")
	views := viewsUnder(app.ProjectScope, 3)
	views.fail = failed
	if _, err := newDeleteRows(views, time.Now, 2, time.Hour).Execute(t.Context(), app.ProjectScope, parent, at, by); !errors.Is(err, failed) {
		t.Fatalf("a failed batch = %v", err)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	views = viewsUnder(app.WorkspaceScope, 3)
	if _, err := newDeleteRows(views, time.Now, 2, time.Hour).Execute(cancelled, app.WorkspaceScope, parent, at, by); !errors.Is(err, context.Canceled) || len(views.batches) != 0 {
		t.Fatalf("a cancelled run=%v batches=%v", err, views.batches)
	}
}

// At the budget the job stops before the next batch, incomplete, without writing: River snoozes it and a later run
// continues.
func TestDeleteRowsStopsAtItsBudget(t *testing.T) {
	views := viewsUnder(app.ProjectScope, 3)
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
	got, err := newDeleteRows(views, now, 2, time.Minute).Execute(t.Context(), app.ProjectScope, parent, at, by)
	if err != nil || got != (app.DeleteRowsResult{Deleted: 2}) || len(views.live[app.ProjectScope]) != 1 {
		t.Fatalf("Execute = %+v, %v, %d left", got, err, len(views.live[app.ProjectScope]))
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
		views := viewsUnder(app.ProjectScope, 1)
		if err := run(newDeleteRows(views, time.Now, 2, time.Hour)); err == nil || len(views.batches) != 0 {
			t.Errorf("%s: err=%v batches=%v; want a refusal before any write", name, err, views.batches)
		}
	}
}
