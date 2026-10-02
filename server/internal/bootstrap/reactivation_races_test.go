package bootstrap

import (
	"bytes"
	"context"
	"maps"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A reactivation through the command's composition against what another
// transaction changes or holds meanwhile (M3 design 3.6's lock table,
// 3.11): what it finds once it has a lock, each lock it takes, at its
// strength, and the connection it runs on. endedMembers prepares each
// database: bob's membership of acme, an admin's, and his memberships of
// Web and Ops ended.

// commandRun is a command's line and error.
type commandRun struct {
	out string
	err error
}

// bobReactivated is the line of bob's reactivation in acme.
const bobReactivated = "reactivated bob@corp.com in acme as admin; project memberships still ended: 2, each restored when the member " +
	"joins or is added to its project\n"

// reactivatingBob runs `nerve workspaces reactivate-member --slug acme
// --email bob@corp.com` on the database at url, on a pool of maxConns
// connections, in the background until ctx ends, and hands over its line
// and error.
func reactivatingBob(ctx context.Context, t *testing.T, url string, maxConns int32) <-chan commandRun {
	t.Helper()
	cfg := testConfig(t, url, false)
	cfg.Database.MaxConns = maxConns
	done := make(chan commandRun, 1)
	go func() {
		var out, logs bytes.Buffer
		err := Workspaces(ctx, cfg, &logs, &out, ReactivateMember("acme", "bob@corp.com"))
		done <- commandRun{out.String(), err}
	}()
	return done
}

// endedIDs are the ids of acme, of bob's account and of his membership of
// acme.
type endedIDs struct {
	acme, bob, bobs uuid.UUID
}

func idsOf(t *testing.T, pool *pgxpool.Pool) endedIDs {
	t.Helper()
	var ids endedIDs
	if err := pool.QueryRow(context.Background(), `SELECT w.id, u.id, m.id FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
		JOIN users u ON u.id = m.member_id WHERE w.slug = 'acme' AND u.email = 'bob@corp.com'`).Scan(&ids.acme, &ids.bob, &ids.bobs); err != nil {
		t.Fatal(err)
	}
	return ids
}

// A reactivation that waits for a lock reads, once it has it, what another
// transaction committed meanwhile, and answers for that (M3 design 3.6,
// 3.11): acme deleted while it waits for acme's row is "No workspace has
// this slug.", and nothing changes; bob's membership made active again
// meanwhile, as accepting an invitation restores it, is reported active,
// and nothing changes; bob's account deactivated while it waits for his
// account's row is reactivated all the same, his membership with it, and
// the line says what is next. The row the other changed is as it left it,
// bob's membership as the case says, every other row as it was.
func TestAReactivationFindsWhatChangedMeanwhile(t *testing.T) {
	for _, tt := range []struct {
		name, holds, waitsOn, change string
		table                        string                       // the table of the row the other changes
		row                          func(ids endedIDs) uuid.UUID // that row
		out, err                     string
		active                       bool // bob's membership of acme after it
	}{
		{"acme deleted", "SELECT 1 FROM workspaces WHERE id = $1 AND $2::uuid IS NOT NULL FOR NO KEY UPDATE", "workspaces",
			"UPDATE workspaces SET deleted_at = now(), updated_at = now() WHERE id = $1 AND $2::uuid IS NOT NULL", "workspaces",
			func(ids endedIDs) uuid.UUID { return ids.acme }, "", "No workspace has this slug.", false},
		{"his membership active again", "SELECT 1 FROM workspaces WHERE id = $1 AND $2::uuid IS NOT NULL FOR NO KEY UPDATE", "workspaces",
			"UPDATE workspace_members SET is_active = true, updated_at = now() WHERE workspace_id = $1 AND member_id = $2", "workspace_members",
			func(ids endedIDs) uuid.UUID { return ids.bobs }, "bob@corp.com is an active member of acme already; nothing changed\n", "", true},
		{"his account deactivated", "SELECT 1 FROM users WHERE id = $2 AND $1::uuid IS NOT NULL FOR NO KEY UPDATE", "users",
			"UPDATE users SET is_active = false, updated_at = now() WHERE id = $2 AND $1::uuid IS NOT NULL", "users",
			func(ids endedIDs) uuid.UUID { return ids.bob },
			strings.TrimSuffix(bobReactivated, "\n") + "; the account is deactivated: run nerve users activate --email bob@corp.com next\n", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			url := pgtest.NewDatabase(t)
			pool := endedMembers(t, url)
			ids := idsOf(t, pool)
			table, id := tt.table, tt.row(ids)
			others := rowsBut(t, pool, []uuid.UUID{id, ids.bobs})
			other := holding(t, pool, tt.holds, ids.acme, ids.bob)
			if tag, err := other.Exec(context.Background(), tt.change, ids.acme, ids.bob); err != nil || tag.RowsAffected() != 1 {
				t.Fatalf("%s: %v, %v; want one row changed", tt.change, tag, err)
			}
			changed := rowJSON(t, other, table, id)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			done := reactivatingBob(ctx, t, url, 4)
			pgtest.WaitForLockWaitOn(t, pool, tt.waitsOn, 5*time.Second)
			if err := other.Commit(context.Background()); err != nil {
				t.Fatal(err)
			}

			run := receiveWithin(t, done, 10*time.Second, "reactivate-member's end")
			if run.out != tt.out || (tt.err == "") != (run.err == nil) || (run.err != nil && !strings.Contains(run.err.Error(), tt.err)) {
				t.Errorf("reactivate-member = %q, %v; want %q, %q", run.out, run.err, tt.out, tt.err)
			}
			if after := rowJSON(t, pool, table, id); !maps.Equal(after, changed) {
				t.Errorf("%s %s after it:\n%v\nwant it as the other transaction left it:\n%v", table, id, after, changed)
			}
			if active := rowJSON(t, pool, "workspace_members", ids.bobs)["is_active"]; active != tt.active {
				t.Errorf("bob's membership of acme after it: active %v, want %v", active, tt.active)
			}
			if after := rowsBut(t, pool, []uuid.UUID{id, ids.bobs}); !maps.Equal(after, others) {
				t.Errorf("every other row after it:\n%v\nwant them as they were:\n%v", after, others)
			}
		})
	}
}

// Each lock of a reactivation is taken in the lock table's order, at its
// strength, through the command's composition (M3 design 3.6's lock table,
// convention 6): bob's account row FOR SHARE, then acme's row FOR NO KEY
// UPDATE, then his membership. Other transactions hold acme's row FOR NO
// KEY UPDATE and his membership FOR SHARE, and let go one at a time;
// lockOn reads each row's strongest lock then. The command waits for
// acme's row holding his account FOR SHARE, no stronger, no weaker, and
// nothing else; then for his membership, holding his account and acme's
// row FOR NO KEY UPDATE. Then it reactivates him, at a moment no earlier
// than acme's release: it read the clock under acme's lock (3.3). The
// holder's FOR SHARE masks a FOR SHARE taken on his membership before
// acme's row; such a lock shows up only as the wait for his membership
// never being seen, PostgreSQL taking no tuple lock for the upgrade.
func TestEachLockOfAReactivationIsItsStrength(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := endedMembers(t, url)
	ids := idsOf(t, pool)
	locks := func() string {
		t.Helper()
		return "his account " + lockOn(t, pool, "users WHERE id = $1", ids.bob) + ", acme " + lockOn(t, pool, "workspaces WHERE id = $1", ids.acme) +
			", his membership " + lockOn(t, pool, "workspace_members WHERE id = $1", ids.bobs)
	}
	holdsMembership := holding(t, pool, "SELECT 1 FROM workspace_members WHERE id = $1 FOR SHARE", ids.bobs)
	holdsAcme := holding(t, pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", ids.acme)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	done := reactivatingBob(ctx, t, url, 4)
	pgtest.WaitForLockWaitOn(t, pool, "workspaces", 5*time.Second)
	// acme's lock here is the holder's, and so is his membership's FOR SHARE.
	if got, want := locks(), "his account FOR SHARE, acme FOR NO KEY UPDATE, his membership FOR SHARE"; got != want {
		t.Errorf("the reactivation waiting for acme's row: %s; want %s", got, want)
	}
	released := time.Now()
	if err := holdsAcme.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	pgtest.WaitForLockWaitOn(t, pool, "workspace_members", 5*time.Second)
	if got, want := locks(), "his account FOR SHARE, acme FOR NO KEY UPDATE, his membership FOR SHARE"; got != want {
		t.Errorf("the reactivation waiting for his membership: %s; want %s", got, want)
	}
	if err := holdsMembership.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}

	if run := receiveWithin(t, done, 10*time.Second, "reactivate-member's end"); run.out != bobReactivated || run.err != nil {
		t.Fatalf("reactivate-member = %q, %v; want %q", run.out, run.err, bobReactivated)
	}
	moment, _ := rowJSON(t, pool, "workspace_members", ids.bobs)["updated_at"].(string)
	if at, err := time.Parse(time.RFC3339Nano, moment); err != nil || at.Before(released.Truncate(time.Microsecond)) {
		t.Errorf("the reactivation's moment %q (%v); want one no earlier than acme's release, %v", moment, err, released)
	}
}

// The command has no deadline of its own: while acme's row is held it
// waits, and an interruption, its context's end as SIGINT or SIGTERM ends
// it (cmd/nerve), rolls it back: it fails, and no row changes. Run again
// once the row is free, it reactivates bob (README, M3 design 17.4).
func TestAnInterruptedReactivationChangesNothing(t *testing.T) {
	url := pgtest.NewDatabase(t)
	pool := endedMembers(t, url)
	before := tableRows(t, pool, riversOwn)
	holdsAcme := holding(t, pool, "SELECT 1 FROM workspaces WHERE slug = 'acme' FOR NO KEY UPDATE")
	ctx, interrupt := context.WithCancel(context.Background())
	defer interrupt()
	done := reactivatingBob(ctx, t, url, 4)
	pgtest.WaitForLockWaitOn(t, pool, "workspaces", 5*time.Second)
	interrupt()

	if run := receiveWithin(t, done, 10*time.Second, "reactivate-member's end"); run.err == nil || run.out != "" {
		t.Errorf("reactivate-member interrupted = %q, %v; want no line and an error", run.out, run.err)
	}
	if err := holdsAcme.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	if after := tableRows(t, pool, riversOwn); !maps.Equal(after, before) {
		t.Errorf("the tables after the interrupted reactivation changed:\n%v\nwant them as they were:\n%v", after, before)
	}
	again, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if run := receiveWithin(t, reactivatingBob(again, t, url, 4), 10*time.Second, "reactivate-member's end"); run.out != bobReactivated ||
		run.err != nil {
		t.Errorf("reactivate-member again = %q, %v; want %q", run.out, run.err, bobReactivated)
	}
}

// The reactivation runs every statement on its transaction's connection
// (M3 design 3.6 convention 2): the account's lock, the workspace's, the
// membership's read, the reactivation and the count of his ended project
// memberships through ProjectMembershipCounts. The command's composition
// runs on a pool of one connection: a statement sent through the pool
// rather than the transaction would wait for a second connection until the
// command's context ends, after 5 seconds, and the command fail.
func TestTheReactivationRunsOnItsTransactionsConnection(t *testing.T) {
	url := pgtest.NewDatabase(t)
	endedMembers(t, url)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if run := receiveWithin(t, reactivatingBob(ctx, t, url, 1), 10*time.Second, "reactivate-member's end"); run.out != bobReactivated || run.err != nil {
		t.Errorf("reactivate-member on a pool of one connection = %q, %v; want %q", run.out, run.err, bobReactivated)
	}
}
