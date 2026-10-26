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
	"testing"
	"time"

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
	path := "/api/v0/projects/" + f.projects[max(s.Project, 0)].ID.String() + "/issues"
	if s.Project < 0 {
		path = memberIssuesPath(f.w, "bob")
	}
	ctx := context.WithValue(context.Background(), listRequestKey{}, id)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, path+"?"+q.Encode(), nil)
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
	for _, n := range []int{10000, 100000} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			roles := newSplitRoles(t)
			trace := &listTrace{}
			pool := listTracePool(t, roles.serverURL, trace)
			f := seedListScale(t, n, roles, pool)
			target := filepath.Join(dir, strconv.Itoa(n))
			listEvidence(t, target, "metadata", listScaleMetadata(t, roles, pool))
			listEvidence(t, target, "distribution", f.checkDistribution(t))
			listEvidence(t, target, "seed", struct {
				Roots       int
				Projects    []listScaleProject
				SeedSeconds float64
				Accounts    any
			}{n, f.projects, f.SeedSeconds, f.w.ids})
			route := newIssueRoute(t, pool, postgres.NewTxManager(pool, 2*time.Second))
			for _, s := range f.scenarios() {
				t.Run(s.Name, func(t *testing.T) {
					want := f.expected(s)
					if len(want) == 0 {
						t.Fatal("empty performance scenario")
					}
					q := maps.Clone(s.Query)
					if q == nil {
						q = url.Values{}
					}
					samples := make([]listScaleSample, 0, 6)
					for i := 0; i < 6; i++ {
						samples = append(samples, scaleRequest(t, f, route, trace, s, q, fmt.Sprintf("%s-%d", s.Name, i)))
					}
					listEvidence(t, target, s.Name+"-samples", samples)
					var continuation listScaleContinuation
					for _, sample := range samples {
						continuation = f.checkPage(t, s, sample.Body, want)
					}
					times := []float64{}
					for _, sample := range samples[1:] {
						times = append(times, sample.ModuleMS)
					}
					sorted := slices.Sorted(slices.Values(times))
					t.Logf("%s roots=%d physical=%d matched=%d warmup_ms=%.3f samples_ms=%v min/median/max=%.3f/%.3f/%.3f", s.Name, n, len(f.rows), len(want), samples[0].ModuleMS, times, sorted[0], sorted[2], sorted[4])
					listEvidence(t, target, s.Name+"-summary", map[string]any{"matched": len(want), "physical": len(f.rows), "warmup_ms": samples[0].ModuleMS, "samples_ms": times, "min_ms": sorted[0], "median_ms": sorted[2], "max_ms": sorted[4]})
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
		})
	}
}
