package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/riverqueue/river"

	identityriver "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/river"
	issueriver "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A full queue snapshot mistakes a real worker finishing for a refused request
// writing. Keep the real Runner active and prove that only its progress changes.
func TestPublishSnapshotIgnoresPeriodicProgress(t *testing.T) {
	w := newIssueWorldWith(t, func(cfg *config.Config) { cfg.Issue.TrimVersionsInterval = time.Second })
	createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{}}`)
	holder := holding(t, w.pool, "SELECT 1 FROM projects WHERE id=$1 FOR NO KEY UPDATE", w.web)
	holderPID := holder.Conn().PgConn().PID()
	rawJobs := func() string {
		t.Helper()
		var rows string
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT coalesce(jsonb_agg(to_jsonb(j) ORDER BY id), '[]'::jsonb)::text FROM river_job j").Scan(&rows); err != nil {
			t.Fatal(err)
		}
		return rows
	}
	var jobID int64
	var waiterPID int
	requestEventuallyWithin(t, "real trim running and waiting for held project", 5*time.Second, func(ctx context.Context) bool {
		if err := w.pool.QueryRow(ctx, `SELECT coalesce((SELECT id FROM river_job WHERE kind='issue.trim_description_versions' AND state='running' ORDER BY id LIMIT 1),0),
          coalesce((SELECT pid FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND $1::int=ANY(pg_blocking_pids(pid)) AND query LIKE '-- name: LockIssueProject :one%' LIMIT 1),0)`, int64(holderPID)).Scan(&jobID, &waiterPID); err != nil {
			t.Fatal(err)
		}
		return jobID != 0 && waiterPID != 0
	})
	rawBefore := rawJobs()
	before := publishSnapshot(t, w)
	requestRelease(t, holder)
	requestEventuallyWithin(t, "the observed trim job completed", 5*time.Second, func(ctx context.Context) bool {
		var state string
		if err := w.pool.QueryRow(ctx, "SELECT state::text FROM river_job WHERE id=$1", jobID).Scan(&state); err != nil {
			t.Fatal(err)
		}
		return state == "completed"
	})
	after := publishSnapshot(t, w)
	if rawBefore == rawJobs() {
		t.Fatal("control: full River rows did not change after real worker completion")
	}
	var result string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT to_jsonb(j)::text FROM river_job j WHERE id=$1", jobID).Scan(&result); err != nil {
		t.Fatal(err)
	}
	t.Logf("held PID=%d waiting LockIssueProject PID=%d actual completed trim=%s", holderPID, waiterPID, result)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("periodic worker progress changed publishSnapshot: before=%v after=%v", before, after)
	}
}

// This kind is deliberately absent from EnqueuedKinds: a new or incorrectly
// wired request job must remain visible to the refusal/rollback assertions.
type snapshotJobArgs struct {
	JobKind string `json:"-"`
	Value   string `json:"value,omitempty"`
}

func (a snapshotJobArgs) Kind() string { return a.JobKind }

// Narrowing the snapshot to known request kinds would hide event probes and
// unknown jobs. Loosening the periodic signature would also hide misrouted or
// malformed enqueues. All inserts below go through the actual River inserter.
func TestPublishSnapshotKeepsRequestJobs(t *testing.T) {
	pool := openPool(t, pgtest.NewDatabase(t))
	w := issueWorld{pool: pool}
	inserter, err := jobs.NewInserter(pool)
	if err != nil {
		t.Fatal(err)
	}
	insert := func(t *testing.T, args river.JobArgs, queue, metadata string) int64 {
		t.Helper()
		tx, err := pool.Begin(pgtest.Soon(t))
		if err != nil {
			t.Fatal(err)
		}
		defer requestRelease(t, tx)
		if err := inserter.InsertTx(pgtest.Soon(t), tx, args, &river.InsertOpts{Queue: queue, Metadata: json.RawMessage(metadata), ScheduledAt: time.Now().Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(pgtest.Soon(t)); err != nil {
			t.Fatal(err)
		}
		var id int64
		if err := pool.QueryRow(pgtest.Soon(t), "SELECT max(id) FROM river_job").Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	exec := func(t *testing.T, sql string, id int64) {
		t.Helper()
		if _, err := pool.Exec(pgtest.Soon(t), sql, id); err != nil {
			t.Fatal(err)
		}
	}
	periodicMetadata := `{"periodic":true,"river:periodic_job_id":"issue.trim_description_versions"}`
	for _, tc := range []struct {
		name            string
		args            river.JobArgs
		queue, metadata string
	}{
		{"request", issueriver.CrawlLinkArgs{LinkID: uuid.NewV7(), URL: "https://example.com/first"}, "default", `{}`},
		{"event probe", issueEventProbeArgs{Event: shared.Event{ID: uuid.NewV7(), Type: "issue.created"}}, "issue_events", `{}`},
		{"unknown", snapshotJobArgs{JobKind: "issue_test.unknown", Value: "retained"}, "default", `{}`},
		{"unknown periodic", snapshotJobArgs{JobKind: "issue_test.unknown"}, "periodic", `{"periodic":true,"river:periodic_job_id":"issue_test.unknown"}`},
		{"wrong queue", issueriver.TrimVersionsArgs{}, "default", periodicMetadata},
		{"wrong args", snapshotJobArgs{JobKind: "issue.trim_description_versions", Value: "unexpected"}, "periodic", periodicMetadata},
		{"missing metadata", issueriver.TrimVersionsArgs{}, "periodic", `{}`},
		{"missing periodic flag", issueriver.TrimVersionsArgs{}, "periodic", `{"river:periodic_job_id":"issue.trim_description_versions"}`},
		{"wrong periodic flag type", issueriver.TrimVersionsArgs{}, "periodic", `{"periodic":"true","river:periodic_job_id":"issue.trim_description_versions"}`},
		{"wrong schedule id", issueriver.TrimVersionsArgs{}, "periodic", `{"periodic":true,"river:periodic_job_id":"other"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := publishSnapshot(t, w)
			id := insert(t, tc.args, tc.queue, tc.metadata)
			inserted := publishSnapshot(t, w)
			if reflect.DeepEqual(before, inserted) {
				t.Fatal("inserted job disappeared from snapshot")
			}
			exec(t, `UPDATE river_job SET args='{"changed":true}'::jsonb WHERE id=$1`, id)
			if reflect.DeepEqual(inserted, publishSnapshot(t, w)) {
				t.Fatal("replacement payload disappeared from snapshot")
			}
			exec(t, "DELETE FROM river_job WHERE id=$1", id)
			if after := publishSnapshot(t, w); !reflect.DeepEqual(before, after) {
				t.Fatalf("deleted job still affects snapshot: %v", after)
			}
		})
	}
	for _, args := range []river.JobArgs{identityriver.CleanupArgs{}, issueriver.AutoArchiveArgs{}, issueriver.TrimVersionsArgs{}, purgeArgs{}} {
		t.Run("legitimate periodic/"+args.Kind(), func(t *testing.T) {
			before := publishSnapshot(t, w)
			id := insert(t, args, "periodic", fmt.Sprintf(`{"periodic":true,"river:periodic_job_id":%q}`, args.Kind()))
			if !reflect.DeepEqual(before, publishSnapshot(t, w)) {
				t.Fatal("legitimate periodic enqueue polluted request snapshot")
			}
			exec(t, "UPDATE river_job SET args='null'::jsonb WHERE id=$1", id)
			if reflect.DeepEqual(before, publishSnapshot(t, w)) {
				t.Fatal("JSON null payload was mistaken for a legitimate periodic job")
			}
			exec(t, "DELETE FROM river_job WHERE id=$1", id)
		})
	}
	id := insert(t, issueriver.CrawlLinkArgs{LinkID: uuid.NewV7(), URL: "https://example.com/stable"}, "default", `{}`)
	for _, assignment := range []string{
		"kind='issue_test.replaced'", "queue='different'", "max_attempts=max_attempts+1", "priority=2", "tags=ARRAY['changed']", "created_at=created_at+interval '1 second'",
	} {
		t.Run("stable column/"+assignment, func(t *testing.T) {
			before := publishSnapshot(t, w)
			exec(t, "UPDATE river_job SET "+assignment+" WHERE id=$1", id)
			if reflect.DeepEqual(before, publishSnapshot(t, w)) {
				t.Fatal("enqueue property change disappeared from snapshot")
			}
		})
	}
	before := publishSnapshot(t, w)
	exec(t, "UPDATE river_job SET id=id+10000 WHERE id=$1", id)
	id += 10000
	if reflect.DeepEqual(before, publishSnapshot(t, w)) {
		t.Fatal("job identity change disappeared from snapshot")
	}
	before = publishSnapshot(t, w)
	exec(t, `UPDATE river_job SET state='completed',attempt=2,attempted_at=now(),finalized_at=now(),attempted_by=ARRAY['worker'],errors=ARRAY['{"error":"retry"}'::jsonb],scheduled_at=now(),metadata='{"worker":"progress"}' WHERE id=$1`, id)
	if after := publishSnapshot(t, w); !reflect.DeepEqual(before, after) {
		t.Fatalf("request job worker progress polluted snapshot: %v", after)
	}
}
