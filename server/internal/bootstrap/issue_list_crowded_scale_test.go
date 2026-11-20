package bootstrap

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// listScaleCrowded runs TestIssueListScale's scenarios M4 measured
// (m4Scenarios) on its seed with ten more projects in the same database,
// n/10 roots each, none of walt's or bob's: every scenario's scope and
// expectation stay the same, while the planner's statistics see fifteen
// projects instead of five. Its evidence is under dir/crowded/<roots>.
func listScaleCrowded(t *testing.T, dir string) {
	for _, n := range []int{10000, 20000, 100000} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			roles := newSplitRoles(t)
			trace := &listTrace{}
			pool := listTracePool(t, roles.serverURL, trace)
			f := seedListScale(t, n, roles, pool, false)
			target := filepath.Join(dir, "crowded", strconv.Itoa(n))
			listEvidence(t, target, "metadata", listScaleMetadata(t, roles, pool))
			// The distribution counts every issue in the database: before the crowd.
			listEvidence(t, target, "distribution", f.checkDistribution(t))
			seeded := listScaleStatistics(t, f.owner)
			crowd := f.crowd(t, n/10)
			listEvidence(t, target, "seed", struct {
				Roots       int
				Projects    []listScaleProject
				Crowd       []listScaleProject
				SeedSeconds float64
				Accounts    any
			}{n, f.projects, crowd, f.SeedSeconds, f.w.ids})
			listEvidence(t, target, "statistics", map[string]any{"seeded": seeded, "crowded": listScaleStatistics(t, f.owner)})
			db := listScaleDB{f: f, pool: pool, trace: trace}
			for _, s := range f.m4Scenarios() {
				t.Run(s.Name, func(t *testing.T) { db.measureScenario(t, target, n, s) })
			}
		})
	}
}

// crowd adds ten projects of alice's in acme with roots roots each, seeded
// as seedProject seeds the issues of a project (the same states, kinds,
// creators and dates; no relations), then analyzes the tables the seed
// analyzes.
func (f listScaleFixture) crowd(t *testing.T, roots int) []listScaleProject {
	t.Helper()
	crowd := make([]listScaleProject, 10)
	for c := range crowd {
		id := createdProject(t, f.w.contract, f.w.base, f.w.tokens["alice"], "acme", fmt.Sprintf("Crowd %d", c), fmt.Sprintf("C%d", c))
		crowd[c] = listScaleProject{ID: id, Roots: roots}
		for _, state := range []string{"Backlog", "Todo", "In Progress", "Done", "Cancelled", "Triage"} {
			crowd[c].States = append(crowd[c].States, stateID(t, f.w.pool, id, state))
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	err := pgx.BeginFunc(ctx, f.owner, func(tx pgx.Tx) error {
		for c, project := range crowd {
			p := len(f.projects) + c
			if _, err := tx.Exec(ctx, `WITH numbers AS (
 SELECT n,n+1 AS seq,0 AS kind FROM generate_series(0,$1::int-1) n
 UNION ALL SELECT k*100,$1::int+k*7+kind,kind FROM generate_series(0,$1::int/100-1) k CROSS JOIN generate_series(1,7) kind)
 INSERT INTO issues (id,workspace_id,project_id,state_id,name,sequence_id,parent_id,priority,created_by_id,updated_by_id,
 created_at,updated_at,start_date,target_date,sort_order,completed_at,archived_at,deleted_at)
 SELECT ('00000000-0000-7000-8000-'||lpad(to_hex(($2::bigint+1)*1000000-seq),12,'0'))::uuid,$3,$4,
 ($5::uuid[])[CASE WHEN kind=1 THEN 6 ELSE (n/5)%5+1 END],'Crowd '||seq,seq,
 CASE WHEN kind>=4 THEN ('00000000-0000-7000-8000-'||lpad(to_hex(($2::bigint+1)*1000000-(n+1)),12,'0'))::uuid END,
 (ARRAY['urgent','high','medium','low','none'])[n%5+1],
 CASE WHEN kind=4 THEN $8::uuid WHEN kind=5 THEN $6::uuid ELSE (ARRAY[$6::uuid,$7::uuid,$8::uuid,NULL])[n%4+1] END,$6,
 '2026-01-01Z'::timestamptz+(n/4)*interval '1 microsecond',
 '2026-01-02Z'::timestamptz+(n%17)*interval '1 microsecond',
 CASE WHEN n%3<>0 THEN '2026-01-01'::date+n%11 END,
 CASE WHEN n%4<>0 THEN '2026-01-21'::date+n%7 END,1000+n%8,
 CASE WHEN (n/5)%5=3 AND kind<>1 THEN '2026-01-03Z'::timestamptz END,
 CASE WHEN kind IN (2,6) THEN '2026-02-01Z'::timestamptz END,
 CASE WHEN kind IN (3,7) THEN '2026-02-02Z'::timestamptz END FROM numbers`,
				project.Roots, p, f.w.acme, project.ID, project.States, f.w.ids["alice"], f.w.ids["bob"], f.w.ids["walt"]); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE projects SET last_issue_sequence=$2 WHERE id=$1`, project.ID, project.Roots+7*(project.Roots/100)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.owner.Exec(ctx, "ANALYZE issues, issue_labels, issue_assignees, issue_subscribers, issue_mentions, projects, project_members, states, labels"); err != nil {
		t.Fatal(err)
	}
	return crowd
}

// listScaleStatistics is what ANALYZE left the planner on issues: its row
// count, and the columns a list's project and state directories join on.
func listScaleStatistics(t *testing.T, owner *pgxpool.Pool) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	var reltuples float64
	if err := owner.QueryRow(ctx, "SELECT reltuples::float8 FROM pg_class WHERE oid='public.issues'::regclass").Scan(&reltuples); err != nil {
		t.Fatal(err)
	}
	rows, err := owner.Query(ctx, `SELECT attname::text, n_distinct::float8, null_frac::float8, coalesce(most_common_freqs[1:3],'{}')::float8[]
 FROM pg_stats WHERE schemaname='public' AND tablename='issues' AND attname IN ('project_id','state_id') ORDER BY attname`)
	if err != nil {
		t.Fatal(err)
	}
	columns := map[string]any{}
	var name string
	var distinct, nulls float64
	var common []float64
	if _, err = pgx.ForEachRow(rows, []any{&name, &distinct, &nulls, &common}, func() error {
		columns[name] = map[string]any{"n_distinct": distinct, "null_frac": nulls, "most_common_freqs_1_3": slices.Clone(common)}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(columns) != 2 {
		t.Fatal("issues statistics", columns)
	}
	return map[string]any{"reltuples": reltuples, "columns": columns}
}
