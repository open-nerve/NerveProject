package app_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// nearID and farID are the ids of newRelate's other work items, near's
// drawn first, so the smaller.
var nearID, farID = uuid.NewV7(), uuid.NewV7()

// relating is the other work items of newRelate's world: near, bob's work
// item 8 of web, and far, carol's work item 3 of ops, OPS (convention 7).
type relating struct {
	near, far domain.Issue
}

// newRelate is createIssueRelations over newUpdate's world, web's work
// item 7 its work item, with near and far in the store, and ops locked by
// LockMany too, as web is.
func newRelate() (*app.CreateIssueRelations, *world, *fakeRelations, uuid.UUID, relating) {
	_, w, id := newUpdate()
	r := relating{
		near: domain.Issue{ID: nearID, WorkspaceID: acme, ProjectID: web, SequenceID: 8, Name: "Near", StateID: backlog.ID, CreatedByID: &bob},
		far:  domain.Issue{ID: farID, WorkspaceID: acme, ProjectID: ops, SequenceID: 3, Name: "Far", CreatedByID: &carol},
	}
	w.issues.others = map[uuid.UUID]domain.Issue{r.near.ID: r.near, r.far.ID: r.far}
	w.project.across = map[uuid.UUID]app.LockedProject{ops: {WorkspaceID: acme, Identifier: "OPS"}}
	rel := &fakeRelations{fakeIssues: w.issues}
	return relateOver(w, rel), w, rel, id, r
}

// relateOver is createIssueRelations over w's locks and rel.
func relateOver(w *world, rel *fakeRelations) *app.CreateIssueRelations {
	return app.NewCreateIssueRelations(app.CreateIssueRelationsDeps{Events: discardEvents{}, Locks: w.locks, Relations: rel, Clock: clockAt{at: clockNow, log: w.log}})
}

// reaching are the calls of alice's addition of relations from the work
// item id to far, then near, as Locks takes them across projects (M4
// design 3.5): the work item read and decided on, then far and near, each
// in its project, before any lock and outside the transaction; in it, web
// and ops locked at once, in id order, then the same reads and decisions
// again.
func reaching(id uuid.UUID, r relating) []string {
	var before, under []string
	for _, read := range []struct {
		item, project uuid.UUID
		action        shared.Action
	}{{id, web, domain.ActionUpdate}, {r.far.ID, ops, domain.ActionRead}, {r.near.ID, web, domain.ActionRead}} {
		before = append(before, line("GetIssue", read.item, "outside tx"), line("Authorize", alice, read.action, acme, read.project, "outside tx"))
		under = append(under, line("GetIssue", read.item), line("Authorize", alice, read.action, acme, read.project))
	}
	return slices.Concat(before, []string{"Begin", line("LockMany", anys(sorted([]uuid.UUID{web, ops}))...)}, under)
}

// An addition of relations, in the order of M4 design 3.5 and 3.10:
// Locks' calls across projects (reaching); whether the others relate to
// the work item already; the clock; the relations; the work item and the
// others marked changed, in id order; the activity; the others read back
// and counted. alice asks for far, of ops, then near, of web, as
// blocking, then as blocked_by. As blocking, each is stored the other way
// round, the other's row, of its project, its blocked_by; as blocked_by,
// as asked, the work item's row, of web. Near's comes first either way.
// Each writes a row of the work item's activity, the type asked, the other
// by its identifier, and one of the other's, of its project, the type as
// the other reads it, WEB-7, near's first. The answer is the relations as
// the work item reads them, the type asked, each other as stored then,
// alice's last change, counted as she sees it in its project: every one of
// near's, in web, and only hers of far's, in ops, where she is a guest who
// sees her own work items alone (Grant.OwnOnly of far's decisions), far
// among them, made by her here.
func TestCreateIssueRelationsRelatesTheWorkItemToOthers(t *testing.T) {
	for _, tt := range []struct {
		kind, back domain.RelationType                                // asked, and as each other reads it
		stored     func(id uuid.UUID, o domain.Issue) domain.Relation // id's relation to o as stored, but its id
	}{
		{domain.Blocking, domain.BlockedBy, func(id uuid.UUID, o domain.Issue) domain.Relation {
			return domain.Relation{WorkspaceID: acme, ProjectID: o.ProjectID, IssueID: o.ID, RelatedID: id, Type: domain.BlockedBy,
				CreatedByID: &alice, CreatedAt: clockNow}
		}},
		{domain.BlockedBy, domain.Blocking, func(id uuid.UUID, o domain.Issue) domain.Relation {
			return domain.Relation{WorkspaceID: acme, ProjectID: web, IssueID: id, RelatedID: o.ID, Type: domain.BlockedBy,
				CreatedByID: &alice, CreatedAt: clockNow}
		}},
	} {
		t.Run(string(tt.kind), func(t *testing.T) {
			uc, w, rel, id, r := newRelate()
			r.far.CreatedByID = &alice
			w.issues.others[farID] = r.far
			w.auth.grants = []shared.Grant{1: {OwnOnly: true}, 4: {OwnOnly: true}}
			w.issues.counts = map[uuid.UUID]app.SubIssueCount{r.near.ID: {All: 2, Own: 1}, r.far.ID: {All: 4, Own: 0}}
			got, err := uc.Execute(as(alice), id, tt.kind, []uuid.UUID{r.far.ID, r.near.ID})
			if err != nil {
				t.Fatal(err)
			}
			others := []uuid.UUID{r.near.ID, r.far.ID}
			calls := slices.Concat(reaching(id, r), []string{line("Related", id, others), "Now", line("InsertRelations", 2),
				line("MarkChanged", sorted([]uuid.UUID{id, r.near.ID, r.far.ID})), line("InsertActivities", 4), line("GetIssues", others),
				counted(others...)})
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			if len(rel.inserted) != 2 {
				t.Fatalf("the relations %+v, want two", rel.inserted)
			}
			for n, o := range []domain.Issue{r.near, r.far} {
				want := tt.stored(id, o)
				want.ID = rel.inserted[n].ID
				if !reflect.DeepEqual(rel.inserted[n], want) || n > 0 && rel.inserted[n].ID.Compare(rel.inserted[n-1].ID) <= 0 {
					t.Errorf("relation %d: %+v, want %+v, its id after the one before", n, rel.inserted[n], want)
				}
			}
			marked := app.ChangedRows{IDs: sorted([]uuid.UUID{id, r.near.ID, r.far.ID}), By: alice, At: clockNow}
			if rel.marked == nil || !reflect.DeepEqual(*rel.marked, marked) {
				t.Errorf("marked %+v, want %+v", rel.marked, marked)
			}
			work := domain.Ref{ID: id, Identifier: "WEB-7"}
			rows := []app.ActivityRow{
				{IssueID: id, ProjectID: web, Activity: domain.RelationAdded(tt.kind, domain.Ref{ID: r.near.ID, Identifier: "WEB-8"})},
				{IssueID: r.near.ID, ProjectID: web, Activity: domain.RelationAdded(tt.back, work)},
				{IssueID: id, ProjectID: web, Activity: domain.RelationAdded(tt.kind, domain.Ref{ID: r.far.ID, Identifier: "OPS-3"})},
				{IssueID: r.far.ID, ProjectID: ops, Activity: domain.RelationAdded(tt.back, work)},
			}
			wantActivityRows(t, w, rows)
			var want []app.IssueRelation
			for n, o := range []domain.Issue{r.near, r.far} {
				o.UpdatedByID, o.UpdatedAt, o.SubIssuesCount = &alice, clockNow, []int{2, 0}[n]
				want = append(want, app.IssueRelation{ID: rel.inserted[n].ID, Type: tt.kind, Issue: o, CreatedAt: clockNow})
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("Execute() = %+v, want %+v", got, want)
			}
		})
	}
}

// Every refusal before any lock and outside the transaction (M4 design
// 3.5): an addition domain.CheckNewRelations refuses, with no call; then a
// work item not there, or whose project the caller does not see,
// issue.not_found, and a role that may not change it, forbidden, before
// any other work item is read. Under the locks, the decision is final: web
// deleted while they waited, or the work item meanwhile, issue.not_found;
// a role that may no longer change it, forbidden; an archived project 409
// project.archived, its work item archived too, then an archived work item
// 409 issue.archived, before any other is checked. None writes a row.
func TestCreateIssueRelationsRefusals(t *testing.T) {
	for _, tt := range []struct {
		name  string
		kind  domain.RelationType
		set   func(w *world, id *uuid.UUID)
		want  error
		calls int // of reaching's, before the use case's own
	}{
		{"an unknown type", "start_before", func(*world, *uuid.UUID) {}, domain.CheckNewRelations("start_before", []uuid.UUID{web}), 0},
		{"no work item", domain.RelatesTo, func(_ *world, id *uuid.UUID) { *id = uuid.NewV7() }, domain.ErrIssueNotFound, 1},
		{"a project not seen", domain.RelatesTo, func(w *world, _ *uuid.UUID) { w.auth.answers = []error{shared.ErrNotVisible} },
			domain.ErrIssueNotFound, 2},
		{"a role not allowed", domain.RelatesTo, func(w *world, _ *uuid.UUID) { w.auth.answers = []error{shared.Forbidden()} },
			shared.Forbidden(), 2},
		{"web deleted while the locks waited", domain.RelatesTo, func(w *world, _ *uuid.UUID) { w.project.locked = nil }, domain.ErrIssueNotFound,
			8},
		{"the work item deleted meanwhile", domain.RelatesTo, func(w *world, _ *uuid.UUID) {
			w.project.waited = func() { w.issues.issue.ID = uuid.NewV7() }
		}, domain.ErrIssueNotFound, 9},
		{"no longer allowed", domain.RelatesTo, func(w *world, _ *uuid.UUID) { w.auth.answers = []error{3: shared.Forbidden()} },
			shared.Forbidden(), 10},
		{"an archived project", domain.RelatesTo, func(w *world, _ *uuid.UUID) { w.project.locked.Archived, w.issues.archivedAt = true, &earlier },
			domain.ErrProjectArchived, 14},
		{"an archived work item", domain.RelatesTo, func(w *world, _ *uuid.UUID) { w.issues.archivedAt = &earlier }, domain.ErrIssueArchived, 14},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, rel, id, r := newRelate()
			tt.set(w, &id)
			_, err := uc.Execute(as(alice), id, tt.kind, []uuid.UUID{r.far.ID, r.near.ID})
			wantRefusal(t, w, err, tt.want, reaching(id, r)[:tt.calls])
			wantNothingWritten(t, w, rel)
		})
	}
}

// wantNothingWritten checks that a write of relations over w and rel wrote
// nothing: no relation inserted or deleted, no work item marked changed,
// no row of activity.
func wantNothingWritten(t *testing.T, w *world, rel *fakeRelations) {
	t.Helper()
	if rel.inserted != nil || rel.deleted != nil || rel.marked != nil || w.issues.activities != nil {
		t.Errorf("wrote %+v, %+v, %+v, %+v; want nothing", rel.inserted, rel.deleted, rel.marked, w.issues.activities)
	}
}

// Each work item an addition names that the work item may not relate to
// is refused at its place in issue_ids (M4 design 3.4, 3.8, 3.10), and
// nothing is written. alice asks for far, then near. Not reached before
// any lock: one not there, the work item itself, another workspace's, one
// whose project she does not see, one she may not read, or another's when
// the guest rule hides others' from her in its project; under the locks,
// one archived, of an archived project, deleted while the locks waited,
// whose project was deleted while they waited, or no longer seen. One not
// reached before the locks is not reached under them either, though she
// sees it there (M4 design 3.5): near, hidden from her before them. Under
// them too, one that relates to the work item already is a duplicate.
// Every problem is reported at once, each at its place: far's and near's
// both, when both are refused.
func TestCreateIssueRelationsRefusesWhatItMayNotRelateTo(t *testing.T) {
	notAllowed, already := shared.Invalid(domain.RelatedNotAllowed(0)), shared.Invalid(domain.RelatedAlready(0))
	for _, tt := range []struct {
		name string
		set  func(c *relateCase)
		want error
	}{
		{"one not there", func(c *relateCase) { c.ask[0] = uuid.NewV7() }, notAllowed},
		{"the work item itself", func(c *relateCase) { c.ask[0] = c.id }, notAllowed},
		{"another workspace's", func(c *relateCase) { c.far.WorkspaceID = uuid.NewV7() }, notAllowed},
		{"of a project not seen", func(c *relateCase) { c.auth.answers = []error{1: shared.ErrNotVisible} }, notAllowed},
		{"not to be read", func(c *relateCase) { c.auth.answers = []error{1: shared.Forbidden()} }, notAllowed},
		{"another's, to a guest of its project who sees his own alone", func(c *relateCase) {
			c.auth.grants = []shared.Grant{1: {OwnOnly: true}}
		}, notAllowed},
		{"archived", func(c *relateCase) { c.far.ArchivedAt = &earlier }, notAllowed},
		{"of an archived project", func(c *relateCase) {
			c.project.across[ops] = app.LockedProject{WorkspaceID: acme, Archived: true, Identifier: "OPS"}
		}, notAllowed},
		{"deleted while the locks waited", func(c *relateCase) { c.project.waited = func() { delete(c.issues.others, farID) } }, notAllowed},
		{"its project deleted while the locks waited", func(c *relateCase) { delete(c.project.across, ops) }, notAllowed},
		{"no longer seen", func(c *relateCase) { c.auth.answers = []error{4: shared.ErrNotVisible} }, notAllowed},
		{"near, not seen before the locks, seen under them", func(c *relateCase) { c.auth.answers = []error{2: shared.ErrNotVisible} },
			shared.Invalid(domain.RelatedNotAllowed(1))},
		{"related already", func(c *relateCase) { c.rel.related = []uuid.UUID{farID} }, already},
		{"both refused", func(c *relateCase) { c.far.ArchivedAt, c.rel.related = &earlier, []uuid.UUID{nearID} },
			shared.Invalid(domain.RelatedNotAllowed(0), domain.RelatedAlready(1))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, rel, id, r := newRelate()
			c := &relateCase{world: w, rel: rel, id: id, far: r.far, ask: []uuid.UUID{farID, nearID}}
			tt.set(c)
			w.issues.others[farID] = c.far
			_, err := uc.Execute(as(alice), id, domain.RelatesTo, c.ask)
			wantRefused(t, err, tt.want)
			wantNothingWritten(t, w, rel)
		})
	}
}

// relateCase is newRelate's world as a case sets it: far is stored as the
// case leaves it, and the addition asks for ask, far and near unless the
// case changes it.
type relateCase struct {
	*world
	rel *fakeRelations
	id  uuid.UUID
	far domain.Issue
	ask []uuid.UUID
}

// A port that fails ends the addition with its failure, as itself, and no
// call after it runs: the transaction rolls back whatever was written. It
// fails, at its first call, the work item's read and its decision, far's
// read and its decision before the locks, the lock, the read of the
// relations there already, the three writes, the read back and the count;
// and, once the precheck's calls have passed, far's read and its decision
// again under the locks (reachUnder), after reaching's calls up to it.
func TestCreateIssueRelationsStopsAtAFailingPort(t *testing.T) {
	for _, port := range []string{"GetIssue", "Authorize", line("GetIssue", farID), line("Authorize", alice, domain.ActionRead, acme, ops),
		"LockMany", "Related", "InsertRelations", "MarkChanged", "InsertActivities", "GetIssues", "SubIssueCounts"} {
		t.Run(port, func(t *testing.T) {
			uc, w, _, id, _ := newRelate()
			w.log.fail = port
			_, err := uc.Execute(as(alice), id, domain.RelatesTo, []uuid.UUID{farID, nearID})
			wantStoppedAt(t, w, err, port)
		})
	}
	for _, tt := range []struct {
		name, port string
		calls      int // of reaching's, the port's the last
	}{
		{"far read again under the locks", line("GetIssue", farID), 11},
		{"far decided on again under the locks", line("Authorize", alice, domain.ActionRead, acme, ops), 12},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, _, id, r := newRelate()
			w.project.waited = func() { w.log.fail = tt.port }
			_, err := uc.Execute(as(alice), id, domain.RelatesTo, []uuid.UUID{farID, nearID})
			wantStopped(t, w, err, tt.port, slices.Equal(w.log.calls, reaching(id, r)[:tt.calls]))
		})
	}
}

// What cannot be is an error, never a refusal: the others not there to
// read back after the write (fakeIssues' lost), which found each undeleted
// under the locks.
func TestCreateIssueRelationsErrsOnWhatCannotBe(t *testing.T) {
	uc, _, rel, id, r := newRelate()
	rel.lost = true
	_, err := uc.Execute(as(alice), id, domain.RelatesTo, []uuid.UUID{r.far.ID, r.near.ID})
	var se *shared.Error
	if err == nil || errors.As(err, &se) {
		t.Errorf("Execute() = %v, want an error that is no refusal", err)
	}
}
