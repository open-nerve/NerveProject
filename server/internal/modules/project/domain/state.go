package domain

import (
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// StateGroup is a state's group, a value of states."group" (M3 design 4.9,
// Plane's StateGroup, db/models/state.py:14-20).
type StateGroup string

// The groups. The triage group holds the intake's state only (M3 design
// 3.17).
const (
	GroupBacklog   StateGroup = "backlog"
	GroupUnstarted StateGroup = "unstarted"
	GroupStarted   StateGroup = "started"
	GroupCompleted StateGroup = "completed"
	GroupCancelled StateGroup = "cancelled"
	GroupTriage    StateGroup = "triage"
)

// stateGroups are the groups of the states the API shows and writes: every
// group but the triage group (M3 design 3.17, 5.2).
var stateGroups = []StateGroup{GroupBacklog, GroupUnstarted, GroupStarted, GroupCompleted, GroupCancelled}

// NewState is a state to create: its values (M3 design 3.17).
type NewState struct {
	Name        string
	Color       string
	Sequence    float64
	Group       StateGroup
	Default     bool
	Description string
}

// DefaultStates are the six states a new project has, in their order:
// Plane's DEFAULT_STATES (db/models/state.py:24-62), Backlog the default,
// Triage the triage state (M3 design 3.17).
func DefaultStates() []NewState {
	return []NewState{
		{Name: "Backlog", Color: "#60646C", Sequence: 15000, Group: GroupBacklog, Default: true},
		{Name: "Todo", Color: "#60646C", Sequence: 25000, Group: GroupUnstarted},
		{Name: "In Progress", Color: "#F59E0B", Sequence: 35000, Group: GroupStarted},
		{Name: "Done", Color: "#46A758", Sequence: 45000, Group: GroupCompleted},
		{Name: "Cancelled", Color: "#9AA4BC", Sequence: 55000, Group: GroupCancelled},
		{Name: "Triage", Color: "#4E5355", Sequence: 65000, Group: GroupTriage},
	}
}

// State is a state of a project as stored and as the API shows it (M3
// design 5.2). The triage state is never one: the state operations do not
// see it (3.17).
type State struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	Description string
	Color       string
	Group       StateGroup
	Default     bool
	Sequence    float64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Place is the state's id, its project and the project's workspace: where
// a write on it takes its locks.
func (s State) Place() (id, workspaceID, projectID uuid.UUID) {
	return s.ID, s.WorkspaceID, s.ProjectID
}

// StateCreate is what the caller asks for when creating a state (M3 design
// 5.1). Its sequence is the project's to give (SequenceAfter); a new state
// is never the default.
type StateCreate struct {
	Name        string
	Color       string
	Group       StateGroup
	Description string
}

// StatePatch is what updateState changes (M3 design 5.1): each field nil
// when it is not given, and kept. Which state is the default changes
// through markDefaultState alone.
type StatePatch struct {
	Name        *string
	Color       *string
	Group       *StateGroup
	Description *string
	Sequence    *float64
}

// maxStateText is the length of states.name and states.color,
// varchar(255), in characters.
const maxStateText = 255

// sequenceStep is the gap between a project's last state and a new one,
// and firstSequence the sequence of a state of a project that has none
// besides its triage state: Plane's State.save() (db/models/state.py:
// 117-128) and the column's default (M3 design 3.17, 4.9).
const (
	sequenceStep  = 15000
	firstSequence = 65535
)

// CheckNewState checks s (M3 design 3.17):
//   - a name and a color of 1–255 characters, not blank, without NUL,
//     which the database cannot store;
//   - a group of the five; the triage group is not_allowed: the intake's
//     state is not made here;
//   - a description without NUL.
//
// Every field with a problem is reported at once, in one 422
// validation_failed, with one problem per field. Whether the name is
// taken is the database's to say.
func CheckNewState(s StateCreate) error {
	return invalid(checkStateText("name", s.Name), checkStateText("color", s.Color), checkGroup(s.Group),
		checkText("description", s.Description))
}

// CheckStatePatch checks the fields p gives by CheckNewState's rules; a
// sequence is any number. Every field with a problem is reported at once.
func CheckStatePatch(p StatePatch) error {
	var found []*shared.FieldError
	if p.Name != nil {
		found = append(found, checkStateText("name", *p.Name))
	}
	if p.Color != nil {
		found = append(found, checkStateText("color", *p.Color))
	}
	if p.Group != nil {
		found = append(found, checkGroup(*p.Group))
	}
	if p.Description != nil {
		found = append(found, checkText("description", *p.Description))
	}
	return invalid(found...)
}

// SequenceAfter is the sequence of a new state of a project whose states
// but its triage state have greatest as their greatest sequence, nil when
// it has none: one step after it, or the column's default (Plane's
// State.save(), whose manager leaves the triage state out; M3 design
// 3.17).
func SequenceAfter(greatest *float64) float64 {
	if greatest == nil {
		return firstSequence
	}
	return *greatest + sequenceStep
}

// CheckGroupKept refuses a write that took a state from its group, by
// deleting it or moving it to another, and left the group with left
// states: every group keeps one (M3 design 3.17).
func CheckGroupKept(left int) error {
	if left == 0 {
		return ErrStateLastInGroup
	}
	return nil
}

func checkStateText(field, s string) *shared.FieldError {
	if f := checkLength(field, s, maxStateText); f != nil {
		return f
	}
	return checkText(field, s)
}

func checkGroup(g StateGroup) *shared.FieldError {
	switch {
	case g == GroupTriage:
		return &shared.FieldError{Field: "group", Code: shared.FieldNotAllowed, Message: "must not be triage: the intake's state is not made here"}
	case !slices.Contains(stateGroups, g):
		return &shared.FieldError{Field: "group", Code: shared.FieldInvalidFormat, Message: "must be backlog, unstarted, started, completed or cancelled"}
	}
	return nil
}
