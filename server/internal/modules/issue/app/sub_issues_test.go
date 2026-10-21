package app_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// listOver is listSubIssues over w's ports.
func listOver(w *world) *app.ListSubIssues {
	return app.NewListSubIssues(w.locks, w.issues)
}

// addOver is addSubIssues over w's ports.
func addOver(w *world) *app.AddSubIssues {
	return app.NewAddSubIssues(app.AddSubIssuesDeps{Locks: w.locks, Issues: w.issues, Clock: clockAt{at: clockNow, log: w.log}})
}

// putUnder stores the work item i of w's store under the work item
// parent.
func putUnder(w *world, i domain.Issue, parent uuid.UUID) {
	i.ParentID = &parent
	w.issues.others[i.ID] = i
}

// sorted are ids in id order.
func sorted(ids []uuid.UUID) []uuid.UUID {
	return slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare)
}

// A list of sub-items, outside any transaction and without a lock (M4
// design 3.4, 3.9): web's work item 7, alice's, read and issue.read
// decided on it, as getIssue does; then its sub-items, anyone's for web's
// admin, only hers for a guest who sees only her own, and for an admin who
// is only web's guest (adminGuest); then their sub-items counted, as she
// sees them: every one, or only hers. The answer is the sub-items as the
// store answers them, each with its count.
func TestListSubIssuesReadsTheSubItemsTheCallerSees(t *testing.T) {
	for _, tt := range []struct {
		name  string
		grant shared.Grant
		whose string
		items func(k kin) []domain.Issue // as the store answers them
		want  []int                      // their counts
	}{
		{"an admin", shared.Grant{ProjectRole: shared.RoleAdmin}, "anyone's", func(k kin) []domain.Issue { return []domain.Issue{k.top, k.up} },
			[]int{3, 2}},
		{"a guest who sees her own", ownOnly, alice.String() + "'s", func(k kin) []domain.Issue { return []domain.Issue{k.mine} }, []int{1}},
		{"an admin who is a guest", adminGuest, alice.String() + "'s", func(k kin) []domain.Issue { return []domain.Issue{k.mine} }, []int{1}},
	} {
		_, w, id := newUpdate()
		k := withKin(w)
		items := tt.items(k)
		w.issues.issue.By, w.auth.grants, w.issues.subItems = alice, []shared.Grant{tt.grant}, items
		w.issues.counts = map[uuid.UUID]app.SubIssueCount{k.top.ID: {All: 3}, k.up.ID: {All: 2, Own: 2}, k.mine.ID: {All: 4, Own: 1}}
		got, err := listOver(w).Execute(as(alice), id)
		want, ids := slices.Clone(items), []uuid.UUID{}
		for n := range want {
			want[n].SubIssuesCount = tt.want[n]
			ids = append(ids, want[n].ID)
		}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("%s: Execute() = %+v, %v; want %+v", tt.name, got, err, want)
		}
		calls := []string{line("GetIssue", id, "outside tx"), line("Authorize", alice, domain.ActionRead, acme, web, "outside tx"),
			line("SubIssues", id, tt.whose, "outside tx"), counted(ids...) + " outside tx"}
		if !slices.Equal(w.log.calls, calls) {
			t.Errorf("%s: calls:\n%q\nwant\n%q", tt.name, w.log.calls, calls)
		}
	}
}

// What refuses a list of sub-items is what refuses getIssue, before any
// sub-item is read: a work item not there, or whose project the caller
// does not see, or that the guest rule hides from him, issue.not_found; a
// role that may not read it, forbidden. A read or a count that fails ends
// it with its failure; a work item without sub-items counts none.
func TestListSubIssuesRefusals(t *testing.T) {
	for _, tt := range []struct {
		name  string
		set   func(w *world, id uuid.UUID) uuid.UUID // the work item asked for
		want  error
		calls int // of the read, the decision, the list and the count
	}{
		{"no work item", func(*world, uuid.UUID) uuid.UUID { return uuid.NewV7() }, domain.ErrIssueNotFound, 1},
		{"a project not seen", func(w *world, id uuid.UUID) uuid.UUID {
			w.auth.answers = []error{shared.ErrNotVisible}
			return id
		}, domain.ErrIssueNotFound, 2},
		{"another's, to a guest who sees only his own", func(w *world, id uuid.UUID) uuid.UUID {
			w.auth.grants = []shared.Grant{ownOnly}
			return id
		}, domain.ErrIssueNotFound, 2},
		{"a role not allowed", func(w *world, id uuid.UUID) uuid.UUID {
			w.auth.answers = []error{shared.Forbidden()}
			return id
		}, shared.Forbidden(), 2},
		{"the sub-items' read fails", func(w *world, id uuid.UUID) uuid.UUID { w.log.fail = "SubIssues"; return id }, errDisk, 3},
		{"the count fails", func(w *world, id uuid.UUID) uuid.UUID { w.log.fail = "SubIssueCounts"; return id }, errDisk, 4},
		{"no sub-items", func(w *world, id uuid.UUID) uuid.UUID { w.issues.subItems = nil; return id }, nil, 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w, id := newUpdate()
			k := withKin(w)
			w.issues.subItems = []domain.Issue{k.up}
			asked := tt.set(w, id)
			got, err := listOver(w).Execute(as(alice), asked)
			if !reflect.DeepEqual(err, tt.want) || len(got) != 0 {
				t.Errorf("Execute() = %+v, %v; want none and %v", got, err, tt.want)
			}
			calls := []string{line("GetIssue", asked, "outside tx"), line("Authorize", alice, domain.ActionRead, acme, web, "outside tx"),
				line("SubIssues", id, "anyone's", "outside tx"), counted(k.up.ID) + " outside tx"}[:tt.calls]
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
		})
	}
}

// An addition of sub-items, in the order of M4 design 3.5: Locks' calls on
// the parent, web's work item 7, issue.update decided on it (lockedAs);
// the work items asked for, read in id order, and the parent's ancestry;
// the parents before of those that move, read; the clock; those that move
// under it, in id order, though asked in another, each a row of its
// activity, alice's at the moment, from its parent before by its
// identifier to WEB-7: up, from none; top, from up, WEB-3. mine, under 7
// already, moves not and writes no row. The answer is those that moved,
// read back, in id order, each under 7, changed last by alice then, its
// sub-items counted.
func TestAddSubIssuesPutsTheWorkItemsUnderIt(t *testing.T) {
	_, w, id := newUpdate()
	k := withKin(w)
	putUnder(w, k.top, k.up.ID)
	putUnder(w, k.mine, id)
	w.issues.counts = map[uuid.UUID]app.SubIssueCount{k.top.ID: {All: 4, Own: 1}}
	asked := []uuid.UUID{k.mine.ID, k.top.ID, k.up.ID}
	got, err := addOver(w).Execute(as(alice), id, asked)
	moved := []uuid.UUID{k.up.ID, k.top.ID}
	var want []domain.Issue
	for _, m := range moved {
		i := w.issues.others[m]
		i.SubIssuesCount = w.issues.counts[m].All
		want = append(want, i)
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Execute() = %+v, %v; want %+v", got, err, want)
	}
	for _, i := range got {
		if !reflect.DeepEqual(i.ParentID, &id) || !reflect.DeepEqual(i.UpdatedByID, &alice) || !i.UpdatedAt.Equal(clockNow) {
			t.Errorf("%s answered under %v, changed last by %v at %v; want under %s, by alice at %v", i.ID, i.ParentID, i.UpdatedByID,
				i.UpdatedAt, id, clockNow)
		}
	}
	calls := append(lockedAs(domain.ActionUpdate, id), line("GetIssues", sorted(asked)), line("Ancestry", id),
		line("GetIssues", []uuid.UUID{k.up.ID}), "Now", line("SetParents", moved, id), line("InsertActivities", 2), line("GetIssues", moved),
		counted(moved...))
	if !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
	if want := (app.ParentRows{IDs: moved, ParentID: id, By: alice, At: clockNow}); w.issues.parented == nil ||
		!reflect.DeepEqual(*w.issues.parented, want) {
		t.Errorf("the rows written %+v, want %+v", w.issues.parented, want)
	}
	seven := &domain.Ref{ID: id, Identifier: "WEB-7"}
	wantActivityRows(t, w, []app.ActivityRow{
		{ProjectID: web, IssueID: k.up.ID, Activity: domain.ParentActivity(domain.Changed[*domain.Ref]{New: seven})},
		{ProjectID: web, IssueID: k.top.ID, Activity: domain.ParentActivity(domain.Changed[*domain.Ref]{Old: &domain.Ref{ID: k.up.ID,
			Identifier: "WEB-3"}, New: seven})},
	})
}

// An addition of work items each under the work item already writes
// nothing, reads not even the clock, and answers none (M4 design 3.9).
func TestAddingSubItemsUnderItAlreadyWritesNothing(t *testing.T) {
	_, w, id := newUpdate()
	k := withKin(w)
	putUnder(w, k.up, id)
	got, err := addOver(w).Execute(as(alice), id, []uuid.UUID{k.up.ID})
	calls := append(lockedAs(domain.ActionUpdate, id), line("GetIssues", []uuid.UUID{k.up.ID}), line("Ancestry", id))
	if err != nil || len(got) != 0 || !slices.Equal(w.log.calls, calls) || w.issues.parented != nil || w.issues.activities != nil {
		t.Errorf("Execute() = %+v, %v after\n%q; want none, after\n%q, nothing written", got, err, w.log.calls, calls)
	}
}

// A guest who sees only her own work items, alice, puts her mine under her
// work item 7 (M4 design 3.4, 3.9): the answer counts mine's sub-items as
// she sees them, by the decision under the lock: hers alone.
func TestAddSubIssuesCountsTheSubItemsTheCallerSees(t *testing.T) {
	_, w, id := newUpdate()
	k := withKin(w)
	w.issues.issue.By, w.auth.grants = alice, []shared.Grant{ownOnly, ownOnly}
	w.issues.counts = map[uuid.UUID]app.SubIssueCount{k.mine.ID: {All: 4, Own: 1}}
	got, err := addOver(w).Execute(as(alice), id, []uuid.UUID{k.mine.ID})
	if err != nil || len(got) != 1 || got[0].ID != k.mine.ID || got[0].SubIssuesCount != 1 {
		t.Errorf("Execute() = %+v, %v; want mine, with the one sub-item of hers", got, err)
	}
}

// What refuses an addition of sub-items (M4 design 3.5, 3.8, 3.9): before
// any read, none named, or one named twice; before any lock, what refuses
// updateIssue's precheck; under the lock, an archived project, then an
// archived parent; then each work item asked for that is not there,
// another project's, archived, the parent itself, one above it, or
// another's for a caller who sees only his own, by the decision under the
// lock, an admin who is only web's guest too (adminGuest), refused as one
// not there is: each at its place, in one 422, a work item it may move
// among them. Nothing is written.
func TestAddSubIssuesRefusals(t *testing.T) {
	gone := uuid.NewV7()
	checked := func(id uuid.UUID, asked []uuid.UUID) []string {
		return append(locked(id), line("GetIssues", sorted(asked)), line("Ancestry", id))
	}
	for _, tt := range []struct {
		name  string
		set   func(w *world, k kin, id uuid.UUID) (parent uuid.UUID, asked []uuid.UUID)
		want  error
		calls func(parent uuid.UUID, asked []uuid.UUID) []string
	}{
		{"none named", func(_ *world, _ kin, id uuid.UUID) (uuid.UUID, []uuid.UUID) { return id, nil }, domain.CheckSubIssueIDs(nil),
			func(uuid.UUID, []uuid.UUID) []string { return nil }},
		{"one named twice", func(_ *world, k kin, id uuid.UUID) (uuid.UUID, []uuid.UUID) { return id, []uuid.UUID{k.up.ID, k.up.ID} },
			domain.CheckSubIssueIDs(make([]uuid.UUID, 2)), func(uuid.UUID, []uuid.UUID) []string { return nil }},
		{"a parent not there", func(_ *world, k kin, _ uuid.UUID) (uuid.UUID, []uuid.UUID) { return gone, []uuid.UUID{k.up.ID} },
			domain.ErrIssueNotFound, func(parent uuid.UUID, _ []uuid.UUID) []string { return locked(parent)[:1] }},
		{"a role not allowed", func(w *world, k kin, id uuid.UUID) (uuid.UUID, []uuid.UUID) {
			w.auth.answers = []error{shared.Forbidden()}
			return id, []uuid.UUID{k.up.ID}
		}, shared.Forbidden(), func(parent uuid.UUID, _ []uuid.UUID) []string { return locked(parent)[:2] }},
		{"an archived project", func(w *world, k kin, id uuid.UUID) (uuid.UUID, []uuid.UUID) {
			w.project.locked.Archived, w.issues.archivedAt = true, &earlier
			return id, []uuid.UUID{k.up.ID}
		}, domain.ErrProjectArchived, func(parent uuid.UUID, _ []uuid.UUID) []string { return locked(parent) }},
		{"an archived parent", func(w *world, _ kin, id uuid.UUID) (uuid.UUID, []uuid.UUID) {
			w.issues.archivedAt = &earlier
			return id, []uuid.UUID{gone}
		}, domain.ErrIssueArchived, func(parent uuid.UUID, _ []uuid.UUID) []string { return locked(parent) }},
		{"each one it may not move", func(w *world, k kin, id uuid.UUID) (uuid.UUID, []uuid.UUID) {
			w.issues.ancestry = map[uuid.UUID][]uuid.UUID{id: {id, k.mine.ID}}
			return id, []uuid.UUID{gone, k.far.ID, k.top.ID, k.shelved.ID, id, k.mine.ID}
		}, shared.Invalid(domain.SubIssueNotAllowed(0), domain.SubIssueNotAllowed(1), domain.SubIssueNotAllowed(3), domain.SubIssueNotAllowed(4),
			domain.SubIssueNotAllowed(5)), checked},
		{"another's, for one who sees her own alone", func(w *world, k kin, id uuid.UUID) (uuid.UUID, []uuid.UUID) {
			w.issues.issue.By, w.auth.grants = alice, []shared.Grant{{}, ownOnly}
			return id, []uuid.UUID{k.mine.ID, k.up.ID}
		}, shared.Invalid(domain.SubIssueNotAllowed(1)), checked},
		{"another's, for an admin who is a guest", func(w *world, k kin, id uuid.UUID) (uuid.UUID, []uuid.UUID) {
			w.issues.issue.By, w.auth.grants = alice, []shared.Grant{adminGuest, adminGuest}
			return id, []uuid.UUID{gone, k.up.ID, k.mine.ID}
		}, shared.Invalid(domain.SubIssueNotAllowed(0), domain.SubIssueNotAllowed(1)), checked},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w, id := newUpdate()
			k := withKin(w)
			parent, asked := tt.set(w, k, id)
			got, err := addOver(w).Execute(as(alice), parent, asked)
			wantRefusal(t, w, err, tt.want, tt.calls(parent, asked))
			if len(got) != 0 || w.issues.parented != nil || w.issues.activities != nil {
				t.Errorf("Execute() = %+v, written %+v and %+v; want none, nothing written", got, w.issues.parented, w.issues.activities)
			}
		})
	}
}

// A port that fails ends the addition with its failure, as itself, and no
// call after it runs: the work items asked for read, the ancestry, the
// parents before read, the write, the activity, the read back, the count.
func TestAddSubIssuesStopsAtAFailingPort(t *testing.T) {
	for _, at := range []string{"asked", "Ancestry", "before", "SetParents", "InsertActivities", "read back", "SubIssueCounts"} {
		t.Run(at, func(t *testing.T) {
			_, w, id := newUpdate()
			k := withKin(w)
			putUnder(w, k.top, k.up.ID)
			putUnder(w, k.mine, id)
			asked := []uuid.UUID{k.top.ID, k.mine.ID}
			w.log.fail = map[string]string{"asked": line("GetIssues", sorted(asked)), "Ancestry": line("Ancestry", id),
				"before": line("GetIssues", []uuid.UUID{k.up.ID}), "SetParents": line("SetParents", []uuid.UUID{k.top.ID}, id),
				"InsertActivities": line("InsertActivities", 1), "read back": line("GetIssues", []uuid.UUID{k.top.ID}),
				"SubIssueCounts": counted(k.top.ID)}[at]
			_, err := addOver(w).Execute(as(alice), id, asked)
			wantStoppedOn(t, w, err, w.log.fail)
		})
	}
}

// losing is a store that loses the work items whose activity it writes,
// so that they are not there to read back.
type losing struct{ *fakeIssues }

func (l losing) InsertActivities(ctx context.Context, rows []app.ActivityRow) error {
	for _, r := range rows {
		delete(l.others, r.IssueID)
	}
	return l.fakeIssues.InsertActivities(ctx, rows)
}

// What cannot be is an error, not a refusal: a work item under one that is
// not there, before the clock, since deleting a work item deletes those
// under it (M4 design 3.6); a work item moved that is not there to read
// back after its write.
func TestAddSubIssuesErrsOnWhatCannotBe(t *testing.T) {
	var se *shared.Error
	_, w, id := newUpdate()
	k := withKin(w)
	putUnder(w, k.top, uuid.NewV7())
	if _, err := addOver(w).Execute(as(alice), id, []uuid.UUID{k.top.ID}); err == nil || errors.As(err, &se) ||
		slices.Contains(w.log.calls, "Now") {
		t.Errorf("Execute() of a work item under one not there = %v after %q; want an error before the clock", err, w.log.calls)
	}
	_, w, id = newUpdate()
	k = withKin(w)
	add := app.NewAddSubIssues(app.AddSubIssuesDeps{Locks: w.locks, Issues: losing{w.issues}, Clock: clockAt{at: clockNow, log: w.log}})
	if got, err := add.Execute(as(alice), id, []uuid.UUID{k.top.ID}); err == nil || errors.As(err, &se) || len(got) != 0 {
		t.Errorf("Execute() of a work item lost after its write = %+v, %v; want an error", got, err)
	}
}
