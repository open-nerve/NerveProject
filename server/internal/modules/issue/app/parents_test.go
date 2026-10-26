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

// kin are the work items a write may put another under, or may not (M4
// design 3.9), in web's world: up, web's work item 3, bob's; top, its work
// item 12, carol's; mine, its work item 10, alice's; shelved, its work
// item 9, archived; far, a work item of another project.
type kin struct{ up, top, mine, shelved, far domain.Issue }

// withKin stores kin in w, besides the work item w keeps, and answers it.
func withKin(w *world) kin {
	made := func(number int, by uuid.UUID) domain.Issue {
		return domain.Issue{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, SequenceID: number, Name: "Part", StateID: backlog.ID,
			CreatedByID: &by}
	}
	k := kin{up: made(3, bob), top: made(12, carol), mine: made(10, alice), shelved: made(9, carol), far: made(4, carol)}
	k.shelved.ArchivedAt, k.far.ProjectID = &earlier, uuid.NewV7()
	w.issues.others = map[uuid.UUID]domain.Issue{}
	for _, i := range []domain.Issue{k.up, k.top, k.mine, k.shelved, k.far} {
		w.issues.others[i.ID] = i
	}
	return k
}

// A creation under a parent, web's work item 3, reads it under the lock
// after the labels and before the number (M4 design 3.5, 3.9): the work
// item is stored and answered under it, and its activity is the
// creation's alone, with no row of its parent (3.13).
func TestCreateIssueUnderAParent(t *testing.T) {
	uc, w := newCreate()
	k := withKin(w)
	got, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "Part", LabelIDs: []uuid.UUID{bug}, ParentID: &k.up.ID})
	if err != nil || w.issues.issue == nil || !reflect.DeepEqual(w.issues.issue.ParentID, &k.up.ID) || !reflect.DeepEqual(got.ParentID,
		&k.up.ID) {
		t.Fatalf("Execute() = %+v, %v, stored %+v; want the work item under %s", got, err, w.issues.issue, k.up.ID)
	}
	id := w.issues.issue.ID
	want := created(id, backlog.ID, nil, []uuid.UUID{bug}, nil, []uuid.UUID{bug}, []uuid.UUID{alice}, 1)
	want = slices.Insert(want, slices.Index(want, line("Next", web)), line("GetIssue", k.up.ID))
	if !slices.Equal(w.log.calls, want) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, want)
	}
	wantActivity(t, w, id, domain.CreationActivity(nil))
}

// A parent a creation may not have is refused at parent_id, after the
// other problems, in one 422, and no number is taken (M4 design 3.4, 3.9):
// a work item not there, another project's, an archived one, and another's
// for a caller who sees only his own work items by the decision under the
// lock (Grant.OwnOnly). By the precheck's alone he may, and under his own
// work item too.
func TestCreateIssueRefusesAParentItMayNotHave(t *testing.T) {
	decided := []string{line("Find", web, "outside tx"), line("Authorize", alice, domain.ActionCreate, acme, web, "outside tx"), "Begin",
		line("Lock", web), line("Authorize", alice, domain.ActionCreate, acme, web), line("States", web)}
	stateNotAllowed := shared.FieldError{Field: "state_id", Code: "not_allowed", Message: "must be a state of the project that is not its triage state"}
	for _, tt := range []struct {
		name   string
		grants []shared.Grant
		parent func(k kin) uuid.UUID
	}{
		{"not there", nil, func(kin) uuid.UUID { return uuid.NewV7() }},
		{"another project's", nil, func(k kin) uuid.UUID { return k.far.ID }},
		{"archived", nil, func(k kin) uuid.UUID { return k.shelved.ID }},
		{"another's, for one who sees his own alone", []shared.Grant{{}, {OwnOnly: true}}, func(k kin) uuid.UUID { return k.up.ID }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w := newCreate()
			k := withKin(w)
			w.auth.grants = tt.grants
			parent := tt.parent(k)
			_, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "a", StateID: &triage.ID, ParentID: &parent})
			refused(t, w, err, shared.Invalid(stateNotAllowed, domain.ParentNotAllowed()), append(slices.Clone(decided), line("GetIssue", parent)))
		})
	}
	for _, tt := range []struct {
		name   string
		grants []shared.Grant
		parent func(k kin) uuid.UUID
	}{
		{"another's, for one who sees his own alone by the precheck's decision", []shared.Grant{{OwnOnly: true}, {}},
			func(k kin) uuid.UUID { return k.up.ID }},
		{"his own, for one who sees his own alone", []shared.Grant{{OwnOnly: true}, {OwnOnly: true}}, func(k kin) uuid.UUID { return k.mine.ID }},
	} {
		uc, w := newCreate()
		k := withKin(w)
		w.auth.grants = tt.grants
		parent := tt.parent(k)
		if got, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "a", ParentID: &parent}); err != nil || !reflect.DeepEqual(got.ParentID,
			&parent) {
			t.Errorf("%s: Execute() = %+v, %v; want the work item under %s", tt.name, got, err, parent)
		}
	}
}

// An update that moves web's work item 7 from under up to under top, puts
// it under top from under none, or takes it from under up (M4 design 3.5,
// 3.9, 3.13), top being under mine, which is neither the work item nor
// under it: after the dates, the new parent read under the lock, then its
// ancestry, then the old parent; the clock; the work item's fields, its
// parent the new one, the others as they were; its activity a row of its
// parent alone, by the old and the new one's identifiers and ids, WEB-3
// and WEB-12, "" and none for none, alice's at the moment. The answer is
// the work item under its new parent, its sub-items counted.
func TestUpdateIssueMovesTheWorkItem(t *testing.T) {
	ref := func(i domain.Issue, identifier string) *domain.Ref {
		return &domain.Ref{ID: i.ID, Identifier: identifier}
	}
	for _, tt := range []struct {
		name     string
		from, to func(k kin) *uuid.UUID
		reads    func(k kin) []string
		change   func(k kin) domain.Changed[*domain.Ref]
	}{
		{"from up to top", func(k kin) *uuid.UUID { return &k.up.ID }, func(k kin) *uuid.UUID { return &k.top.ID },
			func(k kin) []string {
				return []string{line("GetIssue", k.top.ID), line("Ancestry", k.top.ID), line("GetIssues", []uuid.UUID{k.up.ID})}
			},
			func(k kin) domain.Changed[*domain.Ref] {
				return domain.Changed[*domain.Ref]{Old: ref(k.up, "WEB-3"), New: ref(k.top, "WEB-12")}
			}},
		{"from none to top", func(kin) *uuid.UUID { return nil }, func(k kin) *uuid.UUID { return &k.top.ID },
			func(k kin) []string { return []string{line("GetIssue", k.top.ID), line("Ancestry", k.top.ID)} },
			func(k kin) domain.Changed[*domain.Ref] { return domain.Changed[*domain.Ref]{New: ref(k.top, "WEB-12")} }},
		{"from up to none", func(k kin) *uuid.UUID { return &k.up.ID }, func(kin) *uuid.UUID { return nil },
			func(k kin) []string { return []string{line("GetIssues", []uuid.UUID{k.up.ID})} },
			func(k kin) domain.Changed[*domain.Ref] { return domain.Changed[*domain.Ref]{Old: ref(k.up, "WEB-3")} }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newUpdate()
			k := withKin(w)
			w.issues.issue.ParentID = tt.from(k)
			w.issues.ancestry = map[uuid.UUID][]uuid.UUID{k.top.ID: {k.top.ID, k.mine.ID}}
			got, err := uc.Execute(as(alice), id, domain.IssuePatch{SetParentID: true, ParentID: tt.to(k)})
			if err != nil || !reflect.DeepEqual(got.ParentID, tt.to(k)) {
				t.Fatalf("Execute() = %+v, %v; want the work item under %v", got, err, tt.to(k))
			}
			calls := slices.Concat(locked(id), tt.reads(k), []string{"Now", line("UpdateIssue", id), line("InsertActivities", 1),
				line("GetIssue", id), counted(id)})
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			if fields := (app.IssueFields{SortOrder: 65535, ID: id, Name: "Fix", StateID: backlog.ID, Priority: domain.PriorityHigh, ParentID: tt.to(k), By: alice,
				At: clockNow}); w.issues.changed == nil || !reflect.DeepEqual(*w.issues.changed, fields) {
				t.Errorf("the fields written %+v, want %+v", w.issues.changed, fields)
			}
			change := tt.change(k)
			wantActivity(t, w, id, []domain.Activity{domain.ParentActivity(change)})
		})
	}
}

// A parent an update may not set is refused at parent_id, after the other
// problems, in one 422, and nothing is written (M4 design 3.4, 3.9): a
// work item not there, another project's, an archived one, read under the
// lock; the work item itself and one under it, whose ancestries hold it;
// and another's for a caller who sees only his own work items by the
// decision under the lock, as alice, web's admin, may in a project where
// she is its guest, who updates her own work item. Under her own work
// item she may put it. A parent given as the work item has it changes
// nothing and reads none.
func TestUpdateIssueRefusesAParentItMayNotHave(t *testing.T) {
	outOfOrder := shared.FieldError{Field: "target_date", Code: "not_allowed", Message: "must not be before the start date"}
	read := func(parent uuid.UUID) []string { return []string{line("GetIssue", parent)} }
	climbed := func(parent uuid.UUID) []string { return []string{line("GetIssue", parent), line("Ancestry", parent)} }
	ownOnly := func(w *world) {
		w.issues.issue.By, w.auth.grants = alice, []shared.Grant{{OwnOnly: true}, {OwnOnly: true}, {OwnOnly: true}, {OwnOnly: true}}
	}
	for _, tt := range []struct {
		name  string
		set   func(w *world, k kin, id uuid.UUID) uuid.UUID // the parent asked for
		reads func(parent uuid.UUID) []string
	}{
		{"not there", func(*world, kin, uuid.UUID) uuid.UUID { return uuid.NewV7() }, read},
		{"another project's", func(_ *world, k kin, _ uuid.UUID) uuid.UUID { return k.far.ID }, read},
		{"archived", func(_ *world, k kin, _ uuid.UUID) uuid.UUID { return k.shelved.ID }, read},
		{"itself", func(_ *world, _ kin, id uuid.UUID) uuid.UUID { return id }, climbed},
		{"one under it", func(w *world, k kin, id uuid.UUID) uuid.UUID {
			w.issues.ancestry = map[uuid.UUID][]uuid.UUID{k.top.ID: {k.top.ID, id, k.up.ID}}
			return k.top.ID
		}, climbed},
		{"another's, for one who sees her own alone", func(w *world, k kin, _ uuid.UUID) uuid.UUID { ownOnly(w); return k.up.ID }, read},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newUpdate()
			k := withKin(w)
			parent := tt.set(w, k, id)
			_, err := uc.Execute(as(alice), id, domain.IssuePatch{SetStartDate: true, StartDate: october(20), SetTargetDate: true,
				TargetDate: october(19), SetParentID: true, ParentID: &parent})
			unchanged(t, w, err, shared.Invalid(outOfOrder, domain.ParentNotAllowed()), append(locked(id), tt.reads(parent)...))
		})
	}
	uc, w, id := newUpdate()
	k := withKin(w)
	ownOnly(w)
	if got, err := uc.Execute(as(alice), id, domain.IssuePatch{SetParentID: true, ParentID: &k.mine.ID}); err != nil ||
		!reflect.DeepEqual(got.ParentID, &k.mine.ID) {
		t.Errorf("Execute() under her own work item = %+v, %v; want it under %s", got, err, k.mine.ID)
	}
	uc, w, id = newUpdate()
	k = withKin(w)
	w.issues.issue.ParentID = &k.up.ID
	again := k.up.ID
	if got, err := uc.Execute(as(alice), id, domain.IssuePatch{SetParentID: true, ParentID: &again}); err != nil ||
		!reflect.DeepEqual(got.ParentID, &k.up.ID) || !slices.Equal(w.log.calls, append(locked(id), counted(id))) || w.issues.changed != nil {
		t.Errorf("Execute() of the parent it has = %+v, %v after\n%q; want it as it was after\n%q, nothing written", got, err, w.log.calls,
			append(locked(id), counted(id)))
	}
}

// An update compares the parent it sets with the parent its caller is
// answered (M4 design 3.4, 3.9; the owner's ruling A, 2026-10-24): alice,
// a workspace admin who is only a guest of web (adminGuest), sees only her
// own work items there, and her work item 7 is under up, bob's, which she
// does not see, so she is answered it under none. Naming up is a change to
// a work item she does not see, refused as naming one not there is, 422 at
// parent_id, the two refusals equal, nothing written. None, as she was
// answered, changes nothing: up is read under the lock, nothing written,
// not even the clock read, and she is answered the work item as it is,
// under none. Under mine, her own, it moves from up, as stored: its row
// names up, WEB-3, and mine, WEB-10, up read once.
func TestUpdateIssueComparesTheParentAsItsCallerIsAnswered(t *testing.T) {
	underUp := func() (*app.UpdateIssue, *world, uuid.UUID, kin) {
		uc, w, id := newUpdate()
		k := withKin(w)
		w.issues.issue.By, w.issues.issue.ParentID, w.auth.grants = alice, &k.up.ID, []shared.Grant{adminGuest, adminGuest}
		return uc, w, id, k
	}
	// up read under the lock, the parent before, and the answer's read of
	// its parent (Locks.seen)
	before := func(k kin) string { return line("GetIssues", []uuid.UUID{k.up.ID}) }
	shown := func(parent uuid.UUID) string { return line("GetIssues", []uuid.UUID{parent}) }
	var refusals []error
	for _, named := range []func(k kin) uuid.UUID{func(k kin) uuid.UUID { return k.up.ID }, func(kin) uuid.UUID { return uuid.NewV7() }} {
		uc, w, id, k := underUp()
		parent := named(k)
		_, err := uc.Execute(as(alice), id, domain.IssuePatch{SetParentID: true, ParentID: &parent})
		unchanged(t, w, err, shared.Invalid(domain.ParentNotAllowed()), append(locked(id), before(k), line("GetIssue", parent)))
		refusals = append(refusals, err)
	}
	if !reflect.DeepEqual(refusals[0], refusals[1]) {
		t.Errorf("up named: %v; want the refusal of a work item not there, %v", refusals[0], refusals[1])
	}

	uc, w, id, k := underUp()
	got, err := uc.Execute(as(alice), id, domain.IssuePatch{SetParentID: true})
	if calls := append(locked(id), before(k), counted(id), shown(k.up.ID)); err != nil || got.ParentID != nil || w.issues.changed != nil ||
		!slices.Equal(w.log.calls, calls) {
		t.Errorf("none: Execute() = %v, %v after\n%q; want under none, nothing written, after\n%q", got.ParentID, err, w.log.calls, calls)
	}

	uc, w, id, k = underUp()
	got, err = uc.Execute(as(alice), id, domain.IssuePatch{SetParentID: true, ParentID: &k.mine.ID})
	calls := slices.Concat(locked(id), []string{before(k), line("GetIssue", k.mine.ID), line("Ancestry", k.mine.ID), "Now", line("UpdateIssue", id),
		line("InsertActivities", 1), line("GetIssue", id), counted(id), shown(k.mine.ID)})
	if err != nil || !reflect.DeepEqual(got.ParentID, &k.mine.ID) || !slices.Equal(w.log.calls, calls) {
		t.Errorf("under mine: Execute() = %v, %v after\n%q; want under mine after\n%q", got.ParentID, err, w.log.calls, calls)
	}
	wantActivity(t, w, id, []domain.Activity{domain.ParentActivity(domain.Changed[*domain.Ref]{Old: &domain.Ref{ID: k.up.ID, Identifier: "WEB-3"},
		New: &domain.Ref{ID: k.mine.ID, Identifier: "WEB-10"}})})
}

// A failing read of a parent ends the write with its failure, as itself,
// and no call after it runs (M4 design 3.5): the parent a creation asks
// for; the new parent an update asks for, its ancestry, the old one, and
// the old one read, once, to compare with the parent its caller is
// answered, as for a workspace admin who is only a guest of web
// (adminGuest). An old parent not there is an error too, before the clock:
// deleting a work item deletes those under it (3.6).
func TestAParentsFailingReadEndsTheWrite(t *testing.T) {
	create, w := newCreate()
	k := withKin(w)
	w.log.fail = line("GetIssue", k.up.ID)
	_, err := create.Execute(as(alice), web, domain.NewIssue{Name: "a", ParentID: &k.up.ID})
	wantStoppedOn(t, w, err, w.log.fail)
	for _, at := range []string{"new", "ancestry", "old"} {
		t.Run(at, func(t *testing.T) {
			uc, w, id := newUpdate()
			k := withKin(w)
			w.issues.issue.ParentID = &k.up.ID
			w.log.fail = map[string]string{"new": line("GetIssue", k.top.ID), "ancestry": line("Ancestry", k.top.ID),
				"old": line("GetIssues", []uuid.UUID{k.up.ID})}[at]
			_, err := uc.Execute(as(alice), id, domain.IssuePatch{SetParentID: true, ParentID: &k.top.ID})
			wantStoppedOn(t, w, err, w.log.fail)
		})
	}
	uc, w, id := newUpdate()
	k = withKin(w)
	w.issues.issue.By, w.issues.issue.ParentID, w.auth.grants = alice, &k.up.ID, []shared.Grant{adminGuest, adminGuest}
	w.log.fail = line("GetIssues", []uuid.UUID{k.up.ID})
	_, err = uc.Execute(as(alice), id, domain.IssuePatch{SetParentID: true})
	if calls := append(locked(id), w.log.fail); !errors.Is(err, errDisk) || !slices.Equal(w.log.calls, calls) {
		t.Errorf("Execute() = %v after the calls\n%q\nwant %v after\n%q", err, w.log.calls, errDisk, calls)
	}
	uc, w, id = newUpdate()
	k = withKin(w)
	gone := uuid.NewV7()
	w.issues.issue.ParentID = &gone
	var se *shared.Error
	if _, err := uc.Execute(as(alice), id, domain.IssuePatch{SetParentID: true, ParentID: &k.top.ID}); err == nil || errors.As(err, &se) ||
		slices.Contains(w.log.calls, "Now") {
		t.Errorf("Execute() of a work item under one not there = %v after %q; want an error before the clock", err, w.log.calls)
	}
}
