package domain

import (
	"slices"
	"strings"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IssuePatch is what the caller asks to change in a work item (M4 design
// 5.2, IssueUpdate without the fields of later phases): each field nil
// when it is not given, and keeps its value. AssigneeIDs and LabelIDs,
// given, are the work item's new sets of accounts and labels; given empty,
// none. A date may be cleared: SetStartDate and SetTargetDate tell whether
// the date is given, StartDate and TargetDate its day, nil for none.
type IssuePatch struct {
	Name          *string
	StateID       *uuid.UUID
	Priority      *Priority
	AssigneeIDs   *[]uuid.UUID
	LabelIDs      *[]uuid.UUID
	SetStartDate  bool
	StartDate     *shared.Date
	SetTargetDate bool
	TargetDate    *shared.Date
}

// CheckIssuePatch checks the fields p gives by createIssue's rules (M4
// design 3.7, CheckNewIssue) and returns p as it is stored: the name
// without its surrounding white space, of 1–255 characters and without
// NUL; the priority one of the five (given empty, it is refused, not left
// as it is); at most MaxAssignees accounts and MaxLabels labels, each
// named once. Every problem is reported at once, in one 422
// validation_failed; whether the state, each account and each label the
// change adds may be is the use case's, under the project's lock, and so
// is the order of the dates the change leaves (DatesOutOfOrder): they are
// the work item's, merged with those given.
func CheckIssuePatch(p IssuePatch) (IssuePatch, error) {
	var found []*shared.FieldError
	if p.Name != nil {
		name := strings.TrimSpace(*p.Name)
		p.Name = &name
		found = append(found, checkName(name))
	}
	if p.Priority != nil {
		found = append(found, checkPriority(*p.Priority))
	}
	if p.AssigneeIDs != nil {
		found = append(found, checkIDs("assignee_ids", *p.AssigneeIDs, MaxAssignees)...)
	}
	if p.LabelIDs != nil {
		found = append(found, checkIDs("label_ids", *p.LabelIDs, MaxLabels)...)
	}
	if err := invalid(found...); err != nil {
		return IssuePatch{}, err
	}
	return p, nil
}

// Diff is how a set of ids becomes another: the ids it adds and those it
// removes, each in id order, nil for none.
type Diff struct {
	Added   []uuid.UUID
	Removed []uuid.UUID
}

// diffOf is the change of the set from, a work item's, in id order, into
// the set to, given in any order: the ids of to that from has not, and
// those of from that to has not.
func diffOf(from, to []uuid.UUID) Diff {
	var d Diff
	for _, id := range slices.SortedFunc(slices.Values(to), uuid.UUID.Compare) {
		if !slices.Contains(from, id) {
			d.Added = append(d.Added, id)
		}
	}
	for _, id := range from {
		if !slices.Contains(to, id) {
			d.Removed = append(d.Removed, id)
		}
	}
	return d
}

// Changes is what a checked patch changes in a work item (M4 design 3.7):
// the new name, priority, state and dates where each differs from the
// work item's, nil where it is not given or is the same; and the accounts
// and the labels it adds and removes (Diff), where the patch gives the
// new set. An account or a label the work item has and the new set keeps
// is left as it is: its row is not written again.
type Changes struct {
	Name       *string
	Priority   *Priority
	StateID    *uuid.UUID
	StartDate  *DateChange
	TargetDate *DateChange
	Assignees  Diff
	Labels     Diff
}

// ChangesOf is what p, checked (CheckIssuePatch), changes in i.
func ChangesOf(i Issue, p IssuePatch) Changes {
	var c Changes
	if p.Name != nil && *p.Name != i.Name {
		c.Name = p.Name
	}
	if p.Priority != nil && *p.Priority != i.Priority {
		c.Priority = p.Priority
	}
	if p.StateID != nil && *p.StateID != i.StateID {
		c.StateID = p.StateID
	}
	if p.SetStartDate && !sameDay(p.StartDate, i.StartDate) {
		c.StartDate = &DateChange{To: p.StartDate}
	}
	if p.SetTargetDate && !sameDay(p.TargetDate, i.TargetDate) {
		c.TargetDate = &DateChange{To: p.TargetDate}
	}
	if p.AssigneeIDs != nil {
		c.Assignees = diffOf(i.AssigneeIDs, *p.AssigneeIDs)
	}
	if p.LabelIDs != nil {
		c.Labels = diffOf(i.LabelIDs, *p.LabelIDs)
	}
	return c
}

// Unassignable are the problems of the accounts of asked, a patch's
// assignee_ids, that it adds to the work item (added, Changes.Assignees'
// Added) and that may not be assigned (CanAssign), from roles, their
// active project roles, each at its index in asked (M4 design 3.6): an
// account who has left the project, or is its guest, is not assigned anew;
// one the work item has stays.
func Unassignable(asked, added []uuid.UUID, roles map[uuid.UUID]shared.Role) []shared.FieldError {
	var refused []shared.FieldError
	for i, id := range asked {
		if slices.Contains(added, id) && !CanAssign(roles[id]) {
			refused = append(refused, unassignable(i))
		}
	}
	return refused
}

// None reports whether c changes nothing: then the update writes no row.
func (c Changes) None() bool {
	return c.Name == nil && c.Priority == nil && c.StateID == nil && c.StartDate == nil && c.TargetDate == nil && len(c.Assignees.Added) == 0 &&
		len(c.Assignees.Removed) == 0 && len(c.Labels.Added) == 0 && len(c.Labels.Removed) == 0
}
