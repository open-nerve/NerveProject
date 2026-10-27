package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Old's real pending task keeps its identity and request audit. Only projects
// locked by this workspace request appear in its event, in ascending order.
func TestWorkspaceDeletionPublishesOnlyLockedProjects(t *testing.T) {
	w := newIssueWorld(t)
	old := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Old", "OLD")
	q := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Q", "Q")
	var kept struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Kept","slug":"kept"}`, 201), &kept)
	drafts := []uuid.UUID{uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()}
	for n, p := range []*uuid.UUID{&old, &q, &w.web, nil, nil} {
		ws := w.acme
		if n == 4 {
			ws = kept.ID
		}
		if _, err := w.pool.Exec(pgtest.Soon(t), "INSERT INTO draft_issues(id,workspace_id,project_id,created_by_id,updated_by_id) VALUES($1,$2,$3,$4,$4)", drafts[n], ws, p, w.ids["bob"]); err != nil {
			t.Fatal(err)
		}
	}
	oldItem := w.newWorkItemIn(t, old, `{"name":"Old pending"}`)
	pItem := w.newWorkItem(t)
	qItem := w.newWorkItemIn(t, q, `{"name":"Q"}`)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+pItem.String()+"/relations", `{"issue_ids":["`+qItem.String()+`"],"relation_type":"relates_to"}`, 201)
	invitation := uuid.NewV7()
	if _, err := workspacepg.New(w.pool).CreateInvitations(pgtest.Soon(t), []workspaceapp.InvitationRow{{ID: invitation, WorkspaceID: w.acme, Email: "pending@example.com", Role: shared.RoleGuest, CreatedBy: w.ids["alice"], Now: time.Now()}}); err != nil {
		t.Fatal(err)
	}
	keptBefore := foreignKey{table: "draft_issues", column: "workspace_id"}.rows(t, w.pool, kept.ID)
	oldHolder := holdProjectDeletion(t, w.pool, old)
	w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+old.String(), "", 204)
	oldJob := projectDeletionJob(t, w.pool, old)
	deletionWorkerBlocked(t, w.pool, oldHolder, oldJob, "draft_issues")
	holder := holdProjectDeletion(t, w.pool, w.web)
	var received []shared.Event
	publisher := projectDeletionPublisher(t, w.pool, shared.EventSubscriber{Name: "workspace deletion observer", Types: []string{"workspace.deleted", "project.deleted"}, Handle: func(ctx context.Context, e shared.Event) error {
		received = append(received, e)
		if _, ok := postgres.Tx(ctx); !ok {
			t.Error("event lost original transaction")
		}
		var inside, outside int
		if err := postgres.DB(ctx, w.pool).QueryRow(ctx, "SELECT count(*) FROM river_job WHERE kind='issue.delete_rows'").Scan(&inside); err != nil {
			return err
		}
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM river_job WHERE kind='issue.delete_rows'").Scan(&outside); err != nil {
			return err
		}
		if inside != 4 || outside != 1 {
			t.Errorf("tasks visible inside=%d outside=%d want 4/1", inside, outside)
		}
		return nil
	}})
	route := newWorkspaceRouteWithPublisher(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), publisher)
	route.answerWithin(t, w.contract, http.MethodDelete, "/api/v0/workspaces/acme", w.ids["wanda"], "", 204)
	if len(received) != 1 {
		t.Fatalf("events=%+v", received)
	}
	event := received[0]
	wantProjects := []uuid.UUID{q, w.web}
	slices.SortFunc(wantProjects, uuid.UUID.Compare)
	changes := make([]shared.Change, 0, 2)
	for _, id := range wantProjects {
		changes = append(changes, shared.Change{Field: "project", OldID: &id})
	}
	job := parentDeletionJob(t, w.pool, "workspace_drafts", w.acme)
	if event.ID == (uuid.UUID{}) || event.Type != "workspace.deleted" || event.Entity != (shared.EntityRef{Kind: "workspace", ID: w.acme}) || event.WorkspaceID != w.acme || event.ProjectID != (uuid.UUID{}) || event.ActorID == nil || *event.ActorID != w.ids["wanda"] || !event.OccurredAt.Equal(job.At) || !reflect.DeepEqual(event.Changes, changes) {
		t.Fatalf("event=%+v want changes=%+v", event, changes)
	}
	for _, id := range wantProjects {
		j := projectDeletionJob(t, w.pool, id)
		if !j.At.Equal(job.At) || j.By != job.By {
			t.Fatal("project job lost workspace stamp", j)
		}
	}
	if got := projectDeletionJob(t, w.pool, old); got != oldJob {
		t.Fatalf("old task replaced: %+v want %+v", got, oldJob)
	}
	var revoked bool
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at=$2 AND updated_at=$2 AND updated_by_id=$3 FROM workspace_member_invites WHERE id=$1", invitation, job.At, job.By).Scan(&revoked); err != nil || !revoked {
		t.Fatal("invitation not synchronously revoked", err)
	}
	w.send(t, "alice", http.MethodGet, "/api/v0/workspaces/acme", "", 404)
	w.send(t, "alice", http.MethodGet, "/api/v0/issues/"+oldItem.String(), "", 404)
	w.send(t, "alice", http.MethodGet, "/api/v0/issues/"+pItem.String(), "", 404)
	deletionWorkerBlocked(t, w.pool, holder, projectDeletionJob(t, w.pool, w.web), "draft_issues")
	requestRelease(t, holder)
	waitWorkspaceDeletion(t, w.pool, w.acme)
	// Completing the newer request cannot consume Old's rows or replace its job.
	if got := projectDeletionJob(t, w.pool, old); got != oldJob {
		t.Fatal("workspace completion replaced Old", got)
	}
	var oldLive bool
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at IS NULL FROM draft_issues WHERE id=$1", drafts[0]).Scan(&oldLive); err != nil || !oldLive {
		t.Fatal("Old task did not remain pending", err)
	}
	requestRelease(t, oldHolder)
	waitDeletionJob(t, w.pool, oldJob, "completed")
	for n, id := range drafts[:4] {
		stamp := job
		if n == 0 {
			stamp = oldJob
		}
		var correct bool
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at=$2 AND updated_at=$2 AND updated_by_id=$3 FROM draft_issues WHERE id=$1", id, stamp.At, stamp.By).Scan(&correct); err != nil || !correct {
			t.Errorf("draft %s wrong audit: %v", id, err)
		}
	}
	for _, item := range []struct {
		id  uuid.UUID
		job deletionJob
	}{{oldItem, oldJob}, {pItem, job}, {qItem, job}} {
		var correct bool
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at=$2 AND updated_at=$2 AND updated_by_id=$3 FROM issues WHERE id=$1", item.id, item.job.At, item.job.By).Scan(&correct); err != nil || !correct {
			t.Error("issue audit", item.id, err)
		}
	}
	var relationStamp bool
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT bool_and(deleted_at=$1 AND updated_at=$1 AND updated_by_id=$2) FROM issue_relations", job.At, job.By).Scan(&relationStamp); err != nil || !relationStamp {
		t.Fatal("shared relation stamp", err)
	}
	if after := (foreignKey{table: "draft_issues", column: "workspace_id"}).rows(t, w.pool, kept.ID); after != keptBefore {
		t.Fatal("neighbor drafts changed")
	}
}

func TestWorkspaceDeletionEmptyAndPublicationFailure(t *testing.T) {
	for _, failure := range []string{"", "subscriber", "InsertTx"} {
		t.Run(failure, func(t *testing.T) {
			w := newIssueWorld(t)
			var ws struct {
				ID uuid.UUID `json:"id"`
			}
			decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Empty","slug":"empty"}`, 201), &ws)
			draft := uuid.NewV7()
			if _, err := w.pool.Exec(pgtest.Soon(t), "INSERT INTO draft_issues(id,workspace_id,created_by_id) VALUES($1,$2,$3)", draft, ws.ID, w.ids["bob"]); err != nil {
				t.Fatal(err)
			}
			if failure == "InsertTx" {
				if _, err := w.pool.Exec(pgtest.Soon(t), "ALTER TABLE river_job ADD CONSTRAINT reject_deletion CHECK(kind<>'issue.delete_rows') NOT VALID"); err != nil {
					t.Fatal(err)
				}
			}
			before := registrationRows(t, w.pool)
			events := projectDeletionPublisher(t, w.pool, shared.EventSubscriber{Name: "workspace rejection", Types: []string{"workspace.deleted"}, Handle: func(ctx context.Context, e shared.Event) error {
				if len(e.Changes) != 0 {
					t.Error("empty workspace projects")
				}
				if failure == "subscriber" {
					return errors.New("reject workspace subscriber")
				}
				return nil
			}})
			route := newWorkspaceRouteWithPublisher(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), events)
			status := 204
			if failure != "" {
				status = 500
			}
			route.answerWithin(t, w.contract, http.MethodDelete, "/api/v0/workspaces/empty", w.ids["alice"], "", status)
			if failure != "" {
				if registrationRows(t, w.pool) != before {
					t.Fatal("failed publication changed rows/tasks")
				}
				return
			}
			waitWorkspaceDeletion(t, w.pool, ws.ID)
			var correct bool
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT d.deleted_at=w.deleted_at AND d.updated_at=w.deleted_at AND d.updated_by_id=w.updated_by_id FROM draft_issues d JOIN workspaces w ON w.id=d.workspace_id WHERE d.id=$1", draft).Scan(&correct); err != nil || !correct {
				t.Fatal("empty workspace draft", err)
			}
		})
	}
}

// A late subscriber fails after real task insertion and all M3 project writes.
// Both workspaces' full catalog and stable request-task identities roll back.
func TestWorkspaceDeletionSubscriberRollsBackFullCatalog(t *testing.T) {
	w := newIssueWorld(t)
	registerAccount(t, w.contract, w.base, "member@example.com")
	deleted := seedWorkspace(t, w.contract, w.base, w.pool, w.tokens["alice"], "deleted")
	kept := seedWorkspace(t, w.contract, w.base, w.pool, w.tokens["alice"], "kept")
	keys := keysTo(t, w.pool, "workspaces")
	before := map[string]string{}
	for _, k := range keys {
		for _, id := range []uuid.UUID{deleted, kept} {
			before[k.String()+id.String()] = k.rows(t, w.pool, id)
		}
	}
	jobsBefore := registrationRows(t, w.pool)
	seen := 0
	events := projectDeletionPublisher(t, w.pool, shared.EventSubscriber{Name: "reject after workspace inserts", Types: []string{"workspace.deleted"}, Handle: func(ctx context.Context, e shared.Event) error {
		seen++
		var count int
		if err := postgres.DB(ctx, w.pool).QueryRow(ctx, "SELECT count(*) FROM river_job WHERE kind='issue.delete_rows'").Scan(&count); err != nil {
			return err
		}
		if count != len(e.Changes)+1 || len(e.Changes) != 2 {
			t.Errorf("real InsertTx count=%d changes=%d", count, len(e.Changes))
		}
		return errors.New("reject all workspace changes")
	}})
	route := newWorkspaceRouteWithPublisher(t, w.pool, postgres.NewTxManager(w.pool, 2*time.Second), events)
	route.answerWithin(t, w.contract, http.MethodDelete, "/api/v0/workspaces/deleted", w.ids["alice"], "", 500)
	if seen != 1 || registrationRows(t, w.pool) != jobsBefore {
		t.Fatal("subscriber failure leaked rows/tasks", seen)
	}
	for _, k := range keys {
		for _, id := range []uuid.UUID{deleted, kept} {
			if k.rows(t, w.pool, id) != before[k.String()+id.String()] {
				t.Error("rollback changed", k, id)
			}
		}
	}
}
