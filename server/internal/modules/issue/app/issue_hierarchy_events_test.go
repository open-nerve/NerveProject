package app_test

import (
	"errors"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Missing/misaddressed events, publishing unchanged children, or publishing
// before the complete batch/activity write must all fail this fixture.
func TestHierarchyEvents(t *testing.T) {
	for _, fail := range []string{"", "SetParents", "InsertActivities", "publisher"} {
		t.Run(fail, func(t *testing.T) {
			_, w, id := newUpdate()
			k := withKin(w)
			putUnder(w, k.top, k.up.ID)
			putUnder(w, k.mine, id)
			p := &recordedEvents{t: t, before: func(shared.Event) {
				if w.issues.parented == nil || len(w.issues.activities) != 2 {
					t.Fatal("event preceded complete parent/activity batch")
				}
			}}
			if fail == "publisher" {
				p.fail = errDisk
			} else {
				w.log.fail = fail
			}
			uc := app.NewAddSubIssues(app.AddSubIssuesDeps{Events: p, Locks: w.locks, Issues: w.issues, Clock: clockAt{at: clockNow, log: w.log}})
			_, err := uc.Execute(as(alice), id, []uuid.UUID{k.mine.ID, k.top.ID, k.up.ID})
			if fail != "" && !errors.Is(err, errDisk) {
				t.Fatalf("failure lost: %v", err)
			}
			if fail == "SetParents" || fail == "InsertActivities" {
				if len(p.rows) != 0 {
					t.Fatal("failed business batch published")
				}
				return
			}
			if fail == "" && err != nil {
				t.Fatal(err)
			}
			wantIDs := sorted([]uuid.UUID{k.top.ID, k.up.ID})
			n := 2
			if fail == "publisher" {
				n = 1
			}
			if len(p.rows) != n {
				t.Fatalf("events=%d want %d (no parent or unchanged child event)", len(p.rows), n)
			}
			seen := map[uuid.UUID]bool{}
			for at, e := range p.rows {
				want := shared.Change{Field: "parent", Old: evptr(""), New: evptr("WEB-7"), NewID: &id}
				if wantIDs[at] == k.top.ID {
					want.Old, want.OldID = evptr("WEB-3"), &k.up.ID
				}
				wantIssueEvent(t, &recordedEvents{rows: []shared.Event{e}}, "issue.updated", wantIDs[at], []shared.Change{want})
				if seen[e.ID] {
					t.Fatal("batch reused event ID")
				}
				seen[e.ID] = true
			}
			if eventClockReads(w) != 1 {
				t.Fatal("batch clock not shared")
			}
			if fail == "" {
				p.rows = nil
				_, err = uc.Execute(as(alice), id, []uuid.UUID{k.mine.ID, k.top.ID, k.up.ID})
				if err != nil || len(p.rows) != 0 || eventClockReads(w) != 1 {
					t.Fatal("no-op moved/published/read clock", err)
				}
			}
		})
	}
}

func wantRelationEvent(t *testing.T, e shared.Event, kind string, row, project, from, to uuid.UUID, relation string) {
	t.Helper()
	want := []shared.Change{{Field: "relation_type", New: evptr(relation)}, {Field: "issue", NewID: &from}, {Field: "related_issue", NewID: &to}}
	if kind == "issue.relation_removed" {
		want = []shared.Change{{Field: "relation_type", Old: evptr(relation)}, {Field: "issue", OldID: &from}, {Field: "related_issue", OldID: &to}}
	}
	if e.ID == (uuid.UUID{}) || e.ID == row || e.Type != kind || e.Entity != (shared.EntityRef{Kind: "issue_relation", ID: row}) || e.WorkspaceID != acme || e.ProjectID != project || e.ActorID == nil || *e.ActorID != alice || !e.OccurredAt.Equal(clockNow) || !reflect.DeepEqual(e.Changes, want) {
		t.Fatalf("relation event=%+v changes=%+v want=%+v", e, e.Changes, want)
	}
}

func TestRelationEvents(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for _, kind := range []domain.RelationType{domain.RelatesTo, domain.Blocking, domain.Duplicate, domain.BlockedBy} {
			t.Run(line("create", reverse, kind), func(t *testing.T) {
				_, w, rel, id, r := newRelate()
				from, to, project := r.near.ID, r.far.ID, web
				if reverse {
					from, to, project = r.far.ID, r.near.ID, ops
				}
				p := &recordedEvents{t: t, before: func(shared.Event) {
					if len(rel.inserted) != 2 || rel.marked == nil || len(rel.marked.IDs) != 3 || len(w.issues.activities) != 4 {
						t.Fatal("event preceded all relation/audit/activity writes")
					}
				}}
				uc := app.NewCreateIssueRelations(app.CreateIssueRelationsDeps{Events: p, Locks: w.locks, Relations: rel, Clock: clockAt{at: clockNow, log: w.log}})
				_, err := uc.Execute(as(alice), from, kind, []uuid.UUID{to, id})
				if err != nil {
					t.Fatal(err)
				}
				if len(p.rows) != 2 {
					t.Fatalf("events=%d want two relation rows, not four activities", len(p.rows))
				}
				for n, target := range sorted([]uuid.UUID{to, id}) {
					storedFrom, storedTo, storedProject, storedType := from, target, project, string(kind)
					if kind == domain.Blocking {
						storedFrom, storedTo, storedType = target, from, "blocked_by"
						storedProject = web
						if target == r.far.ID {
							storedProject = ops
						}
					}
					wantRelationEvent(t, p.rows[n], "issue.relation_added", rel.inserted[n].ID, storedProject, storedFrom, storedTo, storedType)
				}
				if p.rows[0].ID == p.rows[1].ID || eventClockReads(w) != 1 {
					t.Fatal("relation batch identity/clock")
				}
			})
		}
		for _, alternate := range []bool{false, true} {
			t.Run(line("delete", reverse, alternate), func(t *testing.T) {
				_, w, rel := newUnrelate()
				if reverse {
					fromNear(w, rel)
				}
				r := rel.relations[0]
				if alternate {
					w.auth.answers = []error{0: shared.Forbidden(), 2: shared.Forbidden()}
				}
				p := &recordedEvents{t: t, before: func(shared.Event) {
					if rel.deleted == nil || rel.marked == nil || len(w.issues.activities) != 2 {
						t.Fatal("event preceded both relation ends")
					}
				}}
				uc := app.NewDeleteIssueRelation(app.DeleteIssueRelationDeps{Events: p, Locks: w.locks, Relations: rel, Clock: clockAt{at: clockNow, log: w.log}})
				if err := uc.Execute(as(alice), r.ID); err != nil {
					t.Fatal(err)
				}
				if len(p.rows) != 1 {
					t.Fatalf("removed events=%d want one", len(p.rows))
				}
				wantRelationEvent(t, p.rows[0], "issue.relation_removed", r.ID, r.ProjectID, r.IssueID, r.RelatedID, "blocked_by")
			})
		}
	}
}

func TestRelationEventsFailures(t *testing.T) {
	for _, remove := range []bool{false, true} {
		for _, fail := range []string{"InsertActivities", "MarkChanged", "publisher"} {
			t.Run(line("remove", remove, fail), func(t *testing.T) {
				_, w, rel, id, r := newRelate()
				p := &recordedEvents{t: t}
				if fail == "publisher" {
					p.fail = errDisk
				} else {
					w.log.fail = fail
				}
				var err error
				if remove {
					rel.relations = []domain.Relation{{ID: relationID, WorkspaceID: acme, ProjectID: ops, IssueID: r.far.ID, RelatedID: r.near.ID, Type: domain.BlockedBy}}
					err = app.NewDeleteIssueRelation(app.DeleteIssueRelationDeps{Events: p, Locks: w.locks, Relations: rel, Clock: clockAt{at: clockNow, log: w.log}}).Execute(as(alice), relationID)
				} else {
					_, err = app.NewCreateIssueRelations(app.CreateIssueRelationsDeps{Events: p, Locks: w.locks, Relations: rel, Clock: clockAt{at: clockNow, log: w.log}}).Execute(as(alice), id, domain.Blocking, []uuid.UUID{r.far.ID, r.near.ID})
				}
				if !errors.Is(err, errDisk) {
					t.Fatalf("failure lost: %v", err)
				}
				want := 0
				if fail == "publisher" {
					want = 1
				}
				if len(p.rows) != want {
					t.Fatalf("published %d want %d", len(p.rows), want)
				}
			})
		}
	}
}

func TestHierarchyEventsReadback(t *testing.T) {
	for _, lost := range []bool{false, true} {
		t.Run(line("missing after write", lost), func(t *testing.T) {
			_, w, id := newUpdate()
			k := withKin(w)
			w.issues.lost = lost
			if !lost {
				w.log.fail = "SubIssueCounts"
			}
			p := &recordedEvents{t: t}
			uc := app.NewAddSubIssues(app.AddSubIssuesDeps{Events: p, Locks: w.locks, Issues: w.issues, Clock: clockAt{at: clockNow, log: w.log}})
			if _, err := uc.Execute(as(alice), id, []uuid.UUID{k.top.ID, k.up.ID}); err == nil || len(p.rows) != 0 {
				t.Fatalf("failed readback/seen err=%v events=%d", err, len(p.rows))
			}
		})
	}
}
