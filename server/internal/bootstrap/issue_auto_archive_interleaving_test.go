package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Removing project N lets the request commit through the job's pre-write
// barrier: it wrongly returns 200 instead of issue.archived and no archive is
// made. The reverse direction must preserve the committed nonterminal state.
func TestAutoArchiveAndStateChangeSerialize(t *testing.T) {
	for _, jobFirst := range []bool{false, true} {
		t.Run(fmt.Sprintf("job_first=%t", jobFirst), func(t *testing.T) {
			w := newIssueWorld(t)
			at := time.Now().UTC().Truncate(time.Microsecond)
			id := archiveRaceSeed(t, w, at, 40*24*time.Hour)
			before := archiveStableRow(t, w, id)
			events := newArchiveRaceEvents(t, w)
			requestEvents := newArchiveStateEvents(t)
			g := newGate()
			c := clockAt(at)
			tx := shared.TxManager(postgres.NewTxManager(w.pool, 2*time.Second))
			var locked *gate
			if jobFirst {
				locked = g
			} else {
				tx = heldTx{tx, g}
			}
			job := archiveRaceJob(w, c, events, nil, locked)
			var reply exchange
			n := 0
			request := w.sending(&reply, newIssueRouteWithPublisher(t, w.pool, tx, c, requestEvents), "bob", http.MethodPatch, "/api/v0/issues/"+id.String(), `{"state_id":"`+w.todo.String()+`"}`)
			archive := func() error { var err error; n, err = job.Execute(pgtest.Soon(t)); return err }
			first, second := request, archive
			if jobFirst {
				first, second = archive, request
			}
			archiveInterleave(t, w, g, first, second, nil, nil)
			want := 0
			if jobFirst {
				want = 1
				w.refused(t, reply, 409, "issue.archived")
				if archiveStableRow(t, w, id) != before {
					t.Error("refused state change modified archived issue")
				}
			} else {
				w.answered(t, reply, 200)
				var valid bool
				if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT state_id=$2 AND archived_at IS NULL AND completed_at IS NULL AND updated_at=$3 AND updated_by_id=$4 FROM issues WHERE id=$1`, id, w.todo, at, w.ids["bob"]).Scan(&valid); err != nil || !valid {
					t.Errorf("state request lost: %t %v", valid, err)
				}
				var count int
				if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field='state' AND actor_id=$2 AND created_at=$3 AND updated_at=$3`, id, w.ids["bob"], at).Scan(&count); err != nil || count != 1 {
					t.Errorf("state activity=%d %v", count, err)
				}
			}
			if n != want {
				t.Errorf("archived=%d want=%d", n, want)
			}
			archiveRaceCheck(t, w, id, at, want, events, false)
			changed := requestEvents.snapshot()
			if len(changed) != 1-want {
				t.Errorf("request events=%d want=%d", len(changed), 1-want)
			}
			for _, event := range changed {
				if event.Type != "issue.updated" || event.Entity.ID != id || event.ActorID == nil || *event.ActorID != w.ids["bob"] || !event.OccurredAt.Equal(at) || len(event.Changes) != 1 {
					t.Errorf("state event=%+v", event)
					continue
				}
				change := event.Changes[0]
				if change.Field != "state" || change.OldID == nil || *change.OldID != w.done || change.NewID == nil || *change.NewID != w.todo {
					t.Errorf("state change payload=%+v", change)
				}
			}
		})
	}
}

// Advancing across the cutoff while waiting distinguishes lock-before-Now
// from a stale clock: the 29-day issue becomes 31 days old, and every system
// stamp must use the new instant, without concurrent writes to Fixed.
func TestAutoArchiveReadsClockAfterLock(t *testing.T) {
	w := newIssueWorld(t)
	at := time.Now().UTC().Truncate(time.Microsecond)
	id := archiveRaceSeed(t, w, at, 29*24*time.Hour)
	c := clockAt(at)
	events := newArchiveRaceEvents(t, w)
	holder := holding(t, w.pool, "SELECT 1 FROM projects WHERE id=$1 FOR NO KEY UPDATE", w.web)
	defer func() { _ = holder.Rollback(context.Background()) }()
	job := autoArchiveOn(w, c, events)
	n := 0
	done := run(func() error { var err error; n, err = job.Execute(pgtest.Soon(t)); return err })
	table, ended, err := archiveWaitOrDone(t, w, done)
	c.advance(48 * time.Hour)
	if err := holder.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !ended {
		err = result(t, pgtest.Soon(t), done, "clock archive")
	}
	if err != nil || n != 1 {
		t.Errorf("post-lock cutoff archived=%d err=%v; want 1", n, err)
	}
	archiveRaceCheck(t, w, id, at.Add(48*time.Hour), 1, events, false)
	if table != "projects" {
		t.Errorf("clock waiter=%q ended=%t", table, ended)
	}
}
