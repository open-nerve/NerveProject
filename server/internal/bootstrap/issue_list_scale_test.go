package bootstrap

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

type listScaleSample struct {
	Request    string
	ModuleMS   float64
	SQLMS      map[string]float64
	Statements []listStatement
	Body       string
}

func strconvList(n int) string { return strconv.Itoa(n) }
func scaleRequest(t *testing.T, f listScaleFixture, route moduleRoute, trace *listTrace, s listScaleScenario, q url.Values, id string) listScaleSample {
	t.Helper()
	ctx := context.WithValue(context.Background(), listRequestKey{}, id)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, f.path(s)+"?"+q.Encode(), nil)
	req.Header.Set("Authorization", "Bearer "+f.w.ids["walt"].String())
	rec := httptest.NewRecorder()
	start := time.Now()
	route.router.ServeHTTP(rec, req)
	elapsed := time.Since(start)
	sample := listScaleSample{Request: id, ModuleMS: float64(elapsed) / float64(time.Millisecond), SQLMS: map[string]float64{}, Statements: trace.take(id), Body: rec.Body.String()}
	seen := 0
	var pid uint32
	for _, r := range sample.Statements {
		if r.Error != "" || r.Ended.IsZero() || r.Started.Before(start) || r.Ended.After(start.Add(elapsed)) {
			if dir := os.Getenv("NERVE_LIST_SCALE_EVIDENCE"); dir != "" {
				target := filepath.Join(dir, "failed", strings.ReplaceAll(t.Name(), "/", "-"))
				listEvidence(t, target, id, sample)
				// Replaying the captured typed SQL is diagnostic only. The failed
				// request remains failed; its timeout and measured samples are intact.
				explainListStatements(t, f.w.pool, filepath.Join(target, "plans"), id, sample.Statements)
			}
			t.Fatal("invalid trace boundary", r)
		}
		if pid != 0 && pid != r.PID {
			t.Fatal("read escaped transaction", r)
		}
		pid = r.PID
		class := listStatementClass(listStatementName(r.SQL))
		if class == "list" {
			seen++
			if seen > 1 {
				class = "fallback"
			}
		}
		sample.SQLMS[class] += float64(r.Ended.Sub(r.Started)) / float64(time.Millisecond)
	}
	if rec.Code != 200 {
		t.Fatalf("%s = %d %s", id, rec.Code, rec.Body)
	}
	if seen == 0 {
		t.Fatal("no real list SQL captured", id)
	}
	f.w.contract.CheckResponse(t, req, rec.Result())
	return sample
}

func TestIssueListScale(t *testing.T) {
	if os.Getenv("NERVE_LIST_SCALE") != "1" {
		t.Skip("NERVE_LIST_SCALE=1 required: default skip is not performance validation")
	}
	dir := os.Getenv("NERVE_LIST_SCALE_EVIDENCE")
	if dir == "" {
		dir = t.TempDir()
	}
	t.Log("evidence:", dir)
	for _, n := range []int{10000, 20000, 100000} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			target := filepath.Join(dir, strconv.Itoa(n))
			// The sub-seed has a database of its own: its projects in the
			// scope's would change the statistics the scope's plans are made
			// from, and so the 13 scenarios M4 measured.
			dbs := []listScaleDB{newListScaleDB(t, n, false, target, ""), newListScaleDB(t, n, true, target, "spread-")}
			var scenarios []listScaleScenario
			matched, medians := map[string]int{}, map[string]float64{}
			for _, db := range dbs {
				f, pool, trace := db.f, db.pool, db.trace
				scenarios = append(scenarios, f.scenarios()...)
				for _, s := range f.scenarios() {
					t.Run(s.Name, func(t *testing.T) {
						// A route of its own: a scenario's nine requests stay
						// within the test route's rate limit (burst 100, 600 a
						// minute), which a size's scenarios together would pass.
						route := newIssueRoute(t, pool, postgres.NewTxManager(pool, 2*time.Second))
						want := f.expected(s)
						if len(want) <= s.Tail {
							t.Fatal("empty performance scenario")
						}
						q := maps.Clone(s.Query)
						if q == nil {
							q = url.Values{}
						}
						offset := 0
						if s.Tail > 0 {
							offset = len(want) - s.Tail
							q.Set("cursor", scaleCursor(t, s, want[offset-1]))
						}
						samples := make([]listScaleSample, 0, 6)
						for i := 0; i < 6; i++ {
							samples = append(samples, scaleRequest(t, f, route, trace, s, q, fmt.Sprintf("%s-%d", s.Name, i)))
						}
						listEvidence(t, target, s.Name+"-samples", samples)
						var continuation listScaleContinuation
						for _, sample := range samples {
							continuation = f.checkPage(t, s, sample.Body, want, offset)
						}
						times, sqlTimes := []float64{}, []float64{}
						for _, sample := range samples[1:] {
							times = append(times, sample.ModuleMS)
							sqlTimes = append(sqlTimes, sample.SQLMS["list"])
						}
						sorted, sqlSorted := slices.Sorted(slices.Values(times)), slices.Sorted(slices.Values(sqlTimes))
						matched[s.Name], medians[s.Name] = len(want), sqlSorted[2]
						t.Logf("%s roots=%d physical=%d matched=%d warmup_ms=%.3f samples_ms=%v min/median/max=%.3f/%.3f/%.3f sql_list_ms=%v sql_median=%.3f", s.Name, n, len(f.rows), len(want), samples[0].ModuleMS, times, sorted[0], sorted[2], sorted[4], sqlTimes, sqlSorted[2])
						listEvidence(t, target, s.Name+"-summary", map[string]any{"matched": len(want), "physical": len(f.rows), "warmup_ms": samples[0].ModuleMS, "samples_ms": times, "min_ms": sorted[0], "median_ms": sorted[2], "max_ms": sorted[4], "sql_list_ms": sqlTimes, "sql_list_median_ms": sqlSorted[2]})
						explainListStatements(t, pool, filepath.Join(target, "plans"), s.Name, samples[5].Statements)
						// Actual first cursor -> next page, independent penultimate -> last, last -> empty fallback.
						rows := continuation.Rows
						if continuation.Cursor != "" {
							continuation.Query.Set("cursor", continuation.Cursor)
							next := scaleRequest(t, f, route, trace, s, continuation.Query, s.Name+"-next")
							f.checkRows(t, s, checkListEnvelope(t, next.Body, false, false), rows, 50)
							listEvidence(t, target, s.Name+"-next", next)
						}
						if len(rows) > 1 {
							continuation.Query.Set("cursor", scaleCursor(t, s, rows[len(rows)-2]))
							last := scaleRequest(t, f, route, trace, s, continuation.Query, s.Name+"-last")
							f.checkRows(t, s, checkListEnvelope(t, last.Body, false, false), rows, len(rows)-1)
							listEvidence(t, target, s.Name+"-last", last)
						}
						continuation.Query.Set("cursor", scaleCursor(t, s, rows[len(rows)-1]))
						empty := scaleRequest(t, f, route, trace, s, continuation.Query, s.Name+"-empty")
						listEvidence(t, target, s.Name+"-empty", empty)
						f.checkRows(t, s, checkListEnvelope(t, empty.Body, false, false), rows, len(rows))
						if _, ok := empty.SQLMS["fallback"]; !ok {
							t.Fatal("missing actual fallback SQL")
						}
						explainListStatements(t, pool, filepath.Join(target, "plans"), s.Name+"-empty", empty.Statements)
					})
				}
			}
			listEvidence(t, target, "pairs", scalePairs(t, scenarios, matched, medians))
		})
	}
	t.Run("crowded", func(t *testing.T) { listScaleCrowded(t, dir) })
}

// listScaleDB is a size's seeded database: its fixture, and the runtime
// pool its requests read through with that pool's trace.
type listScaleDB struct {
	f     listScaleFixture
	pool  *pgxpool.Pool
	trace *listTrace
}

// newListScaleDB seeds a database of its own with n roots, the scope's or,
// spread, the sub-seed's, and writes its metadata, distribution and seed to
// target, their names prefixed.
func newListScaleDB(t *testing.T, n int, spread bool, target, prefix string) listScaleDB {
	t.Helper()
	roles := newSplitRoles(t)
	trace := &listTrace{}
	pool := listTracePool(t, roles.serverURL, trace)
	f := seedListScale(t, n, roles, pool, spread)
	listEvidence(t, target, prefix+"metadata", listScaleMetadata(t, roles, pool))
	listEvidence(t, target, prefix+"distribution", f.checkDistribution(t))
	listEvidence(t, target, prefix+"seed", struct {
		Roots       int
		Projects    []listScaleProject
		SeedSeconds float64
		Accounts    any
	}{n, f.projects, f.SeedSeconds, f.w.ids})
	return listScaleDB{f: f, pool: pool, trace: trace}
}

// scalePairs is each workspace scenario's list-SQL median against its
// pair's, the project list's at the same total in the same run, for the pass
// line of M7 P3a spec 2.4, which the ledger judges: no timing is asserted.
// A pair a -run pattern left out is left out.
func scalePairs(t *testing.T, scenarios []listScaleScenario, matched map[string]int, medians map[string]float64) []map[string]any {
	t.Helper()
	pairs := []map[string]any{}
	for _, s := range scenarios {
		workspace, ok := medians[s.Name]
		project, paired := medians[s.Pair]
		if s.Pair == "" || !ok || !paired {
			continue
		}
		ratio := workspace / project
		t.Logf("pair %s/%s matched=%d/%d sql_median=%.3f/%.3f ratio=%.3f", s.Name, s.Pair, matched[s.Name], matched[s.Pair], workspace, project, ratio)
		pairs = append(pairs, map[string]any{"workspace": s.Name, "project": s.Pair, "workspace_matched": matched[s.Name], "project_matched": matched[s.Pair],
			"workspace_sql_median_ms": workspace, "project_sql_median_ms": project, "ratio": ratio})
	}
	return pairs
}
