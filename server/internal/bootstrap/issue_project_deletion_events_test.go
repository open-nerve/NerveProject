package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestProjectDeletionPublishesAndEnqueuesAtomically(t *testing.T) {
	contract, base, pool, alice, aliceID, web, _ := twoProjects(t)
	keys := keysTo(t, pool, "projects")
	before := map[string]string{}
	recorded := map[string][]uuid.UUID{}
	for _, k := range keys {
		before[k.String()] = k.rows(t, pool, web)
		recorded[k.String()] = k.undeleted(t, pool, web)
	}
	holder := holdProjectDeletion(t, pool, web)
	var received []shared.Event
	events := projectDeletionPublisher(t, pool, shared.EventSubscriber{Name: "observe project transaction", Types: []string{"project.deleted"}, Handle: func(ctx context.Context, e shared.Event) error {
		received = append(received, e)
		if _, ok := postgres.Tx(ctx); !ok {
			t.Error("project event escaped request transaction")
		}
		var inside, outside int
		if err := postgres.DB(ctx, pool).QueryRow(ctx, "SELECT count(*) FROM river_job WHERE kind='issue.delete_rows'").Scan(&inside); err != nil {
			return err
		}
		if err := pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM river_job WHERE kind='issue.delete_rows'").Scan(&outside); err != nil {
			return err
		}
		if inside != 1 || outside != 0 {
			t.Errorf("request task visibility inside=%d outside=%d", inside, outside)
		}
		var deleted bool
		if err := postgres.DB(ctx, pool).QueryRow(ctx, "SELECT deleted_at=$2 AND updated_at=$2 AND updated_by_id=$3 FROM projects WHERE id=$1", web, e.OccurredAt, aliceID).Scan(&deleted); err != nil {
			return err
		}
		if !deleted {
			t.Error("event preceded M3 deletion")
		}
		return nil
	}})
	route := newProjectRouteWithPublisher(t, pool, postgres.NewTxManager(pool, 2*time.Second), authorizerOn(pool), workspace.Provide(pool).WorkspaceMembers, events)
	// A synchronous cascade cannot return 204 while the first issue batch is held.
	route.answerWithin(t, contract, http.MethodDelete, "/api/v0/projects/"+web.String(), aliceID, "", 204)
	for _, k := range keys {
		if projectIssueTable(k.table) {
			if got := k.rows(t, pool, web); got != before[k.String()] {
				t.Fatalf("%s changed before worker release", k)
			}
		} else {
			assertProjectDeletionStamp(t, pool, k, recorded[k.String()], web, aliceID)
		}
	}
	for _, id := range queryIDs(t, pool, "SELECT id FROM issues WHERE project_id=$1", web) {
		if status, body := call(t, contract, http.MethodGet, base+"/api/v0/issues/"+id.String(), alice, ""); status != 404 {
			t.Fatalf("pending issue visible: %d %s", status, body)
		}
	}
	job := projectDeletionJob(t, pool, web)
	if len(received) != 1 {
		t.Fatal("project event count", len(received))
	}
	e := received[0]
	var ws uuid.UUID
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT workspace_id FROM projects WHERE id=$1", web).Scan(&ws); err != nil {
		t.Fatal(err)
	}
	if e.ID == (uuid.UUID{}) || e.Type != "project.deleted" || e.Entity != (shared.EntityRef{Kind: "project", ID: web}) || e.WorkspaceID != ws || e.ProjectID != web || e.ActorID == nil || *e.ActorID != aliceID || !e.OccurredAt.Equal(job.At) || len(e.Changes) != 0 {
		t.Fatalf("project event=%+v", e)
	}
	beforeRepeat := registrationRows(t, pool)
	route.answerWithin(t, contract, http.MethodDelete, "/api/v0/projects/"+web.String(), aliceID, "", 404)
	if after := registrationRows(t, pool); after != beforeRepeat {
		t.Fatal("repeat deletion changed rows/jobs")
	}
	if len(received) != 1 {
		t.Fatal("repeat deletion republished")
	}
	deletionWorkerBlocked(t, pool, holder, job, "draft_issues")
	requestRelease(t, holder)
	waitDeletionJob(t, pool, job, "completed")
	for _, k := range keys {
		assertProjectDeletionStamp(t, pool, k, recorded[k.String()], web, aliceID)
	}
}

func TestProjectDeletionPublicationFailuresRollback(t *testing.T) {
	for _, failure := range []string{"subscriber", "InsertTx", "unseen"} {
		t.Run(failure, func(t *testing.T) {
			contract, _, pool, _, aliceID, web, ops := twoProjects(t)
			keys := keysTo(t, pool, "projects")
			snapshots := map[string]string{}
			for _, k := range keys {
				for _, id := range []uuid.UUID{web, ops} {
					snapshots[k.String()+id.String()] = k.rows(t, pool, id)
				}
			}
			if failure == "InsertTx" {
				if _, err := pool.Exec(pgtest.Soon(t), "ALTER TABLE river_job ADD CONSTRAINT reject_deletion CHECK (kind<>'issue.delete_rows') NOT VALID"); err != nil {
					t.Fatal(err)
				}
			}
			before := registrationRows(t, pool)
			seen := 0
			events := projectDeletionPublisher(t, pool, shared.EventSubscriber{Name: "reject project transaction", Types: []string{"project.deleted"}, Handle: func(ctx context.Context, e shared.Event) error {
				seen++
				if _, ok := postgres.Tx(ctx); !ok {
					t.Error("failure observer lost transaction")
				}
				var n int
				if err := postgres.DB(ctx, pool).QueryRow(ctx, "SELECT count(*) FROM river_job WHERE kind='issue.delete_rows'").Scan(&n); err != nil {
					return err
				}
				if n != 1 {
					t.Errorf("subscriber did not follow real InsertTx: %d", n)
				}
				return errors.New("reject project subscriber")
			}})
			route := newProjectRouteWithPublisher(t, pool, postgres.NewTxManager(pool, 2*time.Second), authorizerOn(pool), workspace.Provide(pool).WorkspaceMembers, events)
			actor, status := aliceID, 500
			if failure == "unseen" {
				actor, status = uuid.NewV7(), 404
			}
			route.answerWithin(t, contract, http.MethodDelete, "/api/v0/projects/"+web.String(), actor, "", status)
			wantSeen := 0
			if failure == "subscriber" {
				wantSeen = 1
			}
			if seen != wantSeen {
				t.Errorf("subscriber calls=%d want=%d", seen, wantSeen)
			}
			for _, k := range keys {
				for _, id := range []uuid.UUID{web, ops} {
					if k.rows(t, pool, id) != snapshots[k.String()+id.String()] {
						t.Errorf("%s of %s changed after rejection", k, id)
					}
				}
			}
			if after := registrationRows(t, pool); after != before {
				t.Fatal("rejected project deletion changed full rows/request job identities")
			}
		})
	}
}
