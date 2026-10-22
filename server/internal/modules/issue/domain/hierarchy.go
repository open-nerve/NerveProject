package domain

import (
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A work item's parent (M4 design 3.9): an undeleted work item of its
// project that it is under, however deep, or none. A work item is never
// under itself, nor under a work item under it, so the work items of a
// project make trees; an archived work item takes no new work item under
// it (3.8), and a guest who sees only his own work items puts none under
// another's (3.4).

// ParentChange is a parent a change sets: To, the new parent's id, nil
// when it takes the work item out from under its parent.
type ParentChange struct {
	To *uuid.UUID
}

// Loops reports whether putting the work item id under a work item whose
// ancestry, it and those above it, is ancestry would make a loop: id is
// in it (M4 design 3.9). A parent's update and an addition of sub-items
// hold their work items to it.
func Loops(id uuid.UUID, ancestry []uuid.UUID) bool {
	return slices.Contains(ancestry, id)
}

// ParentNotAllowed is the problem of a parent asked for that may not be
// the work item's (M4 design 3.9).
func ParentNotAllowed() shared.FieldError {
	return shared.FieldError{Field: "parent_id", Code: shared.FieldNotAllowed,
		Message: "must be a work item of the project that is not archived, and neither this one nor under it"}
}

// MaxSubIssues is how many work items an addition of sub-items names at
// most (M4 design 3.9, 5.1).
const MaxSubIssues = 100

// CheckSubIssueIDs checks the work items an addition of sub-items names,
// before any read (M4 design 3.9): at least one, at most MaxSubIssues,
// each named once; every problem in one 422 validation_failed. Whether
// each may go under the work item is the use case's, under the project's
// lock (SubIssueNotAllowed).
func CheckSubIssueIDs(ids []uuid.UUID) error {
	return invalid(checkSomeIDs("sub_issue_ids", ids, MaxSubIssues)...)
}

// SubIssueNotAllowed is the problem of the work item at index i of those
// an addition of sub-items names that may not go under the work item (M4
// design 3.9).
func SubIssueNotAllowed(i int) shared.FieldError {
	return shared.FieldError{Field: fmt.Sprintf("sub_issue_ids[%d]", i), Code: shared.FieldNotAllowed,
		Message: "must be a work item of the project that is not archived, and neither this one nor above it"}
}

// Ref is a work item as an activity names it (M4 design 3.13): its id, and
// its identifier when the row is written.
type Ref struct {
	ID         uuid.UUID
	Identifier string
}

// ParentActivity is the row of a work item put under another parent, or
// under none (row 5 of M4 design 3.13): parent, updated, from the old
// parent's identifier to the new one's, "" for none, their ids the old and
// the new identifier, nil for none. An update writes it (UpdateActivity),
// and so does an addition of sub-items for each one it moves.
func ParentActivity(c Changed[*Ref]) Activity {
	a := Activity{Field: ptr("parent"), Verb: VerbUpdated, OldValue: ptr(refText(c.Old)), NewValue: ptr(refText(c.New)),
		Comment: "updated the parent work item to"}
	if c.Old != nil {
		a.OldIdentifier = &c.Old.ID
	}
	if c.New != nil {
		a.NewIdentifier = &c.New.ID
	}
	return a
}

// refText is r as an activity's value: its identifier, "" for none.
func refText(r *Ref) string {
	if r == nil {
		return ""
	}
	return r.Identifier
}
