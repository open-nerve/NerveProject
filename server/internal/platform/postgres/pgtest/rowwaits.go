package pgtest

import (
	"errors"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RowWaits reports whether a statement that takes lock, such as "FOR
// SHARE", on the row id of table, in a transaction of its own, waits for a
// lock another transaction holds on the row: NOWAIT answers
// lock_not_available (55P03) at once instead of waiting. Its wait for a
// connection of pool ends soon (Soon); any other failure fails the test.
func RowWaits(t testing.TB, pool *pgxpool.Pool, table string, id uuid.UUID, lock string) bool {
	t.Helper()
	_, err := pool.Exec(Soon(t), "SELECT 1 FROM "+table+" WHERE id = $1 "+lock+" NOWAIT", id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "55P03" {
		return true
	}
	if err != nil {
		t.Fatal(err)
	}
	return false
}
