package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	notificationpg "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/postgres"
	notificationriver "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/river"
	notificationapp "github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fanoutScaleSubscribers are the subscribers of the work item of the
// fan-out at scale (M7 design 9, P1b spec 4).
const fanoutScaleSubscribers = 1000

// fanoutScaleHold bounds that fan-out's hold of its work item's row (P1b
// spec 4), and so the wait it adds to a write of the work item, which
// waits for the row at most issue.write_lock_timeout, 5 s.
const fanoutScaleHold = time.Second

// TestAFanoutToAThousandSubscribers measures one fan-out of a work item
// with fanoutScaleSubscribers subscribers (M7 P1b spec 4): its hold of the
// work item's row, from ShareIssue's answer to its transaction's end, at
// most fanoutScaleHold; a PATCH of the work item meanwhile, which waits for
// it; its rows, one to each subscriber; and its run again, which writes
// none. The subscribers are written directly, each acme's and Web's member
// subscribed to alice's work item Scale. alice renames it through the API;
// its fan-out, held (holdFanouts), runs through the module's worker as
// newApp wires it (notificationDeps), on a pool with the app's planner
// settings, timed (timedShareIssue, timedFanoutTx), as the freeze's scale
// times its run. She renames it again, and while that fan-out holds the
// row, a third renaming of hers waits for it (UpdateIssue): the test lets
// the fan-out go on once it has seen the wait, and times both, the hold
// without the test's own. Last the app's River runs the three fan-outs
// held: the first two again, which write none, and the third.
func TestAFanoutToAThousandSubscribers(t *testing.T) {
	dir := measuredScale(t)
	t.Log("fan-out scale evidence:", dir)
	db := pgtest.NewDatabase(t)
	pool := openPool(t, db)
	holdFanouts(t, pool)
	var effective config.Config
	w := newIssueWorldOn(t, db, pool, func(c *config.Config) {
		c.Auth.AccessTokenTTL = time.Hour
		effective = *c
	})
	waitForStartupRuns(t, w.pool)
	environment := scaleEnvironment(t, w, effective)
	environment["base"] = "9a3a388a7a3be57df0c4da76d539ed7a5cead34a"
	environment["seed_version"] = fmt.Sprintf("fanout-v1: one work item of Web, %d subscribers besides alice, each acme's and Web's member", fanoutScaleSubscribers)
	environment["measurement"] = "notification.fanout's worker as newApp wires it, run by the test on a pool with the app's planner settings, timed"
	listEvidence(t, dir, "fanout-environment", environment)
	item := w.newWorkItemIn(t, w.web, `{"name":"Scale"}`)
	if n := releaseFanouts(t, w.pool); n != 1 {
		t.Fatalf("released %d fan-outs; want the creation's", n)
	}
	waitForFanouts(t, w.pool)
	subscribers := seedFanoutSubscribers(t, w, item)
	appPool := freezeScalePool(t, db, nil)
	path := "/api/v0/issues/" + item.String()

	w.send(t, "alice", http.MethodPatch, path, `{"name":"Scale 1"}`, http.StatusOK)
	first := fanoutAtScale(appPool, effective, heldUpdates(t, w.pool, item, 1)[0], nil)
	listEvidence(t, dir, "fanout-first-run", first)
	if first.Err != "" || first.HeldMS > float64(fanoutScaleHold/time.Millisecond) {
		t.Fatalf("the fan-out to %d: %+v; want it to hold the work item's row at most %v", fanoutScaleSubscribers, first, fanoutScaleHold)
	}
	fanoutScaleRows(t, w.pool, first.EventID, subscribers)

	w.send(t, "alice", http.MethodPatch, path, `{"name":"Scale 2"}`, http.StatusOK)
	gate := &fanoutGate{locked: make(chan struct{}, 1), release: make(chan struct{})}
	letGo := sync.OnceFunc(func() { close(gate.release) })
	t.Cleanup(letGo)
	ran := make(chan fanoutScaleRun, 1)
	held := heldUpdates(t, w.pool, item, 2)[1]
	go func() { ran <- fanoutAtScale(appPool, effective, held, gate) }()
	receiveWithin(t, gate.locked, 10*time.Second, "the second fan-out's lock of the work item")
	req := newRequest(t, http.MethodPatch, w.base+path, w.tokens["alice"], []byte(`{"name":"Scale 3"}`))
	w.contract.CheckRequest(t, req)
	sent := time.Now()
	answered := sendInBackground(req)
	pgtest.WaitForStatementLockWaitOn(t, w.pool, "issues", 5*time.Second, "UpdateIssue")
	waiting := time.Now()
	if len(answered) != 0 {
		t.Fatal("the PATCH answered while the fan-out held the work item's row")
	}
	letGo()
	got := receiveWithin(t, answered, 10*time.Second, "the PATCH waiting for the fan-out")
	at := time.Now()
	second := receiveWithin(t, ran, 30*time.Second, "the second fan-out")
	if got.err != nil {
		t.Fatal(got.err)
	}
	w.contract.CheckResponse(t, req, got.res)
	patch := fanoutScalePatch{Status: got.res.StatusCode, ArrivedMS: milliseconds(waiting.Sub(sent)), PatchMS: milliseconds(at.Sub(sent)),
		AfterReleaseMS: milliseconds(at.Sub(second.Released))}
	listEvidence(t, dir, "fanout-patch-during-hold", map[string]any{"fanout": second, "patch": patch})
	if second.Err != "" || second.HeldMS > float64(fanoutScaleHold/time.Millisecond) {
		t.Fatalf("the second fan-out: %+v; want it to hold the work item's row at most %v", second, fanoutScaleHold)
	}
	if patch.Status != http.StatusOK || patch.AfterReleaseMS > float64(fanoutScaleHold/time.Millisecond) {
		t.Fatalf("the PATCH waiting for the fan-out: %+v, %s; want 200 within %v of the fan-out's going on", patch, got.body, fanoutScaleHold)
	}
	fanoutScaleRows(t, w.pool, second.EventID, subscribers)

	if n := releaseFanouts(t, w.pool); n != 3 {
		t.Fatalf("released %d fan-outs; want the three renamings'", n)
	}
	waitForFanouts(t, w.pool)
	runs := fanoutRunsOf(t, w.pool, "issue.updated", item)
	if !reflect.DeepEqual(runs, []fanoutRun{{"completed", 1, 0}, {"completed", 1, 0}, {"completed", 1, 0}}) {
		t.Fatalf("the renamings' fan-outs run by River: %+v; want each completed in its first attempt", runs)
	}
	updates := heldUpdates(t, w.pool, item, 3)
	for _, u := range updates {
		fanoutScaleRows(t, w.pool, u.Args.EventID, subscribers)
	}
	var all int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM notifications WHERE issue_id = $1", item).Scan(&all); err != nil || all != 3*fanoutScaleSubscribers {
		t.Fatalf("the work item's notifications: %d, %v; want the three renamings' %d each", all, err, fanoutScaleSubscribers)
	}
	listEvidence(t, dir, "fanout-rerun", map[string]any{"runs": runs, "notifications": all, "jobs": updates})
	t.Logf("fan-out scale: %d subscribers; first run %.1f ms, the row held %.1f ms of it; second run's hold %.1f ms after the PATCH waited for it; "+
		"PATCH %.1f ms, at the lock %.1f ms after it was sent, answered %.1f ms after the fan-out went on; River's runs again wrote none",
		fanoutScaleSubscribers, first.TotalMS, first.HeldMS, second.HeldMS, patch.PatchMS, patch.ArrivedMS, patch.AfterReleaseMS)
}

// seedFanoutSubscribers writes fanoutScaleSubscribers accounts, each a
// member of acme and of Web and subscribed to item, then analyzes the
// tables, and answers them in id order.
func seedFanoutSubscribers(t *testing.T, w issueWorld, item uuid.UUID) []uuid.UUID {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	member := int(shared.RoleMember)
	for _, s := range []struct {
		sql  string
		args []any
	}{
		{`CREATE TEMP TABLE fanout_scale ON COMMIT DROP AS SELECT uuidv7() AS id, n FROM generate_series(1, $1::int) n`, []any{fanoutScaleSubscribers}},
		{`INSERT INTO users (id, email, password, display_name) SELECT id, 'subscriber' || n || '@example.com', 'x', 'Subscriber ' || n FROM fanout_scale`, nil},
		{`INSERT INTO workspace_members (id, workspace_id, member_id, role) SELECT uuidv7(), $1, id, $2 FROM fanout_scale`, []any{w.acme, member}},
		{`INSERT INTO project_members (id, workspace_id, project_id, member_id, role) SELECT uuidv7(), $1, $2, id, $3 FROM fanout_scale`,
			[]any{w.acme, w.web, member}},
		{`INSERT INTO issue_subscribers (id, workspace_id, project_id, issue_id, subscriber_id, created_by_id, updated_by_id)
			SELECT uuidv7(), $1, $2, $3, id, id, id FROM fanout_scale`, []any{w.acme, w.web, item}},
	} {
		if _, err = tx.Exec(ctx, s.sql, s.args...); err != nil {
			t.Fatalf("%v: %s", err, s.sql)
		}
	}
	rows, err := tx.Query(ctx, "SELECT id FROM fanout_scale ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil || len(ids) != fanoutScaleSubscribers {
		t.Fatalf("subscribers %d, %v; want %d", len(ids), err, fanoutScaleSubscribers)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = w.pool.Exec(ctx, "ANALYZE users, workspace_members, project_members, issue_subscribers"); err != nil {
		t.Fatal(err)
	}
	return ids
}

// heldFanout is a fan-out job as River holds it: its id and arguments.
type heldFanout struct {
	ID   int64
	Args notificationriver.FanoutArgs
}

// heldUpdates reads, in order, the fan-outs of the renamings of item,
// failing the test unless there are want.
func heldUpdates(t *testing.T, pool *pgxpool.Pool, item uuid.UUID, want int) []heldFanout {
	t.Helper()
	rows, err := pool.Query(pgtest.Soon(t), `SELECT id, args FROM river_job WHERE kind = $1 AND args->>'type' = 'issue.updated' AND args->>'entity_id' = $2
		ORDER BY id`, notificationriver.FanoutKind, item.String())
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var jobs []heldFanout
	for rows.Next() {
		var j heldFanout
		var raw []byte
		if err := rows.Scan(&j.ID, &raw); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &j.Args); err != nil {
			t.Fatal(err)
		}
		jobs = append(jobs, j)
	}
	if err := rows.Err(); err != nil || len(jobs) != want {
		t.Fatalf("the renamings' fan-outs: %d, %v; want %d", len(jobs), err, want)
	}
	return jobs
}

// fanoutScaleRows checks the notifications of the renaming event: one to
// each of subscribers, subscribed, of its one change, live.
func fanoutScaleRows(t *testing.T, pool *pgxpool.Pool, event uuid.UUID, subscribers []uuid.UUID) {
	t.Helper()
	var all, right int
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT count(*), count(*) FILTER (WHERE receiver_id = ANY ($2) AND sender = 'subscribed' AND change_seq = 0
			AND deleted_at IS NULL) FROM notifications WHERE event_id = $1`, event, subscribers).Scan(&all, &right); err != nil {
		t.Fatal(err)
	}
	if all != len(subscribers) || right != all {
		t.Fatalf("the notifications of %s: %d, %d of them to a subscriber; want one to each of %d", event, all, right, len(subscribers))
	}
}

// fanoutScaleRun is a timed run of a fan-out (fanoutAtScale): its
// transaction begun, its work item's row locked, the moment ShareIssue
// answered, let go on by the test, at once without a gate, and ended,
// committed or rolled back. HeldMS is the row's hold but the gate's,
// TotalMS the run's but the gate's.
type fanoutScaleRun struct {
	JobID                             int64
	EventID                           uuid.UUID
	Err                               string
	Begun, Locked, Released, Ended    time.Time
	LockedMS, GateMS, HeldMS, TotalMS float64
}

// fanoutGate stops a fan-out once it has locked its work item's row: it
// sends on locked, then waits for release to close.
type fanoutGate struct{ locked, release chan struct{} }

type timedShareIssue struct {
	notificationapp.IssueRowLock
	run  *fanoutScaleRun
	gate *fanoutGate
}

func (l timedShareIssue) ShareIssue(ctx context.Context, id uuid.UUID) (notificationapp.LockedIssue, bool, error) {
	i, found, err := l.IssueRowLock.ShareIssue(ctx, id)
	l.run.Locked = time.Now()
	if l.gate != nil {
		l.gate.locked <- struct{}{}
		select {
		case <-l.gate.release:
		case <-ctx.Done():
		}
	}
	l.run.Released = time.Now()
	return i, found, err
}

type timedFanoutTx struct {
	shared.TxManager
	run *fanoutScaleRun
}

func (x timedFanoutTx) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	x.run.Begun = time.Now()
	err := x.TxManager.WithinTx(ctx, fn)
	x.run.Ended = time.Now()
	return err
}

// fanoutAtScale runs the held fan-out job once through the module's worker
// as newApp wires it over pool with cfg's job settings (notificationDeps),
// within the worker's timeout, its lock and transaction timed and, with a
// gate, stopped once its row is locked. It reports, rather than fails the
// test, so that it may run beside it.
func fanoutAtScale(pool *pgxpool.Pool, cfg config.Config, job heldFanout, gate *fanoutGate) fanoutScaleRun {
	d := notificationDeps(pool, cfg)
	r := &fanoutScaleRun{JobID: job.ID, EventID: job.Args.EventID}
	uc := notificationapp.NewFanout(notificationapp.FanoutDeps{Lock: timedShareIssue{d.Issues, r, gate}, Issues: d.Issues, Projects: d.Projects,
		Readers: d.Readers, Rows: notificationpg.New(pool), Tx: timedFanoutTx{d.JobTx, r}, Logger: d.Logger})
	worker := notificationriver.NewFanoutWorker(uc, d.Logger)
	run := &river.Job[notificationriver.FanoutArgs]{JobRow: &rivertype.JobRow{ID: job.ID, Attempt: 1, MaxAttempts: notificationriver.FanoutMaxAttempts},
		Args: job.Args}
	ctx, cancel := context.WithTimeout(context.Background(), worker.Timeout(run))
	defer cancel()
	start := time.Now()
	if err := worker.Work(ctx, run); err != nil {
		r.Err = err.Error()
	}
	done := time.Now()
	r.LockedMS, r.GateMS = milliseconds(r.Locked.Sub(r.Begun)), milliseconds(r.Released.Sub(r.Locked))
	r.HeldMS, r.TotalMS = milliseconds(r.Ended.Sub(r.Released)), milliseconds(done.Sub(start))-r.GateMS
	return *r
}

// fanoutScalePatch is the PATCH that waits for a fan-out: its status, when
// it was seen waiting for the work item's row after it was sent, when it
// was answered, and that after the fan-out went on.
type fanoutScalePatch struct {
	Status                             int
	ArrivedMS, PatchMS, AfterReleaseMS float64
}

func milliseconds(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
