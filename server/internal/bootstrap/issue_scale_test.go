package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// SP2 measures the intact app and production Runner on six interleaved
// 660,000-row projects. Default skips are not scale validation.
func TestTheDeletionsAtTheScaleTarget(t *testing.T) {
	if os.Getenv("NERVE_MEASURE_SCALE") == "" {
		t.Skip("NERVE_MEASURE_SCALE is not set: explicit scale measurement required")
	}
	dir := os.Getenv("NERVE_SCALE_EVIDENCE")
	if dir == "" {
		dir = filepath.Join("..", "..", "..", ".superpowers", "sdd", "P9-purge", "scale", time.Now().UTC().Format("20060102T150405.000000000"))
	}
	absolute, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	dir = absolute
	t.Log("SP2 evidence:", dir)
	db := pgtest.NewDatabase(t)
	pool := openPool(t, db)
	observeJobStates(t, pool)
	var effective config.Config
	w := newIssueWorldOn(t, db, pool, func(c *config.Config) {
		c.Server.RequestTimeout = 15 * time.Second
		c.Server.WriteTimeout = 60 * time.Second
		c.Database.CommitTimeout = 2 * time.Second
		c.Auth.AccessTokenTTL = time.Hour
		effective = *c
	})
	listEvidence(t, dir, "environment", scaleEnvironment(t, w, effective))
	projects, beta, probe := w.scaleFramework(t)
	ids := make([]uuid.UUID, 0, len(projects))
	for _, p := range projects {
		ids = append(ids, p.ID)
	}
	seeded, analyzed := w.seedAtScale(t, ids)
	listEvidence(t, dir, "fixture", map[string]any{"seed_seconds": seeded.Seconds(), "analyze_seconds": analyzed.Seconds(), "projects": projects, "probe_issue": probe, "seed_context": "15m", "count_context": "3m", "probe_pairs": 20})
	w.checkScaleRows(t, projects, false, dir, "initial")
	marks := w.scaleMarkers(t, projects)
	listEvidence(t, dir, "markers", marks)
	for _, s := range []struct {
		name, path string
		projects   []scaleProject
	}{
		{"project", "/api/v0/projects/" + w.web.String(), projects[:1]},
		{"workspace", "/api/v0/workspaces/beta", projects[1:]},
	} {
		var active int
		if err := pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM river_job WHERE kind='issue.delete_rows' AND state NOT IN ('completed','cancelled','discarded')").Scan(&active); err != nil {
			t.Fatal(err)
		}
		if active != 0 {
			t.Fatalf("baseline has %d active deletion jobs", active)
		}
		baseline := w.scaleSamples(t, probe, s.name+"-baseline", dir)
		listEvidence(t, dir, s.name+"-baseline-summary", scaleSummary(baseline))
		request := w.scaleHTTP(t, "DELETE", s.path, "", "")
		listEvidence(t, dir, s.name+"-delete", request)
		if request.Status != 204 {
			t.Fatalf("DELETE %s=%d %s", s.path, request.Status, request.Body)
		}
		jobs := make([]deletionJob, 0, len(s.projects)+1)
		for _, p := range s.projects {
			jobs = append(jobs, projectDeletionJob(t, pool, p.ID))
		}
		if s.name == "workspace" {
			drafts := parentDeletionJob(t, pool, "workspace_drafts", beta)
			for _, j := range jobs {
				if !j.At.Equal(drafts.At) || j.By != drafts.By {
					t.Fatalf("workspace request audit differs: %+v %+v", j, drafts)
				}
			}
			jobs = append(jobs, drafts)
		}
		jobIDs := make([]int64, 0, len(jobs))
		for _, job := range jobs {
			jobIDs = append(jobIDs, job.ID)
		}
		listEvidence(t, dir, s.name+"-jobs", map[string]any{"ids": jobIDs, "args": jobs})
		w.scaleProcessingSamples(t, probe, s.name, jobs, marks, dir)
		requestEventuallyWithin(t, "SP2 jobs completed", 90*time.Second, func(context.Context) bool {
			done := true
			for _, job := range jobs {
				state := w.scaleSingleRun(t, job, dir)
				if state.State != "completed" {
					done = false
				}
			}
			return done
		})
		for _, job := range jobs {
			w.scaleSingleRun(t, job, dir)
			assertJobRuns(t, pool, job, []int{1}, "completed")
			var raw json.RawMessage
			if err := pool.QueryRow(pgtest.Soon(t), `SELECT row_to_json(j) FROM (SELECT id,kind,args,state,attempt,errors,metadata,queue,max_attempts,created_at,scheduled_at,attempted_at,finalized_at FROM river_job WHERE id=$1) j`, job.ID).Scan(&raw); err != nil {
				t.Fatal(err)
			}
			transitions := jobTransitions(t, pool, job.ID)
			var running, completed time.Time
			for _, tr := range transitions {
				if tr.To == "running" {
					running = tr.At
				}
				if tr.To == "completed" {
					completed = tr.At
				}
			}
			listEvidence(t, dir, "job-"+strconvList(int(job.ID)), map[string]any{"row": raw, "transitions_in_ordinal_order": transitions, "running_seconds": completed.Sub(running).Seconds()})
			t.Logf("SP2 job=%d scope=%s runningCount=1 attempt=1 state=completed errors=0 snoozes=0 running_seconds=%.6f", job.ID, job.Scope, completed.Sub(running).Seconds())
		}
		w.checkScaleRows(t, s.projects, true, dir, s.name+"-final")
	}
	w.scaleHTTP(t, "GET", "/api/v0/issues/"+probe.String(), "", probe.String())
	t.Log("SP2 verified: six projects x 660000; beta 3300000; seven jobs each one running, attempt1, completed, zero errors/snoozes")
}

// Check during processing as well as at the end. Snooze decrements River's
// attempt, so attempt=1 alone is insufficient. Preserve the second real running
// generation before rejecting a small-budget production mutation.
func (w issueWorld) scaleSingleRun(t *testing.T, job deletionJob, dir string) observedJob {
	t.Helper()
	state := readObservedJob(t, w.pool, job)
	transitions := jobTransitions(t, w.pool, job.ID)
	running := 0
	bad := state.Failures != 0 || state.Snoozes != 0 || state.Attempt > 1
	for _, tr := range transitions {
		if tr.To == "running" {
			running++
			if tr.Attempt != 1 {
				bad = true
			}
		}
		if tr.From == "running" && tr.To != "completed" {
			bad = true
		}
	}
	if running > 1 || state.State == "discarded" || state.State == "cancelled" {
		bad = true
	}
	if state.State == "completed" && (running != 1 || state.Attempt != 1) {
		bad = true
	}
	if bad {
		requestEventuallyWithin(t, "second running diagnostic", 3*time.Second, func(context.Context) bool {
			transitions = jobTransitions(t, w.pool, job.ID)
			n := 0
			for _, tr := range transitions {
				if tr.To == "running" {
					n++
				}
			}
			return n > 1 || state.Failures > 0 || state.Attempt > 1
		})
		state = readObservedJob(t, w.pool, job)
		listEvidence(t, dir, "single-run-violation", map[string]any{"id": job.ID, "job": job, "observed": state, "transitions_in_ordinal_order": transitions})
		t.Fatalf("SP2 single-run violation job=%d state=%+v transitions=%+v; want exactly one running/attempt1, zero snoozes/errors", job.ID, state, transitions)
	}
	return state
}
