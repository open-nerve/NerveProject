package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
	"uuid"

	planningpg "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The freeze's scale (M6 P8 spec 2.8): freezeScaleProjects projects, half
// of them on a schedule a week behind the other half's, each with
// freezeScaleCycles two-week cycles: freezeScaleEarlier, about a year and a
// half of them, that had ended at an earlier run, a week before
// freezeScaleDay, which froze them; the last, which ended the day before
// freezeScaleDay on the first schedule and ends six days after it on the
// second; the current one; the next one; and a draft. Each project's work
// items: two in each of the earlier cycles, thirty in the last, twenty in
// the current one and six in none, one assignee each. On freezeScaleDay
// the job's candidates are the first schedule's projects, one cycle each:
// one cycle in eighty is due, the others frozen, not ended or drafts.
const (
	freezeScaleProjects = 5000
	freezeScaleEarlier  = 36
	freezeScaleCycles   = freezeScaleEarlier + 4
	freezeScaleItems    = 2*freezeScaleEarlier + 56
	freezeScaleInCycles = 2*freezeScaleEarlier + 50
)

var freezeScaleDay = time.Date(2044, 6, 15, 0, 0, 0, 0, time.UTC)

// TestTheFreezeAtTheScaleTarget measures planning.freeze_cycles' one run
// on freezeScaleDay: the plan of its candidates' read, which the partial
// index cycles_freeze_due_idx serves, the run's time and each project's
// lock, held from its grant to its transaction's commit. Opt-in, as the
// deletions' scale.
func TestTheFreezeAtTheScaleTarget(t *testing.T) {
	if os.Getenv("NERVE_MEASURE_SCALE") == "" {
		t.Skip("NERVE_MEASURE_SCALE is not set: explicit scale measurement required")
	}
	dir, err := filepath.Abs(os.Getenv("NERVE_SCALE_EVIDENCE"))
	if err != nil || os.Getenv("NERVE_SCALE_EVIDENCE") == "" {
		t.Fatalf("NERVE_SCALE_EVIDENCE must name the evidence directory: %v", err)
	}
	t.Log("freeze scale evidence:", dir)
	db := pgtest.NewDatabase(t)
	var effective config.Config
	w := newIssueWorldOn(t, db, openPool(t, db), func(c *config.Config) {
		c.Auth.AccessTokenTTL = time.Hour
		effective = *c
	})
	// The app's own runs, on the real clock, are over before the fixture,
	// whose cycles are decades ahead of it, and the next is an hour away.
	waitForStartupRuns(t, w.pool)
	environment := scaleEnvironment(t, w, effective)
	environment["seed_version"] = fmt.Sprintf("freeze-v2:%dx%d cycles;%d frozen, 1 due on half;%d items", freezeScaleProjects, freezeScaleCycles, freezeScaleEarlier, freezeScaleItems)
	environment["measurement"] = "planning.freeze_cycles' use case as planning.New wires it, in the job's transactions, on a pool with the app's planner settings"
	listEvidence(t, dir, "freeze-environment", environment)
	all, due, created := w.freezeScaleWorld(t)
	seeded := w.seedFreezeScale(t, all, due)
	trace := &freezeScaleTrace{}
	pool := freezeScalePool(t, db, trace)
	day := func(days int) time.Time { return freezeScaleDay.AddDate(0, 0, days).Add(12 * time.Hour) }
	earlier := freezeScaleRun(t, pool, day(-7))
	listEvidence(t, dir, "freeze-earlier-run", earlier)
	if earlier.Err != "" || earlier.Frozen != freezeScaleProjects*freezeScaleEarlier {
		t.Fatalf("the earlier run froze %d, %s; want the %d cycles ended a week before", earlier.Frozen, earlier.Err, freezeScaleProjects*freezeScaleEarlier)
	}
	// As autovacuum keeps a deployment's cycles: without the earlier run's
	// dead rows, which would only make a scan of the table dearer.
	analyzed := time.Now()
	if _, err := w.pool.Exec(t.Context(), "VACUUM (ANALYZE) cycles,cycle_issues,issues,issue_assignees,issue_labels,projects,states"); err != nil {
		t.Fatal(err)
	}
	listEvidence(t, dir, "freeze-fixture", map[string]any{"projects": freezeScaleProjects, "due_projects": len(due), "cycles_per_project": freezeScaleCycles,
		"frozen_before": freezeScaleEarlier, "items_per_project": freezeScaleItems, "items_in_cycles": freezeScaleInCycles, "day": freezeScaleDay,
		"create_projects_seconds": created.Seconds(), "seed_seconds": seeded.Seconds(), "vacuum_analyze_seconds": time.Since(analyzed).Seconds()})

	at := day(0)
	plan := explainFreezeDue(t, pool, trace, at)
	listEvidence(t, dir, "freeze-candidates-plan", plan)
	if !slices.Equal(plan.Candidates, due) {
		t.Fatalf("candidates=%d; want the %d projects on the first schedule", len(plan.Candidates), len(due))
	}
	if !slices.Contains(plan.Indexes, "cycles_freeze_due_idx") {
		t.Fatalf("the candidates' plan does not use cycles_freeze_due_idx: nodes %v, indexes %v; plan in %s", plan.Nodes, plan.Indexes, dir)
	}
	run := freezeScaleRun(t, pool, at)
	listEvidence(t, dir, "freeze-run", run)
	if run.Err != "" || run.Frozen != len(due) || len(run.Holds) != len(due) {
		t.Fatalf("the run froze %d in %d projects, %s; want one cycle in each of %d", run.Frozen, len(run.Holds), run.Err, len(due))
	}
	for i, h := range run.Holds {
		if h.Project != due[i] || h.Failed {
			t.Fatalf("hold %d = %+v; want project %s, committed", i, h, due[i])
		}
	}
	var frozenAt int
	if err := w.pool.QueryRow(t.Context(), "SELECT count(*) FROM cycles WHERE (progress_snapshot->>'frozen_at')::timestamptz=$1", at).Scan(&frozenAt); err != nil || frozenAt != len(due) {
		t.Fatalf("cycles frozen at the run=%d %v; want %d", frozenAt, err, len(due))
	}
	if again, err := planningpg.New(pool).FreezeDueProjects(t.Context(), shared.DateOf(at)); err != nil || len(again) != 0 {
		t.Fatalf("candidates after the run=%d %v; want none", len(again), err)
	}
	t.Logf("freeze scale: %d candidates of %d projects, plan %v at cost %.2f (without index scans %.2f); run %.3fs, lock held %v ms", len(due), freezeScaleProjects, plan.Nodes, plan.Cost, plan.SeqScanCost, run.Seconds, run.HeldMS)
}

// freezeScaleWorld creates the projects through the API in Acme and
// answers them and, in id order, those on the first schedule: every other
// one.
func (w issueWorld) freezeScaleWorld(t *testing.T) ([]uuid.UUID, []uuid.UUID, time.Duration) {
	t.Helper()
	start := time.Now()
	var all, due []uuid.UUID
	for i := range freezeScaleProjects {
		id := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", fmt.Sprintf("Freeze %04d", i), fmt.Sprintf("FRZ%04d", i))
		all = append(all, id)
		if i%2 == 0 {
			due = append(due, id)
		}
	}
	slices.SortFunc(due, uuid.UUID.Compare)
	return all, due, time.Since(start)
}

// seedFreezeScale writes the projects' cycles and work items, a table at a
// time interleaving the projects as seedAtScale's, then analyzes them. A
// work item done was completed within its cycle, at the latest the day
// before freezeScaleDay.
func (w issueWorld) seedFreezeScale(t *testing.T, projects, due []uuid.UUID) time.Duration {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()
	start := time.Now()
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	alice, bob := w.ids["alice"], w.ids["bob"]
	for _, s := range []struct {
		sql  string
		args []any
	}{
		{`CREATE TEMP TABLE freeze_scale ON COMMIT DROP AS SELECT p.id project_id,p.workspace_id,CASE WHEN p.id=ANY($2) THEN 0 ELSE 7 END behind,
   ARRAY(SELECT s.id FROM states s,unnest(ARRAY['Backlog','Todo','In Progress','Done','Cancelled']) WITH ORDINALITY g(name,o)
   WHERE s.project_id=p.id AND s.deleted_at IS NULL AND s.name=g.name ORDER BY g.o) states
   FROM projects p WHERE p.id=ANY($1)`, []any{projects, due}},
		{`CREATE TEMP TABLE freeze_scale_items ON COMMIT DROP AS SELECT f.*,n,
   CASE WHEN n<=2*$2::int THEN (n-1)/2 WHEN n<=2*$2::int+30 THEN $2::int WHEN n<=2*$2::int+50 THEN $2::int+1 END AS cycle_no,
   $1::date-14*($2::int+1)+behind AS first_start FROM freeze_scale f,generate_series(1,$3::int) n`, []any{freezeScaleDay, freezeScaleEarlier, freezeScaleItems}},
		{`WITH s AS MATERIALIZED (SELECT f.*,j,$1::date-14*($2::int+1)+behind+14*j AS start_date FROM freeze_scale f,generate_series(0,$2::int+3) j ORDER BY j,f.project_id)
   INSERT INTO cycles(id,workspace_id,project_id,name,start_date,end_date,owned_by_id,created_by_id,updated_by_id)
   SELECT uuidv7(),workspace_id,project_id,'Cycle '||j,CASE WHEN j<$2::int+3 THEN start_date END,CASE WHEN j<$2::int+3 THEN start_date+13 END,$3,$3,$3 FROM s`,
			[]any{freezeScaleDay, freezeScaleEarlier, alice}},
		{`WITH s AS MATERIALIZED (SELECT * FROM freeze_scale_items ORDER BY n,project_id)
   INSERT INTO issues(id,workspace_id,project_id,state_id,name,sequence_id,completed_at,created_by_id,updated_by_id)
   SELECT uuidv7(),workspace_id,project_id,states[1+n%5],'Item '||n,n,
   CASE WHEN n%5=3 THEN (least(coalesce(first_start+14*cycle_no+n%14,$1::date-1),$1::date-1)+time '12:00') AT TIME ZONE 'UTC' END,$2,$2 FROM s`,
			[]any{freezeScaleDay, alice}},
		{`UPDATE projects SET last_issue_sequence=$1 WHERE id IN (SELECT project_id FROM freeze_scale)`, []any{freezeScaleItems}},
		{`WITH s AS MATERIALIZED (SELECT i.* FROM issues i JOIN freeze_scale USING(project_id) ORDER BY i.sequence_id,i.project_id)
   INSERT INTO issue_assignees(id,workspace_id,project_id,issue_id,assignee_id,created_by_id,updated_by_id)
   SELECT uuidv7(),workspace_id,project_id,id,CASE WHEN sequence_id%2=0 THEN $1::uuid ELSE $2::uuid END,$1,$1 FROM s`, []any{alice, bob}},
		{`WITH s AS MATERIALIZED (SELECT i.workspace_id,i.project_id,i.id issue_id,c.id cycle_id FROM freeze_scale_items f
   JOIN issues i ON i.project_id=f.project_id AND i.sequence_id=f.n JOIN cycles c ON c.project_id=f.project_id AND c.name='Cycle '||f.cycle_no
   ORDER BY f.n,f.project_id)
   INSERT INTO cycle_issues(id,workspace_id,project_id,cycle_id,issue_id,created_by_id,updated_by_id)
   SELECT uuidv7(),workspace_id,project_id,cycle_id,issue_id,$1,$1 FROM s`, []any{alice}},
	} {
		if _, err = tx.Exec(ctx, s.sql, s.args...); err != nil {
			t.Fatalf("%v: %s", err, s.sql)
		}
	}
	var seeded, states, cycles, items, members int
	if err = tx.QueryRow(ctx, `SELECT (SELECT count(*) FROM freeze_scale),(SELECT count(*) FROM freeze_scale WHERE cardinality(states)=5),
   (SELECT count(*) FROM cycles JOIN freeze_scale USING(project_id)),(SELECT count(*) FROM issues JOIN freeze_scale USING(project_id)),
   (SELECT count(*) FROM cycle_issues JOIN freeze_scale USING(project_id))`).Scan(&seeded, &states, &cycles, &items, &members); err != nil {
		t.Fatal(err)
	}
	if seeded != freezeScaleProjects || states != seeded || cycles != freezeScaleCycles*seeded || items != freezeScaleItems*seeded || members != freezeScaleInCycles*seeded {
		t.Fatalf("freeze scale seed projects/states/cycles/items/members=%d/%d/%d/%d/%d", seeded, states, cycles, items, members)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = w.pool.Exec(ctx, "ANALYZE cycles,cycle_issues,issues,issue_assignees,projects,states"); err != nil {
		t.Fatal(err)
	}
	return time.Since(start)
}
