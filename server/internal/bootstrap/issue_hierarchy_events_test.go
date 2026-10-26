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

func hierarchySnapshots(t *testing.T, w issueWorld, ids []uuid.UUID) map[uuid.UUID]map[string]string {
	t.Helper()
	out := map[uuid.UUID]map[string]string{}
	for _, id := range ids {
		out[id] = lifecycleRows(t, w, id)
	}
	return out
}
func wantHierarchySnapshots(t *testing.T, w issueWorld, before map[uuid.UUID]map[string]string) {
	t.Helper()
	for id, rows := range before {
		if !maps.Equal(rows, lifecycleRows(t, w, id)) {
			t.Errorf("publication failure changed rows of %s", id)
		}
	}
}

func TestHierarchyEventsComposition(t *testing.T) {
	for _, reject := range []bool{false, true} {
		t.Run(fmt.Sprint("final subscriber fails=", reject), func(t *testing.T) {
			w := newIssueWorld(t)
			parent := w.newWorkItem(t)
			old := w.newWorkItem(t)
			first := w.newWorkItem(t)
			second := w.newWorkItemIn(t, w.web, underParent("Moving", old))
			unchanged := w.newWorkItemIn(t, w.web, underParent("Unchanged", parent))
			moved := []uuid.UUID{first, second}
			all := []uuid.UUID{parent, old, first, second, unchanged}
			before := hierarchySnapshots(t, w, all)
			at := time.Date(2044, 5, 6, 7, 8, 9, 123456000, time.UTC)
			clock := &eventCountingClock{base: clockAt(at)}
			var received []shared.Event
			var failure error
			route := issueEventRouteFor(t, w, clock, &received, &failure, issue.EventTypes(), func(ctx context.Context, e shared.Event) error {
				var parents, activities int
				db := postgres.DB(ctx, w.pool)
				if err := db.QueryRow(ctx, "SELECT count(*) FROM issues WHERE id=ANY($1) AND parent_id=$2 AND updated_by_id=$3 AND updated_at=$4", moved, parent, w.ids["bob"], at).Scan(&parents); err != nil {
					return err
				}
				if err := db.QueryRow(ctx, "SELECT count(*) FROM issue_activities WHERE issue_id=ANY($1) AND field='parent' AND actor_id=$2 AND created_at=$3", moved, w.ids["bob"], at).Scan(&activities); err != nil {
					return err
				}
				if parents != 2 || activities != 2 {
					t.Errorf("event preceded batch: parents=%d activities=%d", parents, activities)
				}
				if reject && len(received) == 1 {
					failure = errors.New("last hierarchy subscriber failed")
				}
				return nil
			})
			status := 200
			if reject {
				status = 500
			}
			route.answerWithin(t, w.contract, http.MethodPost, subIssues(parent), w.ids["bob"], subIssueIDs(unchanged, second, first), status)
			if len(received) != 2 {
				t.Fatalf("hierarchy events=%d want two moved children", len(received))
			}
			seen := map[uuid.UUID]bool{}
			for n, e := range received {
				want := shared.Change{Field: "parent", Old: eventValue(""), New: eventValue("WEB-1"), NewID: &parent}
				if n == 1 {
					want.Old, want.OldID = eventValue("WEB-2"), &old
				}
				if e.ID == (uuid.UUID{}) || e.ID == moved[n] || seen[e.ID] || e.Type != "issue.updated" || e.Entity != (shared.EntityRef{Kind: "issue", ID: moved[n]}) || e.WorkspaceID != w.acme || e.ProjectID != w.web || e.ActorID == nil || *e.ActorID != w.ids["bob"] || !e.OccurredAt.Equal(at) || !reflect.DeepEqual(e.Changes, []shared.Change{want}) {
					t.Errorf("hierarchy event=%+v changes=%+v want=%+v", e, e.Changes, want)
				}
				seen[e.ID] = true
			}
			if clock.calls.Load() != 1 {
				t.Fatal("batch did not share clock")
			}
			jobs := 2
			if reject {
				jobs = 0
				wantHierarchySnapshots(t, w, before)
			}
			if issueEventCount(t, w) != jobs {
				t.Fatal("batch jobs did not commit/rollback together")
			}
			if !reject {
				snapshot := hierarchySnapshots(t, w, all)
				route.answerWithin(t, w.contract, http.MethodPost, subIssues(parent), w.ids["bob"], subIssueIDs(second, unchanged, first), 200)
				if len(received) != 2 || clock.calls.Load() != 1 || issueEventCount(t, w) != 2 {
					t.Fatal("unchanged hierarchy published or read clock")
				}
				wantHierarchySnapshots(t, w, snapshot)
			}
		})
	}
}

func TestRelationEventsComposition(t *testing.T) {
	for _, earlierTargets := range []bool{false, true} {
		for _, blocking := range []bool{false, true} {
			for _, reject := range []bool{false, true} {
				t.Run(fmt.Sprintf("earlier targets=%t/blocking=%t/final subscriber fails=%t", earlierTargets, blocking, reject), func(t *testing.T) {
					w := newIssueWorld(t)
					ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
					w.makeMember(t, "walt", ops, shared.RoleMember)
					var from uuid.UUID
					if !earlierTargets {
						from = w.newWorkItem(t)
					}
					targets := []uuid.UUID{w.newWorkItemIn(t, ops, `{"name":"First"}`), w.newWorkItemIn(t, ops, `{"name":"Second"}`)}
					if earlierTargets {
						from = w.newWorkItem(t)
					}
					all := append(slices.Clone(targets), from)
					before := hierarchySnapshots(t, w, all)
					at := time.Date(2044, 5, 6, 7, 8, 9, 123456000, time.UTC)
					clock := &eventCountingClock{base: clockAt(at)}
					var received []shared.Event
					var failure error
					removing := false
					route := issueEventRouteFor(t, w, clock, &received, &failure, issue.EventTypes(), func(ctx context.Context, e shared.Event) error {
						db := postgres.DB(ctx, w.pool)
						var marked, activities int
						if err := db.QueryRow(ctx, "SELECT count(*) FROM issues WHERE id=ANY($1) AND updated_at=$2", all, at).Scan(&marked); err != nil {
							return err
						}
						if err := db.QueryRow(ctx, "SELECT count(*) FROM issue_activities WHERE issue_id=ANY($1) AND created_at=$2 AND comment LIKE '% relation'", all, at).Scan(&activities); err != nil {
							return err
						}
						if !removing {
							var count int
							if err := db.QueryRow(ctx, "SELECT count(*) FROM issue_relations WHERE (issue_id=$1 OR related_issue_id=$1) AND deleted_at IS NULL", from).Scan(&count); err != nil {
								return err
							}
							if count != 2 || marked != 3 || activities != 4 {
								t.Errorf("relation event preceded complete batch: relations=%d audit=%d activities=%d", count, marked, activities)
							}
							if reject && len(received) == 1 {
								failure = errors.New("last relation subscriber failed")
							}
						} else {
							var deleted bool
							var ends []uuid.UUID
							if err := db.QueryRow(ctx, "SELECT deleted_at IS NOT NULL,ARRAY[issue_id,related_issue_id] FROM issue_relations WHERE id=$1", e.Entity.ID).Scan(&deleted, &ends); err != nil {
								return err
							}
							if err := db.QueryRow(ctx, "SELECT count(*) FROM issues WHERE id=ANY($1) AND updated_by_id=$2 AND updated_at=$3", ends, e.ActorID, at).Scan(&marked); err != nil {
								return err
							}
							if err := db.QueryRow(ctx, "SELECT count(*) FROM issue_activities WHERE issue_id=ANY($1) AND actor_id=$2 AND created_at=$3 AND verb='deleted' AND comment LIKE '% relation'", ends, e.ActorID, at).Scan(&activities); err != nil {
								return err
							}
							if !deleted || marked != 2 || activities != 2 {
								t.Errorf("removal event preceded deletion/audit/activity: deleted=%t marked=%d activities=%d", deleted, marked, activities)
							}
						}
						return nil
					})
					kind := "relates_to"
					if blocking {
						kind = "blocking"
					}
					status := 201
					if reject {
						status = 500
					}
					answer := route.answerWithin(t, w.contract, http.MethodPost, relationsOf(from), w.ids["alice"], relating(kind, targets[1], targets[0]), status)
					if len(received) != 2 {
						t.Fatalf("relation events=%d want two rows (not four activities)", len(received))
					}
					made := []answeredRelation{}
					if !reject {
						made = listIn[answeredRelation](t, answer)
					}
					ids := map[uuid.UUID]bool{}
					for n, e := range received {
						project, issueID, relatedID, storedType := w.web, from, targets[n], "relates_to"
						if blocking {
							project, issueID, relatedID, storedType = ops, targets[n], from, "blocked_by"
						}
						rowID := e.Entity.ID
						if !reject {
							rowID = made[n].ID
						}
						wantStoredRelationEvent(t, w, e, "issue.relation_added", rowID, project, issueID, relatedID, storedType, w.ids["alice"], at)
						if ids[e.ID] {
							t.Fatal("batch event identity reused")
						}
						ids[e.ID] = true
						if !reject {
							var p, i, o uuid.UUID
							var typ string
							if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT project_id,issue_id,related_issue_id,relation_type FROM issue_relations WHERE id=$1", rowID).Scan(&p, &i, &o, &typ); err != nil {
								t.Fatal(err)
							}
							if p != project || i != issueID || o != relatedID || typ != storedType {
								t.Fatal("event differs from canonical relation row")
							}
						}
					}
					if clock.calls.Load() != 1 {
						t.Fatal("relation batch did not share clock")
					}
					if reject {
						wantHierarchySnapshots(t, w, before)
						if issueEventCount(t, w) != 0 {
							t.Fatal("last failed event left earlier jobs")
						}
						return
					}
					if issueEventCount(t, w) != 2 {
						t.Fatal("missing committed relation jobs")
					}
					removing = true
					// Bob can update Web only, Walt Ops only. Thus both authorized ends
					// are exercised for each canonical storage direction.
					for n, actor := range []string{"bob", "walt"} {
						snapshot := hierarchySnapshots(t, w, all)
						failure = errors.New("removal subscriber failed")
						route.answerWithin(t, w.contract, http.MethodDelete, relationPath(made[n].ID), w.ids[actor], "", 500)
						wantHierarchySnapshots(t, w, snapshot)
						if issueEventCount(t, w) != 2+n {
							t.Fatal("failed removal committed job")
						}
						failure = nil
						route.answerWithin(t, w.contract, http.MethodDelete, relationPath(made[n].ID), w.ids[actor], "", 204)
						e := received[len(received)-1]
						project, issueID, relatedID, storedType := w.web, from, targets[n], "relates_to"
						if blocking {
							project, issueID, relatedID, storedType = ops, targets[n], from, "blocked_by"
						}
						wantStoredRelationEvent(t, w, e, "issue.relation_removed", made[n].ID, project, issueID, relatedID, storedType, w.ids[actor], at)
						if issueEventCount(t, w) != 3+n {
							t.Fatal("successful removal job missing")
						}
					}
				})
			}
		}
	}
}

func wantStoredRelationEvent(t *testing.T, w issueWorld, e shared.Event, kind string, row, project, from, to uuid.UUID, typ string, actor uuid.UUID, at time.Time) {
	t.Helper()
	want := []shared.Change{{Field: "relation_type", New: eventValue(typ)}, {Field: "issue", NewID: &from}, {Field: "related_issue", NewID: &to}}
	if kind == "issue.relation_removed" {
		want = []shared.Change{{Field: "relation_type", Old: eventValue(typ)}, {Field: "issue", OldID: &from}, {Field: "related_issue", OldID: &to}}
	}
	if e.ID == (uuid.UUID{}) || e.ID == row || e.Type != kind || e.Entity != (shared.EntityRef{Kind: "issue_relation", ID: row}) || e.WorkspaceID != w.acme || e.ProjectID != project || e.ActorID == nil || *e.ActorID != actor || !e.OccurredAt.Equal(at) || !reflect.DeepEqual(e.Changes, want) {
		t.Errorf("canonical relation event=%+v changes=%+v want=%+v", e, e.Changes, want)
	}
}
