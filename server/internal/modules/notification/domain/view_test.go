package domain

import (
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The fields whose ids name another work item (spec 2.6) are parent and
// the four relation types, and every relation the fan-out writes is of one
// of them, on either verb.
func TestWorkItemFields(t *testing.T) {
	got := WorkItemFields()
	slices.Sort(got)
	if want := []string{"blocked_by", "blocking", "duplicate", "parent", "relates_to"}; !slices.Equal(got, want) {
		t.Errorf("WorkItemFields() = %v, want %v", got, want)
	}
	for _, e := range []shared.Event{blockingAdded(), relatesAdded(), blockedByRemoved()} {
		for _, item := range NotifiableOf(e).Changes {
			if !slices.Contains(WorkItemFields(), item.Field) {
				t.Errorf("%s notifies of %s, not a work item field", e.Type, showItem(item))
			}
		}
	}
}

// RefIDs are the work items a page's changes name (spec 2.6): the old and
// new ids of parent and relation changes, each once, in id order; not an
// account's, a label's, a state's, a cycle's or a comment's, nor those of a
// field the module does not know.
func TestRefIDs(t *testing.T) {
	issue := IssueSnapshot{ID: workItem, Name: "Fix it", SequenceID: 12, Identifier: "PROJ"}
	data := func(c ChangeSnapshot) Data { return Data{Issue: issue, Change: c} }
	page := []Data{
		data(ChangeSnapshot{Field: "parent", Verb: VerbUpdated, OldID: ref(parent), NewID: ref(newParent)}),
		data(ChangeSnapshot{Field: "blocking", Verb: VerbCreated, NewID: ref(other)}),
		data(ChangeSnapshot{Field: "state", Verb: VerbUpdated, OldID: ref(todo), NewID: ref(doing)}),
		data(ChangeSnapshot{Field: "labels", Verb: VerbUpdated, NewID: ref(label)}),
		data(ChangeSnapshot{Field: "assignees", Verb: VerbUpdated, OldID: ref(dave)}),
		data(ChangeSnapshot{Field: "cycles", Verb: VerbUpdated, NewID: ref(cycle)}),
		data(ChangeSnapshot{Field: "subtasks", Verb: VerbCreated, NewID: ref(oldLabel)}),
		data(ChangeSnapshot{Field: "comment", Verb: VerbCreated, CommentID: ref(comment)}),
		data(ChangeSnapshot{Field: "parent", Verb: VerbUpdated, OldID: ref(parent)}),
		data(ChangeSnapshot{Field: "blocked_by", Verb: VerbDeleted, OldID: ref(other)}),
		data(ChangeSnapshot{Field: "duplicate", Verb: VerbCreated, NewID: ref(eve)}),
		data(ChangeSnapshot{Field: "relates_to", Verb: VerbDeleted, OldID: ref(carol)}),
	}
	want := []uuid.UUID{parent, newParent, other, eve, carol}
	slices.SortFunc(want, uuid.UUID.Compare)
	if got := RefIDs(page); !slices.Equal(got, want) {
		t.Errorf("RefIDs() = %v, want %v", got, want)
	}
	if got := RefIDs(nil); len(got) != 0 {
		t.Errorf("RefIDs(nil) = %v, want none", got)
	}
}

// A reader's view of a notification (spec 2.4, R7) keeps its work item
// and its change as stored but the ids: those of an account, a label and a
// state as stored; those of parent and the relations each only when the
// reader reads the work item it names now, then with it, else nil with it;
// any other field's never, even when it names a work item he reads.
func TestViewOf(t *testing.T) {
	issue := IssueSnapshot{ID: workItem, Name: "Fix it", SequenceID: 12, Identifier: "PROJ", StateName: "Doing", StateGroup: "started"}
	parentRef := IssueRef{ID: parent, SequenceID: 1, Identifier: "PROJ", Name: "The parent"}
	otherRef := IssueRef{ID: other, SequenceID: 7, Identifier: "OPS", Name: "Another project's"}
	refs := map[uuid.UUID]IssueRef{parent: parentRef, other: otherRef}
	tests := []struct {
		name   string
		change ChangeSnapshot
		want   ChangeView
	}{
		{"state", ChangeSnapshot{Field: "state", Verb: VerbUpdated, OldValue: text("Todo"), NewValue: text("Doing"), OldID: ref(todo), NewID: ref(doing)},
			ChangeView{Field: "state", Verb: VerbUpdated, OldValue: text("Todo"), NewValue: text("Doing"), OldID: ref(todo), NewID: ref(doing)}},
		{"label added", ChangeSnapshot{Field: "labels", Verb: VerbUpdated, OldValue: text(""), NewValue: text("bug"), NewID: ref(label)},
			ChangeView{Field: "labels", Verb: VerbUpdated, OldValue: text(""), NewValue: text("bug"), NewID: ref(label)}},
		{"label removed", ChangeSnapshot{Field: "labels", Verb: VerbUpdated, OldValue: text("ui"), NewValue: text(""), OldID: ref(oldLabel)},
			ChangeView{Field: "labels", Verb: VerbUpdated, OldValue: text("ui"), NewValue: text(""), OldID: ref(oldLabel)}},
		{"assignee", ChangeSnapshot{Field: "assignees", Verb: VerbCreated, NewValue: text("Dave"), NewID: ref(dave)},
			ChangeView{Field: "assignees", Verb: VerbCreated, NewValue: text("Dave"), NewID: ref(dave)}},
		{"parent from one he reads to one he does not", ChangeSnapshot{Field: "parent", Verb: VerbUpdated, OldID: ref(parent), NewID: ref(newParent)},
			ChangeView{Field: "parent", Verb: VerbUpdated, OldID: ref(parent), OldIssue: &parentRef}},
		{"parent from one he does not read to one he does", ChangeSnapshot{Field: "parent", Verb: VerbUpdated, OldID: ref(newParent), NewID: ref(parent)},
			ChangeView{Field: "parent", Verb: VerbUpdated, NewID: ref(parent), NewIssue: &parentRef}},
		{"parent he reads both", ChangeSnapshot{Field: "parent", Verb: VerbUpdated, OldID: ref(other), NewID: ref(parent)},
			ChangeView{Field: "parent", Verb: VerbUpdated, OldID: ref(other), OldIssue: &otherRef, NewID: ref(parent), NewIssue: &parentRef}},
		{"parent removed", ChangeSnapshot{Field: "parent", Verb: VerbUpdated, OldID: ref(parent)},
			ChangeView{Field: "parent", Verb: VerbUpdated, OldID: ref(parent), OldIssue: &parentRef}},
		{"parent he does not read", ChangeSnapshot{Field: "parent", Verb: VerbUpdated, NewID: ref(newParent)},
			ChangeView{Field: "parent", Verb: VerbUpdated}},
		{"blocking", ChangeSnapshot{Field: "blocking", Verb: VerbCreated, NewID: ref(other)},
			ChangeView{Field: "blocking", Verb: VerbCreated, NewID: ref(other), NewIssue: &otherRef}},
		{"blocked_by removed", ChangeSnapshot{Field: "blocked_by", Verb: VerbDeleted, OldID: ref(other)},
			ChangeView{Field: "blocked_by", Verb: VerbDeleted, OldID: ref(other), OldIssue: &otherRef}},
		{"duplicate", ChangeSnapshot{Field: "duplicate", Verb: VerbCreated, NewID: ref(parent)},
			ChangeView{Field: "duplicate", Verb: VerbCreated, NewID: ref(parent), NewIssue: &parentRef}},
		{"relates_to he does not read", ChangeSnapshot{Field: "relates_to", Verb: VerbCreated, NewID: ref(newParent)},
			ChangeView{Field: "relates_to", Verb: VerbCreated}},
		{"cycle", ChangeSnapshot{Field: "cycles", Verb: VerbUpdated, NewValue: text("Sprint 1"), NewID: ref(cycle)},
			ChangeView{Field: "cycles", Verb: VerbUpdated, NewValue: text("Sprint 1")}},
		{"a field the module does not know", ChangeSnapshot{Field: "subtasks", Verb: VerbCreated, OldID: ref(parent), NewID: ref(other)},
			ChangeView{Field: "subtasks", Verb: VerbCreated}},
		{"comment", ChangeSnapshot{Field: "comment", Verb: VerbMentioned, CommentID: ref(comment), Excerpt: text("look")},
			ChangeView{Field: "comment", Verb: VerbMentioned, CommentID: ref(comment), Excerpt: text("look")}},
		{"name", ChangeSnapshot{Field: "name", Verb: VerbUpdated, OldValue: text("Fix it"), NewValue: text("Fix it now")},
			ChangeView{Field: "name", Verb: VerbUpdated, OldValue: text("Fix it"), NewValue: text("Fix it now")}},
		{"creation", ChangeSnapshot{Field: "issue", Verb: VerbCreated}, ChangeView{Field: "issue", Verb: VerbCreated}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ViewOf(Data{Issue: issue, Change: tt.change}, refs)
			if want := (View{Issue: issue, Change: tt.want}); !reflect.DeepEqual(got, want) {
				t.Errorf("ViewOf() = %+v\nwant %+v", got.Change, want.Change)
			}
		})
	}
}

// Every id a notification of each event keeps is in the view of a reader
// who reads every work item: the fields with ids the fan-out writes are
// all answered or resolved, none dropped.
func TestViewOfKeepsTheIDsTheFanOutWrites(t *testing.T) {
	issue := IssueSnapshot{ID: workItem, Name: "Fix it", SequenceID: 12, Identifier: "PROJ"}
	for name, e := range events() {
		for _, item := range NotifiableOf(e).Changes {
			refs := map[uuid.UUID]IssueRef{}
			for _, id := range []*uuid.UUID{item.OldID, item.NewID} {
				if id != nil {
					refs[*id] = IssueRef{ID: *id, SequenceID: 3, Identifier: "PROJ", Name: "Any"}
				}
			}
			got := ViewOf(NewData(issue, item, nil), refs).Change
			if !reflect.DeepEqual(got.OldID, item.OldID) || !reflect.DeepEqual(got.NewID, item.NewID) {
				t.Errorf("%s: the view of %s has ids %s %s", name, showItem(item), named(got.OldID), named(got.NewID))
			}
		}
	}
}
