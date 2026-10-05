package domain

import (
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The six default states are Plane's, in its order: one per group, Backlog
// the only default, Triage the only triage state, none with a description
// (M3 design 3.17).
func TestDefaultStates(t *testing.T) {
	want := []NewState{
		{"Backlog", "#60646C", 15000, "backlog", true, ""},
		{"Todo", "#60646C", 25000, "unstarted", false, ""},
		{"In Progress", "#F59E0B", 35000, "started", false, ""},
		{"Done", "#46A758", 45000, "completed", false, ""},
		{"Cancelled", "#9AA4BC", 55000, "cancelled", false, ""},
		{"Triage", "#4E5355", 65000, "triage", false, ""},
	}
	if got := DefaultStates(); !slices.Equal(got, want) {
		t.Errorf("DefaultStates() =\n%+v\nwant\n%+v", got, want)
	}
}

// Place answers the state's own id, workspace and project, each in its
// place.
func TestStatePlace(t *testing.T) {
	s := State{ID: uuid.NewV7(), WorkspaceID: uuid.NewV7(), ProjectID: uuid.NewV7()}
	if id, workspace, project := s.Place(); id != s.ID || workspace != s.WorkspaceID || project != s.ProjectID {
		t.Errorf("Place() = %s, %s, %s; want %s, %s, %s", id, workspace, project, s.ID, s.WorkspaceID, s.ProjectID)
	}
}

// CheckNewState accepts a state of each of the five groups, a name and a
// color of one and of 255 characters, and a description of several lines.
func TestCheckNewStateAcceptsValidStates(t *testing.T) {
	for _, s := range []StateCreate{
		{Name: "Review", Color: "#F59E0B", Group: GroupStarted},
		{Name: "B", Color: "r", Group: GroupBacklog, Description: "多行\n说明"},
		{Name: strings.Repeat("状", 255), Color: strings.Repeat("c", 255), Group: GroupUnstarted},
		{Name: "Shipped", Color: "#46A758", Group: GroupCompleted},
		{Name: "Won't do", Color: "#9AA4BC", Group: GroupCancelled},
	} {
		if err := CheckNewState(s); err != nil {
			t.Errorf("CheckNewState(%+v) = %v, want nil", s, err)
		}
	}
}

// The problems CheckNewState and CheckStatePatch report: one per field,
// each field's first.
var (
	nameEmpty      = shared.FieldError{Field: "name", Code: "too_short", Message: "must not be empty"}
	nameLong       = shared.FieldError{Field: "name", Code: "too_long", Message: "must be at most 255 characters"}
	nameNUL        = shared.FieldError{Field: "name", Code: "invalid_format", Message: "must not contain a NUL character"}
	colorEmpty     = shared.FieldError{Field: "color", Code: "too_short", Message: "must not be empty"}
	colorLong      = shared.FieldError{Field: "color", Code: "too_long", Message: "must be at most 255 characters"}
	colorNUL       = shared.FieldError{Field: "color", Code: "invalid_format", Message: "must not contain a NUL character"}
	groupTriage    = shared.FieldError{Field: "group", Code: "not_allowed", Message: "must not be triage: the intake's state is not made here"}
	groupUnknown   = shared.FieldError{Field: "group", Code: "invalid_format", Message: "must be backlog, unstarted, started, completed or cancelled"}
	descriptionNUL = shared.FieldError{Field: "description", Code: "invalid_format", Message: "must not contain a NUL character"}
)

// CheckNewState reports every field with a problem at once: a name or a
// color empty, blank, of 256 characters or with NUL; the triage group
// (not_allowed) and a group of none of the six, its case another or empty
// (invalid_format); a description with NUL.
func TestCheckNewStateReportsEveryField(t *testing.T) {
	valid := StateCreate{Name: "Review", Color: "#F59E0B", Group: GroupStarted}
	with := func(change func(*StateCreate)) StateCreate {
		s := valid
		change(&s)
		return s
	}
	for _, tt := range []struct {
		name string
		s    StateCreate
		want []shared.FieldError
	}{
		{"empty name", with(func(s *StateCreate) { s.Name = "" }), []shared.FieldError{nameEmpty}},
		{"blank name", with(func(s *StateCreate) { s.Name = " \t\n" }), []shared.FieldError{nameEmpty}},
		{"name of 256 characters", with(func(s *StateCreate) { s.Name = strings.Repeat("状", 256) }), []shared.FieldError{nameLong}},
		{"name with NUL", with(func(s *StateCreate) { s.Name = "Re\x00view" }), []shared.FieldError{nameNUL}},
		{"empty color", with(func(s *StateCreate) { s.Color = "" }), []shared.FieldError{colorEmpty}},
		{"blank color", with(func(s *StateCreate) { s.Color = "  " }), []shared.FieldError{colorEmpty}},
		{"color of 256 characters", with(func(s *StateCreate) { s.Color = strings.Repeat("c", 256) }), []shared.FieldError{colorLong}},
		{"color with NUL", with(func(s *StateCreate) { s.Color = "#\x00" }), []shared.FieldError{colorNUL}},
		{"the triage group", with(func(s *StateCreate) { s.Group = GroupTriage }), []shared.FieldError{groupTriage}},
		{"a group in upper case", with(func(s *StateCreate) { s.Group = "Started" }), []shared.FieldError{groupUnknown}},
		{"an empty group", with(func(s *StateCreate) { s.Group = "" }), []shared.FieldError{groupUnknown}},
		{"another group", with(func(s *StateCreate) { s.Group = "review" }), []shared.FieldError{groupUnknown}},
		{"description with NUL", with(func(s *StateCreate) { s.Description = "a\x00" }), []shared.FieldError{descriptionNUL}},
		{"all at once", StateCreate{Name: "", Color: "\x00", Group: GroupTriage, Description: "\x00"},
			[]shared.FieldError{nameEmpty, colorNUL, groupTriage, descriptionNUL}},
	} {
		if got := fieldsOf(t, CheckNewState(tt.s)); !slices.Equal(got, tt.want) {
			t.Errorf("%s: CheckNewState() fields %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

// CheckStatePatch checks only the fields given, by CheckNewState's rules:
// an empty patch, a sequence alone (any number), and every field valid
// pass; each field given with a problem is reported, all at once.
func TestCheckStatePatch(t *testing.T) {
	for _, p := range []StatePatch{{}, {Sequence: ptr(-1.5)}, {Sequence: ptr(0.0)},
		{Name: ptr("Review"), Color: ptr("#000"), Group: ptr(GroupCancelled), Description: ptr(""), Sequence: ptr(70000.0)}} {
		if err := CheckStatePatch(p); err != nil {
			t.Errorf("CheckStatePatch(%+v) = %v, want nil", p, err)
		}
	}
	for _, tt := range []struct {
		name string
		p    StatePatch
		want []shared.FieldError
	}{
		{"empty name", StatePatch{Name: ptr("")}, []shared.FieldError{nameEmpty}},
		{"name of 256 characters", StatePatch{Name: ptr(strings.Repeat("a", 256))}, []shared.FieldError{nameLong}},
		{"blank color", StatePatch{Color: ptr(" ")}, []shared.FieldError{colorEmpty}},
		{"color of 256 characters", StatePatch{Color: ptr(strings.Repeat("c", 256))}, []shared.FieldError{colorLong}},
		{"the triage group", StatePatch{Group: ptr(GroupTriage)}, []shared.FieldError{groupTriage}},
		{"another group", StatePatch{Group: ptr(StateGroup("done"))}, []shared.FieldError{groupUnknown}},
		{"description with NUL", StatePatch{Description: ptr("\x00")}, []shared.FieldError{descriptionNUL}},
		{"all at once", StatePatch{Name: ptr("a\x00"), Color: ptr(""), Group: ptr(StateGroup("")), Description: ptr("\x00"), Sequence: ptr(1.0)},
			[]shared.FieldError{nameNUL, colorEmpty, groupUnknown, descriptionNUL}},
	} {
		if got := fieldsOf(t, CheckStatePatch(tt.p)); !slices.Equal(got, tt.want) {
			t.Errorf("%s: CheckStatePatch() fields %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

// A new state comes 15000 after the greatest sequence of its project's
// states but the triage state, 70000 after the six default ones, and at
// the column's default, 65535, when the project has none.
func TestSequenceAfter(t *testing.T) {
	for _, tt := range []struct {
		greatest *float64
		want     float64
	}{{nil, 65535}, {ptr(55000.0), 70000}, {ptr(-1.5), 14998.5}, {ptr(0.0), 15000}} {
		if got := SequenceAfter(tt.greatest); got != tt.want {
			t.Errorf("SequenceAfter(%v) = %v, want %v", tt.greatest, got, tt.want)
		}
	}
}

// A write that left a group without states is project.state_last_in_group;
// one that left it one or more is not refused.
func TestCheckGroupKept(t *testing.T) {
	for _, tt := range []struct {
		left int
		want error
	}{{0, ErrStateLastInGroup}, {1, nil}, {2, nil}} {
		if err := CheckGroupKept(tt.left); err != tt.want {
			t.Errorf("CheckGroupKept(%d) = %v, want %v", tt.left, err, tt.want)
		}
	}
}
