package postgresadapter_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

type searchPerfScenario struct {
	Name, Term string
	Rare, Own  bool
	Project    int // -1 means both visible projects.
}

type searchPerfSample struct {
	SnapshotMS float64
	SQLMS      float64
	Statement  searchPerfStatement
	Rows       []domain.Issue
}

func (f searchPerfFixture) input(t *testing.T, s searchPerfScenario) app.SearchRead {
	t.Helper()
	raw := map[string][]string{"q": {s.Term}, "limit": {"50"}}
	if s.Rare {
		raw["start_date"] = []string{"2026-06-15"}
	}
	q, err := domain.ParseSearchQuery(raw)
	if err != nil {
		t.Fatal(err)
	}
	in := app.SearchRead{ActorID: f.Actor, Query: q}
	for i, p := range f.Projects {
		if s.Project < 0 || s.Project == i {
			in.Projects = append(in.Projects, app.VisibleProject{ID: p.id, OwnOnly: s.Own || i == 1})
		}
	}
	if q.Identifier != "" {
		in.ExactProjectID = &f.Projects[0].id
	}
	return in
}

// Expectations enumerate immutable seed metadata, never returned Search rows.
// For title queries we deliberately place >50 exact-title leaders above all
// other matches. This avoids reimplementing locale-dependent pg_trgm in Go.
// Chinese zero-score ties also put those leaders first by their timestamps.
func (f searchPerfFixture) expected(t *testing.T, s searchPerfScenario, in app.SearchRead) (int, []searchPerfRow) {
	t.Helper()
	var matches []searchPerfRow
	for _, r := range f.Rows {
		if s.Project >= 0 && s.Project != r.Project || r.Deleted || r.Archived || s.Rare && !r.Rare || (s.Own || r.Project == 1) && r.Author != f.Actor {
			continue
		}
		identifier := in.Query.IdentifierNumber != nil && r.Project == 0 && r.Sequence == *in.Query.IdentifierNumber
		numeric := in.Query.SequenceNumber != nil && r.Sequence == *in.Query.SequenceNumber
		if s.Term == "" || identifier || numeric || strings.Contains(r.Name, s.Term) {
			matches = append(matches, r)
		}
	}
	count := len(matches)
	rank := func(r searchPerfRow) int {
		if in.Query.IdentifierNumber != nil && r.Project == 0 && r.Sequence == *in.Query.IdentifierNumber {
			return 0
		}
		if in.Query.SequenceNumber != nil && r.Sequence == *in.Query.SequenceNumber {
			return 1
		}
		return 2
	}
	slices.SortFunc(matches, func(a, b searchPerfRow) int {
		if s.Term == "" {
			if d := b.Updated.Compare(a.Updated); d != 0 {
				return d
			}
		} else {
			if d := rank(a) - rank(b); d != 0 {
				return d
			}
			// This is a controlled leader oracle, not a general similarity
			// implementation. All returned title leaders have the exact term.
			if (a.Name == s.Term) != (b.Name == s.Term) {
				if a.Name == s.Term {
					return -1
				}
				return 1
			}
			if d := b.Created.Compare(a.Created); d != 0 {
				return d
			}
		}
		return strings.Compare(b.ID.String(), a.ID.String())
	})
	if len(matches) > 50 {
		matches = matches[:50]
	}
	if s.Term != "" && in.Query.IdentifierNumber == nil && in.Query.SequenceNumber == nil {
		if len(matches) != 50 {
			t.Fatal("insufficient controlled title leaders", s.Name, len(matches))
		}
		for _, row := range matches {
			if row.Name != s.Term {
				t.Fatal("oracle would require general similarity ordering", s.Name, row.Name)
			}
		}
	}
	return count, matches
}

func searchPerfCheck(t *testing.T, got []domain.Issue, want []searchPerfRow) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("topN size=%d want=%d", len(got), len(want))
	}
	for i, row := range want {
		links := int64(0)
		if row.ExactTitle {
			links = 1
		}
		if got[i].ID != row.ID || got[i].Name != row.Name || got[i].LinkCount != links {
			t.Fatalf("topN[%d]=%s/%s/links%d want=%s/%s/links%d", i, got[i].ID, got[i].Name, got[i].LinkCount, row.ID, row.Name, links)
		}
	}
}

func TestSearch100KPerformance(t *testing.T) {
	if os.Getenv("NERVE_SEARCH_PERF") != "1" {
		t.Skip("NERVE_SEARCH_PERF=1 required: default skip is not performance validation")
	}
	dir := os.Getenv("NERVE_SEARCH_PERF_EVIDENCE")
	if dir == "" {
		dir = t.TempDir()
	}
	t.Log("evidence:", dir)
	trace := &searchPerfTrace{}
	owner, pool := searchPerfPools(t, trace)
	started := time.Now()
	f := seedSearchPerformance(t, owner)
	searchPerfWrite(t, dir, "seed", map[string]any{"version": "search-100k-v1", "seed": searchPerfSeed, "seconds": time.Since(started).Seconds(), "physical_issues": len(f.Rows), "distribution": f.distribution(t, owner)})
	searchPerfWrite(t, dir, "metadata", searchPerfMetadata(t, owner, pool))
	scenarios := []searchPerfScenario{}
	for i, term := range []string{"lo", "login", "登录", "修复登录"} {
		for _, rare := range []bool{false, true} {
			scenarios = append(scenarios, searchPerfScenario{Name: fmt.Sprintf("term%d-rare%t", i, rare), Term: term, Rare: rare, Project: -1})
		}
	}
	scenarios = append(scenarios,
		searchPerfScenario{Name: "empty", Project: -1},
		searchPerfScenario{Name: "numeric", Term: "31415", Project: -1},
		searchPerfScenario{Name: "identifier", Term: "PERFA-31415", Project: -1},
		searchPerfScenario{Name: "own-both", Term: "lo", Own: true, Project: -1},
		searchPerfScenario{Name: "single-project", Term: "login", Project: 0})
	store := postgresadapter.New(pool)
	read := postgres.NewTxManager(pool, 2*time.Second, postgres.ReadOnlySnapshot())
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			in := f.input(t, scenario)
			count, want := f.expected(t, scenario, in)
			searchPerfWrite(t, dir, scenario.Name+"-expected", map[string]any{"candidates": count, "topN": want})
			samples := make([]searchPerfSample, 0, 6)
			for n := 0; n < 6; n++ {
				request := fmt.Sprintf("%s-%d", scenario.Name, n)
				ctx := context.WithValue(context.Background(), searchPerfRequestKey{}, request)
				var got []domain.Issue
				var pid uint32
				start := time.Now()
				err := read.WithinTx(ctx, func(ctx context.Context) error {
					tx, ok := postgres.Tx(ctx)
					if !ok {
						t.Fatal("missing production snapshot")
					}
					pid = tx.Conn().PgConn().PID()
					// The trace keeps the original production Search SQL intact.
					var err error
					got, err = store.Search(ctx, in)
					return err
				})
				elapsed := time.Since(start)
				if err != nil {
					t.Fatal(err)
				}
				statement := trace.take(t, request)
				if statement.PID != pid || statement.Started.Before(start) || statement.Ended.After(start.Add(elapsed)) || statement.Error != "" {
					t.Fatal("invalid actual query trace boundary", statement)
				}
				samples = append(samples, searchPerfSample{SnapshotMS: float64(elapsed) / float64(time.Millisecond), SQLMS: float64(statement.Ended.Sub(statement.Started)) / float64(time.Millisecond), Statement: statement, Rows: got})
				searchPerfWrite(t, dir, scenario.Name+"-samples", samples)
				searchPerfCheck(t, got, want)
			}
			searchPerfWrite(t, dir, scenario.Name+"-samples", samples)
			var module, sql []float64
			for _, sample := range samples[1:] {
				module, sql = append(module, sample.SnapshotMS), append(sql, sample.SQLMS)
			}
			slices.Sort(module)
			slices.Sort(sql)
			statement := samples[5].Statement
			actualCount := searchPerfDiagnosticCount(t, pool, statement)
			if actualCount != count {
				t.Fatalf("diagnostic candidate count=%d want independent=%d", actualCount, count)
			}
			searchPerfWrite(t, dir, scenario.Name+"-summary", map[string]any{"scenario": scenario, "expected_candidates": count, "diagnostic_actual_candidates": actualCount, "expected_topN": want, "first_observed": samples[0], "warm_snapshot_min_median_max_ms": []float64{module[0], module[2], module[4]}, "warm_sql_min_median_max_ms": []float64{sql[0], sql[2], sql[4]}})
			t.Logf("q=%q rare=%t candidates=%d topN=%d first_ms=%.3f warm snapshot min/median/max=%v SQL=%v", scenario.Term, scenario.Rare, count, len(want), samples[0].SnapshotMS, []float64{module[0], module[2], module[4]}, []float64{sql[0], sql[2], sql[4]})
			// Plans and candidate diagnostics run after timing this scenario.
			searchPerfExplain(t, pool, filepath.Join(dir, "plans"), scenario.Name, statement)
		})
	}
}
