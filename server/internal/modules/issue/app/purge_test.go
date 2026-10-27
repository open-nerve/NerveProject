package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"
)

// Counting an errored batch or trusting RETURNING order instead of its maximum
// would lose committed progress or revisit the wrong cursor.
func TestPurgerAdvancesOnlyCommittedBatches(t *testing.T) {
	cutoff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	low, high := uuid.NewV7(), uuid.NewV7()
	if low.Compare(high) > 0 {
		low, high = high, low
	}
	boom := errors.New("failed batch")
	t.Run("committed count and max cursor", func(t *testing.T) {
		var calls []string
		purger := NewPurger([]PurgeStep{
			{Table: "first", Delete: func(_ context.Context, b PurgeBatch) ([]uuid.UUID, error) {
				calls = append(calls, fmt.Sprintf("first:%s", b.After))
				if b.Cutoff != cutoff || b.Limit != 2 {
					t.Fatalf("batch=%+v", b)
				}
				switch len(calls) {
				case 1:
					return []uuid.UUID{low, high}, nil
				case 2:
					if b.After != high {
						t.Fatalf("After=%s, want max %s", b.After, high)
					}
					return nil, nil
				default:
					t.Fatal("extra first batch")
					return nil, nil
				}
			}},
			{Table: "fails", Delete: func(_ context.Context, b PurgeBatch) ([]uuid.UUID, error) {
				calls = append(calls, "fails")
				if b.After != (uuid.UUID{}) {
					t.Fatalf("next table After=%s", b.After)
				}
				return []uuid.UUID{uuid.NewV7()}, fmt.Errorf("delete: %w", boom)
			}},
			{Table: "later", Delete: func(context.Context, PurgeBatch) ([]uuid.UUID, error) {
				t.Fatal("later table called after error")
				return nil, nil
			}},
		})
		count, err := purger.Purge(context.Background(), cutoff, 2)
		if count != 2 || !errors.Is(err, boom) {
			t.Fatalf("Purge=%d,%v; want 2 and original error chain", count, err)
		}
		if want := []string{"first:" + (uuid.UUID{}).String(), "first:" + high.String(), "fails"}; !slices.Equal(calls, want) {
			t.Fatalf("steps=%v, want %v", calls, want)
		}
	})
	t.Run("unordered result uses maximum", func(t *testing.T) {
		calls := 0
		purger := NewPurger([]PurgeStep{{Table: "one", Delete: func(_ context.Context, b PurgeBatch) ([]uuid.UUID, error) {
			calls++
			if calls == 1 {
				return []uuid.UUID{high, low}, nil
			}
			if calls != 2 || b.After != high {
				t.Fatalf("unordered cursor=%s calls=%d; want max %s on the second batch", b.After, calls, high)
			}
			return nil, nil
		}}})
		if count, err := purger.Purge(context.Background(), cutoff, 2); count != 2 || err != nil || calls != 2 {
			t.Fatalf("unordered purge=%d,%v calls=%d", count, err, calls)
		}
	})
	t.Run("empty advances to next table and copies directory", func(t *testing.T) {
		var calls []string
		steps := []PurgeStep{{Table: "one", Delete: func(context.Context, PurgeBatch) ([]uuid.UUID, error) { calls = append(calls, "one"); return nil, nil }}, {Table: "two", Delete: func(context.Context, PurgeBatch) ([]uuid.UUID, error) { calls = append(calls, "two"); return nil, nil }}}
		purger := NewPurger(steps)
		steps[0].Table = "changed"
		tables := purger.Tables()
		if !slices.Equal(tables, []string{"one", "two"}) {
			t.Fatalf("directory=%v, want one,two", tables)
		}
		tables[0] = "changed again"
		if !slices.Equal(purger.Tables(), []string{"one", "two"}) {
			t.Fatalf("directory aliases input/output: %v", purger.Tables())
		}
		count, err := purger.Purge(context.Background(), cutoff, 1)
		if count != 0 || err != nil || !slices.Equal(calls, []string{"one", "two"}) {
			t.Fatalf("empty purge=%d,%v steps=%v", count, err, calls)
		}
	})
	t.Run("cancellation does not start another batch", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		calls := 0
		purger := NewPurger([]PurgeStep{{Table: "one", Delete: func(context.Context, PurgeBatch) ([]uuid.UUID, error) {
			calls++
			cancel()
			return []uuid.UUID{low}, nil
		}}, {Table: "two", Delete: func(context.Context, PurgeBatch) ([]uuid.UUID, error) {
			t.Fatal("new table after cancellation")
			return nil, nil
		}}})
		count, err := purger.Purge(ctx, cutoff, 1)
		if count != 1 || !errors.Is(err, context.Canceled) || calls != 1 {
			t.Fatalf("cancelled purge=%d,%v calls=%d", count, err, calls)
		}
		count, err = purger.Purge(ctx, cutoff, 1)
		if count != 0 || !errors.Is(err, context.Canceled) || calls != 1 {
			t.Fatalf("pre-cancelled purge=%d,%v calls=%d", count, err, calls)
		}
	})
	t.Run("invalid batch never calls store", func(t *testing.T) {
		purger := NewPurger([]PurgeStep{{Table: "one", Delete: func(context.Context, PurgeBatch) ([]uuid.UUID, error) {
			t.Fatal("store called with invalid batch")
			return nil, nil
		}}})
		for _, limit := range []int{-1, 0, 2147483648} {
			if count, err := purger.Purge(context.Background(), cutoff, limit); count != 0 || err == nil {
				t.Fatalf("limit %d: %d,%v", limit, count, err)
			}
		}
	})
}
