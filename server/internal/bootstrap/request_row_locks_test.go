package bootstrap

import (
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// requestRowLock counts only this request's direct transactionid conflicts.
// Tuple queues, other owners and blocking-chain reachability are not row locks.
func requestRowLock(t *testing.T, observer, probes *pgxpool.Pool, request *requestProbe, from string, args ...any) string {
	t.Helper()
	return requestOwnedRowLock(t, observer, probes, request, nil, false, from, args...)
}

// requestFixtureRowLock preserves a matrix's combined strongest state while
// its known row fixture still exists. Callers pass nil after releasing it.
func requestFixtureRowLock(t *testing.T, observer, probes *pgxpool.Pool, request *requestProbe, fixture pgx.Tx, from string, args ...any) string {
	t.Helper()
	return requestOwnedRowLock(t, observer, probes, request, fixture, false, from, args...)
}

// requestGateRowLock is only for callers that already observed their explicit
// gate held. That request is idle in its transaction, not waiting on a DB lock.
func requestGateRowLock(t *testing.T, observer, probes *pgxpool.Pool, request *requestProbe, from string, args ...any) string {
	t.Helper()
	return requestOwnedRowLock(t, observer, probes, request, nil, true, from, args...)
}

func requestOwnedRowLock(t *testing.T, observer, probes *pgxpool.Pool, request *requestProbe, fixture pgx.Tx, atGate bool, from string, args ...any) string {
	t.Helper()
	before, waiting := readRequestLock(t, pgtest.Soon(t), observer, request)
	if before.pid == 0 || (!atGate && !waiting) {
		t.Fatal("request row observation requires its active wait or held gate")
	}
	identity := deactivationIdentityOf(t, observer, before.pid)
	state := func() string {
		var state string
		if err := observer.QueryRow(pgtest.Soon(t), "SELECT state FROM pg_stat_activity WHERE pid=$1", before.pid).Scan(&state); err != nil {
			t.Fatal(err)
		}
		return state
	}
	beforeState := state()
	if atGate && beforeState != "idle in transaction" {
		t.Fatalf("held gate state=%s want idle in transaction", beforeState)
	}
	owners := map[string]string{}
	// A request stopped before its first row lock may not have an XID yet.
	// It cannot own a row lock, and empty XID never names an owner.
	if identity.xid != "" {
		owners[identity.xid] = "request"
	}
	var fixtureIdentity deactivationIdentity
	if fixture != nil {
		fixtureIdentity = deactivationIdentityOf(t, observer, fixture.Conn().PgConn().PID())
		if fixtureIdentity.xid == "" {
			t.Fatal("held row fixture has no XID")
		}
		owners[fixtureIdentity.xid] = "fixture"
	}
	mode := deactivationRowLock(t, observer, probes, owners, from, args...)
	afterIdentity := deactivationIdentityOf(t, observer, before.pid)
	after, afterWaiting := readRequestLock(t, pgtest.Soon(t), observer, request)
	if waiting != afterWaiting || !sameRequestLock(before, after) || identity != afterIdentity || state() != beforeState {
		t.Fatalf("request changed during row observation: before=%+v/%+v after=%+v/%+v", before, identity, after, afterIdentity)
	}
	if fixture != nil && fixtureIdentity != deactivationIdentityOf(t, observer, fixtureIdentity.pid) {
		t.Fatal("row fixture identity changed during observation")
	}
	return mode
}
