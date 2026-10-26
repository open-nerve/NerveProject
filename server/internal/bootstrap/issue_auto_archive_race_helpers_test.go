package bootstrap

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	issuepostgres "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Keep the real catalogue read and stop before the job writes any issue. In
// particular, workspace deletion can now hold workspace N while waiting for
// project N, before the archive activity's FK takes workspace KEY SHARE.
type archiveHeldStates struct {
	issueapp.StateCatalog
	gate *gate
}

func (s archiveHeldStates) States(ctx context.Context, id uuid.UUID) ([]issueapp.StateRef, error) {
	rows, err := s.StateCatalog.States(ctx, id)
	if err == nil {
		err = s.gate.wait(ctx)
	}
	return rows, err
}

// Pause after the real candidate scan; policy changes here must be reread by
// the real project lock, rather than being trusted from candidate discovery.
type archiveHeldPolicies struct {
	issueapp.ArchivePolicies
	gate *gate
}

func (p archiveHeldPolicies) ArchiveProjects(ctx context.Context) ([]uuid.UUID, error) {
	ids, err := p.ArchivePolicies.ArchiveProjects(ctx)
	if err == nil {
		err = p.gate.wait(ctx)
	}
	return ids, err
}

func archiveRaceJob(w issueWorld, c issueapp.Clock, e shared.EventPublisher, scanned, locked *gate) *issueapp.AutoArchive {
	p := project.Provide(w.pool)
	var policies issueapp.ArchivePolicies = p.ArchivePolicies
	var states issueapp.StateCatalog = issueProjects{projects: p.IssueProjects}
	if scanned != nil {
		policies = archiveHeldPolicies{policies, scanned}
	}
	if locked != nil {
		states = archiveHeldStates{states, locked}
	}
	return issueapp.NewAutoArchive(issueapp.AutoArchiveDeps{Policies: policies, Projects: issueProjects{projects: p.IssueProjects}, States: states, Issues: issuepostgres.New(w.pool), Tx: newIssueJobTx(w.pool, config.Config{Database: config.DatabaseConfig{CommitTimeout: 2 * time.Second}, Issue: testIssueConfig}), Clock: c, Events: e})
}

// Even broken lock mutants may publish from overlapping goroutines. Serialize
// the existing transactional subscriber and copy its observations under mu.
type archiveRaceEvents struct {
	mu        sync.Mutex
	events    []shared.Event
	publisher shared.EventPublisher
}

func newArchiveRaceEvents(t *testing.T, w issueWorld) *archiveRaceEvents {
	e := &archiveRaceEvents{}
	e.publisher = archiveDispatcher(t, w, &e.events, nil)
	return e
}
func newArchiveStateEvents(t *testing.T) *archiveRaceEvents {
	t.Helper()
	e := &archiveRaceEvents{}
	d, err := newDispatcher(issue.EventTypes(), []shared.EventSubscriber{{Name: "state change observer", Types: []string{"issue.updated"}, Handle: func(_ context.Context, event shared.Event) error {
		e.events = append(e.events, event) // Publish owns mu, even under lock mutants.
		return nil
	}}})
	if err != nil {
		t.Fatal(err)
	}
	e.publisher = d
	return e
}

func (e *archiveRaceEvents) Publish(ctx context.Context, event shared.Event) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.publisher.Publish(ctx, event)
}
func (e *archiveRaceEvents) snapshot() []shared.Event {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]shared.Event(nil), e.events...)
}

func archiveRaceSeed(t *testing.T, w issueWorld, at time.Time, age time.Duration) uuid.UUID {
	t.Helper()
	id := w.newWorkItemIn(t, w.web, `{"name":"Archive interleaving"}`)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issues SET state_id=$2,updated_at=$3 WHERE id=$1", id, w.done, at.Add(-age)); err != nil {
		t.Fatal(err)
	}
	w.send(t, "alice", "PATCH", "/api/v0/projects/"+w.web.String(), `{"archive_in":1}`, 200)
	return id
}

// Observe the actual waiter, but also allow a broken implementation to finish
// early. Callers still run their final business assertions in either case;
// removal of a lock must not merely turn into a synchronization timeout.
func archiveWaitOrDone(t *testing.T, w issueWorld, done <-chan error) (string, bool, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case err := <-done:
			return "", true, err
		default:
		}
		var table string
		err := w.pool.QueryRow(ctx, `SELECT coalesce((SELECT c.relname FROM pg_stat_activity a JOIN pg_locks l ON l.pid=a.pid JOIN pg_class c ON c.oid=l.relation WHERE a.datname=current_database() AND a.wait_event_type='Lock' AND l.locktype='tuple' LIMIT 1),'')`).Scan(&table)
		if err != nil {
			t.Fatal(err)
		}
		if table != "" {
			return table, false, nil
		}
		select {
		case err := <-done:
			return "", true, err
		case <-tick.C:
		case <-ctx.Done():
			t.Fatal("archive interleaving neither completed nor waited")
		}
	}
}

// All gates are opened and goroutines joined on failed assertions too.
func archiveInterleave(t *testing.T, w issueWorld, g *gate, first, second func() error, whileHeld, afterWait func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var a, b <-chan error
	defer func() {
		select {
		case <-g.open:
		default:
			close(g.open)
		}
		for _, done := range []<-chan error{a, b} {
			if done != nil {
				if err := result(t, ctx, done, "archive cleanup"); err != nil {
					t.Error(err)
				}
			}
		}
	}()
	a = run(first)
	held(t, ctx, g, a, "first archive interleaving side")
	if whileHeld != nil {
		whileHeld()
	}
	b = run(second)
	table, ended, err := archiveWaitOrDone(t, w, b)
	if ended {
		b = nil
	}
	if err != nil {
		t.Error(err)
	}
	if afterWait != nil {
		afterWait()
	}
	close(g.open)
	firstErr := result(t, ctx, a, "first archive interleaving side")
	a = nil
	var secondErr error
	if b != nil {
		secondErr = result(t, ctx, b, "second archive interleaving side")
		b = nil
	}
	if err := errors.Join(firstErr, secondErr); err != nil {
		t.Fatal(err)
	}
	// Defer this structural assertion until both operations have completed, so
	// their business results remain available even to deliberately broken locks.
	if table != "projects" {
		t.Errorf("serialization row=%q, early completion=%t; want projects", table, ended)
	}
}

func archiveRaceCheck(t *testing.T, w issueWorld, id uuid.UUID, at time.Time, n int, e *archiveRaceEvents, deleted bool) {
	t.Helper()
	var archived *time.Time
	var valid bool
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT archived_at, CASE WHEN $3 THEN deleted_at=updated_at AND updated_by_id=$4 ELSE updated_at=$2 AND updated_by_id IS NULL END FROM issues WHERE id=$1`, id, at, deleted, w.ids["alice"]).Scan(&archived, &valid); err != nil {
		t.Fatal(err)
	}
	if n == 0 && archived != nil || n == 1 && (archived == nil || !archived.Equal(at) || !valid) {
		t.Errorf("archive row: archived=%v audit=%t want archives=%d at=%v", archived, valid, n, at)
	}
	var count, bad int
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*),count(*) FILTER (WHERE actor_id IS NOT NULL OR created_at<>$2 OR (NOT $3 AND (updated_at<>$2 OR updated_by_id IS NOT NULL)) OR ($3 AND (deleted_at IS NULL OR deleted_at<>updated_at OR updated_by_id<>$4))) FROM issue_activities WHERE issue_id=$1 AND field='archived_at' AND new_value='archive'`, id, at, deleted, w.ids["alice"]).Scan(&count, &bad); err != nil {
		t.Fatal(err)
	}
	events := e.snapshot()
	if count != n || bad != 0 || len(events) != n || issueEventCount(t, w) != n {
		t.Errorf("archive effects: activities=%d bad=%d events=%d jobs=%d want=%d", count, bad, len(events), issueEventCount(t, w), n)
	}
	for _, event := range events {
		if event.ActorID != nil || event.Entity.ID != id || event.ProjectID != w.web || event.WorkspaceID != w.acme || !event.OccurredAt.Equal(at) {
			t.Errorf("archive event=%+v", event)
		}
	}
}
