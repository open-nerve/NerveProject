package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The issue module's actions: the keys of its rows in the access module's
// rule table (M4 design 3.4).
const (
	// ActionCreateComment creates a comment on a visible work item.
	ActionCreateComment shared.Action = "issue_comment.create"
	// ActionWriteComment gates edits and deletions before relative author checks.
	ActionWriteComment shared.Action = "issue_comment.write"
	// ActionCreate is creating a work item in a project.
	ActionCreate shared.Action = "issue.create"
	// ActionRead is reading a work item of a project.
	ActionRead shared.Action = "issue.read"
	// ActionUpdate is changing a work item of a project.
	ActionUpdate shared.Action = "issue.update"
	// ActionDelete is deleting a work item of a project.
	ActionDelete shared.Action = "issue.delete"
	// ActionArchive is archiving and unarchiving a work item of a project.
	ActionArchive shared.Action = "issue.archive"
	// ActionSearch is looking for a work item in a workspace: finding one
	// by its identifier, which then reads it as ActionRead.
	ActionSearch shared.Action = "issue.search"
)

// Actions lists the module's actions. bootstrap's test holds the union of
// every module's Actions equal to the rule table's keys (M3 design 3.4).
func Actions() []shared.Action {
	return []shared.Action{ActionCreate, ActionRead, ActionUpdate, ActionDelete, ActionArchive, ActionSearch, ActionCreateComment, ActionWriteComment}
}
