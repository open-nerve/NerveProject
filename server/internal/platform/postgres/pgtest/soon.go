package pgtest

import (
	"context"
	"testing"
	"time"
)

// Soon is a context that ends 5 s from now, or with the test: for a
// test's own statement, acquire of a pool's connection or Begin, run while
// transactions the test holds are open. Were the holders to take every
// connection of the pool, or the row it reads, its wait would have no end
// otherwise; past the deadline it fails, and so does the test.
func Soon(t testing.TB) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}
