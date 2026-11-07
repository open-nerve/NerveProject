package app

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"
)

func TestPurgeCursorResume(t *testing.T) {
	cutoff := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	low, high := uuid.NewV7(), uuid.NewV7()
	if low.Compare(high) > 0 {
		low, high = high, low
	}
	lost := errors.New("later page failed")
	calls := 0
	steps := []PurgeStep{
		{Table: "first", Protected: true, Delete: func(_ context.Context, b PurgeBatch) (PurgePage, error) {
			calls++
			if b.Cutoff != cutoff || b.Limit != 2 {
				t.Fatalf("batch=%+v", b)
			}
			switch calls {
			case 1:
				if b.After != uuid.Nil() {
					t.Fatal("first cursor")
				}
				return PurgePage{Scanned: 1, Blocked: 1, After: low}, nil
			case 2:
				if b.After != low {
					t.Fatal("lost blocked cursor")
				}
				return PurgePage{Scanned: 1, Deleted: 1, After: high}, lost
			case 3:
				if b.After != high {
					t.Fatal("lost committed error progress")
				}
				return PurgePage{After: b.After}, nil
			default:
				t.Fatal("repeated exhausted table")
				return PurgePage{}, nil
			}
		}},
		{Table: "last", Delete: func(_ context.Context, b PurgeBatch) (PurgePage, error) {
			calls++
			if b.After != uuid.Nil() {
				t.Fatal("next table cursor")
			}
			return PurgePage{}, nil
		}},
	}
	p := NewPurger(steps)
	steps[0].Table = "changed"
	tables := p.Tables()
	tables[0] = "changed again"
	protected := p.ProtectedTables()
	protected[0] = "also changed"
	if !slices.Equal(p.Tables(), []string{"first", "last"}) || !slices.Equal(p.ProtectedTables(), []string{"first"}) {
		t.Fatal("directory aliases its caller")
	}
	first, err := p.Page(t.Context(), cutoff, 2, PurgeCursor{})
	if err != nil || first.Done || first.Scanned != 1 || first.Deleted != 0 || first.Blocked != 1 || first.Cursor != (PurgeCursor{"first", low}) || calls != 1 {
		t.Fatalf("first=%+v %v calls=%d", first, err, calls)
	}
	second, err := p.Page(t.Context(), cutoff, 2, first.Cursor)
	if !errors.Is(err, lost) || second.Done || second.Deleted != 1 || second.Scanned != 1 || second.Cursor != (PurgeCursor{"first", high}) || calls != 2 {
		t.Fatalf("error progress=%+v %v calls=%d", second, err, calls)
	}
	third, err := p.Page(t.Context(), cutoff, 2, second.Cursor)
	if err != nil || third.Done || third.Cursor != (PurgeCursor{Table: "last"}) || calls != 3 {
		t.Fatalf("table advance=%+v %v", third, err)
	}
	last, err := p.Page(t.Context(), cutoff, 2, third.Cursor)
	if err != nil || !last.Done || calls != 4 {
		t.Fatalf("end=%+v %v calls=%d", last, err, calls)
	}
}

func TestPurgeCursorRejectsInvalidInput(t *testing.T) {
	p := NewPurger([]PurgeStep{{Table: "one", Delete: func(context.Context, PurgeBatch) (PurgePage, error) { t.Fatal("store called"); return PurgePage{}, nil }}})
	for _, cursor := range []PurgeCursor{{Table: "unknown"}, {After: uuid.New()}} {
		if _, err := p.Page(t.Context(), time.Now(), 1, cursor); err == nil {
			t.Fatalf("accepted %+v", cursor)
		}
	}
	for _, limit := range []int{-1, 0, 2147483648} {
		if _, err := p.Page(t.Context(), time.Now(), limit, PurgeCursor{}); err == nil {
			t.Fatalf("accepted limit %d", limit)
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := p.Page(ctx, time.Now(), 1, PurgeCursor{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel=%v", err)
	}
}
