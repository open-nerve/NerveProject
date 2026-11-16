package bootstrap

import (
	"context"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	issueriver "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
	planningriver "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// scalePlanningTables are the deletion scale's planning rows (M6 P8 spec
// 2.8) and the job that deletes each. A project's eight cycles and six
// modules, each module with three members and two links, go by
// planning.delete_rows (planning/app/delete_rows.go, DeletionRowSteps). Its
// work items' memberships, each work item in one cycle and two modules, are
// the issue module's rows: cycle_issues and module_issues go by
// issue.delete_rows, with the work items' other rows (issue/app/
// delete_rows.go, DeletionRowSteps; M6 P3a spec 2.5). A table has Rows rows
// in each project, or PerParent for each of its work items; PerParent is
// each parent's fan-out.
var scalePlanningTables = []struct {
	Name, Job, Parent string
	Rows, PerParent   int
}{
	{"cycles", planningriver.DeleteRowsKind, "", 8, 0},
	{"modules", planningriver.DeleteRowsKind, "", 6, 0},
	{"module_members", planningriver.DeleteRowsKind, "module_id", 18, 3},
	{"module_links", planningriver.DeleteRowsKind, "module_id", 12, 2},
	{"cycle_issues", issueriver.DeleteRowsKind, "issue_id", 0, 1},
	{"module_issues", issueriver.DeleteRowsKind, "issue_id", 0, 2},
}

// seedPlanningAtScale writes the planning rows of the projects in one
// transaction, each table interleaving the projects as seedAtScale's do,
// then analyzes them. The cycles are dated around today, none ended, and
// two have no dates, one of them archived: no freeze locks a project while
// the deletions are measured. The last module is archived. TestPurgeScale's
// seedAtScale writes none of these; its sparse fixture has its own.
func (w issueWorld) seedPlanningAtScale(t *testing.T, projects []uuid.UUID) (time.Duration, time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()
	start := time.Now()
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	alice, bob, walt := w.ids["alice"], w.ids["bob"], w.ids["walt"]
	for _, s := range []struct {
		sql  string
		args []any
	}{
		{`WITH s AS MATERIALIZED (SELECT p.id project_id,p.workspace_id,k FROM projects p,generate_series(0,7) k WHERE p.id=ANY($1) ORDER BY k,p.id)
   INSERT INTO cycles(id,workspace_id,project_id,name,start_date,end_date,owned_by_id,archived_at,created_by_id,updated_by_id)
   SELECT uuidv7(),workspace_id,project_id,'Cycle '||k,CASE WHEN k<6 THEN current_date+14*k-7 END,CASE WHEN k<6 THEN current_date+14*k+6 END,
   $2,CASE WHEN k=7 THEN now() END,$2,$2 FROM s`, []any{projects, alice}},
		{`WITH s AS MATERIALIZED (SELECT p.id project_id,p.workspace_id,k FROM projects p,generate_series(0,5) k WHERE p.id=ANY($1) ORDER BY k,p.id)
   INSERT INTO modules(id,workspace_id,project_id,name,status,lead_id,archived_at,created_by_id,updated_by_id)
   SELECT uuidv7(),workspace_id,project_id,'Module '||k,(ARRAY['backlog','planned','in-progress','paused','completed','cancelled'])[k+1],
   $2,CASE WHEN k=5 THEN now() END,$2,$2 FROM s`, []any{projects, alice}},
		{`WITH s AS MATERIALIZED (SELECT m.*,a FROM modules m,unnest(ARRAY[$2::uuid,$3::uuid,$4::uuid]) a WHERE m.project_id=ANY($1) ORDER BY m.name,a,m.project_id)
   INSERT INTO module_members(id,workspace_id,project_id,module_id,member_id,created_by_id,updated_by_id)
   SELECT uuidv7(),workspace_id,project_id,id,a,$2,$2 FROM s`, []any{projects, alice, bob, walt}},
		{`WITH s AS MATERIALIZED (SELECT m.*,k FROM modules m,generate_series(1,2) k WHERE m.project_id=ANY($1) ORDER BY m.name,k,m.project_id)
   INSERT INTO module_links(id,workspace_id,project_id,module_id,title,url,created_by_id,updated_by_id)
   SELECT uuidv7(),workspace_id,project_id,id,'Spec '||k,'https://nerve.dev/spec/'||k,$2,$2 FROM s`, []any{projects, alice}},
		{`WITH s AS MATERIALIZED (SELECT i.workspace_id,i.project_id,i.id issue_id,c.id cycle_id FROM issues i
   JOIN cycles c ON c.project_id=i.project_id AND c.name='Cycle '||(i.sequence_id%8) WHERE i.project_id=ANY($1) ORDER BY i.sequence_id,i.project_id)
   INSERT INTO cycle_issues(id,workspace_id,project_id,cycle_id,issue_id,created_by_id,updated_by_id)
   SELECT uuidv7(),workspace_id,project_id,cycle_id,issue_id,$2,$2 FROM s`, []any{projects, alice}},
		{`WITH s AS MATERIALIZED (SELECT i.workspace_id,i.project_id,i.id issue_id,m.id module_id FROM issues i CROSS JOIN generate_series(0,1) k
   JOIN modules m ON m.project_id=i.project_id AND m.name='Module '||((i.sequence_id+k)%6) WHERE i.project_id=ANY($1) ORDER BY i.sequence_id,k,i.project_id)
   INSERT INTO module_issues(id,workspace_id,project_id,module_id,issue_id,created_by_id,updated_by_id)
   SELECT uuidv7(),workspace_id,project_id,module_id,issue_id,$2,$2 FROM s`, []any{projects, alice}},
	} {
		if _, err = tx.Exec(ctx, s.sql, s.args...); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	seeded := time.Since(start)
	start = time.Now()
	if _, err = w.pool.Exec(ctx, `ANALYZE cycles,modules,module_members,module_links,cycle_issues,module_issues`); err != nil {
		t.Fatal(err)
	}
	return seeded, time.Since(start)
}

type scalePlanningRows struct {
	Project                              uuid.UUID
	Table, Job                           string
	Want, Total, Live, Stamped, Batches  int
	Parents, MinFanout, MaxFanout, Items int
}

// checkScalePlanningRows checks each project's planning rows against its
// work items, independently of the seeding SQL. A project in gone is
// deleted, and so is each of its rows, none removed (soft deletion, as
// checkScaleRows; TestPurgeScale removes them), each stamped with the
// project's deletion moment and deleter; Batches counts the transactions
// that wrote a table's deleted rows, a batch each. Any other project's rows
// are live, each parent with its fan-out: a deletion leaves its neighbours
// as they were.
func (w issueWorld) checkScalePlanningRows(t *testing.T, projects []uuid.UUID, gone map[uuid.UUID]bool, dir, name string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	var evidence []scalePlanningRows
	for _, p := range projects {
		var items int
		var at *time.Time
		var by *uuid.UUID
		if err := w.pool.QueryRow(ctx, "SELECT (SELECT count(*) FROM issues WHERE project_id=$1),deleted_at,updated_by_id FROM projects WHERE id=$1", p).Scan(&items, &at, &by); err != nil {
			t.Fatal(err)
		}
		if gone[p] != (at != nil) || items == 0 {
			t.Fatalf("SP2 planning project %s deleted_at=%v items=%d; want deleted=%t with work items", p, at, items, gone[p])
		}
		for _, table := range scalePlanningTables {
			r := scalePlanningRows{Project: p, Table: table.Name, Job: table.Job, Want: table.Rows, Items: items}
			if r.Want == 0 {
				r.Want = items * table.PerParent
			}
			if err := w.pool.QueryRow(ctx, "SELECT count(*),count(*) FILTER(WHERE deleted_at IS NULL),count(*) FILTER(WHERE deleted_at=$2 AND updated_at=$2 AND updated_by_id=$3),count(DISTINCT xmin::text) FILTER(WHERE deleted_at IS NOT NULL) FROM "+pgx.Identifier{table.Name}.Sanitize()+" WHERE project_id=$1", p, at, by).Scan(&r.Total, &r.Live, &r.Stamped, &r.Batches); err != nil {
				t.Fatal(err)
			}
			if !gone[p] && table.Parent != "" {
				if err := w.pool.QueryRow(ctx, "SELECT count(*),coalesce(min(n),0),coalesce(max(n),0) FROM (SELECT count(*) n FROM "+pgx.Identifier{table.Name}.Sanitize()+" WHERE project_id=$1 AND deleted_at IS NULL GROUP BY "+pgx.Identifier{table.Parent}.Sanitize()+") s", p).Scan(&r.Parents, &r.MinFanout, &r.MaxFanout); err != nil {
					t.Fatal(err)
				}
			}
			evidence = append(evidence, r)
			listEvidence(t, dir, name+"-counts", evidence)
			switch {
			case r.Total != r.Want:
				t.Fatalf("SP2 planning %s project=%s total=%d want %d", table.Name, p, r.Total, r.Want)
			case gone[p] && (r.Live != 0 || r.Stamped != r.Total):
				t.Fatalf("SP2 planning %s project=%s live/stamped=%d/%d want 0/%d by %s", table.Name, p, r.Live, r.Stamped, r.Total, table.Job)
			case !gone[p] && r.Live != r.Total:
				t.Fatalf("SP2 planning neighbour %s project=%s live=%d want %d", table.Name, p, r.Live, r.Total)
			case !gone[p] && table.Parent != "" && (r.MinFanout != table.PerParent || r.MaxFanout != table.PerParent || r.Parents*table.PerParent != r.Total):
				t.Fatalf("SP2 planning %s project=%s fanout=%d..%d over %d parents want %d", table.Name, p, r.MinFanout, r.MaxFanout, r.Parents, table.PerParent)
			}
		}
	}
}

type scalePlanningJob struct {
	ID                                     int64
	Project                                uuid.UUID
	State                                  string
	Attempt, Failures, Snoozes, Runs       int
	RunningSeconds, FirstRunToFinalSeconds float64
	Transitions                            []jobTransition
}

// scalePlanningJobs finds each deleted project's one planning.delete_rows
// job, enqueued in its deletion's transaction (planning/app/
// delete_rows_on_deletion.go).
func scalePlanningJobs(t *testing.T, w issueWorld, projects []scaleProject) []scalePlanningJob {
	t.Helper()
	jobs := make([]scalePlanningJob, 0, len(projects))
	for _, p := range projects {
		rows, err := w.pool.Query(pgtest.Soon(t), "SELECT id FROM river_job WHERE kind=$1 AND args->>'project_id'=$2", planningriver.DeleteRowsKind, p.ID.String())
		if err != nil {
			t.Fatal(err)
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
		if err != nil || len(ids) != 1 {
			t.Fatalf("project %s planning deletion jobs=%v %v; want exactly one", p.ID, ids, err)
		}
		jobs = append(jobs, scalePlanningJob{ID: ids[0], Project: p.ID})
	}
	return jobs
}

// awaitScalePlanningDeletions waits for each planning job with
// waitPlanningDeletion: its deletion's moment and deleter, the only one of
// its project, completed in its first attempt without a failure. A snooze
// at the job's budget is recorded, not refused: scaleSingleRun's one run
// without a snooze binds the issue jobs alone (M6 P8 spec 2.8). Then each
// job's runs, snoozes and running seconds, its batches in the rows' check.
func (w issueWorld) awaitScalePlanningDeletions(t *testing.T, jobs []scalePlanningJob, dir, name string) {
	t.Helper()
	for i := range jobs {
		j := &jobs[i]
		waitPlanningDeletion(t, w.pool, j.Project)
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT state::text,attempt,coalesce(cardinality(errors),0),coalesce((metadata->>'snoozes')::int,0) FROM river_job WHERE id=$1", j.ID).Scan(&j.State, &j.Attempt, &j.Failures, &j.Snoozes); err != nil {
			t.Fatal(err)
		}
		j.Transitions = jobTransitions(t, w.pool, j.ID)
		var first, running time.Time
		for _, tr := range j.Transitions {
			if tr.To == "running" {
				j.Runs++
				running = tr.At
				if first.IsZero() {
					first = tr.At
				}
			} else if tr.From == "running" {
				j.RunningSeconds += tr.At.Sub(running).Seconds()
			}
		}
		if j.Runs == 0 || j.State != "completed" || j.Transitions[len(j.Transitions)-1].To != "completed" {
			t.Fatalf("SP2 planning job=%d %+v; want its runs observed up to its completion", j.ID, j)
		}
		j.FirstRunToFinalSeconds = j.Transitions[len(j.Transitions)-1].At.Sub(first).Seconds()
		listEvidence(t, dir, name+"-planning-jobs", jobs)
		t.Logf("SP2 planning job=%d project=%s runs=%d snoozes=%d attempt=%d running_seconds=%.6f", j.ID, j.Project, j.Runs, j.Snoozes, j.Attempt, j.RunningSeconds)
	}
}
