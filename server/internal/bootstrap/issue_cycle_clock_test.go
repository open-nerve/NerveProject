package bootstrap

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// cycleEnd is Sprint, a cycle of Web ending three days from now on
// Shanghai's day, Web's time zone (turnCyclesOn), and the last moment of
// that day there: a second before its midnight, which falls at 16:00 UTC
// on the same day, so that UTC's day there is still the cycle's last.
func cycleEnd(t *testing.T, w issueWorld) (sprint uuid.UUID, last time.Time) {
	t.Helper()
	now := time.Now()
	end := shanghaiDay(t, now, 3)
	sprint = w.newCycle(t, "alice", datesBody("Sprint", shanghaiDay(t, now, -1), end))
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	day, err := time.ParseInLocation(time.DateOnly, end, loc)
	if err != nil {
		t.Fatal(err)
	}
	return sprint, day.Add(24*time.Hour - time.Second)
}

// Whether a cycle has ended is judged on the day of the write's clock in
// the project's time zone (M6 P3a spec R2): on the business clock at the
// last second of Sprint's last day in Shanghai, a work item is created in
// it and another moved into it; two seconds later, past Shanghai's
// midnight but still on that day in UTC, and days after the system
// clock's today, both are refused, nothing written. Each write reads its
// clock once (issue_event_writes_test.go's single reading).
func TestJoiningACycleIsJudgedOnTheWritesDay(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	sprint, last := cycleEnd(t, w)
	if last.UTC().Day() != last.Add(2*time.Second).UTC().Day() {
		t.Fatalf("the fixture's moments %s, %s must share their UTC day", last.UTC(), last.Add(2*time.Second).UTC())
	}
	body := `{"cycle_id":"` + sprint.String() + `"}`
	for _, tt := range []struct {
		at              time.Time
		created, change int
	}{{last, http.StatusCreated, http.StatusOK}, {last.Add(2 * time.Second), http.StatusConflict, http.StatusConflict}} {
		item := w.newWorkItemIn(t, w.web, `{"name":"Fix"}`)
		before := w.cycleWritten(t)
		c := &eventCountingClock{base: clockAt(tt.at)}
		route := newIssueRouteWithClock(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), c)
		created := route.answerWithin(t, w.contract, http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", w.ids["bob"],
			`{"name":"Fix","cycle_id":"`+sprint.String()+`"}`, tt.created)
		if n := c.calls.Load(); n != 1 {
			t.Errorf("the creation at %s read the clock %d times, want once", tt.at, n)
		}
		changed := route.answerWithin(t, w.contract, http.MethodPatch, "/api/v0/issues/"+item.String(), w.ids["bob"], body, tt.change)
		if n := c.calls.Load(); n != 2 {
			t.Errorf("the change at %s read the clock %d times, want once", tt.at, n-1)
		}
		if tt.created != http.StatusConflict {
			continue
		}
		for _, answer := range []string{created, changed} {
			if code := refusalOf(t, []byte(answer), false); code != "planning.cycle_completed" {
				t.Errorf("at %s: %s, want planning.cycle_completed", tt.at, answer)
			}
		}
		if after := w.cycleWritten(t); after != before {
			t.Errorf("at %s the refused writes wrote: %s, was %s", tt.at, after, before)
		}
	}
}

// The day is the write's, read after its last lock (M6 P3a spec R2): bob
// moves his work item into Sprint with an image of his in its description
// on the business clock at the last second of Sprint's last day; the
// change waits for the image's row, which a fixture holds, past its checks
// and planning's facts; meanwhile the clock passes Shanghai's midnight.
// Released, the change reads the clock after the image's lock and is
// refused: Sprint has ended on its day. Nothing is written, the image not
// bound.
func TestTheEndOfACycleIsJudgedAfterTheLastLock(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	sprint, last := cycleEnd(t, w)
	item := w.newWorkItemBy(t, "bob", w.web, `{"name":"Fix"}`)
	image := uploadedAssetHTTP(t, w, "bob", "ISSUE_DESCRIPTION", assetScopeBody("issue", item), "body.png", "image/png", assetTestPNG(t))
	c, probe := clockAt(last), requestTx(w.pool)
	route := newIssueRouteWithClock(t, w.pool, probe, c)
	holder := holding(t, w.pool, "SELECT id FROM file_assets WHERE id=$1 FOR SHARE", image.Asset.ID)
	before := w.cycleWritten(t)
	answered := make(chan exchange, 1)
	body := fmt.Sprintf(`{"description_html":%q,"cycle_id":%q}`, issueImageHTML(image.Asset.ID), sprint)
	go func() {
		var e exchange
		e.req, e.rec = route.send(http.MethodPatch, "/api/v0/issues/"+item.String(), w.ids["bob"], body)
		answered <- e
	}()
	requestBlocked(t, w.pool, probe, holder.Conn().PgConn().PID(), "file_assets")
	c.advance(2 * time.Second)
	requestRelease(t, holder)
	w.refused(t, receiveWithin(t, answered, 5*time.Second, "the change's answer"), http.StatusConflict, "planning.cycle_completed")
	if after := w.cycleWritten(t); after != before {
		t.Errorf("the refused change wrote: %s, was %s", after, before)
	}
}

// intoCycle is the body of a change into Web's cycle named name.
func intoCycle(name string) func(*testing.T, issueWorld) string {
	return func(t *testing.T, w issueWorld) string {
		return `{"cycle_id":"` + namedRowID(t, w.pool, "cycles", w.web, name).String() + `"}`
	}
}

// Two changes of one work item into two cycles, the second waiting for
// Web's row while the first holds it (M6 P3a spec R4): the second reads the
// first's cycle under the lock and moves the work item out of it. Both
// answer 200; one cycle row is left undeleted, the second's; the activity
// has the first's joining and the second's move.
func TestTwoChangesOfAWorkItemsCycleSerialize(t *testing.T) {
	prepare := func(t *testing.T, w issueWorld) {
		w.turnCyclesOn(t, w.web)
		w.newCycle(t, "alice", `{"name":"Sprint"}`)
		w.newCycle(t, "alice", `{"name":"Next"}`)
		aWorkItem(t, w)
	}
	inBothOrders(t, prepare, "alice", changesIssue("alice", intoCycle("Sprint")), "bob", changesIssue("bob", intoCycle("Next")),
		func(t *testing.T, w issueWorld, aliceFirst bool, alice, bob exchange) {
			w.answered(t, alice, http.StatusOK)
			w.answered(t, bob, http.StatusOK)
			first, second := "Sprint", "Next"
			if !aliceFirst {
				first, second = second, first
			}
			item := w.workItem(t)
			n := cycleNames{w: w, cycles: map[uuid.UUID]string{namedRowID(t, w.pool, "cycles", w.web, "Sprint"): "Sprint",
				namedRowID(t, w.pool, "cycles", w.web, "Next"): "Next"}, moments: map[string]string{}}
			var undeleted int
			for _, row := range n.cycleRows(t, item) {
				if !strings.Contains(row, ", left by ") {
					undeleted++
				}
			}
			activity := n.cycleActivity(t, item)
			if undeleted != 1 || len(activity) != 3 || !strings.HasPrefix(activity[1], `cycles created ""→"`+first+`"`) ||
				!strings.HasPrefix(activity[2], `cycles updated "`+first+`"→"`+second+`"`) || w.issueCycle(t, item) != second {
				t.Errorf("after %s then %s: rows %q, activity %q", first, second, n.cycleRows(t, item), activity)
			}
		})
}

// issueCycle is the name of the cycle getIssue answers the work item in.
func (w issueWorld) issueCycle(t *testing.T, item uuid.UUID) string {
	t.Helper()
	var got joinedItem
	decodeAnswer(t, w.send(t, "alice", http.MethodGet, "/api/v0/issues/"+item.String(), "", http.StatusOK), &got)
	if got.CycleID == nil {
		return "-"
	}
	var name string
	if err := w.pool.QueryRow(t.Context(), "SELECT name FROM cycles WHERE id = $1", *got.CycleID).Scan(&name); err != nil {
		t.Fatal(err)
	}
	return name
}
