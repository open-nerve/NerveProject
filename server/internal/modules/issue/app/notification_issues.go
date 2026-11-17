package app

import "uuid"

// NotificationIssue is a work item as the module tells the notification
// module about it, its own columns (M7 P1b spec 2.4, 2.5, 2.6): its name,
// its number, its state by id, its creator, nil when not known, and the
// accounts of its undeleted assignees' rows, each once, in id order. The
// state's name and group and the project's identifier are the project
// module's, which bootstrap completes it with; the issue module's
// statements do not read them.
type NotificationIssue struct {
	Name        string
	SequenceID  int
	StateID     uuid.UUID
	CreatedByID *uuid.UUID
	AssigneeIDs []uuid.UUID
}
