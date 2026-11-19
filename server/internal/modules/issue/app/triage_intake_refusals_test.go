package app_test

import (
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// unmoved checks a triage's refusal over w (wantRefusal): no intake row
// written, no activity, the work item not changed, no event.
func unmoved(t *testing.T, w *triageWorld, err, want error, calls []string) {
	t.Helper()
	wantRefusal(t, w.world, err, want, calls)
	if len(w.intake.triages) != 0 || len(w.intake.written) != 0 || w.issues.changed != nil || len(w.events.rows) != 0 {
		t.Errorf("after the refusal: the triages %+v, the activity %+v, the fields %+v, the events %+v; want none", w.intake.triages,
			w.intake.written, w.issues.changed, w.events.rows)
	}
}

// A triage's refusals, in the order of M7 P2a spec 2.5 (pre-flight I9),
// each case breaking the later rules too, so that its own refusal answers
// first: the body (domain.CheckTriage), before any read; a work item not
// there, whose project the caller does not see, or that the guest rule
// hides from him, 404 issue.intake_not_found; a role intake.triage does
// not allow, 403; a work item without an intake row, 404 again, before any
// lock; under the lock, each of them again; then an archived project, 409
// project.archived, before its intake turned off, 409
// issue.intake_disabled, neither reading the clock; then the clock, once,
// and the transition from the row's effective status, 409
// issue.intake_transition_not_allowed; then the 422s that need the moment
// or a read under the lock. Each 404 is the one value a random id answers.
func TestTriageIntakeIssueRefusalOrder(t *testing.T) {
	precheck, lockWaited := lockedUpTo(2), lockedUpTo(5)
	stale := func(w *triageWorld) { w.intake.facts[w.id] = domain.IntakeFacts{Status: domain.IntakeDeclined} }
	shut := func(w *triageWorld) { stale(w); w.project.locked.Archived, w.project.locked.IntakeView = true, false }
	off := func(w *triageWorld) { stale(w); w.project.locked.IntakeView = false }
	plain := func(w *triageWorld) { shut(w); delete(w.intake.facts, w.id) }
	late := domain.Triage{Status: domain.IntakeSnoozed, SnoozedTill: &gone}
	for _, tt := range []struct {
		name   string
		set    func(w *triageWorld)
		random bool
		asked  domain.Triage
		want   error
		calls  func(id uuid.UUID) []string
	}{
		{"a status not one of the five, of a work item not there", plain, true, domain.Triage{Status: "Accepted"},
			shared.Invalid(shared.FieldError{Field: "status", Code: "invalid_format", Message: "must be one of pending, snoozed, accepted, declined, duplicate"}),
			func(uuid.UUID) []string { return nil }},
		{"a work item not there", plain, true, late, domain.ErrIntakeNotFound, func(id uuid.UUID) []string { return []string{line("GetIssue", id, "outside tx")} }},
		{"a project not seen", func(w *triageWorld) { plain(w); w.auth.answers = []error{shared.ErrNotVisible} }, false, late, domain.ErrIntakeNotFound, precheck},
		{"hidden by the guest rule, a role not allowed", func(w *triageWorld) {
			plain(w)
			w.auth.answers, w.auth.grants = []error{shared.Forbidden()}, []shared.Grant{ownOnly}
		}, false, late, domain.ErrIntakeNotFound, precheck},
		{"a role not allowed, no intake row", func(w *triageWorld) { plain(w); w.auth.answers = []error{shared.Forbidden()} }, false, late, shared.Forbidden(),
			precheck},
		{"no intake row", plain, false, late, domain.ErrIntakeNotFound, precheck},
		{"deleted while the lock waited", func(w *triageWorld) { shut(w); w.project.waited = func() { w.issues.issue = nil } }, false, late,
			domain.ErrIntakeNotFound, lockWaited},
		{"no longer allowed", func(w *triageWorld) { shut(w); w.auth.answers = []error{nil, shared.Forbidden()} }, false, late, shared.Forbidden(),
			lockedOn},
		{"its intake row deleted while the lock waited", func(w *triageWorld) {
			shut(w)
			w.project.waited = func() { delete(w.intake.facts, w.id) }
		}, false, late, domain.ErrIntakeNotFound, lockedOn},
		{"archived, its intake off", shut, false, late, domain.ErrProjectArchived, lockedOn},
		{"its intake off", off, false, late, domain.ErrIntakeDisabled, lockedOn},
		{"from declined", stale, false, late, domain.ErrIntakeTransitionNotAllowed, triaged},
		{"a snooze not in the future", func(*triageWorld) {}, false, late,
			shared.Invalid(shared.FieldError{Field: "snoozed_till", Code: "must_be_future", Message: "must be in the future"}), triaged},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newTriage(t)
			tt.set(w)
			id := w.id
			if tt.random {
				id = uuid.NewV7()
			}
			_, err := w.uc.Execute(as(alice), id, tt.asked)
			unmoved(t, w, err, tt.want, tt.calls(id))
			var se *shared.Error
			if tt.want == domain.ErrIntakeNotFound && (!errors.As(err, &se) || se != domain.ErrIntakeNotFound) {
				t.Errorf("Execute() = %#v, want the very value a random id answers, %#v", err, domain.ErrIntakeNotFound)
			}
		})
	}
}

// lockedUpTo are the first n calls of a triage of a work item by alice
// up to the decision under the lock (lockedOn): 2, its precheck's; 5, up
// to the work item read again under the lock.
func lockedUpTo(n int) func(id uuid.UUID) []string {
	return func(id uuid.UUID) []string { return lockedOn(id)[:n] }
}

// lockedOn are the calls of a triage of the work item id by alice up to
// the decision under the lock (lockedAs, on intake.triage).
func lockedOn(id uuid.UUID) []string {
	return lockedAs(domain.ActionTriageIntake, id)
}

// No triage moves an intake row from a terminal status, accepted, declined
// or a duplicate, to any; nor one from pending to pending (M7 P2a spec
// 2.5, R3): 409 issue.intake_transition_not_allowed, once the clock is
// read, before a duplicate's target is read. A triage repeated after one
// has moved the row is refused, not answered again.
func TestTriageIntakeIssueRefusesTheTransition(t *testing.T) {
	asked := []domain.Triage{{Status: domain.IntakePending}, {Status: domain.IntakeSnoozed, SnoozedTill: &soon}, {Status: domain.IntakeAccepted},
		{Status: domain.IntakeDeclined}, {Status: domain.IntakeDuplicate, DuplicateToID: &dup}}
	for _, stored := range []domain.IntakeFacts{{Status: domain.IntakeAccepted}, {Status: domain.IntakeDeclined},
		{Status: domain.IntakeDuplicate, DuplicateToID: &dup}, {Status: domain.IntakePending}} {
		for _, a := range asked {
			if stored.Status == domain.IntakePending && a.Status != domain.IntakePending {
				continue
			}
			t.Run(string(stored.Status)+" to "+string(a.Status), func(t *testing.T) {
				w := newTriage(t)
				w.intake.facts[w.id] = stored
				_, err := w.uc.Execute(as(alice), w.id, a)
				unmoved(t, w, err, domain.ErrIntakeTransitionNotAllowed, triaged(w.id))
			})
		}
	}
}

// A duplicate's target (M7 P2a spec 2.5), read under the lock after the
// transition: a work item not there or deleted, of another project,
// hidden from the caller by the guest rule, in the project's intake, or
// the work item triaged itself, which is not read, each answers the same
// one 422 not_allowed on duplicate_to_id, telling nothing of which. An
// archived work item may be one (pre-flight m6), as one accepted from the
// intake may, and so may one the guest rule shows its own creator.
func TestTriageIntakeIssueChecksTheDuplicatesTarget(t *testing.T) {
	refusal := shared.Invalid(shared.FieldError{Field: "duplicate_to_id", Code: "not_allowed",
		Message: "must be another work item of the project that you see and that is not in its intake"})
	other, archivedAt := uuid.NewV7(), clockNow.Add(-24*time.Hour)
	for _, tt := range []struct {
		name   string
		set    func(w *triageWorld, target *domain.Issue)
		self   bool
		absent bool
		want   error
	}{
		{"not there or deleted", func(*triageWorld, *domain.Issue) {}, false, true, refusal},
		{"of another project", func(_ *triageWorld, i *domain.Issue) { i.ProjectID = other }, false, false, refusal},
		{"hidden by the guest rule", func(w *triageWorld, _ *domain.Issue) { ownWorkItem(w) }, false, false, refusal},
		{"pending in the intake", func(_ *triageWorld, i *domain.Issue) { i.Intake = &domain.IntakeFacts{Status: domain.IntakePending} }, false, false, refusal},
		{"declined in the intake", func(_ *triageWorld, i *domain.Issue) { i.Intake = &domain.IntakeFacts{Status: domain.IntakeDeclined} }, false, false, refusal},
		{"the work item itself", func(*triageWorld, *domain.Issue) {}, true, false, refusal},
		{"archived", func(_ *triageWorld, i *domain.Issue) { i.ArchivedAt = &archivedAt }, false, false, nil},
		{"accepted from the intake", func(_ *triageWorld, i *domain.Issue) { i.Intake = &domain.IntakeFacts{Status: domain.IntakeAccepted} }, false, false, nil},
		{"the caller's own, shown him by the guest rule", func(w *triageWorld, i *domain.Issue) { ownWorkItem(w); i.CreatedByID = &alice }, false, false, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newTriage(t)
			target := w.issues.others[dup]
			tt.set(w, &target)
			w.issues.others[dup] = target
			if tt.absent {
				delete(w.issues.others, dup)
			}
			to := dup
			if tt.self {
				to = w.id
			}
			_, err := w.uc.Execute(as(alice), w.id, domain.Triage{Status: domain.IntakeDuplicate, DuplicateToID: &to})
			if tt.want == nil {
				if err != nil || len(w.intake.triages) != 1 || *w.intake.triages[0].DuplicateToID != dup {
					t.Errorf("Execute() = %v, the triages %+v; want the row a duplicate of %s", err, w.intake.triages, dup)
				}
				return
			}
			calls := triaged(w.id)
			if !tt.self {
				calls = append(calls, line("GetIssue", dup))
			}
			unmoved(t, w, err, tt.want, calls)
		})
	}
}

// ownWorkItem makes alice a caller the guest rule hides others' work items
// from, a workspace admin who is web's guest, whom intake.triage allows,
// and web's work item 7 hers, so that she sees it.
func ownWorkItem(w *triageWorld) {
	w.auth.grants = []shared.Grant{ownOnly, ownOnly}
	w.issues.issue.By = alice
}
