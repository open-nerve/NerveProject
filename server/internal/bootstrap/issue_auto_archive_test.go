package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	issuepostgres "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func autoArchiveOn(w issueWorld, c issueapp.Clock, events shared.EventPublisher) *issueapp.AutoArchive {
	p := project.Provide(w.pool)
	return issueapp.NewAutoArchive(issueapp.AutoArchiveDeps{Policies: p.ArchivePolicies, Projects: issueProjects{projects: p.IssueProjects}, States: issueProjects{projects: p.IssueProjects}, Issues: issuepostgres.New(w.pool), Tx: newIssueJobTx(w.pool, config.Config{Database: config.DatabaseConfig{CommitTimeout: 2 * time.Second}, Issue: testIssueConfig}), Clock: c, Events: events})
}

// The literal dates distinguish 30-day months from calendar months and 31 days.
// Complete row snapshots protect hierarchy, creator, state and all other fields.
func TestAutoArchiveDatabaseBoundariesAndNullActor(t *testing.T) {
	w := newIssueWorld(t)
	at := time.Date(2044, 3, 31, 10, 0, 0, 123456000, time.UTC)
	cutoff := time.Date(2044, 3, 1, 10, 0, 0, 123456000, time.UTC)
	var cancelled uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT id FROM states WHERE project_id=$1 AND "group"='cancelled' AND deleted_at IS NULL`, w.web).Scan(&cancelled); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		state   uuid.UUID
		updated time.Time
		match   bool
	}{
		{"second before", w.done, cutoff.Add(-time.Second), true}, {"microsecond before", w.done, cutoff.Add(-time.Microsecond), true}, {"exact", w.done, cutoff, true}, {"microsecond after", w.done, cutoff.Add(time.Microsecond), false}, {"second after", w.done, cutoff.Add(time.Second), false}, {"cancelled", cancelled, cutoff, true}, {"todo", w.todo, cutoff.Add(-time.Hour), false},
	}
	var ids, want []uuid.UUID
	before := map[uuid.UUID]string{}
	for _, c := range cases {
		id := w.newWorkItem(t)
		ids = append(ids, id)
		if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issues SET state_id=$2,updated_at=$3 WHERE id=$1", id, c.state, c.updated); err != nil {
			t.Fatal(err)
		}
		if len(ids) == 2 {
			if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issues SET parent_id=$2 WHERE id=$1", id, ids[0]); err != nil {
				t.Fatal(err)
			}
		}
		before[id] = archiveStableRow(t, w, id)
		if c.match {
			want = append(want, id)
		}
	}
	var subscribersBefore int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM issue_subscribers").Scan(&subscribersBefore); err != nil {
		t.Fatal(err)
	}
	var badManual int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM issue_activities WHERE project_id=$1 AND actor_id IS DISTINCT FROM $2", w.web, w.ids["alice"]).Scan(&badManual); err != nil || badManual != 0 {
		t.Fatalf("manual actor lost: %d,%v", badManual, err)
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE projects SET archive_in=1 WHERE id=$1", w.web); err != nil {
		t.Fatal(err)
	}
	var received []shared.Event
	events := archiveDispatcher(t, w, &received, nil)
	clock := &eventCountingClock{base: clocktest.At(at)}
	uc := autoArchiveOn(w, clock, events)
	n, err := uc.Execute(pgtest.Soon(t))
	if err != nil || n != 4 {
		t.Fatalf("Execute = %d,%v", n, err)
	}
	slices.SortFunc(want, uuid.UUID.Compare)
	if clock.calls.Load() != 1 || len(received) != len(want) {
		t.Fatalf("clock/events %d/%d", clock.calls.Load(), len(received))
	}
	for i, e := range received {
		if e.Entity.ID != want[i] || e.ActorID != nil || !e.OccurredAt.Equal(at) || e.WorkspaceID != w.acme || e.ProjectID != w.web || !reflect.DeepEqual(e.Changes, []shared.Change{{Field: "archived_at", New: eventValue("archive")}}) {
			t.Errorf("system event=%+v", e)
		}
	}
	for i, id := range ids {
		if got := archiveStableRow(t, w, id); got != before[id] {
			t.Errorf("%s changed non archival columns", cases[i].name)
		}
		var archived *time.Time
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT archived_at FROM issues WHERE id=$1", id).Scan(&archived); err != nil {
			t.Fatal(err)
		}
		if cases[i].match {
			if archived == nil || !archived.Equal(at) {
				t.Errorf("%s archived at %v", cases[i].name, archived)
			}
		} else if archived != nil {
			t.Errorf("%s wrongly archived", cases[i].name)
		}
	}
	var subscribersAfter int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM issue_subscribers").Scan(&subscribersAfter); err != nil || subscribersAfter != subscribersBefore {
		t.Fatalf("subscribers changed: %d -> %d,%v", subscribersBefore, subscribersAfter, err)
	}
	var badSystem, count int
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*),count(*) FILTER (WHERE actor_id IS NOT NULL OR created_at<>$2 OR updated_at<>$2) FROM issue_activities WHERE project_id=$1 AND new_value='archive'`, w.web, at).Scan(&count, &badSystem); err != nil || count != 4 || badSystem != 0 {
		t.Fatalf("system activities=%d invalid=%d err=%v", count, badSystem, err)
	}
	if n, err := uc.Execute(pgtest.Soon(t)); err != nil || n != 0 || len(received) != 4 || issueEventCount(t, w) != 4 {
		t.Fatalf("retry=%d,%v events=%d", n, err, len(received))
	}
}
func archiveStableRow(t *testing.T, w issueWorld, id uuid.UUID) string {
	t.Helper()
	var text string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT (to_jsonb(i)-'archived_at'-'updated_at'-'updated_by_id')::text FROM issues i WHERE id=$1`, id).Scan(&text); err != nil {
		t.Fatal(err)
	}
	return text
}

// A subscriber inserts a real River job after observing the complete write.
// Its error must roll back the UPDATE, activities and inserted job together.
func archiveDispatcher(t *testing.T, w issueWorld, received *[]shared.Event, fail error) shared.EventPublisher {
	t.Helper()
	client, err := jobs.NewInserter(w.pool)
	if err != nil {
		t.Fatal(err)
	}
	d, err := newDispatcher(issue.EventTypes(), []shared.EventSubscriber{{Name: "archive probe", Types: []string{"issue.archived"}, Handle: func(ctx context.Context, e shared.Event) error {
		var pending int
		if err := postgres.DB(ctx, w.pool).QueryRow(ctx, `SELECT count(*) FROM issues i WHERE i.project_id=$1 AND i.archived_at=$2 AND NOT EXISTS (SELECT 1 FROM issue_activities a WHERE a.issue_id=i.id AND a.new_value='archive' AND a.actor_id IS NULL AND a.created_at=$2)`, e.ProjectID, e.OccurredAt).Scan(&pending); err != nil {
			return err
		}
		if pending != 0 {
			return fmt.Errorf("event before all activities: %d", pending)
		}
		*received = append(*received, e)
		if err := (issueInserter{client: client}).InsertTx(ctx, issueEventProbeArgs{Event: e}, &river.InsertOpts{Queue: "issue_events"}); err != nil {
			return err
		}
		return fail
	}}})
	if err != nil {
		t.Fatal(err)
	}
	return d
}
func TestAutoArchivePublishFailureRollsBackRowsActivitiesAndJob(t *testing.T) {
	w := newIssueWorld(t)
	id := w.newWorkItem(t)
	at := time.Date(2044, 3, 31, 10, 0, 0, 123456000, time.UTC)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issues SET state_id=$2,updated_at=$3 WHERE id=$1", id, w.done, at.Add(-40*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE projects SET archive_in=1 WHERE id=$1", w.web); err != nil {
		t.Fatal(err)
	}
	before := issueSnapshot(t, w, id)
	sentinel := errors.New("subscriber rejects archive")
	var received []shared.Event
	uc := autoArchiveOn(w, clocktest.At(at), archiveDispatcher(t, w, &received, sentinel))
	n, err := uc.Execute(pgtest.Soon(t))
	if n != 0 || !errors.Is(err, sentinel) || len(received) != 1 {
		t.Fatalf("Execute = %d,%v events=%d", n, err, len(received))
	}
	if after := issueSnapshot(t, w, id); after != before || issueEventCount(t, w) != 0 {
		t.Fatal("failed event committed issue/activity/job")
	}
	// A retry succeeds exactly once after the subscriber recovers.
	uc = autoArchiveOn(w, clocktest.At(at), archiveDispatcher(t, w, &received, nil))
	if n, err := uc.Execute(pgtest.Soon(t)); n != 1 || err != nil || issueEventCount(t, w) != 1 {
		t.Fatalf("retry=%d,%v", n, err)
	}
}
func issueSnapshot(t *testing.T, w issueWorld, id uuid.UUID) string {
	t.Helper()
	var text string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT jsonb_build_array(to_jsonb(i),(SELECT jsonb_agg(to_jsonb(a) ORDER BY a.id) FROM issue_activities a WHERE a.issue_id=i.id))::text FROM issues i WHERE id=$1`, id).Scan(&text); err != nil {
		t.Fatal(err)
	}
	return text
}

// The worker's real lock wait must keep postgres.LockNotAvailable's chain,
// while the request transaction continues mapping the same failure to 503.
func TestAutoArchiveJobLockTimeoutPreservesPostgresError(t *testing.T) {
	w := newIssueWorld(t)
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"archive_in":1}`, 200)
	holder := holding(t, w.pool, "SELECT 1 FROM projects WHERE id=$1 FOR UPDATE", w.web)
	defer release(t, holder)
	p := project.Provide(w.pool)
	cfg := config.Config{Database: config.DatabaseConfig{CommitTimeout: 2 * time.Second}, Issue: config.IssueConfig{WriteLockTimeout: 300 * time.Millisecond}}
	c := &eventCountingClock{base: clocktest.At(time.Now())}
	probe := &requestProbe{TxManager: newIssueJobTx(w.pool, cfg)}
	uc := issueapp.NewAutoArchive(issueapp.AutoArchiveDeps{Policies: p.ArchivePolicies, Projects: issueProjects{projects: p.IssueProjects}, States: issueProjects{projects: p.IssueProjects}, Issues: issuepostgres.New(w.pool), Tx: probe, Clock: c, Events: discardIssueEvents{}})
	result := make(chan error, 1)
	go func() {
		result <- riveradapter.NewAutoArchiveWorker(uc).Work(context.Background(), &river.Job[riveradapter.AutoArchiveArgs]{})
	}()
	requestBlocked(t, w.pool, probe, holder.Conn().PgConn().PID(), "projects")
	select {
	case err := <-result:
		if !postgres.LockNotAvailable(err) || isBusy(err) {
			t.Fatalf("job error=%v; expected original PG lock error", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("job did not time out")
	}
	if c.calls.Load() != 0 {
		t.Fatal("clock read before lock acquired")
	}
}

// The second project's failure cannot undo a committed first project; a retry
// inserts no second activity or job for the already archived first issue.
func TestAutoArchiveDatabasePartialCommitAndRetry(t *testing.T) {
	w := newIssueWorld(t)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	ids := []uuid.UUID{w.newWorkItem(t), w.newWorkItemIn(t, ops, `{"name":"Ops issue"}`)}
	projects := []uuid.UUID{w.web, ops}
	states := []uuid.UUID{w.done, stateID(t, w.pool, ops, "Done")}
	at := time.Date(2044, 3, 31, 10, 0, 0, 123456000, time.UTC)
	for i, id := range ids {
		if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issues SET state_id=$2,updated_at=$3 WHERE id=$1", id, states[i], at.Add(-40*24*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE projects SET archive_in=1 WHERE id=ANY($1::uuid[])", projects); err != nil {
		t.Fatal(err)
	}
	sentinel := errors.New("second project fails")
	var received []shared.Event
	publisher := archiveProjectFailure{EventPublisher: archiveDispatcher(t, w, &received, nil), project: ops, err: sentinel}
	uc := autoArchiveOn(w, clocktest.At(at), publisher)
	if n, err := uc.Execute(pgtest.Soon(t)); n != 1 || !errors.Is(err, sentinel) || issueEventCount(t, w) != 1 {
		t.Fatalf("partial commit=%d,%v jobs=%d", n, err, issueEventCount(t, w))
	}
	for i, id := range ids {
		var archived bool
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT archived_at IS NOT NULL FROM issues WHERE id=$1", id).Scan(&archived); err != nil || archived != (i == 0) {
			t.Fatalf("issue %d archived=%v err=%v", i, archived, err)
		}
	}
	uc = autoArchiveOn(w, clocktest.At(at), archiveDispatcher(t, w, &received, nil))
	if n, err := uc.Execute(pgtest.Soon(t)); n != 1 || err != nil || issueEventCount(t, w) != 2 {
		t.Fatalf("retry=%d,%v jobs=%d", n, err, issueEventCount(t, w))
	}
	var count int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM issue_activities WHERE issue_id=ANY($1::uuid[]) AND new_value='archive'", ids).Scan(&count); err != nil || count != 2 {
		t.Fatalf("activities=%d,%v", count, err)
	}
}

type archiveProjectFailure struct {
	shared.EventPublisher
	project uuid.UUID
	err     error
}

func (p archiveProjectFailure) Publish(ctx context.Context, e shared.Event) error {
	if err := p.EventPublisher.Publish(ctx, e); err != nil {
		return err
	}
	if e.ProjectID == p.project {
		return p.err
	}
	return nil
}
