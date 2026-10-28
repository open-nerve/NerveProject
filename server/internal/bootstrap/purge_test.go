package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
)

type purgeClockProbe struct {
	now   time.Time
	calls int
}

func (c *purgeClockProbe) Now() time.Time {
	c.calls++
	return c.now.Add(time.Duration(c.calls-1) * time.Hour)
}

type purgeFunc func(context.Context, time.Time, int) (int, error)

func (f purgeFunc) Purge(ctx context.Context, cutoff time.Time, batch int) (int, error) {
	return f(ctx, cutoff, batch)
}

// Reordering modules, reading the clock twice, dropping committed batches on
// error, or continuing after an error must all change this observable result.
func TestPurgeDeletedUsesOneCutoffAndModuleOrder(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprintf("failure=%t", fail), func(t *testing.T) {
			now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
			clock := &purgeClockProbe{now: now}
			lost := errors.New("later batch failed")
			var order []string
			var modules []Purger
			for i, name := range []string{"issue", "project", "workspace", "identity"} {
				modules = append(modules, purgeFunc(func(ctx context.Context, cutoff time.Time, batch int) (int, error) {
					order = append(order, name)
					if ctx != t.Context() || cutoff != time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC) || batch != 7 {
						t.Errorf("%s got ctx=%v cutoff=%s batch=%d", name, ctx, cutoff, batch)
					}
					if fail && i == 1 {
						return 3, fmt.Errorf("project: %w", lost)
					}
					return []int{2, 5, 7, 11}[i], nil
				}))
			}
			total, err := newPurgeDeleted(modules, clock, 60*24*time.Hour, 7).Execute(t.Context())
			wantOrder, wantTotal := []string{"issue", "project", "workspace", "identity"}, 25
			if fail {
				wantOrder, wantTotal = []string{"issue", "project"}, 5
			}
			if !slices.Equal(order, wantOrder) || total != wantTotal || errors.Is(err, lost) != fail || clock.calls != 1 {
				t.Fatalf("order=%v total=%d err=%v clockCalls=%d; want %v/%d/error=%t/1", order, total, err, clock.calls, wantOrder, wantTotal, fail)
			}
		})
	}
}
