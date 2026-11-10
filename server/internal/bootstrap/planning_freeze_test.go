package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	planningpg "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres"
	planningriver "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/river"
	planningapp "github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
	planningdomain "github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// freezeCyclesWith is the freeze job's use case as planning.New wires it,
// on pool, with the cycles' store cycles, the transactions tx and the clock
// c.
func freezeCyclesWith(pool *pgxpool.Pool, cycles planningapp.FreezeStore, tx shared.TxManager, c planningapp.Clock) *planningapp.FreezeCycles {
	projects := planningProjects{projects: project.Provide(pool, nil).PlanningProjects}
	return planningapp.NewFreezeCycles(planningapp.FreezeCyclesDeps{Projects: projects, Groups: projects, Cycles: cycles,
		Stats: planningIssues{stats: issue.Provide(pool, nil).PlanningStats}, Tx: tx, Clock: c})
}

// freezeCyclesOn is freezeCyclesWith the store on pool, in the job's own
// transactions as the app's (newIssueJobTx), on the clock c.
func freezeCyclesOn(pool *pgxpool.Pool, c planningapp.Clock) *planningapp.FreezeCycles {
	tx := newIssueJobTx(pool, config.Config{Database: config.DatabaseConfig{CommitTimeout: 2 * time.Second}, Issue: testIssueConfig})
	return freezeCyclesWith(pool, planningpg.New(pool), tx, c)
}

// cycleIn is the id of the cycle alice creates in project with body.
func (w issueWorld) cycleIn(t *testing.T, project uuid.UUID, body string) uuid.UUID {
	t.Helper()
	var c cycleAnswer
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+project.String()+"/cycles", body, http.StatusCreated), &c)
	return c.ID
}

// frozenOf is the frozen progress of the cycle id as stored, nil when it is
// not frozen.
func frozenOf(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) *planningdomain.Frozen {
	t.Helper()
	var raw []byte
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT progress_snapshot FROM cycles WHERE id = $1", id).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if raw == nil {
		return nil
	}
	f, err := planningdomain.DecodeFrozen(raw)
	if err != nil {
		t.Fatal(err)
	}
	return &f
}

// cyclesBesideTheirFreeze are every cycle's columns but its frozen progress.
func cyclesBesideTheirFreeze(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var rows string
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT coalesce(jsonb_agg(to_jsonb(c) - 'progress_snapshot' ORDER BY id), '[]')::text
		FROM cycles c`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	return rows
}

// The job freezes, on each project's day at its clock's reading, the cycles
// that have ended there (M6 P4 spec 2.5, R5, R6). Its clock stands at 16:30
// UTC on 31 March 2044, already 1 April in Shanghai, and the app's runs on
// from there. Web, in Shanghai, has Ended, from 20 to 31 March, which a, in
// Todo, bob's, labelled Bug, b, bob's work item, done, alice's, c, bob's,
// cancelled, his, labelled Bug, and d, in the backlog, are in, and e,
// archived, and f, deleted, were; Old, from 1 to 10 March, archived; Today,
// on 1 April alone; Draft, without dates; and Gone, from 11 to 12 March,
// deleted. Ops, in UTC, has Last day, from 20 to 31 March, and Done, from 1
// to 10 March. Then Web's cycles are turned off and Ops is archived. The run
// freezes Ended, Old and Done, each as alice reads its live progress then,
// at the run's moment, and not Last day, whose last day is today in UTC,
// Today, Draft or Gone; and it changes nothing else of any cycle. A second
// run freezes none and leaves each freeze as it was.
func TestFreezeCyclesOnEachProjectsDay(t *testing.T) {
	at := time.Date(2044, 3, 31, 16, 30, 0, 0, time.UTC)
	if day := shanghaiDay(t, at, 0); day != "2044-04-01" {
		t.Fatalf("the job's clock %s must be on 1 April in Shanghai, is on %s", at, day)
	}
	w := newIssueWorldWith(t, func(c *config.Config) { c.Clock.Offset = time.Until(at) })
	// The app's own run at start, on its clock, finds nothing yet to freeze.
	waitForStartupRuns(t, w.pool)
	w.turnCyclesOn(t, w.web)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+ops.String(), `{"cycle_view":true,"timezone":"UTC"}`, http.StatusOK)
	// Ended takes its work items while it is ahead, then its days move back.
	ended := w.newCycle(t, "alice", datesBody("Ended", "2044-04-10", "2044-04-20"))
	backlog, cancelled := stateID(t, w.pool, w.web, "Backlog"), stateID(t, w.pool, w.web, "Cancelled")
	alice, bob := w.ids["alice"], w.ids["bob"]
	inEnded := `,"cycle_id":"` + ended.String() + `"`
	item := func(by, name string, state uuid.UUID, assignees, labels []uuid.UUID) uuid.UUID {
		return w.newWorkItemBy(t, by, w.web, workItemBody(name, state, assignees, labels, inEnded))
	}
	item("alice", "A", w.todo, []uuid.UUID{bob}, []uuid.UUID{w.bug})
	item("bob", "B", w.done, []uuid.UUID{alice}, nil)
	item("bob", "C", cancelled, []uuid.UUID{bob}, []uuid.UUID{w.bug})
	item("alice", "D", backlog, nil, nil)
	e, f := item("alice", "E", w.todo, nil, nil), item("bob", "F", w.todo, nil, nil)
	listSeedSQL(t, w, "UPDATE issues SET archived_at = now() WHERE id = $1", e)
	w.send(t, "bob", http.MethodDelete, "/api/v0/issues/"+f.String(), "", http.StatusNoContent)
	w.send(t, "alice", http.MethodPatch, "/api/v0/cycles/"+ended.String(), `{"start_date":"2044-03-20","end_date":"2044-03-31"}`, http.StatusOK)
	old := w.newCycle(t, "alice", datesBody("Old", "2044-03-01", "2044-03-10"))
	w.send(t, "alice", http.MethodPost, "/api/v0/cycles/"+old.String()+"/archive", "", http.StatusOK)
	today := w.newCycle(t, "alice", datesBody("Today", "2044-04-01", "2044-04-01"))
	draft := w.newCycle(t, "alice", `{"name":"Draft"}`)
	gone := w.newCycle(t, "alice", datesBody("Gone", "2044-03-11", "2044-03-12"))
	w.send(t, "alice", http.MethodDelete, "/api/v0/cycles/"+gone.String(), "", http.StatusNoContent)
	lastDay := w.cycleIn(t, ops, datesBody("Last day", "2044-03-20", "2044-03-31"))
	done := w.cycleIn(t, ops, datesBody("Done", "2044-03-01", "2044-03-10"))
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"cycle_view":false}`, http.StatusOK)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+ops.String()+"/archive", "", http.StatusOK)

	before := cyclesBesideTheirFreeze(t, w.pool)
	if n, err := freezeCyclesOn(w.pool, clocktest.At(at)).Execute(pgtest.Soon(t)); err != nil || n != 3 {
		t.Fatalf("the run froze %d, %v; want Ended, Old and Done", n, err)
	}
	frozen := map[string]uuid.UUID{"Ended": ended, "Old": old, "Done": done}
	for name, id := range frozen {
		read := w.cycleProgress(t, "alice", id)
		want := planningdomain.Frozen{Progress: read.Live, FrozenAt: at}
		if got := frozenOf(t, w.pool, id); got == nil || !reflect.DeepEqual(*got, want) || read.Snapshot == nil || !reflect.DeepEqual(*read.Snapshot, want) {
			t.Errorf("%s is frozen as %+v, read as %+v; want its live progress at the run, %+v", name, got, read.Snapshot, want)
		}
	}
	if live := w.cycleProgress(t, "alice", ended).Live; live.Counts != (planningdomain.Counts{Total: 4, Backlog: 1, Unstarted: 1, Completed: 1, Cancelled: 1}) {
		t.Errorf("Ended counts %+v; want a, b, c and d, everyone's", live.Counts)
	}
	for name, id := range map[string]uuid.UUID{"Last day": lastDay, "Today": today, "Draft": draft, "Gone": gone} {
		if got := frozenOf(t, w.pool, id); got != nil {
			t.Errorf("%s is frozen as %+v; want it not frozen", name, got)
		}
	}
	if after := cyclesBesideTheirFreeze(t, w.pool); after != before {
		t.Errorf("the run changed the cycles beside their freeze:\n%s\nwas\n%s", after, before)
	}

	again := map[string]*planningdomain.Frozen{}
	for name, id := range frozen {
		again[name] = frozenOf(t, w.pool, id)
	}
	if n, err := freezeCyclesOn(w.pool, clocktest.At(at.Add(time.Hour))).Execute(pgtest.Soon(t)); err != nil || n != 0 {
		t.Fatalf("a second run froze %d, %v; want none", n, err)
	}
	for name, id := range frozen {
		if got := frozenOf(t, w.pool, id); !reflect.DeepEqual(got, again[name]) {
			t.Errorf("a second run changed %s's freeze to %+v; want %+v", name, got, again[name])
		}
	}
}

// The job's lock wait keeps postgres.LockNotAvailable's chain, which the
// requests' transactions answer 503 for, so River retries it; and the clock
// is read only once before the project's lock, for the candidates' bound
// (M6 P4 spec 2.5, A6): the freeze's own reading waits for the lock.
func TestFreezeJobLockTimeoutPreservesPostgresError(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	ended := w.newCycle(t, "alice", datesBody("Ended", "2044-03-01", "2044-03-10"))
	holder := holding(t, w.pool, "SELECT 1 FROM projects WHERE id=$1 FOR UPDATE", w.web)
	defer release(t, holder)
	cfg := config.Config{Database: config.DatabaseConfig{CommitTimeout: 2 * time.Second}, Issue: config.IssueConfig{WriteLockTimeout: 300 * time.Millisecond}}
	c := &eventCountingClock{base: clocktest.At(time.Date(2044, 4, 2, 0, 0, 0, 0, time.UTC))}
	probe := &requestProbe{TxManager: newIssueJobTx(w.pool, cfg)}
	uc := freezeCyclesWith(w.pool, planningpg.New(w.pool), probe, c)
	result := make(chan error, 1)
	go func() {
		result <- planningriver.NewFreezeCyclesWorker(uc).Work(context.Background(), &river.Job[planningriver.FreezeCyclesArgs]{})
	}()
	requestBlocked(t, w.pool, probe, holder.Conn().PgConn().PID(), "projects")
	select {
	case err := <-result:
		if !postgres.LockNotAvailable(err) || isBusy(err) {
			t.Fatalf("job error=%v; want PostgreSQL's lock error", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the job did not time out")
	}
	if n := c.calls.Load(); n != 1 {
		t.Fatalf("the clock was read %d times; want once, for the candidates, and not for the freeze before its lock", n)
	}
	if got := frozenOf(t, w.pool, ended); got != nil {
		t.Errorf("Ended is frozen as %+v without its project's lock", got)
	}
}

// failingFreezes are the cycles' store whose freeze of cycle fails once it
// wrote.
type failingFreezes struct {
	*planningpg.Store
	cycle uuid.UUID
	err   error
}

func (f failingFreezes) FreezeCycle(ctx context.Context, id uuid.UUID, frozen planningdomain.Frozen) (bool, error) {
	wrote, err := f.Store.FreezeCycle(ctx, id, frozen)
	if err == nil && id == f.cycle {
		return false, f.err
	}
	return wrote, err
}

// A project's failure cannot undo a project committed before it: Web's
// Ended stays frozen, Ops' First and Second, of the failed transaction, are
// not; the retry freezes them at its own moment and leaves Ended's freeze as
// it was (M6 P4 spec 2.5).
func TestFreezeJobPartialCommitAndRetry(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	if w.web.Compare(ops) >= 0 {
		t.Fatalf("Web %s must come before Ops %s among the candidates", w.web, ops)
	}
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+ops.String(), `{"cycle_view":true,"timezone":"UTC"}`, http.StatusOK)
	ended := w.newCycle(t, "alice", datesBody("Ended", "2044-03-01", "2044-03-10"))
	first, second := w.cycleIn(t, ops, datesBody("First", "2044-03-01", "2044-03-05")), w.cycleIn(t, ops, datesBody("Second", "2044-03-06", "2044-03-10"))
	at := time.Date(2044, 4, 2, 1, 2, 3, 0, time.UTC)
	tx := newIssueJobTx(w.pool, config.Config{Database: config.DatabaseConfig{CommitTimeout: 2 * time.Second}, Issue: testIssueConfig})
	sentinel := errors.New("Second's freeze fails")
	n, err := freezeCyclesWith(w.pool, failingFreezes{Store: planningpg.New(w.pool), cycle: second, err: sentinel}, tx, clocktest.At(at)).Execute(pgtest.Soon(t))
	if n != 1 || !errors.Is(err, sentinel) {
		t.Fatalf("partial run = %d, %v; want Ended's 1 and Second's failure", n, err)
	}
	if got := frozenOf(t, w.pool, ended); got == nil || !got.FrozenAt.Equal(at) {
		t.Errorf("Ended is frozen as %+v; want at the failed run's moment", got)
	}
	for name, id := range map[string]uuid.UUID{"First": first, "Second": second} {
		if got := frozenOf(t, w.pool, id); got != nil {
			t.Errorf("%s is frozen as %+v; want its failed transaction rolled back", name, got)
		}
	}
	retry := at.Add(time.Hour)
	if n, err := freezeCyclesOn(w.pool, clocktest.At(retry)).Execute(pgtest.Soon(t)); n != 2 || err != nil {
		t.Fatalf("retry = %d, %v; want First and Second", n, err)
	}
	for name, c := range map[string]struct {
		id uuid.UUID
		at time.Time
	}{"Ended": {ended, at}, "First": {first, retry}, "Second": {second, retry}} {
		if got := frozenOf(t, w.pool, c.id); got == nil || !got.FrozenAt.Equal(c.at) {
			t.Errorf("%s is frozen as %+v; want at %s", name, got, c.at)
		}
	}
}

// assertRuntimeFreeze has the running app, as the runtime role, freeze a
// cycle that has ended: its run, every second here, locks the project,
// reads the cycle's work item and writes its freeze as nerve serves (M6 P4
// spec 2.5). The cycle, current over decades, takes its work item, then its
// end moves to long ago; the owner only reads.
func assertRuntimeFreeze(t *testing.T, roles splitRoles, base string, logs *lockedBuffer) {
	t.Helper()
	contract := apitest.Load(t)
	token := registerAccount(t, contract, base, "freeze-runtime@example.com").AccessToken
	w := issueWorld{contract: contract, base: base, pool: roles.owner, tokens: map[string]string{"alice": token}}
	w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Freeze","slug":"freeze"}`, http.StatusCreated)
	w.web = createdProject(t, contract, base, token, "freeze", "Freeze", "FRZ")
	w.turnCyclesOn(t, w.web)
	ended := w.newCycle(t, "alice", datesBody("Ended", "2020-01-01", "2099-12-31"))
	w.newWorkItemIn(t, w.web, `{"name":"Frozen in","cycle_id":"`+ended.String()+`"}`)
	w.send(t, "alice", http.MethodPatch, "/api/v0/cycles/"+ended.String(), `{"end_date":"2020-01-14"}`, http.StatusOK)
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	deadline := time.NewTimer(15 * time.Second)
	defer deadline.Stop()
	for {
		if f := frozenOf(t, roles.owner, ended); f != nil {
			if f.Total != 1 || f.Unstarted+f.Backlog != 1 || f.FrozenAt.IsZero() {
				t.Errorf("runtime freeze as %s = %+v; want its one work item", roles.serverName, f)
			}
			return
		}
		select {
		case <-tick.C:
		case <-deadline.C:
			t.Fatalf("the cycle was not frozen as %s; logs:\n%s", roles.serverName, logs.String())
		}
	}
}
