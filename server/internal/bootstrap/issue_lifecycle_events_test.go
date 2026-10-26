package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Create children before their root so ID order differs from root-first tree
// traversal. Bob creates the hidden children.
func lifecycleTree(t *testing.T, w issueWorld, guest bool) (uuid.UUID, []uuid.UUID, string) {
	t.Helper()
	actor, creator := "alice", "bob"
	if guest {
		w.adminGuest(t)
		actor, creator = "walt", "walt"
	}
	body := fmt.Sprintf(`{"name":"Lifecycle","state_id":%q,"assignee_ids":[%q],"label_ids":[%q],"description_html":%q}`, w.done, w.ids["bob"], w.bug, mentionHTML(w.ids["bob"]))
	child := w.newWorkItemBy(t, "bob", w.web, body)
	leaf := w.newWorkItemBy(t, "bob", w.web, body)
	root := w.newWorkItemBy(t, creator, w.web, body)
	ids := []uuid.UUID{child, leaf, root}
	underWorkItem(t, w.pool, root, child)
	underWorkItem(t, w.pool, child, leaf)
	for _, id := range ids {
		w.send(t, "bob", http.MethodPost, reactionsOf(id), `{"reaction":"128077"}`, 201)
		comment := w.newComment(t, "bob", id)
		addCommentReaction(t, w, "bob", comment.ID, "128077")
	}
	relate(t, w.pool, leaf, root, "relates_to", w.ids["bob"])
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+leaf.String()+"/archive", "", 200)
	if guest {
		w.send(t, actor, http.MethodGet, "/api/v0/issues/"+child.String(), "", 404)
	}
	slices.SortFunc(ids, uuid.UUID.Compare)
	return root, ids, actor
}

func lifecycleRows(t *testing.T, w issueWorld, id uuid.UUID) map[string]string {
	t.Helper()
	rows := collaborationRows(t, w, id)
	for _, table := range []string{"issue_assignees", "issue_labels", "issue_relations"} {
		rows[table] = descriptionRows(t, w, table, id)
	}
	return rows
}

func TestIssueDeletionEvents(t *testing.T) {
	for _, guest := range []bool{false, true} {
		for _, reject := range []bool{false, true} {
			t.Run(fmt.Sprintf("guest=%t/final subscriber fails=%t", guest, reject), func(t *testing.T) {
				w := newIssueWorld(t)
				root, ids, actor := lifecycleTree(t, w, guest)
				snapshots := map[uuid.UUID]map[string]string{}
				for _, id := range ids {
					snapshots[id] = lifecycleRows(t, w, id)
				}
				at := time.Date(2044, 5, 6, 7, 8, 9, 123456000, time.UTC)
				clock := &eventCountingClock{base: clockAt(at)}
				var received []shared.Event
				var failure error
				route := issueEventRouteFor(t, w, clock, &received, &failure, issue.EventTypes(), func(ctx context.Context, e shared.Event) error {
					// Every table, including every child's activity/version, is already deleted
					// before the first event, observed through the business transaction.
					for table := range snapshots[root] {
						condition := "issue_id=ANY($1)"
						if table == "issues" {
							condition = "id=ANY($1)"
						}
						if table == "comment_reactions" {
							condition = "comment_id IN (SELECT id FROM issue_comments WHERE issue_id=ANY($1))"
						}
						var n int
						if err := postgres.DB(ctx, w.pool).QueryRow(ctx, "SELECT count(*) FROM "+table+" WHERE "+condition+" AND deleted_at IS NULL", ids).Scan(&n); err != nil {
							return err
						}
						if n != 0 {
							t.Errorf("event preceded %s subtree batch: %d live rows", table, n)
						}
					}
					if reject && len(received) == len(ids)-1 {
						failure = errors.New("final deletion subscriber failed")
					}
					return nil
				})
				status := 204
				if reject {
					status = 500
				}
				route.answerWithin(t, w.contract, http.MethodDelete, "/api/v0/issues/"+root.String(), w.ids[actor], "", status)
				if reject {
					for _, id := range ids {
						if !maps.Equal(snapshots[id], lifecycleRows(t, w, id)) {
							t.Error("final subscriber failure committed subtree rows/activities", id)
						}
					}
					if issueEventCount(t, w) != 0 {
						t.Error("final subscriber failure committed earlier River jobs")
					}
				}
				if len(received) != len(ids) {
					t.Fatalf("events=%d want complete subtree %d", len(received), len(ids))
				}
				seen := map[uuid.UUID]bool{}
				for n, e := range received {
					if e.ID == (uuid.UUID{}) || e.ID == ids[n] || seen[e.ID] || e.Type != "issue.deleted" || e.Entity != (shared.EntityRef{Kind: "issue", ID: ids[n]}) || e.ActorID == nil || *e.ActorID != w.ids[actor] || e.WorkspaceID != w.acme || e.ProjectID != w.web || !e.OccurredAt.Equal(at) || len(e.Changes) != 0 {
						t.Errorf("deletion event %d=%+v", n, e)
					}
					seen[e.ID] = true
					if !reject {
						var deleted time.Time
						var by uuid.UUID
						if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at,updated_by_id FROM issues WHERE id=$1", ids[n]).Scan(&deleted, &by); err != nil || !deleted.Equal(at) || by != w.ids[actor] {
							t.Errorf("deleted row stamp %s %v %v %v", ids[n], deleted, by, err)
						}
					}
				}
				wantJobs := len(ids)
				if reject {
					wantJobs = 0
				}
				if issueEventCount(t, w) != wantJobs {
					t.Fatalf("jobs=%d want %d (earlier event jobs must roll back too)", issueEventCount(t, w), wantJobs)
				}
				if clock.calls.Load() != 1 {
					t.Fatal("subtree did not share one business clock reading")
				}
			})
		}
	}
}

func TestIssueArchivalEvents(t *testing.T) {
	w := newIssueWorld(t)
	id := w.doneWorkItem(t)
	at := time.Date(2044, 5, 6, 7, 8, 9, 123456000, time.UTC)
	clock := &eventCountingClock{base: clockAt(at)}
	var received []shared.Event
	var failure error
	route := issueEventRouteFor(t, w, clock, &received, &failure, issue.EventTypes(), func(ctx context.Context, e shared.Event) error {
		var old, new *string
		var actor uuid.UUID
		var occurred time.Time
		if err := postgres.DB(ctx, w.pool).QueryRow(ctx, `SELECT old_value,new_value,actor_id,created_at FROM issue_activities WHERE issue_id=$1 AND field='archived_at' AND deleted_at IS NULL ORDER BY id DESC LIMIT 1`, id).Scan(&old, &new, &actor, &occurred); err != nil {
			return err
		}
		if len(e.Changes) != 1 || !reflect.DeepEqual(e.Changes[0], shared.Change{Field: "archived_at", Old: old, New: new}) || e.ActorID == nil || actor != *e.ActorID || !occurred.Equal(e.OccurredAt) {
			t.Errorf("archival event/activity differ: %+v", e)
		}
		return nil
	})
	path := "/api/v0/issues/" + id.String()
	for _, on := range []bool{true, false} {
		verb, kind := "unarchive", "issue.unarchived"
		want := []shared.Change{{Field: "archived_at", Old: eventValue("archive"), New: eventValue("restore")}}
		if on {
			verb, kind = "archive", "issue.archived"
			want = []shared.Change{{Field: "archived_at", New: eventValue("manual_archive")}}
		}
		// A publication failure must undo both the issue and the new activity/job.
		before := lifecycleRows(t, w, id)
		jobsBefore := issueEventCount(t, w)
		failure = errors.New("archival subscriber failed")
		route.answerWithin(t, w.contract, http.MethodPost, path+"/"+verb, w.ids["bob"], "", 500)
		if !maps.Equal(before, lifecycleRows(t, w, id)) || issueEventCount(t, w) != jobsBefore {
			t.Fatal("archival failure committed issue/activity/job")
		}
		failure = nil
		route.answerWithin(t, w.contract, http.MethodPost, path+"/"+verb, w.ids["bob"], "", 200)
		e := received[len(received)-1]
		if e.ID == (uuid.UUID{}) || e.ID == id || e.Type != kind || e.Entity != (shared.EntityRef{Kind: "issue", ID: id}) || e.ActorID == nil || *e.ActorID != w.ids["bob"] || e.WorkspaceID != w.acme || e.ProjectID != w.web || !e.OccurredAt.Equal(at) || !reflect.DeepEqual(e.Changes, want) {
			t.Fatalf("archival envelope=%+v", e)
		}
		archived, changed, by := archivalOf(t, w, id)
		if (archived != nil) != on || !changed.Equal(e.OccurredAt) || by != *e.ActorID {
			t.Fatal("archival event preceded issue write")
		}
		before = lifecycleRows(t, w, id)
		n, reads := len(received), clock.calls.Load()
		route.answerWithin(t, w.contract, http.MethodPost, path+"/"+verb, w.ids["bob"], "", 200)
		if len(received) != n || clock.calls.Load() != reads || !maps.Equal(before, lifecycleRows(t, w, id)) || issueEventCount(t, w) != jobsBefore+1 {
			t.Fatal("idempotent archival published/wrote/read clock")
		}
	}
	seen := map[uuid.UUID]bool{}
	for _, e := range received {
		if seen[e.ID] {
			t.Fatal("archival reused event identity")
		}
		seen[e.ID] = true
	}
}

func TestIssueDeletionEventsCascade(t *testing.T) {
	// Project/workspace cascades use the provided boundary, which deliberately
	// has no issue publisher. These real batch writes must not enqueue issue jobs.
	for _, workspace := range []bool{false, true} {
		t.Run(fmt.Sprintf("workspace=%t", workspace), func(t *testing.T) {
			w := newIssueWorld(t)
			_, ids, _ := lifecycleTree(t, w, false)
			cascade := issue.Provide(w.pool).IssueCascade
			err := postgres.NewTxManager(w.pool, 2*time.Second).WithinTx(pgtest.Soon(t), func(ctx context.Context) error {
				if workspace {
					return cascade.DeleteWorkspaceIssues(ctx, []uuid.UUID{w.web}, w.ids["alice"], time.Now())
				}
				return cascade.DeleteProjectIssues(ctx, w.web, w.ids["alice"], time.Now())
			})
			if err != nil {
				t.Fatal(err)
			}
			if live := queryIDs(t, w.pool, "SELECT id FROM issues WHERE id=ANY($1) AND deleted_at IS NULL", ids); len(live) != 0 || issueEventCount(t, w) != 0 {
				t.Fatal("cascade failed or enqueued issue events", live)
			}
		})
	}
}
