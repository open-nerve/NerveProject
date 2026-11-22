package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	navigationpg "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres"
	navigationdomain "github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The cap's groups of TestRecentVisitWritesInterleave
// (navigation_recent_visits_interleaving_test.go): two recordings at once,
// and the trim and a visit another transaction holds.

// visitTarget is a recording's target: its kind and its id.
type visitTarget struct {
	kind string
	id   uuid.UUID
}

// recordsTogether is the group of two recordings of one account's at once
// (spec 2.8, 4; re-92 I2): walt has 19 visits in acme, past ones
// (seedPastVisits), when he records two new targets at once, Web and Seen
// 0, and then one new target twice at once, Seen 1. Each pair waits for
// the advisory lock of his visits in acme, which a transaction of the test
// holds (recordingTogether), and once it is let go each recording runs
// after the other: both answer 204, no 23505 and no 500; he has 20 visits
// there, each target's one, and no other row, deleted or not, the trim
// having deleted the oldest for good. These are the results of the lock,
// not what proves a recording takes it: one that took none would run both
// at once, and which committed first would be a race; the lock-order
// test's third phase proves that (TestRecordingARecentVisitTakesItsLocks).
func recordsTogether(t *testing.T, w visitRaceWorld) {
	walt := w.ids["walt"]
	seedPastVisits(t, w.pool, w.web, walt, freshIDs(navigationdomain.RecentVisitCap-1))
	capped := func(t *testing.T, targets ...uuid.UUID) {
		t.Helper()
		rows := visitRowsWhere(t, w.pool, "user_id = $1 AND workspace_id = $2", walt, w.acme)
		live := visitRowsWhere(t, w.pool, "user_id = $1 AND workspace_id = $2 AND deleted_at IS NULL", walt, w.acme)
		if len(rows) != navigationdomain.RecentVisitCap || len(live) != len(rows) {
			t.Errorf("walt's visits in acme: %d rows, %d undeleted; want %d, every one undeleted", len(rows), len(live), navigationdomain.RecentVisitCap)
		}
		for _, target := range targets {
			if of := visitRowsWhere(t, w.pool, "user_id = $1 AND entity_identifier = $2", walt, target); len(of) != 1 || of[0].deletedAt != nil {
				t.Errorf("walt's visits of %s: %v; want one, undeleted", target, of)
			}
		}
	}
	t.Run("two new targets past the cap", func(t *testing.T) {
		w.recordingTogether(t, "walt", visitTarget{"project", w.web}, visitTarget{"issue", w.seen[0]})
		capped(t, w.web, w.seen[0])
	})
	t.Run("one target twice", func(t *testing.T) {
		w.recordingTogether(t, "walt", visitTarget{"issue", w.seen[1]}, visitTarget{"issue", w.seen[1]})
		capped(t, w.web, w.seen[0], w.seen[1])
	})
}

// recordingTogether sends actor's recordings of targets at once: a
// transaction of the test holds the advisory lock of his visits in acme
// (holdingRecentVisits), and each recording is sent and proven waiting for
// it, in LockRecentVisits, by its own request probe (waitingForVisitsLock),
// each its own backend. Then the transaction ends, and each answers 204
// within 10 seconds, checked against the contract. A cell that fails
// before ends the transaction and collects every answer once it is over, so
// that no recording of it is left to run into the next cell.
func (w visitRaceWorld) recordingTogether(t *testing.T, actor string, targets ...visitTarget) {
	t.Helper()
	waitForFanouts(t, w.pool)
	reqs, pending := make([]*http.Request, len(targets)), make([]<-chan answer, len(targets))
	t.Cleanup(func() {
		for _, ch := range pending {
			if ch == nil {
				continue
			}
			select {
			case <-ch:
			case <-time.After(10 * time.Second):
				t.Error("a recording did not answer within 10s")
			}
		}
	})
	holder, letGo := holdingRecentVisits(t, w.pool, w.ids[actor], w.acme)
	waiters := map[uint32]bool{}
	for n, target := range targets {
		reqs[n] = newRequest(t, http.MethodPost, w.base+recordVisitPath, w.tokens[actor], []byte(visitBody(target.kind, target.id)))
		w.contract.CheckRequest(t, reqs[n])
		probe := w.requests.observe(reqs[n])
		pending[n] = sendInBackground(reqs[n])
		if n == 0 {
			pgtest.WaitForStatementLockWaitBehind(t, w.pool, holder, 5*time.Second, "LockRecentVisits")
		}
		waiters[waitingForVisitsLock(t, w.pool, probe, holder.PID()).pid] = true
	}
	if len(waiters) != len(targets) {
		t.Fatalf("%d backends wait for the advisory lock of %s's visits; want each recording's own, %d", len(waiters), actor, len(targets))
	}
	letGo()
	for n := range targets {
		a := receiveWithin(t, pending[n], 10*time.Second, "the answer to "+actor+"'s recording")
		pending[n] = nil
		w.recorded(t, reqs[n], a)
	}
}

// waitingForVisitsLock returns once the recording the request probe
// observes waits for the advisory lock of its account's visits, in
// LockRecentVisits, behind the backend holder: its own wait, by its PID,
// read the same twice (sameRequestLock), its transaction's too. It answers
// that wait; none within 5s fails the test.
func waitingForVisitsLock(t *testing.T, pool *pgxpool.Pool, probe *requestProbe, holder uint32) requestLockSample {
	t.Helper()
	var sample requestLockSample
	requestEventuallyWithin(t, "a recording waiting for the advisory lock of its visits", 5*time.Second, func(ctx context.Context) bool {
		var waiting bool
		sample, waiting = readRequestLock(t, ctx, pool, probe)
		if !waiting || sample.locktype != "advisory" || !strings.Contains(sample.query, "-- name: LockRecentVisits ") ||
			!slices.Contains(sample.blockers, int32(holder)) {
			return false
		}
		after, still := readRequestLock(t, ctx, pool, probe)
		return still && sameRequestLock(sample, after)
	})
	return sample
}

// trimsPastAHeldVisit is the group of the trim and a visit another
// transaction holds (spec 2.5, 2.8, 4; re-92 I3 ③, I4 ①). wanda has 20
// visits in acme, past ones, of Seen 1 to Seen 20, Seen 20's the oldest
// (seedPastVisits). In each cell a transaction of the test soft-deletes
// her oldest visit through the navigation store, as a work item's
// deletion's cascade does, and holds it (holdingVisitsDeletion): the app
// holds no visit it could be stopped at, navigation being issue.deleted's
// last subscriber. Her recording of a new target answers 204, having waited
// for no row (answeredUnlessWaiting): its trim passes over the held one,
// and while the transaction is open she has 21 undeleted visits.
//   - Committed: her held visit is deleted at the transaction's moment, by
//     its deleter, alice, its visit kept; she has 20 undeleted visits, and
//     lists them all, Seen 0, the one she recorded, first.
//   - Rolled back: Seen 19's, held, is undeleted again: she has 21, the
//     transient one past the cap (spec 2.5), and lists 20, not it (R8).
//     Her next recording, Seen 0 again, deletes it for good and leaves her
//     20.
func trimsPastAHeldVisit(t *testing.T, w visitRaceWorld) {
	wanda := w.ids["wanda"]
	seedPastVisits(t, w.pool, w.web, wanda, w.seen[1:])
	lists := func(t *testing.T, want ...uuid.UUID) {
		t.Helper()
		if got := w.listedTargets(t, "wanda", "acme"); !slices.Equal(got, want) {
			t.Errorf("wanda lists %v; want %v", got, want)
		}
	}
	t.Run("committed", func(t *testing.T) {
		at, end := w.recordingPastAHeldVisit(t, "wanda", w.seen[20], w.seen[0])
		end(true)
		if n := w.undeletedVisits(t, wanda); n != navigationdomain.RecentVisitCap {
			t.Errorf("wanda has %d undeleted visits in acme once the transaction committed; want %d", n, navigationdomain.RecentVisitCap)
		}
		held := visitRowsWhere(t, w.pool, "user_id = $1 AND entity_identifier = $2", wanda, w.seen[20])
		if len(held) != 1 || held[0].deletedAt == nil || !held[0].deletedAt.Equal(at) || !held[0].updatedAt.Equal(at) || held[0].updatedBy == nil ||
			*held[0].updatedBy != w.ids["alice"] || !held[0].visitedAt.Before(at) {
			t.Errorf("wanda's visit of Seen 20: %v; want it deleted and last changed at %s by alice, its visit kept", held, at.UTC().Format(time.RFC3339Nano))
		}
		lists(t, w.seen[:20]...)
	})
	t.Run("rolled back", func(t *testing.T) {
		_, end := w.recordingPastAHeldVisit(t, "wanda", w.seen[19], w.seen[20])
		end(false)
		if n := w.undeletedVisits(t, wanda); n != navigationdomain.RecentVisitCap+1 {
			t.Errorf("wanda has %d undeleted visits in acme once the transaction rolled back; want %d", n, navigationdomain.RecentVisitCap+1)
		}
		liveVisit(t, w.pool, wanda, w.seen[19])
		lists(t, slices.Concat(w.seen[20:21], w.seen[:19])...)
		w.visit(t, "wanda", "issue", w.seen[0])
		if n := w.undeletedVisits(t, wanda); n != navigationdomain.RecentVisitCap {
			t.Errorf("wanda has %d undeleted visits in acme after her next recording; want %d", n, navigationdomain.RecentVisitCap)
		}
		if gone := visitRowsWhere(t, w.pool, "user_id = $1 AND entity_identifier = $2", wanda, w.seen[19]); len(gone) != 0 {
			t.Errorf("wanda's visit of Seen 19, past the cap: %v; want it deleted for good", gone)
		}
		lists(t, slices.Concat(w.seen[:1], w.seen[20:21], w.seen[1:19])...)
	})
}

// recordingPastAHeldVisit is actor's recording of his visit of target, a
// new one, while a transaction of the test holds his oldest visit, of the
// work item held, the 20th of his 20 undeleted visits in acme, which it
// soft-deletes as alice's deletion of held would (holdingVisitsDeletion).
// The recording answers 204, checked against the contract, without
// waiting for a row in its trim (answeredUnlessWaiting); he has the cap and
// one more undeleted visits. It answers the transaction's moment and its
// end, which commits or rolls it back.
func (w visitRaceWorld) recordingPastAHeldVisit(t *testing.T, actor string, held, target uuid.UUID) (time.Time, func(commit bool)) {
	t.Helper()
	owner := w.ids[actor]
	visited := queryIDs(t, w.pool, `SELECT entity_identifier FROM user_recent_visits WHERE user_id = $1 AND workspace_id = $2 AND deleted_at IS NULL
		ORDER BY visited_at DESC, id DESC`, owner, w.acme)
	if len(visited) != navigationdomain.RecentVisitCap || visited[len(visited)-1] != held {
		t.Fatalf("%s's undeleted visits in acme, the newest first: %v; want %d, the last %s's", actor, visited, navigationdomain.RecentVisitCap, held)
	}
	waitForFanouts(t, w.pool)
	at := time.Now().UTC().Truncate(time.Microsecond)
	end := holdingVisitsDeletion(t, w.pool, w.acme, held, w.ids["alice"], at)
	req := newRequest(t, http.MethodPost, w.base+recordVisitPath, w.tokens[actor], []byte(visitBody("issue", target)))
	w.contract.CheckRequest(t, req)
	answered := sendInBackground(req)
	a, unwaited := answeredUnlessWaiting(t, w.pool, answered, "TrimRecentVisits")
	if !unwaited {
		end(false)
		receiveWithin(t, answered, 10*time.Second, "the answer to "+actor+"'s recording")
		t.Fatalf("%s's recording waits in its trim for the visit of %s another transaction holds; want it passed over", actor, held)
	}
	w.recorded(t, req, a)
	if n := w.undeletedVisits(t, owner); n != navigationdomain.RecentVisitCap+1 {
		t.Errorf("%s has %d undeleted visits in acme while the visit of %s is held; want %d", actor, n, held, navigationdomain.RecentVisitCap+1)
	}
	return at, end
}

// errHeldRolledBack ends holdingVisitsDeletion's transaction by rolling it
// back.
var errHeldRolledBack = errors.New("the held visits' deletion rolled back")

// holdingVisitsDeletion soft-deletes, in a transaction of its own on pool,
// every undeleted visit of the work item item in workspace, through the
// navigation store's DeleteIssueRecentVisits, at at by by, as the cascade
// of a work item's deletion does in the deletion's transaction (spec 2.6),
// and holds them so, locked. It answers the transaction's end, which
// commits it or rolls it back and waits for it to have ended; the test's
// cleanup rolls it back.
func holdingVisitsDeletion(t *testing.T, pool *pgxpool.Pool, workspace, item, by uuid.UUID, at time.Time) func(commit bool) {
	t.Helper()
	held, decided, ended := make(chan struct{}, 1), make(chan bool, 1), make(chan struct{})
	var err error
	go func() {
		defer close(ended)
		err = postgres.NewTxManager(pool, time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
			if err := navigationpg.New(pool).DeleteIssueRecentVisits(ctx, workspace, item, at, &by); err != nil {
				return err
			}
			held <- struct{}{}
			if !<-decided {
				return errHeldRolledBack
			}
			return nil
		})
	}()
	var once sync.Once
	end := func(commit bool) {
		once.Do(func() {
			decided <- commit
			select {
			case <-ended:
				if commit && err != nil || !commit && !errors.Is(err, errHeldRolledBack) {
					t.Errorf("ending the held deletion of the visits of %s (commit %t): %v", item, commit, err)
				}
			case <-time.After(5 * time.Second):
				t.Errorf("the held deletion of the visits of %s did not end within 5s", item)
			}
		})
	}
	t.Cleanup(func() { end(false) })
	select {
	case <-held:
		return end
	case <-ended:
		t.Fatalf("deleting the visits of %s: %v", item, err)
	case <-time.After(5 * time.Second):
		t.Fatalf("the visits of %s not deleted within 5s", item)
	}
	return nil
}
