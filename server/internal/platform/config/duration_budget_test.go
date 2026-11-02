package config

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/configs"
)

// Both durations can be individually valid while their sum wraps negative.
// Exercise real environment loading, including the exact budget boundary.
func TestDurationBudgetsRejectOverflow(t *testing.T) {
	for _, budget := range []struct {
		name, firstKey, secondKey, errorKey string
		limit                               time.Duration
	}{
		{"issue write", "NERVE_ISSUE__WRITE_GATE__MAX_WAIT", "NERVE_ISSUE__WRITE_LOCK_TIMEOUT", "issue.write_lock_timeout:", 15 * time.Second},
		{"refresh", "NERVE_DATABASE__COMMIT_TIMEOUT", "NERVE_AUTH__REFRESH_DEADLINE", "auth.refresh_deadline:", 8 * time.Second},
	} {
		for _, tt := range []struct {
			name          string
			first, second time.Duration
			valid         bool
		}{
			{"one nanosecond below", time.Second, budget.limit - time.Second - time.Nanosecond, true},
			{"exact budget", time.Second, budget.limit - time.Second, false},
			{"above budget", time.Second, budget.limit - time.Second + time.Nanosecond, false},
			{"first duration alone exceeds budget", budget.limit + time.Second, time.Nanosecond, false},
			{"second duration alone exceeds budget", time.Nanosecond, budget.limit + time.Second, false},
			{"first overflows sum", time.Duration(math.MaxInt64), time.Second, false},
			{"second overflows sum", time.Second, time.Duration(math.MaxInt64), false},
			{"both overflow sum", time.Duration(math.MaxInt64), time.Duration(math.MaxInt64), false},
		} {
			t.Run(budget.name+"/"+tt.name, func(t *testing.T) {
				_, err := Load(Sources{Embedded: configs.FS(), Environ: []string{
					"NERVE_ENV=dev", "NERVE_DATABASE__URL=postgres://test",
					"NERVE_SERVER__REQUEST_TIMEOUT=15s",
					budget.firstKey + "=" + tt.first.String(),
					budget.secondKey + "=" + tt.second.String(),
				}})
				if tt.valid {
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				if err == nil || !strings.Contains(err.Error(), budget.errorKey) {
					t.Fatalf("Load(%s, %s) = %v; want %s rejection", tt.first, tt.second, err, budget.errorKey)
				}
			})
		}
	}
}
