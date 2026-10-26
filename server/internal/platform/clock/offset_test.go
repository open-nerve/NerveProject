package clock_test

import (
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
)

type rawClock struct{ at time.Time }

func (c rawClock) Now() time.Time { return c.at }

// Removing the offset, UTC conversion, or final microsecond truncation changes
// these hand-derived instants. The raw base deliberately preserves nanoseconds.
func TestOffset(t *testing.T) {
	base := rawClock{time.Date(2026, 1, 1, 0, 30, 0, 123456789, time.FixedZone("east", 8*60*60))}
	for _, tt := range []struct {
		name   string
		offset time.Duration
		want   string
	}{
		{"zero", 0, "2025-12-31T16:30:00.123456Z"},
		{"positive crosses year", 8*time.Hour + 321*time.Nanosecond, "2026-01-01T00:30:00.123457Z"},
		{"negative crosses date", -17*time.Hour - 790*time.Nanosecond, "2025-12-30T23:30:00.123455Z"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := (clock.Offset{Base: base, Offset: tt.offset}).Now()
			if got.Format(time.RFC3339Nano) != tt.want || got.Location() != time.UTC {
				t.Fatalf("Now() = %s (%v), want %s UTC", got.Format(time.RFC3339Nano), got.Location(), tt.want)
			}
		})
	}
}
