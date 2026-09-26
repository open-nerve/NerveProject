package app_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
)

// fakeExpiries is the sessions' expiry times, in memory. It deletes the
// way the query does: up to limit of those before now. failAt fails that
// call, counting from 1.
type fakeExpiries struct {
	expires []time.Time
	calls   []int // the limit of each call
	nows    []time.Time
	failAt  int
}

var errDatabaseGone = errors.New("connection reset")

func (f *fakeExpiries) DeleteExpiredSessions(_ context.Context, now time.Time, limit int) (int, error) {
	f.calls = append(f.calls, limit)
	f.nows = append(f.nows, now)
	if len(f.calls) == f.failAt {
		return 0, errDatabaseGone
	}
	deleted := 0
	f.expires = slices.DeleteFunc(f.expires, func(at time.Time) bool {
		if deleted < limit && at.Before(now) {
			deleted++
			return true
		}
		return false
	})
	return deleted, nil
}

// sessionsExpiring returns n expiry times at, then the live ones.
func sessionsExpiring(n int, at time.Time, live int) []time.Time {
	var out []time.Time
	for range n {
		out = append(out, at)
	}
	for range live {
		out = append(out, now.Add(time.Hour))
	}
	return out
}

func TestCleanupSessionsDeletesBatchAfterBatch(t *testing.T) {
	tests := []struct {
		name    string
		expired int
		calls   []int
	}{
		{"none expired", 0, []int{1000}},
		{"fewer than a batch", 3, []int{1000}},
		{"exactly a batch", 1000, []int{1000, 1000}},
		{"two batches and more", 2003, []int{1000, 1000, 1000}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeExpiries{expires: sessionsExpiring(tt.expired, now.Add(-time.Second), 2)}
			var logs bytes.Buffer

			n, err := app.NewCleanupSessions(f, clocktest.At(now), slog.New(slog.NewJSONHandler(&logs, nil))).Execute(context.Background())

			if err != nil || n != tt.expired || len(f.expires) != 2 || !slices.Equal(f.calls, tt.calls) {
				t.Errorf("Execute() = %d, %v with %d sessions left, calls %v; want %d, the 2 live left, calls %v",
					n, err, len(f.expires), f.calls, tt.expired, tt.calls)
			}
			for _, at := range f.nows {
				if !at.Equal(now) {
					t.Errorf("a batch deleted before %v, want the clock's %v", at, now)
				}
			}
			want := fmt.Sprintf(`"msg":"expired sessions deleted","sessions":%d}`, tt.expired)
			if logged := strings.Contains(logs.String(), want); logged != (tt.expired > 0) || tt.expired == 0 && logs.Len() > 0 {
				t.Errorf("logs = %s, want %s when any was deleted, else nothing", logs.String(), want)
			}
		})
	}
}

// Sessions that expire at the clock's now, or after, stay.
func TestCleanupSessionsKeepsWhatExpiresFromNowOn(t *testing.T) {
	f := &fakeExpiries{expires: []time.Time{now.Add(-time.Microsecond), now, now.Add(time.Microsecond)}}

	n, err := app.NewCleanupSessions(f, clocktest.At(now), slog.New(slog.DiscardHandler)).Execute(context.Background())

	if err != nil || n != 1 || !slices.Equal(f.expires, []time.Time{now, now.Add(time.Microsecond)}) {
		t.Errorf("Execute() = %d, %v leaving %v; want 1 deleted, those from now on left", n, err, f.expires)
	}
}

// A failed batch ends the run with the error and what the batches before
// it deleted; nothing is logged as deleted by the failed one.
func TestCleanupSessionsWhenABatchFails(t *testing.T) {
	f := &fakeExpiries{expires: sessionsExpiring(2500, now.Add(-time.Second), 0), failAt: 2}
	var logs bytes.Buffer

	n, err := app.NewCleanupSessions(f, clocktest.At(now), slog.New(slog.NewJSONHandler(&logs, nil))).Execute(context.Background())

	if !errors.Is(err, errDatabaseGone) || n != 1000 || len(f.calls) != 2 || strings.Contains(logs.String(), "expired sessions deleted") {
		t.Errorf("Execute() = %d, %v after %d calls, logs %s; want 1000 and the error after the second call, no log", n, err, len(f.calls), logs.String())
	}
}
