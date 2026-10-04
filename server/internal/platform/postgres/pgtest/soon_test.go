package pgtest_test

import (
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Soon's context ends within 5 seconds of the call.
func TestSoonEndsWithinFiveSeconds(t *testing.T) {
	t.Parallel()
	deadline, ok := pgtest.Soon(t).Deadline()
	if left := time.Until(deadline); !ok || left <= 0 || left > 5*time.Second {
		t.Errorf("Soon() ends in %v (a deadline: %v), want within 5s", left, ok)
	}
}
