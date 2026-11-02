package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"
)

type purgeClockProbe struct {
	now   time.Time
	calls int
}

func (c *purgeClockProbe) Now() time.Time {
	c.calls++
	return c.now.Add(time.Duration(c.calls-1) * time.Hour)
}

type purgeFunc func(context.Context, time.Time, int, purgePageCursor) (purgePageProgress, error)

func (f purgeFunc) Page(ctx context.Context, cutoff time.Time, batch int, cursor purgePageCursor) (purgePageProgress, error) {
	return f(ctx, cutoff, batch, cursor)
}

type purgeContextKey struct{}

func TestPurgeDeletedUsesOneCutoffAndModuleOrder(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			clock := &purgeClockProbe{now: time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)}
			lost := errors.New("later page failed")
			var order []string
			var modules []purgeModule
			ctx := context.WithValue(t.Context(), purgeContextKey{}, "kept")
			for i, name := range []string{"issue", "project", "workspace", "identity"} {
				modules = append(modules, purgeModule{name, purgeFunc(func(ctx context.Context, cutoff time.Time, batch int, _ purgePageCursor) (purgePageProgress, error) {
					order = append(order, name)
					if ctx.Value(purgeContextKey{}) != "kept" || cutoff != time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC) || batch != 13 {
						t.Errorf("%s cutoff=%s batch=%d", name, cutoff, batch)
					}
					if fail && i == 1 {
						return purgePageProgress{Scanned: 3, Deleted: 3}, fmt.Errorf("project: %w", lost)
					}
					n := []int{2, 5, 7, 11}[i]
					return purgePageProgress{Scanned: n, Deleted: n, Done: true}, nil
				})})
			}
			got, err := newPurgeDeleted(modules, clock, time.Now, 60*24*time.Hour, 13, time.Second).Execute(ctx, purgeCursor{})
			wantOrder, wantTotal := []string{"issue", "project", "workspace", "identity"}, 25
			if fail {
				wantOrder, wantTotal = []string{"issue", "project"}, 5
			}
			if !slices.Equal(order, wantOrder) || got.Deleted != wantTotal || errors.Is(err, lost) != fail || clock.calls != 1 || got.Complete == fail {
				t.Fatalf("order=%v got=%+v err=%v clockCalls=%d", order, got, err, clock.calls)
			}
		})
	}
}

func TestPurgeCursorBudgetResumeAndError(t *testing.T) {
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	realNow := now
	clock := &purgeClockProbe{now: now}
	lost := errors.New("partial failure")
	a, b := uuid.NewV7(), uuid.NewV7()
	if a.Compare(b) > 0 {
		a, b = b, a
	}
	calls := 0
	p := newPurgeDeleted([]purgeModule{{"issue", purgeFunc(func(ctx context.Context, cutoff time.Time, batch int, cursor purgePageCursor) (purgePageProgress, error) {
		calls++
		if !cutoff.Equal(now.Add(-60*24*time.Hour)) || batch != 2 {
			t.Fatalf("cutoff/batch changed: %s %d", cutoff, batch)
		}
		realNow = realNow.Add(time.Second)
		switch calls {
		case 1:
			if cursor != (purgePageCursor{}) {
				t.Fatal("initial cursor")
			}
			return purgePageProgress{Cursor: purgePageCursor{"issues", a}, Scanned: 1, Blocked: 1}, nil
		case 2:
			if cursor != (purgePageCursor{"issues", a}) {
				t.Fatalf("blocked page not retained: %+v", cursor)
			}
			return purgePageProgress{Cursor: purgePageCursor{"issues", b}, Scanned: 1, Deleted: 1}, lost
		case 3:
			if cursor != (purgePageCursor{"issues", b}) {
				t.Fatal("committed error cursor lost")
			}
			return purgePageProgress{Cursor: cursor, Done: true}, nil
		default:
			t.Fatal("extra call")
			return purgePageProgress{}, nil
		}
	})}}, clock, func() time.Time { return realNow }, 60*24*time.Hour, 2, time.Second)
	first, err := p.Execute(t.Context(), purgeCursor{})
	if err != nil || first.Complete || first.Blocked != 1 || first.Deleted != 0 || first.Cursor.After != a || calls != 1 {
		t.Fatalf("budget=%+v %v calls=%d", first, err, calls)
	}
	second, err := p.Execute(t.Context(), first.Cursor)
	if !errors.Is(err, lost) || second.Complete || second.Deleted != 1 || second.Cursor.After != b || calls != 2 {
		t.Fatalf("partial=%+v %v calls=%d", second, err, calls)
	}
	last, err := p.Execute(t.Context(), second.Cursor)
	if err != nil || !last.Complete || clock.calls != 1 || calls != 3 {
		t.Fatalf("resume=%+v %v clockCalls=%d calls=%d", last, err, clock.calls, calls)
	}
	for _, cursor := range []purgeCursor{{Version: 2}, {Version: 1, Module: "unknown", Cutoff: now}, {Version: 1, Module: "issue"}, {Table: "uninitialized"}} {
		if _, err := p.Execute(t.Context(), cursor); err == nil {
			t.Fatalf("accepted cursor %+v", cursor)
		}
	}
}

// The previous all-module regression fixtures drain only in tests. Production
// callers must retain the page cursor when their work budget ends.
func drainPurge(p *purgeDeleted, ctx context.Context) (int, error) {
	total := 0
	var cursor purgeCursor
	for attempts := 0; attempts < 10000; attempts++ {
		r, err := p.Execute(ctx, cursor)
		total += r.Deleted
		if err != nil || r.Complete {
			return total, err
		}
		cursor = r.Cursor
	}
	return total, errors.New("test purge did not converge")
}
func drainPurgeModule(p Purger, ctx context.Context, cutoff time.Time, batch int) (int, error) {
	total := 0
	var cursor purgePageCursor
	for attempts := 0; attempts < 10000; attempts++ {
		r, err := p.Page(ctx, cutoff, batch, cursor)
		total += r.Deleted
		if err != nil || r.Done {
			return total, err
		}
		cursor = r.Cursor
	}
	return total, errors.New("test module purge did not converge")
}
