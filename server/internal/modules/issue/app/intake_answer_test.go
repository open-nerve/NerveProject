package app_test

import (
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fixedAt is a clock that answers its moment, its reads not logged: Locks'
// clock where a test does not watch it.
type fixedAt time.Time

func (c fixedAt) Now() time.Time { return time.Time(c) }

// locksClockAt is Locks' clock at the moment at, each read logged as
// "Locks.Now", apart from a write's own clock ("Now").
type locksClockAt struct {
	at  time.Time
	log *callLog
}

func (c locksClockAt) Now() time.Time {
	c.log.calls = append(c.log.calls, "Locks.Now")
	return c.at
}

// newIntakeRead is getIssue over newGet's world, its work item 7, carol's,
// read through the intake rows' fake, which carries stored when it is not
// nil; Locks' clock at clockNow, logged. dup, carol's work item 9 of web,
// is in the store.
func newIntakeRead(stored *domain.IntakeFacts) (*app.GetIssue, *world, *fakeIntake, uuid.UUID) {
	_, w, id := newGet()
	w.issues.others = map[uuid.UUID]domain.Issue{dup: {ID: dup, WorkspaceID: acme, ProjectID: web, SequenceID: 9, CreatedByID: &carol}}
	intake := &fakeIntake{fakeIssues: w.issues, facts: map[uuid.UUID]domain.IntakeFacts{}}
	if stored != nil {
		intake.facts[id] = *stored
	}
	w.locks = app.NewLocks(w.project, intake, w.auth, fakeGate{log: w.log}, fakeTx{log: w.log}, noIssueAttachments{}, locksClockAt{at: clockNow, log: w.log})
	return app.NewGetIssue(w.locks, fakeTx{log: w.log}), w, intake, id
}

// intakeOf is an intake row of the status s, snoozed till till, a
// duplicate of target, each nil for none.
func intakeOf(s domain.IntakeStatus, till *time.Time, target *uuid.UUID) *domain.IntakeFacts {
	return &domain.IntakeFacts{Status: s, SnoozedTill: till, DuplicateToID: target}
}

// A work item's detail answers its intake row as the caller is answered it
// at the moment of the answer (M7 P2a spec 2.3, 2.8): none for a work item
// never submitted; its effective status, pending for a snooze run out, at
// the moment too, its time then none, Locks' clock read once after the
// count for a snooze and for nothing else; the work item it duplicates,
// read by its id after the count and only for a duplicate, when it is an
// undeleted work item of the project the caller sees, else none: one not
// there or deleted, one of another project, and one the guest rule hides
// from him alike. The row itself is not changed.
func TestADetailAnswersItsIntakeRowAsOfItsMoment(t *testing.T) {
	missing, elsewhere := uuid.NewV7(), uuid.NewV7()
	ownOnly := func(w *world) {
		w.issues.issue.By = alice
		w.auth.grants = []shared.Grant{{OwnOnly: true}}
	}
	for _, tt := range []struct {
		name   string
		stored *domain.IntakeFacts
		set    func(w *world)
		want   *domain.IntakeFacts
		read   *uuid.UUID // the duplicate's target read, if any
	}{
		{"never submitted", nil, func(*world) {}, nil, nil},
		{"pending", intakeOf(domain.IntakePending, nil, nil), func(*world) {}, intakeOf(domain.IntakePending, nil, nil), nil},
		{"snoozed", intakeOf(domain.IntakeSnoozed, &soon, nil), func(*world) {}, intakeOf(domain.IntakeSnoozed, &soon, nil), nil},
		{"a snooze run out at the moment", intakeOf(domain.IntakeSnoozed, &clockNow, nil), func(*world) {}, intakeOf(domain.IntakePending, nil, nil), nil},
		{"a snooze run out before it", intakeOf(domain.IntakeSnoozed, &gone, nil), func(*world) {}, intakeOf(domain.IntakePending, nil, nil), nil},
		{"accepted", intakeOf(domain.IntakeAccepted, nil, nil), func(*world) {}, intakeOf(domain.IntakeAccepted, nil, nil), nil},
		{"declined", intakeOf(domain.IntakeDeclined, nil, nil), func(*world) {}, intakeOf(domain.IntakeDeclined, nil, nil), nil},
		{"a duplicate of one seen", intakeOf(domain.IntakeDuplicate, nil, &dup), func(*world) {}, intakeOf(domain.IntakeDuplicate, nil, &dup), &dup},
		{"a duplicate of one not there", intakeOf(domain.IntakeDuplicate, nil, &missing), func(*world) {}, intakeOf(domain.IntakeDuplicate, nil, nil),
			&missing},
		{"a duplicate of another project's", intakeOf(domain.IntakeDuplicate, nil, &elsewhere), func(w *world) {
			w.issues.others[elsewhere] = domain.Issue{ID: elsewhere, WorkspaceID: acme, ProjectID: uuid.NewV7(), CreatedByID: &carol}
		}, intakeOf(domain.IntakeDuplicate, nil, nil), &elsewhere},
		{"a duplicate the guest rule hides", intakeOf(domain.IntakeDuplicate, nil, &dup), ownOnly, intakeOf(domain.IntakeDuplicate, nil, nil), &dup},
		{"a duplicate the guest made", intakeOf(domain.IntakeDuplicate, nil, &dup), func(w *world) {
			ownOnly(w)
			o := w.issues.others[dup]
			o.CreatedByID = &alice
			w.issues.others[dup] = o
		}, intakeOf(domain.IntakeDuplicate, nil, &dup), &dup},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, intake, id := newIntakeRead(tt.stored)
			tt.set(w)
			got, err := uc.Execute(as(alice), id)
			if err != nil || !reflect.DeepEqual(got.Intake, tt.want) {
				t.Fatalf("Execute() = intake %+v, %v; want %+v", got.Intake, err, tt.want)
			}
			calls := []string{"Begin", line("GetIssue", id), line("Authorize", alice, domain.ActionRead, acme, web), counted(id)}
			if tt.stored != nil && tt.stored.Status == domain.IntakeSnoozed {
				calls = append(calls, "Locks.Now")
			}
			if tt.read != nil {
				calls = append(calls, line("GetIssue", *tt.read))
			}
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			if row, ok := intake.facts[id]; tt.stored != nil && (!ok || !reflect.DeepEqual(row, *tt.stored)) {
				t.Errorf("the intake row is %+v after the answer, want %+v", row, *tt.stored)
			}
		})
	}
}

// A write's answer shows the intake row at the write's own moment, not at
// a reading of Locks' clock (M7 P2a spec 2.3, R6): a triage that snoozes
// the work item till soon, at clockNow, answers it snoozed, its time as
// stored, though Locks' clock, never read, is past it; the duplicate a
// triage names is answered too, read after the count as any detail's.
func TestAWriteAnswersItsIntakeRowAtItsMoment(t *testing.T) {
	soonStored := soon.Truncate(time.Microsecond)
	for _, tt := range []struct {
		name  string
		asked domain.Triage
		want  *domain.IntakeFacts
		after []string
	}{
		{"snoozed", domain.Triage{Status: domain.IntakeSnoozed, SnoozedTill: &soon}, intakeOf(domain.IntakeSnoozed, &soonStored, nil), nil},
		{"a duplicate", domain.Triage{Status: domain.IntakeDuplicate, DuplicateToID: &dup}, intakeOf(domain.IntakeDuplicate, nil, &dup),
			[]string{line("GetIssue", dup)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := &locksClockAt{at: later.Add(time.Hour)}
			w := newTriageWith(t, c)
			c.log = w.log
			got, err := w.uc.Execute(as(alice), w.id, tt.asked)
			if err != nil || !reflect.DeepEqual(got.Intake, tt.want) {
				t.Fatalf("Execute() = intake %+v, %v; want %+v", got.Intake, err, tt.want)
			}
			at := slices.Index(w.log.calls, counted(w.id))
			if at < 0 || !slices.Equal(w.log.calls[at+1:], tt.after) || slices.Contains(w.log.calls, "Locks.Now") {
				t.Errorf("calls %q: want %q after the count, and no read of Locks' clock", w.log.calls, tt.after)
			}
		})
	}
}
