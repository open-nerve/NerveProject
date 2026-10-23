package pgtest

import (
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// RowWaits reports whether a statement that takes lock, such as "FOR
// SHARE", on the row id of table, in a transaction of its own, waits for a
// lock another transaction holds on the row: NOWAIT answers
// lock_not_available (postgres.LockNotAvailable) at once instead of
// waiting. Its wait for a connection of pool ends soon (Soon). A row that
// is not there fails the test, since no lock could ever be seen on it, and
// so does any other failure.
func RowWaits(t testing.TB, pool *pgxpool.Pool, table string, id uuid.UUID, lock string) bool {
	t.Helper()
	tag, err := pool.Exec(Soon(t), "SELECT 1 FROM "+table+" WHERE id = $1 "+lock+" NOWAIT", id)
	if postgres.LockNotAvailable(err) {
		return true
	}
	if err != nil {
		t.Fatal(err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("pgtest: %s has no row %s", table, id)
	}
	return false
}
