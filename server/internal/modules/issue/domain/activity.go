package domain

import "uuid"

// Verb is what a row of a work item's activity says was done: Plane's
// values, which the page's renderer reads (M4 design 3.13).
type Verb string

// The verbs, the values of issue_activities.verb.
const (
	VerbCreated Verb = "created"
	VerbUpdated Verb = "updated"
	VerbDeleted Verb = "deleted"
)

// Activity is a row of a work item's activity as M4 design 3.13's table
// gives it: the field it is on, its verb, the old and new values and the
// ids they refer to, and Plane's English sentence without its brand, which
// the API does not show (3.13). A nil field, value or id is NULL.
type Activity struct {
	Field         *string
	Verb          Verb
	OldValue      *string
	NewValue      *string
	OldIdentifier *uuid.UUID
	NewIdentifier *uuid.UUID
	Comment       string
}

// Assignee is an account a work item is assigned to, as its activity names
// him: his id and his display name when the row is written.
type Assignee struct {
	ID          uuid.UUID
	DisplayName string
}

// CreationActivity is the rows a work item's creation writes, in order (M4
// design 3.13): row 1, the creation, then row 2 for each of assignees in
// the order given, the accounts it is assigned to, the default assignee
// too. The labels it is created with write none (Plane the same).
func CreationActivity(assignees []Assignee) []Activity {
	rows := []Activity{{Verb: VerbCreated, Comment: "created the work item"}}
	for _, a := range assignees {
		rows = append(rows, Activity{Field: ptr("assignees"), Verb: VerbUpdated, OldValue: ptr(""), NewValue: ptr(a.DisplayName),
			NewIdentifier: ptr(a.ID), Comment: "added assignee "})
	}
	return rows
}

func ptr[T any](v T) *T {
	return &v
}
