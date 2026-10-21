package app_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

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

// A failing read of a parent ends the write with its failure, as itself,
// and no call after it runs (M4 design 3.5): the parent a creation asks
// for.
func TestAParentsFailingReadEndsTheWrite(t *testing.T) {
	stopped := func(t *testing.T, w *world, err error, call string) {
		t.Helper()
		if !errors.Is(err, errDisk) || len(w.log.calls) == 0 || w.log.calls[len(w.log.calls)-1] != call {
			t.Errorf("Execute() = %v after the calls\n%q\nwant %v, %s the last call", err, w.log.calls, errDisk, call)
		}
	}
	create, w := newCreate()
	k := withKin(w)
	w.log.fail = line("GetIssue", k.up.ID)
	_, err := create.Execute(as(alice), web, domain.NewIssue{Name: "a", ParentID: &k.up.ID})
	stopped(t, w, err, w.log.fail)
}
