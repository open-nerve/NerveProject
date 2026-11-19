package domain

import (
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IntakeStatus is the status of a work item's intake row, as
// intake_issues.status stores it (M7 design 3.5, P2a spec 2.2): Plane's
// numbers -2…2 by name.
type IntakeStatus string

// The five statuses. A work item whose intake row is in any but accepted
// is in triage (spec 2.3, R2).
const (
	IntakePending   IntakeStatus = "pending"
	IntakeSnoozed   IntakeStatus = "snoozed"
	IntakeAccepted  IntakeStatus = "accepted"
	IntakeDeclined  IntakeStatus = "declined"
	IntakeDuplicate IntakeStatus = "duplicate"
)

// NewIntakeIssue is what the caller asks for when submitting a work item
// to a project's intake (M7 P2a spec 2.4): a new work item's name,
// description, priority, assignees, labels and target date, each as
// NewIssue has it. It has no state, start date, parent, cycle or modules:
// a submission is put in the project's triage state, outside its plans,
// until a triage accepts it.
type NewIntakeIssue struct {
	DescriptionHTML *string
	Name            string
	Priority        *Priority
	AssigneeIDs     []uuid.UUID
	LabelIDs        []uuid.UUID
	TargetDate      *shared.Date
}

// CheckNewIntakeIssue checks n as CheckNewIssue checks the new work item
// it asks for, and answers that work item as it is stored.
func CheckNewIntakeIssue(n NewIntakeIssue) (NewIssue, error) {
	return CheckNewIssue(NewIssue{DescriptionHTML: n.DescriptionHTML, Name: n.Name, Priority: n.Priority, AssigneeIDs: n.AssigneeIDs,
		LabelIDs: n.LabelIDs, TargetDate: n.TargetDate})
}

// intakeStatuses are the values intake_issues.status takes, as the
// contract lists them.
var intakeStatuses = []IntakeStatus{IntakePending, IntakeSnoozed, IntakeAccepted, IntakeDeclined, IntakeDuplicate}

// intakeTransitions are the statuses a triage may move an intake row to,
// by its effective status (spec 2.5, R3): from pending to any but pending;
// from snoozed to any, snoozed again with another time and pending to
// unsnooze it.
var intakeTransitions = map[IntakeStatus][]IntakeStatus{
	IntakePending: {IntakeSnoozed, IntakeAccepted, IntakeDeclined, IntakeDuplicate},
	IntakeSnoozed: {IntakePending, IntakeSnoozed, IntakeAccepted, IntakeDeclined, IntakeDuplicate},
	// The terminal statuses: no triage moves a row out of them; a request
	// repeated after one has moved it is refused, not answered again.
	IntakeAccepted:  nil,
	IntakeDeclined:  nil,
	IntakeDuplicate: nil,
}

// EffectiveStatus is the status at now of an intake row stored as stored,
// snoozed till snoozedTill (M7 design 3.5, spec 2.3): pending for a snooze
// whose time is at now or before it, the stored status otherwise. A
// triage's transition, a detail's answer and a guard each read it at their
// one moment; the stored row is not rewritten when its snooze runs out.
func EffectiveStatus(stored IntakeStatus, snoozedTill *time.Time, now time.Time) IntakeStatus {
	if stored == IntakeSnoozed && snoozedTill != nil && !snoozedTill.After(now) {
		return IntakePending
	}
	return stored
}

// IntakeFacts are a work item's undeleted intake row as a read of the
// work item carries it (spec 2.3): its stored status; till when it is
// snoozed, set exactly while the status is snoozed; and the work item it
// duplicates, set exactly while the status is duplicate, which is no
// foreign key and may name a work item deleted since (design 15 D2).
type IntakeFacts struct {
	Status        IntakeStatus
	SnoozedTill   *time.Time
	DuplicateToID *uuid.UUID
}

// AsOf is f as a work item's detail answers it at the moment now gives
// (spec 2.8), which it asks once, and only of a snooze, the one status
// whose answer the moment changes: its effective status; till when it is
// snoozed only while that is snoozed, none once the snooze has run out;
// the work item it duplicates only while it is a duplicate, as stored,
// whose sight by the reader is the use case's to decide. f itself is not
// changed.
func (f IntakeFacts) AsOf(now func() time.Time) IntakeFacts {
	shown := IntakeFacts{Status: f.Status}
	switch f.Status {
	case IntakeSnoozed:
		if shown.Status = EffectiveStatus(f.Status, f.SnoozedTill, now()); shown.Status == IntakeSnoozed {
			shown.SnoozedTill = f.SnoozedTill
		}
	case IntakeDuplicate:
		shown.DuplicateToID = f.DuplicateToID
	}
	return shown
}

// CheckOpen answers 409 issue.intake_closed when f's stored status is
// accepted, declined or duplicate (M7 P2b spec 2.4, DP-7): a content edit
// is for an item still to triage, pending or snoozed, and so is an asset's
// write on a work item in the intake (P2c spec 2.2), never accepted there.
// The stored status alone decides, read under the project's lock, no
// moment: a snooze is open whether its time has come or not.
func (f IntakeFacts) CheckOpen() error {
	if f.Status == IntakePending || f.Status == IntakeSnoozed {
		return nil
	}
	return ErrIntakeClosed
}

// Transition checks a triage of f to the status to, one CheckTriage
// accepted, at now (spec 2.5, R3), and answers the status it moves from,
// f's effective one, which its activity records: a move the transitions
// allow from it, else 409 issue.intake_transition_not_allowed, pending to
// pending and any move from a terminal status too. A snooze whose time has
// come moves as pending.
func (f IntakeFacts) Transition(to IntakeStatus, now time.Time) (IntakeStatus, error) {
	from := EffectiveStatus(f.Status, f.SnoozedTill, now)
	if !slices.Contains(intakeTransitions[from], to) {
		return "", ErrIntakeTransitionNotAllowed
	}
	return from, nil
}

// Triage is a triage as its request asks for it (spec 2.5): the status,
// any string, as the contract's body types it; the time a snooze lasts
// till and the work item a duplicate duplicates, nil when not given.
type Triage struct {
	Status        IntakeStatus
	SnoozedTill   *time.Time
	DuplicateToID *uuid.UUID
}

// CheckTriage checks t's body, before any read (spec 2.5): its status one
// of the five as they are spelt, else invalid_format, the only problem
// then, as which fields belong with it is not known; snoozed_till required
// for snoozed and not allowed with any other status; duplicate_to_id the
// same for duplicate. Every problem in one 422 validation_failed, in the
// order status, snoozed_till, duplicate_to_id. Whether the time is in the
// future (CheckSnoozedTill) and whether the target may be duplicated are
// the use case's, under the lock, after the transition.
func CheckTriage(t Triage) error {
	if !slices.Contains(intakeStatuses, t.Status) {
		return invalid(&shared.FieldError{Field: "status", Code: shared.FieldInvalidFormat,
			Message: "must be one of pending, snoozed, accepted, declined, duplicate"})
	}
	return invalid(onlyWith(t.Status, IntakeSnoozed, "snoozed_till", t.SnoozedTill != nil),
		onlyWith(t.Status, IntakeDuplicate, "duplicate_to_id", t.DuplicateToID != nil))
}

// onlyWith is the problem of field, given or not, in a triage to status,
// when the field belongs with owner alone: required when status is owner
// and it is not given, not_allowed when status is another and it is.
func onlyWith(status, owner IntakeStatus, field string, given bool) *shared.FieldError {
	switch {
	case status == owner && !given:
		return &shared.FieldError{Field: field, Code: shared.FieldRequired, Message: "is required for status " + string(owner)}
	case status != owner && given:
		return &shared.FieldError{Field: field, Code: shared.FieldNotAllowed, Message: "is allowed only for status " + string(owner)}
	}
	return nil
}

// CheckSnoozedTill is till as a snooze at now stores it (spec 2.5): in
// UTC at the microsecond the database keeps, which must be after now,
// taken so too (identity's CheckAPIToken), else one 422 must_be_future on
// snoozed_till. The use case asks it under the lock, at the moment read
// after its last lock.
func CheckSnoozedTill(till, now time.Time) (time.Time, error) {
	till = till.UTC().Truncate(time.Microsecond)
	if !till.After(now.UTC().Truncate(time.Microsecond)) {
		return time.Time{}, shared.Invalid(shared.FieldError{Field: "snoozed_till", Code: shared.FieldMustBeFuture, Message: "must be in the future"})
	}
	return till, nil
}

// DuplicateNotAllowed is the problem of a duplicate's target that may not
// be one (spec 2.5): a work item not there or deleted, of another project,
// hidden from the caller by the guest rule, in the project's intake, or
// the work item triaged itself. Each is the same problem, so that it tells
// nothing of the target; an archived target may be one (pre-flight m6).
func DuplicateNotAllowed() shared.FieldError {
	return shared.FieldError{Field: "duplicate_to_id", Code: shared.FieldNotAllowed,
		Message: "must be another work item of the project that you see and that is not in its intake"}
}

// TriageActivity is the row of a triage's activity (spec 2.5, R7): on
// intake, the new status its verb, from the status the row moved from,
// its effective one, to the new one, each by its name, Plane's sentence.
// It names no work item a duplicate duplicates, which its reader may not
// see.
func TriageActivity(from, to IntakeStatus) Activity {
	return Activity{Field: ptr("intake"), Verb: Verb(to), OldValue: ptr(string(from)), NewValue: ptr(string(to)),
		Comment: "updated the intake status"}
}
