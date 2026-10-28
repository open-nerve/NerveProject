package bootstrap

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

type jobTransition struct {
	JobID    int64
	From, To string
	Attempt  int
	At       time.Time
}

// Install before starting the app: its cleanup stops and joins workers before
// this cleanup drops the observer. No production schema migration is involved.
func observeJobStates(t testing.TB, pool *pgxpool.Pool) {
	t.Helper()
	for _, sql := range []string{
		`CREATE SCHEMA job_observer`,
		`CREATE TABLE job_observer.transitions (ordinal bigint GENERATED ALWAYS AS IDENTITY, job_id bigint NOT NULL, old_state text NOT NULL, new_state text NOT NULL, attempt integer NOT NULL, at timestamptz NOT NULL)`,
		`CREATE FUNCTION job_observer.record() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO job_observer.transitions(job_id,old_state,new_state,attempt,at) VALUES(NEW.id,OLD.state::text,NEW.state::text,NEW.attempt,clock_timestamp()); RETURN NEW; END $$`,
		`CREATE TRIGGER observe_job_state AFTER UPDATE OF state ON river_job FOR EACH ROW WHEN(OLD.state IS DISTINCT FROM NEW.state) EXECUTE FUNCTION job_observer.record()`,
		`DO $$ BEGIN IF EXISTS(SELECT 1 FROM pg_roles WHERE rolname='nerve_runtime') THEN GRANT USAGE ON SCHEMA job_observer TO nerve_runtime; GRANT INSERT,SELECT ON job_observer.transitions TO nerve_runtime; GRANT USAGE ON SEQUENCE job_observer.transitions_ordinal_seq TO nerve_runtime; GRANT EXECUTE ON FUNCTION job_observer.record() TO nerve_runtime; END IF; END $$`,
	} {
		if _, err := pool.Exec(pgtest.Soon(t), sql); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := pool.Exec(ctx, `DROP TRIGGER observe_job_state ON river_job; DROP SCHEMA job_observer CASCADE`); err != nil {
			t.Error(err)
		}
	})
}

func jobTransitions(t testing.TB, pool *pgxpool.Pool, id int64) []jobTransition {
	t.Helper()
	rows, err := pool.Query(pgtest.Soon(t), `SELECT job_id,old_state,new_state,attempt,at FROM job_observer.transitions WHERE job_id=$1 ORDER BY ordinal`, id)
	if err != nil {
		t.Fatal(err)
	}
	got, err := pgx.CollectRows(rows, pgx.RowToStructByPos[jobTransition])
	if err != nil {
		t.Fatal(err)
	}
	return got
}

type observedJob struct {
	State                      string
	Attempt, Failures, Snoozes int
	Finalized                  *time.Time
	Errors                     []byte
}

func readObservedJob(t testing.TB, pool *pgxpool.Pool, job deletionJob) observedJob {
	t.Helper()
	var out observedJob
	var raw []byte
	var kind, queue string
	var maximum int
	err := pool.QueryRow(pgtest.Soon(t), `SELECT state::text,attempt,coalesce(cardinality(errors),0),coalesce((metadata->>'snoozes')::int,0),finalized_at,to_json(errors),kind,args,queue,max_attempts FROM river_job WHERE id=$1`, job.ID).Scan(&out.State, &out.Attempt, &out.Failures, &out.Snoozes, &out.Finalized, &out.Errors, &kind, &raw, &queue, &maximum)
	if err != nil {
		t.Fatal(err)
	}
	got := deletionJob{ID: job.ID}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got != job || kind != "issue.delete_rows" || queue != "default" || maximum != 25 {
		t.Fatalf("job identity/options changed: %+v %s %s %d", got, kind, queue, maximum)
	}
	return out
}

func awaitObservedJob(t *testing.T, pool *pgxpool.Pool, job deletionJob, ready func(observedJob) bool) observedJob {
	t.Helper()
	var got observedJob
	requestEventuallyWithin(t, "River lifecycle", 20*time.Second, func(context.Context) bool {
		got = readObservedJob(t, pool, job)
		return ready(got)
	})
	t.Logf("job=%d state=%s attempt=%d errors=%s snoozes=%d transitions=%+v", job.ID, got.State, got.Attempt, got.Errors, got.Snoozes, jobTransitions(t, pool, job.ID))
	return got
}

func assertJobRuns(t testing.TB, pool *pgxpool.Pool, job deletionJob, attempts []int, terminal string) {
	t.Helper()
	transitions := jobTransitions(t, pool, job.ID)
	n := 0
	running := false
	for _, tr := range transitions {
		if tr.At.IsZero() || tr.JobID != job.ID {
			t.Fatal("invalid transition", tr)
		}
		if tr.To == "running" {
			if running || n >= len(attempts) || tr.Attempt != attempts[n] {
				t.Fatalf("unexpected running: %+v want %v", transitions, attempts)
			}
			n++
			running = true
		} else if tr.From == "running" {
			if !running {
				t.Fatal("unpaired completion", transitions)
			}
			running = false
		}
	}
	if running || n != len(attempts) || len(transitions) == 0 || transitions[len(transitions)-1].To != terminal {
		t.Fatalf("incomplete lifecycle: %+v want %v -> %s", transitions, attempts, terminal)
	}
}

func TestJobStateObserverRunsWithRuntimeRole(t *testing.T) {
	roles := newSplitRoles(t)
	observeJobStates(t, roles.owner)
	w := newIssueWorldOn(t, roles.serverURL, roles.owner, func(*config.Config) {})
	createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{}}`)
	holder := holdProjectDeletion(t, roles.owner, w.web)
	w.send(t, "alice", "DELETE", "/api/v0/projects/"+w.web.String(), "", 204)
	job := projectDeletionJob(t, roles.owner, w.web)
	deletionWorkerBlocked(t, roles.server, holder, job, "draft_issues")
	requestRelease(t, holder)
	waitDeletionJob(t, roles.owner, job, "completed")
	assertJobRuns(t, roles.server, job, []int{1}, "completed")
}
