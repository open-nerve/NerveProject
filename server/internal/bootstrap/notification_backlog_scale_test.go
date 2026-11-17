package bootstrap

import (
	"fmt"
	"testing"
	"time"
	"uuid"

	notificationriver "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// backlogProduction are the settings of server/configs/config.yaml the
// backlog's app takes where testConfig's differ: the pool and the workers
// the fan-outs share with the requests (database.max_conns 10, testConfig's
// 4; jobs.default_workers 10, the same), the lock wait of a work item's
// write and the server's deadlines. The rate limits stay testConfig's
// roomy ones, which no sample reaches, and the access token lives an hour,
// past the run.
func backlogProduction(c *config.Config) {
	c.Database.MaxConns = 10
	c.Jobs = config.JobsConfig{DefaultWorkers: 10, PeriodicWorkers: 2, ShutdownTimeout: 10 * time.Second}
	c.Issue.WriteLockTimeout = 5 * time.Second
	c.Server.ReadHeaderTimeout, c.Server.ReadTimeout, c.Server.WriteTimeout = 5*time.Second, 30*time.Second, 60*time.Second
	c.Server.RequestTimeout = 15 * time.Second
	c.Auth.AccessTokenTTL = time.Hour
}

// backlogUnderWay is how many of the round's fan-outs have completed when
// the samples of the drain begin: some ten seconds of it at River's pace,
// a fetch of default_workers jobs per FetchCooldown (100 ms), past its
// first fetches, so that the samples meet the drain as it runs on.
const backlogUnderWay = 1000

// backlogProgress is the round's fan-outs at a moment of the drain: the
// seconds since the round committed, those completed and those not ended.
type backlogProgress struct {
	Seconds         float64
	Completed, Left int
}

// TestRequestsWhileAFanoutBacklogDrains measures the app's requests while
// the fan-outs one auto-archive round enqueues at once drain, in an app
// with production's pool and workers (backlogProduction), which the
// fan-outs share with the requests. The round is T14's: archiveScaleItems
// done work items of the project Archive, each with bob, a member of it,
// as its subscriber, so that each fan-out writes a row; the use case as
// autoArchiveOn wires it, with the app's subscribers (archivePublishers),
// whose one insert commits the round's fan-outs into the app's default
// queue as the app's own round would. Before it, the queue drained: M4's
// probe pairs (scaleSamples) on alice's work item of Web, another project
// of acme, and on hers of Archive, which the round leaves be, after a
// warm-up of the same pairs on each. Once backlogUnderWay of the round's
// fan-outs have completed, the same pairs on each, while the others remain;
// then the time until every one has completed, and their notifications. A
// request that fails stops the run (scaleSamples); a p95 of the drain above
// both three times its idle one and 50 ms is reported as an error.
func TestRequestsWhileAFanoutBacklogDrains(t *testing.T) {
	dir := measuredScale(t)
	t.Log("fan-out backlog evidence:", dir)
	db := pgtest.NewDatabase(t)
	var effective config.Config
	w := newIssueWorldOn(t, db, openPool(t, db), func(c *config.Config) {
		backlogProduction(c)
		effective = *c
	})
	waitForStartupRuns(t, w.pool)
	environment := scaleEnvironment(t, w, effective)
	environment["base"] = "9a3a388a7a3be57df0c4da76d539ed7a5cead34a"
	environment["seed_version"] = fmt.Sprintf("fanout-backlog-v1: Archive, %d done work items inactive 60 days at the round, bob subscribed to each; a probe in Web and in Archive", archiveScaleItems)
	environment["measurement"] = "M4's probe pairs through the intact app, idle then while one auto-archive round's fan-outs drain on the app's pool and default workers"
	environment["production_overrides"] = "database.max_conns 10, jobs 10/2/10s, issue.write_lock_timeout 5s, server deadlines 5s/30s/60s/15s; rate limits roomy, token 1h"
	listEvidence(t, dir, "fanout-backlog-environment", environment)
	archive := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Archive", "ARC")
	w.makeMember(t, "bob", archive, shared.RoleMember)
	seeded := seedArchiveScale(t, w, []uuid.UUID{archive})
	if _, err := w.pool.Exec(t.Context(), `INSERT INTO issue_subscribers (id, workspace_id, project_id, issue_id, subscriber_id, created_by_id, updated_by_id)
		SELECT uuidv7(), workspace_id, project_id, id, $2, $2, $2 FROM issues WHERE project_id = $1 AND deleted_at IS NULL`, archive, w.ids["bob"]); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(t.Context(), "ANALYZE issue_subscribers"); err != nil {
		t.Fatal(err)
	}
	other, same := w.newWorkItemIn(t, w.web, `{"name":"Probe"}`), w.newWorkItemIn(t, archive, `{"name":"Probe"}`)
	listEvidence(t, dir, "fanout-backlog-fixture", map[string]any{"archive": archive, "items": archiveScaleItems, "subscriber": w.ids["bob"],
		"probe_other": other, "probe_same": same, "seed_seconds": seeded.Seconds()})
	drainDefaultQueue(t, w.pool)
	w.scaleSamples(t, other, "fanout-backlog-warmup-other", dir)
	w.scaleSamples(t, same, "fanout-backlog-warmup-same", dir)
	drainDefaultQueue(t, w.pool)
	idleOther := w.scaleSamples(t, other, "fanout-backlog-idle-other", dir)
	idleSame := w.scaleSamples(t, same, "fanout-backlog-idle-same", dir)
	drainDefaultQueue(t, w.pool)

	// The round's fan-outs are those of its events, not of the probes' writes.
	const round = "kind = $1 AND args->>'type' = 'issue.archived' AND args->>'project_id' = $2"
	count := func(condition string) int {
		var n int
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM river_job WHERE "+round+" AND "+condition, notificationriver.FanoutKind,
			archive.String()).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	left := func() int { return count("finalized_at IS NULL") }
	run := w
	run.pool = freezeScalePool(t, db, nil)
	_, publisher := archivePublishers(t, run.pool)
	turnAutoArchive(t, w.pool, archive, 1)
	start := time.Now()
	archived, err := autoArchiveOn(run, clocktest.At(archiveScaleDay), publisher).Execute(t.Context())
	committed := time.Now()
	turnAutoArchive(t, w.pool, archive, 0)
	if err != nil || archived != archiveScaleItems {
		t.Fatalf("the round archived %d, %v; want %d", archived, err, archiveScaleItems)
	}
	var committedAt time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT clock_timestamp()").Scan(&committedAt); err != nil {
		t.Fatal(err)
	}
	for deadline := committed.Add(2 * time.Minute); count("state = 'completed'") < backlogUnderWay; time.Sleep(100 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("fewer than %d of the round's fan-outs completed within 2 minutes", backlogUnderWay)
		}
	}
	groups := map[string]float64{"under_way_seconds": time.Since(committed).Seconds(), "other_left_before": float64(left())}
	drainOther := w.scaleSamples(t, other, "fanout-backlog-drain-other", dir)
	groups["other_left_after"], groups["same_left_before"] = float64(left()), float64(left())
	drainSame := w.scaleSamples(t, same, "fanout-backlog-drain-same", dir)
	groups["same_left_after"], groups["samples_ended_seconds"] = float64(left()), time.Since(committed).Seconds()
	listEvidence(t, dir, "fanout-backlog-groups", groups)
	if groups["same_left_after"] == 0 {
		t.Fatalf("the backlog drained before the samples ended, %v: they were not taken while it drained", groups)
	}

	var progress []backlogProgress
	for deadline := committed.Add(30 * time.Minute); ; time.Sleep(time.Second) {
		p := backlogProgress{Seconds: time.Since(committed).Seconds(), Left: left(), Completed: count("state = 'completed'")}
		progress = append(progress, p)
		if p.Left == 0 {
			break
		}
		if time.Now().After(deadline) {
			listEvidence(t, dir, "fanout-backlog-progress", progress)
			t.Fatalf("%d of the round's fan-outs not ended within 30 minutes", p.Left)
		}
	}
	observed := time.Since(committed)
	listEvidence(t, dir, "fanout-backlog-progress", progress)
	var jobs, completed, retried, errored, notifications, items, toBob int
	var firstAttempt, lastFinalized time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*), count(*) FILTER (WHERE state = 'completed'), count(*) FILTER (WHERE attempt > 1),
		count(*) FILTER (WHERE errors IS NOT NULL), min(attempted_at), max(finalized_at) FROM river_job WHERE `+round,
		notificationriver.FanoutKind, archive.String()).Scan(&jobs, &completed, &retried, &errored, &firstAttempt, &lastFinalized); err != nil {
		t.Fatal(err)
	}
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*), count(DISTINCT issue_id), count(*) FILTER (WHERE receiver_id = $2) FROM notifications WHERE project_id = $1",
		archive, w.ids["bob"]).Scan(&notifications, &items, &toBob); err != nil {
		t.Fatal(err)
	}
	p95 := func(samples []scaleSample, method string) float64 {
		return scaleSummary(samples)[method].(map[string]any)["p95_ms"].(float64)
	}
	var exceeded []string
	for _, g := range []struct {
		name        string
		idle, drain []scaleSample
	}{{"other project", idleOther, drainOther}, {"same project", idleSame, drainSame}} {
		for _, method := range []string{"GET", "PATCH"} {
			if idle, drain := p95(g.idle, method), p95(g.drain, method); drain > 3*idle && drain > 50 {
				exceeded = append(exceeded, fmt.Sprintf("%s %s p95 %.2f ms, idle %.2f ms", g.name, method, drain, idle))
			}
		}
	}
	summary := map[string]any{"head": environment["head"], "round_seconds": committed.Sub(start).Seconds(), "groups": groups,
		"idle_other": scaleSummary(idleOther), "idle_same": scaleSummary(idleSame), "drain_other": scaleSummary(drainOther), "drain_same": scaleSummary(drainSame),
		"fanouts": jobs, "completed": completed, "retried": retried, "errored": errored, "drain_seconds": lastFinalized.Sub(firstAttempt).Seconds(),
		"complete_seconds": lastFinalized.Sub(committedAt).Seconds(), "observed_seconds": observed.Seconds(), "notifications": notifications,
		"notified_items": items, "notifications_to_bob": toBob, "exceeded": exceeded}
	listEvidence(t, dir, "fanout-backlog-summary", summary)
	t.Logf("fan-out backlog: %d fan-outs, %d completed (%d retried, %d with errors), drained in %.1f s (first attempt to last end), all complete %.1f s after the round committed; %d notifications of %d work items, %d to bob",
		jobs, completed, retried, errored, summary["drain_seconds"], summary["complete_seconds"], notifications, items, toBob)
	for _, g := range []string{"idle_other", "drain_other", "idle_same", "drain_same"} {
		s := summary[g].(map[string]any)
		get, patch := s["GET"].(map[string]any), s["PATCH"].(map[string]any)
		t.Logf("%s: GET p50 %.2f p95 %.2f max %.2f ms; PATCH p50 %.2f p95 %.2f max %.2f ms", g, get["p50_ms"], get["p95_ms"], get["max_ms"],
			patch["p50_ms"], patch["p95_ms"], patch["max_ms"])
	}
	if jobs != archiveScaleItems || completed != archiveScaleItems || notifications != archiveScaleItems || items != archiveScaleItems || toBob != archiveScaleItems {
		t.Errorf("fan-outs %d, completed %d, notifications %d of %d work items, %d to bob; want %d of each", jobs, completed, notifications, items, toBob,
			archiveScaleItems)
	}
	if len(exceeded) > 0 {
		t.Errorf("a p95 of the drain above three times its idle one and 50 ms: %v", exceeded)
	}
}
