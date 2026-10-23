package pgtest_test

import (
	"context"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// RowWaits sees a lock another transaction holds on the row that the lock
// asked for conflicts with, and nothing else: FOR SHARE waits for the row
// held FOR NO KEY UPDATE; not for the one held FOR KEY SHARE, with which
// it does not conflict; not for a row nobody holds; and not for the first
// once its holder is done. A row that is not there fails the test, rather
// than read as a row nobody holds.
func TestRowWaitsSeesALockThatConflicts(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	pool := newPool(t, pgtest.NewDatabase(t))
	held, keyShared, free := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	if _, err := pool.Exec(ctx, "CREATE TABLE a (id uuid PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO a VALUES ($1), ($2), ($3)", held, keyShared, free); err != nil {
		t.Fatal(err)
	}
	holder, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Rollback(ctx) }()
	for _, lock := range []struct {
		id     uuid.UUID
		clause string
	}{{held, "FOR NO KEY UPDATE"}, {keyShared, "FOR KEY SHARE"}} {
		if _, err := holder.Exec(ctx, "SELECT 1 FROM a WHERE id = $1 "+lock.clause, lock.id); err != nil {
			t.Fatal(err)
		}
	}
	for _, row := range []struct {
		name string
		id   uuid.UUID
		want bool
	}{{"held", held, true}, {"key-shared", keyShared, false}, {"free", free, false}} {
		if got := pgtest.RowWaits(t, pool, "a", row.id, "FOR SHARE"); got != row.want {
			t.Errorf("RowWaits() of the %s row = %v, want %v", row.name, got, row.want)
		}
	}
	if err := holder.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if pgtest.RowWaits(t, pool, "a", held, "FOR SHARE") {
		t.Error("RowWaits() of the held row once its holder committed = true, want false")
	}
	absent := uuid.NewV7()
	failed := fatalOf(t, func(tb testing.TB) { pgtest.RowWaits(tb, pool, "a", absent, "FOR SHARE") })
	if want := "pgtest: a has no row " + absent.String(); failed != want {
		t.Errorf("RowWaits() of a row that is not there failed with %q, want %q", failed, want)
	}
}
