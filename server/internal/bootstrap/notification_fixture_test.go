package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification"
	notificationriver "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// seedNotification writes, directly, one notification of the work item item
// for receiver, nerve's own (an automatic archive's, M7 P1b spec 2.3), in
// the item's workspace and project and with its creator, deleted when and
// as the item is: a live item's notification is live, nerve's writing, with
// no audit account; a deleted item's is deleted at its instant by its
// deleter, deleted_at, updated_at and updated_by_id as the item's, as the
// item's deletion deletes its notifications (spec 2.2, 2.8). The deletion
// and purge guards write the notification module's rows as fixtures.
func seedNotification(t *testing.T, pool *pgxpool.Pool, item, receiver uuid.UUID) {
	t.Helper()
	if tag, err := pool.Exec(pgtest.Soon(t), `INSERT INTO notifications (id, workspace_id, project_id, receiver_id, issue_id, issue_created_by_id, sender, data,
			event_id, change_seq, updated_by_id, created_at, updated_at, deleted_at)
		SELECT $1, workspace_id, project_id, $2, id, created_by_id, 'subscribed',
			jsonb_build_object('issue', jsonb_build_object('id', id, 'name', name, 'sequence_id', sequence_id),
				'change', jsonb_build_object('field', 'archived_at', 'verb', 'updated', 'new_value', 'archive')),
			$3, 0, CASE WHEN deleted_at IS NOT NULL THEN updated_by_id END, coalesce(deleted_at, now()), coalesce(deleted_at, now()), deleted_at
		FROM issues WHERE id = $4`, uuid.NewV7(), receiver, uuid.NewV7(), item); err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("seeding a notification of %s: %d rows, %v; want 1", item, tag.RowsAffected(), err)
	}
}

// seedItemNotification is seedNotification of the project's work item
// named name, which seedIssues writes.
func seedItemNotification(t *testing.T, pool *pgxpool.Pool, project uuid.UUID, name string, receiver uuid.UUID) {
	t.Helper()
	items := queryIDs(t, pool, "SELECT id FROM issues WHERE project_id = $1 AND name = $2", project, name)
	if len(items) != 1 {
		t.Fatalf("the work items %s of %s: %v; want one", name, project, items)
	}
	seedNotification(t, pool, items[0], receiver)
}

// notificationJob is a deleted project's notification.delete_rows job as
// River holds it.
type notificationJob struct {
	ID                         int64
	State                      string
	Attempt, Failures, Snoozes int
}

// notificationDeletionJob reads the one notification.delete_rows job of the
// deleted project, false while there is none: the project's deletion
// moment and deleter its arguments, on the default queue with River's 25
// attempts. A second job fails the test.
func notificationDeletionJob(ctx context.Context, t *testing.T, pool *pgxpool.Pool, project uuid.UUID) (notificationJob, bool) {
	t.Helper()
	var at time.Time
	var by uuid.UUID
	if err := pool.QueryRow(ctx, "SELECT deleted_at,updated_by_id FROM projects WHERE id=$1", project).Scan(&at, &by); err != nil {
		t.Fatal(err)
	}
	rows, err := pool.Query(ctx, `SELECT id,args,state::text,attempt,coalesce(cardinality(errors),0),coalesce((metadata->>'snoozes')::int,0),queue,max_attempts
		FROM river_job WHERE kind=$1 AND args->>'project_id'=$2`, notificationriver.DeleteRowsKind, project.String())
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var found []notificationJob
	for rows.Next() {
		var job notificationJob
		var raw []byte
		var queue string
		var maxAttempts int
		if err := rows.Scan(&job.ID, &raw, &job.State, &job.Attempt, &job.Failures, &job.Snoozes, &queue, &maxAttempts); err != nil {
			t.Fatal(err)
		}
		var args notificationriver.DeleteRowsArgs
		var fields map[string]any
		if err := json.Unmarshal(raw, &args); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		if args.ProjectID != project || !args.At.Equal(at) || args.By != by || len(fields) != 3 || queue != "default" || maxAttempts != 25 {
			t.Fatalf("notification deletion job lost the deletion's context: %s %s/%d; want project %s at %s by %s", raw, queue, maxAttempts, project, at, by)
		}
		found = append(found, job)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(found) > 1 {
		t.Fatalf("project %s notification deletion jobs=%d want exactly one", project, len(found))
	}
	if len(found) == 0 {
		return notificationJob{}, false
	}
	return found[0], true
}

// waitNotificationDeletion waits for the one notification.delete_rows job
// of the deleted project (notificationDeletionJob) to complete in its first
// attempt without a failure: its snoozes while the project's work items
// were being deleted consumed no attempt.
func waitNotificationDeletion(t *testing.T, pool *pgxpool.Pool, project uuid.UUID) {
	t.Helper()
	requestEventuallyWithin(t, "notification deletion completed", 15*time.Second, func(ctx context.Context) bool {
		job, ok := notificationDeletionJob(ctx, t, pool, project)
		if !ok {
			return false
		}
		if job.Failures != 0 || job.Attempt > 1 || job.State == "cancelled" || job.State == "discarded" {
			t.Fatalf("notification deletion job failed: %+v", job)
		}
		return job.State == "completed"
	})
}

// notificationDeps are the notification module's dependencies over pool as
// newApp builds them, with cfg's job settings: the issue, the project and
// the access modules' ports, each from its own Provide, converted.
func notificationDeps(pool *pgxpool.Pool, cfg config.Config) notification.Deps {
	workspaces, projects := workspace.Provide(pool, nil), project.Provide(pool, nil)
	decider := access.NewDecider(access.DeciderDeps{WorkspaceRoles: workspaces.WorkspaceRolesMany,
		ProjectAccess: accessProjectsMany{projects: projects.ProjectAccessMany}})
	return notification.Deps{Pool: pool, Issues: notificationIssues{issues: issue.Provide(pool, nil).NotificationIssues},
		Projects: notificationProjects{projects: projects.IssueProjects}, Readers: notificationReaders{decider: decider},
		JobTx: newIssueJobTx(pool, cfg), RuntimeNow: time.Now, Logger: slog.New(slog.DiscardHandler),
		DeleteBatch: cfg.Notification.DeleteBatch, DeleteBudget: cfg.Notification.DeleteBudget}
}

// notificationJobsForTest are the notification module's request workers as
// newApp registers them, for a test that runs its own Runner in place of the
// app's: the writes it publishes enqueue notification.fanout, and its
// deletion events notification.delete_rows too.
func notificationJobsForTest(pool *pgxpool.Pool, cfg config.Config) []jobs.Job {
	return notification.New(notificationDeps(pool, cfg)).Jobs()
}

// waitForFanouts returns once every notification.fanout job inserted so far
// has completed: the writes of a fixture each enqueue the fan-out of their
// notifications (M7 P1b spec 2.9), which locks the work item's row FOR SHARE
// and inserts notifications, its foreign keys locking their workspace's,
// project's and receivers' rows FOR KEY SHARE. A test that then holds such a
// row or the issues table, observes their locks, or reads River's pending
// jobs, waits for them first, as for the start-time runs
// (waitForStartupRuns), so that no pending fan-out queues on what it holds,
// fails at a lock timeout meanwhile or shows among its locks (R10). The
// failure lists the fan-outs not completed.
//
// While fan-outs wait to be fetched it wakes the app's producer itself
// (nudgeFanouts), at most once per River's fetch cooldown. A fan-out done
// still reads running until River's batch completer writes its state, which
// it does every 250 ms.
func waitForFanouts(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	var nudged time.Time
	for {
		var left string
		if err := pool.QueryRow(pgtest.Soon(t), `SELECT coalesce(string_agg(format('%s %s attempt %s errors %s', id, state, attempt, coalesce(to_json(errors)::text, '[]')), E'\n' ORDER BY id), '')
			FROM river_job WHERE kind = $1 AND state <> 'completed'`, notificationriver.FanoutKind).Scan(&left); err != nil {
			t.Fatal(err)
		}
		if left == "" {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("notification fan-outs not completed within 15s:\n%s", left)
		}
		if time.Since(nudged) >= river.FetchCooldownDefault {
			nudgeFanouts(t, pool)
			nudged = time.Now()
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// nudgeFanouts wakes the app's producer for each queue an available fan-out
// is on: a River client sends a queue's insert notification at most once per
// FetchCooldown (100 ms), so a fan-out inserted within 100 ms of another job
// of its queue sends none and waits for the producer's poll
// (FetchPollInterval, 1 s), half a second on average (P1b T10's
// measurement); a job made due by an update sends none at all. The nudge
// copies River v0.47's internals: the channel "<current schema>.river_insert"
// (riverpgxv5's PGNotifyMany and Listener; the app sets no schema) and the
// payload {"queue": …} the producer decodes. Should River change them, the
// nudge wakes nothing and the fan-outs wait for the poll: slower, no
// different.
func nudgeFanouts(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(pgtest.Soon(t), `SELECT pg_notify(current_schema() || '.river_insert', json_build_object('queue', queue)::text)
		FROM (SELECT DISTINCT queue FROM river_job WHERE kind = $1 AND state = 'available') AS waiting`, notificationriver.FanoutKind); err != nil {
		t.Fatal(err)
	}
}

// newHeldFanoutWorld is newIssueWorldWith, configure's, whose fan-outs wait
// until the test releases them (holdFanouts): when each runs, against what
// the test holds, changes or deletes, is the test's to set. The hold is put
// in place before the app starts, so that the app's cleanup stops its
// workers before the hold's cleanup removes it.
func newHeldFanoutWorld(t *testing.T, configure func(*config.Config)) issueWorld {
	t.Helper()
	dbURL := pgtest.NewDatabase(t)
	pool := openPool(t, dbURL)
	holdFanouts(t, pool)
	return newIssueWorldOn(t, dbURL, pool, configure)
}

// holdFanouts has every notification.fanout job inserted into pool's
// database wait until releaseFanouts lets it run: a trigger schedules it a
// day ahead as it is inserted, and River's producer fetches only the jobs
// that are due. The job is otherwise the app's own, which its subscriber
// inserts in the write's transaction, its arguments untouched (M7 P1b spec
// 2.9).
func holdFanouts(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, sql := range []string{
		`CREATE SCHEMA fanout_hold`,
		`CREATE FUNCTION fanout_hold.later() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN NEW.scheduled_at = clock_timestamp() + interval '1 day'; RETURN NEW; END $$`,
		fmt.Sprintf(`CREATE TRIGGER hold_fanouts BEFORE INSERT ON river_job FOR EACH ROW WHEN (NEW.kind = '%s') EXECUTE FUNCTION fanout_hold.later()`,
			notificationriver.FanoutKind),
	} {
		if _, err := pool.Exec(pgtest.Soon(t), sql); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := pool.Exec(ctx, `DROP TRIGGER hold_fanouts ON river_job; DROP SCHEMA fanout_hold CASCADE`); err != nil {
			t.Error(err)
		}
	})
}

// releaseFanouts lets every fan-out held so far run (holdFanouts), wakes
// the app's producer to fetch them (nudgeFanouts), and answers how many it
// released. A retry River scheduled is not held, and is left to its time.
func releaseFanouts(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	tag, err := pool.Exec(pgtest.Soon(t), `UPDATE river_job SET scheduled_at = clock_timestamp()
		WHERE kind = $1 AND state = 'available' AND scheduled_at > clock_timestamp() + interval '1 hour'`, notificationriver.FanoutKind)
	if err != nil {
		t.Fatal(err)
	}
	nudgeFanouts(t, pool)
	return int(tag.RowsAffected())
}
