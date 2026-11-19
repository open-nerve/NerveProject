package app_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// keptIntake is the intake rows' fake that keeps each deletion of an
// intake row alone too, in order.
type keptIntake struct {
	*fakeIntake
	deletions []app.IntakeDeletion
}

func (f *keptIntake) DeleteIntake(ctx context.Context, d app.IntakeDeletion) error {
	f.deletions = append(f.deletions, d)
	return f.fakeIntake.DeleteIntake(ctx, d)
}

// intakeDeleteWorld is web's world (newUpdate) with its intake turned on
// and its work item 7, carol's, in its intake, pending, in the triage
// state: Locks read the work items through the intake rows' fake, so that
// each read carries its intake row; the deletion core deletes the rows
// through the work items' fake, its relations through RelationDeletions;
// events keeps what it publishes.
type intakeDeleteWorld struct {
	*world
	uc     *app.DeleteIntakeIssue
	intake *keptIntake
	events *recordedEvents
	id     uuid.UUID
}

func newIntakeDelete(t *testing.T) *intakeDeleteWorld {
	_, w, id := newUpdate()
	w.project.locked.IntakeView = true
	w.issues.issue.StateID = triage.ID
	intake := &keptIntake{fakeIntake: &fakeIntake{fakeIssues: w.issues, facts: map[uuid.UUID]domain.IntakeFacts{id: {Status: domain.IntakePending}}}}
	w.locks = app.NewLocks(w.project, intake, w.auth, fakeGate{}, fakeTx{log: w.log}, noIssueAttachments{}, fixedAt(clockNow))
	events := &recordedEvents{t: t}
	removal := app.NewIssueRemoval(app.IssueRemovalDeps{Events: events, Rows: w.issues, Relations: app.NewRelationDeletions(w.issues, w.project)})
	uc := app.NewDeleteIntakeIssue(app.DeleteIntakeIssueDeps{Locks: w.locks, Delete: removal, Intake: intake, Clock: clockAt{at: clockNow, log: w.log}})
	return &intakeDeleteWorld{world: w, uc: uc, intake: intake, events: events, id: id}
}

// deletedOn are the calls of a deletion of the intake work item id by
// alice up to the decision under the lock (lockedAs, on intake.delete).
func deletedOn(id uuid.UUID) []string {
	return lockedAs(domain.ActionDeleteIntake, id)
}

// A work item of the intake not accepted, pending, snoozed, a snooze run
// out, declined or a duplicate, is deleted with its intake row (M7 P2b
// spec 2.5, R6, DP-10) through the deletion core, as deleteIssue deletes
// one: under the lock, its subtree read, the clock read once, the rows of
// each table in the global order, its intake row among them, all alice's
// at that moment, and issue.deleted. Its intake row is not deleted alone,
// and no activity is written.
func TestDeleteIntakeIssueDeletesAWorkItemNotAccepted(t *testing.T) {
	for what, stored := range map[string]domain.IntakeFacts{"pending": {Status: domain.IntakePending},
		"snoozed": {Status: domain.IntakeSnoozed, SnoozedTill: &soon}, "a snooze run out": {Status: domain.IntakeSnoozed, SnoozedTill: &gone},
		"declined": {Status: domain.IntakeDeclined}, "a duplicate": {Status: domain.IntakeDuplicate, DuplicateToID: &dup}} {
		t.Run(what, func(t *testing.T) {
			w := newIntakeDelete(t)
			w.intake.facts[w.id] = stored
			if err := w.uc.Execute(as(alice), w.id); err != nil {
				t.Fatal(err)
			}
			if calls := slices.Concat(deletedOn(w.id), []string{line("Subtree", w.id), "Now"}, deletingRows([]uuid.UUID{w.id})); !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			deletion := app.IssueDeletion{IssueIDs: []uuid.UUID{w.id}, By: alice, At: clockNow}
			if want := slices.Repeat([]app.IssueDeletion{deletion}, 14); !reflect.DeepEqual(w.issues.deletions, want) {
				t.Errorf("the deletions %+v, want %+v", w.issues.deletions, want)
			}
			wantIssueEvent(t, w.events, domain.EventIssueDeleted, w.id, nil)
			if len(w.intake.deletions) != 0 || len(w.intake.written) != 0 {
				t.Errorf("the intake row deleted alone %+v, the activity %+v; want neither", w.intake.deletions, w.intake.written)
			}
		})
	}
}

// An accepted work item of the intake stays (M7 P2b spec 2.5, R6, re-86
// m5): under the lock, the clock read once, then its intake row alone is
// deleted, alice's at that moment; no row of the work item's is deleted,
// no activity written, no event published. Its work item archived since
// changes nothing: no issue.archived.
func TestDeleteIntakeIssueDeletesOnlyTheIntakeRowOfAnAcceptedItem(t *testing.T) {
	for what, set := range map[string]func(w *intakeDeleteWorld){
		"accepted":                func(*intakeDeleteWorld) {},
		"accepted, then archived": func(w *intakeDeleteWorld) { w.issues.archivedAt = &earlier },
	} {
		t.Run(what, func(t *testing.T) {
			w := newIntakeDelete(t)
			w.intake.facts[w.id] = domain.IntakeFacts{Status: domain.IntakeAccepted}
			w.issues.issue.StateID = backlog.ID
			set(w)
			if err := w.uc.Execute(as(alice), w.id); err != nil {
				t.Fatal(err)
			}
			if calls := append(deletedOn(w.id), "Now", line("DeleteIntake", w.id)); !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			if want := []app.IntakeDeletion{{IssueID: w.id, By: alice, At: clockNow}}; !reflect.DeepEqual(w.intake.deletions, want) {
				t.Errorf("the intake rows deleted %+v, want %+v", w.intake.deletions, want)
			}
			if _, ok := w.intake.facts[w.id]; ok || len(w.issues.deletions) != 0 || len(w.intake.written) != 0 || len(w.events.rows) != 0 {
				t.Errorf("the intake row kept %v, the deletions %+v, the activity %+v, the events %+v; want the row alone deleted",
					ok, w.issues.deletions, w.intake.written, w.events.rows)
			}
		})
	}
}

// The decision is on intake.delete, the work item's creator its target's
// (AllowCreator, R11), before the lock and again under it; a work item of
// no known creator has none.
func TestDeleteIntakeIssueDecidesWithItsCreator(t *testing.T) {
	for what, tt := range map[string]struct {
		by      uuid.UUID
		creator uuid.UUID
	}{"carol's": {carol, carol}, "of no known creator": {uuid.UUID{}, uuid.UUID{}}} {
		t.Run(what, func(t *testing.T) {
			w := newIntakeDelete(t)
			w.issues.issue.By = tt.by
			if err := w.uc.Execute(as(alice), w.id); err != nil {
				t.Fatal(err)
			}
			target := shared.Target{WorkspaceID: acme, ProjectID: web, CreatorID: tt.creator}
			if want := []shared.Target{target, target}; !reflect.DeepEqual(w.auth.targets, want) {
				t.Errorf("decided on %+v, want %+v", w.auth.targets, want)
			}
		})
	}
}

// A deletion's refusals, in the order of M7 P2b spec 2.5, each case
// breaking the later rules too, so that its own refusal answers first: a
// work item not there, whose project the caller does not see, or that the
// guest rule hides from him, 404 issue.intake_not_found; a role
// intake.delete does not allow, the member who did not submit it among
// them, 403, whether it has an intake row or not (re-86 m5: the creator is
// the decision's, before the row); then a work item without an intake row,
// 404 again, before any lock; under the lock, each of them again; then an
// archived project, 409 project.archived, before its intake turned off,
// 409 issue.intake_disabled, also when turned off while the lock waited
// and for an accepted one. None reads the clock or deletes a row; each 404
// is the one value a random id answers.
func TestDeleteIntakeIssueRefusalOrder(t *testing.T) {
	shut := func(w *intakeDeleteWorld) { w.project.locked.Archived, w.project.locked.IntakeView = true, false }
	plain := func(w *intakeDeleteWorld) { shut(w); delete(w.intake.facts, w.id) }
	for _, tt := range []struct {
		name   string
		set    func(w *intakeDeleteWorld)
		random bool
		want   error
		calls  int // of deletedOn's
	}{
		{"a work item not there", plain, true, domain.ErrIntakeNotFound, 1},
		{"a project not seen", func(w *intakeDeleteWorld) { plain(w); w.auth.answers = []error{shared.ErrNotVisible} }, false, domain.ErrIntakeNotFound, 2},
		{"hidden by the guest rule, a role not allowed", func(w *intakeDeleteWorld) {
			plain(w)
			w.auth.answers, w.auth.grants = []error{shared.Forbidden()}, []shared.Grant{ownOnly}
		}, false, domain.ErrIntakeNotFound, 2},
		{"a role not allowed, no intake row", func(w *intakeDeleteWorld) { plain(w); w.auth.answers = []error{shared.Forbidden()} }, false, shared.Forbidden(), 2},
		{"a role not allowed, its intake row", func(w *intakeDeleteWorld) { shut(w); w.auth.answers = []error{shared.Forbidden()} }, false, shared.Forbidden(), 2},
		{"a guest's own work item, a role not allowed", func(w *intakeDeleteWorld) {
			shut(w)
			w.issues.issue.By, w.auth.answers, w.auth.grants = alice, []error{shared.Forbidden()}, []shared.Grant{ownOnly}
		}, false, shared.Forbidden(), 2},
		{"no intake row", plain, false, domain.ErrIntakeNotFound, 2},
		{"deleted while the lock waited", func(w *intakeDeleteWorld) { shut(w); w.project.waited = func() { w.issues.issue = nil } }, false,
			domain.ErrIntakeNotFound, 5},
		{"no longer allowed", func(w *intakeDeleteWorld) { shut(w); w.auth.answers = []error{nil, shared.Forbidden()} }, false, shared.Forbidden(), 6},
		{"its intake row deleted while the lock waited", func(w *intakeDeleteWorld) {
			shut(w)
			w.project.waited = func() { delete(w.intake.facts, w.id) }
		}, false, domain.ErrIntakeNotFound, 6},
		{"archived, its intake off", shut, false, domain.ErrProjectArchived, 6},
		{"its intake off", func(w *intakeDeleteWorld) { w.project.locked.IntakeView = false }, false, domain.ErrIntakeDisabled, 6},
		{"its intake turned off while the lock waited", func(w *intakeDeleteWorld) {
			w.project.waited = func() { w.project.locked.IntakeView = false }
		}, false, domain.ErrIntakeDisabled, 6},
		{"accepted, its intake off", func(w *intakeDeleteWorld) {
			w.intake.facts[w.id] = domain.IntakeFacts{Status: domain.IntakeAccepted}
			w.project.locked.IntakeView = false
		}, false, domain.ErrIntakeDisabled, 6},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newIntakeDelete(t)
			tt.set(w)
			id := w.id
			if tt.random {
				id = uuid.NewV7()
			}
			err := w.uc.Execute(as(alice), id)
			wantRefusal(t, w.world, err, tt.want, deletedOn(id)[:tt.calls])
			var se *shared.Error
			if tt.want == domain.ErrIntakeNotFound && (!errors.As(err, &se) || se != domain.ErrIntakeNotFound) {
				t.Errorf("Execute() = %#v, want the very value a random id answers, %#v", err, domain.ErrIntakeNotFound)
			}
			if len(w.issues.deletions) != 0 || len(w.intake.deletions) != 0 || len(w.events.rows) != 0 {
				t.Errorf("after the refusal: the deletions %+v, the intake rows deleted %+v, the events %+v; want none", w.issues.deletions,
					w.intake.deletions, w.events.rows)
			}
		})
	}
}

// A port that fails ends the deletion with its failure, as itself, and no
// call after it runs: the transaction rolls back whatever was deleted.
func TestDeleteIntakeIssueStopsAtAFailingPort(t *testing.T) {
	for _, tt := range []struct {
		port   string
		stored domain.IntakeStatus
	}{
		{"GetIssue", domain.IntakePending}, {"Authorize", domain.IntakePending}, {"Lock", domain.IntakePending}, {"Subtree", domain.IntakePending},
		{"DeleteIssueIntakeRows", domain.IntakeDeclined}, {"DeleteIntake", domain.IntakeAccepted},
	} {
		t.Run(tt.port, func(t *testing.T) {
			w := newIntakeDelete(t)
			w.intake.facts[w.id] = domain.IntakeFacts{Status: tt.stored}
			w.log.fail = tt.port
			wantStoppedAt(t, w.world, w.uc.Execute(as(alice), w.id), tt.port)
		})
	}
}
