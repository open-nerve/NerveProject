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

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// serializationPause is only a test barrier before the real business callback.
// Its unrelated users lock is not part of an issue write's production lock graph.
type serializationPause struct {
	shared.TxManager
	user uuid.UUID
}

func (p serializationPause) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return p.TxManager.WithinTx(ctx, func(ctx context.Context) error {
		tx, ok := postgres.Tx(ctx)
		if !ok {
			return fmt.Errorf("serialization pause has no transaction")
		}
		tag, err := tx.Exec(ctx, "SELECT 1 FROM users WHERE id=$1 FOR SHARE", p.user)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("serialization pause found %d users, want one", tag.RowsAffected())
		}
		return fn(ctx)
	})
}

// A table-wide waiter would release the first transaction for trim while the
// second request still waits before its project lock. Its eventual 404 and zero
// writes alone cannot distinguish this from a real lock-under-recheck story.
func TestIssueSerializationIgnoresTrimWaiter(t *testing.T) {
	w := newIssueWorld(t)
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"network":0}`, http.StatusOK)
	aWorkItem(t, w)
	before := collaborationRows(t, w, w.workItem(t))
	stamps := w.workItemStamps(t, w.workItem(t))
	users := holding(t, w.pool, "SELECT 1 FROM users WHERE id=$1 FOR UPDATE", w.ids["walt"])
	usersPID := users.Conn().PgConn().PID()
	var releaseOnce sync.Once
	releaseUsers := func() { releaseOnce.Do(func() { requestRelease(t, users) }) }
	defer releaseUsers()

	var firstProbe, secondProbe *requestProbe
	var firstGate *gate
	trimReady := make(chan struct{})
	type trimming struct {
		probe   *requestProbe
		collect func()
	}
	trimStarted := make(chan trimming, 1)
	var collectCoordinator func()
	first := func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		held := tx.(heldTx)
		firstProbe, firstGate = held.TxManager.(*requestProbe), held.gate
		send := removesBob(t, w, tx, e)
		return func() error {
			// Cleanup joins the first request before the second. Release its
			// pause before the first can report completion, including on failure.
			defer releaseUsers()
			return send()
		}
	}
	second := func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		// This independent observation also works with the old plain TxManager.
		// A missing new helper probe is never the regression's failure condition.
		secondProbe = &requestProbe{TxManager: tx}
		if observing, ok := tx.(*requestProbe); ok {
			observing.onLockWait = func(sample requestLockSample) {
				select {
				case <-trimReady:
					if strings.Contains(sample.query, "FROM users") && slices.Contains(sample.blockers, int32(usersPID)) {
						releaseUsers()
					}
				default:
				}
			}
		}
		// Positive PostgreSQL observations establish both barriers. No elapsed
		// time is used to infer that the serializer has kept the first gate shut.
		collectCoordinator = requestBackground(t, func() error {
			requestBlocked(t, w.pool, secondProbe, usersPID, "users")
			probe, collect := observedTrim(t, w.pool, func() { releaseUsers(); requestOpen(firstGate) })
			trimStarted <- trimming{probe, collect}
			requestBlocked(t, w.pool, probe, requestPID(t, firstProbe), "projects")
			close(trimReady)
			return nil
		}, func() { releaseUsers(); requestOpen(firstGate) })
		return changesIssue("bob", named("Must not write"))(t, w, serializationPause{secondProbe, w.ids["walt"]}, e)
	}

	var removed, changed exchange
	w.serializeAtObserved(t, "projects", first, second, &removed, &changed, func(holder *requestProbe) {
		defer releaseUsers()
		receiveWithin(t, trimReady, 2*time.Second, "trim waiting while the second request is paused")
		trim := receiveWithin(t, trimStarted, time.Second, "identified trim")
		// Collect trim after the serializer releases the first, not from this
		// callback: trim itself is waiting for that still-held transaction.
		trimStarted <- trim
		firstPID := requestPID(t, holder)
		sample, waiting := readRequestLock(t, pgtest.Soon(t), w.pool, secondProbe)
		chain := requestBlockerChain(t, w.pool, sample.pid)
		if !waiting || !strings.Contains(sample.query, "projects") || !slices.Contains(chain, int32(firstPID)) {
			t.Errorf("serializer accepted trim instead of the second request: secondPID=%d wait=%t SQL=%q blockers=%v chain=%v; want its projects wait reaching firstPID=%d", sample.pid, waiting, sample.query, sample.blockers, chain, firstPID)
		}
		if sample.pid == requestPID(t, trim.probe) {
			t.Error("trim supplied the second request's identity")
		}
		if lock := requestGateRowLock(t, w.pool, w.pool, holder, "projects WHERE id=$1", w.web); lock != "FOR NO KEY UPDATE" {
			t.Fatal("first request lost its project N lock", lock)
		}
	})
	collectCoordinator()
	trim := receiveWithin(t, trimStarted, time.Second, "trim completion handle")
	trim.collect()
	w.answered(t, removed, http.StatusNoContent)
	w.refused(t, changed, http.StatusNotFound, "issue.not_found")
	sameCollaborationRows(t, w, w.workItem(t), before, stamps)
}
