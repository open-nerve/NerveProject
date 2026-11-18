package domain

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// intakeNow is a triage's moment in these tests: at a whole microsecond,
// as a moment read for a write is stored.
var intakeNow = time.Date(2026, 11, 19, 9, 0, 0, 123456000, time.UTC)

// allIntakeStatuses are the five statuses, as the contract lists them.
var allIntakeStatuses = []IntakeStatus{IntakePending, IntakeSnoozed, IntakeAccepted, IntakeDeclined, IntakeDuplicate}

// The transitions (spec 2.5, R3), from each effective status to each of
// the five: pending to any but pending; snoozed to any, snoozed again (a
// new time) and pending (unsnoozed) too; accepted, declined and duplicate
// to none. Each refusal is the one 409; each move answers the status it
// moved from.
func TestIntakeTransitions(t *testing.T) {
	later := intakeNow.Add(time.Hour)
	allowed := map[IntakeStatus][]IntakeStatus{
		IntakePending: {IntakeSnoozed, IntakeAccepted, IntakeDeclined, IntakeDuplicate},
		IntakeSnoozed: {IntakePending, IntakeSnoozed, IntakeAccepted, IntakeDeclined, IntakeDuplicate},
	}
	for _, from := range allIntakeStatuses {
		facts := IntakeFacts{Status: from}
		if from == IntakeSnoozed {
			facts.SnoozedTill = &later
		}
		for _, to := range allIntakeStatuses {
			got, err := facts.Transition(to, intakeNow)
			if slices.Contains(allowed[from], to) {
				if err != nil || got != from {
					t.Errorf("%s → %s = %q, %v; want it moved from %s", from, to, got, err, from)
				}
				continue
			}
			if !errors.Is(err, ErrIntakeTransitionNotAllowed) || got != "" {
				t.Errorf("%s → %s = %q, %v; want 409 issue.intake_transition_not_allowed", from, to, got, err)
			}
		}
	}
}

// A snooze whose time has come moves as pending (spec 2.3, 2.5): to any
// status but pending, from pending, its time at the moment of the triage
// or before it. One whose time is a microsecond after the moment is still
// snoozed: it may be unsnoozed, from snoozed.
func TestAnExpiredSnoozeMovesAsPending(t *testing.T) {
	for name, till := range map[string]time.Time{"at the moment": intakeNow, "before it": intakeNow.Add(-time.Hour)} {
		facts := IntakeFacts{Status: IntakeSnoozed, SnoozedTill: &till}
		if got, err := facts.Transition(IntakePending, intakeNow); !errors.Is(err, ErrIntakeTransitionNotAllowed) {
			t.Errorf("a snooze run out %s → pending = %q, %v; want 409, pending to pending", name, got, err)
		}
		for _, to := range []IntakeStatus{IntakeSnoozed, IntakeAccepted, IntakeDeclined, IntakeDuplicate} {
			if got, err := facts.Transition(to, intakeNow); err != nil || got != IntakePending {
				t.Errorf("a snooze run out %s → %s = %q, %v; want it moved from pending", name, to, got, err)
			}
		}
	}
	next := intakeNow.Add(time.Microsecond)
	if got, err := (IntakeFacts{Status: IntakeSnoozed, SnoozedTill: &next}).Transition(IntakePending, intakeNow); err != nil || got != IntakeSnoozed {
		t.Errorf("a snooze a microsecond away → pending = %q, %v; want it moved from snoozed", got, err)
	}
}

// The effective status (design 3.5, spec 2.3): a stored snoozed whose time
// is at the moment or before it is pending; one whose time is after it,
// by a nanosecond too, is snoozed; every other status is as stored.
func TestEffectiveStatus(t *testing.T) {
	earlier, later := intakeNow.Add(-time.Hour), intakeNow.Add(time.Hour)
	for _, tt := range []struct {
		name   string
		stored IntakeStatus
		till   *time.Time
		want   IntakeStatus
	}{
		{"pending", IntakePending, nil, IntakePending},
		{"snoozed till later", IntakeSnoozed, &later, IntakeSnoozed},
		{"snoozed till a nanosecond later", IntakeSnoozed, new(intakeNow.Add(1)), IntakeSnoozed},
		{"snoozed till the moment", IntakeSnoozed, new(intakeNow), IntakePending},
		{"snoozed till the moment in another zone", IntakeSnoozed, new(intakeNow.In(time.FixedZone("CST", 8*3600))), IntakePending},
		{"snoozed till earlier", IntakeSnoozed, &earlier, IntakePending},
		{"accepted", IntakeAccepted, nil, IntakeAccepted},
		{"declined", IntakeDeclined, nil, IntakeDeclined},
		{"duplicate", IntakeDuplicate, nil, IntakeDuplicate},
	} {
		if got := EffectiveStatus(tt.stored, tt.till, intakeNow); got != tt.want {
			t.Errorf("%s: EffectiveStatus() = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// A work item is in triage while it has an intake row that is not
// accepted (spec 2.3, R2): pending, snoozed, a snooze run out too,
// declined and duplicate; not accepted, and not without a row.
func TestInTriage(t *testing.T) {
	earlier, target := intakeNow.Add(-time.Hour), uuid.NewV7()
	for _, tt := range []struct {
		name   string
		intake *IntakeFacts
		want   bool
	}{
		{"no intake row", nil, false},
		{"pending", &IntakeFacts{Status: IntakePending}, true},
		{"snoozed, run out", &IntakeFacts{Status: IntakeSnoozed, SnoozedTill: &earlier}, true},
		{"accepted", &IntakeFacts{Status: IntakeAccepted}, false},
		{"declined", &IntakeFacts{Status: IntakeDeclined}, true},
		{"duplicate", &IntakeFacts{Status: IntakeDuplicate, DuplicateToID: &target}, true},
	} {
		if got := (Issue{Intake: tt.intake}).InTriage(); got != tt.want {
			t.Errorf("%s: InTriage() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// A triage's body (spec 2.5): its status one of the five as spelt, else
// invalid_format alone, whatever else it gives; snoozed_till required for
// snoozed and not allowed with any other status, duplicate_to_id the same
// for duplicate; every problem at once, status, snoozed_till then
// duplicate_to_id. Whether the time is in the future and the target may
// be is not the body's: a time past, and any target, pass here.
func TestCheckTriage(t *testing.T) {
	earlier, target := intakeNow.Add(-time.Hour), uuid.NewV7()
	unknown := shared.FieldError{Field: "status", Code: "invalid_format", Message: "must be one of pending, snoozed, accepted, declined, duplicate"}
	tillRequired := shared.FieldError{Field: "snoozed_till", Code: "required", Message: "is required for status snoozed"}
	tillRefused := shared.FieldError{Field: "snoozed_till", Code: "not_allowed", Message: "is allowed only for status snoozed"}
	targetRequired := shared.FieldError{Field: "duplicate_to_id", Code: "required", Message: "is required for status duplicate"}
	targetRefused := shared.FieldError{Field: "duplicate_to_id", Code: "not_allowed", Message: "is allowed only for status duplicate"}
	for _, tt := range []struct {
		name string
		in   Triage
		want []shared.FieldError
	}{
		{"pending", Triage{Status: IntakePending}, nil},
		{"snoozed, a time past", Triage{Status: IntakeSnoozed, SnoozedTill: &earlier}, nil},
		{"accepted", Triage{Status: IntakeAccepted}, nil},
		{"declined", Triage{Status: IntakeDeclined}, nil},
		{"duplicate", Triage{Status: IntakeDuplicate, DuplicateToID: &target}, nil},
		{"empty", Triage{}, []shared.FieldError{unknown}},
		{"in another case, with a time", Triage{Status: "Snoozed", SnoozedTill: &earlier}, []shared.FieldError{unknown}},
		{"Plane's number, with a target", Triage{Status: "-1", DuplicateToID: &target}, []shared.FieldError{unknown}},
		{"a state's group", Triage{Status: "triage"}, []shared.FieldError{unknown}},
		{"with a space", Triage{Status: "accepted "}, []shared.FieldError{unknown}},
		{"snoozed without a time", Triage{Status: IntakeSnoozed}, []shared.FieldError{tillRequired}},
		{"duplicate without a target", Triage{Status: IntakeDuplicate}, []shared.FieldError{targetRequired}},
		{"pending with a time", Triage{Status: IntakePending, SnoozedTill: &earlier}, []shared.FieldError{tillRefused}},
		{"accepted with a target", Triage{Status: IntakeAccepted, DuplicateToID: &target}, []shared.FieldError{targetRefused}},
		{"snoozed with a target, without a time", Triage{Status: IntakeSnoozed, DuplicateToID: &target},
			[]shared.FieldError{tillRequired, targetRefused}},
		{"duplicate with a time, without a target", Triage{Status: IntakeDuplicate, SnoozedTill: &earlier},
			[]shared.FieldError{tillRefused, targetRequired}},
		{"declined with both", Triage{Status: IntakeDeclined, SnoozedTill: &earlier, DuplicateToID: &target},
			[]shared.FieldError{tillRefused, targetRefused}},
	} {
		err := CheckTriage(tt.in)
		var se *shared.Error
		if tt.want == nil && err != nil || tt.want != nil && (!errors.As(err, &se) || se.Code != "validation_failed" ||
			!reflect.DeepEqual(se.Fields, tt.want)) {
			t.Errorf("%s: CheckTriage() = %v, want %+v", tt.name, err, tt.want)
		}
	}
}

// A snooze's time (spec 2.5) is taken in UTC at the microsecond the
// database keeps, and must then be after the moment, taken so too:
// identity's CheckAPIToken. A time within the moment's microsecond, or
// before it, is one 422 must_be_future.
func TestCheckSnoozedTill(t *testing.T) {
	later := intakeNow.Add(time.Hour)
	ragged := time.Date(2026, 11, 19, 9, 0, 0, 123456789, time.UTC)
	for _, tt := range []struct {
		name     string
		till     time.Time
		now      time.Time
		want     time.Time
		accepted bool
	}{
		{"an hour later", later, intakeNow, later, true},
		{"a microsecond later", intakeNow.Add(time.Microsecond), intakeNow, intakeNow.Add(time.Microsecond), true},
		{"within a later microsecond", later.Add(999), intakeNow, later, true},
		{"in another zone", later.In(time.FixedZone("CST", 8*3600)), intakeNow, later, true},
		{"the next microsecond of a ragged moment", ragged.Add(211), ragged, intakeNow.Add(time.Microsecond), true},
		{"the moment", intakeNow, intakeNow, time.Time{}, false},
		{"within the moment's microsecond", intakeNow.Add(500), intakeNow, time.Time{}, false},
		{"within a ragged moment's microsecond, after it", ragged.Add(100), ragged, time.Time{}, false},
		{"earlier", intakeNow.Add(-time.Hour), intakeNow, time.Time{}, false},
	} {
		got, err := CheckSnoozedTill(tt.till, tt.now)
		if tt.accepted {
			if err != nil || !got.Equal(tt.want) || got.Location() != time.UTC {
				t.Errorf("%s: CheckSnoozedTill() = %s, %v; want %s", tt.name, got, err, tt.want)
			}
			continue
		}
		var se *shared.Error
		want := []shared.FieldError{{Field: "snoozed_till", Code: "must_be_future", Message: "must be in the future"}}
		if !errors.As(err, &se) || se.Code != "validation_failed" || !reflect.DeepEqual(se.Fields, want) || !got.IsZero() {
			t.Errorf("%s: CheckSnoozedTill() = %s, %v; want 422 must_be_future", tt.name, got, err)
		}
	}
}

// The intake's errors (spec 2.9): their codes and statuses.
func TestIntakeErrors(t *testing.T) {
	for _, tt := range []struct {
		err    *shared.Error
		code   string
		status int
	}{
		{ErrInTriage, "issue.in_triage", 409},
		{ErrIntakeDisabled, "issue.intake_disabled", 409},
		{ErrIntakeNotFound, "issue.intake_not_found", 404},
		{ErrIntakeTransitionNotAllowed, "issue.intake_transition_not_allowed", 409},
	} {
		if tt.err.Code != tt.code || tt.err.ProblemStatus() != tt.status {
			t.Errorf("%s = %d, want %s %d", tt.err.Code, tt.err.ProblemStatus(), tt.code, tt.status)
		}
	}
}
