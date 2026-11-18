package domain

import (
	"slices"
	"uuid"
)

// workItemFields are the fields of the changes whose ids name another work
// item (P1c spec 2.6): parent, and the four relation types relationItem
// writes as a relation's field, the type its end reads.
var workItemFields = []string{"parent", "relates_to", "duplicate", "blocking", "blocked_by"}

// storedIDFields are the fields of the changes whose ids a reader is
// answered as stored (R7): an account's, a label's, a state's.
var storedIDFields = []string{"assignees", "labels", "state"}

// WorkItemFields are the fields of the changes whose ids name another work
// item (spec 2.6), which a reader's view resolves: parent and the four
// relation types. The bootstrap holds the issue module's every relation
// type to be one of them, so a new one is not answered as another field.
func WorkItemFields() []string { return slices.Clone(workItemFields) }

// IssueRef is a work item a change names as its reader reads it now (spec
// 2.4, 2.6): its id, its number, its project's identifier and its name.
type IssueRef struct {
	ID         uuid.UUID
	SequenceID int
	Identifier string
	Name       string
}

// View is a notification's data as its reader is answered (spec 2.4): its
// work item as stored, and its change's view.
type View struct {
	Issue  IssueSnapshot
	Change ChangeView
}

// ChangeView is a change as its reader is answered (spec 2.4, R7): as
// stored, but its ids. Those of storedIDFields are answered as stored;
// those of WorkItemFields each only when the reader reads the work item it
// names now, then with that work item as OldIssue or NewIssue, else nil
// with nil; those of any other field never.
type ChangeView struct {
	Field              string
	Verb               Verb
	OldValue, NewValue *string
	OldID, NewID       *uuid.UUID
	OldIssue, NewIssue *IssueRef
	CommentID          *uuid.UUID
	Excerpt            *string
}

// RefIDs are the work items the changes of page name, which its reader's
// views resolve (spec 2.6): the ids of WorkItemFields' changes, each once,
// in id order.
func RefIDs(page []Data) []uuid.UUID {
	var ids []uuid.UUID
	for _, d := range page {
		if !slices.Contains(workItemFields, d.Change.Field) {
			continue
		}
		for _, id := range []*uuid.UUID{d.Change.OldID, d.Change.NewID} {
			if id != nil {
				ids = append(ids, *id)
			}
		}
	}
	slices.SortFunc(ids, uuid.UUID.Compare)
	return slices.Compact(ids)
}

// ViewOf is d as its reader is answered (spec 2.4, R7), refs being the
// work items he reads now of those RefIDs names, by id.
func ViewOf(d Data, refs map[uuid.UUID]IssueRef) View {
	c := d.Change
	v := ChangeView{Field: c.Field, Verb: c.Verb, OldValue: c.OldValue, NewValue: c.NewValue, CommentID: c.CommentID, Excerpt: c.Excerpt}
	switch {
	case slices.Contains(storedIDFields, c.Field):
		v.OldID, v.NewID = c.OldID, c.NewID
	case slices.Contains(workItemFields, c.Field):
		v.OldID, v.OldIssue = resolved(c.OldID, refs)
		v.NewID, v.NewIssue = resolved(c.NewID, refs)
	}
	return View{Issue: d.Issue, Change: v}
}

// resolved is id, kept, and the work item it names as refs has it; nil and
// nil when id is nil or names one refs does not have.
func resolved(id *uuid.UUID, refs map[uuid.UUID]IssueRef) (*uuid.UUID, *IssueRef) {
	if id == nil {
		return nil, nil
	}
	ref, ok := refs[*id]
	if !ok {
		return nil, nil
	}
	kept := *id
	return &kept, &ref
}
