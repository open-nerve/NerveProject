package bootstrap

import (
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// This is one production purge job, one attempt resumed across budget-bounded
// runs, after all preparation workers have stopped. Calling Purger directly,
// deleting nothing or reusing a historical completed job cannot satisfy these
// observations.
func TestPurgeScale(t *testing.T) {
	if os.Getenv("NERVE_PURGE_SCALE") != "1" {
		t.Skip("NERVE_PURGE_SCALE=1 required: default skip is not scale validation")
	}
	dir := os.Getenv("NERVE_PURGE_SCALE_EVIDENCE")
	if dir == "" {
		dir = t.TempDir()
	}
	var err error
	dir, err = filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("purge scale evidence:", dir)
	roles := newSplitRoles(t)
	observeJobStates(t, roles.owner)
	f := preparePurgeScale(t, roles, dir)
	before := make([]map[string]int64, 0, len(f.Scopes))
	for i, scope := range f.Scopes {
		counts := purgeScopeCounts(t, roles.owner, scope, i < 2)
		for table, n := range counts {
			if n == 0 {
				t.Fatalf("empty %d/%s fixture would make purge validation vacuous", i, table)
			}
		}
		before = append(before, counts)
	}
	retained := purgeRetentionSnapshot(t, roles.owner, f.Scopes[1:])
	want := maps.Clone(before[0])
	for _, table := range []string{"file_assets", "workspace_user_properties", "workspace_member_invites", "workspace_members", "workspaces"} {
		want[table] = 0
	}
	// The runs' account proves no cascade did a table's work only if nothing
	// outside the old scope is purgeable. Only aged rows are older than a day,
	// and the cutoff falls minutes after anchor-60d.
	purgeable := purgeableRows(t, roles.owner, f.Anchor.Add(-60*24*time.Hour), want)
	listEvidence(t, dir, "before", map[string]any{"counts": before, "retention": retained, "purgeable": purgeable})
	if !reflect.DeepEqual(purgeable, want) {
		t.Fatalf("purgeable rows %v; want only the old scope %v", purgeable, want)
	}
	previousID, watermark := purgeScaleBoundary(t, roles.owner)
	trace := &purgeScaleTrace{}
	var logs lockedBuffer
	cfg := testConfig(t, roles.serverURL, false)
	constructing := time.Now()
	a, err := newAppWithPoolFactory(t.Context(), cfg, slog.New(slog.NewTextHandler(&logs, nil)), migrations.FS(), testWebUI, trace.pool)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.close)
	var role, jit, mode string
	var ownsTable bool
	if err := a.pool.QueryRow(pgtest.Soon(t), `SELECT current_user,current_user=pg_get_userbyid(relowner),current_setting('jit'),current_setting('plan_cache_mode') FROM pg_class WHERE oid='issues'::regclass`).Scan(&role, &ownsTable, &jit, &mode); err != nil || role != roles.serverName || ownsTable || jit != "off" || mode != "force_custom_plan" {
		t.Fatalf("purge runtime role=%s owner=%t jit=%s mode=%s err=%v", role, ownsTable, jit, mode, err)
	}
	environment := listScaleMetadata(t, roles, a.pool)
	environment["seed_version"] = "purge-scale-v1: 20k roots/660k core rows plus sparse API rows; 59d/live controls"
	environment["measurement"] = "production Runner and runtime app pool; owner only prepares and observes"
	environment["cleanup"], environment["jobs"], environment["issue"] = cfg.Cleanup, cfg.Jobs, cfg.Issue
	environment["run_budget"] = cfg.Files.CleanupBudget.String()
	listEvidence(t, dir, "environment", environment)
	started := time.Now()
	listEvidence(t, dir, "boundary", map[string]any{"previous_purge_id": previousID, "database_watermark": watermark, "app_start": started, "app_construction_seconds": started.Sub(constructing).Seconds(), "runtime_role": role})
	runBuiltApp(t, a)
	job := awaitPurgeScaleJob(t, roles.owner, previousID, dir)
	ended := time.Now()
	listEvidence(t, dir, "job", job)
	listEvidence(t, dir, "sql", trace.snapshot())
	if job.ID <= previousID || job.Kind != "cleanup.purge_deleted" || job.Queue != "periodic" || job.Attempt != 1 || job.Attempted == nil || job.Finalized == nil || job.Created.Before(watermark) || job.Attempted.Before(watermark) || job.Finalized.Before(*job.Attempted) {
		t.Fatalf("purge completion is not bound to this run: %+v previous=%d watermark=%s", job, previousID, watermark)
	}
	assertJobRuns(t, roles.owner, deletionJob{ID: job.ID}, slices.Repeat([]int{1}, job.Snoozes+1), "completed")
	listEvidence(t, dir, "transitions", jobTransitions(t, roles.owner, job.ID))
	runs := purgeScaleRuns(t, roles.owner, job.ID)
	listEvidence(t, dir, "runs", runs)
	after := make([]map[string]int64, 0, len(f.Scopes))
	for i, scope := range f.Scopes {
		after = append(after, purgeScopeCounts(t, roles.owner, scope, i < 2))
	}
	retainedAfter := purgeRetentionSnapshot(t, roles.owner, f.Scopes[1:])
	listEvidence(t, dir, "after", map[string]any{"counts": after, "retention": retainedAfter})
	for table, n := range after[0] {
		if n != 0 {
			t.Errorf("purge left old %s rows=%d (before %d)", table, n, before[0][table])
		}
	}
	if !reflect.DeepEqual(retained, retainedAfter) || !reflect.DeepEqual(before[1:], after[1:]) {
		t.Error("purge changed 59-day or live rows; complete before/after snapshots retained")
	}
	cutoff, cut, sql := purgeScaleSQLSummary(t, trace.snapshot(), role, started, ended, want)
	account := purgeScaleRunSummary(t, runs, cutoff, cut, want)
	if strings.Contains(logs.String(), "permission denied") {
		t.Error("purge runtime permission failure: " + logs.String())
	}
	listEvidence(t, dir, "summary", map[string]any{
		"job_id": job.ID, "attempt": job.Attempt, "queue": job.Queue, "state": job.State,
		"errors": job.Errors, "snoozes": job.Snoozes, "runtime_role": role, "runs": account,
		"queue_seconds": runs[0].Started.Sub(job.Created).Seconds(), "sql": sql,
		"first_run_to_completion_seconds": runs[len(runs)-1].Ended.Sub(runs[0].Started).Seconds(), "app_start_to_observed_completion_seconds": ended.Sub(started).Seconds(),
		"seed_seconds": f.Seed.Seconds(), "analyze_seconds": f.Analyze.Seconds(), "analyze_after_aging_seconds": f.AgedAnalyze.Seconds(),
		"measurement": "real Runner and runtime pool, budget-bounded runs resumed by snooze; includes pgx trace and River transition observer, no row DELETE trigger; no production time SLA",
	})
	if !t.Failed() {
		t.Logf("purge scale: job=%d attempt 1 in %d runs, runtime=%s, running_seconds=%.6f; old scope removed by its own statements; 59d/live rows unchanged", job.ID, len(runs), role, account["running_seconds"])
	}
}
