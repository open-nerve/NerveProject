// Package domain holds the issue module's rules (M4 design 6.2): pure
// functions and values.
package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Priority is a work item's priority, a value of issues.priority (M4
// design 3.7).
type Priority string

// The priorities, from the most to the least urgent.
const (
	PriorityUrgent Priority = "urgent"
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
	PriorityNone   Priority = "none"
)

// priorities are the values issues.priority takes.
var priorities = []Priority{PriorityUrgent, PriorityHigh, PriorityMedium, PriorityLow, PriorityNone}

// StateGroup is the group of a work item's state, as the project module's
// states.group stores it (M3 design 4.9).
type StateGroup string

// The groups the work items' rules name: a work item in the completed
// group is completed (M4 design 3.7), and one in it or in the cancelled
// group may be archived (3.8); the triage group's state takes no work item
// through the API.
const (
	GroupCompleted StateGroup = "completed"
	GroupCancelled StateGroup = "cancelled"
	GroupTriage    StateGroup = "triage"
)

// Archivable reports whether a work item in a state of group may be
// archived (M4 design 3.8): in the completed or the cancelled group, not
// in any other.
func Archivable(group StateGroup) bool {
	return group == GroupCompleted || group == GroupCancelled
}

// Issue is a work item as stored and as the API shows it (M4 design 5.2),
// with the accounts of its undeleted assignees' rows and the labels of its
// undeleted labels' rows, in the accounts' and the labels' id order, and
// the caller's view of it. The fields of later phases come with them (M4
// design 12, constraint 5).
type Issue struct {
	ID                  uuid.UUID
	WorkspaceID         uuid.UUID
	ProjectID           uuid.UUID
	SequenceID          int
	Name                string
	DescriptionHTML     string
	DescriptionStripped *string
	StateID             uuid.UUID
	Priority            Priority
	StartDate           *shared.Date
	TargetDate          *shared.Date
	ParentID            *uuid.UUID
	SortOrder           float64
	CompletedAt         *time.Time
	ArchivedAt          *time.Time
	AssigneeIDs         []uuid.UUID
	LabelIDs            []uuid.UUID
	CreatedByID         *uuid.UUID
	UpdatedByID         *uuid.UUID
	CreatedAt           time.Time
	UpdatedAt           time.Time
	// The caller's view (M4 design 3.9): the number of its undeleted,
	// unarchived direct sub-items that he sees, by his grant on its
	// project. A read of the store leaves it 0; it is counted when an
	// answer is built (the app's Locks.seen), and a creation answers 0, a
	// new work item having none.
	SubIssuesCount int
}

// HiddenFrom reports whether the guest rule hides i from the account actor,
// whose grant on i's project is ownOnly (M4 design 3.4): the project's
// guest who sees only his own work items does not see one another created,
// nor one whose creator is not known. It hides nothing from any other
// caller.
func (i Issue) HiddenFrom(actor uuid.UUID, ownOnly bool) bool {
	return ownOnly && (i.CreatedByID == nil || *i.CreatedByID != actor)
}

// NewIssue is what the caller asks for when creating a work item (M4
// design 5.2, IssueCreate without the fields of later phases): StateID nil
// for the project's default state, Priority nil for none, a date nil for
// none, ParentID nil for no parent. A field given is checked as given, a
// priority given empty too.
type NewIssue struct {
	DescriptionHTML *string
	Name            string
	StateID         *uuid.UUID
	Priority        *Priority
	AssigneeIDs     []uuid.UUID
	LabelIDs        []uuid.UUID
	StartDate       *shared.Date
	TargetDate      *shared.Date
	ParentID        *uuid.UUID
}

// maxName is the length of issues.name, varchar(255), in characters.
const maxName = 255

// MaxAssignees and MaxLabels are the most accounts and labels a request
// sets on a work item (M4 design 3.7).
const (
	MaxAssignees = 100
	MaxLabels    = 100
)

// sortOrderStep is the gap between the greatest sort order of a state's
// work items and a new one's, and firstSortOrder a state's first work
// item's: Plane's Issue.save() and the column's default (M4 design 3.7).
const (
	sortOrderStep  = 10000
	firstSortOrder = 65535
)

// CheckNewIssue checks n (M4 design 3.7) and returns it as it is stored:
// the name without its surrounding white space, of 1–255 characters and
// without NUL; the priority one of the five, none when it is not given
// (nil), so never nil once checked; at most MaxAssignees accounts and
// MaxLabels labels, each named once; the start date not after the target
// date, as the request gives both (at target_date). Every problem is
// reported at once, in one 422 validation_failed; whether the state, each
// assignee, each label and the parent may be is the use case's, under the
// project's lock.
func CheckNewIssue(n NewIssue) (NewIssue, error) {
	n.Name = strings.TrimSpace(n.Name)
	if n.DescriptionHTML == nil {
		n.DescriptionHTML = ptr(defaultDescriptionHTML)
	}
	if n.Priority == nil {
		n.Priority = ptr(PriorityNone)
	}
	found := []*shared.FieldError{checkName(n.Name), checkPriority(*n.Priority), checkDescription(n.DescriptionHTML)}
	found = append(found, checkIDs("assignee_ids", n.AssigneeIDs, MaxAssignees)...)
	found = append(found, checkIDs("label_ids", n.LabelIDs, MaxLabels)...)
	found = append(found, checkNewDates(n.StartDate, n.TargetDate))
	if err := invalid(found...); err != nil {
		return NewIssue{}, err
	}
	return n, nil
}

// assignableRoles are the project roles of the members a work item is
// assigned to: not its guests (M4 design 3.7).
var assignableRoles = []shared.Role{shared.RoleAdmin, shared.RoleMember}

// CanAssign reports whether an active member of a work item's project, of
// project role role, may be assigned to it, as an assignee asked for or as
// the project's default assignee (M4 design 3.7). The set is named, not a
// bound.
func CanAssign(role shared.Role) bool {
	return slices.Contains(assignableRoles, role)
}

// Assignees are the accounts a new work item is assigned to, in id order
// (M4 design 3.7), from roles, the active project roles of the accounts
// asked for and of the project's default assignee, nil for none: those
// asked for, each of whom CanAssign; when none is asked for, the default
// assignee if he may be, else no one, without a problem. Each account asked
// for who may not be is a problem at its index, and then no one is.
func Assignees(asked []uuid.UUID, defaultAssignee *uuid.UUID, roles map[uuid.UUID]shared.Role) ([]uuid.UUID, []shared.FieldError) {
	if len(asked) == 0 {
		if defaultAssignee != nil && CanAssign(roles[*defaultAssignee]) {
			return []uuid.UUID{*defaultAssignee}, nil
		}
		return nil, nil
	}
	var refused []shared.FieldError
	for i, id := range asked {
		if !CanAssign(roles[id]) {
			refused = append(refused, unassignable(i))
		}
	}
	if len(refused) > 0 {
		return nil, refused
	}
	return slices.SortedFunc(slices.Values(asked), uuid.UUID.Compare), nil
}

// unassignable is the problem of the i-th account asked for, who may not
// be assigned (CanAssign).
func unassignable(i int) shared.FieldError {
	return shared.FieldError{Field: fmt.Sprintf("assignee_ids[%d]", i), Code: shared.FieldNotAllowed,
		Message: "must be an active member of the project who is not its guest"}
}

// UnknownLabel is the problem of the i-th label asked for, which is not an
// undeleted label of the project (M4 design 3.7).
func UnknownLabel(i int) shared.FieldError {
	return shared.FieldError{Field: fmt.Sprintf("label_ids[%d]", i), Code: shared.FieldNotAllowed, Message: "must be a label of the project"}
}

// StateNotAllowed is the problem of a state asked for that is not an
// undeleted state of the project, or is its triage state (M4 design 3.7).
func StateNotAllowed() shared.FieldError {
	return shared.FieldError{Field: "state_id", Code: shared.FieldNotAllowed, Message: "must be a state of the project that is not its triage state"}
}

// SortOrderAfter is the sort order of a new work item in a state whose
// work items, archived ones too, have greatest as their greatest sort
// order, nil when it has none: one step after it, or the column's default
// (M4 design 3.7).
func SortOrderAfter(greatest *float64) float64 {
	if greatest == nil {
		return firstSortOrder
	}
	return *greatest + sortOrderStep
}

// CompletedAt is the completed_at of a work item put at now in a state of
// group (M4 design 3.7): now in the completed group, none in any other,
// the cancelled group too.
func CompletedAt(group StateGroup, now time.Time) *time.Time {
	if group != GroupCompleted {
		return nil
	}
	return &now
}

// invalid is the 422 of the problems found, in order, or nil when there is
// none.
func invalid(found ...*shared.FieldError) error {
	var fields []shared.FieldError
	for _, f := range found {
		if f != nil {
			fields = append(fields, *f)
		}
	}
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}

// checkName refuses a name, without its surrounding white space, that is
// empty, longer than issues.name, or with NUL, which Postgres cannot store.
func checkName(name string) *shared.FieldError {
	switch {
	case name == "":
		return &shared.FieldError{Field: "name", Code: shared.FieldTooShort, Message: "must not be empty"}
	case utf8.RuneCountInString(name) > maxName:
		return &shared.FieldError{Field: "name", Code: shared.FieldTooLong, Message: fmt.Sprintf("must be at most %d characters", maxName)}
	case strings.ContainsRune(name, 0):
		return &shared.FieldError{Field: "name", Code: shared.FieldInvalidFormat, Message: "must not contain a NUL character"}
	}
	return nil
}

// checkPriority refuses a priority given that is not one of the five as
// they are spelt: in another case, or empty.
func checkPriority(p Priority) *shared.FieldError {
	if !slices.Contains(priorities, p) {
		return &shared.FieldError{Field: "priority", Code: shared.FieldInvalidFormat, Message: "must be one of urgent, high, medium, low, none"}
	}
	return nil
}

// checkIDs refuses a list of field longer than limit, as its one problem:
// no id of a list too long is looked at; otherwise each id named before in
// the list, at its index.
func checkIDs(field string, ids []uuid.UUID, limit int) []*shared.FieldError {
	if len(ids) > limit {
		return []*shared.FieldError{{Field: field, Code: shared.FieldTooLong, Message: fmt.Sprintf("must name at most %d", limit)}}
	}
	var found []*shared.FieldError
	named := make(map[uuid.UUID]bool, len(ids))
	for i, id := range ids {
		if named[id] {
			found = append(found, &shared.FieldError{Field: fmt.Sprintf("%s[%d]", field, i), Code: shared.FieldDuplicate, Message: "is listed before"})
		}
		named[id] = true
	}
	return found
}

// checkSomeIDs is checkIDs of a list of field that must name a work item:
// an empty one is refused, as its one problem.
func checkSomeIDs(field string, ids []uuid.UUID, limit int) []*shared.FieldError {
	if len(ids) == 0 {
		return []*shared.FieldError{{Field: field, Code: shared.FieldTooShort, Message: "must name a work item"}}
	}
	return checkIDs(field, ids, limit)
}
