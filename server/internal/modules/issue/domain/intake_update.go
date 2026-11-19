package domain

import (
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IntakeIssueUpdate is what the caller asks to change in a work item of a
// project's intake (M7 P2b spec 2.4): its name, description, priority,
// assignees, labels and target date, each as IssuePatch has it, and
// skip_activity, by updateIssue's rule. Nothing else: its state, start
// date, parent, cycle and modules stay as a submission leaves them until
// a triage accepts it.
type IntakeIssueUpdate struct {
	DescriptionHTML *string
	SkipActivity    *bool
	Name            *string
	Priority        *Priority
	AssigneeIDs     *[]uuid.UUID
	LabelIDs        *[]uuid.UUID
	SetTargetDate   bool
	TargetDate      *shared.Date
}

// CheckIntakeIssueUpdate checks u as CheckIssuePatch checks the patch of
// the same fields, and answers that patch as it is stored.
func CheckIntakeIssueUpdate(u IntakeIssueUpdate) (IssuePatch, error) {
	return CheckIssuePatch(IssuePatch{DescriptionHTML: u.DescriptionHTML, SkipActivity: u.SkipActivity, Name: u.Name, Priority: u.Priority,
		AssigneeIDs: u.AssigneeIDs, LabelIDs: u.LabelIDs, SetTargetDate: u.SetTargetDate, TargetDate: u.TargetDate})
}

// ContentOnly reports whether u gives no field but the name, the
// description and skip_activity: all a guest who submitted the work item
// may give (spec 2.4, R4). A field given, even as null or as its value
// now, is given.
func (u IntakeIssueUpdate) ContentOnly() bool {
	return u.Priority == nil && u.AssigneeIDs == nil && u.LabelIDs == nil && !u.SetTargetDate
}
