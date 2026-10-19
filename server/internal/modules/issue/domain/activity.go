package domain

import "uuid"

// Verb is what a row of a work item's activity says was done: Plane's
// values, which the page's renderer reads (M4 design 3.13).
type Verb string

// The verbs the rows are written with, values of issue_activities.verb.
const (
	VerbCreated Verb = "created"
	VerbUpdated Verb = "updated"
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

// Assignee is an account a work item is assigned to, or unassigned from,
// as its activity names him: his id and his display name when the row is
// written.
type Assignee struct {
	ID          uuid.UUID
	DisplayName string
}

// assigneeAdded is the row of a, an account a work item is assigned to:
// row 2 at its creation and row 12 at an update, which M4 design 3.13
// gives the same columns: assignees, updated, from "" to his display name,
// his id the new identifier.
func assigneeAdded(a Assignee) Activity {
	return Activity{Field: ptr("assignees"), Verb: VerbUpdated, OldValue: ptr(""), NewValue: ptr(a.DisplayName),
		NewIdentifier: ptr(a.ID), Comment: "added assignee "}
}

// CreationActivity is the rows a work item's creation writes, in order (M4
// design 3.13): row 1, the creation, then row 2 for each of assignees in
// the order given, the accounts it is assigned to, the default assignee
// too. The labels it is created with write none (Plane the same).
func CreationActivity(assignees []Assignee) []Activity {
	rows := []Activity{{Verb: VerbCreated, Comment: "created the work item"}}
	for _, a := range assignees {
		rows = append(rows, assigneeAdded(a))
	}
	return rows
}

// Named is a row an activity names by its name, a state or a label: its
// id, and its name when the activity is written.
type Named struct {
	ID   uuid.UUID
	Name string
}

// Changed is a value an update changes: the old one and the new one.
type Changed[T any] struct {
	Old, New T
}

// Update is what an update of a work item changes, as its activity names
// it (M4 design 3.13): its name, its priority and its state, each nil when
// it is left as it is; the labels and the accounts it adds and removes,
// each in id order.
type Update struct {
	Name             *Changed[string]
	Priority         *Changed[Priority]
	State            *Changed[Named]
	LabelsAdded      []Named
	LabelsRemoved    []Named
	AssigneesAdded   []Assignee
	AssigneesRemoved []Assignee
}

// UpdateActivity is the rows an update writes, in the order of M4 design
// 3.13's table: row 3, the name; row 6, the priority; row 7, the state, by
// its names and its ids; then each label added (row 10) and removed (row
// 11), by its name and its id; then each account assigned (row 12) and
// unassigned (row 13), by his display name and his id; each list in the
// order given. A field left as it is writes no row (Plane the same).
func UpdateActivity(u Update) []Activity {
	var rows []Activity
	if u.Name != nil {
		rows = append(rows, Activity{Field: ptr("name"), Verb: VerbUpdated, OldValue: ptr(u.Name.Old), NewValue: ptr(u.Name.New),
			Comment: "updated the name to"})
	}
	if u.Priority != nil {
		rows = append(rows, Activity{Field: ptr("priority"), Verb: VerbUpdated, OldValue: ptr(string(u.Priority.Old)),
			NewValue: ptr(string(u.Priority.New)), Comment: "updated the priority to"})
	}
	if u.State != nil {
		rows = append(rows, Activity{Field: ptr("state"), Verb: VerbUpdated, OldValue: ptr(u.State.Old.Name), NewValue: ptr(u.State.New.Name),
			OldIdentifier: ptr(u.State.Old.ID), NewIdentifier: ptr(u.State.New.ID), Comment: "updated the state to"})
	}
	for _, l := range u.LabelsAdded {
		rows = append(rows, Activity{Field: ptr("labels"), Verb: VerbUpdated, OldValue: ptr(""), NewValue: ptr(l.Name),
			NewIdentifier: ptr(l.ID), Comment: "added label "})
	}
	for _, l := range u.LabelsRemoved {
		rows = append(rows, Activity{Field: ptr("labels"), Verb: VerbUpdated, OldValue: ptr(l.Name), NewValue: ptr(""),
			OldIdentifier: ptr(l.ID), Comment: "removed label "})
	}
	for _, a := range u.AssigneesAdded {
		rows = append(rows, assigneeAdded(a))
	}
	for _, a := range u.AssigneesRemoved {
		rows = append(rows, Activity{Field: ptr("assignees"), Verb: VerbUpdated, OldValue: ptr(a.DisplayName), NewValue: ptr(""),
			OldIdentifier: ptr(a.ID), Comment: "removed assignee "})
	}
	return rows
}

func ptr[T any](v T) *T {
	return &v
}
