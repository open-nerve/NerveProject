package bootstrap

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	notificationriver "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fanoutJob is a notification.fanout job of a work item as River holds it: its id, arguments, state, attempt and
// errors' messages.
type fanoutJob struct {
	ID      int64
	Args    notificationriver.FanoutArgs
	State   string
	Attempt int
	Errors  []string
}

// fanoutJobsOn reads the notification.fanout jobs of the work item's events, in the order they were inserted.
func fanoutJobsOn(t *testing.T, pool *pgxpool.Pool, item uuid.UUID) []fanoutJob {
	t.Helper()
	rows, err := pool.Query(pgtest.Soon(t), `SELECT id, args, state::text, attempt, array(SELECT e->>'error' FROM unnest(errors) AS e)
		FROM river_job WHERE kind = $1 AND args->>'entity_id' = $2 ORDER BY id`, notificationriver.FanoutKind, item.String())
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var jobs []fanoutJob
	for rows.Next() {
		var j fanoutJob
		var raw []byte
		if err := rows.Scan(&j.ID, &raw, &j.State, &j.Attempt, &j.Errors); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &j.Args); err != nil {
			t.Fatal(err)
		}
		jobs = append(jobs, j)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return jobs
}

// sentKey is what makes a notification one: its event, its receiver and its change's position (R4).
type sentKey struct {
	Event, Receiver uuid.UUID
	Seq             int16
}

// A fan-out retried or repeated sends each notification once (M7 P1b spec 2.9, R4), through the app's job. alice creates
// a work item assigned to bob and walt, which sends each the creation itself and both assignments; then renames it
// twice, which sends each the name's change, at the same position both times.
//
// The renaming's fan-out waits for the work item's row, which a transaction holds FOR NO KEY UPDATE, past the job
// transaction's lock_timeout, issue.write_lock_timeout's (bootstrap/gate.go newIssueJobTx), 300 ms in this world: its
// first attempt fails at the timeout (55P03). The transaction then lets go and River retries the job, after its
// backoff of a second (River's default retry policy: the attempt to the fourth power, in seconds), which sends the change
// once.
//
// Then each of the three jobs is run again (River's JobRetry), and its arguments, the same event, enqueued once more as
// a job of their own: every run completes, and the work item's notifications are the first ones, each once, with the
// same ids.
func TestAFanoutRetriedOrRepeatedSendsEachNotificationOnce(t *testing.T) {
	w := newHeldFanoutWorld(t, func(cfg *config.Config) { cfg.Issue.WriteLockTimeout = 300 * time.Millisecond })
	bob, walt := w.ids["bob"], w.ids["walt"]
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	item := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Fix","assignee_ids":%s}`, idArray(bob, walt)))
	path := "/api/v0/issues/" + item.String()
	releaseFanouts(t, w.pool)
	waitForFanouts(t, w.pool)

	w.send(t, "alice", http.MethodPatch, path, `{"name":"Fix it"}`, http.StatusOK)
	held := holding(t, w.pool, "SELECT 1 FROM issues WHERE id = $1 FOR NO KEY UPDATE", item)
	if n := releaseFanouts(t, w.pool); n != 1 {
		t.Fatalf("released %d fan-outs; want the renaming's", n)
	}
	pgtest.WaitForStatementLockWaitBehind(t, w.pool, held.Conn().PgConn(), 5*time.Second, "ShareIssue")
	requestEventuallyWithin(t, "the renaming's fan-out failing at its lock timeout", 5*time.Second, func(context.Context) bool {
		jobs := fanoutJobsOn(t, w.pool, item)
		return len(jobs) == 2 && len(jobs[1].Errors) > 0
	})
	if err := held.Rollback(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
	waitForFanouts(t, w.pool)
	w.send(t, "alice", http.MethodPatch, path, `{"name":"Fix it now"}`, http.StatusOK)
	releaseFanouts(t, w.pool)
	waitForFanouts(t, w.pool)

	jobs := fanoutJobsOn(t, w.pool, item)
	if len(jobs) != 3 || jobs[0].Args.Type != "issue.created" || jobs[1].Args.Type != "issue.updated" || jobs[2].Args.Type != "issue.updated" {
		t.Fatalf("the work item's fan-outs %+v; want its creation's and two renamings'", jobs)
	}
	for n, attempt := range []int{1, 2, 1} {
		if got := jobs[n]; got.State != "completed" || got.Attempt != attempt || len(got.Errors) != attempt-1 {
			t.Errorf("fan-out %d of %s: %s in attempt %d, errors %q; want completed in attempt %d", n, got.Args.Type, got.State, got.Attempt, got.Errors,
				attempt)
		}
	}
	if errs := jobs[1].Errors; len(errs) != 1 || !strings.Contains(errs[0], "lock timeout") || !strings.Contains(errs[0], "55P03") {
		t.Errorf("the renaming's first attempt failed with %q; want its lock timeout (55P03)", errs)
	}
	var retried time.Duration
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT attempted_at - (errors[1]->>'at')::timestamptz FROM river_job WHERE id = $1`, jobs[1].ID).Scan(&retried); err != nil {
		t.Fatal(err)
	}
	t.Logf("the renaming's fan-out failed at its lock timeout; River ran it again %s later", retried)
	var want []sentKey
	created := jobs[0].Args
	for _, receiver := range []uuid.UUID{bob, walt} {
		want = append(want, sentKey{created.EventID, receiver, -1})
		for _, assignee := range []uuid.UUID{bob, walt} {
			want = append(want, sentKey{created.EventID, receiver, assignedAt(t, created, assignee)})
		}
		want = append(want, sentKey{jobs[1].Args.EventID, receiver, 0}, sentKey{jobs[2].Args.EventID, receiver, 0})
	}
	events := []uuid.UUID{created.EventID, jobs[1].Args.EventID, jobs[2].Args.EventID}
	ids, sent := sentNotifications(t, w.pool, item, events...)
	if got := keysOf(sent, events); !reflect.DeepEqual(got, sortedKeys(want, events)) {
		t.Fatalf("the work item's notifications %+v; want each once, %+v", got, sortedKeys(want, events))
	}

	client, err := river.NewClient(riverpgxv5.New(w.pool), &river.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, job := range jobs {
		if _, err := client.JobRetry(pgtest.Soon(t), job.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := client.Insert(pgtest.Soon(t), job.Args, nil); err != nil {
			t.Fatal(err)
		}
	}
	if n := releaseFanouts(t, w.pool); n != len(jobs) {
		t.Fatalf("released %d fan-outs; want the %d jobs enqueued again", n, len(jobs))
	}
	waitForFanouts(t, w.pool)
	again := fanoutJobsOn(t, w.pool, item)
	if len(again) != 2*len(jobs) {
		t.Fatalf("the work item's fan-outs %+v; want each twice", again)
	}
	for n, job := range again {
		// The first three are the jobs run again, one attempt more and no new error; the last three, their copies.
		want := fanoutJob{ID: job.ID, Args: jobs[n%len(jobs)].Args, State: "completed", Attempt: 1, Errors: []string{}}
		if n < len(jobs) {
			want.Attempt, want.Errors = jobs[n].Attempt+1, jobs[n].Errors
		}
		if !reflect.DeepEqual(job, want) {
			t.Errorf("fan-out %d run again: %+v; want %+v", n, job, want)
		}
	}
	if againIDs, againSent := sentNotifications(t, w.pool, item, events...); !slices.Equal(againIDs, ids) || !reflect.DeepEqual(againSent, sent) {
		t.Errorf("the notifications after each fan-out ran again %v %+v; want the first ones alone, %v", againIDs, keysOf(againSent, events), ids)
	}
}

// keysOf are the notifications' keys; sortedKeys, keys in the order sentNotifications reads them.
func keysOf(sent []sentNotification, events []uuid.UUID) []sentKey {
	keys := make([]sentKey, len(sent))
	for i, n := range sent {
		keys[i] = sentKey{n.Event, n.Receiver, n.Seq}
	}
	return sortedKeys(keys, events)
}

func sortedKeys(keys []sentKey, events []uuid.UUID) []sentKey {
	return slices.SortedStableFunc(slices.Values(keys), func(a, b sentKey) int {
		return cmp.Or(cmp.Compare(slices.Index(events, a.Event), slices.Index(events, b.Event)), a.Receiver.Compare(b.Receiver), cmp.Compare(a.Seq, b.Seq))
	})
}
