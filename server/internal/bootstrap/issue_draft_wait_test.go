package bootstrap

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func draftNoLock(t *testing.T, w issueWorld, table string, id uuid.UUID) {
	t.Helper()
	if pgtest.RowWaits(t, w.pool, table, id, "FOR UPDATE") {
		t.Fatalf("unexpected %s row lock", table)
	}
}
func draftExec(t *testing.T, db postgres.Querier, sql string, args ...any) {
	t.Helper()
	if _, err := db.Exec(pgtest.Soon(t), sql, args...); err != nil {
		t.Fatal(err)
	}
}
func draftCommit(t *testing.T, tx pgx.Tx) {
	t.Helper()
	if err := tx.Commit(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
}

func draftWrite(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange, method, path, body string) func() error {
	t.Helper()
	return w.sending(e, newIssueRoute(t, w.pool, tx), "alice", method, path, body)
}
func draftSerialize(t *testing.T, w issueWorld, first, second racingWrite, a, b *exchange, table string, after func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	g := newGate()
	ta, tb := requestTx(w.pool), requestTx(w.pool)
	fa := first(t, w, heldTx{ta, g}, a)
	fb := second(t, w, tb, b)
	_, ea, eb := serializeWaiting(t, ctx, g, fa, fb, func() { requestBlocked(t, w.pool, tb, requestPID(t, ta), table) }, after)
	if ea != nil || eb != nil {
		t.Fatal(ea, eb)
	}
}
func draftPublication(id uuid.UUID) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return draftWrite(t, w, tx, e, "POST", publishPath(id), "")
	}
}
func draftMove(id, target uuid.UUID) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return draftWrite(t, w, tx, e, "PATCH", draftPath(id), `{"project_id":"`+target.String()+`"}`)
	}
}
