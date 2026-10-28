package bootstrap

import (
	"context"
	"errors"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func TestLateDeleteRowsAfterPurge(t *testing.T) {
	for _, parent := range []string{"projects", "workspaces"} {
		t.Run(parent, func(t *testing.T) {
			f := newDeletionLifecycle(t, func(*config.Config) {}, time.Now, nil)
			w := f.w
			w.tokens["member"] = registerAccount(t, w.contract, w.base, "member@example.com").AccessToken
			ws := seedWorkspace(t, w.contract, w.base, w.pool, w.tokens["alice"], "late")
			id := ws
			path := "/api/v0/workspaces/late"
			if parent == "projects" {
				id = queryIDs(t, w.pool, `SELECT id FROM projects WHERE workspace_id=$1 ORDER BY id`, ws)[0]
				path = "/api/v0/projects/" + id.String()
			}
			catalog := map[foreignKey][]uuid.UUID{}
			for _, k := range keysTo(t, w.pool, parent) {
				catalog[k] = append(k.undeleted(t, w.pool, id), k.deleted(t, w.pool, id)...)
				if len(catalog[k]) == 0 {
					t.Fatal("missing late cascade fixture", k)
				}
			}
			accounts := tableRows(t, w.pool, func(name string) bool { return name != "public.users" && name != "public.profiles" })
			kept := map[foreignKey]string{}
			for _, k := range keysTo(t, w.pool, "projects") {
				kept[k] = k.rows(t, w.pool, w.web)
			}
			setDeletionControl(t, w.pool, true, false)
			actor := "alice"
			if parent == "projects" {
				actor = "member"
			}
			w.send(t, actor, "DELETE", path, "", 204)
			var pending []deletionJob
			if parent == "projects" {
				pending = append(pending, projectDeletionJob(t, w.pool, id))
			} else {
				for _, p := range queryIDs(t, w.pool, `SELECT id FROM projects WHERE workspace_id=$1 ORDER BY id`, ws) {
					pending = append(pending, projectDeletionJob(t, w.pool, p))
				}
				pending = append(pending, parentDeletionJob(t, w.pool, "workspace_drafts", ws))
			}
			// Age only the requested parent: this isolates its real CASCADE from
			// earlier state/label purges. The original job's request audit is retained.
			if _, err := w.pool.Exec(pgtest.Soon(t), `UPDATE `+parent+` SET deleted_at=clock_timestamp()-interval '61 days' WHERE id=$1`, id); err != nil {
				t.Fatal(err)
			}
			if _, err := retentionPurge(w, time.Now()).Execute(pgtest.Soon(t)); err != nil {
				t.Fatal(err)
			}
			assertLateCatalogEmpty(t, w, catalog)
			setDeletionControl(t, w.pool, false, false)
			for _, j := range pending {
				releaseDeletionJob(t, w.pool, j, 0)
				got := awaitObservedJob(t, w.pool, j, func(j observedJob) bool { return j.State == "completed" })
				if got.Attempt != 1 || got.Failures != 0 {
					t.Fatal(got)
				}
				assertJobRuns(t, w.pool, j, []int{1}, "completed")
			}
			for _, r := range f.execution.snapshot() {
				if r.Err != nil || r.Result.Deleted != 0 || !r.Result.Complete {
					t.Fatal("late job did not finish at zero candidates", r)
				}
			}
			assertLateCatalogEmpty(t, w, catalog)
			for k, before := range kept {
				if k.rows(t, w.pool, w.web) != before {
					t.Fatal("late purge changed live neighbor", k)
				}
			}
			if after := tableRows(t, w.pool, func(name string) bool { return name != "public.users" && name != "public.profiles" }); !reflect.DeepEqual(accounts, after) {
				t.Fatal("late purge changed accounts")
			}
			w.send(t, "alice", "GET", "/api/v0/projects/"+w.web.String(), "", 200)
		})
	}
}

func assertLateCatalogEmpty(t testing.TB, w issueWorld, catalog map[foreignKey][]uuid.UUID) {
	t.Helper()
	for k, ids := range catalog {
		var n int
		if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*) FROM `+k.table+` WHERE id=ANY($1)`, ids).Scan(&n); err != nil || n != 0 {
			t.Fatalf("late cascade left %s rows=%d: %v", k, n, err)
		}
	}
}

// The two actual statements acquire opposite child orders. The trigger only
// pauses DELETE after PostgreSQL locked B; it takes no business-row lock itself.
func TestLateDeleteRowsAndPurgeRetry(t *testing.T) {
	for _, victim := range []string{"deletion", "purge"} {
		t.Run(victim, func(t *testing.T) {
			workerDetect, purgeDetect := "5s", "30s"
			if victim == "purge" {
				workerDetect, purgeDetect = "30s", "1s"
			}
			var purgeErrors = make(chan error, 8)
			f := newDeletionLifecycle(t, func(c *config.Config) {
				c.Issue.DeleteBatch = 2
				c.Issue.DeleteBudget = time.Minute
				c.Issue.WriteLockTimeout = time.Minute
				u, err := url.Parse(c.Database.URL)
				if err != nil {
					t.Fatal(err)
				}
				q := u.Query()
				q.Set("deadlock_timeout", workerDetect)
				u.RawQuery = q.Encode()
				c.Database.URL = u.String()
			}, time.Now, func(pool *pgxpool.Pool) []jobs.Job {
				pc := pool.Config()
				pc.ConnConfig.RuntimeParams["deadlock_timeout"] = purgeDetect
				pc.ConnConfig.RuntimeParams["enable_indexscan"] = "off"
				pc.ConnConfig.RuntimeParams["enable_bitmapscan"] = "off"
				pp, err := pgxpool.NewWithConfig(t.Context(), pc)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(pp.Close)
				p := retentionPurge(issueWorld{pool: pp}, time.Now())
				for i, m := range p.modules {
					p.modules[i] = purgeFunc(func(ctx context.Context, cutoff time.Time, batch int) (int, error) {
						n, err := m.Purge(ctx, cutoff, batch)
						if err != nil {
							purgeErrors <- err
						}
						return n, err
					})
				}
				return []jobs.Job{purgeJob(p, 2*time.Second)}
			})
			w := f.w
			a := w.newWorkItemIn(t, w.web, `{"name":"A"}`)
			b := w.newWorkItemIn(t, w.web, `{"name":"B"}`)
			// The API creates the self-CASCADE and moves A's tuple after B. The
			// dedicated purge connection uses a heap scan; observed ownership below
			// is authoritative, never an assumption about the planner's row order.
			w.send(t, "alice", "PATCH", "/api/v0/issues/"+a.String(), `{"parent_id":"`+b.String()+`"}`, 200)
			catalog := map[foreignKey][]uuid.UUID{}
			for _, k := range keysTo(t, w.pool, "projects") {
				catalog[k] = append(k.undeleted(t, w.pool, w.web), k.deleted(t, w.pool, w.web)...)
			}
			holder := holding(t, w.pool, `SELECT 1 FROM pg_advisory_xact_lock(734120)`)
			defer requestRelease(t, holder)
			for _, sql := range []string{
				`CREATE FUNCTION job_observer.pause_cascade() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock(734120); RETURN OLD; END $$`,
				`CREATE TRIGGER pause_cascade BEFORE DELETE ON issues FOR EACH ROW WHEN(OLD.id='` + b.String() + `'::uuid) EXECUTE FUNCTION job_observer.pause_cascade()`,
			} {
				if _, err := w.pool.Exec(pgtest.Soon(t), sql); err != nil {
					t.Fatal(err)
				}
			}
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if _, err := w.pool.Exec(ctx, `DROP TRIGGER pause_cascade ON issues`); err != nil {
					t.Error(err)
				}
			})
			setDeletionControl(t, w.pool, true, true)
			w.send(t, "alice", "DELETE", "/api/v0/projects/"+w.web.String(), "", 204)
			job := projectDeletionJob(t, w.pool, w.web)
			if _, err := w.pool.Exec(pgtest.Soon(t), `UPDATE projects SET deleted_at=clock_timestamp()-interval '61 days' WHERE id=$1`, w.web); err != nil {
				t.Fatal(err)
			}
			var purgePID uint32
			var purgeID int64
			requestEventuallyWithin(t, "real parent CASCADE paused on B", 10*time.Second, func(ctx context.Context) bool {
				return w.pool.QueryRow(ctx, `SELECT a.pid,j.id FROM pg_stat_activity a CROSS JOIN river_job j WHERE $1::int=ANY(pg_blocking_pids(a.pid)) AND a.query LIKE '%DELETE FROM projects%' AND j.kind='cleanup.purge_deleted' AND j.state='running'`, holder.Conn().PgConn().PID()).Scan(&purgePID, &purgeID) == nil
			})
			parentProbe := &requestProbe{}
			parentProbe.pid.Store(purgePID)
			if mode := requestRowLock(t, w.pool, w.pool, parentProbe, "issues WHERE id=$1", b); mode != "FOR UPDATE" {
				t.Fatal("CASCADE does not own B", mode)
			}
			setDeletionControl(t, w.pool, false, true)
			releaseDeletionJob(t, w.pool, job, 0)
			pid := deletionWorkerBlockedBy(t, w.pool, purgePID, job, "issues")
			workerProbe := &requestProbe{}
			workerProbe.pid.Store(pid)
			if mode := requestRowLock(t, w.pool, w.pool, workerProbe, "issues WHERE id=$1", a); mode != "FOR NO KEY UPDATE" {
				t.Fatal("batch does not own A", mode)
			}
			if !pgtest.RowWaits(t, w.pool, "issues", a, "FOR UPDATE") || !pgtest.RowWaits(t, w.pool, "issues", b, "FOR UPDATE") {
				t.Fatal("missing real child lock")
			}
			t.Logf("actual lock graph: parent purge job=%d PID=%d owns B=%s; deletion job=%d PID=%d owns A=%s and waits for B", purgeID, purgePID, b, job.ID, pid, a)
			requestRelease(t, holder)
			if victim == "deletion" {
				failed := awaitObservedJob(t, w.pool, job, func(j observedJob) bool { return j.State == "completed" || j.Failures > 0 && j.State != "running" })
				var pgerr *pgconn.PgError
				failures := f.tx.failures()
				if len(failures) != 1 || failures[0].Inside != failures[0].Outside || !errors.As(failures[0].Outside, &pgerr) || pgerr.Code != "40P01" {
					t.Fatal("not original deletion deadlock", failures, failed)
				}
				t.Logf("raw deletion transaction SQLSTATE=%s error=%v", pgerr.Code, failures[0].Outside)
				if !strings.Contains(string(failed.Errors), "40P01") {
					t.Fatal("River lost deadlock", failed)
				}
				setDeletionControl(t, w.pool, false, false)
				releaseDeletionJob(t, w.pool, job, 1)
				got := awaitObservedJob(t, w.pool, job, func(j observedJob) bool { return j.State == "completed" })
				if got.Attempt != 2 || got.Failures != 1 {
					t.Fatal(got)
				}
				assertJobRuns(t, w.pool, job, []int{1, 2}, "completed")
			} else {
				err := receiveWithin(t, purgeErrors, 10*time.Second, "parent CASCADE deadlock")
				var pgerr *pgconn.PgError
				if !errors.As(err, &pgerr) || pgerr.Code != "40P01" {
					t.Fatal("not real parent deadlock", err)
				}
				waitDeletionJob(t, w.pool, job, "completed")
			}
			var state string
			var attempts, failures int
			requestEventuallyWithin(t, "same purge job converges", 20*time.Second, func(ctx context.Context) bool {
				if err := w.pool.QueryRow(ctx, `SELECT state::text,attempt,coalesce(cardinality(errors),0) FROM river_job WHERE id=$1`, purgeID).Scan(&state, &attempts, &failures); err != nil {
					t.Fatal(err)
				}
				return state == "completed"
			})
			want := 1
			if victim == "purge" {
				want = 2
			}
			if attempts != want || failures != want-1 {
				t.Fatal("purge did not retry same ID", purgeID, state, attempts, failures)
			}
			attemptsWant := []int{1}
			if victim == "purge" {
				attemptsWant = append(attemptsWant, 2)
			}
			assertJobRuns(t, w.pool, deletionJob{ID: purgeID}, attemptsWant, "completed")
			var rawErrors string
			if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT coalesce(to_json(errors)::text,'') FROM river_job WHERE id=$1`, purgeID).Scan(&rawErrors); err != nil {
				t.Fatal(err)
			}
			if victim == "purge" && !strings.Contains(rawErrors, "40P01") {
				t.Fatal("purge's River errors lost SQLSTATE", rawErrors)
			}
			tr := jobTransitions(t, w.pool, purgeID)
			assertLateCatalogEmpty(t, w, catalog)
			t.Logf("victim=%s; deletion %d and original purge %d converged; purge transitions=%+v", victim, job.ID, purgeID, tr)
		})
	}
}
