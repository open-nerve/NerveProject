package app_test

import (
	"errors"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A content edit's refusals, in the order of M7 P2b spec 2.4 (DP-8), each
// case breaking the later rules too, so that its own refusal answers
// first: a body the domain refuses, 422, before any read; a work item not
// there, whose project the caller does not see, or that the guest rule
// hides from him, 404 issue.intake_not_found; a role intake.update does
// not allow, 403; then the edit's check: a work item without an intake
// row, 404 again, before the 403 of one the caller, not its admin, did not
// submit (re-86 m5: a member who did not submit it is 404 without the row
// and 403 with it), then a guest's field but its name and description,
// 403; all before any lock; under the lock, each of them again, the
// decision's grant as it is then; then an archived project, 409
// project.archived, before its intake turned off, 409
// issue.intake_disabled, each also when it changed while the lock waited;
// then a work item closed, 409 issue.intake_closed, also when accepted
// while the lock waited; then the update core's 422s. No refusal reads the
// clock or writes; each 404 is the one value a random id answers.
func TestUpdateIntakeIssueRefusalOrder(t *testing.T) {
	low, unknown := domain.PriorityLow, uuid.NewV7()
	closed := func(w *intakeEditWorld) { w.intake.facts[w.id] = domain.IntakeFacts{Status: domain.IntakeAccepted} }
	shut := func(w *intakeEditWorld) {
		closed(w)
		w.project.locked.Archived, w.project.locked.IntakeView = true, false
	}
	plain := func(w *intakeEditWorld) { shut(w); delete(w.intake.facts, w.id) }
	skip := true
	bad := domain.IntakeIssueUpdate{Name: name("Mend"), SkipActivity: &skip}
	_, invalid := domain.CheckIntakeIssueUpdate(bad)
	for _, tt := range []struct {
		name   string
		set    func(w *intakeEditWorld)
		edit   *domain.IntakeIssueUpdate // nil for a name, a priority and a label not web's
		random bool
		want   error
		calls  int // of edited's
	}{
		{"a body the domain refuses", plain, &bad, true, invalid, 0},
		{"a work item not there", plain, nil, true, domain.ErrIntakeNotFound, 1},
		{"a project not seen", func(w *intakeEditWorld) { plain(w); w.auth.answers = []error{shared.ErrNotVisible} }, nil, false,
			domain.ErrIntakeNotFound, 2},
		{"hidden by the guest rule, a role not allowed", func(w *intakeEditWorld) {
			plain(w)
			w.auth.answers, w.auth.grants = []error{shared.Forbidden()}, []shared.Grant{ownOnly}
		}, nil, false, domain.ErrIntakeNotFound, 2},
		{"hidden by the guest rule", func(w *intakeEditWorld) { plain(w); w.auth.grants = []shared.Grant{ownOnly} }, nil, false,
			domain.ErrIntakeNotFound, 2},
		{"a role not allowed, no intake row", func(w *intakeEditWorld) { plain(w); w.auth.answers = []error{shared.Forbidden()} }, nil, false,
			shared.Forbidden(), 2},
		{"no intake row, a member who did not submit it", func(w *intakeEditWorld) { plain(w); w.auth.grants = []shared.Grant{webMember} }, nil,
			false, domain.ErrIntakeNotFound, 2},
		{"a member who did not submit it, its intake row", func(w *intakeEditWorld) { shut(w); w.auth.grants = []shared.Grant{webMember} }, nil,
			false, shared.Forbidden(), 2},
		{"a guest who submitted it, its priority", func(w *intakeEditWorld) {
			shut(w)
			w.issues.issue.By, w.auth.grants = alice, []shared.Grant{everyItem}
		}, nil, false, shared.Forbidden(), 2},
		{"deleted while the lock waited", func(w *intakeEditWorld) { shut(w); w.project.waited = func() { w.issues.issue = nil } }, nil, false,
			domain.ErrIntakeNotFound, 5},
		{"no longer allowed", func(w *intakeEditWorld) { shut(w); w.auth.answers = []error{nil, shared.Forbidden()} }, nil, false,
			shared.Forbidden(), 6},
		{"its intake row deleted while the lock waited", func(w *intakeEditWorld) {
			shut(w)
			w.project.waited = func() { delete(w.intake.facts, w.id) }
		}, nil, false, domain.ErrIntakeNotFound, 6},
		{"a member since the lock waited, carol's", func(w *intakeEditWorld) { shut(w); w.auth.grants = []shared.Grant{webAdmin, webMember} }, nil,
			false, shared.Forbidden(), 6},
		{"a guest since the lock waited, her own, its priority", func(w *intakeEditWorld) {
			shut(w)
			w.issues.issue.By, w.auth.grants = alice, []shared.Grant{webAdmin, everyItem}
		}, nil, false, shared.Forbidden(), 6},
		{"archived, its intake off, accepted", shut, nil, false, domain.ErrProjectArchived, 6},
		{"archived while the lock waited", func(w *intakeEditWorld) { w.project.waited = func() { w.project.locked.Archived = true } }, nil,
			false, domain.ErrProjectArchived, 6},
		{"its intake off, accepted", func(w *intakeEditWorld) { closed(w); w.project.locked.IntakeView = false }, nil, false,
			domain.ErrIntakeDisabled, 6},
		{"its intake turned off while the lock waited", func(w *intakeEditWorld) {
			w.project.waited = func() { w.project.locked.IntakeView = false }
		}, nil, false, domain.ErrIntakeDisabled, 6},
		{"accepted, a label not web's", closed, nil, false, domain.ErrIntakeClosed, 6},
		{"accepted while the lock waited", func(w *intakeEditWorld) { w.project.waited = func() { closed(w) } }, nil, false,
			domain.ErrIntakeClosed, 6},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newIntakeEdit(t)
			tt.set(w)
			id, edit := w.id, domain.IntakeIssueUpdate{Name: name("Mend"), Priority: &low, LabelIDs: set(unknown)}
			if tt.random {
				id = uuid.NewV7()
			}
			if tt.edit != nil {
				edit = *tt.edit
			}
			_, err := w.edit.Execute(as(alice), id, edit)
			wantRefusal(t, w.world, err, tt.want, edited(id)[:tt.calls])
			var se *shared.Error
			if tt.want == domain.ErrIntakeNotFound && (!errors.As(err, &se) || se != domain.ErrIntakeNotFound) {
				t.Errorf("Execute() = %#v, want the very value a random id answers, %#v", err, domain.ErrIntakeNotFound)
			}
			untouched(t, w)
		})
	}
}

// The update core's refusals come last (spec 2.4, DP-8): a label not
// web's, of a work item open in an intake turned on, is the core's 422,
// once it has read the labels; nothing is written.
func TestUpdateIntakeIssueAnswersTheCoresRefusalsLast(t *testing.T) {
	w := newIntakeEdit(t)
	unknown := uuid.NewV7()
	_, err := w.edit.Execute(as(alice), w.id, domain.IntakeIssueUpdate{LabelIDs: set(bug, unknown)})
	wantRefusal(t, w.world, err, shared.Invalid(domain.UnknownLabel(1)), append(edited(w.id), line("Labels", web, []uuid.UUID{unknown})))
	untouched(t, w)
}

// A port that fails ends the edit with its failure, as itself, and no
// call after it runs: the transaction rolls back whatever was written.
func TestUpdateIntakeIssueStopsAtAFailingPort(t *testing.T) {
	for _, port := range []string{"GetIssue", "Authorize", "Lock", "UpdateIssue", "InsertActivities"} {
		t.Run(port, func(t *testing.T) {
			w := newIntakeEdit(t)
			w.log.fail = port
			_, err := w.edit.Execute(as(alice), w.id, domain.IntakeIssueUpdate{Name: name("Mend")})
			wantStoppedAt(t, w.world, err, port)
		})
	}
}
