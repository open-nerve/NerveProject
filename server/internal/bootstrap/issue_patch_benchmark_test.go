package bootstrap

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// measuredScale answers the evidence directory of a scale measurement,
// NERVE_SCALE_EVIDENCE's, and skips the test unless NERVE_MEASURE_SCALE is
// set: opt-in, as the deletions' and the freeze's scale.
func measuredScale(t *testing.T) string {
	t.Helper()
	if os.Getenv("NERVE_MEASURE_SCALE") == "" {
		t.Skip("NERVE_MEASURE_SCALE is not set: explicit scale measurement required")
	}
	dir, err := filepath.Abs(os.Getenv("NERVE_SCALE_EVIDENCE"))
	if err != nil || os.Getenv("NERVE_SCALE_EVIDENCE") == "" {
		t.Fatalf("NERVE_SCALE_EVIDENCE must name the evidence directory: %v", err)
	}
	return dir
}

// patchBenchmarkSubscribers subscribe to the benchmark's probe besides
// alice, its creator, each a member of Web of the role given: wanda, acme's
// admin, joins it as its admin.
var patchBenchmarkSubscribers = []struct {
	name string
	role shared.Role
}{{"bob", shared.RoleMember}, {"walt", shared.RoleMember}, {"wanda", shared.RoleAdmin}, {"olga", shared.RoleMember}}

// TestThePatchBenchmark measures the PATCH of a work item with subscribers
// by M4's probe (scaleSamples: twenty reads and renamings, one after the
// other) on a world of its own, small: the cost P1b adds to a work item's
// write, its fan-out's InsertTx in the write's transaction, against P1b's
// base, the same file run on the same machine before and after (M7 P1b
// spec R9). The probe is alice's work item in Web, which bob, walt, wanda
// and olga, each of Web, subscribe to through the API, so that each
// renaming notifies four. After a warm-up of twenty pairs, two groups of
// twenty: quiet, each pair once the default queue is drained, the jobs of
// the writes before it completed, and quietPace after the PATCH before it,
// so that a PATCH pays for its own enqueueing alone; and busy, M4's pairs
// back to back, so that a PATCH may also wait for an earlier write's
// fan-out, which holds the probe's row FOR SHARE. It uses only what P1b's
// base has, so that it runs there as it is.
func TestThePatchBenchmark(t *testing.T) {
	dir := measuredScale(t)
	t.Log("patch benchmark evidence:", dir)
	db := pgtest.NewDatabase(t)
	var effective config.Config
	w := newIssueWorldOn(t, db, openPool(t, db), func(c *config.Config) {
		c.Auth.AccessTokenTTL = time.Hour
		effective = *c
	})
	waitForStartupRuns(t, w.pool)
	environment := scaleEnvironment(t, w, effective)
	environment["base"] = "9a3a388a7a3be57df0c4da76d539ed7a5cead34a"
	environment["seed_version"] = "patch-benchmark-v1: alice's probe work item of Web, four subscribers besides her"
	environment["measurement"] = "M4's probe pairs through the intact app: 20 warm-up; 20 quiet, each once the default queue is drained and 2 s after the PATCH before it; 20 busy, back to back"
	listEvidence(t, dir, "patch-benchmark-environment", environment)
	inWorkspaceOf(t, w.pool, w.web, w.ids["olga"], w.ids["alice"], shared.RoleMember)
	probe := w.newWorkItemIn(t, w.web, `{"name":"Probe"}`)
	for _, s := range patchBenchmarkSubscribers {
		if s.name != "bob" {
			w.makeMember(t, s.name, w.web, s.role)
		}
		w.send(t, s.name, http.MethodPost, "/api/v0/issues/"+probe.String()+"/subscribe", "", http.StatusNoContent)
	}
	var subscribers int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM issue_subscribers WHERE issue_id = $1 AND deleted_at IS NULL", probe).
		Scan(&subscribers); err != nil || subscribers != len(patchBenchmarkSubscribers)+1 {
		t.Fatalf("the probe's subscribers: %d, %v; want alice and %d", subscribers, err, len(patchBenchmarkSubscribers))
	}
	warmup := w.scaleSamples(t, probe, "patch-benchmark-warmup", dir)
	quiet, gaps := w.quietScaleSamples(t, probe, "patch-benchmark-quiet", dir, warmup[len(warmup)-1].Ended)
	drainDefaultQueue(t, w.pool)
	busy := w.scaleSamples(t, probe, "patch-benchmark-busy", dir)
	drainDefaultQueue(t, w.pool)
	summary := map[string]any{"head": environment["head"], "quiet": scaleSummary(quiet), "quiet_gaps_ms": freezeScaleQuantiles(gaps),
		"busy": scaleSummary(busy)}
	listEvidence(t, dir, "patch-benchmark-summary", summary)
	patch := func(samples []scaleSample) string {
		s := scaleSummary(samples)["PATCH"].(map[string]any)
		return fmt.Sprintf("p50 %.2f p95 %.2f max %.2f ms", s["p50_ms"], s["p95_ms"], s["max_ms"])
	}
	t.Logf("patch benchmark at %s: quiet PATCH %s, each pair %.0f to %.0f ms after the PATCH before it; busy PATCH %s", environment["head"],
		patch(quiet), slices.Min(gaps), slices.Max(gaps), patch(busy))
}

// quietPace is the time from a quiet pair's PATCH to the next pair, at
// P1b's base and after it alike: longer than the default queue takes to
// drain after it (a fan-out waits at most River's poll, 1 s, to be
// fetched, and runs; its state is written within 250 ms), so that every
// quiet pair follows the same idle time, which by itself slows a request.
const quietPace = 2 * time.Second

// quietScaleSamples are scaleSamples' pairs, each once the default queue
// is drained (drainDefaultQueue) and quietPace after the PATCH before it,
// the first after last; and the time each pair followed the PATCH before
// it, in milliseconds.
func (w issueWorld) quietScaleSamples(t *testing.T, probe uuid.UUID, phase, dir string, last time.Time) ([]scaleSample, []float64) {
	t.Helper()
	samples := make([]scaleSample, 0, 40)
	gaps := make([]float64, 0, 20)
	path := "/api/v0/issues/" + probe.String()
	for i := range 20 {
		drainDefaultQueue(t, w.pool)
		time.Sleep(time.Until(last.Add(quietPace)))
		read := w.scaleHTTP(t, "GET", path, "", probe.String())
		gaps = append(gaps, float64(read.Started.Sub(last))/float64(time.Millisecond))
		name := fmt.Sprintf("%s-%d", phase, i)
		write := w.scaleHTTP(t, "PATCH", path, fmt.Sprintf(`{"name":%q}`, name), probe.String())
		samples, last = append(samples, read, write), write.Ended
		listEvidence(t, dir, phase+"-samples", samples)
		var item struct {
			Name string `json:"name"`
		}
		decodeAnswer(t, write.Body, &item)
		if read.Status != http.StatusOK || write.Status != http.StatusOK || item.Name != name {
			t.Fatalf("probe pair %d: read %d, write %d of %q; want 200, 200 of %q", i, read.Status, write.Status, item.Name, name)
		}
	}
	return samples, gaps
}

// drainDefaultQueue returns once every job of the default queue has ended,
// the work items' writes' among them: before P1b none, after it each
// write's fan-out (M7 P1b spec 2.9). A job done still reads running until
// River's batch completer writes its state.
func drainDefaultQueue(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		var left int
		if err := pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM river_job WHERE queue = 'default' AND finalized_at IS NULL").Scan(&left); err != nil {
			t.Fatal(err)
		}
		if left == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%d jobs of the default queue not ended within 15s", left)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
