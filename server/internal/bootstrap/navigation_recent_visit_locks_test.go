package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	navigationpg "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres"
	navigationdomain "github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A recording of one's recent visit, recordRecentVisit, takes its locks in
// the one order of M7 P3d spec 2.8 (R9), as bootstrap wires it: its target's
// workspace's row FOR SHARE first, in its transaction; then its target's
// project's row FOR SHARE, the project itself for a project's visit, and no
// lock on a work item's row; then, once it has read and decided again
// under them, the transaction-level advisory lock of his visits in the
// workspace, its last lock; and its trim, past it, waits for no row. A
// recording is a write of the workspace's, no write on a project
// (writesOnAProject): its matrix rows are of the workspace level's columns
// alone, and this test stands for the project's lock table. In acme, whose
// admin alice made Web (issueWorld), alice records her visit of a work item
// of Web and of Web, in each phase; once the start-time jobs have run
// (waitForStartupRuns), and with no fan-out pending (waitForFanouts),
// another transaction holds a lock and each recording is sent:
//   - The workspace first: the other transaction holds acme's row FOR NO
//     KEY UPDATE, as a change of a role in it or its deletion does. The
//     recording waits for acme's row in the statement that shares it
//     (ShareDirectoryWorkspaceByID), and meanwhile holds no lock on Web's
//     row or on the work item's.
//   - The project after its workspace: the other transaction holds Web's
//     row FOR NO KEY UPDATE, as a work item's deletion, a member's removal,
//     an archiving or the project's deletion does. The recording waits for
//     Web's row (ShareNavigationProject), and meanwhile holds acme's row FOR
//     SHARE and no lock on the work item's.
//   - The advisory lock after its parents: the other transaction holds the
//     advisory lock of alice's visits in acme, taken through the navigation
//     store's LockRecentVisits (holdingRecentVisits), as another recording
//     of hers there does. The recording waits for it in LockRecentVisits,
//     behind that transaction: a probe of the waiter by the holder's
//     backend and the statement (pgtest.WaitForStatementLockWaitBehind)
//     finds it, and the request's own probe (readRequestLock) that it is
//     this request's wait, for an advisory lock. Meanwhile it holds acme's
//     and Web's rows FOR SHARE and no lock on the work item's. A recording
//     that took no advisory lock would not wait here at all.
//   - The trim passes over a held row: alice has past visits in acme, each
//     visited before every visit of hers the phases recorded, written
//     directly (seedPastVisits) until two are past the cap, and another
//     transaction holds the first of those two in the trim's order FOR
//     UPDATE, as a work item's deletion holds it while it soft-deletes it
//     (FOR NO KEY UPDATE). The recording, a revisit, answers 204
//     without waiting for it (answeredUnlessWaiting): its trim deletes the
//     other one past the cap for good and leaves the held one, which keeps
//     her at the cap and one more until her next recording (spec 2.5).
//
// Each wait of the first three phases is the request's own, its PID's, and
// caused by the other transaction, in the statement of the lock the phase
// is about; each row it holds or does not is its own transaction's lock
// (requestRowLock); meanwhile it holds no advisory lock (requestAdvisoryLocks)
// and has written no table (requestWrittenTables). Once the other
// transaction ends, the recording answers 204, as it would alone. Each
// recording in each phase is a subtest, which names it when it fails; the
// first that fails ends the test, as a recording left waiting could hold
// what the next one's probes look at.
func TestRecordingARecentVisitTakesItsLocks(t *testing.T) {
	dbURL := pgtest.NewDatabase(t)
	w := newIssueWorldOnRequests(t, dbURL, openPool(t, dbURL), func(*config.Config) {}, &requestQueries{})
	item := w.newWorkItemIn(t, w.web, `{"name":"Visited"}`)
	alice := w.ids["alice"]
	acme, web, itemRow := lockedRow{"workspaces WHERE id = $1", w.acme}, lockedRow{"projects WHERE id = $1", w.web},
		lockedRow{"issues WHERE id = $1", item}
	rows := func(rs ...lockedRow) []lockedRow { return rs }
	targets := []struct {
		kind string
		id   uuid.UUID
	}{{"issue", item}, {"project", w.web}}
	holdingRow := func(sql string, id uuid.UUID) func(t *testing.T) (*pgconn.PgConn, func()) {
		return func(t *testing.T) (*pgconn.PgConn, func()) {
			other := holding(t, w.pool, sql, id)
			return other.Conn().PgConn(), func() { release(t, other) }
		}
	}
	rowWait := func(table string) func(t *testing.T, probe *requestProbe, holder *pgconn.PgConn) uint32 {
		return func(t *testing.T, probe *requestProbe, holder *pgconn.PgConn) uint32 {
			return requestBlocked(t, w.pool, probe, holder.PID(), table)
		}
	}
	advisoryWait := func(t *testing.T, probe *requestProbe, holder *pgconn.PgConn) uint32 {
		pgtest.WaitForStatementLockWaitBehind(t, w.pool, holder, 5*time.Second, "LockRecentVisits")
		sample, waiting := readRequestLock(t, pgtest.Soon(t), w.pool, probe)
		if !waiting || sample.locktype != "advisory" || !slices.Contains(sample.blockers, int32(holder.PID())) {
			t.Fatalf("the recording waits %v for %q, behind %v; want its wait for an advisory lock, behind the holder %d", waiting,
				sample.locktype, sample.blockers, holder.PID())
		}
		return sample.pid
	}
	phases := []struct {
		name, statement string
		hold            func(t *testing.T) (*pgconn.PgConn, func())
		waits           func(t *testing.T, probe *requestProbe, holder *pgconn.PgConn) uint32
		shared, free    []lockedRow
	}{
		{"its workspace first", "ShareDirectoryWorkspaceByID", holdingRow("SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", w.acme),
			rowWait("workspaces"), nil, rows(web, itemRow)},
		{"its project after its workspace", "ShareNavigationProject", holdingRow("SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", w.web),
			rowWait("projects"), rows(acme), rows(itemRow)},
		{"the advisory lock after its parents", "LockRecentVisits", func(t *testing.T) (*pgconn.PgConn, func()) {
			return holdingRecentVisits(t, w.pool, alice, w.acme)
		}, advisoryWait, rows(acme, web), rows(itemRow)},
	}
	waitForStartupRuns(t, w.pool)
	for _, phase := range phases {
		for _, target := range targets {
			if !t.Run(phase.name+"/"+target.kind, func(t *testing.T) {
				req := w.recordingRequest(t, target.kind, target.id)
				waitForFanouts(t, w.pool)
				holder, letGo := phase.hold(t)
				probe := w.requests.observe(req)
				answered := sendInBackground(req)
				pid := phase.waits(t, probe, holder)
				sample, waiting := readRequestLock(t, pgtest.Soon(t), w.pool, probe)
				if !waiting || !strings.Contains(sample.query, "-- name: "+phase.statement+" ") {
					t.Errorf("the recording waits in %q; want its %s", strings.Join(strings.Fields(sample.query), " "), phase.statement)
				}
				for _, r := range phase.shared {
					if lock := requestRowLock(t, w.pool, w.pool, probe, r.from, r.id); lock != "FOR SHARE" {
						t.Errorf("the recording holds %s %s while it waits in %s; want FOR SHARE", r, lock, phase.statement)
					}
				}
				for _, r := range phase.free {
					if lock := requestRowLock(t, w.pool, w.pool, probe, r.from, r.id); lock != "no lock" {
						t.Errorf("the recording holds %s %s while it waits in %s; want no lock", r, lock, phase.statement)
					}
				}
				if held := requestAdvisoryLocks(t, w.pool, pid); held != 0 {
					t.Errorf("the recording holds %d advisory locks while it waits in %s; want none", held, phase.statement)
				}
				if written := requestWrittenTables(t, w.pool, pid); len(written) != 0 {
					t.Errorf("the recording has written %v while it waits in %s; want nothing written", written, phase.statement)
				}
				letGo()
				w.recorded(t, req, receiveWithin(t, answered, 10*time.Second, "the answer to the recording"))
			}) {
				t.FailNow()
			}
		}
	}
	for _, target := range targets {
		if !t.Run("its trim passing over a held visit/"+target.kind, func(t *testing.T) {
			past := w.pastTheCap(t, alice)
			req := w.recordingRequest(t, target.kind, target.id)
			waitForFanouts(t, w.pool)
			other := holding(t, w.pool, "SELECT 1 FROM user_recent_visits WHERE id = $1 FOR UPDATE", past[0])
			a, unwaited := answeredUnlessWaiting(t, w.pool, sendInBackground(req), "TrimRecentVisits")
			if !unwaited {
				release(t, other)
				t.Fatalf("the recording's trim waits for alice's visit %s, which another transaction holds; want it passed over", past[0])
			}
			w.recorded(t, req, a)
			left := queryIDs(t, w.pool, "SELECT id FROM user_recent_visits WHERE id = ANY ($1::uuid[]) AND deleted_at IS NULL", past)
			if n := w.undeletedVisits(t, alice); !slices.Equal(left, past[:1]) || n != navigationdomain.RecentVisitCap+1 {
				t.Errorf("alice's visits past the cap left %v, %d undeleted in acme; want the held one alone, %v, of %d", left, n, past[:1],
					navigationdomain.RecentVisitCap+1)
			}
			release(t, other)
		}) {
			t.FailNow()
		}
	}
}

// recordingRequest is alice's recording of her visit of the target of kind
// with id, checked against the contract.
func (w issueWorld) recordingRequest(t *testing.T, kind string, id uuid.UUID) *http.Request {
	t.Helper()
	req := newRequest(t, http.MethodPost, w.base+"/api/v0/me/recent-visits", w.tokens["alice"],
		fmt.Appendf(nil, `{"entity_type":%q,"entity_id":%q}`, kind, id))
	w.contract.CheckRequest(t, req)
	return req
}

// recorded checks a, the answer to the recording req: 204, as the contract
// declares it.
func (w issueWorld) recorded(t *testing.T, req *http.Request, a answer) {
	t.Helper()
	if a.err != nil {
		t.Fatal(a.err)
	}
	w.contract.CheckResponse(t, req, a.res)
	if a.res.StatusCode != http.StatusNoContent {
		t.Errorf("the recording = %d %s, want 204", a.res.StatusCode, a.body)
	}
}

// undeletedVisits is how many undeleted recent visits owner has in acme.
func (w issueWorld) undeletedVisits(t *testing.T, owner uuid.UUID) int {
	t.Helper()
	var n int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM user_recent_visits WHERE user_id = $1 AND workspace_id = $2 AND deleted_at IS NULL",
		owner, w.acme).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// pastTheCap writes past visits of owner's in acme (seedPastVisits) until he
// has two undeleted visits there past the cap, and answers those two, by
// visit, the newest first, then by id descending, as the trim orders them:
// a recording of a target among his newest keeps the cap's newest and
// deletes the two.
func (w issueWorld) pastTheCap(t *testing.T, owner uuid.UUID) []uuid.UUID {
	t.Helper()
	seedPastVisits(t, w.pool, w.web, owner, freshIDs(navigationdomain.RecentVisitCap+2-w.undeletedVisits(t, owner)))
	past := queryIDs(t, w.pool, `SELECT id FROM user_recent_visits WHERE user_id = $1 AND workspace_id = $2 AND deleted_at IS NULL
		ORDER BY visited_at DESC, id DESC OFFSET $3`, owner, w.acme, navigationdomain.RecentVisitCap)
	if len(past) != 2 {
		t.Fatalf("%s's visits in acme past the cap = %v; want two", owner, past)
	}
	return past
}

// seedPastVisits writes, directly, owner's recent visits of the work items
// items of project, items[0] visited a day and a second ago and each next
// one a second before the last, before every visit a test records, each
// made and last changed by him at its visit. entity_identifier being no
// foreign key, an id no work item has will do (freshIDs). The visit of
// items[n] is of the id ids[n] when ids are given, else of a new one.
// items may be empty.
func seedPastVisits(t *testing.T, pool *pgxpool.Pool, project, owner uuid.UUID, items []uuid.UUID, ids ...uuid.UUID) {
	t.Helper()
	if len(ids) == 0 {
		ids = freshIDs(len(items))
	}
	if len(ids) != len(items) {
		t.Fatalf("seeding past visits of %s: %d ids for %d work items", owner, len(ids), len(items))
	}
	if tag, err := pool.Exec(pgtest.Soon(t), `INSERT INTO user_recent_visits (id, workspace_id, project_id, user_id, entity_type, entity_identifier,
			visited_at, created_by_id, updated_by_id, created_at, updated_at)
		SELECT v.id, p.workspace_id, p.id, $3::uuid, 'issue', v.item, v.at, $3::uuid, $3::uuid, v.at, v.at
		FROM projects AS p,
			(SELECT id, item, now() - interval '1 day' - n::float8 * interval '1 second' AS at
				FROM unnest($1::uuid[], $2::uuid[]) WITH ORDINALITY AS u (id, item, n)) AS v
		WHERE p.id = $4`, ids, items, owner, project); err != nil || tag.RowsAffected() != int64(len(items)) {
		t.Fatalf("seeding %d past visits of %s: %d rows, %v", len(items), owner, tag.RowsAffected(), err)
	}
}

// freshIDs is n new ids, in increasing order, of no row.
func freshIDs(n int) []uuid.UUID {
	ids := make([]uuid.UUID, n)
	for i := range ids {
		ids[i] = uuid.NewV7()
	}
	return ids
}

// requestAdvisoryLocks is how many advisory locks the backend pid holds.
func requestAdvisoryLocks(t *testing.T, pool *pgxpool.Pool, pid uint32) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM pg_locks WHERE pid = $1 AND granted AND locktype = 'advisory'", pid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// holdingRecentVisits holds, in a transaction of its own on pool, the
// advisory lock of user's recent visits in workspace, taken through the
// navigation store's LockRecentVisits, as a recording's last lock takes it
// (P3d spec 2.8): the key is that statement's alone. It answers the
// holder's connection, by which a probe names what waits behind it
// (pgtest.WaitForStatementLockWaitBehind), and its release, which commits
// the transaction, nothing written, and waits for it to have ended; the
// test's cleanup releases it too. The holder waits for no row.
func holdingRecentVisits(t *testing.T, pool *pgxpool.Pool, user, workspace uuid.UUID) (*pgconn.PgConn, func()) {
	t.Helper()
	held, letGo, ended := make(chan *pgconn.PgConn, 1), make(chan struct{}), make(chan struct{})
	var err error
	go func() {
		defer close(ended)
		err = postgres.NewTxManager(pool, time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			if err := navigationpg.New(pool).LockRecentVisits(ctx, user, workspace); err != nil {
				return err
			}
			tx, _ := postgres.Tx(ctx)
			held <- tx.Conn().PgConn()
			<-letGo
			return nil
		})
	}()
	var once sync.Once
	letItGo := func() {
		once.Do(func() {
			close(letGo)
			select {
			case <-ended:
				if err != nil {
					t.Error(err)
				}
			case <-time.After(5 * time.Second):
				t.Errorf("the holder of the advisory lock of %s's recent visits in %s did not end within 5s", user, workspace)
			}
		})
	}
	t.Cleanup(letItGo)
	select {
	case conn := <-held:
		return conn, letItGo
	case <-ended:
		t.Fatalf("holding the advisory lock of %s's recent visits in %s: %v", user, workspace, err)
	case <-time.After(5 * time.Second):
		t.Fatalf("no advisory lock of %s's recent visits in %s held within 5s", user, workspace)
	}
	return nil, nil
}
