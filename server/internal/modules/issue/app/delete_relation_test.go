package app_test

import (
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// relationID is the id of newUnrelate's relation.
var relationID = uuid.NewV7()

// newUnrelate is deleteIssueRelation over newRelate's world, with a
// relation of near and far stored: far blocked by near, stored from far,
// made by carol at earlier.
func newUnrelate() (*app.DeleteIssueRelation, *world, *fakeRelations) {
	_, w, rel, _, _ := newRelate()
	rel.relations = []domain.Relation{{ID: relationID, WorkspaceID: acme, ProjectID: ops, IssueID: farID, RelatedID: nearID,
		Type: domain.BlockedBy, CreatedByID: &carol, CreatedAt: earlier}}
	return unrelateOver(w, rel), w, rel
}

// unrelateOver is deleteIssueRelation over w's locks and rel.
func unrelateOver(w *world, rel *fakeRelations) *app.DeleteIssueRelation {
	return app.NewDeleteIssueRelation(app.DeleteIssueRelationDeps{Locks: w.locks, Relations: rel, Clock: clockAt{at: clockNow, log: w.log}})
}

// unrelating are the calls of alice's deletion of newUnrelate's relation,
// as Locks takes them on either work item (M4 design 3.5, 3.10): before
// any lock and outside the transaction, the relation read, both work items
// read, far first, stored as issue_id, then near, and the decision on far,
// in ops, then, when that refuses, on near, in web, before of them; in the
// transaction, web and ops locked at once, in id order, both work items
// read again, the decisions again, under of them, and the relation read
// again.
func unrelating(before, under int) []string {
	reads := []string{line("GetIssue", farID), line("GetIssue", nearID)}
	decisions := []string{line("Authorize", alice, domain.ActionUpdate, acme, ops), line("Authorize", alice, domain.ActionUpdate, acme, web)}
	var outside []string
	for _, c := range slices.Concat([]string{line("GetRelation", relationID)}, reads, decisions[:before]) {
		outside = append(outside, c+" outside tx")
	}
	return slices.Concat(outside, []string{"Begin", line("LockMany", anys(sorted([]uuid.UUID{web, ops}))...)}, reads, decisions[:under],
		[]string{line("GetRelation", relationID)})
}

// through is calls up to call, the last; all of them for "".
func through(calls []string, call string) []string {
	if call == "" {
		return calls
	}
	return calls[:slices.Index(calls, call)+1]
}

// archived archives the work item id of w's others.
func archived(w *world, id uuid.UUID) {
	i := w.issues.others[id]
	i.ArchivedAt = &earlier
	w.issues.others[id] = i
}

// A deletion of a relation, in the order of M4 design 3.5 and 3.10:
// Locks' calls on either work item (unrelating); the clock; the relation
// deleted; both work items marked changed, in id order; the activity. The
// caller may change either work item: far, stored as issue_id, decided on
// first, or, when he may not change it, near, decided on next, whether his
// role may not, or the guest rule hides far from him in ops. Each writes
// a row of far's activity first, of ops, its blocked_by WEB-8 deleted,
// then one of near's, of web, its blocking OPS-3 deleted, both alice's at
// the clock's moment.
func TestDeleteIssueRelationDeletesItOnEitherWorkItem(t *testing.T) {
	for _, tt := range []struct {
		name          string
		set           func(w *world)
		before, under int
	}{
		{"decided on far", func(*world) {}, 1, 1},
		{"far not allowed, decided on near", func(w *world) { w.auth.answers = []error{0: shared.Forbidden(), 2: shared.Forbidden()} }, 2, 2},
		{"far hidden by the guest rule, decided on near", func(w *world) {
			w.auth.grants = []shared.Grant{0: {OwnOnly: true}, 2: {OwnOnly: true}}
		}, 2, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, rel := newUnrelate()
			tt.set(w)
			if err := uc.Execute(as(alice), relationID); err != nil {
				t.Fatal(err)
			}
			changed := sorted([]uuid.UUID{nearID, farID})
			calls := slices.Concat(unrelating(tt.before, tt.under), []string{"Now", line("DeleteRelation", relationID), line("MarkChanged", changed),
				line("InsertActivities", 2)})
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			if want := (app.RelationDeletion{ID: relationID, By: alice, At: clockNow}); rel.deleted == nil || !reflect.DeepEqual(*rel.deleted, want) {
				t.Errorf("deleted %+v, want %+v", rel.deleted, want)
			}
			if want := (app.ChangedRows{IDs: changed, By: alice, At: clockNow}); rel.marked == nil || !reflect.DeepEqual(*rel.marked, want) {
				t.Errorf("marked %+v, want %+v", rel.marked, want)
			}
			rows := []app.ActivityRow{
				{IssueID: farID, ProjectID: ops, Activity: domain.RelationRemoved(domain.BlockedBy, domain.Ref{ID: nearID, Identifier: "WEB-8"})},
				{IssueID: nearID, ProjectID: web, Activity: domain.RelationRemoved(domain.Blocking, domain.Ref{ID: farID, Identifier: "OPS-3"})},
			}
			if len(w.issues.activities) != len(rows) {
				t.Fatalf("the activity %+v, want %d rows", w.issues.activities, len(rows))
			}
			for n, row := range w.issues.activities {
				want := rows[n]
				want.ID, want.WorkspaceID, want.ActorID, want.At = row.ID, acme, alice, clockNow
				if !reflect.DeepEqual(row, want) || n > 0 && row.ID.Compare(w.issues.activities[n-1].ID) <= 0 {
					t.Errorf("activity row %d: %+v, want %+v, its id after the row before", n, row, want)
				}
			}
		})
	}
}

// Every refusal of a deletion (M4 design 3.5, 3.10), none of which writes
// a row. Before any lock and outside the transaction: no relation, or a
// work item of it not there, issue.relation_not_found; neither work item
// seen, by its project or by the guest rule, issue.relation_not_found too,
// and a role that may change neither, when he sees one, forbidden. Under
// the locks, the decision is final: either project deleted while they
// waited, before any decision, either work item deleted meanwhile, or the
// relation, or neither seen any more, issue.relation_not_found; neither to
// be changed any more, forbidden; either work item of an archived project
// 409 project.archived, then either archived work item 409 issue.archived.
// Each answer is compared whole: a relation neither of whose work items
// the caller sees is the very issue.relation_not_found of one not there,
// detail and all (8.4).
func TestDeleteIssueRelationRefusals(t *testing.T) {
	notFound, forbidden, hidden := domain.ErrRelationNotFound, shared.Forbidden(), shared.ErrNotVisible
	update := func(project uuid.UUID) string { return line("Authorize", alice, domain.ActionUpdate, acme, project) }
	for _, tt := range []struct {
		name          string
		set           func(w *world, rel *fakeRelations)
		want          error
		before, under int
		last          string // the last call of unrelating's, "" for all
	}{
		{"no relation", func(_ *world, rel *fakeRelations) { rel.relations = nil }, notFound, 0, 0,
			line("GetRelation", relationID, "outside tx")},
		{"far not there", func(w *world, _ *fakeRelations) { delete(w.issues.others, farID) }, notFound, 0, 0,
			line("GetIssue", farID, "outside tx")},
		{"near not there", func(w *world, _ *fakeRelations) { delete(w.issues.others, nearID) }, notFound, 0, 0,
			line("GetIssue", nearID, "outside tx")},
		{"neither seen", func(w *world, _ *fakeRelations) { w.auth.answers = []error{hidden, hidden} }, notFound, 2, 0, update(web) + " outside tx"},
		{"far not seen, near hidden by the guest rule", func(w *world, _ *fakeRelations) {
			w.auth.answers, w.auth.grants = []error{hidden}, []shared.Grant{1: {OwnOnly: true}}
		}, notFound, 2, 0, update(web) + " outside tx"},
		{"neither to be changed", func(w *world, _ *fakeRelations) { w.auth.answers = []error{forbidden, forbidden} }, forbidden, 2, 0,
			update(web) + " outside tx"},
		{"far not seen, near not to be changed", func(w *world, _ *fakeRelations) { w.auth.answers = []error{hidden, forbidden} }, forbidden, 2, 0,
			update(web) + " outside tx"},
		{"far not to be changed, near not seen", func(w *world, _ *fakeRelations) { w.auth.answers = []error{forbidden, hidden} }, forbidden, 2, 0,
			update(web) + " outside tx"},
		{"ops deleted while the locks waited, near no longer to be changed", func(w *world, _ *fakeRelations) {
			w.project.waited = func() { delete(w.project.across, ops) }
			w.auth.answers = []error{1: hidden, 2: forbidden}
		}, notFound, 1, 0, line("GetIssue", nearID)},
		{"web deleted while the locks waited, far no longer to be changed", func(w *world, _ *fakeRelations) {
			w.project.waited = func() { w.project.locked = nil }
			w.auth.answers = []error{1: forbidden, 2: hidden}
		}, notFound, 1, 0, line("GetIssue", nearID)},
		{"far deleted meanwhile", func(w *world, _ *fakeRelations) { w.project.waited = func() { delete(w.issues.others, farID) } }, notFound, 1, 0,
			line("GetIssue", farID)},
		{"the relation deleted meanwhile", func(w *world, rel *fakeRelations) { w.project.waited = func() { rel.relations = nil } }, notFound, 1, 1,
			""},
		{"neither seen any more", func(w *world, _ *fakeRelations) { w.auth.answers = []error{1: hidden, 2: hidden} }, notFound, 1, 2, update(web)},
		{"neither to be changed any more", func(w *world, _ *fakeRelations) { w.auth.answers = []error{1: forbidden, 2: forbidden} }, forbidden, 1,
			2, update(web)},
		{"ops archived", func(w *world, _ *fakeRelations) {
			w.project.across[ops] = app.LockedProject{WorkspaceID: acme, Archived: true, Identifier: "OPS"}
		}, domain.ErrProjectArchived, 1, 1, ""},
		{"web archived", func(w *world, _ *fakeRelations) { w.project.locked.Archived = true }, domain.ErrProjectArchived, 1, 1, ""},
		{"far archived", func(w *world, _ *fakeRelations) { archived(w, farID) }, domain.ErrIssueArchived, 1, 1, ""},
		{"near archived", func(w *world, _ *fakeRelations) { archived(w, nearID) }, domain.ErrIssueArchived, 1, 1, ""},
		{"web archived, far archived too", func(w *world, _ *fakeRelations) {
			w.project.locked.Archived = true
			archived(w, farID)
		}, domain.ErrProjectArchived, 1, 1, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, rel := newUnrelate()
			tt.set(w, rel)
			err := uc.Execute(as(alice), relationID)
			wantRefusal(t, w, err, tt.want, through(unrelating(tt.before, tt.under), tt.last))
			if !reflect.DeepEqual(err, tt.want) {
				t.Errorf("Execute() = %#v, want %#v whole", err, tt.want)
			}
			if rel.deleted != nil || rel.marked != nil || w.issues.activities != nil {
				t.Errorf("wrote %+v, %+v, %+v; want nothing", rel.deleted, rel.marked, w.issues.activities)
			}
		})
	}
}

// A port that fails ends the deletion with its failure, as itself, and no
// call after it runs: the transaction rolls back whatever was written.
func TestDeleteIssueRelationStopsAtAFailingPort(t *testing.T) {
	for _, port := range []string{"GetRelation", "GetIssue", line("GetIssue", nearID), "Authorize", "LockMany", "DeleteRelation", "MarkChanged",
		"InsertActivities"} {
		t.Run(port, func(t *testing.T) {
			uc, w, _ := newUnrelate()
			w.log.fail = port
			wantStoppedAt(t, w, uc.Execute(as(alice), relationID), port)
		})
	}
}
