package domain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ids are n fresh ids, in the order drawn: ascending.
func ids(n int) []uuid.UUID {
	out := make([]uuid.UUID, n)
	for i := range out {
		out[i] = uuid.NewV7()
	}
	return out
}

// A name of one character and of 255, each priority, none given, a hundred
// assignees and labels, the dates in order, on the same day, or one alone,
// a parent: accepted, and returned as stored: the name without its
// surrounding white space, the priority none when none is given, the dates
// and the parent as given.
func TestCheckNewIssueAccepts(t *testing.T) {
	state, parent := uuid.NewV7(), uuid.NewV7()
	tests := []struct {
		name string
		in   NewIssue
		want NewIssue
	}{
		{"one character, trimmed, none", NewIssue{Name: "  x\t"}, NewIssue{Name: "x", Priority: ptr(PriorityNone)}},
		{"255 characters", NewIssue{Name: strings.Repeat("é", 255), Priority: ptr(PriorityLow)},
			NewIssue{Name: strings.Repeat("é", 255), Priority: ptr(PriorityLow)}},
		{"everything", NewIssue{Name: "Fix", StateID: &state, Priority: ptr(PriorityUrgent), AssigneeIDs: ids(100), LabelIDs: ids(100),
			StartDate: october(12), TargetDate: october(19), ParentID: &parent},
			NewIssue{Name: "Fix", StateID: &state, Priority: ptr(PriorityUrgent), StartDate: october(12), TargetDate: october(19),
				ParentID: &parent}},
		{"the same day", NewIssue{Name: "a", StartDate: october(19), TargetDate: october(19)},
			NewIssue{Name: "a", Priority: ptr(PriorityNone), StartDate: october(19), TargetDate: october(19)}},
		{"a start date alone", NewIssue{Name: "a", StartDate: october(30)}, NewIssue{Name: "a", Priority: ptr(PriorityNone), StartDate: october(30)}},
		{"a target date alone", NewIssue{Name: "a", TargetDate: october(1)}, NewIssue{Name: "a", Priority: ptr(PriorityNone), TargetDate: october(1)}},
	}
	for _, p := range []Priority{PriorityUrgent, PriorityHigh, PriorityMedium, PriorityLow, PriorityNone} {
		tests = append(tests, struct {
			name string
			in   NewIssue
			want NewIssue
		}{string(p), NewIssue{Name: "a", Priority: ptr(p)}, NewIssue{Name: "a", Priority: ptr(p)}})
	}
	for _, tt := range tests {
		got, err := CheckNewIssue(tt.in)
		tt.want.AssigneeIDs, tt.want.LabelIDs = tt.in.AssigneeIDs, tt.in.LabelIDs
		if err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: CheckNewIssue() = %+v, %v; want %+v", tt.name, got, err, tt.want)
		}
	}
}

// A blank name, one of 256 characters, one with NUL, a priority outside the
// five, in another case or given empty, an account or a label named twice,
// a hundred and one of either, a start date after the target date (at
// target_date): each refused, all of them in one 422 that names each by its
// field and place. A list too long is reported alone: its ids, a repeat
// among them, are not looked at.
func TestCheckNewIssueReportsEveryProblem(t *testing.T) {
	a, b := uuid.NewV7(), uuid.NewV7()
	many := append(ids(MaxAssignees-1), a, a)
	priority := []shared.FieldError{{Field: "priority", Code: "invalid_format", Message: "must be one of urgent, high, medium, low, none"}}
	dates := shared.FieldError{Field: "target_date", Code: "not_allowed", Message: "must not be before the start date"}
	tests := []struct {
		name string
		in   NewIssue
		want []shared.FieldError
	}{
		{"blank", NewIssue{Name: " \n "}, []shared.FieldError{{Field: "name", Code: "too_short", Message: "must not be empty"}}},
		{"256 characters", NewIssue{Name: strings.Repeat("x", 256)},
			[]shared.FieldError{{Field: "name", Code: "too_long", Message: "must be at most 255 characters"}}},
		{"NUL", NewIssue{Name: "a\x00b"}, []shared.FieldError{{Field: "name", Code: "invalid_format", Message: "must not contain a NUL character"}}},
		{"another case", NewIssue{Name: "a", Priority: ptr[Priority]("Urgent")}, priority},
		{"given empty", NewIssue{Name: "a", Priority: ptr[Priority]("")}, priority},
		{"repeats", NewIssue{Name: "a", AssigneeIDs: []uuid.UUID{a, b, a, b, a}, LabelIDs: []uuid.UUID{b, b}}, []shared.FieldError{
			{Field: "assignee_ids[2]", Code: "duplicate", Message: "is listed before"},
			{Field: "assignee_ids[3]", Code: "duplicate", Message: "is listed before"},
			{Field: "assignee_ids[4]", Code: "duplicate", Message: "is listed before"},
			{Field: "label_ids[1]", Code: "duplicate", Message: "is listed before"},
		}},
		{"a start date after the target date", NewIssue{Name: "a", StartDate: october(20), TargetDate: october(19)}, []shared.FieldError{dates}},
		{"too many, with repeats, and everything else", NewIssue{Name: "", Priority: ptr[Priority]("high!"), AssigneeIDs: many, LabelIDs: many,
			StartDate: october(20), TargetDate: october(19)},
			[]shared.FieldError{
				{Field: "name", Code: "too_short", Message: "must not be empty"},
				priority[0],
				{Field: "assignee_ids", Code: "too_long", Message: "must name at most 100"},
				{Field: "label_ids", Code: "too_long", Message: "must name at most 100"},
				dates,
			}},
	}
	for _, tt := range tests {
		var e *shared.Error
		if _, err := CheckNewIssue(tt.in); !errors.As(err, &e) || e.Code != "validation_failed" || !reflect.DeepEqual(e.Fields, tt.want) {
			t.Errorf("%s: CheckNewIssue() = %v; want validation_failed with %+v", tt.name, err, tt.want)
		}
	}
}

// A project's admins and members may be assigned; its guests, and accounts
// of any other role or of none, may not (M4 design 3.7).
func TestCanAssign(t *testing.T) {
	for role, want := range map[shared.Role]bool{shared.RoleAdmin: true, shared.RoleMember: true, shared.RoleGuest: false, 0: false, 10: false,
		25: false} {
		if got := CanAssign(role); got != want {
			t.Errorf("CanAssign(%d) = %v, want %v", role, got, want)
		}
	}
}

// The accounts asked for, in id order, when each may be assigned, and the
// default assignee not added to them, though he may be; each who may not
// be, a guest or no member, is a problem at his index, and then no one is
// assigned. None asked for: the default assignee if he is an active admin
// or member, else no one, without a problem (M4 design 3.7).
func TestAssignees(t *testing.T) {
	ann, ben, gus, out := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	roles := map[uuid.UUID]shared.Role{ann: shared.RoleAdmin, ben: shared.RoleMember, gus: shared.RoleGuest}
	notAllowed := func(field string) shared.FieldError {
		return shared.FieldError{Field: field, Code: "not_allowed", Message: "must be an active member of the project who is not its guest"}
	}
	tests := []struct {
		name     string
		asked    []uuid.UUID
		fallback *uuid.UUID
		want     []uuid.UUID
		problems []shared.FieldError
	}{
		{"asked for, in id order; the default is not used", []uuid.UUID{ben, ann}, &gus, []uuid.UUID{ann, ben}, nil},
		{"asked for: an assignable default not asked for is not added", []uuid.UUID{ben}, &ann, []uuid.UUID{ben}, nil},
		{"a guest and no member", []uuid.UUID{ann, gus, ben, out}, &ann, nil, []shared.FieldError{notAllowed("assignee_ids[1]"),
			notAllowed("assignee_ids[3]")}},
		{"none asked for: the default member", nil, &ben, []uuid.UUID{ben}, nil},
		{"none asked for: the default admin", []uuid.UUID{}, &ann, []uuid.UUID{ann}, nil},
		{"none asked for: the default is a guest", nil, &gus, nil, nil},
		{"none asked for: the default left", nil, &out, nil, nil},
		{"none asked for, no default", nil, nil, nil, nil},
	}
	for _, tt := range tests {
		got, problems := Assignees(tt.asked, tt.fallback, roles)
		if !reflect.DeepEqual(got, tt.want) || !reflect.DeepEqual(problems, tt.problems) {
			t.Errorf("%s: Assignees() = %v, %+v; want %v, %+v", tt.name, got, problems, tt.want, tt.problems)
		}
	}
}

// A state's first work item takes 65535, the next one step after the
// greatest (M4 design 3.7).
func TestSortOrderAfter(t *testing.T) {
	greatest := 70000.5
	if got := SortOrderAfter(nil); got != 65535 {
		t.Errorf("SortOrderAfter(nil) = %v, want 65535", got)
	}
	if got := SortOrderAfter(&greatest); got != 80000.5 {
		t.Errorf("SortOrderAfter(70000.5) = %v, want 80000.5", got)
	}
}

// A work item is completed at now in the completed group, and in no other,
// the cancelled group neither (M4 design 3.7).
func TestCompletedAt(t *testing.T) {
	now := time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC)
	if got := CompletedAt(GroupCompleted, now); got == nil || !got.Equal(now) {
		t.Errorf("CompletedAt(completed) = %v, want %v", got, now)
	}
	for _, g := range []StateGroup{"backlog", "unstarted", "started", "cancelled", GroupTriage} {
		if got := CompletedAt(g, now); got != nil {
			t.Errorf("CompletedAt(%s) = %v, want none", g, got)
		}
	}
}

// A work item is archived in the completed and the cancelled groups, and
// in no other (M4 design 3.8).
func TestArchivable(t *testing.T) {
	for group, want := range map[StateGroup]bool{"backlog": false, "unstarted": false, "started": false, GroupCompleted: true,
		GroupCancelled: true, GroupTriage: false} {
		if got := Archivable(group); got != want {
			t.Errorf("Archivable(%s) = %v, want %v", group, got, want)
		}
	}
}

// A state asked for that is not the project's, or is its triage state, is
// state_id not_allowed; a label that is not the project's, label_ids[i]
// not_allowed (M4 design 3.7).
func TestDependentRowProblems(t *testing.T) {
	if got, want := StateNotAllowed(), (shared.FieldError{Field: "state_id", Code: "not_allowed",
		Message: "must be a state of the project that is not its triage state"}); got != want {
		t.Errorf("StateNotAllowed() = %+v, want %+v", got, want)
	}
	if got, want := UnknownLabel(3), (shared.FieldError{Field: "label_ids[3]", Code: "not_allowed", Message: "must be a label of the project"}); got != want {
		t.Errorf("UnknownLabel(3) = %+v, want %+v", got, want)
	}
}

// The guest rule hides from a caller who sees only his own work items each
// work item another created, the other's id drawn before his or after it,
// and one whose creator is not known; from any other caller, none (M4
// design 3.4): it tells his own by its creator, not by the ids' order.
func TestTheGuestRuleHidesAnothersWorkItem(t *testing.T) {
	creator := ids(3)
	earlier, gus, other := creator[0], creator[1], creator[2]
	tests := []struct {
		name    string
		by      *uuid.UUID
		ownOnly bool
		want    bool
	}{
		{"his own", &gus, true, false},
		{"another's", &other, true, true},
		{"another's, whose id is drawn before his", &earlier, true, true},
		{"one whose creator is not known", nil, true, true},
		{"another's, to a caller who sees every work item", &other, false, false},
		{"one whose creator is not known, to a caller who sees every work item", nil, false, false},
	}
	for _, tt := range tests {
		if got := (Issue{CreatedByID: tt.by}).HiddenFrom(gus, tt.ownOnly); got != tt.want {
			t.Errorf("%s: HiddenFrom() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
