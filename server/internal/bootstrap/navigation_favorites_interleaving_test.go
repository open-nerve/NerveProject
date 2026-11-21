package bootstrap

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A write of one's favorite interleaved with a write that decides it (M7
// P3c spec 2.5, 2.6, 2.8, 4), each on the app as it runs, both through its
// API: in each cell another transaction holds the one row both writes
// lock, the target's project's, a view's, a folder's or a favorite's, with
// a lock each waits for; the first write is sent and is proven waiting for
// it at its own lock statement; then the second, proven waiting at its
// own, behind the first, whose PID blocks it; meanwhile neither holds a
// lock on the rows the total order of favorites' rows takes after the one
// it waits for (free). Then the other transaction ends: the first takes
// the row and commits before the second takes it, and the second decides
// under its locks on what the first left. Each wait is the request's own,
// its PID's, and caused by the other transaction (requestBlocked), each
// row it holds or does not its own transaction's lock (requestRowLock).

// TestFavoriteWritesInterleaveWithArchives races a creation, a change and a
// deletion of a favorite, in each order, with what decides it: a project's
// archive, a cycle's archive, a cycle's deletion, a view's deletion, a
// project's deletion and a member's removal (favoriteCompetitors); and a
// folder's deletion with a move of a favorite into it, a creation of one in
// it and a change of one in it that names it its parent again
// (favoritesRaceAFolder); and the deletion of a favorite, and of a folder
// that holds it, with a creation of a favorite of its target again by the
// same account (favoritesRaceAgain). One world, issueWorld's on an observed
// app, with Web's cycles and views turned on: each cell makes its own
// target, so that no cell sees another's; once the start-time runs have
// run, so that none queues on a row a cell holds (03 §5), each group is a
// subtest, and each cell one of it. No work item is made, so no fan-out
// is pending; a project's deletion waits for its jobs, so that none takes
// the app's connections from the next cell's writes.
func TestFavoriteWritesInterleaveWithArchives(t *testing.T) {
	dbURL := pgtest.NewDatabase(t)
	w := newIssueWorldOnRequests(t, dbURL, openPool(t, dbURL), func(*config.Config) {}, &requestQueries{})
	w.turnCyclesOn(t, w.web)
	w.changeProject(t, w.web, `{"issue_views_view":true}`)
	waitForStartupRuns(t, w.pool)
	for _, c := range favoriteCompetitors(w) {
		t.Run(c.name, func(t *testing.T) { favoritesRaceATarget(t, w, c) })
	}
	t.Run("a folder's deletion", func(t *testing.T) { favoritesRaceAFolder(t, w) })
	t.Run("one's favorite deleted and created again", func(t *testing.T) { favoritesRaceAgain(t, w) })
}

// favoriteRacer is one write of a race through the app: the account that
// sends it, its method, its path and its body; the statement, its sqlc
// query's name, at which it waits for the row held; and the rows it holds
// no lock on meanwhile (free).
type favoriteRacer struct {
	actor, method, path, body, statement string
	free                                 []lockedRow
}

// favoriteRowLocked is the lockedRow of the favorite id.
func favoriteRowLocked(id uuid.UUID) lockedRow { return lockedRow{"user_favorites WHERE id = $1", id} }

// racedAnswer is what a racer answered: its request and its whole answer.
type racedAnswer struct {
	req   *http.Request
	whole wholeAnswer
}

// race has another transaction hold the row id of table with lock, then
// sends first and second through the app, each proven waiting for it at
// its statement, the second behind the first, and neither holding a lock
// on its free rows; the other transaction then ends, and each answers
// within 10 seconds, checked against the contract. A cell that fails
// before ends the other transaction and collects both answers once it is
// over, so that no write of it is left to run into the next cell.
func (w issueWorld) race(t *testing.T, table, lock string, id uuid.UUID, first, second favoriteRacer) (racedAnswer, racedAnswer) {
	t.Helper()
	waitForFanouts(t, w.pool)
	racers := []favoriteRacer{first, second}
	pending := make([]<-chan answer, len(racers))
	t.Cleanup(func() {
		for _, ch := range pending {
			if ch == nil {
				continue
			}
			select {
			case <-ch:
			case <-time.After(10 * time.Second):
				t.Error("a raced write did not answer within 10s")
			}
		}
	})
	other := holding(t, w.pool, "SELECT 1 FROM "+table+" WHERE id = $1 "+lock, id)
	holder := other.Conn().PgConn().PID()
	reqs, probes := make([]*http.Request, len(racers)), make([]*requestProbe, len(racers))
	var firstPID uint32
	for n, r := range racers {
		var body []byte
		if r.body != "" {
			body = []byte(r.body)
		}
		reqs[n] = newRequest(t, r.method, w.base+r.path, w.tokens[r.actor], body)
		w.contract.CheckRequest(t, reqs[n])
		probes[n] = w.requests.observe(reqs[n])
		pending[n] = sendInBackground(reqs[n])
		pid := requestBlocked(t, w.pool, probes[n], holder, table)
		sample, waiting := readRequestLock(t, pgtest.Soon(t), w.pool, probes[n])
		if !waiting || !strings.Contains(sample.query, "-- name: "+r.statement+" ") {
			t.Errorf("%s %s waits (%v) in %q; want its %s", r.method, r.path, waiting, strings.Join(strings.Fields(sample.query), " "), r.statement)
		}
		if n == 0 {
			firstPID = pid
		} else if !slices.Contains(sample.blockers, int32(firstPID)) {
			t.Errorf("%s %s waits behind %v; want behind the first write's PID %d", r.method, r.path, sample.blockers, firstPID)
		}
	}
	for n, r := range racers {
		for _, row := range r.free {
			if got := requestRowLock(t, w.pool, w.pool, probes[n], row.from, row.id); got != "no lock" {
				t.Errorf("%s %s holds %s %s while it waits at %s; want no lock", r.method, r.path, row, got, r.statement)
			}
		}
	}
	if err := other.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	out := make([]racedAnswer, len(racers))
	for n := range racers {
		a := receiveWithin(t, pending[n], 10*time.Second, "the answer to "+reqs[n].Method+" "+reqs[n].URL.Path)
		pending[n] = nil
		if a.err != nil {
			t.Fatal(a.err)
		}
		w.contract.CheckResponse(t, reqs[n], a.res)
		out[n] = racedAnswer{reqs[n], wholeAnswerOf(a.res, a.body)}
	}
	return out[0], out[1]
}

// answeredRace checks that a answered want and returns its body.
func answeredRace(t *testing.T, a racedAnswer, want int) string {
	t.Helper()
	if a.whole.status != want {
		t.Errorf("%s %s = %d %s; want %d", a.req.Method, a.req.URL.Path, a.whole.status, a.whole.body, want)
	}
	return a.whole.body
}

// refusedRace checks that a answered want with the refusal code (refusalOf:
// the code, then each field's).
func refusedRace(t *testing.T, a racedAnswer, want int, code string) {
	t.Helper()
	if body := answeredRace(t, a, want); a.whole.status == want && refusalOf(t, []byte(body), false) != code {
		t.Errorf("%s %s = %s; want %s", a.req.Method, a.req.URL.Path, body, code)
	}
}

// refusedRaceAsAbsent checks that a, the actor's change or deletion of a
// favorite with body, answered 404 navigation.favorite_not_found, byte for
// byte as the same write on an id no favorite has (spec 2.7).
func (w issueWorld) refusedRaceAsAbsent(t *testing.T, a racedAnswer, actor, body string) {
	t.Helper()
	absent := w.whole(t, actor, a.req.Method, favoritePath(uuid.NewV7()), body)
	if absent.status != http.StatusNotFound || problemCode(t, []byte(absent.body)) != "navigation.favorite_not_found" {
		t.Fatalf("%s of an absent favorite as %s: %+v; want 404 navigation.favorite_not_found", a.req.Method, actor, absent)
	}
	if a.whole != absent {
		t.Errorf("%s %s as %s:\n%+v\nwant, as an absent one's,\n%+v", a.req.Method, a.req.URL.Path, actor, a.whole, absent)
	}
}

// liveFavoritesOf counts every account's undeleted favorites of the target
// id, and those whose parent it is.
func liveFavoritesOf(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM user_favorites WHERE (entity_identifier = $1 OR parent_id = $1) AND deleted_at IS NULL",
		id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// favoriteDeletedBefore checks that the favorite id is deleted by by, and
// last changed by him at that moment, before at when at is given: its own
// deletion's, which a later cascade or job leaves as it is.
func favoriteDeletedBefore(t *testing.T, pool *pgxpool.Pool, id, by uuid.UUID, at time.Time) {
	t.Helper()
	r := favoriteRowOf(t, pool, id)
	if r.deletedAt == nil || !r.deletedAt.Equal(r.updatedAt) || r.updatedBy != by || (!at.IsZero() && !r.deletedAt.Before(at)) {
		t.Errorf("the favorite %s: %s; want deleted and last changed by %s at one moment, before %s", id, r, by, at.UTC().Format(time.RFC3339Nano))
	}
}
