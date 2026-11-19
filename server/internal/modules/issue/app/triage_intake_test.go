package app_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// soon and later are moments after the clock's, soon with nanoseconds the
// database does not keep; gone is one before it; dup is web's work item 9,
// carol's, the target newTriage stores for a duplicate.
var (
	soon  = clockNow.Add(time.Hour)
	later = clockNow.Add(48 * time.Hour).Truncate(time.Microsecond)
	gone  = clockNow.Add(-time.Minute)
	dup   = uuid.NewV7()
)

// triageWorld is web's world (newUpdate) with its intake turned on and its
// work item 7, carol's, submitted to it: pending, in the triage state. The
// triage's use case runs over it with the update core's: Locks and the
// update read the work items through the intake rows' fake, so that each
// read carries the work item's intake row, as the GetIssue family's does;
// events keeps what the update core publishes. dup, carol's work item 9 of
// web, not in its intake, is in the store.
type triageWorld struct {
	*world
	uc     *app.TriageIntakeIssue
	intake *fakeIntake
	events *recordedEvents
	id     uuid.UUID
}

func newTriage(t *testing.T) *triageWorld {
	return newTriageWith(t, fixedAt(clockNow))
}

// newTriageWith is newTriage with Locks' clock locks.
func newTriageWith(t *testing.T, locks app.Clock) *triageWorld {
	_, w, id := newUpdate()
	w.project.locked.IntakeView = true
	w.issues.issue.StateID = triage.ID
	w.issues.others = map[uuid.UUID]domain.Issue{dup: {ID: dup, WorkspaceID: acme, ProjectID: web, SequenceID: 9, CreatedByID: &carol}}
	intake := &fakeIntake{fakeIssues: w.issues, facts: map[uuid.UUID]domain.IntakeFacts{id: {Status: domain.IntakePending}}}
	w.locks = app.NewLocks(w.project, intake, w.auth, fakeGate{}, fakeTx{log: w.log}, noIssueAttachments{}, locks)
	events := &recordedEvents{t: t}
	clock := clockAt{at: clockNow, log: w.log}
	update := app.NewIssueUpdate(app.IssueUpdateDeps{Events: events, Sanitizer: descriptionSanitizer{}, Locks: w.locks, Issues: intake,
		States: w.project, Labels: w.project, Members: w.project, Profiles: w.profiles})
	uc := app.NewTriageIntakeIssue(app.TriageIntakeIssueDeps{Locks: w.locks, Update: update, Intake: intake, Issues: intake, States: w.project,
		Clock: clock})
	return &triageWorld{world: w, uc: uc, intake: intake, events: events, id: id}
}

// triaged are the calls of a triage of the work item id by alice up to the
// clock: Locks' on intake.triage (lockedAs), then the clock, read once,
// after the lock.
func triaged(id uuid.UUID) []string {
	return append(lockedAs(domain.ActionTriageIntake, id), "Now")
}

// intakeMoved is the row of a triage's activity from the status from to
// to, as M7 P2a spec 2.5 and R7 give it: on intake, the new status its
// verb, the two statuses' names its values, no work item named.
func intakeMoved(from, to domain.IntakeStatus) domain.Activity {
	field, old, new := "intake", string(from), string(to)
	return domain.Activity{Field: &field, Verb: domain.Verb(to), OldValue: &old, NewValue: &new, Comment: "updated the intake status"}
}

// wantWritten checks every activity row w's request wrote, across its
// inserts: one of web's work item for each of activities, in their order,
// in acme, alice's at the clock's moment, each id after the row's before.
func wantWritten(t *testing.T, w *triageWorld, activities ...domain.Activity) {
	t.Helper()
	if len(w.intake.written) != len(activities) {
		t.Fatalf("the activity %+v, want %d rows", w.intake.written, len(activities))
	}
	for i, r := range w.intake.written {
		want := app.ActivityRow{ID: r.ID, WorkspaceID: acme, ProjectID: web, IssueID: w.id, Activity: activities[i], ActorID: &alice, At: clockNow}
		if !reflect.DeepEqual(r, want) || i > 0 && r.ID.Compare(w.intake.written[i-1].ID) <= 0 {
			t.Errorf("activity row %d: %+v, want %+v, its id after the row before", i, r, want)
		}
	}
}

// wantAnswer checks a triage's answer got: the work item as stored, read
// through the intake rows' fake, with its intake row as triaged, facts.
func wantAnswer(t *testing.T, w *triageWorld, got domain.Issue, facts domain.IntakeFacts) {
	t.Helper()
	stored, _, _ := w.intake.GetIssue(context.Background(), w.id)
	if !reflect.DeepEqual(got, stored) || got.Intake == nil || !reflect.DeepEqual(*got.Intake, facts) {
		t.Errorf("Execute() = %+v, want the work item as stored, %+v, its intake row %+v", got, stored, facts)
	}
}

// A triage that accepts a work item (M7 P2a spec 2.5, R6), under the
// lock, in this order: the clock, read once; the project's default state
// read; the intake row, accepted, alice's at that moment; its activity,
// from the row's effective status, pending for a snooze run out, to
// accepted; then the update core with the patch {state: Backlog}, at the
// same moment: the state read again and checked, the work item moved from
// Triage to Backlog, not completed, its activity after the triage's,
// issue.updated with the state's change alone, the work item read back as
// alice sees it. The clock is read once, after the lock.
func TestTriageIntakeIssueAcceptsIntoTheDefaultState(t *testing.T) {
	for _, tt := range []struct {
		name   string
		stored domain.IntakeFacts
		from   domain.IntakeStatus
	}{
		{"pending", domain.IntakeFacts{Status: domain.IntakePending}, domain.IntakePending},
		{"snoozed", domain.IntakeFacts{Status: domain.IntakeSnoozed, SnoozedTill: &soon}, domain.IntakeSnoozed},
		{"a snooze run out", domain.IntakeFacts{Status: domain.IntakeSnoozed, SnoozedTill: &gone}, domain.IntakePending},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newTriage(t)
			id := w.id
			w.intake.facts[id] = tt.stored
			got, err := w.uc.Execute(as(alice), id, domain.Triage{Status: domain.IntakeAccepted})
			if err != nil {
				t.Fatalf("Execute() = %v", err)
			}
			calls := append(triaged(id), line("States", web), line("TriageIntake", id, domain.IntakeAccepted), line("InsertActivities", 1),
				line("States", web), line("UpdateIssue", id), line("InsertActivities", 1), line("GetIssue", id), counted(id))
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			if want := []app.IntakeTriage{{IssueID: id, Status: domain.IntakeAccepted, By: alice, At: clockNow}}; !reflect.DeepEqual(w.intake.triages, want) {
				t.Errorf("the triages written %+v, want %+v", w.intake.triages, want)
			}
			moved := domain.Changed[domain.Named]{Old: domain.Named{ID: triage.ID, Name: "Triage"}, New: domain.Named{ID: backlog.ID, Name: "Backlog"}}
			wantWritten(t, w, append([]domain.Activity{intakeMoved(tt.from, domain.IntakeAccepted)}, domain.UpdateActivity(domain.Update{State: &moved})...)...)
			if f := w.issues.changed; f == nil || f.StateID != backlog.ID || f.CompletedAt != nil || f.By != alice || !f.At.Equal(clockNow) {
				t.Errorf("the fields written %+v, want Backlog, not completed, alice's at %v", f, clockNow)
			}
			wantIssueEvent(t, w.events, domain.EventIssueUpdated, id,
				[]shared.Change{{Field: "state", Old: evptr("Triage"), New: evptr("Backlog"), OldID: &triage.ID, NewID: &backlog.ID}})
			wantAnswer(t, w, got, domain.IntakeFacts{Status: domain.IntakeAccepted})
			if got.StateID != backlog.ID {
				t.Errorf("Execute() in the state %s, want Backlog", got.StateID)
			}
		})
	}
}

// An accepted work item is completed by its new state's group, as any
// change of state completes one (M7 P2a spec 2.5, domain.CompletedAt): in
// a project whose default state is of the completed group, at the triage's
// moment.
func TestTriageIntakeIssueAcceptsIntoADefaultStateThatCompletes(t *testing.T) {
	w := newTriage(t)
	notDefault, completing := backlog, done
	notDefault.Default, completing.Default = false, true
	w.project.states = []app.StateRef{notDefault, completing, triage}
	if _, err := w.uc.Execute(as(alice), w.id, domain.Triage{Status: domain.IntakeAccepted}); err != nil {
		t.Fatalf("Execute() = %v", err)
	}
	if f := w.issues.changed; f == nil || f.StateID != done.ID || f.CompletedAt == nil || !f.CompletedAt.Equal(clockNow) {
		t.Errorf("the fields written %+v, want Done, completed at %v", f, clockNow)
	}
}

// What cannot be is an error, not a refusal, and an accept it stops writes
// nothing (M7 P2a spec 2.3, 2.5): a project without a default state, as
// for a creation; one whose default state is of the triage group, into
// which an accept would leave the work item in Triage while its intake row
// says accepted. The default state is read under the lock, after the
// clock, before the intake row: no row is written, no activity, the work
// item is not changed and no event is published.
func TestTriageIntakeIssueAcceptNeedsADefaultStateOutsideTriage(t *testing.T) {
	notDefault, triageDefault := backlog, triage
	notDefault.Default, triageDefault.Default = false, true
	for _, tt := range []struct {
		name   string
		states []app.StateRef
	}{
		{"no default state", []app.StateRef{notDefault, done, triage}},
		{"a default state of the triage group", []app.StateRef{notDefault, done, triageDefault}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newTriage(t)
			w.project.states = tt.states
			_, err := w.uc.Execute(as(alice), w.id, domain.Triage{Status: domain.IntakeAccepted})
			var se *shared.Error
			if err == nil || errors.As(err, &se) {
				t.Errorf("Execute() = %v, want an error that is no refusal", err)
			}
			if calls := append(triaged(w.id), line("States", web)); !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			if len(w.intake.triages) != 0 || len(w.intake.written) != 0 || w.issues.changed != nil || len(w.events.rows) != 0 {
				t.Errorf("the triages %+v, the activity %+v, the fields %+v, the events %+v; want none", w.intake.triages, w.intake.written,
					w.issues.changed, w.events.rows)
			}
		})
	}
}

// A triage that does not accept (M7 P2a spec 2.5, R3): from pending to
// snoozed, declined or a duplicate; from snoozed to pending, snoozed till
// another time, declined or a duplicate; a snooze run out, at the moment
// too, moves as pending. Under the lock, the clock is read once; a
// duplicate's target read; the intake row written, its snooze stored at
// the microsecond, alice's at that moment; its activity, from the
// effective status; the work item read back as alice sees it, a
// duplicate's target read again for the answer. The work item is not
// changed, no event is published, no one is notified.
func TestTriageIntakeIssueMovesItsRow(t *testing.T) {
	pending, snoozed := domain.IntakeFacts{Status: domain.IntakePending}, domain.IntakeFacts{Status: domain.IntakeSnoozed, SnoozedTill: &soon}
	runOut, atTheMoment := domain.IntakeFacts{Status: domain.IntakeSnoozed, SnoozedTill: &gone}, domain.IntakeFacts{Status: domain.IntakeSnoozed, SnoozedTill: &clockNow}
	soonStored := soon.Truncate(time.Microsecond)
	snooze := func(till *time.Time) domain.Triage {
		return domain.Triage{Status: domain.IntakeSnoozed, SnoozedTill: till}
	}
	duplicate := domain.Triage{Status: domain.IntakeDuplicate, DuplicateToID: &dup}
	for _, tt := range []struct {
		name   string
		stored domain.IntakeFacts
		asked  domain.Triage
		from   domain.IntakeStatus
		facts  domain.IntakeFacts
	}{
		{"pending, snoozed", pending, snooze(&soon), domain.IntakePending, domain.IntakeFacts{Status: domain.IntakeSnoozed, SnoozedTill: &soonStored}},
		{"pending, declined", pending, domain.Triage{Status: domain.IntakeDeclined}, domain.IntakePending, domain.IntakeFacts{Status: domain.IntakeDeclined}},
		{"pending, a duplicate", pending, duplicate, domain.IntakePending, domain.IntakeFacts{Status: domain.IntakeDuplicate, DuplicateToID: &dup}},
		{"snoozed, unsnoozed", snoozed, domain.Triage{Status: domain.IntakePending}, domain.IntakeSnoozed, pending},
		{"snoozed, till later", snoozed, snooze(&later), domain.IntakeSnoozed, domain.IntakeFacts{Status: domain.IntakeSnoozed, SnoozedTill: &later}},
		{"snoozed, declined", snoozed, domain.Triage{Status: domain.IntakeDeclined}, domain.IntakeSnoozed, domain.IntakeFacts{Status: domain.IntakeDeclined}},
		{"snoozed, a duplicate", snoozed, duplicate, domain.IntakeSnoozed, domain.IntakeFacts{Status: domain.IntakeDuplicate, DuplicateToID: &dup}},
		{"a snooze run out, declined", runOut, domain.Triage{Status: domain.IntakeDeclined}, domain.IntakePending, domain.IntakeFacts{Status: domain.IntakeDeclined}},
		{"a snooze run out at the moment, snoozed again", atTheMoment, snooze(&later), domain.IntakePending,
			domain.IntakeFacts{Status: domain.IntakeSnoozed, SnoozedTill: &later}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newTriage(t)
			id := w.id
			w.intake.facts[id] = tt.stored
			got, err := w.uc.Execute(as(alice), id, tt.asked)
			if err != nil {
				t.Fatalf("Execute() = %v", err)
			}
			calls := triaged(id)
			if tt.asked.DuplicateToID != nil {
				calls = append(calls, line("GetIssue", dup))
			}
			calls = append(calls, line("TriageIntake", id, tt.asked.Status), line("InsertActivities", 1), line("GetIssue", id), counted(id))
			if tt.asked.DuplicateToID != nil {
				calls = append(calls, line("GetIssue", dup)) // the answer's target, read as any detail's (M7 P2a spec 2.8)
			}
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			want := []app.IntakeTriage{{IssueID: id, Status: tt.facts.Status, SnoozedTill: tt.facts.SnoozedTill, DuplicateToID: tt.facts.DuplicateToID,
				By: alice, At: clockNow}}
			if !reflect.DeepEqual(w.intake.triages, want) {
				t.Errorf("the triages written %+v, want %+v", w.intake.triages, want)
			}
			wantWritten(t, w, intakeMoved(tt.from, tt.asked.Status))
			if w.issues.changed != nil || len(w.events.rows) != 0 {
				t.Errorf("the fields written %+v, the events %+v; want none", w.issues.changed, w.events.rows)
			}
			wantAnswer(t, w, got, tt.facts)
		})
	}
}
